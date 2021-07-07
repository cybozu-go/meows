package runner

import (
	"fmt"
	"os"
	"path/filepath"

	constants "github.com/cybozu-go/github-actions-controller"
)

type environments struct {
	// Environments
	podName           string
	podNamespace      string
	runnerToken       string
	runnerOrg         string
	runnerRepo        string
	runnerPoolName    string
	extendDuration    string
	slackAgentSvcName string

	// Directory/File Paths
	runnerDir string
	workDir   string

	extendFlagFile    string
	failureFlagFile   string
	cancelledFlagFile string
	successFlagFile   string

	configCommand   string
	listenerCommand string
}

func newRunnerEnvs() environments {
	envs := environments{
		podName:           os.Getenv(constants.PodNameEnvName),
		podNamespace:      os.Getenv(constants.PodNamespaceEnvName),
		runnerToken:       os.Getenv(constants.RunnerTokenEnvName),
		runnerOrg:         os.Getenv(constants.RunnerOrgEnvName),
		runnerRepo:        os.Getenv(constants.RunnerRepoEnvName),
		runnerPoolName:    os.Getenv(constants.RunnerPoolNameEnvName),
		extendDuration:    os.Getenv(constants.ExtendDurationEnvName),
		slackAgentSvcName: os.Getenv(constants.SlackAgentEnvName),
	}
	// Directory/File Paths
	envs.runnerDir = filepath.Join("/runner")
	envs.workDir = filepath.Join(envs.runnerDir, "_work")

	envs.extendFlagFile = filepath.Join(os.TempDir(), "extend")
	envs.failureFlagFile = filepath.Join(os.TempDir(), "failure")
	envs.cancelledFlagFile = filepath.Join(os.TempDir(), "cancelled")
	envs.successFlagFile = filepath.Join(os.TempDir(), "success")

	envs.configCommand = filepath.Join(envs.runnerDir, "config.sh")
	envs.listenerCommand = filepath.Join(envs.runnerDir, "bin", "Runner.Listener")
	return envs
}

func (e *environments) CheckEnvs() error {
	if len(e.podName) == 0 {
		return fmt.Errorf("%s must be set", constants.PodNameEnvName)
	}
	if len(e.podNamespace) == 0 {
		return fmt.Errorf("%s must be set", constants.PodNamespaceEnvName)
	}
	if len(e.runnerToken) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerTokenEnvName)
	}
	if len(e.runnerOrg) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerOrgEnvName)
	}
	if len(e.runnerRepo) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerRepoEnvName)
	}
	if len(e.runnerPoolName) == 0 {
		return fmt.Errorf("%s must be set", constants.RunnerPoolNameEnvName)
	}
	if len(e.extendDuration) == 0 {
		e.extendDuration = "20m"
	}

	// SLACK_AGENT_SERVICE_NAME is optional

	return nil
}
