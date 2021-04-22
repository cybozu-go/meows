package kindtest

import (
	"errors"
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testRunner() {
	It("should register self-hosted runners to GitHub Actions", func() {
		stdout, stderr, err := kustomizeBuild("./manifests/runnerpool")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-f", "-")

		By("confirming all runner pods are ready")
		Eventually(func() error {
			return isDeploymentReady("runnerpool-sample", runnerNS, 3)
		}).ShouldNot(HaveOccurred())
	})

	It("should register self-hosted runners to GitHub Actions", func() {
		By("counting the number of self-hosted runners fetched via GitHub Actions API")
		pods, err := fetchPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())

		// Set interval and limit considering rate limit.
		Eventually(func() error {
			return equalNumExistingRunners(pods, numRunners)
		}, 5*time.Minute, 15*time.Second).ShouldNot(HaveOccurred())
	})

	It("should run a success job on a self-hosted runner Pod and delete the Pod immediately", func() {
		By("getting pods list before triggering workflow dispatch")
		before, err := fetchPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "success" workflow`)
		err = triggerWorkflowDispatch("success.yaml")
		Expect(err).ShouldNot(HaveOccurred())

		By("confirming one Pod is recreated")
		var delPodNames []string
		Eventually(func() error {
			after, err := fetchPods(runnerNS, runnerSelector)
			delPodNames, _ = getRecretedPods(before, after)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}, 5*time.Minute, time.Second).ShouldNot(HaveOccurred())

		By("confirming that one of the slack-agent pod emitted a dummy message to stdout")
		//{"level":"info","ts":1623123384.4349568,"caller":"agent/server.go:141","msg":"success to send slack message","pod":"test-runner/runnerpool-sample-5f4fbff6bb-wpjq6"}
		Eventually(func() error {
			stdout, stderr, err := execAtLocal(
				"sh", nil,
				"-c", fmt.Sprintf(
					"kubectl logs -n %s -l app=slack-agent | grep \"success to send slack message\" | grep -q %s",
					runnerNS, delPodNames[0],
				),
			)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).ShouldNot(HaveOccurred())
	})

	It("should run a failure job on a self-hosted runner Pod and delete the Pod after a while", func() {
		By("getting pods list before triggering workflow dispatch")
		before, err := fetchPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "failure" workflow`)
		err = triggerWorkflowDispatch("failure.yaml")
		Expect(err).ShouldNot(HaveOccurred())

		By("confirming the job is finished and one Pod has deletion time annotation")
		var shouldBeDeletedAt string
		Eventually(func() error {
			after, err := fetchPods(runnerNS, runnerSelector)
			if err != nil {
				return err
			}
			for _, po := range after.Items {
				if v, ok := po.Annotations[constants.PodDeletionTimeKey]; ok {
					fmt.Println("====== Pod should be deleted at " + v)
					shouldBeDeletedAt = v
					return nil
				}
			}
			return errors.New("one pod should have annotation " + constants.PodDeletionTimeKey)
		}, time.Minute, time.Second).ShouldNot(HaveOccurred())
		now := time.Now().UTC()
		fmt.Println("====== Current time is " + now.Format(time.RFC3339))

		By("confirming the timestamp value is around 30 sec later from now")
		t, err := time.Parse(time.RFC3339, shouldBeDeletedAt)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(t.After(now.Add(20 * time.Second))).To(BeTrue())
		Expect(t.Before(now.Add(30 * time.Second))).To(BeTrue())

		By("confirming one Pod is recreated")
		Eventually(func() error {
			after, err := fetchPods(runnerNS, runnerSelector)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}, 5*time.Minute, time.Second).ShouldNot(HaveOccurred())
		fmt.Println("====== Pod was actually deleted at " + time.Now().UTC().Format(time.RFC3339))
	})

	It("should delete RunnerPool properly", func() {
		By("getting pods list before deleting RunnerPool")
		pods, err := fetchPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())

		By("deleting deployment by finalizer")
		stdout, stderr, err := kubectl("delete", "runnerpools", "-n", runnerNS, poolName)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		Eventually(func() error {
			_, _, err := kubectl("get", "deployment", "-n", runnerNS, poolName)
			if err == nil {
				return errors.New("deployment is not deleted yet")
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runners are deleted via GitHub Actions API")
		// Set interval and limit considering rate limit.
		Eventually(func() error {
			return equalNumExistingRunners(pods, 0)
		}, 5*time.Minute, 30*time.Second).ShouldNot(HaveOccurred())
	})
}
