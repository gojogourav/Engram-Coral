package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gojogourav/engram/cmd/internal/k8s"
	"google.golang.org/genai"
)

type AgentRequest struct {
	Query string `json:"query"`
}

type AgentResponse struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (g *Gateway) AgentChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	reply := g.runConversationalAgent(req.Query)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AgentResponse{Result: reply})
}

func (g *Gateway) runConversationalAgent(userPrompt string) string {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.LLMClient.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Printf("Agent: failed to init client: %v", err)
		return "Internal error: could not initialise AI client."
	}

	tools := []*genai.Tool{{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "run_coral_query",
				Description: "Execute a SQL query against the Coral data fabric.",
				Parameters: &genai.Schema{
					Type:     genai.TypeObject,
					Required: []string{"sql"},
					Properties: map[string]*genai.Schema{
						"sql": {Type: genai.TypeString, Description: "The SQL query to execute."},
					},
				},
			},
			{
				Name:        "k8s_command",
				Description: "Execute a Kubernetes operation in natural language.",
				Parameters: &genai.Schema{
					Type:     genai.TypeObject,
					Required: []string{"command"},
					Properties: map[string]*genai.Schema{
						"command": {Type: genai.TypeString, Description: "Natural language K8s command."},
					},
				},
			},
		},
	}}

	// ── System prompt ─────────────────────────────────────────────────────────
	// Key design decisions:
	// 1. Hard-code the known-good repo so the model never wastes turns searching
	// 2. Give exact working query templates for the most common tasks
	// 3. Explain the [GET] bug so the model doesn't retry bad queries
	// 4. Limit scope: answer from data you have, don't keep digging

	systemPrompt := `You are Engram, an SRE assistant. You have two tools: run_coral_query and k8s_command.

═══════════════════════════════════════════════
CRITICAL FACTS — MEMORISE THESE, NEVER DEVIATE
═══════════════════════════════════════════════
• The ONLY active repo is: owner='gojogourav', repo='engram-test-repo'
• Do NOT search for workflows or repos. The workflow name is 'CI'.
• To get failed jobs, query github.jobs directly with a known run_id.
• If a query returns "[GET]" it means that table requires a mandatory filter you are missing. Do not retry the same query.

═════════════════════════
CORAL SCHEMA (exact only)
═════════════════════════
github.commits:
  SELECT sha, author__login, commit__message, commit__author__date
  FROM github.commits
  WHERE owner='gojogourav' AND repo='engram-test-repo'
  LIMIT 10

github.repo_action_workflow_runs:
  SELECT id, workflow_id, status, conclusion, head_sha, created_at
  FROM github.repo_action_workflow_runs
  WHERE owner='gojogourav' AND repo='engram-test-repo' AND workflow_id=<NUMBER>
  LIMIT 10
  ⚠ workflow_id MUST be a known integer constant. The 'CI' workflow id is typically in the 100M range — fetch it from github.workflows first.

github.workflows:
  SELECT id, name
  FROM github.workflows
  WHERE owner='gojogourav' AND repo='engram-test-repo'
  LIMIT 10
  ✓ This is the ONLY table that does NOT require extra filters.

github.jobs:
  SELECT name, conclusion, failed_step_names
  FROM github.jobs
  WHERE owner='gojogourav' AND repo='engram-test-repo' AND run_id=<NUMBER>
  ⚠ run_id MUST be a known integer from a workflow run.

github.user_repos:
  SELECT full_name, private
  FROM github.user_repos
  LIMIT 10
  (No WHERE clause needed)

══════════════
STRICT RULES
══════════════
1. NO JOINs, NO subqueries.
2. Always LIMIT 10 or fewer.
3. If a query returns "[]" → no data exists, tell the user.
4. If a query returns "[GET]" → you used a table without a required filter. Stop and tell the user what filter is needed.
5. Answer ONLY from data you actually received. Never invent results.
6. You have a MAX of 6 tool calls. Spend them wisely. A typical task needs 2-3.

══════════════════
OPTIMAL WORKFLOWS
══════════════════
"show failed jobs" → 2 steps:
  1. SELECT id FROM github.workflows WHERE owner='gojogourav' AND repo='engram-test-repo'
  2. SELECT id, conclusion FROM github.repo_action_workflow_runs WHERE owner='gojogourav' AND repo='engram-test-repo' AND workflow_id=<id from step 1> LIMIT 5
  3. SELECT name, conclusion, failed_step_names FROM github.jobs WHERE owner='gojogourav' AND repo='engram-test-repo' AND run_id=<id from step 2 where conclusion='failure'>

"recent commits" → 1 step:
  SELECT sha, author__login, commit__message FROM github.commits WHERE owner='gojogourav' AND repo='engram-test-repo' LIMIT 5

K8s cluster has: broken-app, coredns, local-path-provisioner`

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		},
		Tools: tools,
	}

	session, err := client.Chats.Create(ctx, "gemini-2.5-flash", config, nil)
	if err != nil {
		log.Printf("Agent: failed to create session: %v", err)
		return "Internal error: could not start AI session."
	}

	// ── Agentic loop ─────────────────────────────────────────────────────────

	nextPart := genai.Part{Text: userPrompt}
	var gathered []string // collects intermediate findings to show if we hit the limit

	for turn := 0; turn < 6; turn++ {
		resp, err := session.SendMessage(ctx, nextPart)
		if err != nil {
			log.Printf("Agent turn %d error: %v", turn+1, err)
			return fmt.Sprintf("AI error on turn %d: %v\n\nPartial findings:\n%s",
				turn+1, err, strings.Join(gathered, "\n"))
		}

		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			break
		}

		parts := resp.Candidates[0].Content.Parts
		var textBuf strings.Builder
		var toolCall *genai.FunctionCall

		for _, p := range parts {
			if p.Text != "" {
				textBuf.WriteString(p.Text)
			}
			if p.FunctionCall != nil && toolCall == nil {
				toolCall = p.FunctionCall
			}
		}

		// No tool call → done
		if toolCall == nil {
			answer := strings.TrimSpace(textBuf.String())
			if answer == "" {
				if len(gathered) > 0 {
					return "Here's what I found:\n\n" + strings.Join(gathered, "\n\n")
				}
				return "Done — no output."
			}
			return answer
		}

		// Execute tool
		log.Printf("Agent turn %d: tool=%q", turn+1, toolCall.Name)
		toolResult := g.dispatchTool(toolCall)
		log.Printf("  → preview: %s", agentTruncate(toolResult, 200))

		// Stash interesting results so we can surface them if we hit the turn limit
		if toolResult != "" && toolResult != "[]" && toolResult != "[GET]" {
			gathered = append(gathered, fmt.Sprintf("**%s result:**\n```\n%s\n```",
				toolCall.Name, agentTruncate(toolResult, 500)))
		}

		nextPart = genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name:     toolCall.Name,
				Response: map[string]any{"result": toolResult},
			},
		}
	}

	// Hit the limit — return whatever we gathered rather than a useless error
	if len(gathered) > 0 {
		return "I reached the reasoning limit, but here's the raw data I collected:\n\n" +
			strings.Join(gathered, "\n\n") +
			"\n\nTip: ask a more specific question or specify the run_id directly."
	}
	return "I reached the reasoning limit without finding useful data. " +
		"Try: \"show recent commits\" or \"list my repos\" for simpler queries."
}

