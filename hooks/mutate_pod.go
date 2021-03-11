package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	actionscontroller "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/github"
)

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups="",resources=pods,verbs=create,versions=v1,name=pod-hook.actions.cybozu.com,path=/pod/mutate,mutating=true,sideEffects=none,admissionReviewVersions={v1,v1beta1}

// PodMutator is a mutating webhook for Pods.
type PodMutator struct {
	k8sClient client.Client
	decoder   *admission.Decoder

	githubClient github.RegistrationTokenGenerator
}

// NewPodMutator creates a mutating webhook for Pods.
func NewPodMutator(
	k8sClient client.Client,
	decoder *admission.Decoder,
	githubClient github.RegistrationTokenGenerator,
) http.Handler {
	return &webhook.Admission{
		Handler: PodMutator{
			k8sClient:    k8sClient,
			decoder:      decoder,
			githubClient: githubClient,
		},
	}
}

// Handle implements admission.Handler interface.
func (m PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if _, ok := pod.Labels[actionscontroller.RunnerWebhookLabelKey]; !ok {
		return admission.Allowed(fmt.Sprintf("skipped because pod does not have %s label", actionscontroller.RunnerWebhookLabelKey))
	}

	if len(pod.Spec.Containers) == 0 {
		return admission.Denied("denied because pod has no containers")
	}

	var container *corev1.Container
	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]
		if c.Name == actionscontroller.RunnerContainerName {
			container = c
			break
		}
	}
	if container == nil {
		return admission.Denied(fmt.Sprintf("denied because pod has no container name %s", actionscontroller.RunnerContainerName))
	}

	token, err := m.githubClient.CreateRegistrationToken(ctx)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  actionscontroller.RunnerTokenEnvName,
		Value: token,
	})

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
