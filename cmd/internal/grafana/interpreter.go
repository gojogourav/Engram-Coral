package grafana

import (
	"fmt"
	"strings"
)

func (c *Client) InterpretCommand(translatedText string) (string, error) {
	text := strings.ToLower(strings.TrimSpace(translatedText))
	switch {
	case contains(text, "summary", "overview", "status", "how are we doing"):
		return c.GetMetricSummary()
	case contains(text, "webhooks", "ci failures", "builds failed"):
		return c.querySingle("voxdeploy_webhooks_received_total", "Total CI failures caught")
	case contains(text, "prs", "pull requests"):
		return c.querySingle("voxdeploy_prs_opened_total", "Total PRs opened")
	case contains(text, "success rate"):
		stats, err := c.GetDashboardStats()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Success rate: %.1f%%", stats.SuccessRate), nil
	case contains(text, "latency", "how fast", "ai speed", "speed of ai", "speed", "fast"):
		stats, err := c.GetDashboardStats()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Average AI latency: %.1fs, p95: %.1fs", stats.AvgAILatency, stats.P95AILatency), nil
	case contains(text, "errors", "failures", "failed"):
		return c.querySingle("voxdeploy_fix_failed_total", "Total pipeline failures")
	default:
		return "", fmt.Errorf("could not interpret grafana command: %q", translatedText)
	}
}
func (c *Client) querySingle(promQL, label string) (string, error) {
	results, err := c.Query(promQL)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return fmt.Sprintf("%s: 0", label), nil
	}
	return fmt.Sprintf("%s: %.0f", label, results[0].Value), nil
}

func contains(text string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}
