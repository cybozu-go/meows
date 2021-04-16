package hooks

import (
	"context"
	"fmt"
	"net/http"

	constants "github.com/cybozu-go/github-actions-controller"
	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups=actions.cybozu.com,resources=runnerpools,verbs=create;update,versions=v1alpha1,name=runnerpool-hook.actions.cybozu.com,path=/runnerpool/validate,mutating=false,sideEffects=none,admissionReviewVersions={v1alpha1,v1,v1beta1}

// RunnerPoolValidator is a validation webhook for RunnerPools.
type RunnerPoolValidator struct {
	client  client.Client
	log     logr.Logger
	decoder *admission.Decoder

	repositoryNameList []string
}

// NewRunnerPoolValidator creates a validating webhook for RunnerPools.
func NewRunnerPoolValidator(
	k8sClient client.Client,
	log logr.Logger,
	decoder *admission.Decoder,
	repositoryNameList []string,
) http.Handler {
	return &webhook.Admission{
		Handler: RunnerPoolValidator{
			client:             k8sClient,
			log:                log,
			decoder:            decoder,
			repositoryNameList: repositoryNameList,
		},
	}
}

// Handle implements admission.Handler interface.
func (v RunnerPoolValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	namespacedName := types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}
	v.log.Info("start validating runnerpool", "name", namespacedName)

	rp := &actionsv1alpha1.RunnerPool{}
	err := v.decoder.Decode(req, rp)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	found := false
	for _, r := range v.repositoryNameList {
		if r == rp.Spec.RepositoryName {
			found = true
			break
		}
	}
	if !found {
		return admission.Denied(fmt.Sprintf("runnerpool is not specifying one of the valid repository names. The valid repository names are %v", v.repositoryNameList))
	}

	var container *v1.Container
	for _, c := range rp.Spec.Template.Spec.Containers {
		if c.Name == constants.RunnerContainerName {
			container = &c
			break
		}
	}
	if container == nil {
		return admission.Denied(fmt.Sprintf("the container named %s should exist in the runnerpool", constants.RunnerContainerName))
	}

	reservedEnvNames := []string{
		constants.PodNameEnvName,
		constants.PodNamespaceEnvName,
		constants.RunnerOrgEnvName,
		constants.RunnerRepoEnvName,
	}
	for _, e := range container.Env {
		for _, re := range reservedEnvNames {
			if e.Name == re {
				return admission.Denied(fmt.Sprintf("runner container can not use %v env name as it is revserved by runnerpool controller", re))
			}
		}
	}

	return admission.Allowed("ok")
}
