package kindtest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testBootstrap() {
	It("delete namespaces", func() {
		// Delete namespaces if exists.
		kubectlSafe("delete", "namespace", "-l", "runner-test=true")
		_, _, err := kubectl("get", "ns", controllerNS)
		if err == nil {
			kubectlSafe("delete", "ns", controllerNS)
		}
	})

	It("create namespaces", func() {
		createNamespace(controllerNS)
		kubectlSafe("label", "ns", controllerNS, "meows.cybozu.com/pod-mutate=ignore")
		createNamespace(runner1NS)
		kubectlSafe("label", "ns", runner1NS, "runner-test=true")
		createNamespace(runner2NS)
		kubectlSafe("label", "ns", runner2NS, "runner-test=true")
		createNamespace(runner3NS)
		kubectlSafe("label", "ns", runner3NS, "runner-test=true")
	})

	It("should deploy CRD", func() {
		By("applying manifests")
		stdout, stderr, err := kustomizeBuild("../config/crd")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-f", "-")
	})

	It("should deploy controller successfully", func() {
		By("creating secret for controller")
		kubectlSafe("create", "secret", "generic", "github-app-secret",
			"-n", controllerNS,
			"--from-literal=app-id="+githubAppID,
			"--from-literal=app-installation-id="+githubAppInstallationID,
			"--from-file=app-private-key="+githubAppPrivateKeyPath,
		)

		By("applying manifests")
		stdout, stderr, err := kustomizeBuild("./manifests/controller")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-f", "-")

		By("confirming all controller pods are ready")
		waitDeployment(controllerNS, "meows-controller", 2)
	})

	It("should deploy slack-agent successfully", func() {
		By("creating secret for slack-agent")
		kubectlSafe("create", "secret", "generic", "slack-app-secret",
			"-n", controllerNS,
			"--from-literal=SLACK_CHANNEL="+slackChannel,
			"--from-literal=SLACK_APP_TOKEN="+slackAppToken,
			"--from-literal=SLACK_BOT_TOKEN="+slackBotToken,
		)

		By("applying manifests")
		stdout, stderr, err := kustomizeBuild("./manifests/slack-agent")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s, err: %v", stdout, stderr, err)
		kubectlSafeWithInput(stdout, "apply", "-n", controllerNS, "-f", "-")

		By("confirming all slack-agent pods are ready")
		waitDeployment(controllerNS, "slack-agent", 2)
	})
}
