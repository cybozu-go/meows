package e2e

import (
	"bytes"
	"os/exec"
	"path/filepath"
)

var _ = execAtLocal
var _ = kubectl
var _ = kubectlWithInput

func execAtLocal(cmd string, input []byte, args ...string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	command := exec.Command(cmd, args...)
	command.Stdout = &stdout
	command.Stderr = &stderr

	if len(input) != 0 {
		command.Stdin = bytes.NewReader(input)
	}

	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func kubectl(args ...string) ([]byte, []byte, error) {
	return execAtLocal(filepath.Join(binDir, "kubectl"), nil, args...)
}

func kubectlWithInput(input []byte, args ...string) ([]byte, []byte, error) {
	return execAtLocal(filepath.Join(binDir, "kubectl"), input, args...)
}
