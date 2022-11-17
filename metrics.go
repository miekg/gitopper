package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "gitopper"

var (
	metricMachineInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "machine",
		Name:      "info",
		Help:      "Info on the machine",
	}, []string{"machine"})

	metricServiceHash = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "service",
		Name:      "info",
		Help:      "Current hash value for this service",
	}, []string{"service", "hash", "state"})

	metricServiceFrozen = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "service",
		Name:      "frozen_count_total",
		Help:      "Total number of frozen services",
	})

	metricServiceOk = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "service",
		Name:      "ok_count_total",
		Help:      "Total number of normal running services",
	})

	metricGitFail = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "service",
		Name:      "git_error_total",
	}, []string{"service"})

	metricGitOps = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "service",
		Name:      "git_ops_total",
	}, []string{"service"})
)
