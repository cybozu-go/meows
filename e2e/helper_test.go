package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/google/go-github/v33/github"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = kubectlWithInput

func kubectl(args ...string) ([]byte, []byte, error) {
	return execAtLocal(filepath.Join(binDir, "kubectl"), nil, args...)
}

func kubectlWithInput(input []byte, args ...string) ([]byte, []byte, error) {
	return execAtLocal(filepath.Join(binDir, "kubectl"), input, args...)
}

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

func fetchPods(namespace, selector string) (*corev1.PodList, error) {
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

func isDeploymentReady(name, namespace string, replicas int) error {
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

func equalNumRecreatedPods(before, after *corev1.PodList, numRecreated int) error {
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
	if cntNotFound != numRecreated {
		return fmt.Errorf(
			"%d pod should be recreated: before %#v after %#v",
			numRecreated,
			getPodNames(before),
			getPodNames(after),
		)
	}
	return nil
}

func equalNumExistingRunners(
	ctx context.Context,
	pods *corev1.PodList,
	numExisting int,
) error {
	runners, res, err := githubClient.Actions.ListRunners(
		ctx,
		orgName,
		repoName,
		&github.ListOptions{Page: 0, PerPage: 100},
	)
	if err != nil {
		return err
	}
	if res.NextPage != 0 {
		panic("more than 100 runners exist: please delete them manually before running a test")
	}

	runnerMap := make(map[string]struct{})
	for _, r := range runners.Runners {
		if r == nil || r.Name == nil {
			continue
		}
		runnerMap[*r.Name] = struct{}{}
	}

	found := make([]string, 0, len(pods.Items))
	for _, p := range pods.Items {
		if _, ok := runnerMap[p.Name]; ok {
			found = append(found, p.Name)
		}
	}

	if len(found) != numExisting {
		return fmt.Errorf(
			"%d runners should exist: pods %#v runners %#v",
			numExisting,
			found,
			runnerMap,
		)
	}
	return nil
}

func triggerWorkflowDispatch(ctx context.Context, workflowName string) error {
	res, err := githubClient.Actions.CreateWorkflowDispatchEventByFileName(
		ctx,
		orgName,
		repoName,
		workflowName,
		github.CreateWorkflowDispatchEventRequest{Ref: "main"},
	)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("got invalid status code: %d", res.StatusCode)
	}
	return nil
}
