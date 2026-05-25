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

func FetchAndExtractLogs(logUrl, githubToken string) (string, error) {
	req, err := http.NewRequest("GET", logUrl, nil)

	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.Header.Del("Authorization")
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download logs: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API rejected request (Status: %d). Details: %s", resp.StatusCode, string(errorBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read zip body: %w", err)
	}

	//  ZIP logic here is actually perfectly fine, as long as 'body' is a real ZIP file!
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

type GitTreeResponse struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"` // "blob" is a file, "tree" is a folder
	} `json:"tree"`
}

func FileStructure(owner, repo, branch, token string, client *http.Client) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, branch)

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
