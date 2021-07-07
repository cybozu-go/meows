package runner

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/cybozu-go/github-actions-controller/metrics"
	"github.com/cybozu-go/well"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Runner struct {
	envs       environments
	listenAddr string

	deletionTime atomic.Value
}

func NewRunner(listenAddr string) (*Runner, error) {
	r := Runner{
		envs:       newRunnerEnvs(),
		listenAddr: listenAddr,
	}

	r.deletionTime.Store(time.Time{})
	if err := r.envs.CheckEnvs(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(r.envs.workDir, 0755); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Runner) Run(ctx context.Context) error {
	registry := prometheus.DefaultRegisterer
	metrics.Init(registry, r.envs.runnerPoolName)

	env := well.NewEnvironment(ctx)
	env.Go(r.runListener)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/"+constants.DeletionTimeEndpoint, http.HandlerFunc(r.deletionTimeHandler))
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
	metrics.UpdatePodState(metrics.Initializing)
	configArgs := []string{
		"--unattended",
		"--replace",
		"--name", r.envs.podName,
		"--labels", r.envs.podNamespace + "/" + r.envs.runnerPoolName,
		"--url", fmt.Sprintf("https://github.com/%s/%s", r.envs.runnerOrg, r.envs.runnerRepo),
		"--token", r.envs.runnerToken,
		"--work", r.envs.workDir,
	}
	if _, err := runCommand(ctx, r.envs.runnerDir, r.envs.configCommand, configArgs...); err != nil {
		return err
	}

	metrics.UpdatePodState(metrics.Running)
	if err := r.runService(ctx); err != nil {
		return err
	}

	metrics.UpdatePodState(metrics.Debugging)
	extend := isFileExists(r.envs.extendFlagFile)
	err := r.updateDeletionTime(extend)
	if err != nil {
		return err
	}
	if err := r.notifyToSlack(ctx, extend); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (r *Runner) runService(ctx context.Context) error {
	for {
		code, err := runCommand(ctx, r.envs.runnerDir, r.envs.listenerCommand, "run", "--startuptype", "service", "--once")
		if _, ok := err.(*exec.ExitError); !ok {
			return err
		}

		// This logic is based on the following code.
		// ref: https://github.com/actions/runner/blob/v2.278.0/src/Misc/layoutbin/RunnerService.js
		fmt.Println("Runner listener exited with error code", code)
		switch code {
		case 0:
			fmt.Println("Runner listener exit with 0 return code, stop the service, no retry needed.")
			return nil
		case 1:
			fmt.Println("Runner listener exit with terminated error, stop the service, no retry needed.")
			return fmt.Errorf("runner listener exit with terminated error: %v", err)
		case 2:
			fmt.Println("Runner listener exit with retryable error, re-launch runner in 5 seconds.")
			metrics.IncrementListenerExitState(metrics.RetryableError)
		case 3:
			fmt.Println("Runner listener exit because of updating, re-launch runner in 5 seconds.")
			metrics.IncrementListenerExitState(metrics.Updating)
		default:
			fmt.Println("Runner listener exit with undefined return code, re-launch runner in 5 seconds.")
			metrics.IncrementListenerExitState(metrics.Undefined)
		}

		// Sleep 5 seconds to wait for the update process finish.
		time.Sleep(5 * time.Second)
	}
}

func (r *Runner) updateDeletionTime(extend bool) error {
	if extend {
		dur, err := time.ParseDuration(r.envs.extendDuration)
		if err != nil {
			return err
		}
		fmt.Printf("Annotate pod with the time %s later\n", r.envs.extendDuration)
		r.deletionTime.Store(time.Now().UTC().Add(dur))
	} else {
		fmt.Println("Annotate pod with current time")
		r.deletionTime.Store(time.Now().UTC())
	}
	return nil
}

func (r *Runner) notifyToSlack(ctx context.Context, extend bool) error {
	var jobResult string
	switch {
	case isFileExists(r.envs.failureFlagFile):
		jobResult = agent.JobResultFailure
	case isFileExists(r.envs.cancelledFlagFile):
		jobResult = agent.JobResultCancelled
	case isFileExists(r.envs.successFlagFile):
		jobResult = agent.JobResultSuccess
	default:
		jobResult = agent.JobResultUnknown
	}
	if len(r.envs.slackAgentSvcName) != 0 {
		fmt.Println("Send an notification to slack jobResult = ", jobResult)
		c, err := agent.NewClient(fmt.Sprintf("http://%s", r.envs.slackAgentSvcName))
		if err != nil {
			return err
		}
		jobInfo, err := agent.GetJobInfoFromFile(agent.DefaultJobInfoFile)
		if err != nil {
			return err
		}
		return c.PostResult(ctx, "", jobResult, extend, r.envs.podNamespace, r.envs.podName, jobInfo)
	} else {
		fmt.Println("Skip sending an notification to slack because SLACK_AGENT_SERVICE_NAME is blank")
	}
	return nil
}

func (r *Runner) deletionTimeHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if dt, ok := r.deletionTime.Load().(time.Time); ok {
		if !dt.IsZero() {
			fmt.Fprintln(w, dt.UTC().Format(time.RFC3339))
		}
	}
}
