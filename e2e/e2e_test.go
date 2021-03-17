package e2e

import (
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testE2E() {
	runnerSelector := fmt.Sprintf(
		"%s=%s,%s=%s",
		constants.RunnerOrgLabelKey, orgName,
		constants.RunnerRepoLabelKey, repoName,
	)

	It("should prepare for the test", func() {
		By("confirming no runner is registered")
		err := fetchAndCompareRunners(0)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should deploy manager successfully and make deployment", func() {
		By("counting the number of ready deployments")
		Eventually(func() error {
			return confirmDeploymentIsReady(
				"controller-manager",
				systemNS,
				2,
			)
		}).ShouldNot(HaveOccurred())

		By("counting the number of ready deployments")
		Eventually(func() error {
			return confirmDeploymentIsReady(
				"runnerpool-sample",
				runnerNS,
				numRunners,
			)
		}).ShouldNot(HaveOccurred())
	})

	It("should register self-hosted runner to GitHub Actions", func() {
		By("counting the number of self-hosted runners fetched with GitHub Actions API")
		Eventually(func() error {
			return fetchAndCompareRunners(numRunners)
		}, 3*time.Minute, 10*time.Second).ShouldNot(HaveOccurred())
	})

	It("should run a job on a self-hosted runner Pod and delete Pod immediately", func() {
		// TODO: description

		By("getting pods list before triggering workflow dispatch")
		before, err := getPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "success" workflow`)
		err = triggerWorkflowDispatch("success")
		Expect(err).ShouldNot(HaveOccurred())

		By("confirming one Pod is recreated after job finishes")
		Eventually(func() error {
			after, err := getPods(runnerNS, runnerSelector)
			if err != nil {
				return err
			}
			return comparePodNames(before, after, 1)
		}).ShouldNot(HaveOccurred())
	})

	It("should run a failure job on a self-hosted runner Pod and delete Pod after a while", func() {
		// TODO: description

		By("getting pods list before triggering workflow dispatch")
		before, err := getPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "fail" workflow`)
		err = triggerWorkflowDispatch("fail")
		Expect(err).ShouldNot(HaveOccurred())

		By("confirming no Pod is recreated within 1 minutes after job finishes")
		Eventually(func() error {
			after, err := getPods(runnerNS, runnerSelector)
			if err != nil {
				return err
			}
			return comparePodNames(before, after, 0)
		}, 10*time.Second, time.Minute).ShouldNot(HaveOccurred())

		By("confirming one Pod is recreated after 1 minutes has passed after job finishes")
		Eventually(func() error {
			after, err := getPods(runnerNS, runnerSelector)
			if err != nil {
				return err
			}
			return comparePodNames(before, after, 1)
		}, 10*time.Second, time.Minute).ShouldNot(HaveOccurred())
	})

	It("should delete deployment properly for finalizer", func() {
		By("deleting RunnerPool")
		stdout, stderr, err := kubectl("delete", "runnerpools", "-n", runnerNS)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))
	})

	It("should delete runners properly from GitHub Actions", func() {
		By("confirming runners are deleted with GitHub Actions API")
		Eventually(func() error {
			return fetchAndCompareRunners(0)
		}, 3*time.Minute, 10*time.Second).ShouldNot(HaveOccurred())
	})
}
