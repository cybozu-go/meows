package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/agent"
	"github.com/cybozu-go/github-actions-controller/metrics"
	"github.com/cybozu-go/well"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var config struct {
	metricsAddress string
}

var (
	// Environments
	podName           = os.Getenv(constants.PodNameEnvName)
	podNamespace      = os.Getenv(constants.PodNamespaceEnvName)
	runnerToken       = os.Getenv(constants.RunnerTokenEnvName)
	runnerOrg         = os.Getenv(constants.RunnerOrgEnvName)
	runnerRepo        = os.Getenv(constants.RunnerRepoEnvName)
	runnerPoolName    = os.Getenv(constants.RunnerPoolNameEnvName)
	extendDuration    = os.Getenv(constants.ExtendDurationEnvName)
	slackAgentSvcName = os.Getenv(constants.SlackAgentEnvName)

	// Directory/File Paths
	runnerDir = filepath.Join("/runner")
	workDir   = filepath.Join(runnerDir, "_work")

	extendFlagFile    = filepath.Join(os.TempDir(), "extend")
	failureFlagFile   = filepath.Join(os.TempDir(), "failure")
	cancelledFlagFile = filepath.Join(os.TempDir(), "cancelled")
	successFlagFile   = filepath.Join(os.TempDir(), "success")

	configCommand   = filepath.Join(runnerDir, "config.sh")
	listenerCommand = filepath.Join(runnerDir, "bin", "Runner.Listener")
)

var rootCmd = &cobra.Command{
	Use:   "entrypoint",
	Short: "GitHub Actions runner Entrypoint",
	Long:  "GitHub Actions runner Entrypoint",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return checkEnvs()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.MkdirAll(workDir, 0755); err != nil {
			return err
		}
		registry := prometheus.DefaultRegisterer
		metrics.Init(registry, runnerPoolName)

		well.Go(func(ctx context.Context) error {
			metrics.UpdatePodState(metrics.Initializing)
			configArgs := []string{
				"--unattended",
				"--replace",
				"--name", podName,
				"--labels", podNamespace + "/" + runnerPoolName,
				"--url", fmt.Sprintf("https://github.com/%s/%s", runnerOrg, runnerRepo),
				"--token", runnerToken,
				"--work", workDir,
			}
			if _, err := runCommand(ctx, runnerDir, configCommand, configArgs...); err != nil {
				return err
			}

			metrics.UpdatePodState(metrics.Running)
			if err := runService(ctx); err != nil {
				return err
			}

			metrics.UpdatePodState(metrics.Debugging)
			extend, err := annotatePod(ctx)
			if err != nil {
				return err
			}
			if err := notifyToSlack(ctx, extend); err != nil {
				return err
			}

			<-ctx.Done()
			return nil
		})

		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", promhttp.Handler())
		metricsServ := &well.HTTPServer{
			Server: &http.Server{
				Addr:    config.metricsAddress,
				Handler: metricsMux,
			},
		}
		if err := metricsServ.ListenAndServe(); err != nil {
			return err
		}

		well.Stop()
		return well.Wait()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	fs := rootCmd.Flags()
	fs.StringVar(&config.metricsAddress, "metrics-address", fmt.Sprintf(":%d", constants.RunnerMetricsPort), "Listening address and port for metrics.")
}

func checkEnvs() error {
	if len(podName) == 0 {
		return fmt.Errorf("%s must be set", constants.PodNameEnvName)
	}
	if len(podNamespace) == 0 {
		return fmt.Errorf("%s must be set", constants.PodNamespaceEnvName)
	}
	if len(runnerToken) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerTokenEnvName)
	}
	if len(runnerOrg) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerOrgEnvName)
	}
	if len(runnerRepo) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerRepoEnvName)
	}
	if len(runnerPoolName) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerPoolNameEnvName)
	}
	if len(extendDuration) == 0 {
		extendDuration = "20m"
	}

	// SLACK_AGENT_SERVICE_NAME is optional

	return nil
}

func runCommand(ctx context.Context, workDir, commandStr string, args ...string) (int, error) {
	command := exec.CommandContext(ctx, commandStr, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Dir = workDir
	command.Env = removedEnv()
	err := command.Run()
	return command.ProcessState.ExitCode(), err
}

func removedEnv() []string {
	rmList := []string{
		constants.PodNameEnvName,
		constants.PodNamespaceEnvName,
		constants.RunnerTokenEnvName,
		constants.RunnerOrgEnvName,
		constants.RunnerRepoEnvName,
		constants.RunnerPoolNameEnvName,
		constants.SlackAgentEnvName,
	}
	var removedEnv []string
	rmMap := make(map[string]struct{})
	for _, v := range rmList {
		rmMap[v] = struct{}{}
	}
	for _, target := range os.Environ() {
		keyvalue := strings.SplitN(target, "=", 2)
		if _, ok := rmMap[keyvalue[0]]; !ok {
			removedEnv = append(removedEnv, target)
		}
	}
	return removedEnv
}

func runService(ctx context.Context) error {
	for {
		code, err := runCommand(ctx, runnerDir, listenerCommand, "run", "--startuptype", "service", "--once")
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

func isFileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func annotatePod(ctx context.Context) (bool, error) {
	if isFileExists(extendFlagFile) {
		dur, err := time.ParseDuration(extendDuration)
		if err != nil {
			return false, err
		}
		fmt.Printf("Annotate pod with the time %s later\n", extendDuration)
		agent.AnnotateDeletionTime(ctx, podName, podNamespace, time.Now().Add(dur))
		return true, nil
	} else {
		fmt.Println("Annotate pod with current time")
		agent.AnnotateDeletionTime(ctx, podName, podNamespace, time.Now())
		return false, nil
	}
}

func notifyToSlack(ctx context.Context, extend bool) error {
	var jobResult string
	switch {
	case isFileExists(failureFlagFile):
		jobResult = agent.JobResultFailure
	case isFileExists(cancelledFlagFile):
		jobResult = agent.JobResultCancelled
	case isFileExists(successFlagFile):
		jobResult = agent.JobResultSuccess
	default:
		jobResult = agent.JobResultUnknown
	}
	if len(slackAgentSvcName) != 0 {
		fmt.Println("Send an notification to slack jobResult = ", jobResult)
		c, err := agent.NewClient(fmt.Sprintf("http://%s", slackAgentSvcName))
		if err != nil {
			return err
		}
		jobInfo, err := agent.GetJobInfoFromFile(agent.DefaultJobInfoFile)
		if err != nil {
			return err
		}
		return c.PostResult(ctx, "", jobResult, extend, podNamespace, podName, jobInfo)
	} else {
		fmt.Println("Skip sending an notification to slack because SLACK_AGENT_SERVICE_NAME is blank")
	}
	return nil
}
