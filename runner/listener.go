package runner

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/cybozu-go/github-actions-controller/metrics"
)

type listener interface {
	configure(ctx context.Context, configArgs []string) error
	listen(ctx context.Context) error
}

type listenerImpl struct {
	runnerDir       string
	configCommand   string
	listenerCommand string
}

func newListener(runnerDir, configCommand, listenerCommand string) listener {
	return &listenerImpl{
		runnerDir:       runnerDir,
		configCommand:   configCommand,
		listenerCommand: listenerCommand,
	}
}

func (e *listenerImpl) configure(ctx context.Context, configArgs []string) error {
	_, err := runCommand(ctx, e.runnerDir, e.configCommand, configArgs...)
	return err
}

func (e *listenerImpl) listen(ctx context.Context) error {
	for {
		code, err := runCommand(ctx, e.runnerDir, e.listenerCommand, "run", "--startuptype", "service", "--once")
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
			fmt.Println("Runner listener exit with retryable error, re-launch runner in 10 seconds.")
			metrics.IncrementListenerExitState(metrics.RetryableError)
		case 3:
			fmt.Println("Runner listener exit because of updating, re-launch runner in 10 seconds.")
			metrics.IncrementListenerExitState(metrics.Updating)
		default:
			fmt.Println("Runner listener exit with undefined return code, re-launch runner in 10 seconds.")
			metrics.IncrementListenerExitState(metrics.Undefined)
		}

		// Sleep 10 seconds to wait for the update process finish.
		time.Sleep(10 * time.Second)
	}
}
