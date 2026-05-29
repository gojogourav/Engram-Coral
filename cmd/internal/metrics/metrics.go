package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	WebHookRecieved = promauto.NewCounter(prometheus.CounterOpts{
		Name: "engram_webhooks_received_total",
		Help: "Total number of CI failure webhooks received from GitHub",
	})

	FixesGenerated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "engram",
		Help: "Total number of AI fixes successfully generated",
	})

	PRsOpened = promauto.NewCounter(prometheus.CounterOpts{
		Name: "engram_prs_opened_total",
		Help: "Total number of pull requests automatically opened",
	})

	FixFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "engram_fix_failed_total",
		Help: "Total number of times the fix pipeline failed at any stage",
	})

	DiffApplyErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "engram_diff_apply_errors_total",
		Help: "Number of times the diff parser or applier failed",
	})

	AILatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "engram_ai_generation_seconds",
		Help:    "Time taken for Gemini to generate the fix diff",
		Buckets: []float64{1, 2, 5, 10, 20, 30, 60},
	})

	FullPipelineLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "engram_pipeline_duration_seconds",
		Help:    "Total time from webhook received to PR opened",
		Buckets: []float64{5, 10, 20, 30, 60, 120, 300},
	})

	FilesPatched = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "engram_files_patched_per_fix",
		Help:    "Number of files patched per fix",
		Buckets: []float64{1, 2, 3, 5, 10},
	})
)
