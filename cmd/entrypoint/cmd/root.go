package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/agent"

	"github.com/spf13/cobra"
)

var (
	// Environments
	podName           string
	podNs             string
	runnerToken       string
	runnerOrg         string
	runnerRepo        string
	extendDuration    string
	slackAgentSvcName string
)

var path struct {
	runner    string
	workdir   string
	tmp       string
	extend    string
	failure   string
	cancelled string
	success   string
}

var rootCmd = &cobra.Command{
	Use:   "entrypoint",
	Short: "GitHub Actions runner Entrypoint",
	Long:  "GitHub Actions runner Entrypoint",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := getEnvs(); err != nil {
			return err
		}
		setupPaths()
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.MkdirAll(path.workdir, 0777); err != nil {
			return err
		}

		runConfig(context.TODO())
		runSvc(context.TODO())

		extend, err := annotatePods(context.TODO())
		if err != nil {
			return err
		}
		if err := slackNotify(extend); err != nil {
			return err
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getEnvs() error {
	podName = os.Getenv(constants.PodNameEnvName)
	if len(podName) == 0 {
		return fmt.Errorf("%s must be set", constants.PodNameEnvName)
	}
	podNs = os.Getenv(constants.PodNamespaceEnvName)
	if len(podName) == 0 {
		return fmt.Errorf("%s must be set", constants.PodNamespaceEnvName)
	}
	runnerToken = os.Getenv(constants.RunnerTokenEnvName)
	if len(runnerToken) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerTokenEnvName)
	}
	runnerOrg = os.Getenv(constants.RunnerOrgEnvName)
	if len(runnerOrg) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerOrgEnvName)
	}
	runnerRepo = os.Getenv(constants.RunnerRepoEnvName)
	if len(runnerRepo) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerRepoEnvName)
	}
	extendDuration = os.Getenv(constants.ExtendDurationEnvName)
	if len(extendDuration) == 0 {
		extendDuration = "20m"
	}

	// SLACK_AGENT_SERVICE_NAME is optional
	slackAgentSvcName = os.Getenv(constants.SlackAgentServiceNameEnvName)

	return nil
}

func setupPaths() {
	path.runner = filepath.Join("/runner")
	path.workdir = filepath.Join(path.runner, "_work")
	path.tmp = filepath.Join("/tmp")
	path.extend = filepath.Join(path.tmp, "extend")
	path.failure = filepath.Join(path.tmp, "failure")
	path.cancelled = filepath.Join(path.tmp, "cancelled")
	path.success = filepath.Join(path.tmp, "success")
}

func runConfig(ctx context.Context) error {
	command := exec.CommandContext(ctx, filepath.Join(path.runner, "config.sh"), "--unattended", "--replace", "--name", podName, "--url", fmt.Sprintf("https://github.com/%s/%s", runnerOrg, runnerRepo), "--token", runnerToken, "--work", path.workdir)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

func runSvc(ctx context.Context) error {
	command := exec.CommandContext(ctx, filepath.Join(path.runner, "bin", "runsvc.sh"))
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

func annotatePods(ctx context.Context) (bool, error) {
	if _, err := os.Stat(path.extend); err == nil {
		dur, err := time.ParseDuration(extendDuration)
		if err != nil {
			return false, err
		}
		fmt.Printf("Annotate pods with the time %s later\n", extendDuration)
		agent.AnnotateDeletionTime(ctx, podName, podNs, time.Now().Add(dur))
		return true, nil
	} else {
		fmt.Println("Annotate pods with current time")
		agent.AnnotateDeletionTime(ctx, podName, podNs, time.Now())
		return false, nil
	}
}

func slackNotify(extend bool) error {
	var jobResult string
	if _, err := os.Stat(path.failure); err == nil {
		jobResult = "failure"
	} else if _, err := os.Stat(path.cancelled); err == nil {
		jobResult = "cancelled"
	} else if _, err := os.Stat(path.success); err == nil {
		jobResult = "success"
	} else {
		jobResult = "unknown"
	}
	if len(slackAgentSvcName) != 0 {
		fmt.Println("Send an notification to slack")
		c, err := agent.NewClient(fmt.Sprintf("http://%s", slackAgentSvcName))
		if err != nil {
			return err
		}
		jobInfo, err := agent.GetJobInfoFromFile(agent.DefaultJobInfoFile)
		if err != nil {
			return err
		}
		return c.PostResult("", jobResult, extend, podNs, podName, jobInfo)
	} else {
		fmt.Println("Skip sending an notification to slack because SLACK_AGENT_SERVICE_NAME is blank")
	}
	return nil
}
