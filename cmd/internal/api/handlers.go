package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gojogourav/engram/cmd/internal/docker"
	"github.com/gojogourav/engram/cmd/internal/github"
	"github.com/gojogourav/engram/cmd/internal/grafana"
	"github.com/gojogourav/engram/cmd/internal/incident"
	"github.com/gojogourav/engram/cmd/internal/k8s"
	"github.com/gojogourav/engram/cmd/internal/llm"
	"github.com/gojogourav/engram/cmd/internal/metrics"
	"github.com/gojogourav/engram/cmd/internal/store"
	"github.com/gojogourav/engram/internal/diff"
	"github.com/shirou/gopsutil/v4/cpu"
)

// -----------------------------------------------------------------------------
// CORAL SQL EXECUTION
// -----------------------------------------------------------------------------
func runCoralQuery(query string) (string, error) {
	cmd := exec.Command("coral", "sql", query, "--format", "json")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	raw := string(out)

	// Strip panic noise — find JSON boundaries
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start != -1 && end != -1 && end > start {
		raw = raw[start : end+1]
	} else if start != -1 {
		raw = raw[start:]
	}

	if err != nil && raw == "" {
		return "", fmt.Errorf("coral execution failed: %v\nOutput: %s", err, string(out))
	}
	return raw, nil
}

// -----------------------------------------------------------------------------
// STRUCTS & GATEWAY
// -----------------------------------------------------------------------------
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

// -----------------------------------------------------------------------------
// GITHUB WEBHOOK ENTRYPOINT
// -----------------------------------------------------------------------------
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

	// Tell Prometheus we caught a failure!
	metrics.WebHookRecieved.Inc()
	log.Println("CI FAILED — Repo:", payload.Repository.FullName)

	go g.processFailedBuild(payload, repoCfg)
	w.WriteHeader(http.StatusOK)
}

