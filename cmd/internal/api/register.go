package api

import (
	"encoding/json"
	"net/http"

	"github.com/gojogourav/engram/cmd/internal/store"
)

type RegisterRequest struct {
	Repo          string `json:"repo"`
	GithubToken   string `json:"github_token"`
	WebhookSecret string `json:"webhook_secret"`
	KubeconfigB64 string `json:"kubeconfig_b64"`
	DockerHost    string `json:"docker_host"`
	GrafanaURL    string `json:"grafana_url"`
	GrafanaToken  string `json:"grafana_token"`
}

func (g *Gateway) RegisterRepoHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	// body, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	http.Error(w, "cannot read body", http.StatusBadRequest)
	// 	return
	// }

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}
	if req.Repo == "" || req.GithubToken == "" || req.WebhookSecret == "" {
		http.Error(w, "repo, github_token and webhook_secret are required", http.StatusBadRequest)
		return
	}

	cfg, err := store.BuildRepoConfig(
		req.GithubToken,
		req.WebhookSecret,
		req.KubeconfigB64,
		req.DockerHost,
		req.GrafanaURL,
		req.GrafanaToken,
		g.HTTPClient,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	g.Store.Register(req.Repo, cfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "registered",
		"repo":   req.Repo,
	})
}

func (g *Gateway) ListReposHandler(w http.ResponseWriter, r *http.Request) {
	repos := g.Store.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"repos": repos,
	})
}
