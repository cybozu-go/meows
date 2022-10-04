package kindtest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/runner"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v41/github"
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

func createGitHubCredSecret(namespace, name, appID, appInstallationID, privateKeyPath string) {
	stdout, stderr, err := kubectl("create", "secret", "generic", name,
		"-n", namespace,
		"--from-literal=app-id="+appID,
		"--from-literal=app-installation-id="+appInstallationID,
		"--from-file=app-private-key="+privateKeyPath,
	)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
}

func waitDeployment(namespace, name string, replicas int) {
	EventuallyWithOffset(1, func() error {
		stdout, stderr, err := kubectl("get", "deployment", name, "-n", namespace, "-o", "json")
		if err != nil {
			return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		}

		d := new(appsv1.Deployment)
		err = json.Unmarshal(stdout, d)
		if err != nil {
			return err
		}

		if int(d.Status.AvailableReplicas) != replicas {
			return fmt.Errorf("AvailableReplicas is not %d: %d", replicas, int(d.Status.AvailableReplicas))
		}

		return nil
	}).ShouldNot(HaveOccurred())
}

func isNotFoundFromStderr(stderr []byte) bool {
	return strings.Contains(string(stderr), "(NotFound)")
}

func fetchRunnerPods(namespace, runnerpool string) (*corev1.PodList, error) {
	selector := fmt.Sprintf("%s=%s", constants.AppInstanceLabelKey, runnerpool)
	stdout, stderr, err := kubectl("get", "pods", "-n", namespace, "-l", selector, "-o", "json")
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

func getPodNames(pods *corev1.PodList) []string {
	l := make([]string, len(pods.Items))
	for i := range pods.Items {
		l[i] = pods.Items[i].Name
	}
	sort.Strings(l)
	return l
}

func isPodReady(po *corev1.Pod) bool {
	for i := range po.Status.Conditions {
		cond := &po.Status.Conditions[i]
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func getStatus(po *corev1.Pod) (*runner.Status, error) {
	stdout, stderr, err := kubectl("exec", po.Name, "-n", po.Namespace,
		"--", "curl", "-s", fmt.Sprintf("localhost:%d/%s", constants.RunnerListenPort, constants.StatusEndPoint))
	if err != nil {
		return nil, fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
	}
	st := new(runner.Status)
	err = json.Unmarshal(stdout, st)
	if err != nil {
		return nil, err
	}

	return st, nil
}

func putDeletionTime(po *corev1.Pod, tm time.Time) error {
	stdout, stderr, err := kubectl("exec", po.Name, "-n", po.Namespace,
		"--", "curl", "-s", "-XPUT", fmt.Sprintf("localhost:%d/%s", constants.RunnerListenPort, constants.DeletionTimeEndpoint),
		"-H", "Content-Type: application/json",
		"-d", fmt.Sprintf("{\"deletion_time\":\"%s\"}", tm.Format(time.RFC3339)))
	if err != nil {
		return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
	}
	return nil
}

func findDebuggingPod(pods *corev1.PodList) (*corev1.Pod, *runner.Status) {
	for i := range pods.Items {
		pod := &pods.Items[i]
		if !isPodReady(pod) {
			continue
		}
		st, err := getStatus(pod)
		if err != nil || st.State != constants.RunnerPodStateDebugging {
			continue
		}
		return pod, st
	}
	return nil, nil
}

func waitJobCompletion(namespace, runnerpool string) (*corev1.Pod, *runner.Status) {
	var po *corev1.Pod
	var status *runner.Status
	EventuallyWithOffset(1, func() error {
		pods, err := fetchRunnerPods(namespace, runnerpool)
		if err != nil {
			return err
		}
		po, status = findDebuggingPod(pods)
		if po == nil {
			return errors.New("one pod should become debugging state")
		}
		return nil
	}, 3*time.Minute, 500*time.Millisecond).ShouldNot(HaveOccurred())
	return po, status
}

func waitRunnerPodTerminating(namespace, name string) time.Time {
	var tm time.Time
	EventuallyWithOffset(1, func() error {
		stdout, stderr, err := kubectl("get", "pod", "-n", namespace, name, "-o", "json")
		if isNotFoundFromStderr(stderr) {
			tm = time.Now()
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to get pod; stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		}

		pod := new(corev1.Pod)
		err = json.Unmarshal(stdout, pod)
		if err != nil {
			return fmt.Errorf("stdout: %s, err: %v", stdout, err)
		}
		if pod.DeletionTimestamp == nil {
			return fmt.Errorf("pod is not deleted")
		}
		tm = pod.DeletionTimestamp.Time
		return nil
	}, 3*time.Minute, 500*time.Millisecond).ShouldNot(HaveOccurred())
	return tm
}

func waitDeletion(kind, namespace, name string) {
	EventuallyWithOffset(1, func() error {
		stdout, stderr, err := kubectl("get", kind, "-n", namespace, name)
		if !isNotFoundFromStderr(stderr) {
			return fmt.Errorf("%s %s/%s is not deleted yet; stdout: %s, stderr: %s, err: %v", kind, namespace, name, stdout, stderr, err)
		}
		return nil
	}).ShouldNot(HaveOccurred())
}

func slackMessageShouldBeSent(pod *corev1.Pod, channel string) {
	// When a message is successfully sent, the following log will be output from one of slack-agent pods.
	// {"level":"info","ts":1632841077.9362473,"caller":"agent/server.go:161","msg":"success to send slack message","pod":"kindtest-2021-09-28-145507-test-runner1/runnerpool1-84c6ff54f-tn89r","channel":"#test1"}

	stdout, stderr, err := kubectl("logs", "-n", controllerNS, "-l", "app.kubernetes.io/component=slack-agent")
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to get slack-agent log, stdout: %s, stderr: %s, err: %v", stdout, stderr, err)

	podName := pod.Namespace + "/" + pod.Name
	var matchLine string
	reader := bufio.NewReader(bytes.NewReader(stdout))
	for {
		line, isPrefix, err := reader.ReadLine()
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "no match line, pod: %d, stdout: %s", podName, stdout)
		ExpectWithOffset(1, isPrefix).NotTo(BeTrue(), "too long line, line: %s", line)
		if strings.Contains(string(line), podName) {
			matchLine = string(line)
			break
		}
	}
	ExpectWithOffset(1, matchLine).To(ContainSubstring("success to send slack message"), "msg is not match")
	ExpectWithOffset(1, matchLine).To(ContainSubstring(channel), "channel is not match")
}

func fetchOnlineRunnerNames(repoName, label string) ([]string, error) {
	return fetchRunnerNames(repoName, label, "online")
}

func fetchAllRepositoryRunnerNames(label string) ([]string, error) {
	return fetchRunnerNames(repoName, label, "")
}

func fetchAllOrganizationRunnerNames(label string) ([]string, error) {
	return fetchRunnerNames("", label, "")
}

func fetchRunnerNames(repoName, label, status string) ([]string, error) {
	var runners *github.Runners
	var res *github.Response
	var err error
	ctx := context.Background()
	opts := github.ListOptions{Page: 0, PerPage: 100}
	if repoName == "" {
		runners, res, err = githubClient.Actions.ListOrganizationRunners(ctx, orgName, &opts)
	} else {
		runners, res, err = githubClient.Actions.ListRunners(ctx, orgName, repoName, &opts)
	}
	if err != nil {
		return nil, err
	}
	if res.NextPage != 0 {
		panic("more than 100 runners exist: please delete them manually before running a test")
	}

	runnerNames := []string{}
OUTER:
	for _, r := range runners.Runners {
		if r.GetName() == "" || (status != "" && r.GetStatus() != status) {
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

func waitRepositoryRunnerPods(namespace, runnerpool string, replicas int) []string {
	return waitRunnerPods(namespace, runnerpool, replicas, repoName)
}

func waitOrganizationRunnerPods(namespace, runnerpool string, replicas int) []string {
	return waitRunnerPods(namespace, runnerpool, replicas, "")
}

func waitRunnerPods(namespace, runnerpool string, replicas int, repoName string) []string {
	var podNames []string
	EventuallyWithOffset(2, func() error {
		pods, err := fetchRunnerPods(namespace, runnerpool)
		if err != nil {
			return err
		}
		if len(pods.Items) != replicas {
			return fmt.Errorf("pods length expected %d, actual %d", replicas, len(pods.Items))
		}
		podNames = getPodNames(pods)

		// checking runners is online.
		label := namespace + "/" + runnerpool
		runnerNames, err := fetchOnlineRunnerNames(repoName, label)
		if err != nil {
			return err
		}
		if len(runnerNames) != len(podNames) || !cmp.Equal(runnerNames, podNames) {
			return fmt.Errorf("%d runners should exist: pods %#v runners %#v", len(podNames), podNames, runnerNames)
		}
		return nil
	}).ShouldNot(HaveOccurred())

	return podNames
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
	err := os.WriteFile(filepath.Join(testRepoWorkDir, workflowFile), buf.Bytes(), 0644)
	Expect(err).ShouldNot(HaveOccurred())

	gitSafe("add", workflowFile)
	gitSafe("commit", "-m", "["+testID+"] "+filename)
	gitSafe("push", "--set-upstream", "origin", testBranch)
}
