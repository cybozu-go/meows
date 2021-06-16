package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "runner"
)

type PodStatus string
type JobStatus string
type JobResult string

const (
	Initializing = PodStatus("initializing")
	Running      = PodStatus("running")
	Debugging    = PodStatus("debugging")

	Listening = JobStatus("listening")
	Assigned  = JobStatus("assigned")
	Finished  = JobStatus("finished")

	Success   = JobResult("success")
	Failure   = JobResult("failure")
	Cancelled = JobResult("cancelled")
	Unknown   = JobResult("unknown")
)

var (
	AllPodStatus = []PodStatus{
		Initializing,
		Running,
		Debugging,
	}

	AllJobStatus = []JobStatus{
		Listening,
		Assigned,
		Finished,
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
		"runner_pool_name": name,
	}
	podStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "pod_status",
			Help:        "`status` label show the status of the pod.",
			ConstLabels: labels,
		},
		[]string{"status"},
	)
	jobStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "job_status",
			Help:        "`status` label show the status of the job.",
			ConstLabels: labels,
		},
		[]string{"status"},
	)
	jobResult = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "job_result",
			Help:        "`result` label show the result of the job.",
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

func UpdatePodStatus(label PodStatus) {
	for _, labelStatus := range AllPodStatus {
		var val float64 = 0
		if labelStatus == label {
			val = 1
		}
		podStatus.WithLabelValues(string(labelStatus)).Set(val)
	}
}

func UpdateJobStatus(label JobStatus) {
	for _, labelStatus := range AllJobStatus {
		var val float64 = 0
		if labelStatus == label {
			val = 1
		}
		jobStatus.WithLabelValues(string(labelStatus)).Set(val)
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
