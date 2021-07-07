package controllers

import (
	"context"
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/github"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	statusOnline  = "online"
	statusOffline = "offline"
)

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

// RunnerWatcher watches registered self-hosted runners periodically
type RunnerWatcher struct {
	k8sClient client.Client
	log       logr.Logger
	interval  time.Duration

	githubClient    github.RegistrationTokenGenerator
	repositoryNames []string
}

// NewRunnerWatcher returns RunnerWatcher
func NewRunnerWatcher(
	k8sClient client.Client,
	log logr.Logger,
	interval time.Duration,
	githubClient github.RegistrationTokenGenerator,
	repositoryNames []string,
) manager.Runnable {
	return &RunnerWatcher{
		k8sClient:       k8sClient,
		log:             log,
		interval:        interval,
		githubClient:    githubClient,
		repositoryNames: repositoryNames,
	}
}

// Start starts loop to update Actions runner token
func (r *RunnerWatcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)

	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			r.log.Info("run a watching loop")
			err := r.run(ctx)
			if err != nil {
				r.log.Error(err, "failed to run a watching loop")
			}
		}
	}
}

func (r *RunnerWatcher) run(ctx context.Context) error {
	selector, err := metav1.LabelSelectorAsSelector(
		&metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.RunnerOrgLabelKey: r.githubClient.GetOrganizationName(),
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
	// This ensures that the watcher sweeps runners on the certain repositories
	// even if there is no Pod existing.
	for _, repo := range r.repositoryNames {
		podSets[repo] = make(map[string]struct{})
	}

	for _, po := range podList.Items {
		name := types.NamespacedName{Name: po.Name, Namespace: po.Namespace}

		repo, ok := po.Labels[constants.RunnerRepoLabelKey]
		if !ok {
			r.log.Info(fmt.Sprintf("pod does not have %s label, so skip", constants.RunnerRepoLabelKey), "pod", name)
			continue
		}

		if podSets[repo] == nil {
			r.log.Info(fmt.Sprintf("pod has an unregistered repository name %s, so skip", repo), "pod", name)
			continue
		}
		podSets[repo][po.Name] = struct{}{}
	}

	for repo, podSet := range podSets {
		runners, err := r.githubClient.ListRunners(ctx, repo)
		if err != nil {
			r.log.Error(err, "failed to list runners")
			return err
		}

		r.log.Info(fmt.Sprintf("%d pods and %d runners were found", len(podSet), len(runners)))
		for _, runner := range runners {
			if runner.Name == nil || runner.ID == nil || runner.Status == nil {
				err := fmt.Errorf("runner should have name, ID and status %#v", runner)
				r.log.Error(err, "got invalid runner")
				return err
			}
			if _, ok := podSet[*runner.Name]; !ok {
				if *runner.Status == statusOnline {
					r.log.Info("skipped deleting online runner", "runner", *runner.Name, "runner_id", *runner.ID)
					continue
				}

				err := r.githubClient.RemoveRunner(ctx, repo, *runner.ID)
				if err != nil {
					r.log.Error(err, "failed to remove runner", "runner", *runner.Name, "runner_id", *runner.ID)
					return err
				}
				r.log.Info("removed runner", "runner", *runner.Name, "runner_id", *runner.ID)
			}
		}
	}
	return nil
}
