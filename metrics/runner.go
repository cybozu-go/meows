package metrics

import (
	constants "github.com/cybozu-go/meows"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricsNamespace    = "meows"
	controllerSubsystem = "controller"
	runnerPoolSubsystem = "runnerpool"
	runnerSubsystem     = "runner"
)

var allRunnerPodState = []string{
	constants.RunnerPodStateInitializing,
	constants.RunnerPodStateRunning,
	constants.RunnerPodStateDebugging,
	constants.RunnerPodStateStale,
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

func UpdateRunnerPodState(curState string) {
	for _, state := range allRunnerPodState {
		var val float64
		if state == curState {
			val = 1
		}
		podStateVec.WithLabelValues(string(state)).Set(val)
	}
}

func IncrementListenerExitState(state string) {
	listenerExitStateCountVec.WithLabelValues(string(state)).Inc()
}
