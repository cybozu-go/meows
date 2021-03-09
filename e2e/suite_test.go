package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var binDir string

func TestE2e(t *testing.T) {
	if os.Getenv("E2ETEST") == "" {
		t.Skip("Run under e2e/")

	}
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(5 * time.Minute)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	By("Getting the directory path which contains some binaries")
	binDir = os.Getenv("BINDIR")
	Expect(binDir).ShouldNot(BeEmpty())
	fmt.Println("This test uses the binaries under " + binDir)
})

var _ = Describe("TopoLVM", func() {
	Context("e2e", testE2E)
})
