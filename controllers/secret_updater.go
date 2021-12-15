package controllers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	constants "github.com/cybozu-go/meows"
	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/well"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretUpdater creates a registration token for self-hosted runners and updates a secret periodically.
// It generates one goroutine for each RunnerPool CR.
type SecretUpdater interface {
	Start(*meowsv1alpha1.RunnerPool, *github.ClientCredential) error
	Stop(*meowsv1alpha1.RunnerPool) error
	StopAll()
}

type secretUpdater struct {
	log                 logr.Logger
	k8sClient           client.Client
	githubClientFactory github.ClientFactory
	mu                  sync.Mutex
	stopped             bool
	processes           map[string]*updateProcess
}

func NewSecretUpdater(log logr.Logger, k8sClient client.Client, githubClientFactory github.ClientFactory) SecretUpdater {
	return &secretUpdater{
		log:                 log.WithName("SecretUpdater"),
		k8sClient:           k8sClient,
		githubClientFactory: githubClientFactory,
		processes:           map[string]*updateProcess{},
	}
}

func (u *secretUpdater) Start(rp *meowsv1alpha1.RunnerPool, cred *github.ClientCredential) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.stopped {
		return errors.New("SecretUpdater is already stopped")
	}

	rpNamespacedName := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	if _, ok := u.processes[rpNamespacedName]; !ok {
		githubClient, err := u.githubClientFactory.New(cred)
		if err != nil {
			return fmt.Errorf("failed to create a github client; %w", err)
		}
		process := newUpdateProcess(
			u.log.WithValues("runnerpool", rpNamespacedName),
			u.k8sClient,
			githubClient,
			rp,
		)
		process.start()
		u.processes[rpNamespacedName] = process
	}
	return nil
}

func (u *secretUpdater) Stop(rp *meowsv1alpha1.RunnerPool) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	rpNamespacedName := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	if process, ok := u.processes[rpNamespacedName]; ok {
		if err := process.stop(); err != nil {
			return err
		}
		delete(u.processes, rpNamespacedName)
	}
	return nil
}

func (u *secretUpdater) StopAll() {
	u.mu.Lock()
	defer u.mu.Unlock()

	for _, process := range u.processes {
		process.stop()
	}
	u.processes = nil
	u.stopped = true
}

type updateProcess struct {
	// Given from outside. Not update internally.
	log          logr.Logger
	k8sClient    client.Client
	githubClient github.Client
	rpNamespace  string
	rpName       string
	secretName   string
	owner        string
	repo         string

	// Update internally.
	env               *well.Environment
	cancel            context.CancelFunc
	retryCountMetrics prometheus.Counter
	deleteMetrics     func()
}

func newUpdateProcess(log logr.Logger, k8sClient client.Client, githubClient github.Client, rp *meowsv1alpha1.RunnerPool) *updateProcess {
	rpNamespacedName := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	return &updateProcess{
		log:               log,
		k8sClient:         k8sClient,
		githubClient:      githubClient,
		rpNamespace:       rp.Namespace,
		rpName:            rp.Name,
		secretName:        rp.GetRunnerSecretName(),
		owner:             rp.GetOwner(),
		repo:              rp.GetRepository(),
		retryCountMetrics: metrics.RunnerPoolSecretRetryCount.WithLabelValues(rpNamespacedName),
		deleteMetrics: func() {
			metrics.RunnerPoolSecretRetryCount.DeleteLabelValues(rpNamespacedName)
		},
	}
}

func (p *updateProcess) start() {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.env = well.NewEnvironment(ctx)
	p.env.Go(func(ctx context.Context) error {
		p.run(ctx)
		p.deleteMetrics()
		return nil
	})
	p.env.Stop()
}

func (p *updateProcess) stop() error {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
		if err := p.env.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func (p *updateProcess) run(ctx context.Context) {
	p.log.Info("start a secret updater process")
	waitTime := time.Second
	for {
		select {
		case <-ctx.Done():
			p.log.Info("stop a secret updater process")
			return
		case <-time.After(waitTime):
		}

		s, err := p.getSecret(ctx)
		if apierrors.IsNotFound(err) {
			p.log.Error(err, "secret is not found")
			waitTime = time.Second
			continue
		} else if err != nil {
			p.log.Error(err, "failed to get secret, retry after 1 minutes")
			p.retryCountMetrics.Inc()
			waitTime = time.Minute
			continue
		}

		if need, updateTime := p.needUpdate(s); !need {
			p.log.Info("wait until next update time", "updateTime", updateTime.Format(time.RFC3339))
			waitTime = time.Until(updateTime)
			continue
		}

		expiresAt, err := p.updateSecret(ctx, s)
		if err != nil {
			p.log.Error(err, "failed to update secret, retry after 1 minutes")
			p.retryCountMetrics.Inc()
			waitTime = time.Minute
			continue
		}

		p.log.Info("secret is successfully updated", "expiresAt", expiresAt.Format(time.RFC3339))
		waitTime = time.Second
	}
}

func (p *updateProcess) getSecret(ctx context.Context) (*corev1.Secret, error) {
	s := new(corev1.Secret)
	err := p.k8sClient.Get(ctx, types.NamespacedName{Namespace: p.rpNamespace, Name: p.secretName}, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (p *updateProcess) needUpdate(s *corev1.Secret) (bool, time.Time) {
	expiresAtStr, ok := s.Annotations[constants.RunnerSecretExpiresAtAnnotationKey]
	if !ok {
		p.log.Error(nil, "not annotated expires-at")
		return true, time.Time{}
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		p.log.Error(err, "invalid value expires-at")
		return true, time.Time{}
	}
	updateTime := expiresAt.Add(-5 * time.Minute)

	if time.Now().After(updateTime) {
		return true, time.Time{}
	}

	return false, updateTime
}

func (p *updateProcess) updateSecret(ctx context.Context, s *corev1.Secret) (time.Time, error) {
	runnerToken, err := p.githubClient.CreateRegistrationToken(ctx, p.owner, p.repo)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create actions registration token; %w", err)
	}
	expiresAt := runnerToken.GetExpiresAt().Time

	newS := s.DeepCopy()
	newS.Annotations = mergeMap(s.Annotations, map[string]string{
		constants.RunnerSecretExpiresAtAnnotationKey: expiresAt.Format(time.RFC3339),
	})
	newS.StringData = map[string]string{
		constants.RunnerTokenFileName: runnerToken.GetToken(),
	}
	patch := client.MergeFrom(s)

	err = p.k8sClient.Patch(ctx, newS, patch)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to patch secret; %w", err)
	}
	return expiresAt, nil
}
