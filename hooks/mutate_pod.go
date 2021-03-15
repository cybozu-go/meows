package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	actionscontroller "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/github"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups="",resources=pods,verbs=create,versions=v1,name=pod-hook.actions.cybozu.com,path=/pod/mutate,mutating=true,sideEffects=none,admissionReviewVersions={v1,v1beta1}

// PodMutator is a mutating webhook for Pods.
type PodMutator struct {
	k8sClient client.Client
	log       logr.Logger
	decoder   *admission.Decoder

	githubClient github.RegistrationTokenGenerator
}

// NewPodMutator creates a mutating webhook for Pods.
func NewPodMutator(
	k8sClient client.Client,
	log logr.Logger,
	decoder *admission.Decoder,
	githubClient github.RegistrationTokenGenerator,
) http.Handler {
	return &webhook.Admission{
		Handler: PodMutator{
			k8sClient:    k8sClient,
			log:          log,
			decoder:      decoder,
			githubClient: githubClient,
		},
	}
}

// Handle implements admission.Handler interface.
func (m PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	m.log.Info("start mutating %s pod in %s namespace", req.Name, req.Namespace)

	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err != nil {
		m.log.Error(err, "failed to decode %s pod in %s namespace)", pod.Name, pod.Namespace)
		return admission.Errored(http.StatusBadRequest, err)
	}

	repo, ok := pod.Labels[actionscontroller.RunnerRepoLabelKey]
	if !ok {
		m.log.Info(fmt.Sprintf("skipped because pod does not have %s label", actionscontroller.RunnerRepoLabelKey))
		return admission.Allowed("ok")
	}

	token, err := m.githubClient.CreateRegistrationToken(ctx, repo)
	if err != nil {
		m.log.Error(err, "failed to create actions registration token for %s repository", repo)
		return admission.Errored(http.StatusInternalServerError, err)
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
		err := fmt.Errorf("pod should have a container named %s", actionscontroller.RunnerContainerName)
		m.log.Error(err, fmt.Sprintf("unable to find container in %s pod", pod.Name))
		return admission.Errored(http.StatusBadRequest, err)
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  actionscontroller.RunnerTokenEnvName,
		Value: token,
	})
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		m.log.Error(err, "failed to create actions registration token for %s repository", repo)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
