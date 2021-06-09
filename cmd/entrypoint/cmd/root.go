package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	constants "github.com/cybozu-go/github-actions-controller"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "entrypoint",
	Short: "GitHub Actions runner Entrypoint",
	Long:  "GitHub Actions runner Entrypoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := os.Getenv(constants.PodNameEnvName)
		if len(podName) == 0 {
			return fmt.Errorf("%s must be set", constants.PodNameEnvName)
		}
		runnerToken := os.Getenv(constants.RunnerTokenEnvName)
		if len(runnerToken) == 0 {
			return fmt.Errorf("%s must be set", constants.RunnerTokenEnvName)
		}
		runnerOrg := os.Getenv(constants.RunnerOrgEnvName)
		if len(runnerOrg) == 0 {
			return fmt.Errorf("%s must be set", constants.RunnerOrgEnvName)
		}
		runnerRepo := os.Getenv(constants.RunnerRepoEnvName)
		if len(runnerRepo) == 0 {
			return fmt.Errorf("%s must be set", constants.RunnerRepoEnvName)
		}

		if err := os.MkdirAll(filepath.Join("runner", "_work"), 0777); err != nil {
			return fmt.Errorf("mkdir error %+v", err)
		}

		var stdout, stderr bytes.Buffer
		command := exec.Command(filepath.Join("/", "runner", "config.sh"), "--unattended", "--replace", "--name", podName, "--url", fmt.Sprintf("https://github.com/%s/%s", runnerOrg, runnerRepo), "--token", runnerToken, "--work", filepath.Join("/", "runner", "_work"))
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err != nil {
			return err
		}
		command = exec.Command(filepath.Join("/", "runner", "bin", "runsvc.sh"))
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err != nil {
			return err
		}

		extendDuration := os.Getenv(constants.ExtendDurationEnvName)
		if len(extendDuration) == 0 {
			// TODO: dont hardcode the default value
			extendDuration = "20m"
		}

		// var extend bool
		if _, err := os.Stat(filepath.Join("/", "tmp", "extend")); err == nil {
			fmt.Printf("Annotate pods with the time %s later\n", extendDuration)
			// TODO:
			// extend = true
		} else {
			fmt.Println("Annotate pods with current time")
			// TODO:
			// extend = false
		}

		slackAgentSvcName := os.Getenv(constants.SlackAgentServiceNameEnvName)
		if len(slackAgentSvcName) != 0 {
			fmt.Println("Send an notification to slack")
			// TODO:
		} else {
			fmt.Println("Skip sending an notification to slack because SLACK_AGENT_SERVICE_NAME is blank")
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
