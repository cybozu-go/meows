package controllers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	constants "github.com/cybozu-go/meows"
	"github.com/cybozu-go/meows/agent"
	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/meows/runner"
	"github.com/cybozu-go/well"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete;update

// RunnerManager manages runner pods and runners registered in GitHub.
// It generates one goroutine for each RunnerPool CR to manage them.
type RunnerManager interface {
	StartOrUpdate(*meowsv1alpha1.RunnerPool, *github.ClientCredential) error
	Stop(*meowsv1alpha1.RunnerPool) error
	StopAll()
}

type runnerManager struct {
	log                 logr.Logger
	k8sClient           client.Client
	scheme              *runtime.Scheme
	githubClientFactory github.ClientFactory
	runnerPodClient     runner.Client
	interval            time.Duration
	mu                  sync.Mutex
	stopped             bool
	processes           map[string]*manageProcess
}

func NewRunnerManager(log logr.Logger, k8sClient client.Client, scheme *runtime.Scheme, githubClientFactory github.ClientFactory, runnerPodClient runner.Client, interval time.Duration) RunnerManager {
	return &runnerManager{
		log:                 log.WithName("RunnerManager"),
		k8sClient:           k8sClient,
		scheme:              scheme,
		githubClientFactory: githubClientFactory,
		runnerPodClient:     runnerPodClient,
		interval:            interval,
		processes:           map[string]*manageProcess{},
	}
}

func (m *runnerManager) StartOrUpdate(rp *meowsv1alpha1.RunnerPool, cred *github.ClientCredential) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return errors.New("RunnerManager is already stopped")
	}

	rpNamespacedName := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	if _, ok := m.processes[rpNamespacedName]; !ok {
		githubClient, err := m.githubClientFactory.New(cred)
		if err != nil {
			return fmt.Errorf("failed to create a github client; %w", err)
		}
		process, err := newManageProcess(
			m.log.WithValues("runnerpool", rpNamespacedName),
			m.k8sClient,
			m.scheme,
			githubClient,
			m.runnerPodClient,
			m.interval,
			rp,
		)
		if err != nil {
			return err
		}
		process.start()
		m.processes[rpNamespacedName] = process
		return nil
	}
	return m.processes[rpNamespacedName].update(rp)
}

func (m *runnerManager) Stop(rp *meowsv1alpha1.RunnerPool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rpNamespacedName := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	if process, ok := m.processes[rpNamespacedName]; ok {
		if err := process.stop(); err != nil {
			return err
		}
		delete(m.processes, rpNamespacedName)
	}
	return nil
}

func (m *runnerManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, process := range m.processes {
		process.stop()
	}
	m.processes = nil
	m.stopped = true
}

type manageProcess struct {
	// Given from outside. Not update internally.
	log                   logr.Logger
	k8sClient             client.Client
	scheme                *runtime.Scheme
	githubClient          github.Client
	runnerPodClient       runner.Client
	slackAgentClient      *agent.Client
	interval              time.Duration
	rpNamespace           string
	rpName                string
	owner                 string
	repo                  string
	replicas              int32 // This field will be accessed from multiple goroutines. So use mutex to access.
	maxRunnerPods         int32 // This field will be accessed from multiple goroutines. So use mutex to access.
	needSlackNotification bool
	slackChannel          string
	slackAgentServiceName string
	extendDuration        time.Duration
	recreateDeadline      time.Duration
	denyDisruption        bool

	// Update internally.
	lastCheckTime   time.Time
	env             *well.Environment
	cancel          context.CancelFunc
	prevRunnerNames []string
	mu              sync.Mutex
	deleteMetrics   func()
}

