package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricsNamespace    = "meows"
	controllerSubsystem = "controller"
	runnerPoolSubsystem = "runnerpool"
	runnerSubsystem     = "runner"
)

type runnerPodState string

const (
	Initializing = runnerPodState("initializing")
	Running      = runnerPodState("running")
	Debugging    = runnerPodState("debugging")
	Stale        = runnerPodState("stale")
)

var AllRunnerPodState = []runnerPodState{
	Initializing,
	Running,
	Debugging,
	Stale,
}

type listenerExitState string

const (
	RetryableError = listenerExitState("retryable_error")
	Updating       = listenerExitState("updating")
	Undefined      = listenerExitState("undefined")
)

var AllListenerExitState = []listenerExitState{
	RetryableError,
	Updating,
	Undefined,
}

// Runner pod related metrics
var (
	podStateVec               *prometheus.GaugeVec
	listenerExitStateCountVec *prometheus.CounterVec
)

func InitRunnerPodMetrics(registry prometheus.Registerer, name string) {
	labels := prometheus.Labels{
		"runnerpool": name,
	}
	podStateVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   metricsNamespace,
			Subsystem:   runnerSubsystem,
			Name:        "pod_state",
			Help:        "1 if the state of the runner pod is the state specified by the `state` label",
			ConstLabels: labels,
		},
		[]string{"state"},
	)

	listenerExitStateCountVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   metricsNamespace,
			Subsystem:   runnerSubsystem,
			Name:        "listener_exit_state",
			Help:        "Counter for exit codes returned by the Runner.Listener",
			ConstLabels: labels,
		},
		[]string{"state"},
	)

	registry.MustRegister(
		podStateVec,
		listenerExitStateCountVec,
	)
}

func UpdateRunnerPodState(label runnerPodState) {
	for _, labelState := range AllRunnerPodState {
		var val float64
		if labelState == label {
			val = 1
		}
		podStateVec.WithLabelValues(string(labelState)).Set(val)
	}
}

func IncrementListenerExitState(label listenerExitState) {
	listenerExitStateCountVec.WithLabelValues(string(label)).Inc()
}
