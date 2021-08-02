package kindtest

import (
	"errors"
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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
			return isDeploymentReady(runner1PoolName, runner1NS, numRunners)
		}).ShouldNot(HaveOccurred())

		By("confirming all runner2 pods are ready")
		Eventually(func() error {
			return isDeploymentReady(runner2PoolName, runner2NS, numRunners)
		}).ShouldNot(HaveOccurred())
	})

	It("should register self-hosted runners to GitHub Actions", func() {
		By("confirming runners on GitHub Actions: " + runner1NS + "/" + runner1PoolName)
		Eventually(func() error {
			runner1Pods, err := fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			if len(runner1Pods.Items) != numRunners {
				return fmt.Errorf("runner1Pods length expected %d, actual %d", numRunners, len(runner1Pods.Items))
			}
			runner1PodNames := getPodNames(runner1Pods)
			return compareExistingRunners(runner1NS+"/"+runner1PoolName, runner1PodNames)
		}).ShouldNot(HaveOccurred())

		By("confirming runners on GitHub Actions: " + runner2NS + "/" + runner2PoolName)
		Eventually(func() error {
			runner2Pods, err := fetchRunnerPods(runner2NS, runner2PoolName)
			if err != nil {
				return err
			}
			if len(runner2Pods.Items) != numRunners {
				return fmt.Errorf("runner2Pods length expected %d, actual %d", numRunners, len(runner2Pods.Items))
			}
			runner2PodNames := getPodNames(runner2Pods)
			return compareExistingRunners(runner2NS+"/"+runner2PoolName, runner2PodNames)
		}).ShouldNot(HaveOccurred())
	})

	It("should run the job-success on a self-hosted runner Pod and delete the Pod immediately", func() {
		By("getting pods list before running workflow")
		var before *corev1.PodList
		Eventually(func() error {
			var err error
			before, err = fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			if len(before.Items) != numRunners {
				return fmt.Errorf("runner1Pods length expected %d, actual %d", numRunners, len(before.Items))
			}
			beforeNames := getPodNames(before)
			return compareExistingRunners(runner1NS+"/"+runner1PoolName, beforeNames)
		}).ShouldNot(HaveOccurred())

		By(`running "job-success" workflow`)
		pushWorkflowFile("job-success.tmpl.yaml", runner1NS, runner1PoolName)

		By("confirming one Pod is recreated")
		var delPodNames []string
		Eventually(func() error {
			after, err := fetchRunnerPods(runner1NS, runner1PoolName)
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
					"kubectl logs -n %s -l app.kubernetes.io/component=slack-agent | grep \"success to send slack message\" | grep -q %s",
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
		By("getting pods list before running workflow")
		var before *corev1.PodList
		Eventually(func() error {
			var err error
			before, err = fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			if len(before.Items) != numRunners {
				return fmt.Errorf("runner1Pods length expected %d, actual %d", numRunners, len(before.Items))
			}
			beforeNames := getPodNames(before)
			return compareExistingRunners(runner1NS+"/"+runner1PoolName, beforeNames)
		}).ShouldNot(HaveOccurred())

		By(`running "job-failure" workflow`)
		pushWorkflowFile("job-failure.tmpl.yaml", runner1NS, runner1PoolName)

		By("confirming the job is finished and get deletion time from API of one Pod")
		var podName string
		var deletionTime time.Time
		Eventually(func() error {
			// DEBUG
			// stdout, _, _ := kubectl("get", "pods", "-A")
			// fmt.Println("=== kubectl get pod -A")
			// fmt.Println(string(stdout))

			after, err := fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			podName, deletionTime = findPodToBeDeleted(after)
			if podName == "" {
				return errors.New("one pod should get deletion time from /" + constants.DeletionTimeEndpoint)
			}
			return nil
		}, 3*time.Minute, time.Second).ShouldNot(HaveOccurred())

		now := time.Now().UTC()
		fmt.Println("====== Pod: " + podName)
		fmt.Println("====== Current time:  " + now.Format(time.RFC3339))
		fmt.Println("====== Deletion time: " + deletionTime.Format(time.RFC3339))

		By("confirming the timestamp value is around 30 sec later from now")
		Expect(deletionTime.After(now.Add(20 * time.Second))).To(BeTrue())
		Expect(deletionTime.Before(now.Add(30 * time.Second))).To(BeTrue())

		By("confirming one Pod is recreated")
		Eventually(func() error {
			after, err := fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}).ShouldNot(HaveOccurred())
		fmt.Println("====== Pod was actually deleted at " + time.Now().UTC().Format(time.RFC3339))
	})

	It("should be successful the job that makes sure invisible environment variables", func() {
		By("getting pods list before running workflow")
		var before *corev1.PodList
		Eventually(func() error {
			var err error
			before, err = fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			if len(before.Items) != numRunners {
				return fmt.Errorf("runner1Pods length expected %d, actual %d", numRunners, len(before.Items))
			}
			beforeNames := getPodNames(before)
			return compareExistingRunners(runner1NS+"/"+runner1PoolName, beforeNames)
		}).ShouldNot(HaveOccurred())

		By(`running "check-env" workflow that makes sure invisible environment variables`)
		pushWorkflowFile("check-env.tmpl.yaml", runner1NS, runner1PoolName)

		By("confirming the job is finished and get deletion time from API of one Pod")
		var podName string
		var deletionTime time.Time
		Eventually(func() error {
			// DEBUG
			// stdout, _, _ := kubectl("get", "pods", "-A")
			// fmt.Println("=== kubectl get pod -A")
			// fmt.Println(string(stdout))

			after, err := fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			podName, deletionTime = findPodToBeDeleted(after)
			if podName == "" {
				return errors.New("one pod should get deletion time from /" + constants.DeletionTimeEndpoint)
			}
			return nil
		}, 3*time.Minute, time.Second).ShouldNot(HaveOccurred())

		now := time.Now().UTC()
		fmt.Println("====== Pod: " + podName)
		fmt.Println("====== Current time:  " + now.Format(time.RFC3339))
		fmt.Println("====== Deletion time: " + deletionTime.Format(time.RFC3339))

		By("confirming the timestamp value before now")
		Expect(deletionTime.Before(now)).To(BeTrue())

		By("confirming one Pod is recreated")
		Eventually(func() error {
			after, err := fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}).ShouldNot(HaveOccurred())
		fmt.Println("====== Pod was actually deleted at " + time.Now().UTC().Format(time.RFC3339))
	})

	It("should run a setup command", func() {
		By("getting pods list before running workflow")
		var before *corev1.PodList
		Eventually(func() error {
			var err error
			before, err = fetchRunnerPods(runner2NS, runner2PoolName)
			if err != nil {
				return err
			}
			if len(before.Items) != numRunners {
				return fmt.Errorf("runner1Pods length expected %d, actual %d", numRunners, len(before.Items))
			}
			beforeNames := getPodNames(before)
			return compareExistingRunners(runner2NS+"/"+runner2PoolName, beforeNames)
		}).ShouldNot(HaveOccurred())

		By(`running "setup-command" workflow`)
		pushWorkflowFile("setup-command.tmpl.yaml", runner2NS, runner2PoolName)

		By("confirming the job is finished and get deletion time from API of one Pod")
		var podName string
		var deletionTime time.Time
		Eventually(func() error {
			// DEBUG
			// stdout, _, _ := kubectl("get", "pods", "-A")
			// fmt.Println("=== kubectl get pod -A")
			// fmt.Println(string(stdout))

			after, err := fetchRunnerPods(runner2NS, runner2PoolName)
			if err != nil {
				return err
			}
			podName, deletionTime = findPodToBeDeleted(after)
			if podName == "" {
				return errors.New("one pod should get deletion time from /" + constants.DeletionTimeEndpoint)
			}
			return nil
		}, 3*time.Minute, time.Second).ShouldNot(HaveOccurred())

		now := time.Now().UTC()
		fmt.Println("====== Pod: " + podName)
		fmt.Println("====== Current time:  " + now.Format(time.RFC3339))
		fmt.Println("====== Deletion time: " + deletionTime.Format(time.RFC3339))

		By("confirming the timestamp value before now")
		Expect(deletionTime.Before(now)).To(BeTrue())

		By("confirming one Pod is recreated")
		Eventually(func() error {
			after, err := fetchRunnerPods(runner2NS, runner2PoolName)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}).ShouldNot(HaveOccurred())
		fmt.Println("====== Pod was actually deleted at " + time.Now().UTC().Format(time.RFC3339))
	})

	It("should delete after PUT request to a Pod", func() {
		By("getting pods list before API request")
		before, err := fetchRunnerPods(runner1NS, runner1PoolName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(before.Items).Should(HaveLen(numRunners))
		Eventually(func() error {
			next, err := fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			err = equalNumRecreatedPods(before, next, 0)
			if err != nil {
				before = next
				return err
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("PUT request to one Pod")
		fmt.Println("PUT request to ", before.Items[0].Name)
		putDeletionTime(&before.Items[0], time.Now().UTC())

		By("confirming PUT request and get deletion time from API of one Pod")
		var podName string
		var deletionTime time.Time
		Eventually(func() error {
			after, err := fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			podName, deletionTime = findPodToBeDeleted(after)
			if podName == "" {
				return errors.New("one pod should get deletion time from /" + constants.DeletionTimeEndpoint)
			}
			return nil
		}, 3*time.Minute, time.Second).ShouldNot(HaveOccurred())

		now := time.Now().UTC()
		fmt.Println("====== Pod: " + podName)
		fmt.Println("====== Current time:  " + now.Format(time.RFC3339))
		fmt.Println("====== Deletion time: " + deletionTime.Format(time.RFC3339))

		By("confirming the timestamp value before now")
		Expect(deletionTime.Before(now)).To(BeTrue())

		By("confirming one Pod is recreated")
		Eventually(func() error {
			after, err := fetchRunnerPods(runner1NS, runner1PoolName)
			if err != nil {
				return err
			}
			return equalNumRecreatedPods(before, after, 1)
		}).ShouldNot(HaveOccurred())
		fmt.Println("====== Pod was actually deleted at " + time.Now().UTC().Format(time.RFC3339))
	})

	It("should delete RunnerPool properly", func() {
		By("deleting runner1")
		stdout, stderr, err := kubectl("delete", "runnerpools", "-n", runner1NS, runner1PoolName)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		Eventually(func() error {
			_, _, err := kubectl("get", "deployment", "-n", runner1NS, runner1PoolName)
			if err == nil {
				return errors.New("deployment is not deleted yet")
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runners are deleted from GitHub Actions")
		Eventually(func() error {
			runnerNames, err := fetchRunnerNames(runner1NS + "/" + runner1PoolName)
			if err != nil {
				return err
			}
			if len(runnerNames) != 0 {
				return fmt.Errorf("%d runners still exist: runners %#v", len(runnerNames), runnerNames)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runner2 pods are existing")
		runner2Pods, err := fetchRunnerPods(runner2NS, runner2PoolName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(runner2Pods.Items).Should(HaveLen(numRunners))
		runner2PodNames := getPodNames(runner2Pods)

		By("counting the number of self-hosted runners fetched via GitHub Actions API")
		Eventually(func() error {
			runner2Names, err := fetchRunnerNames(runner2NS + "/" + runner2PoolName)
			if err != nil {
				return err
			}
			if len(runner2Names) != numRunners || !cmp.Equal(runner2PodNames, runner2Names) {
				return fmt.Errorf("%d runners should exist: pods %#v runners %#v", numRunners, runner2PodNames, runner2Names)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("deleting runner2")
		stdout, stderr, err = kubectl("delete", "runnerpools", "-n", runner2NS, runner2PoolName)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		Eventually(func() error {
			_, _, err := kubectl("get", "deployment", "-n", runner2NS, runner2PoolName)
			if err == nil {
				return errors.New("deployment is not deleted yet")
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runners are deleted from GitHub Actions")
		Eventually(func() error {
			runnerNames, err := fetchRunnerNames(runner2NS + "/" + runner2PoolName)
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
