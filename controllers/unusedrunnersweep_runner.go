package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/cybozu-go/github-actions-controller/github"
)

// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create

// UnusedRunnerSweeper sweeps unregistered GitHub Actions Token periodically
type UnusedRunnerSweeper struct {
	log      logr.Logger
	interval time.Duration

	k8sClient    client.Client
	githubClient github.RegistrationTokenGenerator
}

// NewUnusedRunnerSweeper returns OldTokenSweeper
func NewUnusedRunnerSweeper(
	log logr.Logger,
	interval time.Duration,
	k8sClient client.Client,
	githubClient github.RegistrationTokenGenerator,
) manager.Runnable {
	return &UnusedRunnerSweeper{
		log:          log,
		interval:     interval,
		k8sClient:    k8sClient,
		githubClient: githubClient,
	}
}

// Start starts loop to update Actions runner token
func (r *UnusedRunnerSweeper) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)

	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := r.run(ctx)
			if err != nil {
				r.log.Error(err, "failed to run a loop")
				return err
			}
		}
	}
}

func (r *UnusedRunnerSweeper) run(ctx context.Context) error {
	var podList corev1.PodList
	// TODO
	err := r.k8sClient.List(ctx, &podList)
	if err != nil {
		return err
	}

	podSet := make(map[string]struct{})
	for _, p := range podList.Items {
		podSet[p.Name] = struct{}{}
	}

	runners, err := r.githubClient.ListRunners(ctx, "TODO")
	if err != nil {
		return err
	}

	for _, runner := range runners {
		if runner.Name == nil || runner.ID == nil {
			continue
		}
		if _, ok := podSet[*runner.Name]; !ok {
			err := r.githubClient.RemoveRunner(ctx, "TODO", *runner.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
