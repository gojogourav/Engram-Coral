package grafana

import (
	"fmt"

	"github.com/gojogourav/engram/cmd/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

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

	// Read directly from in-memory Prometheus registry
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

func (c *Client) GetMetricSummary() (string, error) {
	stats, err := c.GetDashboardStats()
	if err != nil {
		return "", fmt.Errorf("failed to get stats: %w", err)
	}
	return fmt.Sprintf(
		"VoxDeploy has caught %.0f CI failures, generated %.0f fixes, and opened %.0f pull requests. Success rate is %.1f%%. Average AI latency is %.1fs.",
		stats.WebhooksTotal, stats.FixesGenerated, stats.PRsOpened, stats.SuccessRate, stats.AvgAILatency,
	), nil
}
