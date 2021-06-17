package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "runner"
)

type PodState string
type JobResult string

const (
	Initializing = PodState("initializing")
	Running      = PodState("running")
	Debugging    = PodState("debugging")

	Success   = JobResult("success")
	Failure   = JobResult("failure")
	Cancelled = JobResult("cancelled")
	Unknown   = JobResult("unknown")
)

var (
	AllPodState = []PodState{
		Initializing,
		Running,
		Debugging,
	}

	AllJobResult = []JobResult{
		Success,
		Failure,
		Cancelled,
		Unknown,
	}

	podStatus *prometheus.GaugeVec
	jobStatus *prometheus.GaugeVec
	jobResult *prometheus.GaugeVec
)

func Init(registry prometheus.Registerer, name string) {
	labels := prometheus.Labels{
		"runner_pool": name,
	}
	podStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "pod_status",
			Help:        "1 if the state of the runner pod is the state specified by the `state` label",
			ConstLabels: labels,
		},
		[]string{"state"},
	)
	jobResult = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "job_result",
			Help:        "1 if the result of the job is the result specified by the `result` label",
			ConstLabels: labels,
		},
		[]string{"result"},
	)
	registry.MustRegister(
		podStatus,
		jobStatus,
		jobResult,
	)
}

func UpdatePodState(label PodState) {
	for _, labelStatus := range AllPodState {
		var val float64 = 0
		if labelStatus == label {
			val = 1
		}
		podStatus.WithLabelValues(string(labelStatus)).Set(val)
	}
}

func UpdateJobResult(label JobResult) {
	for _, labelResult := range AllJobResult {
		var val float64 = 0
		if labelResult == label {
			val = 1
		}
		jobResult.WithLabelValues(string(labelResult)).Set(val)
	}
}
