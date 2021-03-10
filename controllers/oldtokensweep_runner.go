package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/cybozu-go/github-actions-controller/github"
)

// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create

// OldTokenSweeper sweeps unregistered GitHub Actions Token periodically
type OldTokenSweeper struct {
	log      logr.Logger
	recorder record.EventRecorder
	interval time.Duration

	k8sClient    client.Client
	githubClient github.RegistrationTokenGenerator
}

// NewOldTokenSweeper returns OldTokenSweeper
func NewOldTokenSweeper(
	log logr.Logger,
	recorder record.EventRecorder,
	interval time.Duration,
	k8sClient client.Client,
	githubClient github.RegistrationTokenGenerator,
) manager.Runnable {
	return &OldTokenSweeper{
		log:          log,
		recorder:     recorder,
		interval:     interval,
		k8sClient:    k8sClient,
		githubClient: githubClient,
	}
}

// Start starts loop to update Actions runner token
func (r *OldTokenSweeper) Start(ctx context.Context) error {
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

func (r *OldTokenSweeper) run(ctx context.Context) error {
	var podList corev1.PodList
	err := r.k8sClient.List(ctx, &podList)
	if err != nil {
		return err
	}

	podSet := make(map[string]struct{})
	for _, p := range podList.Items {
		podSet[p.Name] = struct{}{}
	}

	runners, err := r.githubClient.ListOrganizationRunners(ctx)
	if err != nil {
		return err
	}

	for _, runner := range runners {
		if runner.Name == nil || runner.ID == nil {
			continue
		}
		if _, ok := podSet[*runner.Name]; !ok {
			err := r.githubClient.RemoveOrganizationRunner(ctx, *runner.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
