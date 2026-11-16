package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/metrics"
	. "github.com/onsi/ginkgo/v2"
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
	AfterEach(func() {
		time.Sleep(time.Second)
	})

	It("should change states", func() {
		By("starting runner")
		resetEnv(false)
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
		metricsShouldNotExist("meows_runner_job_info")

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
		metricsShouldHaveValue("meows_runner_job_info",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{
						"runnerpool":   Equal("fake-pod-ns/fake-runnerpool"),
						"repository":   Equal("meows"),
						"workflow":     Equal("workflow"),
						"job_id":       Equal("job"),
						"run_id":       Equal("123"),
						"pull_request": Equal("45"),
					}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)

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
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo": PointTo(MatchFields(IgnoreExtras, Fields{
				"Actor":      Equal("actor"),
				"Repository": Equal("meows"),
				"GitRef":     Equal("branch"),
				"StartedAt":  PointTo(BeTemporally("~", startedAt)),
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
		metricsShouldHaveValue("meows_runner_job_info",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{
						"runnerpool":   Equal("fake-pod-ns/fake-runnerpool"),
						"repository":   Equal("meows"),
						"workflow":     Equal("workflow"),
						"job_id":       Equal("job"),
						"run_id":       Equal("123"),
						"pull_request": Equal("45"),
					}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)
	})

	It("should expose sanitized job info metrics", func() {
		By("starting runner")
		resetEnv(false)
		listener := newListenerMock()
		cancel := startRunner(listener)
		defer cancel()
		listener.configureCh <- nil
		time.Sleep(time.Second)

		By("writing a job info without run ID")
		writeJobInfoFile(`{"repository":"owner/repo","workflow_name":"workflow","job_id":"job"}`)
		metricsShouldNotExist("meows_runner_job_info")

		By("writing a job info with an invalid run ID")
		writeJobInfoFile(`{"repository":"owner/repo","run_id":"123"}`)
		metricsShouldNotExist("meows_runner_job_info")

		By("writing a job info with a long workflow name and no pull request")
		writeJobInfoFile(fmt.Sprintf(`{"repository":"owner/repo","workflow_name":"%s","job_id":"job","run_id":1}`, strings.Repeat("あ", 200)))
		metricsShouldHaveValue("meows_runner_job_info",
			MatchAllElementsWithIndex(IndexIdentity, Elements{
				"0": PointTo(MatchAllFields(Fields{
					"Label": MatchAllKeys(Keys{
						"runnerpool":   Equal("fake-pod-ns/fake-runnerpool"),
						"repository":   Equal("owner/repo"),
						"workflow":     Equal(strings.Repeat("あ", 128)),
						"job_id":       Equal("job"),
						"run_id":       Equal("1"),
						"pull_request": BeEmpty(),
					}),
					"Value": BeNumerically("==", 1.0),
				})),
			}),
		)
	})

	It("should extend default duration when extend file exists", func() {
		By("starting runner")
		resetEnv(true)
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
			"DeletionTime": BeNil(),
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

	It("should extend via deletion_time API", func() {
		By("starting runner")
		resetEnv(false)
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
			"DeletionTime": BeNil(),
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
		resetEnv(false)
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
		resetEnv(false)
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
		resetEnv(false)
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
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
	})

	It("should become failure status when failure file is created", func() {
		By("starting runner with creating failure file")
		resetEnv(false)
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
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
	})

	It("should become cancelled status when cancelled file is created", func() {
		By("starting runner with creating cancelled file")
		resetEnv(false)
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
			"DeletionTime": BeNil(),
			"Extend":       PointTo(BeFalse()),
			"JobInfo":      BeNil(),
			"SlackChannel": BeEmpty(),
		})))
	})

	It("should be update the status SlackChannel when slack_channel file is created", func() {
		By("starting runner with creating slack_channel file")
		resetEnv(false)
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
			"DeletionTime": BeNil(),
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
	fmt.Println(configArgs)
	return <-l.configureCh
}

func (l *listenerMock) listen(ctx context.Context) error {
	ret := <-l.listenCh
	for _, file := range l.flagFiles {
		createFlagFile(file)
	}
	return ret
}

func resetEnv(orgRunner bool) {
	ExpectWithOffset(1, os.RemoveAll(testRunnerDir)).To(Succeed())
	ExpectWithOffset(1, os.RemoveAll(testWorkDir)).To(Succeed())
	ExpectWithOffset(1, os.RemoveAll(testVarDir)).To(Succeed())
	ExpectWithOffset(1, os.MkdirAll(testRunnerDir, 0755)).To(Succeed())
	ExpectWithOffset(1, os.MkdirAll(testWorkDir, 0755)).To(Succeed())
	ExpectWithOffset(1, os.MkdirAll(testVarDir, 0755)).To(Succeed())
	createFakeTokenFile()

	os.Setenv(constants.PodNameEnvName, "fake-pod-name")
	os.Setenv(constants.PodNamespaceEnvName, "fake-pod-ns")
	os.Setenv(constants.RunnerPoolNameEnvName, "fake-runnerpool")
	os.Setenv(constants.RunnerOptionEnvName, "{}")
	if orgRunner {
		os.Setenv(constants.RunnerOrgEnvName, "fake-org")
		os.Unsetenv(constants.RunnerRepoEnvName)
	} else {
		os.Setenv(constants.RunnerRepoEnvName, "fake-org/fake-repo")
		os.Unsetenv(constants.RunnerOrgEnvName)
	}
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

var startedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func createJobInfoFile() {
	jobInfo := &JobInfo{
		Actor:          "actor",
		Repository:     "meows",
		GitRef:         "branch",
		WorkflowName:   "workflow",
		JobID:          "job",
		RunID:          123,
		PullRequestNum: 45,
		StartedAt:      &startedAt,
	}
	data, err := json.Marshal(jobInfo)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(testVarDir, "github.env"), data, 0664)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

func writeJobInfoFile(data string) {
	err := os.WriteFile(filepath.Join(testVarDir, "github.env"), []byte(data), 0664)
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
