package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/google/go-github/v33/github"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func kubectl(args ...string) ([]byte, []byte, error) {
	return execAtLocal(filepath.Join(binDir, "kubectl"), nil, args...)
}

var _ = kubectlWithInput

func kubectlWithInput(input []byte, args ...string) ([]byte, []byte, error) {
	return execAtLocal(filepath.Join(binDir, "kubectl"), input, args...)
}

var _ = execAtLocal

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

func getPodNames(pods *corev1.PodList) []string {
	l := make([]string, len(pods.Items))
	for i, v := range pods.Items {
		l[i] = v.Name
	}
	return l
}

func comparePodNames(before, after *corev1.PodList, numNotFound int) error {
	if len(before.Items) != len(after.Items) {
		return fmt.Errorf(
			"length mismatch: expected %#v actual %#v",
			getPodNames(before),
			getPodNames(after),
		)
	}
	beforeMap := make(map[string]struct{})
	for _, v := range before.Items {
		beforeMap[v.Name] = struct{}{}
	}
	cntNotFound := 0
	for _, v := range after.Items {
		if _, ok := beforeMap[v.Name]; !ok {
			cntNotFound++
		}
	}
	if cntNotFound != numNotFound {
		return fmt.Errorf(
			"one pod should be recreated: expect %#v actual %#v",
			getPodNames(before),
			getPodNames(after),
		)
	}
	return nil
}

func fetchAndCompareRunners(numShouldExist int) error {
	runners, err := fetchRegisterredRunners()
	if err != nil {
		return err
	}

	if len(runners) != numShouldExist {
		names := make([]string, len(runners))
		for _, v := range runners {
			names = append(names, *v.Name)
		}
		return fmt.Errorf(
			"length mismatch: expected %d actual %d, %#v",
			numRunners,
			len(runners),
			names,
		)
	}
	return nil
}

func fetchRegisterredRunners() ([]*github.Runner, error) {
	return nil, nil
}

func triggerWorkflowDispatch(workflow string) error {
	return nil
}

func getPods(namespace, selector string) (*corev1.PodList, error) {
	stdout, stderr, err := kubectl(
		"get", "pods",
		"-n", namespace,
		"-l", selector,
		"-o", "json",
	)
	if err != nil {
		return nil, fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
	}

	pods := new(corev1.PodList)
	err = json.Unmarshal(stdout, pods)
	if err != nil {
		return nil, fmt.Errorf("stdout: %s, err: %v", stdout, err)
	}
	return pods, nil
}

func confirmDeploymentIsReady(name, namespace string, replicas int) error {
	stdout, stderr, err := kubectl(
		"get", "deployment", name,
		"-n", namespace,
		"-o", "json",
	)
	if err != nil {
		return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
	}

	d := new(appsv1.Deployment)
	err = json.Unmarshal(stdout, d)
	if err != nil {
		return err
	}

	if int(d.Status.AvailableReplicas) != replicas {
		return fmt.Errorf(
			"AvailableReplicas is not %d: %d",
			replicas, int(d.Status.AvailableReplicas),
		)
	}
	return nil
}
