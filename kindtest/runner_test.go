package kindtest

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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
		waitDeployment(runner1NS, runnerPool1Name, runnerPool1Replicas)
		waitRunnerPods(runner1NS, runnerPool1Name, runnerPool1Replicas)

		By("confirming all runner2 pods are ready")
		waitDeployment(runner2NS, runnerPool2Name, runnerPool2Replicas)
		waitRunnerPods(runner2NS, runnerPool2Name, runnerPool2Replicas)
	})

	It("should run the job-success on a runner pod and delete the pod immediately", func() {
		By("running 'job-success' workflow")
		waitRunnerPods(runner1NS, runnerPool1Name, runnerPool1Replicas)
		pushWorkflowFile("job-success.tmpl.yaml", runner1NS, runnerPool1Name)
		assignedPod, status := waitJobCompletion(runner1NS, runnerPool1Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
		})))

		By("confirming the pod deletion")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletionTime: ", *status.DeletionTime)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.DeletionTime))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test1")
	})

	It("should run the job-cancelled on a runner pod and delete the pod immediately", func() {
		By("running 'job-cancelled' workflow")
		waitRunnerPods(runner2NS, runnerPool2Name, runnerPool2Replicas)
		pushWorkflowFile("job-cancelled.tmpl.yaml", runner2NS, runnerPool2Name)
		assignedPod, status := waitJobCompletion(runner2NS, runnerPool2Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("cancelled"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
		})))

		By("confirming the pod deletion")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletionTime: ", *status.DeletionTime)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.DeletionTime))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test2")
	})

	It("should run the job-failure on a runner pod and delete the pod after a while", func() {
		By("running 'job-failure' workflow")
		waitRunnerPods(runner1NS, runnerPool1Name, runnerPool1Replicas)
		pushWorkflowFile("job-failure.tmpl.yaml", runner1NS, runnerPool1Name)
		assignedPod, status := waitJobCompletion(runner1NS, runnerPool1Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("failure"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt.Add(30*time.Second), 3*time.Second)),
			"Extend":       PointTo(BeTrue()),
			"JobInfo":      Not(BeNil()),
		})))

		By("confirming the pod deletion")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletionTime: ", *status.DeletionTime)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.DeletionTime))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test1")
	})

	It("should extend pod with the deletion time API", func() {
		By("running 'job-failure' workflow")
		waitRunnerPods(runner2NS, runnerPool2Name, runnerPool2Replicas)
		pushWorkflowFile("job-failure.tmpl.yaml", runner2NS, runnerPool2Name)
		assignedPod, status := waitJobCompletion(runner2NS, runnerPool2Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("failure"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt.Add(30*time.Second), 3*time.Second)),
			"Extend":       PointTo(BeTrue()),
			"JobInfo":      Not(BeNil()),
		})))

		By("sending request to the pod")
		extendTo := time.Now().Add(45 * time.Second).Truncate(time.Second)
		err := putDeletionTime(assignedPod, extendTo)
		Expect(err).NotTo(HaveOccurred())
		status, err = getStatus(assignedPod)
		Expect(err).NotTo(HaveOccurred())

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("failure"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": PointTo(BeTemporally("==", extendTo)),
			"Extend":       PointTo(BeTrue()),
			"JobInfo":      Not(BeNil()),
		})))

		By("confirming the pod deletion")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletionTime: ", *status.DeletionTime)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.DeletionTime))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test2")
	})

	It("should be successful the job that makes sure invisible environment variables", func() {
		By("running 'check-env' workflow")
		waitRunnerPods(runner1NS, runnerPool1Name, runnerPool1Replicas)
		pushWorkflowFile("check-env.tmpl.yaml", runner1NS, runnerPool1Name)
		assignedPod, status := waitJobCompletion(runner1NS, runnerPool1Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
		})))

		By("confirming the pod deletion")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletionTime: ", *status.DeletionTime)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.DeletionTime))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test1")
	})

	It("should run a setup command", func() {
		By("running 'setup-command' workflow")
		waitRunnerPods(runner2NS, runnerPool2Name, runnerPool2Replicas)
		pushWorkflowFile("setup-command.tmpl.yaml", runner2NS, runnerPool2Name)
		assignedPod, status := waitJobCompletion(runner2NS, runnerPool2Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
		})))

		By("confirming the pod deletion")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletionTime: ", *status.DeletionTime)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.DeletionTime))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test2")
	})

	It("should delete RunnerPool properly", func() {
		By("deleting runner1")
		stdout, stderr, err := kubectl("delete", "runnerpools", "-n", runner1NS, runnerPool1Name)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		By("confirming to delete resources")
		waitDeletion("deployment", runner1NS, runnerPool1Name)
		waitDeletion("secret", runner1NS, "runner-token-"+runnerPool1Name)

		By("confirming runners are deleted from GitHub Actions")
		Eventually(func() error {
			runnerNames, err := fetchAllRunnerNames(runner1NS + "/" + runnerPool1Name)
			if err != nil {
				return err
			}
			if len(runnerNames) != 0 {
				return fmt.Errorf("%d runners still exist: runners %#v", len(runnerNames), runnerNames)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runner2 pods are existing")
		runner2Pods, err := fetchRunnerPods(runner2NS, runnerPool2Name)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(runner2Pods.Items).Should(HaveLen(runnerPool2Replicas))
		runner2PodNames := getPodNames(runner2Pods)

		By("counting the number of self-hosted runners fetched via GitHub Actions API")
		Eventually(func() error {
			runner2Names, err := fetchAllRunnerNames(runner2NS + "/" + runnerPool2Name)
			if err != nil {
				return err
			}
			if len(runner2Names) != runnerPool2Replicas || !cmp.Equal(runner2PodNames, runner2Names) {
				return fmt.Errorf("%d runners should exist: pods %#v runners %#v", runnerPool2Replicas, runner2PodNames, runner2Names)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("deleting runner2")
		stdout, stderr, err = kubectl("delete", "runnerpools", "-n", runner2NS, runnerPool2Name)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		By("confirming to delete resources")
		waitDeletion("deployment", runner2NS, runnerPool2Name)
		waitDeletion("secret", runner2NS, "runner-token-"+runnerPool2Name)

		By("confirming runners are deleted from GitHub Actions")
		Eventually(func() error {
			runnerNames, err := fetchAllRunnerNames(runner2NS + "/" + runnerPool2Name)
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
