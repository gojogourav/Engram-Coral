package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gojogourav/engram/cmd/internal/api"
	"github.com/gojogourav/engram/cmd/internal/docker"
	"github.com/gojogourav/engram/cmd/internal/grafana"
	"github.com/gojogourav/engram/cmd/internal/k8s"
	"github.com/gojogourav/engram/cmd/internal/llm"
	"github.com/gojogourav/engram/cmd/internal/store"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"

	"github.com/joho/godotenv" // ← NEW
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func autoRegisterWebhook(owner, repo, token, secret, webhookURL string) error {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/hooks", owner, repo)

	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"workflow_run"},
		"config": map[string]string{
			"url":          webhookURL,
			"content_type": "json",
			"secret":       secret,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		log.Println(" Successfully registered webhook with GitHub API!")
		return nil
	} else if resp.StatusCode == 422 {
		log.Println(" Webhook already exists for this URL. Proceeding...")
		return nil
	}

	return fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
}

func main() {
	// ← NEW: Load the .env file into memory before doing anything else
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found. Falling back to system variables.")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	webhookSecret := os.Getenv("WEBHOOK_SECRET")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	groqAPIKey := os.Getenv("GROQ_API_KEY")

	if githubToken == "" || webhookSecret == "" || geminiAPIKey == "" {
		log.Fatal("missing required env vars: GITHUB_TOKEN, WEBHOOK_SECRET, GEMINI_API_KEY")
	}

	httpClient := &http.Client{}

	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Printf("K8s not available: %v", err)
	}
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Printf("Docker not available: %v", err)
	}
	grafanaClient := grafana.NewClient("http://localhost:9090", "", httpClient)

	repoStore := store.NewRepoStore()
	repoStore.Register("yujiblack/Vortex-Test", &store.RepoConfig{
		GithubToken:   githubToken,
		WebhookSecret: webhookSecret,
		K8sClient:     k8sClient,
		DockerClient:  dockerClient,
		GrafanaClient: grafanaClient,
	})

	gateway := &api.Gateway{
		GithubToken:   githubToken,
		WebHookSecret: webhookSecret,
		HTTPClient:    httpClient,
		LLMClient:     &llm.Client{APIKey: geminiAPIKey, HTTPClient: httpClient},
		DockerClient:  dockerClient,
		GrafanaClient: grafanaClient,
		K8sClient:     k8sClient,
		Store:         repoStore,
		GroqAPIKey:    groqAPIKey,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", gateway.WebHookHandler)
	mux.HandleFunc("/docker/command", gateway.DockerCommandHandler)
	mux.HandleFunc("/grafana/command", gateway.GrafanaCommandHandler)
	mux.HandleFunc("/grafana/stats", gateway.GrafanaStatsHandler)
	mux.HandleFunc("/k8s/command", gateway.K8sCommandHandler)
	mux.HandleFunc("/repos/register", gateway.RegisterRepoHandler)
	mux.HandleFunc("/repos", gateway.ListReposHandler)
	mux.HandleFunc("/voice/transcribe", gateway.VoiceTranscribeHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/alerts/prometheus", gateway.PrometheusAlertHandler)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("GET /k8s/deployments", gateway.ListDeploymentsHandler)
	mux.HandleFunc("POST /k8s/scale", gateway.ScaleDeploymentHandler)
	mux.HandleFunc("POST /k8s/restart", gateway.RestartDeploymentHandler)
	mux.HandleFunc("/coral/query", gateway.CoralQueryHandler)
	// port := os.Getenv("PORT")
	// if port == "" {
	// 	port = "8080"
	// }
	// log.Printf("VoxDeploy listening on port %s", port)
	// if err := http.ListenAndServe(":"+port, mux); err != nil {
	// 	log.Fatalf("server failed: %v", err)
	// }

	fmt.Println("Spinning ngrok tunnel")
	ctx := context.Background()
	listner, err := ngrok.Listen(ctx,
		config.HTTPEndpoint(),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		log.Fatalf("Failed to spin up ngrok tunnel: %v", err)
	}

	publicURL := listner.URL()
	log.Printf("Secure tunnel established at: %s", publicURL)
	log.Printf("Your Webhook Endpoint: %s/webhook", publicURL)

	webhookEndpoint := publicURL + "/webhook"
	err = autoRegisterWebhook("gojogourav", "engram-test-repo", githubToken, webhookSecret, webhookEndpoint)
	if err != nil {
		log.Printf("Failed to auto-register webhook: %v", err)
	}

	// 2. Serve your exact same HTTP multiplexer
	if err := http.Serve(listner, mux); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
