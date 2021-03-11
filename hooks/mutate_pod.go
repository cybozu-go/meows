package hooks

import (
	"context"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var pmLogger = ctrl.Log.WithName("pod-mutator")

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups="",resources=pods,verbs=create,versions=v1,name=pod-hook.actions.cybozu.com,path=/pod/mutate,mutating=true,sideEffects=none,admissionReviewVersions={v1,v1beta1}

// PodMutator is a mutating webhook for Pods.
type PodMutator struct {
	client  client.Client
	decoder *admission.Decoder
}

// NewPodMutator creates a mutating webhook for Pods.
func NewPodMutator(c client.Client, dec *admission.Decoder) http.Handler {
	return &webhook.Admission{Handler: PodMutator{c, dec}}
}

// Handle implements admission.Handler interface.
func (m PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)

	}
	if len(pod.Spec.Containers) == 0 {
		return admission.Denied("pod has no containers")
	}

	// short cut
	if len(pod.Spec.Volumes) == 0 {
		return admission.Allowed("no volumes")
	}

	return admission.Allowed("no volumes")
}
