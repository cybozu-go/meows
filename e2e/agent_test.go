package e2e

import (
	"encoding/json"
	"fmt"

	constants "github.com/cybozu-go/github-actions-controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

const testPodName = "testpod"

func testAgent() {
	It("should receive a request from Pod and annotate the Pod with the deletion time", func() {
		By("confirming the Pod fot test is created")
		Eventually(func() error {
			stdout, stderr, err := kubectl("get", "pod", testPodName, "-n", runnerNS)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			var pod corev1.Pod
			err = json.Unmarshal(stdout, &pod)
			if err != nil {
				return err
			}

			if _, ok := pod.Annotations[constants.PodDeletionTimeKey]; ok {
				return fmt.Errorf("pod should not have %s annotation", constants.PodDeletionTimeKey)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("sending a request to Slack agent")
		p := struct {
			JobName      string `json:"job_name"`
			PodNamespace string `json:"pod_namespace"`
			PodName      string `json:"pod_name"`
		}{
			"sample",
			runnerNS,
			testPodName,
		}
		b, err := json.Marshal(p)
		Expect(err).ShouldNot(HaveOccurred())

		stdout, stderr, err := kubectl(
			"exec", "-it", "-n", runnerNS, testPodName,
			"--",
			"curl", "slack-agent/slack/fail",
			"-H", "Content-Type: application/json",
			"-d", string(b),
		)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		Eventually(func() error {
			stdout, stderr, err := kubectl("get", "pod", testPodName, "-n", runnerNS)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}

			var pod corev1.Pod
			err = json.Unmarshal(stdout, &pod)
			if err != nil {
				return err
			}

			if _, ok := pod.Annotations[constants.PodDeletionTimeKey]; !ok {
				return fmt.Errorf("pod should have %s annotation", constants.PodDeletionTimeKey)
			}
			return nil
		}).ShouldNot(HaveOccurred())
	})
}
