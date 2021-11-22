package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Controller related metrics
var (
	RunnerPoolSecretRetryCount *prometheus.GaugeVec
	runnerPoolReplicas         *prometheus.GaugeVec
	runnerOnlineVec            *prometheus.GaugeVec
	runnerBusyVec              *prometheus.GaugeVec
	runnerLabelSet             = map[string]map[string]struct{}{}
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

	runnerLabelSet = map[string]map[string]struct{}{}

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
	if _, ok := runnerLabelSet[runnerpool]; !ok {
		runnerLabelSet[runnerpool] = map[string]struct{}{}
	}
	runnerLabelSet[runnerpool][runner] = struct{}{}

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

func DeleteRunnerMetrics(runnerpool string, runners ...string) {
	labelSet, ok := runnerLabelSet[runnerpool]
	if !ok {
		return
	}
	for _, r := range runners {
		runnerOnlineVec.DeleteLabelValues(runnerpool, r)
		runnerBusyVec.DeleteLabelValues(runnerpool, r)
		delete(labelSet, r)
	}
	if len(labelSet) == 0 {
		delete(runnerLabelSet, runnerpool)
	}
}

func DeleteAllRunnerMetrics(runnerpool string) {
	labelSet, ok := runnerLabelSet[runnerpool]
	if !ok {
		return
	}
	runners := make([]string, 0, len(labelSet))
	for r := range labelSet {
		runners = append(runners, r)
	}
	DeleteRunnerMetrics(runnerpool, runners...)
}
