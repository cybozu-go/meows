package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/well"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	JobResultSuccess   = "success"
	JobResultFailure   = "failure"
	JobResultCancelled = "cancelled"
	JobResultUnknown   = "unknown"
)

type Runner struct {
	envs       *environments
	listenAddr string
	listener   Listener

	// Status
	mu           sync.Mutex
	state        string
	result       string
	finishedAt   *time.Time
	deletionTime *time.Time
	extend       *bool
	jobInfo      *JobInfo
	slackChannel string

	// Directory/File Paths
	runnerDir         string
	workDir           string
	tokenPath         string
	jobInfoFile       string
	slackChannelFile  string
	startedFlagFile   string
	extendFlagFile    string
	failureFlagFile   string
	cancelledFlagFile string
	successFlagFile   string
}

type Status struct {
	State        string     `json:"state,omitempty"`
	Result       string     `json:"result,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	DeletionTime *time.Time `json:"deletion_time,omitempty"`
	Extend       *bool      `json:"extend,omitempty"`
	JobInfo      *JobInfo   `json:"job_info,omitempty"`
	SlackChannel string     `json:"slack_channel,omitempty"`
}

type DeletionTimePayload struct {
	DeletionTime time.Time `json:"deletion_time"`
}

func NewRunner(listener Listener, listenAddr, runnerDir, workDir, varDir string) (*Runner, error) {
	envs, err := newRunnerEnvs()
	if err != nil {
		return nil, err
	}

	r := Runner{
		envs:              envs,
		listenAddr:        listenAddr,
		listener:          listener,
		runnerDir:         runnerDir,
		workDir:           workDir,
		tokenPath:         filepath.Join(varDir, "runnertoken"),
		jobInfoFile:       filepath.Join(varDir, "github.env"),
		slackChannelFile:  filepath.Join(varDir, "slack_channel"),
		startedFlagFile:   filepath.Join(varDir, "started"),
		extendFlagFile:    filepath.Join(varDir, "extend"),
		failureFlagFile:   filepath.Join(varDir, "failure"),
		cancelledFlagFile: filepath.Join(varDir, "cancelled"),
		successFlagFile:   filepath.Join(varDir, "success"),
	}
	return &r, nil
}

func (r *Runner) Run(ctx context.Context) error {
	registry := prometheus.NewRegistry()
	metrics.InitRunnerPodMetrics(registry, r.envs.podNamespace+"/"+r.envs.runnerPoolName)

	env := well.NewEnvironment(ctx)
	env.Go(r.runListener)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.InstrumentMetricHandler(registry, promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))
	mux.Handle("/"+constants.DeletionTimeEndpoint, http.HandlerFunc(r.deletionTimeHandler))
	mux.Handle("/"+constants.StatusEndPoint, http.HandlerFunc(r.statusHandler))
	serv := &well.HTTPServer{
		Env: env,
		Server: &http.Server{
			Addr:    r.listenAddr,
			Handler: mux,
		},
	}
	if err := serv.ListenAndServe(); err != nil {
		return err
	}

	env.Stop()
	return env.Wait()
}

func (r *Runner) runListener(ctx context.Context) error {
	logger := log.FromContext(ctx)
	if isFileExists(r.startedFlagFile) {
		metrics.UpdateRunnerPodState(constants.RunnerPodStateStale)
		logger.Info("Pod is stale; waiting for deletion")
		r.updateState(constants.RunnerPodStateStale)
		<-ctx.Done()
		return nil
	}
	if _, err := os.Create(r.startedFlagFile); err != nil {
		return err
	}

	metrics.UpdateRunnerPodState(constants.RunnerPodStateInitializing)
	r.updateState(constants.RunnerPodStateInitializing)
	if len(r.envs.setupCommand) != 0 {
		if _, err := runCommand(ctx, r.runnerDir, r.envs.setupCommand[0], r.envs.setupCommand[1:]...); err != nil {
			return err
		}
	}

	b, err := os.ReadFile(r.tokenPath)
	if err != nil {
		return fmt.Errorf("failed load %s; %w", r.tokenPath, err)
	}

	configURL := fmt.Sprintf("https://github.com/%s", r.envs.runnerOrg)
	if r.envs.runnerRepo != "" {
		configURL = configURL + "/" + r.envs.runnerRepo
	}

	configArgs := []string{
		"--unattended",
		"--replace",
		"--name", r.envs.podName,
		"--labels", r.envs.podNamespace + "/" + r.envs.runnerPoolName,
		"--url", configURL,
		"--token", string(b),
		"--work", r.workDir,
		"--ephemeral",
	}
	if err := r.listener.configure(ctx, configArgs); err != nil {
		return err
	}

	metrics.UpdateRunnerPodState(constants.RunnerPodStateRunning)
	r.updateState(constants.RunnerPodStateRunning)
	if err := r.listener.listen(ctx); err != nil {
		return err
	}

	metrics.UpdateRunnerPodState(constants.RunnerPodStateDebugging)
	r.updateToDebugginState(logger)

	<-ctx.Done()
	return nil
}

func (r *Runner) updateState(state string) {
	r.mu.Lock()
	r.state = state
	r.mu.Unlock()
}

func (r *Runner) updateToDebugginState(logger logr.Logger) {
	var result string
	switch {
	case isFileExists(r.failureFlagFile):
		result = JobResultFailure
	case isFileExists(r.cancelledFlagFile):
		result = JobResultCancelled
	case isFileExists(r.successFlagFile):
		result = JobResultSuccess
	default:
		result = JobResultUnknown
	}
	extend := isFileExists(r.extendFlagFile)

	finishedAt := time.Now().UTC()
	var deletionTime time.Time
	if extend {
		deletionTime = finishedAt.Add(r.envs.extendDuration)
	} else {
		deletionTime = finishedAt
	}

	jobInfo, err := GetJobInfoFromFile(r.jobInfoFile)
	if err != nil {
		logger.Error(err, "failed to read job info")
	}

	slackChannel, err := r.readSlackChannel()
	if err != nil {
		logger.Error(err, "failed to read file for slack channel")
	}

	r.mu.Lock()
	r.state = constants.RunnerPodStateDebugging
	r.result = result
	r.finishedAt = &finishedAt
	r.deletionTime = &deletionTime
	r.extend = &extend
	r.jobInfo = jobInfo
	r.slackChannel = slackChannel
	r.mu.Unlock()
}

func (r *Runner) readSlackChannel() (string, error) {
	file, err := os.Open(r.slackChannelFile)
	if err != nil {
		return "", err
	}

	s, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(s), "\n"), nil
}

func (r *Runner) deletionTimeHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var dt DeletionTimePayload
	if req.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}
	err := json.NewDecoder(req.Body).Decode(&dt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// FIXME: Should we check runner state here?
	r.mu.Lock()
	r.deletionTime = &dt.DeletionTime
	r.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (r *Runner) statusHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var st Status
	r.mu.Lock()
	st.State = r.state
	st.Result = r.result
	st.FinishedAt = r.finishedAt
	st.DeletionTime = r.deletionTime
	st.Extend = r.extend
	st.JobInfo = r.jobInfo
	st.SlackChannel = r.slackChannel
	r.mu.Unlock()

	res, err := json.Marshal(st)
	if err != nil {
		http.Error(w, "Failed to marshal status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}
