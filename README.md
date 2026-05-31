# Engram — Autonomous SRE Platform

> CI breaks. Engram fixes it. Automatically.

Engram is an AI-powered Site Reliability Engineering platform that detects failed GitHub Actions workflows, diagnoses the root cause using Gemini 2.5 Flash, generates a code fix, and opens a pull request — all without human intervention. It exposes a ChatOps interface for natural-language Kubernetes operations and an Observability agent that reasons over live infrastructure data using the **Coral federated SQL data fabric**.

---

## How It Works

```
GitHub Push → CI Fails → Webhook → Engram
                                      │
                          ┌───────────┴────────────┐
                          │   Coral SQL Queries     │
                          │  github.commits         │
                          │  github.jobs            │
                          │  github.workflows       │
                          └───────────┬────────────┘
                                      │
                             Gemini 2.5 Flash
                             (LogParser + FixGenerator)
                                      │
                             Git Diff Generated
                                      │
                             Human Approval Gate
                                      │
                             Fix Branch + PR Opened
```

---

## The Role of Coral

**Coral is the data layer that makes Engram possible.**

Coral is a federated SQL engine that exposes external APIs — GitHub, Grafana, Kubernetes — as queryable SQL tables. Instead of writing API integration code for each data source, Engram issues plain SQL queries and Coral handles authentication, pagination, rate limiting, and response normalization transparently.

### Why Coral instead of direct API calls

| Without Coral | With Coral |
|---|---|
| 3 separate GitHub API endpoints to find a failed commit | One SQL query: `SELECT sha FROM github.commits WHERE repo = 'x'` |
| Manual pagination through workflow runs | `LIMIT 10` clause |
| Separate auth token management per service | Single Coral connection |
| 50+ lines of Go HTTP client code | 1 line of SQL |
| Rigid schema tied to API response shape | Uniform column-based access |

### Coral tables used in Engram

```sql
-- Find which commit triggered a failure
SELECT sha, author__login, commit__message, commit__author__date
FROM github.commits
WHERE owner = 'gojogourav' AND repo = 'engram-test-repo'
LIMIT 10;

-- Get the workflow ID for a repo
SELECT id, name
FROM github.workflows
WHERE owner = 'gojogourav' AND repo = 'engram-test-repo';

-- Get recent workflow run outcomes
SELECT id, workflow_id, status, conclusion, head_sha, created_at
FROM github.repo_action_workflow_runs
WHERE owner = 'gojogourav' AND repo = 'engram-test-repo'
  AND workflow_id = 123456789
LIMIT 5;

-- Find which job step failed
SELECT name, conclusion, failed_step_names
FROM github.jobs
WHERE owner = 'gojogourav' AND repo = 'engram-test-repo'
  AND run_id = 987654321;

-- Blast radius: enumerate all repos
SELECT full_name, private
FROM github.user_repos
LIMIT 20;
```

### Coral in the autonomous healing pipeline

When a CI failure webhook arrives, Engram uses Coral to build the **blast radius context** before calling Gemini:

```go
// cmd/internal/api/gateway.go
query := fmt.Sprintf(
    "SELECT sha, author__login, commit__message FROM github.commits WHERE owner = '%s' AND repo = '%s' AND sha = '%s' LIMIT 1",
    owner, repo, sha,
)
result, err := runCoralQuery(query)
```

This commit context — author, message, timestamp — is injected into the Gemini prompt alongside the raw CI logs. The model uses the combined context to generate a more accurate fix.

### Coral in the Observability Agent

The Observability page features a **Gemini tool-calling agent** where `run_coral_query` is registered as a tool. The agent autonomously decides which SQL queries to run, executes them via Coral, and reasons over the results:

```
User: "Are the last 3 failures all failing at the same step?"

Agent turn 1 → run_coral_query("SELECT id FROM github.workflows WHERE repo='engram-test-repo'")
Agent turn 2 → run_coral_query("SELECT id FROM github.repo_action_workflow_runs WHERE workflow_id=123 LIMIT 3")
Agent turn 3 → run_coral_query("SELECT name, failed_step_names FROM github.jobs WHERE run_id=456")
Agent turn 4 → "Yes — all three failed at the 'Run Tests' step. This is a systemic issue."
```

