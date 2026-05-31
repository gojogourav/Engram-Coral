package github

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxLogBytes = 30_000

func truncateLog(log string) string {
	if len(log) <= maxLogBytes {
		return log
	}
	// Keep the END of the log — that's where errors are
	return "...[truncated]\n" + log[len(log)-maxLogBytes:]
}

func FetchAndExtractLogs(logUrl, githubToken string) (string, error) {
	req, err := http.NewRequest("GET", logUrl, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Accept", "application/vnd.github+json") // Good practice for GitHub API

	// THE FIX: Custom client that drops the auth header when following the redirect to AWS S3
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// S3 doesn't want GitHub's token. Drop it on redirect.
			req.Header.Del("Authorization")
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download logs: %w", err)
	}
	defer resp.Body.Close()

	// THE UNMASKING: Print exactly why it failed if it's not a 200 OK
	if resp.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API rejected request (Status: %d). Details: %s", resp.StatusCode, string(errorBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read zip body: %w", err)
	}

	// Your ZIP logic here is actually perfectly fine, as long as 'body' is a real ZIP file!
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return "", fmt.Errorf("failed to read zip structure (GitHub might have sent an empty/corrupted file): %w", err)
	}

	var combinedLogs bytes.Buffer
	for _, zipFile := range zipReader.File {
		func() {
			f, err := zipFile.Open()
			if err != nil {
				return
			}
			defer f.Close()

			content, err := io.ReadAll(f)
			if err == nil {
				combinedLogs.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n", zipFile.Name))
				combinedLogs.Write(content)
			}
		}()
	}

	return combinedLogs.String(), nil

}
func FilterBuildErrors(logs string) string {
	var sb strings.Builder
	// for _, line := range strings.Split(logs, "\n") {
	// 	// Keep lines that look like actual errors
	// 	if strings.Contains(line, "FAIL") ||
	// 		strings.Contains(line, "Error") ||
	// 		strings.Contains(line, "error") ||
	// 		strings.Contains(line, "undefined") ||
	// 		strings.Contains(line, "cannot") ||
	// 		strings.Contains(line, "syntax") ||
	// 		strings.Contains(line, ".go:") ||
	// 		strings.Contains(line, "--- FAIL") ||
	// 		strings.Contains(line, "FAIL\t") {
	// 		sb.WriteString(line + "\n")
	// 	}
	// }

	keywords := []string{
		".go:", "--- FAIL", "FAIL\t",
		// Python
		"Traceback (most recent call last):", "File \"", ".py\", line",
		// JavaScript / TypeScript / Node
		"ReferenceError:", "TypeError:", ".js:", ".ts:", "ERR!",
		// Universal (Works for almost anything)
		"Error", "error", "undefined", "cannot", "syntax", "fatal:",
	}

	for _, line := range strings.Split(logs, "\n") {
		for _, kw := range keywords {
			if strings.Contains(line, kw) {
				sb.WriteString(line + "\n")
				break
			}
		}
	}

	return sb.String()
}

type GitTreeResponse struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"` // "blob" is a file, "tree" is a folder
	} `json:"tree"`
}

func FileStructure(owner, repo, sha, token string, client *http.Client) (string, error) {
	// Note: We use the exact commit SHA instead of the branch to be bulletproof
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, sha)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api error, status: %d", resp.StatusCode)
	}
	var treeResp GitTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		return "", fmt.Errorf("failed to decode tree: %w", err)
	}

	var validPaths []string
	for _, item := range treeResp.Tree {
		if item.Type != "blob" {
			continue
		}
		if strings.HasPrefix(item.Path, "node_modules/") ||
			strings.HasPrefix(item.Path, "vendor/") ||
			strings.HasPrefix(item.Path, ".git/") ||
			strings.HasPrefix(item.Path, "dist/") ||
			strings.HasPrefix(item.Path, "build/") {
			continue
		}
		validPaths = append(validPaths, item.Path)
	}

	return strings.Join(validPaths, "\n"), nil
}
