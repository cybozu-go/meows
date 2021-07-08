package kindtest

import (
	"errors"
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testRunner() {
	It("should create runner pods", func() {
		By("creating runnerpool1")
		stdout, stderr, err := kustomizeBuild("./manifests/runnerpool1")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-n", runner1NS, "-f", "-")

		By("creating runnerpool2")
		stdout, stderr, err = kustomizeBuild("./manifests/runnerpool2")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-n", runner2NS, "-f", "-")

		By("confirming all runner1 pods are ready")
		Eventually(func() error {
			return isDeploymentReady(runnerPool1Name, runner1NS, numRunners)
		}).ShouldNot(HaveOccurred())

		By("confirming all runner2 pods are ready")
		Eventually(func() error {
			return isDeploymentReady(runnerPool2Name, runner2NS, numRunners)
		}).ShouldNot(HaveOccurred())
	})

	It("should register self-hosted runners to GitHub Actions", func() {
		By("getting runner1 pods name")
		runner1Pods, err := fetchPods(runner1NS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(runner1Pods.Items).Should(HaveLen(numRunners))
		runner1PodNames := getPodNames(runner1Pods)

		By("getting runner2 pods name")
		runner2Pods, err := fetchPods(runner2NS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(runner2Pods.Items).Should(HaveLen(numRunners))
		runner2PodNames := getPodNames(runner2Pods)

		By("confirming runners on GitHub Actions: " + runner1NS + "/" + runnerPool1Name)
		Eventually(func() error {
			return compareExistingRunners(runner1NS+"/"+runnerPool1Name, runner1PodNames)
		}).ShouldNot(HaveOccurred())

		By("confirming runners on GitHub Actions: " + runner2NS + "/" + runnerPool2Name)
		Eventually(func() error {
			return compareExistingRunners(runner2NS+"/"+runnerPool2Name, runner2PodNames)
		}).ShouldNot(HaveOccurred())
	})

	It("should run the job-success on a self-hosted runner Pod and delete the Pod immediately", func() {
		By("getting pods list before triggering workflow dispatch")
		before, err := fetchPods(runner1NS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "job-success" workflow`)
		pushWorkflowFile("job-success.tmpl.yaml", runner1NS, runnerPool1Name)

		By("confirming one Pod is recreated")
		var delPodNames []string
		Eventually(func() error {
			after, err := fetchPods(runner1NS, runnerSelector)
			delPodNames, _ = getRecretedPods(before, after)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}).ShouldNot(HaveOccurred())

		By("confirming that one of the slack-agent pod emitted a dummy message to stdout")
		//{"level":"info","ts":1623123384.4349568,"caller":"agent/server.go:141","msg":"success to send slack message","pod":"test-runner/runnerpool-sample-5f4fbff6bb-wpjq6"}
		Eventually(func() error {
			stdout, stderr, err := execAtLocal(
				"sh", nil,
				"-c", fmt.Sprintf(
					"kubectl logs -n %s -l app=slack-agent | grep \"success to send slack message\" | grep -q %s",
					controllerNS, delPodNames[0],
				),
			)
			if err != nil {
				return fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
			}
			return nil
		}).ShouldNot(HaveOccurred())
	})

	It("should run the job-failure on a self-hosted runner Pod and delete the Pod after a while", func() {
		By("getting pods list before triggering workflow dispatch")
		before, err := fetchPods(runner1NS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "job-failure" workflow`)
		pushWorkflowFile("job-failure.tmpl.yaml", runner1NS, runnerPool1Name)

		By("confirming the job is finished and get deletion time from API of one Pod")
		var shouldBeDeletedAt string
		Eventually(func() error {
			stdout, _, _ := kubectl("get", "pods", "-A")
			fmt.Println("=== kubectl get pod -A")
			fmt.Println(string(stdout))

			after, err := fetchPods(runner1NS, runnerSelector)
			if err != nil {
				return err
			}
			for _, po := range after.Items {
				v, err := getDeletionTime(po)
				if err != nil {
					return err
				}
				if v != "" {
					fmt.Println("====== Pod should be deleted at " + v)
					shouldBeDeletedAt = v
					return nil
				}
			}
			return errors.New("one pod should get deletion time from /" + constants.DeletionTimeEndpoint)
		}, 3*time.Minute, time.Second).ShouldNot(HaveOccurred())
		now := time.Now().UTC()
		fmt.Println("====== Current time is " + now.Format(time.RFC3339))

		By("confirming the timestamp value is around 30 sec later from now")
		t, err := time.Parse(time.RFC3339, shouldBeDeletedAt)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(t.After(now.Add(20 * time.Second))).To(BeTrue())
		Expect(t.Before(now.Add(30 * time.Second))).To(BeTrue())

		By("confirming one Pod is recreated")
		Eventually(func() error {
			after, err := fetchPods(runner1NS, runnerSelector)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}).ShouldNot(HaveOccurred())
		fmt.Println("====== Pod was actually deleted at " + time.Now().UTC().Format(time.RFC3339))
	})

	It("should be successful the job that makes sure invisible environment variables", func() {
		By("getting pods list before triggering workflow dispatch")
		before, err := fetchPods(runner1NS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "check-env" workflow that makes sure invisible environment variables`)
		pushWorkflowFile("check-env.tmpl.yaml", runner1NS, runnerPool1Name)

		By("confirming the job is finished and get deletion time from API of one Pod")
		var shouldBeDeletedAt string
		Eventually(func() error {
			stdout, _, _ := kubectl("get", "pods", "-A")
			fmt.Println("=== kubectl get pod -A")
			fmt.Println(string(stdout))

			after, err := fetchPods(runner1NS, runnerSelector)
			if err != nil {
				return err
			}
			for _, po := range after.Items {
				v, err := getDeletionTime(po)
				if err != nil {
					return err
				}
				if v != "" {
					fmt.Println("====== Pod should be deleted at " + v)
					shouldBeDeletedAt = v
					return nil
				}
			}
			return errors.New("one pod should get deletion time from /" + constants.DeletionTimeEndpoint)
		}, 3*time.Minute, time.Second).ShouldNot(HaveOccurred())
		now := time.Now().UTC()
		fmt.Println("====== Current time is " + now.Format(time.RFC3339))

		By("confirming the timestamp value is now")
		t, err := time.Parse(time.RFC3339, shouldBeDeletedAt)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(t.After(now.Add(-5 * time.Second))).To(BeTrue())
		Expect(t.Before(now.Add(5 * time.Second))).To(BeTrue())

		By("confirming one Pod is recreated")
		Eventually(func() error {
			after, err := fetchPods(runner1NS, runnerSelector)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}).ShouldNot(HaveOccurred())
		fmt.Println("====== Pod was actually deleted at " + time.Now().UTC().Format(time.RFC3339))
	})

	It("should run a setup command", func() {
		By("getting pods list before triggering workflow dispatch")
		before, err := fetchPods(runner2NS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))

		By(`running "setup-command" workflow`)
		pushWorkflowFile("setup-command.tmpl.yaml", runner2NS, runnerPool2Name)

		By("confirming the job is finished and get deletion time from API of one Pod")
		var shouldBeDeletedAt string
		Eventually(func() error {
			stdout, _, _ := kubectl("get", "pods", "-A")
			fmt.Println("=== kubectl get pod -A")
			fmt.Println(string(stdout))

			after, err := fetchPods(runner2NS, runnerSelector)
			if err != nil {
				return err
			}
			for _, po := range after.Items {
				v, err := getDeletionTime(po)
				if err != nil {
					return err
				}
				if v != "" {
					fmt.Println("====== Pod should be deleted at " + v)
					shouldBeDeletedAt = v
					return nil
				}
			}
			return errors.New("one pod should get deletion time from /" + constants.DeletionTimeEndpoint)
		}, 3*time.Minute, time.Second).ShouldNot(HaveOccurred())
		now := time.Now().UTC()
		fmt.Println("====== Current time is " + now.Format(time.RFC3339))

		By("confirming the timestamp value is now")
		t, err := time.Parse(time.RFC3339, shouldBeDeletedAt)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(t.After(now.Add(-5 * time.Second))).To(BeTrue())
		Expect(t.Before(now.Add(5 * time.Second))).To(BeTrue())

		By("confirming one Pod is recreated")
		Eventually(func() error {
			after, err := fetchPods(runner2NS, runnerSelector)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}).ShouldNot(HaveOccurred())
		fmt.Println("====== Pod was actually deleted at " + time.Now().UTC().Format(time.RFC3339))
	})

	It("should delete RunnerPool properly", func() {
		By("deleting runner1")
		stdout, stderr, err := kubectl("delete", "runnerpools", "-n", runner1NS, runnerPool1Name)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		Eventually(func() error {
			_, _, err := kubectl("get", "deployment", "-n", runner1NS, runnerPool1Name)
			if err == nil {
				return errors.New("deployment is not deleted yet")
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runners are deleted from GitHub Actions")
		Eventually(func() error {
			runnerNames, err := fetchRunnerNames(runner1NS + "/" + runnerPool1Name)
			if err != nil {
				return err
			}
			if len(runnerNames) != 0 {
				return fmt.Errorf("%d runners still exist: runners %#v", len(runnerNames), runnerNames)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runner2 pods are existing")
		runner2Pods, err := fetchPods(runner2NS, runnerSelector)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(runner2Pods.Items).Should(HaveLen(numRunners))
		runner2PodNames := getPodNames(runner2Pods)

		By("counting the number of self-hosted runners fetched via GitHub Actions API")
		Eventually(func() error {
			runner2Names, err := fetchRunnerNames(runner2NS + "/" + runnerPool2Name)
			if err != nil {
				return err
			}
			if len(runner2Names) != numRunners || !cmp.Equal(runner2PodNames, runner2Names) {
				return fmt.Errorf("%d runners should exist: pods %#v runners %#v", numRunners, runner2PodNames, runner2Names)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("deleting runner2")
		stdout, stderr, err = kubectl("delete", "runnerpools", "-n", runner2NS, runnerPool2Name)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		Eventually(func() error {
			_, _, err := kubectl("get", "deployment", "-n", runner2NS, runnerPool2Name)
			if err == nil {
				return errors.New("deployment is not deleted yet")
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runners are deleted from GitHub Actions")
		Eventually(func() error {
			runnerNames, err := fetchRunnerNames(runner2NS + "/" + runnerPool2Name)
			if err != nil {
				return err
			}
			if len(runnerNames) != 0 {
				return fmt.Errorf("%d runners still exist: runners %#v", len(runnerNames), runnerNames)
			}
			return nil
		}).ShouldNot(HaveOccurred())
	})
}
