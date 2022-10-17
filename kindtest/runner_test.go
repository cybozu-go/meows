package kindtest

import (
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func testRunner() {
	It("should create runner pods", func() {
		By("creating repo-runnerpool1")
		createGitHubCredSecret(repoRunner1NS, "meows-github-cred", githubAppID, githubAppInstallationID, githubAppPrivateKeyPath)
		stdout, stderr, err := kustomizeBuild("./manifests/repo-runnerpool1")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-n", repoRunner1NS, "-f", "-")

		By("creating repo-runnerpool2")
		createGitHubCredSecret(repoRunner2NS, "meows-github-cred", githubAppID, githubAppInstallationID, githubAppPrivateKeyPath)
		stdout, stderr, err = kustomizeBuild("./manifests/repo-runnerpool2")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-n", repoRunner2NS, "-f", "-")

		By("creating org-runnerpool1")
		createGitHubCredSecret(orgRunner1NS, "github-cred-foo", githubAppID, githubAppInstallationID, githubAppPrivateKeyPath)
		stdout, stderr, err = kustomizeBuild("./manifests/org-runnerpool1")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-n", orgRunner1NS, "-f", "-")

		By("confirming all repo-runner1 pods are ready")
		waitDeployment(repoRunner1NS, repoRunnerPool1Name, repoRunnerPool1Replicas)
		waitRepositoryRunnerPods(repoRunner1NS, repoRunnerPool1Name, repoRunnerPool1Replicas)

		By("confirming all repo-runner2 pods are ready")
		waitDeployment(repoRunner2NS, repoRunnerPool2Name, repoRunnerPool2Replicas)
		waitRepositoryRunnerPods(repoRunner2NS, repoRunnerPool2Name, repoRunnerPool2Replicas)

		By("confirming all org-runner1 pods are ready")
		waitDeployment(orgRunner1NS, orgRunnerPool1Name, orgRunnerPool1Replicas)
		waitOrganizationRunnerPods(orgRunner1NS, orgRunnerPool1Name, orgRunnerPool1Replicas)
	})

	It("should run the job-success on a runner pod and delete the pod immediately", func() {
		By("running 'job-success' workflow")
		waitRepositoryRunnerPods(repoRunner1NS, repoRunnerPool1Name, repoRunnerPool1Replicas)
		pushWorkflowFile("job-success.tmpl.yaml", repoRunner1NS, repoRunnerPool1Name)
		assignedPod, status := waitJobCompletion(repoRunner1NS, repoRunnerPool1Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
			"SlackChannel": Equal("#test2"),
		})))

		By("confirming the pod terminating")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.FinishedAt))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test2")

		By("waiting for the pod deleted")
		waitDeletion("pod", assignedPod.Namespace, assignedPod.Name)
	})

	It("should run the job-cancelled on a runner pod and delete the pod immediately", func() {
		By("running 'job-cancelled' workflow")
		waitRepositoryRunnerPods(repoRunner2NS, repoRunnerPool2Name, repoRunnerPool2Replicas)
		pushWorkflowFile("job-cancelled.tmpl.yaml", repoRunner2NS, repoRunnerPool2Name)
		assignedPod, status := waitJobCompletion(repoRunner2NS, repoRunnerPool2Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("cancelled"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
			"SlackChannel": BeEmpty(),
		})))

		By("confirming the pod terminating")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.FinishedAt))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test2")

		By("waiting for the pod deleted")
		waitDeletion("pod", assignedPod.Namespace, assignedPod.Name)
	})

	It("should run the job-failure on a runner pod and delete the pod after a while", func() {
		By("running 'job-failure' workflow")
		waitRepositoryRunnerPods(repoRunner1NS, repoRunnerPool1Name, repoRunnerPool1Replicas)
		pushWorkflowFile("job-failure.tmpl.yaml", repoRunner1NS, repoRunnerPool1Name)
		assignedPod, status := waitJobCompletion(repoRunner1NS, repoRunnerPool1Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("failure"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeTrue()),
			"JobInfo":      Not(BeNil()),
			"SlackChannel": BeEmpty(),
		})))

		By("checking pdb")
		_, stderr, err := kubectl("evict", "-n", repoRunner1NS, assignedPod.Name)
		Expect(err).Should(HaveOccurred())
		// The error message should be: "Error: Cannot evict pod as it would violate the pod's disruption budget."
		Expect(string(stderr)).Should(ContainSubstring("Error: Cannot evict pod as it would violate the pod's disruption budget."))

		By("confirming the pod terminating")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", (*status.FinishedAt).Add(30*time.Second)))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test1")

		By("waiting for the pod deleted")
		waitDeletion("pod", assignedPod.Namespace, assignedPod.Name)
	})

	It("should extend pod with the deletion time API", func() {
		By("running 'job-failure' workflow")
		waitRepositoryRunnerPods(repoRunner2NS, repoRunnerPool2Name, repoRunnerPool2Replicas)
		pushWorkflowFile("job-failure.tmpl.yaml", repoRunner2NS, repoRunnerPool2Name)
		assignedPod, status := waitJobCompletion(repoRunner2NS, repoRunnerPool2Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("failure"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeTrue()),
			"JobInfo":      Not(BeNil()),
			"SlackChannel": BeEmpty(),
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
			"SlackChannel": BeEmpty(),
		})))

		By("confirming the pod terminating")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletionTime: ", *status.DeletionTime)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.DeletionTime))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test2")

		By("waiting for the pod deleted")
		waitDeletion("pod", assignedPod.Namespace, assignedPod.Name)
	})

	It("should be successful the job that makes sure invisible environment variables", func() {
		By("running 'check-env' workflow")
		waitRepositoryRunnerPods(repoRunner1NS, repoRunnerPool1Name, repoRunnerPool1Replicas)
		pushWorkflowFile("check-env.tmpl.yaml", repoRunner1NS, repoRunnerPool1Name)
		assignedPod, status := waitJobCompletion(repoRunner1NS, repoRunnerPool1Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
			"SlackChannel": BeEmpty(),
		})))

		By("confirming the pod terminating")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.FinishedAt))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test1")

		By("waiting for the pod deleted")
		waitDeletion("pod", assignedPod.Namespace, assignedPod.Name)
	})

	It("should run a setup command", func() {
		By("running 'setup-command' workflow")
		waitRepositoryRunnerPods(repoRunner2NS, repoRunnerPool2Name, repoRunnerPool2Replicas)
		pushWorkflowFile("setup-command.tmpl.yaml", repoRunner2NS, repoRunnerPool2Name)
		assignedPod, status := waitJobCompletion(repoRunner2NS, repoRunnerPool2Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
			"SlackChannel": BeEmpty(),
		})))

		By("confirming the pod terminating")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.FinishedAt))

		By("confirming a slack message is successfully sent")
		slackMessageShouldBeSent(assignedPod, "#test2")

		By("waiting for the pod deleted")
		waitDeletion("pod", assignedPod.Namespace, assignedPod.Name)
	})

	It("should run the job-success on a organization level runner pod and delete the pod immediately", func() {
		By("running 'job-success' workflow")
		waitOrganizationRunnerPods(orgRunner1NS, orgRunnerPool1Name, orgRunnerPool1Replicas)
		pushWorkflowFile("job-success.tmpl.yaml", orgRunner1NS, orgRunnerPool1Name)
		assignedPod, status := waitJobCompletion(orgRunner1NS, orgRunnerPool1Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
			"SlackChannel": Equal("#test2"),
		})))

		By("confirming the pod terminating")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.FinishedAt))

		By("confirming a slack message is successfully sent to the channel specified by environment variable")
		slackMessageShouldBeSent(assignedPod, "#test2")

		By("waiting for the pod deleted")
		waitDeletion("pod", assignedPod.Namespace, assignedPod.Name)
	})

	It("should run the slack-channel-specified on a organization level runner pod and delete the pod immediately", func() {
		By("running 'slack-channel-specified' workflow")
		waitOrganizationRunnerPods(orgRunner1NS, orgRunnerPool1Name, orgRunnerPool1Replicas)
		pushWorkflowFile("slack-channel-specified.tmpl.yaml", orgRunner1NS, orgRunnerPool1Name)
		assignedPod, status := waitJobCompletion(orgRunner1NS, orgRunnerPool1Name)
		finishedAt := time.Now()

		By("checking status")
		Expect(status).To(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 3*time.Second)),
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      Not(BeNil()),
			"SlackChannel": Equal("#test1"),
		})))

		By("confirming the pod terminating")
		deletedAt := waitRunnerPodTerminating(assignedPod.Namespace, assignedPod.Name)
		fmt.Println("- FinishedAt  : ", *status.FinishedAt)
		fmt.Println("- DeletedAt   : ", deletedAt)
		Expect(deletedAt).To(BeTemporally(">", *status.FinishedAt))

		By("confirming a slack message is successfully sent to the channel specified in the /var/meows/slack_channel file updated in workflow.")
		slackMessageShouldBeSent(assignedPod, "#test1")

		By("waiting for the pod deleted")
		waitDeletion("pod", assignedPod.Namespace, assignedPod.Name)
	})

	It("should delete RunnerPool properly", func() {
		By("deleting runner1")
		stdout, stderr, err := kubectl("delete", "runnerpools", "-n", repoRunner1NS, repoRunnerPool1Name)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		By("confirming to delete resources")
		waitDeletion("deployment", repoRunner1NS, repoRunnerPool1Name)
		waitDeletion("secret", repoRunner1NS, "runner-token-"+repoRunnerPool1Name)

		By("confirming runners are deleted from GitHub Actions")
		Eventually(func() error {
			runnerNames, err := fetchAllRepositoryRunnerNames(repoRunner1NS + "/" + repoRunnerPool1Name)
			if err != nil {
				return err
			}
			if len(runnerNames) != 0 {
				return fmt.Errorf("%d runners still exist: runners %#v", len(runnerNames), runnerNames)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runner2 pods are existing")
		repoRunner2Pods, err := fetchRunnerPods(repoRunner2NS, repoRunnerPool2Name)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(repoRunner2Pods.Items).Should(HaveLen(repoRunnerPool2Replicas))
		runner2PodNames := getPodNames(repoRunner2Pods)

		By("counting the number of self-hosted runners fetched via GitHub Actions API")
		Eventually(func() error {
			runner2Names, err := fetchAllRepositoryRunnerNames(repoRunner2NS + "/" + repoRunnerPool2Name)
			if err != nil {
				return err
			}
			if len(runner2Names) != repoRunnerPool2Replicas || !cmp.Equal(runner2PodNames, runner2Names) {
				return fmt.Errorf("%d runners should exist: pods %#v runners %#v", repoRunnerPool2Replicas, runner2PodNames, runner2Names)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("deleting runner2")
		stdout, stderr, err = kubectl("delete", "runnerpools", "-n", repoRunner2NS, repoRunnerPool2Name)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		By("confirming to delete resources")
		waitDeletion("deployment", repoRunner2NS, repoRunnerPool2Name)
		waitDeletion("secret", repoRunner2NS, "runner-token-"+repoRunnerPool2Name)

		By("confirming runners are deleted from GitHub Actions")
		Eventually(func() error {
			runnerNames, err := fetchAllRepositoryRunnerNames(repoRunner2NS + "/" + repoRunnerPool2Name)
			if err != nil {
				return err
			}
			if len(runnerNames) != 0 {
				return fmt.Errorf("%d runners still exist: runners %#v", len(runnerNames), runnerNames)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("confirming runner3 pods are existing")
		orgRunner3Pods, err := fetchRunnerPods(orgRunner1NS, orgRunnerPool1Name)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(orgRunner3Pods.Items).Should(HaveLen(orgRunnerPool1Replicas))
		runner3PodNames := getPodNames(orgRunner3Pods)

		By("counting the number of self-hosted runners fetched via GitHub Actions API")
		Eventually(func() error {
			runner3Names, err := fetchAllOrganizationRunnerNames(orgRunner1NS + "/" + orgRunnerPool1Name)
			if err != nil {
				return err
			}
			if len(runner3Names) != orgRunnerPool1Replicas || !cmp.Equal(runner3PodNames, runner3Names) {
				return fmt.Errorf("%d runners should exist: pods %#v runners %#v", orgRunnerPool1Replicas, runner3PodNames, runner3Names)
			}
			return nil
		}).ShouldNot(HaveOccurred())

		By("deleting runner3")
		stdout, stderr, err = kubectl("delete", "runnerpools", "-n", orgRunner1NS, orgRunnerPool1Name)
		Expect(err).ShouldNot(HaveOccurred(), fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout, stderr, err))

		By("confirming to delete resources")
		waitDeletion("deployment", orgRunner1NS, orgRunnerPool1Name)
		waitDeletion("secret", orgRunner1NS, "runner-token-"+orgRunnerPool1Name)

		By("confirming runners are deleted from GitHub Actions")
		Eventually(func() error {
			runnerNames, err := fetchAllOrganizationRunnerNames(orgRunner1NS + "/" + orgRunnerPool1Name)
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
