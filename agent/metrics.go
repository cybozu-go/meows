package agent

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "runner"
	subsystem = "pod"
)

var (
	StatePending prometheus.Gauge
)

func MetricsInit(registry prometheus.Registerer, name string) {
	labels := prometheus.Labels{
		"name": name,
	}
	StatePending = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "state",
		Help:      "",
		ConstLabels: mergeLabels(labels, prometheus.Labels{
			"state": "pending",
		}),
	})

	registry.MustRegister(
		StatePending,
	)
}

func mergeLabels(labels ...prometheus.Labels) prometheus.Labels {
	merged := make(prometheus.Labels, 0)
	for _, l := range labels {
		for k, v := range l {
			merged[k] = v
		}
	}
	return merged
}
