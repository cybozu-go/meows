package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/slack-go/slack"
)

// InteractiveEventHandler handles interactive events from Slack.
func InteractiveEventHandler(cb *slack.InteractionCallback) error {
	n, err := extractNameFromJobResultMsg(cb)
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	command := exec.Command(
		"kubectl",
		"annotate",
		"pods",
		"-n",
		n.Namespace,
		n.Name,
		constants.PodDeletionTimeKey,
		"--overwrite",
	)
	command.Stdout = os.Stdout
	command.Stderr = &stderr
	err = command.Run()
	if err != nil {
		return fmt.Errorf(
			"failed to annotate %s in %s with %s: err=%#v, stderr=%s",
			n.Name,
			n.Namespace,
			constants.PodDeletionTimeKey,
			err,
			stderr.String(),
		)
	}
	return nil
}
