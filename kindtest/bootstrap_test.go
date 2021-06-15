package kindtest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testBootstrap() {
	It("remove namespaces", func() {
		// Delete namespaces if exists.
		_, _, err := kubectl("get", "ns", runnerNS)
		if err == nil {
			kubectlSafe("delete", "ns", runnerNS)
		}
		_, _, err = kubectl("get", "ns", controllerNS)
		if err == nil {
			kubectlSafe("delete", "ns", controllerNS)
		}
	})

	It("should deploy controller successfully", func() {
		By("creating namespace and secret for controller")
		createNamespace(controllerNS)
		kubectlSafe("create", "secret", "generic", "github-app-secret",
			"-n", controllerNS,
			"--from-literal=app-id="+githubAppID,
			"--from-literal=app-installation-id="+githubAppInstallationID,
			"--from-file=app-private-key="+githubAppPrivateKeyPath,
		)

		By("apply manifests")
		stdout, stderr, err := kustomizeBuild("./manifests/controller")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-f", "-")

		By("confirming all controller pods are ready")
		Eventually(func() error {
			return isDeploymentReady("controller-manager", controllerNS, 2)
		}).ShouldNot(HaveOccurred())
	})

	It("should deploy slack-agent successfully", func() {
		By("creating namespace and secret for slack-agent")
		createNamespace(runnerNS)
		kubectlSafe("label", "ns", runnerNS, "actions.cybozu.com/pod-mutate=true")
		kubectlSafe("label", "ns", runnerNS, "actions.cybozu.com/runnerpool-validate=true")
		kubectlSafe("create", "secret", "generic", "slack-app-secret",
			"-n", runnerNS,
			"--from-literal=SLACK_CHANNEL="+slackChannel,
			"--from-literal=SLACK_APP_TOKEN="+slackAppToken,
			"--from-literal=SLACK_BOT_TOKEN="+slackBotToken,
		)

		By("apply manifests")
		stdout, stderr, err := kustomizeBuild("./manifests/slack-agent")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-f", "-")

		By("confirming all slack-agent pods are ready")
		Eventually(func() error {
			return isDeploymentReady("slack-agent", runnerNS, 2)
		}).ShouldNot(HaveOccurred())
	})
}
