package controllers

import (
	"context"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete

// PodSweeper sweeps Pods managed by RunnerPool controller
type PodSweeper struct {
	k8sClient client.Client
	log       logr.Logger
	interval  time.Duration

	organizationName string
}

// NewPodSweeper returns PodSweeper
func NewPodSweeper(
	k8sClient client.Client,
	log logr.Logger,
	interval time.Duration,

	organizationName string,
) manager.Runnable {
	return &PodSweeper{
		k8sClient:        k8sClient,
		log:              log,
		interval:         interval,
		organizationName: organizationName,
	}
}

// Start starts loop to update Actions runner token
func (r *PodSweeper) Start(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)

	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			r.log.Info("run a sweeping loop")
			err := r.run(ctx)
			if err != nil {
				r.log.Error(err, "failed to run a sweeping loop")
				return err
			}
		}
	}
}

func (r *PodSweeper) run(ctx context.Context) error {
	selector, err := metav1.LabelSelectorAsSelector(
		&metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.RunnerOrgLabelKey: r.organizationName,
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

	now := time.Now().UTC()
	for _, po := range podList.Items {
		name := types.NamespacedName{Name: po.GetName(), Namespace: po.GetNamespace()}

		v, ok := po.Annotations[constants.PodDeletionTimeKey]
		if !ok {
			continue
		}

		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			r.log.Error(err, "skipped deleting pod because failed to parse annotation with "+v, "name", name)
			return err
		}
		if t.After(now) {
			continue
		}

		err = r.k8sClient.Delete(ctx, &po)
		if err != nil {
			r.log.Error(err, "failed to delete pod", "name", name)
			return err
		}
	}

	return nil
}