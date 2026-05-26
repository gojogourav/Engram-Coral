package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gojogourav/engram/cmd/internal/docker"
	"github.com/gojogourav/engram/cmd/internal/github"
	"github.com/gojogourav/engram/cmd/internal/grafana"
	"github.com/gojogourav/engram/cmd/internal/k8s"
	"github.com/gojogourav/engram/cmd/internal/llm"
	"github.com/gojogourav/engram/cmd/internal/metrics"
	"github.com/gojogourav/engram/cmd/internal/store"
	"github.com/gojogourav/engram/internal/diff"
)

type WebHookPayload struct {
	Action      string `json:"action"`
	WorkflowRun struct {
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		LogsURL    string `json:"logs_url"`
		HeadBranch string `json:"head_branch"`
		HeadSHA    string `json:"head_sha"`
	} `json:"workflow_run"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type Gateway struct {
	GithubToken   string
	WebHookSecret string
	HTTPClient    *http.Client
	LLMClient     *llm.Client
	DockerClient  *docker.Client
	GrafanaClient *grafana.Client
	K8sClient     *k8s.Client
	Store         *store.RepoStore
	GroqAPIKey    string
}

func (g *Gateway) getRepoCfg(fullName string) *store.RepoConfig {
	if cfg, ok := g.Store.Get(fullName); ok {
		return cfg
	}
	return &store.RepoConfig{
		GithubToken:   g.GithubToken,
		WebhookSecret: g.WebHookSecret,
		K8sClient:     g.K8sClient,
		DockerClient:  g.DockerClient,
		GrafanaClient: g.GrafanaClient,
	}
}

func (g *Gateway) WebHookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	log.Printf("Webhook received, event: %s, body length: %d",
		r.Header.Get("X-Github-Event"), len(body))

	var payload WebHookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Failed to unmarshal payload: %v", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	repoCfg := g.getRepoCfg(payload.Repository.FullName)

	if !verifyGitHubSignature(r, body, repoCfg.WebhookSecret) {
		log.Printf(" Signature verification failed for %s — rejecting request", payload.Repository.FullName)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	log.Printf(" Action: %q, Conclusion: %q, Repo: %s",
		payload.Action, payload.WorkflowRun.Conclusion, payload.Repository.FullName)

	if payload.Action != "completed" || payload.WorkflowRun.Conclusion != "failure" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ignored: Not a failed workflow completion"))
		return
	}

	metrics.WebHookRecieved.Inc()
	log.Println("CI FAILED — Repo:", payload.Repository.FullName)

	go g.processFailedBuild(payload, repoCfg)
	w.WriteHeader(http.StatusOK)
}

func (g *Gateway) processFailedBuild(payload WebHookPayload, repoCfg *store.RepoConfig) {
	pipelineStart := time.Now()
	defer func() {
		metrics.FullPipelineLatency.Observe(time.Since(pipelineStart).Seconds())
	}()

	owner := payload.Repository.Owner.Login
	repo := payload.Repository.Name
	branch := payload.WorkflowRun.HeadBranch
	token := repoCfg.GithubToken

	log.Println("Starting log download...")
	combinedLogs, err := github.FetchAndExtractLogs(payload.WorkflowRun.LogsURL, token)
	if err != nil {
		log.Printf("Failed to extract logs: %v", err)
		metrics.FixFailed.Inc()
		return
	}
	filteredLogs := github.FilterBuildErrors(combinedLogs)
	log.Printf("Filtered logs (%d chars):\n%s", len(filteredLogs), filteredLogs[:min(len(filteredLogs), 300)])

	repoTree, err := github.FileStructure(owner, repo, branch, token, g.HTTPClient)
	if err != nil {
		log.Printf("Failed to fetch file structure: %v", err)
		metrics.FixFailed.Inc()
		return
	}

	log.Println("Starting AI log parsing...")
	brokenFilePaths, err := g.LLMClient.LogParser(filteredLogs, repoTree)
	if err != nil {
		log.Printf("AI failed to parse logs: %v", err)
		metrics.FixFailed.Inc()
		return
	}

	if len(brokenFilePaths) == 0 {
		log.Println("AI could not pinpoint specific source files.")
		return
	}

	log.Printf("Targets: %v", brokenFilePaths)

	var (
		mu          sync.Mutex
		wg          sync.WaitGroup
		fileContext = make(map[string]string)
	)

	for _, filePath := range brokenFilePaths {
		wg.Add(1)
		go func(fp string) {
			defer wg.Done()
			code, err := github.FetchFileContent(owner, repo, fp, token, g.HTTPClient)
			if err != nil {
				log.Printf(" Could not fetch %s: %v", fp, err)
				return
			}
			mu.Lock()
			fileContext[fp] = code
			mu.Unlock()
		}(filePath)
	}
	wg.Wait()

	for _, fp := range brokenFilePaths {
		if strings.HasSuffix(fp, "_test.go") {
			sourceFile := strings.TrimSuffix(fp, "_test.go") + ".go"
			if _, exists := fileContext[sourceFile]; !exists {
				code, err := github.FetchFileContent(owner, repo, sourceFile, token, g.HTTPClient)
				if err == nil {
					fileContext[sourceFile] = code
					log.Printf("Also fetched source file: %s", sourceFile)
				}
			}
		}
	}

	if len(fileContext) == 0 {
		log.Printf("❌ Could not fetch any of the target files — aborting fix pipeline")
		metrics.FixFailed.Inc()
		return
	}

	log.Println(" AI is generating the code fix...")

	aiStart := time.Now()
	gitDiff, err := g.LLMClient.FixGenerator("Fix the bug causing the build failure.", combinedLogs, repoTree, fileContext)
	metrics.AILatency.Observe(time.Since(aiStart).Seconds())
	if err != nil {
		log.Printf("❌ AI failed to generate fix: %v", err)
		metrics.FixFailed.Inc()
		return
	}

	if strings.TrimSpace(gitDiff) == "" {
		log.Printf("❌ AI returned an empty diff — nothing to apply")
		metrics.FixFailed.Inc()
		return
	}

	log.Printf("🔍 Raw diff:\n%s", gitDiff)

	fileDiffs, err := diff.ParseDiff(gitDiff)
	if err != nil {
		log.Printf("❌ Failed to parse diff: %v", err)
		metrics.DiffApplyErrors.Inc()
		return
	}

	patchedFiles, err := diff.ApplyDiff(fileDiffs, fileContext)
	if err != nil {
		log.Printf("❌ Failed to apply diff: %v", err)
		metrics.DiffApplyErrors.Inc()
		return
	}

	metrics.FixesGenerated.Inc()
	metrics.FilesPatched.Observe(float64(len(patchedFiles)))
	log.Printf(" %d files patched, pushing PR...", len(patchedFiles))

	prURL, err := g.applyAndPush(owner, repo, branch, payload.WorkflowRun.HeadSHA, patchedFiles, token)
	if err != nil {
		log.Printf("❌ Failed to push PR: %v", err)
		metrics.FixFailed.Inc()
		return
	}

	metrics.PRsOpened.Inc()
	log.Printf(" PR opened: %s", prURL)
}

func (g *Gateway) applyAndPush(owner, repo, baseBranch, headSHA string, fileContext map[string]string, token string) (string, error) {
	fixBranch := fmt.Sprintf("engram/fix-%s", headSHA[:7])

	err := github.CreateBranch(owner, repo, fixBranch, headSHA, token, g.HTTPClient)
	if err != nil {
		return "", fmt.Errorf("failed to create fix branch: %w", err)
	}

	for filePath, newContent := range fileContext {
		encoded := base64.StdEncoding.EncodeToString([]byte(newContent))
		err := github.UpdateFile(owner, repo, fixBranch, filePath, encoded, token, g.HTTPClient)
		if err != nil {
			return "", fmt.Errorf("failed to update %s: %w", filePath, err)
		}
	}

	prURL, err := github.CreatePullRequest(owner, repo, fixBranch, baseBranch, token, g.HTTPClient)
	if err != nil {
		return "", fmt.Errorf("failed to open PR: %w", err)
	}

	return prURL, nil
}

func (g *Gateway) DockerCommandHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	dockerClient := g.DockerClient
	if repo != "" {
		if cfg, ok := g.Store.Get(repo); ok && cfg.DockerClient != nil {
			dockerClient = cfg.DockerClient
		}
	}
	if dockerClient == nil {
		http.Error(w, "Docker not available", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	cmd, err := docker.InterpretCommand(req.Text)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not interpret command: %v", err), http.StatusBadRequest)
		return
	}

	result, err := dockerClient.ExecuteCommand(cmd)
	if err != nil {
		http.Error(w, fmt.Sprintf("docker command failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"result": result,
	})
}

func (g *Gateway) GrafanaCommandHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	grafanaClient := g.GrafanaClient
	if repo != "" {
		if cfg, ok := g.Store.Get(repo); ok && cfg.GrafanaClient != nil {
			grafanaClient = cfg.GrafanaClient
		}
	}
	if grafanaClient == nil {
		http.Error(w, "Grafana not available", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	result, err := grafanaClient.InterpretCommand(req.Text)
	if err != nil {
		http.Error(w, fmt.Sprintf("grafana query failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"result": result,
	})
}

func (g *Gateway) GrafanaStatsHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	grafanaClient := g.GrafanaClient
	if repo != "" {
		if cfg, ok := g.Store.Get(repo); ok && cfg.GrafanaClient != nil {
			grafanaClient = cfg.GrafanaClient
		}
	}
	if grafanaClient == nil {
		http.Error(w, "Grafana not available", http.StatusServiceUnavailable)
		return
	}

	stats, err := grafanaClient.GetDashboardStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (g *Gateway) K8sCommandHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	k8sClient := g.K8sClient
	if repo != "" {
		if cfg, ok := g.Store.Get(repo); ok && cfg.K8sClient != nil {
			k8sClient = cfg.K8sClient
		}
	}
	if k8sClient == nil {
		http.Error(w, "Kubernetes not available", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	cmd, err := k8s.InterpretCommand(req.Text)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not interpret command: %v", err), http.StatusBadRequest)
		return
	}

	result, err := k8sClient.ExecuteCommand(cmd)
	if err != nil {
		http.Error(w, fmt.Sprintf("k8s command failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"result": result,
	})
}
