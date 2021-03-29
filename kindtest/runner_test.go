package kindtest

import (
	"context"
	"errors"
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testRunner() {
	ctx := context.Background()

	It("should deploy manager successfully and have it deploy deployment", func() {
		By("confirming all controller pods are ready")
		Eventually(func() error {
			return isDeploymentReady(
				"controller-manager",
				systemNS,
				2,
			)
		}).ShouldNot(HaveOccurred())

		By("confirming all runner pods are ready")
		Eventually(func() error {
			return isDeploymentReady(
				poolName,
				runnerNS,
				numRunners,
			)
		}).ShouldNot(HaveOccurred())
	})

	It("should register self-hosted runners to GitHub Actions", func() {
		By("counting the number of self-hosted runners fetched via GitHub Actions API")
		pods, err := fetchPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() error {
			return equalNumExistingRunners(ctx, pods, numRunners)
		}, 2*time.Minute, 15*time.Second).ShouldNot(HaveOccurred())
	})

	It("should run a success job on a self-hosted runner Pod and delete the Pod immediately", func() {
		By("getting pods list before triggering workflow dispatch")
		before, err := fetchPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "success" workflow`)
		err = triggerWorkflowDispatch(ctx, "success.yaml")
		Expect(err).ShouldNot(HaveOccurred())

		By("confirming one Pod is recreated")
		Eventually(func() error {
			after, err := fetchPods(runnerNS, runnerSelector)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}, 2*time.Minute, time.Second).ShouldNot(HaveOccurred())
	})

	It("should run a failure job on a self-hosted runner Pod and delete the Pod after a while", func() {
		By("getting pods list before triggering workflow dispatch")
		before, err := fetchPods(runnerNS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "failure" workflow`)
		err = triggerWorkflowDispatch(ctx, "failure.yaml")
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

		// NOTE:
		// Runner env EXTEND_DURATION=30s
		// controller flag --pod-sweep-interval=1s
		By("confirming no Pod is recreated within 20 seconds after job finishes")
		t, err := time.Parse(time.RFC3339, shouldBeDeletedAt)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(now.Add(20 * time.Second).Before(t)).To(BeTrue())
		Expect(now.Add(30 * time.Second).After(t)).To(BeTrue())

		By("confirming one Pod is recreated in 30 seconds after job finishes")
		Eventually(func() error {
			after, err := fetchPods(runnerNS, runnerSelector)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}, 2*time.Minute, time.Second).ShouldNot(HaveOccurred())
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
		Eventually(func() error {
			return equalNumExistingRunners(ctx, pods, 0)
		}, 3*time.Minute, 30*time.Second).ShouldNot(HaveOccurred())
	})
}