func newManageProcess(log logr.Logger, k8sClient client.Client, scheme *runtime.Scheme, githubClient github.Client, runnerPodClient runner.Client, interval time.Duration, rp *meowsv1alpha1.RunnerPool) (*manageProcess, error) {
	extendDuration, _ := time.ParseDuration(rp.Spec.Notification.ExtendDuration)
	recreateDeadline, _ := time.ParseDuration(rp.Spec.RecreateDeadline)

	agentName := constants.DefaultSlackAgentServiceName
	if rp.Spec.Notification.Slack.AgentServiceName != "" {
		agentName = rp.Spec.Notification.Slack.AgentServiceName
	}
	agentClient, err := agent.NewClient(agentName)
	if err != nil {
		return nil, err
	}

	rpNamespacedName := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	process := &manageProcess{
		log:                   log,
		k8sClient:             k8sClient,
		scheme:                scheme,
		githubClient:          githubClient,
		runnerPodClient:       runnerPodClient,
		interval:              interval,
		rpNamespace:           rp.Namespace,
		rpName:                rp.Name,
		owner:                 rp.GetOwner(),
		repo:                  rp.GetRepository(),
		replicas:              rp.Spec.Replicas,
		maxRunnerPods:         rp.Spec.MaxRunnerPods,
		slackAgentClient:      agentClient,
		needSlackNotification: rp.Spec.Notification.Slack.Enable,
		slackChannel:          rp.Spec.Notification.Slack.Channel,
		slackAgentServiceName: agentName,
		extendDuration:        extendDuration,
		recreateDeadline:      recreateDeadline,
		denyDisruption:        rp.Spec.DenyDisruption,
		lastCheckTime:         time.Now().UTC(),
		deleteMetrics: func() {
			metrics.DeleteAllRunnerMetrics(rpNamespacedName)
			metrics.DeleteRunnerPoolMetrics(rpNamespacedName)
		},
	}
	return process, nil
}

func (p *manageProcess) update(rp *meowsv1alpha1.RunnerPool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.replicas = rp.Spec.Replicas
	p.maxRunnerPods = rp.Spec.MaxRunnerPods
	p.needSlackNotification = rp.Spec.Notification.Slack.Enable
	p.slackChannel = rp.Spec.Notification.Slack.Channel

	extendDuration, _ := time.ParseDuration(rp.Spec.Notification.ExtendDuration)
	p.extendDuration = extendDuration
	recreateDeadline, _ := time.ParseDuration(rp.Spec.RecreateDeadline)
	p.recreateDeadline = recreateDeadline
	p.denyDisruption = rp.Spec.DenyDisruption

	agentName := constants.DefaultSlackAgentServiceName
	if rp.Spec.Notification.Slack.AgentServiceName != "" {
		agentName = rp.Spec.Notification.Slack.AgentServiceName
	}
	if p.slackAgentServiceName != agentName {
		err := p.slackAgentClient.UpdateServerURL(agentName)
		if err != nil {
			return err
		}
		p.slackAgentServiceName = agentName
	}

	return nil

}

func (p *manageProcess) rpNamespacedName() string {
	return p.rpNamespace + "/" + p.rpName
}

func (p *manageProcess) start() {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.env = well.NewEnvironment(ctx)
	p.env.Go(func(ctx context.Context) error {
		p.run(ctx)
		p.deleteMetrics()

		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p.deleteAllRunners(ctx2)
		return nil
	})
	p.env.Stop()
}

func (p *manageProcess) stop() error {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
		if err := p.env.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func (p *manageProcess) run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.log.Info("start a runner manager process")
	for {
		select {
		case <-ctx.Done():
			p.log.Info("stop a runner manager process")
			return
		case <-ticker.C:
			err := p.runOnce(ctx)
			if err != nil {
				p.log.Error(err, "failed to run a runner manager process")
			}
		}
	}
}

func (p *manageProcess) runOnce(ctx context.Context) error {
	podList, err := p.fetchRunnerPods(ctx)
	if err != nil {
		return err
	}
	runnerList, err := p.fetchRunners(ctx)
	if err != nil {
		return err
	}
	p.updateMetrics(podList, runnerList)

	err = p.maintainRunnerPods(ctx, runnerList, podList)
	if err != nil {
		return err
	}
	err = p.deleteOfflineRunners(ctx, runnerList, podList)
	if err != nil {
		return err
	}

	return nil
}

func (p *manageProcess) fetchRunnerPods(ctx context.Context) (*corev1.PodList, error) {
	selector, err := metav1.LabelSelectorAsSelector(
		&metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.AppNameLabelKey:      constants.AppName,
				constants.AppComponentLabelKey: constants.AppComponentRunner,
				constants.AppInstanceLabelKey:  p.rpName,
			},
		},
	)
	if err != nil {
		p.log.Error(err, "failed to make label selector")
		return nil, err
	}

	podList := new(corev1.PodList)
	err = p.k8sClient.List(ctx, podList, client.InNamespace(p.rpNamespace), client.MatchingLabelsSelector{
		Selector: selector,
	})
	if err != nil {
		p.log.Error(err, "failed to list pods")
		return nil, err
	}
	return podList, nil
}

