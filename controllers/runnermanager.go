package controllers

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	constants "github.com/cybozu-go/meows"
	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/meows/metrics"
	rc "github.com/cybozu-go/meows/runner/client"
	"github.com/cybozu-go/well"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete

type RunnerManager interface {
	StartOrUpdate(*meowsv1alpha1.RunnerPool)
	Stop(string)
}

func namespacedName(namespace, name string) string {
	return namespace + "/" + name
}

type RunnerManagerImpl struct {
	log             logr.Logger
	interval        time.Duration
	k8sClient       client.Client
	githubClient    github.Client
	runnerPodClient rc.Client

	loops map[string]*managerLoop
}

func NewRunnerManager(log logr.Logger, interval time.Duration, k8sClient client.Client, githubClient github.Client, runnerPodClient rc.Client) RunnerManager {
	return &RunnerManagerImpl{
		log:             log,
		interval:        interval,
		k8sClient:       k8sClient,
		githubClient:    githubClient,
		runnerPodClient: runnerPodClient,
		loops:           map[string]*managerLoop{},
	}
}

func (m *RunnerManagerImpl) StartOrUpdate(rp *meowsv1alpha1.RunnerPool) {
	rpNamespacedName := namespacedName(rp.Namespace, rp.Name)
	if _, ok := m.loops[rpNamespacedName]; !ok {
		loop := &managerLoop{
			log:             m.log.WithValues("runnerpool", rpNamespacedName),
			interval:        m.interval,
			k8sClient:       m.k8sClient,
			githubClient:    m.githubClient,
			runnerPodClient: m.runnerPodClient,
			rpNamespace:     rp.Namespace,
			rpName:          rp.Name,
			repository:      rp.Spec.RepositoryName,
			replicas:        rp.Spec.Replicas,
		}
		loop.start()
		m.loops[rpNamespacedName] = loop
	} else {
		m.loops[rpNamespacedName].update(rp)
	}
}

func (m *RunnerManagerImpl) Stop(rpNamespacedName string) {
	if loop, ok := m.loops[rpNamespacedName]; ok {
		loop.stop()
		delete(m.loops, rpNamespacedName)
	}
}

type managerLoop struct {
	// Given from outside. Not update internally.
	log             logr.Logger
	interval        time.Duration
	k8sClient       client.Client
	githubClient    github.Client
	runnerPodClient rc.Client
	rpNamespace     string
	rpName          string
	repository      string
	replicas        int32 // This field will be accessed from some goroutines. So use atomic package to access.

	// Update internally.
	env             *well.Environment
	cancel          context.CancelFunc
	prevRunnerNames []string
}

func (m *managerLoop) rpNamespacedName() string {
	return m.rpNamespace + "/" + m.rpName
}

// Start starts loop to manage Actions runner
func (m *managerLoop) start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.env = well.NewEnvironment(ctx)

	m.env.Go(func(ctx context.Context) error {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		m.log.Info("start a watching loop")
		for {
			select {
			case <-ctx.Done():
				m.log.Info("stop a watching loop")
				return nil
			case <-ticker.C:
				err := m.runOnce(ctx)
				if err != nil {
					m.log.Error(err, "failed to run a watching loop")
				}
			}
		}
	})
	m.env.Stop()
}

func (m *managerLoop) stop() error {
	if m.cancel == nil {
		return errors.New("runner manager is not started")
	}

	m.cancel()
	if err := m.env.Wait(); err != nil {
		return err
	}

	for _, runner := range m.prevRunnerNames {
		metrics.DeleteRunnerMetrics(m.rpNamespacedName(), runner)
	}
	metrics.DeleteRunnerPoolMetrics(m.rpNamespacedName())
	return nil
}

func (m *managerLoop) update(rp *meowsv1alpha1.RunnerPool) {
	atomic.StoreInt32(&m.replicas, rp.Spec.Replicas)
}

func (m *managerLoop) runOnce(ctx context.Context) error {
	podList, err := m.fetchRunnerPods(ctx)
	if err != nil {
		return err
	}
	runnerList, err := m.fetchRunners(ctx)
	if err != nil {
		return err
	}

	m.updateMetrics(ctx, podList, runnerList)

	err = m.deleteRunners(ctx, podList, runnerList)
	if err != nil {
		return err
	}
	err = m.deleteRunnerPods(ctx, podList)
	if err != nil {
		return err
	}
	return nil
}

