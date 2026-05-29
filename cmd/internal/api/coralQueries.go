package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

func getFailedRun(owner, repo string, workflowID int) (string, error) {
	query := fmt.Sprintf(
		"SELECT id, conclusion, head_sha FROM github.repo_action_workflow_runs WHERE owner = '%s' AND repo = '%s' AND workflow_id = %d ORDER BY created_at DESC LIMIT 1",
		owner, repo, workflowID,
	)

	cmd := exec.Command("coral", "sql", query)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Coral execution failed: %v | Output: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

type CoralRun struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HeadSHA    string `json:"head_sha"`
}

func getLatestRunDetails(owner, repo string, workflowID int) (*CoralRun, error) {
	query := fmt.Sprintf(
		"SELECT id, status, conclusion, head_sha FROM github.repo_action_workflow_runs WHERE owner = '%s' AND repo = '%s' AND workflow_id = %d ORDER BY created_at DESC LIMIT 1",
		owner, repo, workflowID,
	)

	cmd := exec.Command("coral", "sql", query, "--format", "json")
	stdout, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("coral execution failed: %v", err)
	}

	var runs []CoralRun
	if err := json.Unmarshal(stdout, &runs); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	if len(runs) == 0 {
		return nil, fmt.Errorf("no runs found")
	}

	return &runs[0], nil
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

	cmd := exec.Command("coral", "sql", query, "--format", "json")

	stdout, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("coral execution failed: %v", err)
	}

	var jobs []GitHubJob
	if err := json.Unmarshal(stdout, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse job JSON: %v", err)
	}

	return jobs, nil
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

	cmd := exec.Command("coral", "sql", req.Query, "--format", "json")
	stdout, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Coral query failed: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(stdout)
}
