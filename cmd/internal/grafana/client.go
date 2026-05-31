package grafana

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gojogourav/engram/cmd/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: httpClient,
	}
}

// -----------------------------------------------------------------------------
// CORAL SQL ENGINE INTEGRATION (With Rust Panic Bypass)
// -----------------------------------------------------------------------------

func (c *Client) CoralQuery(sqlQuery string) ([]map[string]interface{}, error) {
	// Execute: coral sql "<query>" --format json
	cmd := exec.Command("coral", "sql", sqlQuery, "--format", "json")

	out, err := cmd.CombinedOutput()
	raw := string(out)

	// HACKATHON FIX: Strip Rust panic noise by finding the JSON array boundaries
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")

	var cleanJSON []byte
	if start != -1 && end != -1 && end >= start {
		cleanJSON = []byte(raw[start : end+1])
	} else {
		// If there are no brackets, it's either an empty response or a fatal error
		if err != nil {
			return nil, fmt.Errorf("coral execution failed: %v | output: %s", err, raw)
		}
		return []map[string]interface{}{}, nil
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(cleanJSON, &results); err != nil {
		return nil, fmt.Errorf("failed to parse coral JSON output: %w | clean json: %s", err, string(cleanJSON))
	}

	return results, nil
}

// -----------------------------------------------------------------------------
// INTERNAL PROMETHEUS METRICS (Ultra-efficient O(1) memory reads)
// -----------------------------------------------------------------------------

type EngramStats struct {
	WebhooksTotal  float64 `json:"webhooks_total"`
	FixesGenerated float64 `json:"fixes_generated"`
	PRsOpened      float64 `json:"prs_opened"`
	FixesFailed    float64 `json:"fixes_failed"`
	SuccessRate    float64 `json:"success_rate"`
	AvgAILatency   float64 `json:"avg_ai_latency"`
	P95AILatency   float64 `json:"p95_ai_latency"`
	DiffErrors     float64 `json:"diff_errors"`
}

func getCounterValue(c prometheus.Counter) float64 {
	m := &dto.Metric{}
	c.Write(m)
	return m.GetCounter().GetValue()
}

func getHistogramSum(h prometheus.Histogram) (float64, float64) {
	m := &dto.Metric{}
	h.Write(m)
	hist := m.GetHistogram()
	return hist.GetSampleSum(), float64(hist.GetSampleCount())
}

func (c *Client) GetDashboardStats() (*EngramStats, error) {
	stats := &EngramStats{}

	stats.WebhooksTotal = getCounterValue(metrics.WebHookRecieved)
	stats.FixesGenerated = getCounterValue(metrics.FixesGenerated)
	stats.PRsOpened = getCounterValue(metrics.PRsOpened)
	stats.FixesFailed = getCounterValue(metrics.FixFailed)
	stats.DiffErrors = getCounterValue(metrics.DiffApplyErrors)

	sum, count := getHistogramSum(metrics.AILatency)
	if count > 0 {
		stats.AvgAILatency = sum / count
	}

	if stats.WebhooksTotal > 0 {
		stats.SuccessRate = (stats.PRsOpened / stats.WebhooksTotal) * 100
	}

	return stats, nil
}

// -----------------------------------------------------------------------------
// CHATOPS NLP INTERPRETER
// -----------------------------------------------------------------------------

func (c *Client) InterpretCommand(translatedText string) (string, error) {
	text := strings.ToLower(strings.TrimSpace(translatedText))

	// Fast-path internal memory stats
	switch {
	case contains(text, "summary", "overview", "status", "how are we doing"):
		stats, _ := c.GetDashboardStats()
		return fmt.Sprintf(
			"VoxDeploy caught %.0f CI failures, generated %.0f fixes, and opened %.0f PRs. Success rate: %.1f%%. Avg AI latency: %.1fs.",
			stats.WebhooksTotal, stats.FixesGenerated, stats.PRsOpened, stats.SuccessRate, stats.AvgAILatency,
		), nil

	case contains(text, "dashboards", "find dashboards"):
		// EXAMPLE: Using Coral to search Grafana dynamically
		results, err := c.CoralQuery("SELECT uid, title FROM grafana.dashboards LIMIT 3")
		if err != nil {
			return "", err
		}

		var sb strings.Builder
		sb.WriteString("Here are your active Grafana Dashboards:\n")
		for _, row := range results {
			sb.WriteString(fmt.Sprintf("- %v (UID: %v)\n", row["title"], row["uid"]))
		}
		return sb.String(), nil

	case contains(text, "alerts", "firing"):
		// EXAMPLE: Using Coral to find firing alerts
		results, err := c.CoralQuery("SELECT name, state FROM grafana.alerts WHERE state = 'firing'")
		if err != nil {
			return "", err
		}
		if len(results) == 0 {
			return "✅ All systems stable. No firing alerts in Grafana.", nil
		}

		return fmt.Sprintf("⚠️ Found %d active alerts currently firing in Grafana.", len(results)), nil
	}

	return "", fmt.Errorf("could not interpret command: %q", translatedText)
}

func contains(text string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}
