package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	constants "github.com/cybozu-go/github-actions-controller"
	"github.com/cybozu-go/github-actions-controller/github"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
	log := m.log.WithValues("name", types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	})
	log.Info("start mutating pod")

	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err != nil {
		log.Error(err, "failed to decode pod")
		return admission.Errored(http.StatusBadRequest, err)
	}

	org, ok := pod.Labels[constants.RunnerOrgLabelKey]
	if !ok {
		log.Info(
			fmt.Sprintf(
				"skipped because pod does not have %s label",
				constants.RunnerOrgLabelKey,
			),
		)
		return admission.Allowed("non-target")
	}
	targetOrg := m.githubClient.GetOrganizationName()
	if org != targetOrg {
		log.Info(
			fmt.Sprintf(
				"skipped because pod organizationName is not %s",
				targetOrg,
			),
		)
		return admission.Allowed("non-target")
	}

	repo, ok := pod.Labels[constants.RunnerRepoLabelKey]
	if !ok {
		log.Info(
			fmt.Sprintf(
				"skipped because pod does not have %s label",
				constants.RunnerRepoLabelKey,
			),
		)
		return admission.Allowed("non-target")
	}

	var container *corev1.Container
	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]
		if c.Name == constants.RunnerContainerName {
			container = c
			break
		}
	}
	if container == nil {
		err := fmt.Errorf("pod should have a container named %s", constants.RunnerContainerName)
		log.Error(err, "unable to find target container")
		return admission.Errored(http.StatusBadRequest, err)
	}

	token, err := m.githubClient.CreateRegistrationToken(ctx, repo)
	if err != nil {
		log.Error(err, "failed to create actions registration token", "repository", repo)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	for _, v := range container.Env {
		if v.Name == constants.RunnerTokenEnvName {
			return admission.Allowed("token already exists")
		}
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  constants.RunnerTokenEnvName,
		Value: token,
	})
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		log.Error(err, "failed to serialize pod manifest")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
