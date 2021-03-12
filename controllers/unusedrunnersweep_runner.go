package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	actionscontroller "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/github"
)

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

// UnusedRunnerSweeper sweeps unregistered GitHub Actions Token periodically
type UnusedRunnerSweeper struct {
	k8sClient client.Client
	log       logr.Logger
	interval  time.Duration

	githubClient     github.RegistrationTokenGenerator
	organizationName string
}

// NewUnusedRunnerSweeper returns OldTokenSweeper
func NewUnusedRunnerSweeper(
	k8sClient client.Client,
	log logr.Logger,
	interval time.Duration,
	githubClient github.RegistrationTokenGenerator,
	organizationName string,
) manager.Runnable {
	return &UnusedRunnerSweeper{
		k8sClient:        k8sClient,
		log:              log,
		interval:         interval,
		githubClient:     githubClient,
		organizationName: organizationName,
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
	selector, err := metav1.LabelSelectorAsSelector(
		&metav1.LabelSelector{
			MatchLabels: map[string]string{
				actionscontroller.RunnerOrgLabelKey: r.organizationName,
			},
		},
	)
	if err != nil {
		r.log.Error(err, "failed to make label selector")
		return err
	}

	var podList corev1.PodList
	err = r.k8sClient.List(ctx, &podList, client.MatchingLabelsSelector{
		Selector: selector,
	})
	if err != nil {
		r.log.Error(err, "failed to list pods")
		return err
	}

	podSets := make(map[string]map[string]struct{})
	for _, po := range podList.Items {
		repo, ok := po.Labels[actionscontroller.RunnerRepoLabelKey]
		if !ok {
			err := fmt.Errorf("pod should have %s label", actionscontroller.RunnerRepoLabelKey)
			r.log.Error(err, "unable to get repository name")
			return err
		}

		if podSets[repo] == nil {
			podSets[repo] = make(map[string]struct{})
		}
		podSets[repo][po.Name] = struct{}{}
	}

	for repo, podSet := range podSets {
		runners, err := r.githubClient.ListRunners(ctx, repo)
		if err != nil {
			r.log.Error(err, "failed to list runners")
			return err
		}

		for _, runner := range runners {
			if runner.Name == nil || runner.ID == nil {
				continue
			}
			if _, ok := podSet[*runner.Name]; !ok {
				err := r.githubClient.RemoveRunner(ctx, repo, *runner.ID)
				if err != nil {
					r.log.Error(err, fmt.Sprintf("failed to remove runner %s (id: %d)", *runner.Name, *runner.ID))
					return err
				}
			}
		}
	}
	return nil
}