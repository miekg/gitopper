package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// do we have a latecy that we can track?

var (
	metricServiceHash = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gitopper",
		Subsystem: "service",
		Name:      "hash",
		Help:      "Current hash for this service.",
	}, []string{"service"})

	metricServiceState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gitopper",
		Subsystem: "service",
		Name:      "state",
		Help:      "Current state for  this service.",
	}, []string{"service"})

	metricServiceTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "gitopper",
		Subsystem: "service",
		Name:      "change_timestamp",
		Help:      "Timestamp for last service change.",
	}, []string{"service"})
)