func (p *manageProcess) fetchRunners(ctx context.Context) ([]*github.Runner, error) {
	runnerList, err := p.githubClient.ListRunners(ctx, p.owner, p.repo, []string{p.rpNamespacedName()})
	if err != nil {
		p.log.Error(err, "failed to list runners")
		return nil, err
	}
	return runnerList, nil
}

func (p *manageProcess) updateMetrics(podList *corev1.PodList, runnerList []*github.Runner) {
	p.mu.Lock()
	metrics.UpdateRunnerPoolMetrics(p.rpNamespacedName(), int(p.replicas))
	p.mu.Unlock()

	var currentRunnerNames []string
	for _, runner := range runnerList {
		metrics.UpdateRunnerMetrics(p.rpNamespacedName(), runner.Name, runner.Online, runner.Busy)
		currentRunnerNames = append(currentRunnerNames, runner.Name)
	}

	// Sometimes, offline runners will be deleted from github automatically.
	// Therefore, compare the past runners with the current runners and remove the metrics for the deleted runners.
	removedRunners := difference(p.prevRunnerNames, currentRunnerNames)
	metrics.DeleteRunnerMetrics(p.rpNamespacedName(), removedRunners...)
	p.prevRunnerNames = currentRunnerNames
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

func (p *manageProcess) maintainRunnerPods(ctx context.Context, runnerList []*github.Runner, podList *corev1.PodList) error {
	now := time.Now().UTC()
	lastCheckTime := p.lastCheckTime
	p.lastCheckTime = now

	var numUnlabeledPods int32
	for i := range podList.Items {
		po := &podList.Items[i]
		if _, ok := po.Labels[appsv1.DefaultDeploymentUniqueLabelKey]; !ok {
			numUnlabeledPods++
		}
	}

	p.mu.Lock()
	needNotification := p.needSlackNotification
	slackChannel := p.slackChannel
	extendDuration := p.extendDuration
	recreateDeadline := p.recreateDeadline
	numRemovablePods := p.maxRunnerPods - p.replicas - numUnlabeledPods // numRemovablePods can be a negative number.
	p.mu.Unlock()

	for i := range podList.Items {
		po := &podList.Items[i]
		log := p.log.WithValues("pod", types.NamespacedName{Namespace: po.Namespace, Name: po.Name}.String())
		if po.Status.Phase != corev1.PodRunning {
			log.Info("skip because the status of the pod is not Running", "phase", po.Status.Phase)
			continue
		}

		status, err := p.runnerPodClient.GetStatus(ctx, po.Status.PodIP)
		if err != nil {
			log.Error(err, "failed to get status, skipped maintaining runner pod")
			continue
		}

		if status.State == constants.RunnerPodStateStale {
			err = p.k8sClient.Delete(ctx, po)
			if err != nil && !apierrors.IsNotFound(err) {
				log.Error(err, "failed to delete stale runner pod")
			} else {
				log.Info("deleted stale runner pod")
			}
			continue
		}

		if status.State == constants.RunnerPodStateDebugging {
			needExtend := status.Extend != nil && *status.Extend && extendDuration != 0

			if needNotification && status.FinishedAt.After(lastCheckTime) {
				ch := slackChannel
				if status.SlackChannel != "" {
					ch = status.SlackChannel
				}
				err := p.slackAgentClient.PostResult(ctx, ch, status.Result, needExtend, po.Namespace, po.Name, status.JobInfo)
				if err != nil {
					log.Error(err, "failed to send a notification to slack-agent")
				} else {
					log.Info("sent a notification to slack-agent")
				}
			}

			var needDelete bool
			switch {
			case status.DeletionTime != nil:
				needDelete = now.After(*status.DeletionTime)
			case needExtend:
				needDelete = now.After((*status.FinishedAt).Add(extendDuration))
			default:
				needDelete = true
			}
			if needDelete {
				err := p.k8sClient.Delete(ctx, po)
				if err != nil && !apierrors.IsNotFound(err) {
					log.Error(err, "failed to delete debugging runner pod")
				} else {
					log.Info("deleted debugging runner pod")
				}
				continue
			}

			if status.DeletionTime == nil && status.FinishedAt != nil && status.FinishedAt.After(lastCheckTime) {
				err := p.runnerPodClient.PutDeletionTime(ctx, po.Status.PodIP, status.FinishedAt.Add(extendDuration))
				if err != nil {
					log.Error(err, "failed to set deletion time")
				}
			}
		}

		podRecreateTime := po.CreationTimestamp.Add(recreateDeadline)
		if podRecreateTime.Before(now) && !(runnerBusy(runnerList, po.Name) || status.State == constants.RunnerPodStateDebugging) {
			err = p.k8sClient.Delete(ctx, po)
			if err != nil && !apierrors.IsNotFound(err) {
				log.Error(err, "failed to delete runner pod that exceeded recreate deadline")
			} else {
				log.Info("deleted runner pod that exceeded recreate deadline")
			}
			continue
		}

		// When a job is assigned, the runner pod will be removed from replicaset control.
		if runnerBusy(runnerList, po.Name) || status.State == constants.RunnerPodStateDebugging {
			if p.denyDisruption {
				// if RunnerPool.spec.denyDisruption == true, protect the busy runner by a dedicated PDB.
				if po.Labels[constants.RunnerPodName] != po.Name {
					pdb := &policyv1.PodDisruptionBudget{}
					pdb.SetName(po.Name)
					pdb.SetNamespace(po.Namespace)
					_, err := ctrl.CreateOrUpdate(ctx, p.k8sClient, pdb, func() error {
						pdb.Spec.Selector = &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.RunnerPodName: po.Name,
							},
						}
						pdb.Spec.MinAvailable = &intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1,
						}
						return ctrl.SetControllerReference(po, pdb, p.scheme)
					})
					if err != nil {
						log.Error(err, "failed to create or update protection pdb")
						continue
					}
					log.Info("created or updated protection pdb")
					po.Labels[constants.RunnerPodName] = po.Name
					err = p.k8sClient.Update(ctx, po)
					if err != nil {
						log.Error(err, "failed to relabel runner pod")
						continue
					}
					log.Info("relabeled runner pod")
				}
			}

			if numRemovablePods <= 0 {
				continue
			}
			if _, ok := po.Labels[appsv1.DefaultDeploymentUniqueLabelKey]; !ok {
				continue
			}
			delete(po.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
			err = p.k8sClient.Update(ctx, po)
			if err != nil {
				log.Error(err, "failed to unlink (update) runner pod")
				continue
			}
			numRemovablePods--
			log.Info("unlinked (updated) runner pod")
		}
	}
	return nil
}

