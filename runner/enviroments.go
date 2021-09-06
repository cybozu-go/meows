package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	constants "github.com/cybozu-go/meows"
)

// Omittable options
type Option struct {
	SetupCommand []string `json:"setup_command,omitempty"`
}

type environments struct {
	podName        string
	podNamespace   string
	runnerOrg      string
	runnerRepo     string
	runnerPoolName string
	extendDuration time.Duration
	setupCommand   []string
}

func newRunnerEnvs() (*environments, error) {
	envs := &environments{
		podName:        os.Getenv(constants.PodNameEnvName),
		podNamespace:   os.Getenv(constants.PodNamespaceEnvName),
		runnerOrg:      os.Getenv(constants.RunnerOrgEnvName),
		runnerRepo:     os.Getenv(constants.RunnerRepoEnvName),
		runnerPoolName: os.Getenv(constants.RunnerPoolNameEnvName),
	}
	if err := envs.validateRequiredEnvs(); err != nil {
		return nil, err
	}

	str := os.Getenv(constants.ExtendDurationEnvName)
	if len(str) == 0 {
		envs.extendDuration = 20 * time.Minute
	} else {
		dur, err := time.ParseDuration(str)
		if err != nil {
			return nil, fmt.Errorf("failed to perse %s; %w", constants.ExtendDurationEnvName, err)
		}
		envs.extendDuration = dur
	}

	optionRaw := os.Getenv(constants.RunnerOptionEnvName)
	var opt Option
	if err := json.Unmarshal([]byte(optionRaw), &opt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s; %w", constants.RunnerOptionEnvName, err)
	}
	envs.setupCommand = opt.SetupCommand

	return envs, nil
}

func (e *environments) validateRequiredEnvs() error {
	if len(e.podName) == 0 {
		return fmt.Errorf("%s must be set", constants.PodNameEnvName)
	}
	if len(e.podNamespace) == 0 {
		return fmt.Errorf("%s must be set", constants.PodNamespaceEnvName)
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
	return nil
}
