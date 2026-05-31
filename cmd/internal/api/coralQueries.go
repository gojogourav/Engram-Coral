package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CoralRun struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HeadSHA    string `json:"head_sha"`
}

type GitHubJob struct {
	Name            string `json:"name"`
	Conclusion      string `json:"conclusion"`
	FailedStepNames string `json:"failed_step_names"`
}

func GetWorkflowFailures(owner, repo string, runID int64) ([]GitHubJob, error) {
	query := fmt.Sprintf(
		"SELECT name, conclusion, failed_step_names FROM github.jobs WHERE owner = '%s' AND repo = '%s' AND run_id = %d AND conclusion = 'failure'",
		owner, repo, runID,
	)
	raw, err := runCoralQuery(query)
	if err != nil {
		return nil, fmt.Errorf("coral execution failed: %v", err)
	}
	var jobs []GitHubJob
	if err := json.Unmarshal([]byte(raw), &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse job JSON: %v\nRaw: %s", err, raw)
	}
	return jobs, nil
}

func getLatestRunDetails(owner, repo string, workflowID int) (*CoralRun, error) {
	query := fmt.Sprintf(
		"SELECT id, status, conclusion, head_sha FROM github.repo_action_workflow_runs WHERE owner = '%s' AND repo = '%s' AND workflow_id = %d ORDER BY created_at DESC LIMIT 1",
		owner, repo, workflowID,
	)
	raw, err := runCoralQuery(query)
	if err != nil {
		return nil, fmt.Errorf("coral execution failed: %v", err)
	}
	var runs []CoralRun
	if err := json.Unmarshal([]byte(raw), &runs); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	if len(runs) == 0 {
		return nil, fmt.Errorf("no runs found")
	}
	return &runs[0], nil
}

func (g *Gateway) CoralQueryHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	var req struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	result, err := runCoralQuery(req.Query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Coral query failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(result))
}

func getFailedRun(owner, repo string, workflowID int) (string, error) {
	query := fmt.Sprintf(
		"SELECT id, conclusion, head_sha FROM github.repo_action_workflow_runs WHERE owner = '%s' AND repo = '%s' AND workflow_id = %d ORDER BY created_at DESC LIMIT 1",
		owner, repo, workflowID,
	)
	return runCoralQuery(query)
}
