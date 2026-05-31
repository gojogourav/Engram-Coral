package api

// handlers_fix.go — Replace the stub implementations in gateway.go
// Drop this file in cmd/internal/api/ and delete the stub bodies from gateway.go

import (
	"encoding/json"
	"net/http"

	"github.com/gojogourav/engram/cmd/internal/docker"
	"github.com/gojogourav/engram/cmd/internal/k8s"
)

// ─── shared request/response types ───────────────────────────────────────────

type commandRequest struct {
	Text string `json:"text"`
}

type commandResponse struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func writeCommandResult(w http.ResponseWriter, result string, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(commandResponse{Error: err.Error()})
		return
	}
	json.NewEncoder(w).Encode(commandResponse{Result: result})
}

// ─── K8s ──────────────────────────────────────────────────────────────────────

func (g *Gateway) K8sCommandHandler(w http.ResponseWriter, r *http.Request) {
	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		http.Error(w, "missing text field", http.StatusBadRequest)
		return
	}

	if g.K8sClient == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(commandResponse{
			Error: "Kubernetes client not initialised — check KUBECONFIG or in-cluster config",
		})
		return
	}

	cmd, err := k8s.InterpretCommand(req.Text)
	if err != nil {
		writeCommandResult(w, "", err)
		return
	}

	result, err := g.K8sClient.ExecuteCommand(cmd)
	writeCommandResult(w, result, err)
}

// ─── Docker ───────────────────────────────────────────────────────────────────

func (g *Gateway) DockerCommandHandler(w http.ResponseWriter, r *http.Request) {
	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		http.Error(w, "missing text field", http.StatusBadRequest)
		return
	}

	if g.DockerClient == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(commandResponse{
			Error: "Docker client not initialised — is Docker running?",
		})
		return
	}

	cmd, err := docker.InterpretCommand(req.Text)
	if err != nil {
		writeCommandResult(w, "", err)
		return
	}

	result, err := g.DockerClient.ExecuteCommand(cmd)
	writeCommandResult(w, result, err)
}

// ─── Grafana ──────────────────────────────────────────────────────────────────

func (g *Gateway) GrafanaCommandHandler(w http.ResponseWriter, r *http.Request) {
	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		http.Error(w, "missing text field", http.StatusBadRequest)
		return
	}

	if g.GrafanaClient == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(commandResponse{
			Error: "Grafana client not initialised",
		})
		return
	}

	result, err := g.GrafanaClient.InterpretCommand(req.Text)
	writeCommandResult(w, result, err)
}
