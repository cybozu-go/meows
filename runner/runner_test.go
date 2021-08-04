package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/meows/runner/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("Runner", func() {
	meowsDir := filepath.Join("..", "tmp", "meows")
	startedFlagFile := filepath.Join(meowsDir, "started")
	listenerMock := newlistenerMock()
	runnerClient := client.NewClient()
	var r *Runner
	var ctx context.Context
	var cancel context.CancelFunc

	BeforeEach(func() {
		err := os.MkdirAll(meowsDir, 0755)
		Expect(err).ToNot(HaveOccurred())
		if isFileExists(startedFlagFile) {
			err := os.Remove(startedFlagFile)
			Expect(err).ToNot(HaveOccurred())
		}
		os.Setenv(constants.PodNameEnvName, "fake-pod-name")
		os.Setenv(constants.PodNamespaceEnvName, "fake-pod-ns")
		os.Setenv(constants.RunnerTokenEnvName, "fake-runner-token")
		os.Setenv(constants.RunnerOrgEnvName, "fake-org")
		os.Setenv(constants.RunnerRepoEnvName, "fake-repo")
		os.Setenv(constants.RunnerPoolNameEnvName, "fake-runnerpool")
		os.Setenv(constants.RunnerOptionEnvName, "{}")
		r, err = NewRunner(fmt.Sprintf(":%d", constants.RunnerListenPort))
		Expect(err).ToNot(HaveOccurred())
		r.listener = listenerMock
		r.envs.startedFlagFile = startedFlagFile
		r.envs.workDir = filepath.Join(meowsDir, "_work")
		reg := prometheus.NewRegistry()
		metrics.InitRunnerPodMetrics(reg, r.envs.runnerPoolName)
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
		time.Sleep(5 * time.Second)
	})

	It("should not set deletion time", func() {
		By("run runner")
		go func() {
			defer GinkgoRecover()
			err := r.Run(ctx)
			Expect(err).ToNot(HaveOccurred())
		}()
		By("check deletion time")
		time.Sleep(5 * time.Second)
		Eventually(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tm, err := runnerClient.GetDeletionTime(ctx, "localhost")
			if err != nil {
				return err
			}
			if tm.IsZero() {
				return nil
			}
			return errors.New("Deletion time is not Zero. r.deletionTime: " + tm.Format(time.RFC3339))
		}).Should(Succeed())
	})

	It("should set deletion time when startedFlagFile exist", func() {
		_, err := os.Create(r.envs.startedFlagFile)
		Expect(err).ToNot(HaveOccurred())
		By("run runner")
		go func() {
			defer GinkgoRecover()
			err := r.Run(ctx)
			Expect(err).ToNot(HaveOccurred())
		}()
		By("check deletion time")
		time.Sleep(5 * time.Second)
		Eventually(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tm, err := runnerClient.GetDeletionTime(ctx, "localhost")
			if err != nil {
				return err
			}
			if !tm.IsZero() {
				return nil
			}
			return errors.New("Deletion time is Zero.")
		}).Should(Succeed())
	})
})

func TestRunner(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(10 * time.Second)
	SetDefaultEventuallyPollingInterval(time.Second)
	SetDefaultConsistentlyDuration(10 * time.Second)
	SetDefaultConsistentlyPollingInterval(time.Second)

	RunSpecs(t, "Runner Suite")
}

type listenerMock struct {
}

func (e *listenerMock) configure(ctx context.Context, configArgs []string) error {
	return nil
}

func (e *listenerMock) listen(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func newlistenerMock() listener {
	return &listenerMock{}
}
