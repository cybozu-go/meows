package agent

import (
	"context"
	"fmt"
	"time"

	constants "github.com/cybozu-go/github-actions-controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	p := []byte(fmt.Sprintf(
		`{ "metadata": { "annotations": { "%s": "%s" } } }`,
		constants.PodDeletionTimeKey,
		t.UTC().Format(time.RFC3339),
	))

	_, err = clientset.
		CoreV1().
		Pods(namespace).
		Patch(ctx, name, types.StrategicMergePatchType, p, metav1.PatchOptions{})
	return err
}
