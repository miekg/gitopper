package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "gitopper"

var (
	metricMachineInfo = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "machine_info",
		Help:      "Info on the machine",
	}, []string{"machine"})

	metricServiceHash = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "service_info",
		Help:      "Current hash value for this service",
	}, []string{"service", "hash", "state"})

	metricServiceFrozen = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "service_frozen_count_total",
		Help:      "Total number of frozen services",
	})

	metricServiceOk = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "service_ok_count_total",
		Help:      "Total number of normal running services",
	})
)
