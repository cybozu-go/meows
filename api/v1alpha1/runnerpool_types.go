// Package v1alpha1 contains API Schema definitions for the actions v1alpha1 API group
//+kubebuilder:object:generate=true
//+groupName=actions.cybozu.com
package v1alpha1

import (
	"fmt"

	constants "github.com/cybozu-go/github-actions-controller"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// NOTE: Some lines of the code are proudly copied from Kubernetes project.
// https://github.com/kubernetes/apimachinery/tree/master/pkg/apis/meta/v1
// https://github.com/kubernetes/api/tree/master/core/v1
// https://github.com/kubernetes/api/tree/master/apps/v1

// RunnerPoolSpec defines the desired state of RunnerPool
type RunnerPoolSpec struct {
	// RepositoryName describes repository name to register Pods as self-hosted
	// runners.
	RepositoryName string `json:"repositoryName"`

	// SlackAgentServiceName is a Service name of Slack agent.
	// +optional
	SlackAgentServiceName *string `json:"slackAgentServiceName,omitempty"`

	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Label selector for pods. Existing ReplicaSets whose pods are
	// selected by this will be the ones affected by this deployment.
	// It must match the pod template's labels.
	Selector *metav1.LabelSelector `json:"selector"`

	// Template describes the pods that will be created.
	Template PodTemplateSpec `json:"template"`

	// The deployment strategy to use to replace existing pods with new ones.
	// +optional
	Strategy appsv1.DeploymentStrategy `json:"strategy,omitempty"`
}

// PodTemplateSpec describes the data a pod should have when created from a template.
// This is slightly modified from corev1.PodTemplateSpec.
type PodTemplateSpec struct {
	// Standard object's metadata.  The name in this metadata is ignored.
	// +optional
	ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the pod.
	Spec corev1.PodSpec `json:"spec"`
}

// ObjectMeta is metadata of objects.
// This is partially copied from metav1.ObjectMeta.
type ObjectMeta struct {
	// Labels is a map of string keys and values.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is a map of string keys and values.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// RunnerPoolStatus defines status of RunnerPool
type RunnerPoolStatus struct {
	// Bound is true when the child Deployment is created.
	// +optional
	Bound bool `json:"bound,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RunnerPool is the Schema for the runnerpools API
type RunnerPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerPoolSpec   `json:"spec"`
	Status RunnerPoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RunnerPoolList contains a list of RunnerPool
type RunnerPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RunnerPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RunnerPool{}, &RunnerPoolList{})
}

func (s *RunnerPoolSpec) validateCreate() field.ErrorList {
	var allErrs field.ErrorList
	var container *v1.Container
	for _, c := range s.Template.Spec.Containers {
		if c.Name == constants.RunnerContainerName {
			container = &c
			break
		}
	}
	if container == nil {
		allErrs = append(allErrs, field.Required(&field.Path{}, fmt.Sprintf("container %s is missing", constants.RunnerContainerName)))
		return allErrs
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
				allErrs = append(allErrs, field.Required(&field.Path{}, fmt.Sprintf("container %s has the reserved environment variable, %v", constants.RunnerContainerName, re)))
			}
		}
	}
	return allErrs
}

func (s *RunnerPoolSpec) validateUpdate(old RunnerPoolSpec) field.ErrorList {
	return nil
}

func (s *RunnerPoolSpec) validateDelete() field.ErrorList {
	return nil
}

func init() {
	SchemeBuilder.Register(&RunnerPool{}, &RunnerPoolList{})
}
