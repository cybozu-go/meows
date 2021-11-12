package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/metrics"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	testRunnerDir = filepath.Join("..", "tmp", "runner")
	testWorkDir   = filepath.Join("..", "tmp", "runner", "_work")
	testVarDir    = filepath.Join("..", "tmp", "var", "meows")
)

var _ = Describe("Runner", func() {
	BeforeEach(func() {
		Expect(os.RemoveAll(testRunnerDir)).To(Succeed())
		Expect(os.RemoveAll(testWorkDir)).To(Succeed())
		Expect(os.RemoveAll(testVarDir)).To(Succeed())
		Expect(os.MkdirAll(testRunnerDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(testWorkDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(testVarDir, 0755)).To(Succeed())
		createFakeTokenFile()

		os.Setenv(constants.PodNameEnvName, "fake-pod-name")
		os.Setenv(constants.PodNamespaceEnvName, "fake-pod-ns")
		os.Setenv(constants.RunnerOrgEnvName, "fake-org")
		os.Setenv(constants.RunnerRepoEnvName, "fake-repo")
		os.Setenv(constants.RunnerPoolNameEnvName, "fake-runnerpool")
		os.Setenv(constants.RunnerOptionEnvName, "{}")
	})

	AfterEach(func() {
		time.Sleep(time.Second)
	})

	It("should change states", func() {
		By("starting runner")
		listener := newListenerMock()
		cancel := startRunner(listener)
		defer cancel()

		By("checking initializing state")
		flagFileShouldExist("started")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("initializing"),
			"Result":       BeEmpty(),
			"FinishedAt":   BeNil(),
			"DeletionTime": BeNil(),
			"Extend":       BeNil(),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
		metricsShouldHaveValue("meows_runner_pod_state",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("debugging")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("initializing")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("running")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("stale")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)
		metricsShouldNotExist("meows_runner_listener_exit_state")

		By("checking running state")
		createJobInfoFile()
		listener.configureCh <- nil
		time.Sleep(time.Second)

		flagFileShouldExist("started")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("running"),
			"Result":       BeEmpty(),
			"FinishedAt":   BeNil(),
			"DeletionTime": BeNil(),
			"Extend":       BeNil(),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
		metricsShouldHaveValue("meows_runner_pod_state",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("debugging")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("initializing")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("running")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("stale")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)
		metricsShouldNotExist("meows_runner_listener_exit_state")

		By("checking debugging state")
		listener.listenCh <- nil
		finishedAt := time.Now()
		time.Sleep(time.Second)

		flagFileShouldExist("started")
		flagFileShouldNotExist("extend")
		flagFileShouldNotExist("failure")
		flagFileShouldNotExist("cancelled")
		flagFileShouldNotExist("success")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("unknown"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"Extend":       PointTo(BeFalse()),
			"JobInfo": PointTo(MatchFields(IgnoreExtras, Fields{
				"Actor":      Equal("actor"),
				"Repository": Equal("meows"),
				"GitRef":     Equal("branch"),
			})),
			"SlackChannel": BeEmpty(),
		})))
		metricsShouldHaveValue("meows_runner_pod_state",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("debugging")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("initializing")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("running")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("stale")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)
		metricsShouldNotExist("meows_runner_listener_exit_state")
	})

	It("should extend default duration when extend file exists", func() {
		By("starting runner")
		listener := newListenerMock("extend")
		cancel := startRunner(listener)
		defer cancel()
		listener.configureCh <- nil
		listener.listenCh <- nil
		finishedAt := time.Now()
		time.Sleep(time.Second)

		By("checking outputs")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("unknown"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt.Add(20*time.Minute), 500*time.Millisecond)),
			"Extend":       PointTo(BeTrue()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
		metricsShouldHaveValue("meows_runner_pod_state",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("debugging")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("initializing")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("running")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("stale")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)
		metricsShouldNotExist("meows_runner_listener_exit_state")
	})

	It("should extend specified duration when EXTEND_DURATION is specified", func() {
		By("starting runner with extend duration")
		os.Setenv(constants.ExtendDurationEnvName, "1h")
		listener := newListenerMock("extend")
		cancel := startRunner(listener)
		defer cancel()
		listener.configureCh <- nil
		listener.listenCh <- nil
		finishedAt := time.Now()
		time.Sleep(time.Second)

		By("checking outputs")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("unknown"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt.Add(time.Hour), 500*time.Millisecond)),
			"Extend":       PointTo(BeTrue()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
		metricsShouldHaveValue("meows_runner_pod_state",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("debugging")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("initializing")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("running")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("stale")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)
		metricsShouldNotExist("meows_runner_listener_exit_state")

		os.Unsetenv(constants.ExtendDurationEnvName)
	})

	It("should extend via deletion_time API", func() {
		By("starting runner")
		listener := newListenerMock("extend", "failure")
		cancel := startRunner(listener)
		defer cancel()
		listener.configureCh <- nil
		listener.listenCh <- nil
		finishedAt := time.Now()
		time.Sleep(time.Second)

		By("checking outputs")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("failure"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt.Add(20*time.Minute), 500*time.Millisecond)),
			"Extend":       PointTo(BeTrue()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
		metricsShouldHaveValue("meows_runner_pod_state",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("debugging")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("initializing")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("running")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("stale")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)
		metricsShouldNotExist("meows_runner_listener_exit_state")

		By("requesting API")
		extendTo := time.Now().Add(2 * time.Hour)
		NewClient().PutDeletionTime(context.Background(), "localhost", extendTo)

		By("checking outputs")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("failure"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"DeletionTime": PointTo(BeTemporally("~", extendTo, 500*time.Millisecond)),
			"Extend":       PointTo(BeTrue()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
		metricsShouldHaveValue("meows_runner_pod_state",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("debugging")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("initializing")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("running")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("stale")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)
		metricsShouldNotExist("meows_runner_listener_exit_state")
	})

	It("should become stale state when started file exists", func() {
		By("starting runner with started file")
		createFlagFile("started")
		listener := newListenerMock()
		cancel := startRunner(listener)
		defer cancel()

		By("checking outputs")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("stale"),
			"Result":       BeEmpty(),
			"FinishedAt":   BeNil(),
			"DeletionTime": BeNil(),
			"Extend":       BeNil(),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
		metricsShouldHaveValue("meows_runner_pod_state",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("debugging")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("initializing")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("running")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("stale")}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)
		metricsShouldNotExist("meows_runner_listener_exit_state")
	})

	It("should run setup command", func() {
		By("starting runner with setup command")
		opt, err := json.Marshal(&Option{
			SetupCommand: []string{"bash", "-c", "touch ./dummy"},
		})
		Expect(err).NotTo(HaveOccurred())
		os.Setenv(constants.RunnerOptionEnvName, string(opt))

		listener := newListenerMock()
		cancel := startRunner(listener)
		defer cancel()

		By("checking outputs")
		_, err = os.Stat(filepath.Join(testRunnerDir, "dummy")) // setup command is run at runner root dir.
		Expect(err).ToNot(HaveOccurred())

		flagFileShouldExist("started")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("initializing"),
			"Result":       BeEmpty(),
			"FinishedAt":   BeNil(),
			"DeletionTime": BeNil(),
			"Extend":       BeNil(),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
		metricsShouldHaveValue("meows_runner_pod_state",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("debugging")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"1": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("initializing")}),
					"Value": BeNumerically("==", 1.0),
				})),
				"2": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("running")}),
					"Value": BeNumerically("==", 0.0),
				})),
				"3": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{"runnerpool": Equal("fake-pod-ns/fake-runnerpool"), "state": Equal("stale")}),
					"Value": BeNumerically("==", 0.0),
				})),
			}),
		)
		metricsShouldNotExist("meows_runner_listener_exit_state")
	})

	It("should become success status when success file is created", func() {
		By("starting runner with creating success file")
		listener := newListenerMock("success")
		cancel := startRunner(listener)
		defer cancel()
		listener.configureCh <- nil
		listener.listenCh <- nil
		finishedAt := time.Now()
		time.Sleep(time.Second)

		By("checking outputs")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
	})

	It("should become failure status when failure file is created", func() {
		By("starting runner with creating failure file")
		listener := newListenerMock("failure")
		cancel := startRunner(listener)
		defer cancel()
		listener.configureCh <- nil
		listener.listenCh <- nil
		finishedAt := time.Now()
		time.Sleep(time.Second)

		By("checking outputs")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("failure"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
	})

	It("should become cancelled status when cancelled file is created", func() {
		By("starting runner with creating cancelled file")
		listener := newListenerMock("cancelled")
		cancel := startRunner(listener)
		defer cancel()
		listener.configureCh <- nil
		listener.listenCh <- nil
		finishedAt := time.Now()
		time.Sleep(time.Second)

		By("checking outputs")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("cancelled"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
	})

	It("should be update the status SlackChannel when slack_channel file is created", func() {
		By("starting runner with creating slack_channel file")
		listener := newListenerMock("success")
		cancel := startRunner(listener)
		defer cancel()

		Expect(os.MkdirAll(filepath.Join(testVarDir), 0755)).To(Succeed())
		slackChannelFile := filepath.Join(testVarDir, "slack_channel")
		err := os.WriteFile(slackChannelFile, []byte("#test1\n"), 0664)
		Expect(err).ToNot(HaveOccurred())

		listener.configureCh <- nil
		listener.listenCh <- nil
		finishedAt := time.Now()
		time.Sleep(time.Second)

		By("checking outputs")
		statusShouldHaveValue(PointTo(MatchAllFields(Fields{
			"State":        Equal("debugging"),
			"Result":       Equal("success"),
			"FinishedAt":   PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"DeletionTime": PointTo(BeTemporally("~", finishedAt, 500*time.Millisecond)),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      BeNil(),
			"SlackChannel": Equal("#test1"),
		})))

		By("remove slack_channel file")
		err = os.Remove(slackChannelFile)
		Expect(err).ToNot(HaveOccurred())
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
	flagFiles   []string
	configureCh chan error
	listenCh    chan error
}

func newListenerMock(flagFiles ...string) *listenerMock {
	return &listenerMock{
		flagFiles:   flagFiles,
		configureCh: make(chan error),
		listenCh:    make(chan error),
	}
}

func (l *listenerMock) configure(ctx context.Context, configArgs []string) error {
	return <-l.configureCh
}

func (l *listenerMock) listen(ctx context.Context) error {
	ret := <-l.listenCh
	for _, file := range l.flagFiles {
		createFlagFile(file)
	}
	return ret
}

func startRunner(listener Listener) context.CancelFunc {
	r, err := NewRunner(listener, fmt.Sprintf(":%d", constants.RunnerListenPort), testRunnerDir, testWorkDir, testVarDir)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ctx, cancel := context.WithCancel(context.Background())
	logger := zap.New()
	ctx = log.IntoContext(ctx, logger)
	go func() {
		defer GinkgoRecover()
		Expect(r.Run(ctx)).To(Succeed())
	}()
	time.Sleep(2 * time.Second) // delay
	return cancel
}

func createFakeTokenFile() {
	ExpectWithOffset(1, os.MkdirAll(filepath.Join(testVarDir, constants.SecretsDirName), 0755)).To(Succeed())
	err := os.WriteFile(filepath.Join(testVarDir, constants.SecretsDirName, constants.RunnerTokenFileName), []byte("faketoken"), 0664)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func createJobInfoFile() {
	jobInfo := &JobInfo{
		Actor:      "actor",
		Repository: "meows",
		GitRef:     "branch",
	}
	data, err := json.Marshal(jobInfo)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(testVarDir, "github.env"), data, 0664)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func createFlagFile(filename string) {
	_, err := os.Create(filepath.Join(testVarDir, filename))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func flagFileShouldExist(filename string) {
	_, err := os.Stat(filepath.Join(testVarDir, filename))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func flagFileShouldNotExist(filename string) {
	_, err := os.Stat(filepath.Join(testVarDir, filename))
	ExpectWithOffset(1, err).To(HaveOccurred())
}

func statusShouldHaveValue(matcher gomegatypes.GomegaMatcher) {
	runnerClient := NewClient()
	st, err := runnerClient.GetStatus(context.Background(), "localhost")
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	ExpectWithOffset(1, st).To(matcher)
}

func metricsShouldNotExist(name string) {
	_, err := metrics.FetchGauge(context.Background(), "http://localhost:8080/metrics", name)
	ExpectWithOffset(1, err).Should(MatchError(metrics.ErrNotExist))
}

func metricsShouldHaveValue(name string, matcher gomegatypes.GomegaMatcher) {
	m, err := metrics.FetchGauge(context.Background(), "http://localhost:8080/metrics", name)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	ExpectWithOffset(1, m).To(matcher)
}
