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
func InteractiveEventHandler(cb *slack.InteractionCallback) (interface{}, error) {
	name, namespace, err := extractNameFromJobResultMsg(cb)
	if err != nil {
		log.Error("failed to extract namespace and name from message", map[string]interface{}{
			log.FnError: err,
			"cb":        cb.Message.Attachments,
		})
		return nil, err
	}

	var stderr bytes.Buffer
	command := exec.Command(
		"kubectl", "annotate", "pods",
		"-n", namespace, name,
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
			"name":      name,
			"namespace": namespace,
		})
		return nil, fmt.Errorf(
			"failed to annotate %s in %s with %s: err=%#v, stderr=%s",
			name,
			namespace,
			constants.PodDeletionTimeKey,
			err,
			stderr.String(),
		)
	}
	return makeExtendCallbackMsg(name, namespace), nil
}

func makeExtendCallbackMsg(
	podNamespace string,
	podName string,
) []slack.Attachment {
	return []slack.Attachment{
		{
			// "warning" is yellow
			Color: "warning",
			Text: fmt.Sprintf(
				"%s in %s is extended successfully",
				podName,
				podNamespace,
			),
		},
	}
}

func extractNameFromJobResultMsg(body *slack.InteractionCallback) (string, string, error) {
	if len(body.Message.Attachments) != 1 {
		return "", "", fmt.Errorf(
			"length of attachments should be 1, but got %d: %#v",
			len(body.Message.Attachments),
			body.Message.Attachments,
		)
	}

	var name, namespace string
	a := body.Message.Attachments[0]
	for _, v := range a.Fields {
		switch v.Title {
		case podNameTitle:
			name = v.Value
		case podNamespaceTitle:
			namespace = v.Value
		}
	}

	if len(name) == 0 {
		return "", "", fmt.Errorf(`the field "%s" should not be empty`, podNameTitle)
	}
	if len(namespace) == 0 {
		return "", "", fmt.Errorf(`the field "%s" should not be empty`, podNamespaceTitle)
	}

	return name, namespace, nil
}
