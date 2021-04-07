package agent

import (
	"context"
	"encoding/json"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AnnotateDeletionTime annotates a given time to a Pod.
func AnnotateDeletionTime(
	ctx context.Context,
	name string,
	namespace string,
	t time.Time,
) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	po, err := clientset.
		CoreV1().
		Pods(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	patched := po.DeepCopy()
	if patched.Annotations == nil {
		patched.Annotations = make(map[string]string)
	}
	patched.Annotations[constants.PodDeletionTimeKey] = t.UTC().Format(time.RFC3339)

	before, err := json.Marshal(po)
	if err != nil {
		return err
	}

	after, err := json.Marshal(patched)
	if err != nil {
		return err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(before, after, &corev1.Pod{})
	if err != nil {
		return err
	}

	_, err = clientset.
		CoreV1().
		Pods(namespace).
		Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}
