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
	Starting    = PodStatus("starting")
	Registering = PodStatus("registering")
	Deleting    = PodStatus("deleting")

	Waiting   = JobStatus("waiting")
	Running   = JobStatus("running")
	Completed = JobStatus("completed")

	Success   = JobResult("success")
	Failure   = JobResult("failure")
	Cancelled = JobResult("cancelled")
	Unknown   = JobResult("unknown")
)

var (
	AllPodStatus = []PodStatus{
		Starting,
		Registering,
		Deleting,
	}

	AllJobStatus = []JobStatus{
		Waiting,
		Running,
		Completed,
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
		podStatus.WithLabelValues(string(labelStatus)).Set(val)
	}
}

func UpdateJobResult(label JobResult) {
	for _, labelResult := range AllJobResult {
		var val float64 = 0
		if labelResult == label {
			val = 1
		}
		podStatus.WithLabelValues(string(labelResult)).Set(val)
	}
}
