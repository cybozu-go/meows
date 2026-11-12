package metrics

import (
	"strconv"
	"strings"
	"unicode/utf8"

	constants "github.com/cybozu-go/meows"
	"github.com/prometheus/client_golang/prometheus"
)

// maxJobInfoLabelLength is the maximum number of characters of a string label value of meows_runner_job_info.
// The values come from a file that the job itself writes, so they are not trusted.
const maxJobInfoLabelLength = 128

// JobInfoLabels represents the label values of meows_runner_job_info.
type JobInfoLabels struct {
	Repository  string
	Workflow    string
	JobID       string
	RunID       int
	PullRequest int
}

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

// InitRunnerPodMetrics initializes the metrics of a runner pod.
// jobInfoFunc is called on each scrape and returns the information about the job that the runner is running.
// It returns nil if no job information is available.
func InitRunnerPodMetrics(registry prometheus.Registerer, name string, jobInfoFunc func() *JobInfoLabels) {
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

	if jobInfoFunc != nil {
		registry.MustRegister(&jobInfoCollector{
			desc: prometheus.NewDesc(
				prometheus.BuildFQName(metricsNamespace, runnerSubsystem, "job_info"),
				"Information about the job that the runner is running. The value is always 1.",
				[]string{"repository", "workflow", "job_id", "run_id", "pull_request"},
				labels,
			),
			jobInfoFunc: jobInfoFunc,
		})
	}
}

type jobInfoCollector struct {
	desc        *prometheus.Desc
	jobInfoFunc func() *JobInfoLabels
}

func (c *jobInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *jobInfoCollector) Collect(ch chan<- prometheus.Metric) {
	info := c.jobInfoFunc()
	if info == nil || info.RunID <= 0 {
		return
	}
	var pullRequest string
	if info.PullRequest > 0 {
		pullRequest = strconv.Itoa(info.PullRequest)
	}
	m, err := prometheus.NewConstMetric(c.desc, prometheus.GaugeValue, 1,
		sanitizeLabelValue(info.Repository),
		sanitizeLabelValue(info.Workflow),
		sanitizeLabelValue(info.JobID),
		strconv.Itoa(info.RunID),
		pullRequest,
	)
	if err != nil {
		return
	}
	ch <- m
}

// sanitizeLabelValue drops invalid UTF-8 sequences and truncates the value to maxJobInfoLabelLength characters.
func sanitizeLabelValue(s string) string {
	s = strings.ToValidUTF8(s, "")
	if utf8.RuneCountInString(s) <= maxJobInfoLabelLength {
		return s
	}
	return string([]rune(s)[:maxJobInfoLabelLength])
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
