package e2e

import (
	"encoding/json"
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func testAgent() {
	const testPodName = "testpod"

	It("should receive a request from Pod and confirm that the Pod is deleted", func() {
		By("getting a test Pod")
		Eventually(func() error {
			stdout, stderr, err := kubectl("get", "pod", "-n", runnerNS, testPodName)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("sending a request to Slack agent")
		stdout, stderr, err := kubectl(
			"exec",
			"-n", runnerNS, testPodName,
			"--",
			"slack-agent", "client",
			"-n", runnerNS, testPodName,
			"-a", "slack-agent",
			"-j", "dummyjob",
		)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		By("confirming that the Pod has an annotation")
		Eventually(func() error {
			stdout, stderr, err := kubectl(
				"get", "pod",
				"-n", runnerNS, testPodName,
				"-o", "json",
			)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			var pod corev1.Pod
			err = json.Unmarshal(stdout, &pod)
			if err != nil {
				return err
			}
			if _, ok := pod.Annotations[constants.PodDeletionTimeKey]; !ok {
				return fmt.Errorf(
					"%s in %s should have %s annotation",
					pod.Name,
					pod.Namespace,
					constants.PodDeletionTimeKey,
				)
			}
			return nil
		}, 30*time.Second).ShouldNot(HaveOccurred())
	})
}
