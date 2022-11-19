package gitcmd

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricGitFail = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "gitopper",
		Subsystem: "machine",
		Name:      "git_error_total",
		Help:      "Total number of git operations that failed.",
	})

	metricGitOps = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "gitopper",
		Subsystem: "machine",
		Name:      "git_ops_total",
		Help:      "Total number of git operations.",
	})
)