// dispatchTool executes a tool call and returns the result string.
func (g *Gateway) dispatchTool(call *genai.FunctionCall) string {
	switch call.Name {

	case "run_coral_query":
		sql, _ := call.Args["sql"].(string)
		if sql == "" {
			return "error: sql argument was empty"
		}
		log.Printf("  SQL: %s", sql)
		raw, err := runCoralQuery(sql)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		// Pretty-print so the model can parse it cleanly
		var parsed any
		if jsonErr := json.Unmarshal([]byte(raw), &parsed); jsonErr == nil {
			if pretty, pErr := json.MarshalIndent(parsed, "", "  "); pErr == nil {
				raw = string(pretty)
			}
		}
		if len(raw) > 4000 {
			raw = raw[:4000] + "\n...[truncated]"
		}
		return raw

	case "k8s_command":
		command, _ := call.Args["command"].(string)
		if command == "" {
			return "error: command argument was empty"
		}
		log.Printf("  K8s: %s", command)
		if g.K8sClient == nil {
			return "error: Kubernetes client not initialised"
		}
		cmd, err := k8s.InterpretCommand(command)
		if err != nil {
			return fmt.Sprintf("error: could not interpret %q: %v", command, err)
		}
		result, err := g.K8sClient.ExecuteCommand(cmd)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return result

	default:
		return fmt.Sprintf("error: unknown tool %q", call.Name)
	}
}

func agentTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