Three REST API endpoints. Zero API client code. Coral handles everything.

---

## Architecture

```
engram/
├── cmd/
│   ├── main.go                        # Entry point, ngrok tunnel, route registration
│   └── internal/
│       ├── api/
│       │   ├── gateway.go             # Webhook handler, autonomous healing pipeline
│       │   ├── agent.go               # Gemini tool-calling agent (Observability)
│       │   ├── handlers_fix.go        # K8s / Docker / Grafana command handlers
│       │   ├── k8s_handlers.go        # Scale, restart, list deployments
│       │   ├── coral.go               # Coral query execution + JSON cleanup
│       │   ├── auth.go                # GitHub webhook HMAC verification
│       │   ├── prometheus_handler.go  # Alertmanager webhook → self-healing
│       │   ├── register.go            # Repo registration handler
│       │   └── transcribe.go          # Groq Whisper voice transcription
│       ├── llm/
│       │   ├── client.go              # Gemini GenerateContent wrapper
│       │   └── prompts.go             # LogParser + FixGenerator system prompts
│       ├── k8s/
│       │   ├── client.go              # Kubernetes client initialisation
│       │   └── commands.go            # InterpretCommand + ExecuteCommand
│       ├── docker/
│       │   ├── client.go              # Docker client
│       │   ├── commands.go            # Container operations
│       │   └── containers.go          # List, stop, start, restart, logs
│       ├── grafana/
│       │   └── client.go              # Prometheus metrics + Coral SQL bridge
│       ├── incident/
│       │   └── store.go               # In-memory incident state machine
│       ├── store/
│       │   └── repo_store.go          # Per-repo config registry
│       ├── metrics/
│       │   └── prometheus.go          # Custom Prometheus counters + histograms
│       └── github/
│           ├── content.go             # Fetch file content from GitHub API
│           ├── logs.go                # Download + extract CI log ZIPs
│           ├── tree.go                # Repo file structure walker
│           └── pr.go                  # Branch creation + file update + PR open
├── internal/
│   └── diff/                          # Unified diff parser + applier
└── frontend/                          # Next.js 14 App Router
    └── app/
        ├── page.tsx                   # Mission Control (fleet overview)
        ├── war-room/page.tsx          # Live incident pipeline tracker
        ├── chatops/page.tsx           # Observability agent (Gemini + Coral)
        ├── chat/page.tsx              # ChatOps (K8s / Docker / Grafana commands)
        └── settings/page.tsx          # Repo + infrastructure registration
```

---

## Features

### Autonomous CI Healing
- Listens for `workflow_run` GitHub webhooks
- Downloads and filters CI log ZIPs
- Uses Coral to pull commit context and job failure details
- Sends enriched context to Gemini LogParser to identify broken files
- Generates a unified git diff via Gemini FixGenerator
- Applies the diff, creates a fix branch, opens a PR
- Confidence scoring based on number of jobs, files patched

### Human Approval Gate
- All fixes pause at `pending_approval` before touching the repo
- Approval via the War Room UI or `/approve` chat command
- 10-minute timeout — auto-rejects if no human responds

### Observability Agent
- Gemini 2.5 Flash with tool-calling (`run_coral_query`, `k8s_command`)
- Multi-step reasoning loop (up to 6 turns)
- Translates natural language questions into multi-hop Coral SQL
- Falls back gracefully — surfaces raw data if turn limit is hit
- Schema-aware system prompt prevents hallucinated column names

### ChatOps
- `/k8s` — list pods, list deployments, scale, restart, get logs, show crashing pods
- `/docker` — list, stop, start, restart containers, get logs
- `/grafana` — summary stats, firing alerts, dashboard list
- `/incident` — view incident state
- `/approve` — approve a pending fix
- Free-text queries route to the Gemini agent automatically

