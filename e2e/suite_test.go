package e2e

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/google/go-github/v33/github"
	"github.com/kelseyhightower/envconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	systemNS   = "actions-system"
	runnerNS   = "default"
	poolName   = "runnerpool-sample"
	numRunners = 3

	orgName  = "neco-test"
	repoName = "github-actions-controller-ci"
)

var (
	binDir string

	githubClient *github.Client

	runnerSelector = fmt.Sprintf(
		"%s=%s,%s=%s",
		constants.RunnerOrgLabelKey, orgName,
		constants.RunnerRepoLabelKey, repoName,
	)
)

func TestE2e(t *testing.T) {
	if os.Getenv("E2ETEST") == "" {
		t.Skip("Run under e2e/")

	}
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(5 * time.Minute)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	By("Getting the directory path which contains some binaries")
	binDir = os.Getenv("BIN_DIR")
	Expect(binDir).ShouldNot(BeEmpty())
	fmt.Println("This test uses the binaries under " + binDir)

	By("initializing github client")
	var e struct {
		AppID             int64  `split_words:"true"`
		AppInstallationID int64  `split_words:"true"`
		AppPrivateKeyPath string `split_words:"true"`
	}
	err := envconfig.Process("github", &e)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(e.AppID).ShouldNot(BeZero())
	Expect(e.AppInstallationID).ShouldNot(BeZero())
	Expect(e.AppPrivateKeyPath).ShouldNot(BeEmpty())

	rt, err := ghinstallation.NewKeyFromFile(
		http.DefaultTransport,
		e.AppID,
		e.AppInstallationID,
		e.AppPrivateKeyPath,
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	githubClient = github.NewClient(&http.Client{Transport: rt})
})

var _ = Describe("TopoLVM", func() {
	Context("E2E", testE2E)
})
