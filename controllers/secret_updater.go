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

type metricsSet struct {
	retryCount prometheus.Gauge
}

func newSecretUpdater(log logr.Logger, client client.Client, githubClient github.Client) secretUpdater {
	return secretUpdater{
		log:          log.WithName("secretUpdater"),
		client:       client,
		githubClient: githubClient,
		processes:    map[string]*updateProcess{},
	}
}

type secretUpdater struct {
	log          logr.Logger
	client       client.Client
	githubClient github.Client
	processes    map[string]*updateProcess
}

func (u secretUpdater) start(ctx context.Context, rp *meowsv1alpha1.RunnerPool) error {
	rpNamespacedNameStr := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	if _, ok := u.processes[rpNamespacedNameStr]; !ok {
		process := newUpdateProcess(
			u.log,
			u.client,
			u.githubClient,
			rp.Spec.RepositoryName,
			rpNamespacedNameStr,
			types.NamespacedName{Namespace: rp.Namespace, Name: rp.GetRunnerSecretName()},
		)
		err := process.start(ctx)
		if err != nil {
			return err
		}
		u.processes[rpNamespacedNameStr] = process
	}
	return nil
}

func (u *secretUpdater) stop(ctx context.Context, rp *meowsv1alpha1.RunnerPool) error {
	rpNamespacedNameStr := types.NamespacedName{Namespace: rp.Namespace, Name: rp.Name}.String()
	if process, ok := u.processes[rpNamespacedNameStr]; ok {
		if err := process.stop(ctx); err != nil {
			return err
		}
		delete(u.processes, rpNamespacedNameStr)
	}
	return nil
}

type updateProcess struct {
	log                  logr.Logger
	client               client.Client
	githubClient         github.Client
	repositoryName       string
	secretNamespacedName types.NamespacedName
	env                  *well.Environment
	cancel               context.CancelFunc

	metrics       metricsSet
	deleteMetrics func()
}

func newUpdateProcess(log logr.Logger, client client.Client, githubClient github.Client, repositoryName, rpNamespacedNameStr string, secretNamespacedName types.NamespacedName) *updateProcess {
	return &updateProcess{
		log:                  log.WithValues("runnerpool", rpNamespacedNameStr),
		client:               client,
		githubClient:         githubClient,
		secretNamespacedName: secretNamespacedName,
		repositoryName:       repositoryName,
		metrics: metricsSet{
			retryCount: metrics.RunnerPoolSecretRetryCount.WithLabelValues(rpNamespacedNameStr),
		},
		deleteMetrics: func() {
			metrics.RunnerPoolSecretRetryCount.DeleteLabelValues(rpNamespacedNameStr)
		},
	}
}

func (p *updateProcess) start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.env = well.NewEnvironment(ctx)
	p.env.Go(func(ctx context.Context) error {
		p.metrics.retryCount.Set(0)
		defer func() {
			p.deleteMetrics()
		}()
		p.run(ctx)
		return nil
	})
	p.env.Stop()
	return nil
}

func (p *updateProcess) run(ctx context.Context) {
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
				p.metrics.retryCount.Inc()
				updateTime = 5 * time.Minute
				continue
			}
		}
		expiresAt, err := p.getExpiresAt(ctx)
		if err != nil {
			p.log.Error(err, "failed to get expires-at, retry after 5 minutes")
			p.metrics.retryCount.Inc()
			updateTime = 5 * time.Minute
			continue
		}
		updateTime = time.Until(expiresAt.Add(-5 * time.Minute))
		p.log.Info("decide when to update next", "expiresAt", expiresAt.String(), "updateTime", updateTime.String())
		p.metrics.retryCount.Set(0)
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
		"runnertoken": runnerToken.GetToken(),
	}
	patch := client.MergeFrom(s)

	err = p.client.Patch(ctx, newS, patch)
	if err != nil {
		p.log.Error(err, "failed to patch secret")
		return err
	}
	return nil
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
	err := p.client.Get(ctx, p.secretNamespacedName, s)
	if err != nil {
		return nil, err
	}
	return s, nil
}
