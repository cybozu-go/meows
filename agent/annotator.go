package agent

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
)

// AnnotateDeletionTime annotates a given time to a Pod.
func AnnotateDeletionTime(name string, namespace string, t time.Time) error {
	var stdout, stderr bytes.Buffer
	command := exec.Command(
		"kubectl", "annotate", "pods",
		"-n", namespace, name,
		fmt.Sprintf(
			"%s=%s",
			constants.PodDeletionTimeKey,
			t.UTC().Format(time.RFC3339),
		),
		"--overwrite",
	)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		return fmt.Errorf(
			"failed to annotate %s in %s with %s: err=%#v, stderr=%s, stdout=%s",
			name,
			namespace,
			constants.PodDeletionTimeKey,
			err,
			stderr.String(),
			stdout.String(),
		)
	}
	return nil
}
