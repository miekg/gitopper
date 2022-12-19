package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// do we have a latecy that we can track?

var (
	metricServiceState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gitopper",
		Subsystem: "service",
		Name:      "state",
		Help:      "Current state for this service.",
	}, []string{"service"})

	metricServiceTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gitopper",
		Subsystem: "service",
		Name:      "change_time_seconds",
		Help:      "Timestamp for last state change for this service.",
	}, []string{"service"})
)
