package controllers

import (
	"context"
	"time"

	constants "github.com/cybozu-go/meows"
	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/github"
	"github.com/cybozu-go/meows/metrics"
	"github.com/cybozu-go/well"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretUpdater creates a registration token for self-hosted runners and updates a secret periodically.
// It generates one goroutine for each RunnerPool CR.
type SecretUpdater interface {
	Start(context.Context, *meowsv1alpha1.RunnerPool) error
	Stop(context.Context, *meowsv1alpha1.RunnerPool) error
}

type secretUpdater struct {
	log          logr.Logger
	k8sClient    client.Client
	githubClient github.Client
	processes    map[string]*updateProcess
}

func NewSecretUpdater(log logr.Logger, k8sClient client.Client, githubClient github.Client) SecretUpdater {
	return &secretUpdater{
		log:          log.WithName("SecretUpdater"),
		k8sClient:    k8sClient,
		githubClient: githubClient,
		processes:    map[string]*updateProcess{},
	}
}

func (u *secretUpdater) Start(ctx context.Context, rp *meowsv1alpha1.RunnerPool) error {
	rpNamespacedName := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	if _, ok := u.processes[rpNamespacedName]; !ok {
		process := newUpdateProcess(
			u.log.WithValues("runnerpool", rpNamespacedName),
			u.k8sClient,
			u.githubClient,
			rp,
		)
		process.start(ctx)
		u.processes[rpNamespacedName] = process
	}
	return nil
}

func (u *secretUpdater) Stop(ctx context.Context, rp *meowsv1alpha1.RunnerPool) error {
	rpNamespacedName := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	if process, ok := u.processes[rpNamespacedName]; ok {
		if err := process.stop(ctx); err != nil {
			return err
		}
		delete(u.processes, rpNamespacedName)
	}
	return nil
}

type updateProcess struct {
	// Given from outside. Not update internally.
	log            logr.Logger
	k8sClient      client.Client
	githubClient   github.Client
	rpNamespace    string
	rpName         string
	secretName     string
	repositoryName string

	// Update internally.
	env               *well.Environment
	cancel            context.CancelFunc
	retryCountMetrics prometheus.Gauge
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
		repositoryName:    rp.Spec.RepositoryName,
		retryCountMetrics: metrics.RunnerPoolSecretRetryCount.WithLabelValues(rpNamespacedName),
		deleteMetrics: func() {
			metrics.RunnerPoolSecretRetryCount.DeleteLabelValues(rpNamespacedName)
		},
	}
}

func (p *updateProcess) start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.env = well.NewEnvironment(ctx)
	p.env.Go(func(ctx context.Context) error {
		defer p.deleteMetrics()
		p.run(ctx)
		return nil
	})
	p.env.Stop()
}

func (p *updateProcess) stop(ctx context.Context) error {
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
	p.retryCountMetrics.Set(0)
	p.log.Info("wait for an empty secret to be created by reconcile")
	for {
		_, err := p.getSecret(ctx)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			p.log.Info("stop a secret updater process")
			return
		case <-time.After(1 * time.Second):
		}
	}

	p.log.Info("start a secret updater process")
	updateTime := 1 * time.Second
	for {
		select {
		case <-ctx.Done():
			p.log.Info("stop a secret updater process")
			return
		case <-time.After(updateTime):
			err := p.updateSecret(ctx)
			if err != nil {
				p.log.Error(err, "failed to update secret, retry after 5 minutes")
				p.retryCountMetrics.Inc()
				updateTime = 5 * time.Minute
				continue
			}
		}
		expiresAt, err := p.getExpiresAt(ctx)
		if err != nil {
			p.log.Error(err, "failed to get expires-at, retry after 5 minutes")
			p.retryCountMetrics.Inc()
			updateTime = 5 * time.Minute
			continue
		}
		updateTime = time.Until(expiresAt.Add(-5 * time.Minute))
		p.log.Info("decide when to update next", "expiresAt", expiresAt.String(), "updateTime", updateTime.String())
		p.retryCountMetrics.Set(0)
	}
}

func (p *updateProcess) updateSecret(ctx context.Context) error {
	s, err := p.getSecret(ctx)
	if err != nil {
		return err
	}

	runnerToken, err := p.githubClient.CreateRegistrationToken(ctx, p.repositoryName)
	if err != nil {
		p.log.Error(err, "failed to create actions registration token", "repository", p.repositoryName)
		return err
	}

	newS := s.DeepCopy()
	newS.Annotations = mergeMap(s.Annotations, map[string]string{
		constants.RunnerSecretExpiresAtAnnotationKey: runnerToken.GetExpiresAt().Format(time.RFC3339),
	})
	newS.StringData = map[string]string{
		constants.RunnerTokenFileName: runnerToken.GetToken(),
	}
	patch := client.MergeFrom(s)

	err = p.k8sClient.Patch(ctx, newS, patch)
	if err != nil {
		p.log.Error(err, "failed to patch secret")
		return err
	}
	return nil
}

func (p *updateProcess) getExpiresAt(ctx context.Context) (time.Time, error) {
	s, err := p.getSecret(ctx)
	if err != nil {
		return time.Time{}, err
	}
	expiresAtStr, ok := s.Annotations[constants.RunnerSecretExpiresAtAnnotationKey]
	if !ok {
		p.log.Info("not annotated expires-at")
		return time.Now(), nil
	}
	return time.Parse(time.RFC3339, expiresAtStr)
}

func (p *updateProcess) getSecret(ctx context.Context) (*corev1.Secret, error) {
	s := new(corev1.Secret)
	err := p.k8sClient.Get(ctx, types.NamespacedName{Namespace: p.rpNamespace, Name: p.secretName}, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}
