package kindtest

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testAgent() {
	It("should receive a request from Pod", func() {
		By("getting a test Pod")
		pods, err := fetchPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(pods.Items).ShouldNot(HaveLen(0))
		podName := pods.Items[0].GetName()

		By("sending a request to Slack agent")
		u := uuid.NewString()
		stdout, stderr, err := kubectl(
			"exec",
			"-n", runnerNS, podName,
			"--",
			"slack-agent", "client",
			"-n", runnerNS, podName,
			"-a", "slack-agent",
			"-w", u,
		)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		By("confirming that one of the slack-agent pod emitted a dummy message to stdout")
		Eventually(func() error {
			stdout, stderr, err := execAtLocal(
				"sh", nil,
				"-c", fmt.Sprintf(
					"kubectl logs -n %s -l app=slack-agent -c notifier | grep -q %s",
					runnerNS, u,
				),
			)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}, 30*time.Second).ShouldNot(HaveOccurred())
	})
}
