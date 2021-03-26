package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/log"
	"github.com/slack-go/slack"
)

// InteractiveEventHandler handles interactive events from Slack.
func InteractiveEventHandler(cb *slack.InteractionCallback) error {
	n, err := extractNameFromJobResultMsg(cb)
	if err != nil {
		log.Error("failed to extract namespace and name from message", map[string]interface{}{
			log.FnError: err,
			"cb":        cb.Message.Attachments,
		})
		return err
	}

	var stderr bytes.Buffer
	command := exec.Command(
		"kubectl", "annotate", "pods",
		"-n", n.Namespace, n.Name,
		fmt.Sprintf(
			"%s=%s",
			constants.PodDeletionTimeKey,
			// added duration is now fixed, but can be configurable later.
			time.Now().Add(20*time.Minute).UTC().Format(time.RFC3339),
		),
		"--overwrite",
	)
	command.Stdout = os.Stdout
	command.Stderr = &stderr
	err = command.Run()
	if err != nil {
		log.Error("failed to annotate pod", map[string]interface{}{
			log.FnError: err,
			"name":      n.Name,
			"namespace": n.Namespace,
		})
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
