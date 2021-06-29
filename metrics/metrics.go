package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "runner"
)

type PodState string
type ListenerExitState string

const (
	Initializing = PodState("initializing")
	Running      = PodState("running")
	Debugging    = PodState("debugging")

	Successful      = ListenerExitState("successful")
	TerminatedError = ListenerExitState("terminated_error")
	RetryableError  = ListenerExitState("retryable_error")
	Updating        = ListenerExitState("updating")
	Undefined       = ListenerExitState("undefined")
)

var (
	AllPodState = []PodState{
		Initializing,
		Running,
		Debugging,
	}

	AllListenerExitState = []ListenerExitState{
		Successful,
		TerminatedError,
		RetryableError,
		Updating,
		Undefined,
	}

	podState          *prometheus.GaugeVec
	listenerExitState *prometheus.CounterVec
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

	listenerExitState = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Name:        "listener_exit_state",
			Help:        "Counter for exit codes returned by the Runner.Listener",
			ConstLabels: labels,
		},
		[]string{"state"},
	)

	registry.MustRegister(
		podState,
		listenerExitState,
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

func IncrementListenerExitState(label ListenerExitState) {
	listenerExitState.WithLabelValues(string(label)).Inc()
}
