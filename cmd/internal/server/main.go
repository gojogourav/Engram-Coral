package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gojogourav/engram/cmd/internal/k8s"
)

func main() {
	GITHUB_TOKEN := os.Getenv("GITHUB_TOKEN")
	WEBHOOK_SECRET := os.Getenv("WEBHOOK_SECRET")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	groqAPIKey := os.Getenv("GROQ_API_KEY")

	if GITHUB_TOKEN == "" || WEBHOOK_SECRET == "" || geminiAPIKey == "" {
		log.Fatal("missing required env vars: GITHUB_TOKEN, WEBHOOK_SECRET, GEMINI_API_KEY")
	}

	httpClient := &http.Client{}

	k8sClient, err := k8s.NewClient()

	if err != nil {
		log.Printf("K8 is not available :%v", err)
	}

	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Printf("Docker not available: %v", err)
	}
	grafanaClient := grafana.NewClient("http://localhost:9090", "", httpClient)

}
