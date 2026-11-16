package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// jobDurationBuckets are the buckets of meows_runnerpool_job_duration_seconds, from 1 minute to 6 hours.
var jobDurationBuckets = []float64{60, 300, 600, 900, 1200, 1800, 2700, 3600, 5400, 7200, 10800, 14400, 21600}

// knownJobResults are the values of the result label. Other values are recorded as "unknown".
var knownJobResults = map[string]struct{}{
	"success":   {},
	"failure":   {},
	"cancelled": {},
	"unknown":   {},
}

// Controller related metrics
var (
	RunnerPoolSecretRetryCount *prometheus.CounterVec
	runnerPoolReplicas         *prometheus.GaugeVec
	runnerOnlineVec            *prometheus.GaugeVec
	runnerBusyVec              *prometheus.GaugeVec
	jobResultsVec              *prometheus.CounterVec
	jobDurationVec             *prometheus.HistogramVec
	runnerLabelSet             map[string]map[string]struct{} // runnerpool -> runner -> struct{}
	runnerLabelSetMutex        sync.Mutex
)

func InitControllerMetrics(registry prometheus.Registerer) {
	RunnerPoolSecretRetryCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
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

	jobResultsVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: runnerPoolSubsystem,
			Name:      "job_results_total",
			Help:      "The number of finished jobs by result",
		},
		[]string{"runnerpool", "result"},
	)

	jobDurationVec = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: runnerPoolSubsystem,
			Name:      "job_duration_seconds",
			Help:      "The duration of finished jobs from job-started to the end of the job",
			Buckets:   jobDurationBuckets,
		},
		[]string{"runnerpool", "result"},
	)

	runnerLabelSet = map[string]map[string]struct{}{}

	registry.MustRegister(
		RunnerPoolSecretRetryCount,
		runnerPoolReplicas,
		runnerOnlineVec,
		runnerBusyVec,
		jobResultsVec,
		jobDurationVec,
	)
}

func UpdateRunnerPoolMetrics(runnerpool string, replicas int) {
	runnerPoolReplicas.WithLabelValues(runnerpool).Set(float64(replicas))
}

func DeleteRunnerPoolMetrics(runnerpool string) {
	runnerPoolReplicas.DeleteLabelValues(runnerpool)
	jobResultsVec.DeletePartialMatch(prometheus.Labels{"runnerpool": runnerpool})
	jobDurationVec.DeletePartialMatch(prometheus.Labels{"runnerpool": runnerpool})
}

// InitJobMetrics exposes the job metrics of the RunnerPool with zero values for all results.
// Without this, the first job of each result would not be counted as an increase by rate() or increase().
func InitJobMetrics(runnerpool string) {
	for result := range knownJobResults {
		jobResultsVec.WithLabelValues(runnerpool, result)
		jobDurationVec.WithLabelValues(runnerpool, result)
	}
}

// RecordJobResult records the result of a finished job.
// duration is nil if the start time of the job is unknown.
func RecordJobResult(runnerpool, result string, duration *time.Duration) {
	if _, ok := knownJobResults[result]; !ok {
		result = "unknown"
	}
	jobResultsVec.WithLabelValues(runnerpool, result).Inc()
	if duration != nil {
		jobDurationVec.WithLabelValues(runnerpool, result).Observe(duration.Seconds())
	}
}

func UpdateRunnerMetrics(runnerpool, runner string, online, busy bool) {
	runnerLabelSetMutex.Lock()
	defer runnerLabelSetMutex.Unlock()

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
	runnerLabelSetMutex.Lock()
	defer runnerLabelSetMutex.Unlock()

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
	runnerLabelSetMutex.Lock()
	defer runnerLabelSetMutex.Unlock()

	labelSet, ok := runnerLabelSet[runnerpool]
	if !ok {
		return
	}
	for r := range labelSet {
		runnerOnlineVec.DeleteLabelValues(runnerpool, r)
		runnerBusyVec.DeleteLabelValues(runnerpool, r)
	}
	delete(runnerLabelSet, runnerpool)
}
