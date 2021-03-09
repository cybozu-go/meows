package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	k8sClient client.Client
	namespace string

	githubClient        github.Registerer
	repositoryOwnerName string
	repositoryName      string
}

// NewActionsTokenUpdator returns a new ActionsTokenUpdator struct
func NewActionsTokenUpdator(
	log logr.Logger,
	recorder record.EventRecorder,
	interval time.Duration,
	k8sClient client.Client,
	namespace string,
	githubClient *github.Client,
	repositoryOwnerName string,
	repositoryName string,
) manager.Runnable {
	return &ActionsTokenUpdator{
		log:                 log,
		recorder:            recorder,
		interval:            interval,
		k8sClient:           k8sClient,
		namespace:           namespace,
		githubClient:        githubClient,
		repositoryOwnerName: repositoryOwnerName,
		repositoryName:      repositoryName,
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
	token, err := u.githubClient.CreateRegistrationToken(
		ctx,
		u.repositoryOwnerName,
		u.repositoryName,
	)
	if err != nil {
		return err
	}

	var s corev1.Secret
	err = u.k8sClient.Get(ctx, types.NamespacedName{
		Namespace: u.namespace,
		Name:      secretName,
	}, &s)
	switch {
	case apierrors.IsNotFound(err):
		return u.k8sClient.Create(ctx, u.makeSecret(token))
	case err == nil:
		return u.k8sClient.Update(ctx, u.makeSecret(token))
	default:
		return err
	}
}

func (u *ActionsTokenUpdator) makeSecret(token string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		StringData: map[string]string{
			tokenSecretKey: token,
		},
		Type: corev1.SecretTypeOpaque,
	}
}