### Self-Healing via Prometheus Alerts
- Alertmanager webhook endpoint at `/alerts/prometheus`
- Receives `OOMKilled`, `CrashLoopBackOff`, `HighCPU` alerts
- Fetches live pod JSON via kubectl
- AI decides whether to restart or scale the deployment
- Executes the remediation autonomously

### Voice Interface
- Groq Whisper (`whisper-large-v3`) via `/voice/transcribe`
- Base64 audio → transcript + detected language
- Frontend mic button for hands-free ChatOps

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22 |
| AI | Gemini 2.5 Flash (`google.golang.org/genai`) |
| Data Fabric | **Coral** (federated SQL over GitHub + Grafana) |
| Kubernetes | `k8s.io/client-go` |
| Docker | `github.com/moby/moby/client` |
| Metrics | Prometheus + `promhttp` |
| Tunnel | ngrok Go SDK |
| Voice | Groq Whisper API |
| Frontend | Next.js 14, TypeScript, Tailwind CSS |
| Diff Engine | Custom unified diff parser + applier |

---

## Setup

### Prerequisites
- Go 1.22+
- Node.js 18+
- A running Kubernetes cluster (kind, minikube, or real)
- Docker (optional)
- Coral CLI installed and authenticated
- ngrok account

### Environment variables

```bash
# .env
GITHUB_TOKEN=ghp_xxxxxxxxxxxx        # needs repo + workflow permissions
WEBHOOK_SECRET=your_secret_here
GEMINI_API_KEY=your_gemini_key
GROQ_API_KEY=your_groq_key           # optional, for voice
NGROK_AUTHTOKEN=your_ngrok_token
```

### Run

```bash
# Backend
cd cmd
go run main.go

# Frontend (separate terminal)
cd frontend
npm install
npm run dev
```

The backend will:
1. Start the Go server on `:8080`
2. Spin up an ngrok tunnel
3. Auto-register the webhook with GitHub

Open `http://localhost:3000` for the frontend.

---

## API Reference

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/webhook` | GitHub `workflow_run` webhook |
| `POST` | `/api/agent/chat` | Observability agent (Gemini + Coral) |
| `POST` | `/k8s/command` | Natural language K8s operations |
| `POST` | `/docker/command` | Natural language Docker operations |
| `POST` | `/grafana/command` | Grafana / stats queries |
| `POST` | `/coral/query` | Raw Coral SQL execution |
| `POST` | `/api/approve` | Approve a pending incident fix |
| `GET`  | `/api/incidents` | List all incidents |
| `GET`  | `/api/state?id=INC-xxx` | Get single incident state |
| `GET`  | `/api/stats` | Prometheus + incident metrics |
| `POST` | `/repos/register` | Register a repo with its config |
| `POST` | `/alerts/prometheus` | Alertmanager webhook |
| `POST` | `/voice/transcribe` | Groq Whisper transcription |
| `GET`  | `/metrics` | Prometheus scrape endpoint |
| `GET`  | `/health` | Health check |

---

## Prometheus Metrics

| Metric | Type | Description |
|---|---|---|
| `engram_webhooks_received_total` | Counter | CI failure webhooks received |
| `engram_prs_opened_total` | Counter | Pull requests opened by Engram |
| `engram_fix_failed_total` | Counter | Pipeline failures at any stage |
| `engram_diff_apply_errors_total` | Counter | Diff parse/apply failures |
| `engram_ai_generation_seconds` | Histogram | Gemini response latency |
| `engram_pipeline_duration_seconds` | Histogram | End-to-end fix pipeline latency |

---

## Incident Lifecycle

```
detecting → aggregating → diagnosing → sandboxing → pending_approval → healing → healed
                                                                              ↓
                                                                           failed
```

Each stage is persisted in the in-memory `incident.Store` and polled by the War Room frontend every 1.5 seconds.

---

## License

MIT
