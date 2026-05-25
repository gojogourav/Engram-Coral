package github

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ContentResponse struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func FetchFileContent(owner, repo, path, githubToken string, client *http.Client) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("github api request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch file (status %d): %s", resp.StatusCode, string(respBody))
	}

	var contentResp ContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&contentResp); err != nil {
		return "", fmt.Errorf("failed to decode github response: %w", err)
	}

	if contentResp.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(contentResp.Content)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 content: %w", err)
		}
		return string(decoded), nil
	}

	return contentResp.Content, nil

}
