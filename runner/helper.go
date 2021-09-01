package runner

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	constants "github.com/cybozu-go/meows"
)

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
		constants.RunnerOptionEnvName,
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

func getFileUpdateTime(filename string) time.Time {
	stat, _ := os.Stat(filename)
	return stat.ModTime().UTC()
}