func (m *managerLoop) fetchRunnerPods(ctx context.Context) (*corev1.PodList, error) {
	selector, err := metav1.LabelSelectorAsSelector(
		&metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.AppNameLabelKey:      constants.AppName,
				constants.AppComponentLabelKey: constants.AppComponentRunner,
				constants.AppInstanceLabelKey:  m.rpName,
			},
		},
	)
	if err != nil {
		m.log.Error(err, "failed to make label selector")
		return nil, err
	}

	podList := new(corev1.PodList)
	err = m.k8sClient.List(ctx, podList, client.InNamespace(m.rpNamespace), client.MatchingLabelsSelector{
		Selector: selector,
	})
	if err != nil {
		m.log.Error(err, "failed to list pods")
		return nil, err
	}
	return podList, nil
}

func (m *managerLoop) fetchRunners(ctx context.Context) ([]*github.Runner, error) {
	runners, err := m.githubClient.ListRunners(ctx, m.repository)
	if err != nil {
		m.log.Error(err, "failed to list runners")
		return nil, err
	}

	ret := []*github.Runner{}
	for _, runner := range runners {
		if runner.ID == 0 || runner.Name == "" {
			err := fmt.Errorf("runner should have ID and name %#v", runner)
			m.log.Error(err, "got invalid runner")
			continue
		}

		var ownRunner bool
		for _, label := range runner.Labels {
			if label == m.rpNamespacedName() {
				ownRunner = true
				break
			}
		}
		if !ownRunner {
			continue
		}

		ret = append(ret, runner)
	}
	return ret, nil
}

func (m *managerLoop) updateMetrics(ctx context.Context, podList *corev1.PodList, runnerList []*github.Runner) {
	metrics.UpdateRunnerPoolMetrics(m.rpNamespacedName(), int(atomic.LoadInt32(&m.replicas)))

	var currentRunnerNames []string
	for _, runner := range runnerList {
		metrics.UpdateRunnerMetrics(m.rpNamespacedName(), runner.Name, runner.Online, runner.Busy)
		currentRunnerNames = append(currentRunnerNames, runner.Name)
	}

	// Sometimes, offline runners will be deleted from github automatically.
	// Therefore, compare the past runners with the current runners and remove the metrics for the deleted runners.
	for _, removedRunnerName := range difference(m.prevRunnerNames, currentRunnerNames) {
		metrics.DeleteRunnerMetrics(m.rpNamespacedName(), removedRunnerName)
	}
	m.prevRunnerNames = currentRunnerNames
}

func difference(prev, current []string) []string {
	set := map[string]bool{}
	for _, val := range current {
		set[val] = true
	}

	var ret []string
	for _, val := range prev {
		if !set[val] {
			ret = append(ret, val)
		}
	}
	return ret
}

func (m *managerLoop) deleteRunners(ctx context.Context, podList *corev1.PodList, runnerList []*github.Runner) error {
	for _, runner := range runnerList {
		if runner.Online || podExists(runner.Name, podList) {
			continue
		}

		err := m.githubClient.RemoveRunner(ctx, m.repository, runner.ID)
		if err != nil {
			m.log.Error(err, "failed to remove runner", "runner", runner.Name, "runner_id", runner.ID)
			return err
		}
		m.log.Info("removed runner", "runner", runner.Name, "runner_id", runner.ID)
	}
	return nil
}

func podExists(name string, podList *corev1.PodList) bool {
	for i := range podList.Items {
		if podList.Items[i].Name == name {
			return true
		}
	}
	return false
}

func (m *managerLoop) deleteRunnerPods(ctx context.Context, podList *corev1.PodList) error {
	now := time.Now().UTC()
	for i := range podList.Items {
		po := &podList.Items[i]
		t, err := m.runnerPodClient.GetDeletionTime(ctx, po.Status.PodIP)
		if err != nil {
			m.log.Error(err, "skipped deleting pod because failed to get the deletion time from the runner pod API", "pod", namespacedName(po.Namespace, po.Name))
			continue
		}
		if t.IsZero() || t.After(now) {
			continue
		}

		err = m.k8sClient.Delete(ctx, po)
		if err != nil {
			m.log.Error(err, "failed to delete pod", "pod", namespacedName(po.Namespace, po.Name))
			return err
		}
		m.log.Info("removed pod", "pod", namespacedName(po.Namespace, po.Name))
	}
	return nil
}
