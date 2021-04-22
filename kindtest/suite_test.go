package kindtest

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/google/go-github/v33/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	controllerNS = "actions-system"
	runnerNS     = "test-runner"
	poolName     = "runnerpool-sample"
	numRunners   = 3
	orgName      = "neco-test"
	repoName     = "github-actions-controller-ci"
)

var (
	binDir                  = os.Getenv("BIN_DIR")
	githubAppID             = os.Getenv("GITHUB_APP_ID")
	githubAppInstallationID = os.Getenv("GITHUB_APP_INSTALLATION_ID")
	githubAppPrivateKeyPath = os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")
	slackChannel            = os.Getenv("SLACK_CHANNEL")
	slackAppToken           = os.Getenv("SLACK_APP_TOKEN")
	slackBotToken           = os.Getenv("SLACK_BOT_TOKEN")
)

var (
	githubClient   *github.Client
	runnerSelector = fmt.Sprintf(
		"%s=%s,%s=%s",
		constants.RunnerOrgLabelKey, orgName,
		constants.RunnerRepoLabelKey, repoName,
	)
)

func TestOnKind(t *testing.T) {
	if os.Getenv("KINDTEST") == "" {
		t.Skip("Skip running kindtest/")
	}
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(5 * time.Minute)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	RunSpecs(t, "KindTest Suite")
}

var _ = BeforeSuite(func() {
	By("checking env variables")
	Expect(binDir).ShouldNot(BeEmpty())
	fmt.Println("This test uses the binaries under " + binDir)

	Expect(githubAppID).ShouldNot(BeEmpty())
	Expect(githubAppInstallationID).ShouldNot(BeEmpty())
	Expect(githubAppPrivateKeyPath).ShouldNot(BeEmpty())
	Expect(slackChannel).ShouldNot(BeEmpty())
	Expect(slackAppToken).ShouldNot(BeEmpty())
	Expect(slackBotToken).ShouldNot(BeEmpty())

	By("initializing github client")
	appID, err := strconv.ParseInt(githubAppID, 10, 64)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(appID).ShouldNot(BeZero())

	appInstallID, err := strconv.ParseInt(githubAppInstallationID, 10, 64)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(appInstallID).ShouldNot(BeZero())

	rt, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, appInstallID, githubAppPrivateKeyPath)
	Expect(err).ShouldNot(HaveOccurred())
	githubClient = github.NewClient(&http.Client{Transport: rt})
})

var _ = Describe("github-actions-controller", func() {
	Context("bootstrap", testBootstrap)
	Context("runner", testRunner)
})
