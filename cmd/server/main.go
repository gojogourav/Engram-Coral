package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gojogourav/engram/cmd/internal/api"
	"github.com/gojogourav/engram/cmd/internal/docker"
	"github.com/gojogourav/engram/cmd/internal/grafana"
	"github.com/gojogourav/engram/cmd/internal/k8s"
	"github.com/gojogourav/engram/cmd/internal/llm"
	"github.com/gojogourav/engram/cmd/internal/store"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
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
	mux.HandleFunc("/voice/transcribe", gateway.VoiceTranscribeHandler) // ← NEW
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("GET /k8s/deployments", gateway.ListDeploymentsHandler)
	mux.HandleFunc("POST /k8s/scale", gateway.ScaleDeploymentHandler)
	mux.HandleFunc("POST /k8s/restart", gateway.RestartDeploymentHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("VoxDeploy listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
