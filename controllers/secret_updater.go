package controllers

import (
	"context"
	"time"

	constants "github.com/cybozu-go/meows"
	meowsv1alpha1 "github.com/cybozu-go/meows/api/v1alpha1"
	"github.com/cybozu-go/meows/github"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func NewSecretUpdater(client client.Client, interval time.Duration, githubClient github.Client) SecretUpdater {
	return SecretUpdater{
		client:       client,
		interval:     interval,
		githubClient: githubClient,
	}
}

type SecretUpdater struct {
	client       client.Client
	interval     time.Duration
	githubClient github.Client
}

// Start implements Runnable.Start
func (w SecretUpdater) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	logger := log.FromContext(ctx).WithName("SecretUpdater")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			logger.Info("start secret watcher")
			err := w.secretUpdate(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (w SecretUpdater) secretUpdate(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("SecretUpdater")
	rps := meowsv1alpha1.RunnerPoolList{}
	err := w.client.List(ctx, &rps)
	if err != nil {
		return err
	}
	for i := range rps.Items {
		rp := &rps.Items[i]
		s := new(corev1.Secret)
		err := w.client.Get(ctx, types.NamespacedName{Name: rp.GetRunnerSecretName(), Namespace: rp.Namespace}, s)
		if err != nil {
			logger.Error(err, "failed to get secret")
			continue
		}
		expiresAtStr, ok := s.Annotations[constants.RunnerSecretExpiresAtAnnotationKey]
		if ok {
			expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
			if err != nil {
				logger.Error(err, "failed to parse expires-at annotation")
				continue
			}
			if expiresAt.Add(-5 * time.Minute).After(time.Now()) {
				logger.Info("skip update secret", "runnerpool_name", rp.Name, "expires_at", expiresAtStr)
				continue
			}
			logger.Info("update token as it has expired", "runnerpool_name", rp.Name, "expires_at", expiresAtStr)
		} else {
			logger.Info("create first token", "runnerpool_name", rp.Name)
		}

		runnerToken, err := w.githubClient.CreateRegistrationToken(ctx, rp.Spec.RepositoryName)
		if err != nil {
			logger.Error(err, "failed to create actions registration token", "repository", rp.Spec.RepositoryName)
			return err
		}

		newS := s.DeepCopy()
		newS.Annotations = mergeMap(s.Annotations, map[string]string{
			constants.RunnerSecretExpiresAtAnnotationKey: runnerToken.GetExpiresAt().Time.Format(time.RFC3339),
		})
		newS.StringData = map[string]string{
			"runnertoken": runnerToken.GetToken(),
		}
		patch := client.MergeFrom(s)

		err = w.client.Patch(ctx, newS, patch)
		if err != nil {
			logger.Error(err, "failed to patch secret")
			return err
		}
	}
	return nil
}
