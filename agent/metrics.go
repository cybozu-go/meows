package agent

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "runner_pool"
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

func MetricsInit(registry prometheus.Registerer, name string) {
	labels := prometheus.Labels{
		"runner_pool_name": name,
	}
	podStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "pod",
			Help:        "`status` label show the status of the pod.",
			ConstLabels: labels,
		},
		[]string{"status"},
	)
	jobStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "job",
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
		if labelStatus == label {
			podStatus.WithLabelValues(string(labelStatus)).Set(1)
		} else {
			podStatus.WithLabelValues(string(labelStatus)).Set(0)
		}
	}
}

func UpdateJobStatus(label JobStatus) {
	for _, labelStatus := range AllJobStatus {
		if labelStatus == label {
			podStatus.WithLabelValues(string(labelStatus)).Set(1)
		} else {
			podStatus.WithLabelValues(string(labelStatus)).Set(0)
		}
	}
}

func UpdateJobResult(label JobResult) {
	for _, labelResult := range AllJobResult {
		if labelResult == label {
			podStatus.WithLabelValues(string(labelResult)).Set(1)
		} else {
			podStatus.WithLabelValues(string(labelResult)).Set(0)
		}
	}
}
