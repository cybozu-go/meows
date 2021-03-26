// Package v1alpha1 contains API Schema definitions for the actions v1alpha1 API group
//+kubebuilder:object:generate=true
//+groupName=actions.cybozu.com
package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// SlackAgentURL is a URL of Slack agent.
	// +optional
	SlackAgentURL *string `json:"slackAgentURL,omitempty"`

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
