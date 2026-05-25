package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"google.golang.org/genai"
)

type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

func (c *Client) FixGenerator(translatedCommand, errorLog, repoMap string, fileContexts map[string]string) (string, error) {
	ctx := context.Background()
	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: c.APIKey})
	if err != nil {
		return "", fmt.Errorf("failed to create genai client: %w", err)
	}

	var sourceCodeBuilder strings.Builder
	for path, code := range fileContexts {
		sourceCodeBuilder.WriteString(fmt.Sprintf("\n--- FILE: %s ---\n%s\n", path, code))
	}

	userPrompt := fmt.Sprintf(
		"INSTRUCTION:\n%s\n\nFILES PROVIDED:\n%s",
		translatedCommand,
		sourceCodeBuilder.String(),
	)
	temp := float32(0.0)

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(BuildFixPrompt(errorLog, repoMap), genai.RoleUser),
		Temperature:       &temp,
	}

	var result *genai.GenerateContentResponse
	for attempt := 1; attempt <= 3; attempt++ {
		result, err = genaiClient.Models.GenerateContent(
			ctx, "gemini-2.5-flash", genai.Text(userPrompt), config,
		)
		if err == nil {
			break
		}
		log.Printf("Gemini attempt %d failed: %v", attempt, err)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt*5) * time.Second)
		}
	}

	if err != nil {
		return "", fmt.Errorf("gemini fix generation failed: %w", err)
	}

	return cleanMarkdownBlocks(result.Text()), nil
}

func cleanMarkdownBlocks(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) > 2 {
			text = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	return strings.TrimSpace(text)
}

func (c *Client) LogParser(errorLog string, repoMap string) ([]string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: c.APIKey})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type:  genai.TypeArray,
			Items: &genai.Schema{Type: genai.TypeString},
		},
	}

	prompt := BuildPrompt(errorLog, repoMap)

	var result *genai.GenerateContentResponse
	for attempt := 1; attempt <= 3; attempt++ {
		result, err = client.Models.GenerateContent(
			ctx, "gemini-2.5-flash", genai.Text(prompt), config,
		)
		if err == nil {
			break
		}
		log.Printf("⚠️ Gemini attempt %d failed: %v", attempt, err)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt*5) * time.Second)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("gemini extraction failed after 3 attempts: %w", err)
	}

	var filePaths []string
	if err := json.Unmarshal([]byte(result.Text()), &filePaths); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from Gemini: %w", err)
	}

	// Validate: reject anything that looks like a diff line, not a file path
	var valid []string
	for _, fp := range filePaths {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			continue
		}
		if strings.HasPrefix(fp, "---") ||
			strings.HasPrefix(fp, "+++") ||
			strings.HasPrefix(fp, "@@") ||
			strings.HasPrefix(fp, "-\t") ||
			strings.HasPrefix(fp, "+\t") ||
			strings.Contains(fp, "\n") ||
			strings.ContainsAny(fp, " \t") && !strings.Contains(fp, "/") {
			log.Printf("⚠️ LogParser returned a non-path entry, skipping: %q", fp)
			continue
		}
		valid = append(valid, fp)
	}

	if len(valid) == 0 && len(filePaths) > 0 {
		return nil, fmt.Errorf("LogParser returned %d entries but none were valid file paths — prompt may be misconfigured", len(filePaths))
	}

	return valid, nil
}
