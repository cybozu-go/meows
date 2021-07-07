package kindtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v33/github"
	. "github.com/onsi/gomega"
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

func kubectlSafe(args ...string) []byte {
	stdout, stderr, err := kubectl(args...)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
	return stdout
}

func kubectlSafeWithInput(input []byte, args ...string) []byte {
	stdout, stderr, err := kubectlWithInput(input, args...)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
	return stdout
}

func kustomizeBuild(dir string) ([]byte, []byte, error) {
	return execAtLocal(filepath.Join(binDir, "kustomize"), nil, "build", dir)
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

func createNamespace(ns string) {
	stdout, stderr, err := kubectl("create", "namespace", ns)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)

	EventuallyWithOffset(1, func() error {
		stdout, stderr, err := kubectl("get", "sa", "default", "-n", ns)
		if err != nil {
			return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		}
		return nil
	}).Should(Succeed())
}

func getPodNames(pods *corev1.PodList) []string {
	l := make([]string, len(pods.Items))
	for i, v := range pods.Items {
		l[i] = v.Name
	}
	sort.Strings(l)
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

func getRecretedPods(before, after *corev1.PodList) ([]string, []string) {
	delPodNames := make([]string, 0)
	addPodNames := make([]string, 0)

	beforeMap := make(map[string]struct{})
	for _, v := range before.Items {
		beforeMap[v.Name] = struct{}{}
	}

	afterMap := make(map[string]struct{})
	for _, v := range after.Items {
		afterMap[v.Name] = struct{}{}
	}

	for _, v := range before.Items {
		if _, ok := afterMap[v.Name]; !ok {
			delPodNames = append(delPodNames, v.Name)
		}
	}

	for _, v := range after.Items {
		if _, ok := beforeMap[v.Name]; !ok {
			addPodNames = append(addPodNames, v.Name)
		}
	}
	return delPodNames, addPodNames
}

func getDeletionTime(po corev1.Pod) (string, error) {
	stdout, stderr, err := kubectl(
		"exec", po.Name,
		"-n", po.Namespace,
		"--",
		"curl", "-s", fmt.Sprintf("localhost:%d/%s", constants.RunnerListenPort, constants.DeletionTimeEndpoint),
	)
	if err != nil {
		return "", fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
	}
	return strings.TrimRight(string(stdout), "\n"), nil
}

func equalNumRecreatedPods(before, after *corev1.PodList, numRecreated int) error {
	if len(before.Items) != len(after.Items) {
		return fmt.Errorf(
			"length mismatch: expected %#v actual %#v",
			getPodNames(before),
			getPodNames(after),
		)
	}
	delPodNames, addPodNames := getRecretedPods(before, after)
	if len(delPodNames) != numRecreated || len(addPodNames) != numRecreated {
		return fmt.Errorf(
			"%d pod should be recreated: before %#v after %#v",
			numRecreated,
			getPodNames(before),
			getPodNames(after),
		)
	}
	return nil
}

func fetchRunnerNames(label string) ([]string, error) {
	runners, res, err := githubClient.Actions.ListRunners(context.Background(), orgName, repoName, &github.ListOptions{Page: 0, PerPage: 100})
	if err != nil {
		return nil, err
	}
	if res.NextPage != 0 {
		panic("more than 100 runners exist: please delete them manually before running a test")
	}

	runnerNames := []string{}
OUTER:
	for _, r := range runners.Runners {
		if r == nil || r.Name == nil || r.Status == nil || *r.Status != "online" {
			continue
		}

		for _, l := range r.Labels {
			if l.GetName() == label {
				runnerNames = append(runnerNames, *r.Name)
				continue OUTER
			}
		}
	}
	sort.Strings(runnerNames)

	return runnerNames, nil
}

func compareExistingRunners(label string, podNames []string) error {
	runnerNames, err := fetchRunnerNames(label)
	if err != nil {
		return err
	}
	if len(runnerNames) != len(podNames) || !cmp.Equal(runnerNames, podNames) {
		return fmt.Errorf("%d runners should exist: pods %#v runners %#v", len(podNames), podNames, runnerNames)
	}
	return nil
}

func gitSafe(args ...string) {
	var stdout, stderr bytes.Buffer
	command := exec.Command("git", args...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	command.Dir = testRepoWorkDir

	err := command.Run()
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout.String(), stderr.String(), err)
}

func pushWorkflowFile(filename, namespace, runnerPoolName string) {
	workflowFile := filepath.Join(".github", "workflows", "testjob.yaml")

	var buf bytes.Buffer
	tpl := template.Must(template.ParseFiles(filepath.Join("./workflows", filename)))
	tpl.Execute(&buf, map[string]string{
		"Namespace":  namespace,
		"RunnerPool": runnerPoolName,
	})
	err := ioutil.WriteFile(filepath.Join(testRepoWorkDir, workflowFile), buf.Bytes(), 0644)
	Expect(err).ShouldNot(HaveOccurred())

	gitSafe("add", workflowFile)
	gitSafe("commit", "-m", "["+testID+"] "+filename)
	gitSafe("push", "--set-upstream", "origin", testBranch)
}
