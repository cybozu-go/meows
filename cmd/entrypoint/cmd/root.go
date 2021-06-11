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
	"github.com/cybozu-go/well"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

const (
	metricsDefaultAddr = ":8080"
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
	extendDuration    = os.Getenv(constants.ExtendDurationEnvName)
	slackAgentSvcName = os.Getenv(constants.SlackAgentServiceNameEnvName)

	// Directory/File Paths
	runnerDir = filepath.Join("/runner")
	workDir   = filepath.Join(runnerDir, "_work")

	extendFlagFile    = filepath.Join(os.TempDir(), "extend")
	failureFlagFile   = filepath.Join(os.TempDir(), "failure")
	cancelledFlagFile = filepath.Join(os.TempDir(), "cancelled")
	successFlagFile   = filepath.Join(os.TempDir(), "success")

	configCommand = filepath.Join(runnerDir, "config.sh")
	runSVCCommand = filepath.Join(runnerDir, "bin", "runsvc.sh")
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

		configArgs := []string{
			"--unattended",
			"--replace",
			"--name", podName,
			"--url", fmt.Sprintf("https://github.com/%s/%s", runnerOrg, runnerRepo),
			"--token", runnerToken,
			"--work", workDir,
		}
		well.Go(func(ctx context.Context) error {
			agent.StatePending.Set(1)
			if err := runCommand(ctx, runnerDir, configCommand, configArgs...); err != nil {
				return err
			}
			agent.StatePending.Set(0)
			if err := runCommand(ctx, runnerDir, runSVCCommand); err != nil {
				return err
			}

			extend, err := annotatePod(ctx)
			if err != nil {
				return err
			}
			if err := notifyToSlack(ctx, extend); err != nil {
				return err
			}

			select {
			case <-ctx.Done():
			case <-time.After(time.Duration(1<<63 - 1)):
			}
			return nil
		})

		registry := prometheus.DefaultRegisterer
		agent.MetricsInit(registry, podName)

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
	fs.StringVar(&config.metricsAddress, "metrics-address", metricsDefaultAddr, "Listening address and port for metrics.")
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
	if len(extendDuration) == 0 {
		extendDuration = "20m"
	}

	// SLACK_AGENT_SERVICE_NAME is optional

	return nil
}

func runCommand(ctx context.Context, workDir, commandStr string, args ...string) error {
	command := exec.CommandContext(ctx, commandStr, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Dir = workDir
	command.Env = removedEnv()
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

func removedEnv() []string {
	rmList := []string{
		constants.PodNameEnvName,
		constants.PodNamespaceEnvName,
		constants.RunnerTokenEnvName,
		constants.RunnerOrgEnvName,
		constants.RunnerRepoEnvName,
		constants.SlackAgentServiceNameEnvName,
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
