package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/cybozu-go/github-actions-controller/github"
)

const (
	secretName = "actions-token"

	tokenSecretKey = "GITHUB_ACTIONS_TOKEN"
)

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create

// ActionsTokenUpdator updates GitHub Actions Token periodically
type ActionsTokenUpdator struct {
	log      logr.Logger
	recorder record.EventRecorder
	interval time.Duration

	k8sClient    client.Client
	githubClient github.RegistrationTokenGenerator
}

// NewActionsTokenUpdator returns a new ActionsTokenUpdator struct
func NewActionsTokenUpdator(
	log logr.Logger,
	recorder record.EventRecorder,
	interval time.Duration,
	k8sClient client.Client,
	githubClient github.RegistrationTokenGenerator,
) manager.Runnable {
	return &ActionsTokenUpdator{
		log:          log,
		recorder:     recorder,
		interval:     interval,
		k8sClient:    k8sClient,
		githubClient: githubClient,
	}
}

// Start starts loop to update Actions runner token
func (u *ActionsTokenUpdator) Start(ctx context.Context) error {
	ticker := time.NewTicker(u.interval)

	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := u.reconcile(ctx)
			if err != nil {
				u.log.Error(err, "failed to reconcile")
				return err
			}
		}
	}
}

func (u *ActionsTokenUpdator) reconcile(ctx context.Context) error {
	return nil
}
