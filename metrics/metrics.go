package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "runner"
)

type PodState string

const (
	Initializing = PodState("initializing")
	Running      = PodState("running")
	Debugging    = PodState("debugging")
)

var (
	AllPodState = []PodState{
		Initializing,
		Running,
		Debugging,
	}

	podState *prometheus.GaugeVec
)

func Init(registry prometheus.Registerer, name string) {
	labels := prometheus.Labels{
		"runner_pool": name,
	}
	podState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "pod_state",
			Help:        "1 if the state of the runner pod is the state specified by the `state` label",
			ConstLabels: labels,
		},
		[]string{"state"},
	)
	registry.MustRegister(
		podState,
	)
}

func UpdatePodState(label PodState) {
	for _, labelState := range AllPodState {
		var val float64 = 0
		if labelState == label {
			val = 1
		}
		podState.WithLabelValues(string(labelState)).Set(val)
	}
}