// -----------------------------------------------------------------------------
// AUTONOMOUS HEALING PIPELINE
// -----------------------------------------------------------------------------
func (g *Gateway) processFailedBuild(payload WebHookPayload, repoCfg *store.RepoConfig) {
	pipelineStart := time.Now()
	defer func() {
		metrics.FullPipelineLatency.Observe(time.Since(pipelineStart).Seconds())
	}()

	owner := payload.Repository.Owner.Login
	repo := payload.Repository.Name
	branch := payload.WorkflowRun.HeadBranch
	token := repoCfg.GithubToken

	incidentID := fmt.Sprintf("INC-%s", payload.WorkflowRun.HeadSHA[:7])
	incident.Global.Create(incidentID, payload.Repository.FullName, payload.WorkflowRun.HeadSHA, branch)
	log.Printf("Incident created: %s", incidentID)

	incident.Global.Advance(incidentID, incident.StageAggregating)
	log.Printf("Stage: Aggregating — running Coral queries")

	blastRadiusData, rawSQL := getBlastRadiusContext(owner, repo, payload.WorkflowRun.HeadSHA)
	failedJobs, err := GetWorkflowFailures(owner, repo, payload.WorkflowRun.ID)
	if err != nil {
		log.Printf("Coral data aggregation failed: %v", err)
		incident.Global.Advance(incidentID, incident.StageFailed)
		return
	}

	// Build Summary String
	var jobSummary strings.Builder
	for _, job := range failedJobs {
		jobSummary.WriteString(fmt.Sprintf("- Job '%s' failed at step: '%s'\n", job.Name, job.FailedStepNames))
	}

	// Update Store with aggregated data
	incident.Global.Update(incidentID, func(i *incident.Incident) {
		i.CommitContext = blastRadiusData
		i.CoralSQLQuery = rawSQL
		i.FailedJobs = jobSummary.String()
	})

	// 4. CONTEXT: Fetch Logs and Structure for AI
	combinedLogs, err := github.FetchAndExtractLogs(payload.WorkflowRun.LogsURL, token)
	if err != nil {
		log.Printf("Failed to extract logs: %v", err)
		incident.Global.Advance(incidentID, incident.StageFailed)
		return
	}
	filteredLogs := github.FilterBuildErrors(combinedLogs)

	repoTree, err := github.FileStructure(owner, repo, branch, token, g.HTTPClient)
	if err != nil {
		log.Printf("Failed to fetch repository tree: %v", err)
		incident.Global.Advance(incidentID, incident.StageFailed)
		return
	}
	// 5. DIAGNOSE: Send context to LLM
	incident.Global.Advance(incidentID, incident.StageDiagnosing)
	enrichedContext := fmt.Sprintf(
		"=== FAILED JOBS ===\n%s\n=== BLAST RADIUS (Coral JSON) ===\n%s\n=== RAW LOGS ===\n%s",
		jobSummary.String(), blastRadiusData, filteredLogs,
	)

	log.Printf("🔍 DEBUG: repoTree length: %d characters", len(repoTree))
	log.Printf("🔍 DEBUG: filteredLogs length: %d characters", len(filteredLogs))
	if len(filteredLogs) == 0 && len(combinedLogs) > 0 {
		log.Printf("⚠️ WARNING: Log filter stripped EVERYTHING! Raw log snippet: %s", combinedLogs[:min(len(combinedLogs), 300)])
	}
	// 👉 AI LATENCY TIMER 1 (LogParser)
	aiStart1 := time.Now()
	brokenFilePaths, err := g.LLMClient.LogParser(enrichedContext, repoTree)
	metrics.AILatency.Observe(time.Since(aiStart1).Seconds())

	if err != nil || len(brokenFilePaths) == 0 {
		log.Printf("LogParser failed: %v", err)
		incident.Global.Advance(incidentID, incident.StageFailed)
		return
	}

	// 6. SANDBOX & APPLY: Fetch files and generate diffs
	incident.Global.Advance(incidentID, incident.StageSandboxing)

	// Fetch file content in parallel
	fileContext := g.fetchFilesConcurrently(brokenFilePaths, owner, repo, token)

	// 👉 AI LATENCY TIMER 2 (FixGenerator)
	aiStart2 := time.Now()
	gitDiff, err := g.LLMClient.FixGenerator("Fix the bug causing the build failure.", enrichedContext, repoTree, fileContext)
	metrics.AILatency.Observe(time.Since(aiStart2).Seconds())

	if err != nil || strings.TrimSpace(gitDiff) == "" {
		incident.Global.Advance(incidentID, incident.StageFailed)
		return
	}

	fileDiffs, err := diff.ParseDiff(gitDiff)
	if err != nil {
		log.Printf("ParseDiff failed: %v", err)
		incident.Global.Advance(incidentID, incident.StageFailed)
		return
	}

	// JIT Fetch missing files if the AI hallucinated a path
	diffLines := strings.Split(gitDiff, "\n")
	for _, line := range diffLines {
		var fileName string
		if strings.HasPrefix(line, "--- a/") {
			fileName = strings.TrimPrefix(line, "--- a/")
		} else if strings.HasPrefix(line, "+++ b/") {
			fileName = strings.TrimPrefix(line, "+++ b/")
		}
		fileName = strings.TrimSpace(fileName)
		if fileName != "" && fileName != "/dev/null" {
			if _, exists := fileContext[fileName]; !exists {

				// 🚀 CORAL SQL REPLACEMENT (JIT Fetch)
				query := fmt.Sprintf("SELECT content FROM github.contents WHERE owner = '%s' AND repo = '%s' AND path = '%s' LIMIT 1", owner, repo, fileName)

				if rawJSON, err := runCoralQuery(query); err == nil {
					var results []map[string]interface{}
					if json.Unmarshal([]byte(rawJSON), &results) == nil && len(results) > 0 {
						if contentStr, ok := results[0]["content"].(string); ok {
							fileContext[fileName] = contentStr
							continue // Successfully fetched!
						}
					}
				}

				// If we reach here, the JIT fetch failed
				log.Printf("JIT Fetch failed via Coral for %s", fileName)
				incident.Global.Advance(incidentID, incident.StageFailed)
				return
			}
		}
	}

	patchedFiles, err := diff.ApplyDiff(fileDiffs, fileContext)
	if err != nil {
		log.Printf("ApplyDiff failed: %v", err)
		incident.Global.Advance(incidentID, incident.StageFailed)
		return
	}

	// 7. APPROVE & HEAL
	confidence := calculateConfidence(failedJobs, patchedFiles)
	incident.Global.Update(incidentID, func(i *incident.Incident) {
		i.GeneratedDiff = gitDiff
		i.ConfidenceScore = confidence
	})

	incident.Global.Advance(incidentID, incident.StagePending)
	if !incident.Global.WaitForApproval(incidentID) {
		incident.Global.Advance(incidentID, incident.StageFailed)
		return
	}

	incident.Global.Advance(incidentID, incident.StageHealing)

	inc, _ := incident.Global.Get(incidentID)
	prURL, err := g.applyAndPush(owner, repo, branch, payload.WorkflowRun.HeadSHA, patchedFiles, token, inc)

	// 👉 SUCCESS METRICS UPDATED HERE
	if err == nil {
		incident.Global.Update(incidentID, func(i *incident.Incident) { i.PRUrl = prURL })
		incident.Global.Advance(incidentID, incident.StageHealed)

		// Tell Prometheus the fix worked!
		metrics.FixesGenerated.Inc()
		metrics.PRsOpened.Inc()
	}
}

// -----------------------------------------------------------------------------
// UTILITIES
// -----------------------------------------------------------------------------
func keys(m map[string]string) []string {
	k := make([]string, 0, len(m))
	for key := range m {
		k = append(k, key)
	}
	return k
}

func getBlastRadiusContext(owner, repo, sha string) (string, string) {
	query := fmt.Sprintf(
		"SELECT sha, author__login, commit__message, commit__author__date FROM github.commits WHERE owner = '%s' AND repo = '%s' AND sha = '%s' LIMIT 1",
		owner, repo, sha,
	)
	result, err := runCoralQuery(query)
	if err != nil {
		return "", query
	}
	return result, query
}

