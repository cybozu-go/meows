package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Controller related metrics
var (
	RunnerPoolSecretRetryCount *prometheus.GaugeVec

	runnerPoolReplicas *prometheus.GaugeVec
	runnerOnlineVec    *prometheus.GaugeVec
	runnerBusyVec      *prometheus.GaugeVec
)

func InitControllerMetrics(registry prometheus.Registerer) {
	RunnerPoolSecretRetryCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: runnerPoolSubsystem,
			Name:      "secret_retry_count",
			Help:      "The number of times meows retried continuously to get github token",
		},
		[]string{"runnerpool"},
	)

	runnerPoolReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: runnerPoolSubsystem,
			Name:      "replicas",
			Help:      "the number of the RunnerPool replicas",
		},
		[]string{"runnerpool"},
	)

	runnerOnlineVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: runnerSubsystem,
			Name:      "online",
			Help:      "1 if the runner is online",
		},
		[]string{"runnerpool", "runner"},
	)

	runnerBusyVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: runnerSubsystem,
			Name:      "busy",
			Help:      "1 if the runner is busy",
		},
		[]string{"runnerpool", "runner"},
	)

	registry.MustRegister(
		RunnerPoolSecretRetryCount,
		runnerPoolReplicas,
		runnerOnlineVec,
		runnerBusyVec,
	)
}

func UpdateRunnerPoolMetrics(runnerpool string, replicas int) {
	runnerPoolReplicas.WithLabelValues(runnerpool).Set(float64(replicas))
}

func DeleteRunnerPoolMetrics(runnerpool string) {
	runnerPoolReplicas.DeleteLabelValues(runnerpool)
}

func UpdateRunnerMetrics(runnerpool, runner string, online, busy bool) {
	var val1 float64
	if online {
		val1 = 1.0
	}
	runnerOnlineVec.WithLabelValues(runnerpool, runner).Set(val1)

	var val2 float64
	if busy {
		val2 = 1.0
	}
	runnerBusyVec.WithLabelValues(runnerpool, runner).Set(val2)
}

func DeleteRunnerMetrics(runnerpool, runner string) {
	runnerOnlineVec.DeleteLabelValues(runnerpool, runner)
	runnerBusyVec.DeleteLabelValues(runnerpool, runner)
}