func runnerBusy(runnerList []*github.Runner, name string) bool {
	for _, runner := range runnerList {
		if runner.Name == name {
			return runner.Busy
		}
	}
	return false
}

func (p *manageProcess) deleteOfflineRunners(ctx context.Context, runnerList []*github.Runner, podList *corev1.PodList) error {
	for _, runner := range runnerList {
		if runner.Online || podExists(runner.Name, podList) {
			continue
		}
		err := p.githubClient.RemoveRunner(ctx, p.owner, p.repo, runner.ID)
		if err != nil {
			p.log.Error(err, "failed to remove runner", "runner", runner.Name, "runner_id", runner.ID)
			return err
		}
		p.log.Info("removed runner", "runner", runner.Name, "runner_id", runner.ID)
	}
	return nil
}

func (p *manageProcess) deleteAllRunners(ctx context.Context) error {
	runnerList, err := p.githubClient.ListRunners(ctx, p.owner, p.repo, []string{p.rpNamespacedName()})
	if err != nil {
		p.log.Error(err, "failed to list runners")
		return err
	}
	for _, runner := range runnerList {
		err := p.githubClient.RemoveRunner(ctx, p.owner, p.repo, runner.ID)
		if err != nil {
			p.log.Error(err, "failed to remove runner", "runner", runner.Name, "runner_id", runner.ID)
			return err
		}
		p.log.Info("removed runner", "runner", runner.Name, "runner_id", runner.ID)
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