func calculateConfidence(jobs []GitHubJob, patchedFiles map[string]string) int {
	score := 50
	if len(jobs) > 0 {
		score += 20
	}
	if len(patchedFiles) > 0 && len(patchedFiles) <= 3 {
		score += 20
	}
	if len(patchedFiles) > 3 {
		score -= 10
	}
	for _, job := range jobs {
		if job.FailedStepNames != "" {
			score += 10
			break
		}
	}
	if score > 99 {
		score = 99
	}
	return score
}

func (g *Gateway) applyAndPush(owner, repo, baseBranch, headSHA string, fileContext map[string]string, token string, inc *incident.Incident) (string, error) {
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
func (g *Gateway) fetchFilesConcurrently(filePaths []string, owner, repo, token string) map[string]string {
	var (
		mu          sync.Mutex
		wg          sync.WaitGroup
		fileContext = make(map[string]string)
	)

	for _, filePath := range filePaths {
		wg.Add(1)
		go func(fp string) {
			defer wg.Done()

			query := fmt.Sprintf(
				"SELECT content FROM github.contents WHERE owner = '%s' AND repo = '%s' AND path = '%s' LIMIT 1",
				owner, repo, fp,
			)

			rawJSON, err := runCoralQuery(query)
			if err != nil {
				log.Printf("Coral could not fetch %s: %v", fp, err)
				return
			}

			var results []map[string]interface{}
			if err := json.Unmarshal([]byte(rawJSON), &results); err == nil && len(results) > 0 {
				if contentStr, ok := results[0]["content"].(string); ok {
					// 🚀 THE FIX: Decode the Base64! (GitHub wraps base64 in newlines, so we strip them first)
					cleanB64 := strings.ReplaceAll(contentStr, "\n", "")
					decodedBytes, err := base64.StdEncoding.DecodeString(cleanB64)

					mu.Lock()
					if err == nil {
						fileContext[fp] = string(decodedBytes)
						log.Printf(" DECODED FILE CONTENT: %s", string(decodedBytes)[:min(len(decodedBytes), 60)])
					} else {
						fileContext[fp] = contentStr // Fallback just in case
					}
					mu.Unlock()
				}
			}
		}(filePath)
	}
	wg.Wait()
	return fileContext
}
func (g *Gateway) VerifyLatestFailure(payload WebHookPayload) (bool, error) {
	latestRun, err := getLatestRunDetails(
		payload.Repository.Owner.Login,
		payload.Repository.Name,
		int(payload.WorkflowRun.ID),
	)
	if err != nil {
		return false, fmt.Errorf("state verification failed: %w", err)
	}

	if latestRun.Conclusion != "failure" {
		return false, nil
	}
	return true, nil
}

// -----------------------------------------------------------------------------
// EXTERNAL COMMAND HANDLERS (Unchanged)
// -----------------------------------------------------------------------------
// func (g *Gateway) DockerCommandHandler(w http.ResponseWriter, r *http.Request) {
// 	// ... [Your existing Docker handler code here] ...
// 	// Note: Leaving this out of the snippet to keep it clean, but keep your existing one!
// 	w.WriteHeader(http.StatusOK)
// }
// func (g *Gateway) GrafanaCommandHandler(w http.ResponseWriter, r *http.Request) {
// 	w.WriteHeader(http.StatusOK)
// }

var (
	TotalRequests atomic.Uint64
	Error5xxCount atomic.Uint64
)

func (g *Gateway) GrafanaStatsHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cpuPercents, err := cpu.Percent(0, false)
	realCPU := 0.0
	if err == nil && len(cpuPercents) > 0 {
		realCPU = cpuPercents[0]
	}

	total := TotalRequests.Load()
	errors := Error5xxCount.Load()
	realErrorRate := 0.0
	if total > 0 {
		realErrorRate = (float64(errors) / float64(total)) * 100.0
	}

	stats := map[string]interface{}{
		"cpu_usage":         realCPU,
		"memory_mb":         m.Alloc / 1024 / 1024,
		"error_rate":        realErrorRate,
		"active_goroutines": runtime.NumGoroutine(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// func (g *Gateway) K8sCommandHandler(w http.ResponseWriter, r *http.Request) {
// 	w.WriteHeader(http.StatusOK)
// }

func (g *Gateway) ListIncidentsHandler(w http.ResponseWriter, r *http.Request) {
	incidents := incident.Global.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incidents)
}
func (g *Gateway) GetStateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if inc, ok := incident.Global.Get(id); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(inc)
		return
	}
	http.Error(w, "incident not found", http.StatusNotFound)
}
func (g *Gateway) ApproveHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if incident.Global.Approve(req.ID) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "approved", "id": req.ID})
		return
	}
	http.Error(w, "not found", http.StatusNotFound)
}
func appendToAuditLog(incidentID, repo, errorMsg, diff string) {}
