package api

import (
	"encoding/json"
	"net/http"
)

type ScaleRequest struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int32  `json:"replicas"`
}

type RestartRequest struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (g *Gateway) ListDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	namespace := r.URL.Query().Get("namespace") // Optional

	cfg, ok := g.Store.Get(repo)
	if !ok || cfg.K8sClient == nil {
		http.Error(w, "Kubernetes client not available for this repo", http.StatusServiceUnavailable)
		return
	}

	deployments, err := cfg.K8sClient.GetDeployments(r.Context(), namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"repo":        repo,
		"deployments": deployments,
	})
}

func (g *Gateway) ScaleDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	cfg, ok := g.Store.Get(repo)
	if !ok || cfg.K8sClient == nil {
		http.Error(w, "Kubernetes client not available for this repo", http.StatusServiceUnavailable)
		return
	}

	var req ScaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := cfg.K8sClient.ScaleDeployment(r.Context(), req.Namespace, req.Name, req.Replicas); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Deployment scaled successfully",
	})
}

func (g *Gateway) RestartDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	cfg, ok := g.Store.Get(repo)
	if !ok || cfg.K8sClient == nil {
		http.Error(w, "Kubernetes client not available for this repo", http.StatusServiceUnavailable)
		return
	}

	var req RestartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := cfg.K8sClient.RestartDeployment(r.Context(), req.Namespace, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Deployment restarted successfully",
	})
}
