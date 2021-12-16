// Package v1alpha1 contains API Schema definitions for the meows v1alpha1 API group
//+kubebuilder:object:generate=true
//+groupName=meows.cybozu.com
package v1alpha1

import (
	"fmt"
	"time"

	constants "github.com/cybozu-go/meows"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var reservedEnvNames = map[string]bool{
	constants.PodNameEnvName:        true,
	constants.PodNamespaceEnvName:   true,
	constants.RunnerOrgEnvName:      true,
	constants.RunnerRepoEnvName:     true,
	constants.RunnerPoolNameEnvName: true,
	constants.RunnerOptionEnvName:   true,
}

// RunnerPoolSpec defines the desired state of RunnerPool
type RunnerPoolSpec struct {
	// RepositoryName describes repository name to register Pods as self-hosted runners.
	// If this field is omitted or the empty string (`""`) is specified, Pods will be registered as organization-level self-hosted runners.
	// +optional
	RepositoryName string `json:"repositoryName"`

	// CredentialSecretName is a Secret name that contains a GitHub Credential.
	// If this field is omitted or the empty string (`""`) is specified, meows uses the default secret name (`meows-github-cred`).
	// +optional
	CredentialSecretName string `json:"credentialSecretName"`

	// Number of desired runner pods to accept a new job. Defaults to 1.
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Number of desired runner pods to keep. Defaults to 0.
	// If this field is 0, it will keep the number of pods specified in replicas.
	// +kubebuilder:default=0
	// +optional
	MaxRunnerPods int32 `json:"maxRunnerPods,omitempty"`

	// Command that runs when the runner pods will be created.
	// +optional
	SetupCommand []string `json:"setupCommand,omitempty"`

	// Configuration of a Slack agent.
	// +optional
	SlackAgent SlackAgentConfig `json:"slackAgent,omitempty"`

	// Deadline for the Pod to be recreated.
	// +kubebuilder:default="24h"
	// +optional
	RecreateDeadline string `json:"recreateDeadline,omitempty"`

	// Template describes the runner pods that will be created.
	// +optional
	Template RunnerPodTemplateSec `json:"template,omitempty"`
}

type SlackAgentConfig struct {
	// ServiceName is a Service name of Slack agent.
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

	// Slack channel which the job results are reported.
	// If this field is omitted, the default channel specified in slack-agent options will be used.
	// +optional
	Channel string `json:"channel,omitempty"`
}

type RunnerPodTemplateSec struct {
	// Standard object's metadata.  Only `annotations` and `labels` are valid.
	// +optional
	ObjectMeta `json:"metadata"`

	// Docker image name for the runner container.
	// +optional
	Image string `json:"image,omitempty"`

	// Image pull policy for the runner container.
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is a list of secret names in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Security options for the runner container.
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// List of environment variables to set in the runner container.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Compute Resources required by the runner container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// WorkVolume is the volume source for the working directory.
	// If pod is not given a volume definition, it uses an empty dir.
	// +optional
	WorkVolume *corev1.VolumeSource `json:"workVolume,omitempty"`

	// Pod volumes to mount into the runner container's filesystem.
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// List of volumes that can be mounted by containers belonging to the pod.
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// Name of the service account that the Pod use.
	// +kubebuilder:default="default"
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
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
	return s.validateCommon()
}

func (s *RunnerPoolSpec) validateUpdate(old RunnerPoolSpec) field.ErrorList {
	var allErrs field.ErrorList
	p := field.NewPath("spec")

	if s.RepositoryName != old.RepositoryName {
		pp := p.Child("repositoryName")
		allErrs = append(allErrs, field.Forbidden(pp, "the field is immutable"))
	}

	return append(allErrs, s.validateCommon()...)
}

func (s *RunnerPoolSpec) validateCommon() field.ErrorList {
	var allErrs field.ErrorList
	p := field.NewPath("spec")

	if !(s.MaxRunnerPods == 0 || s.Replicas <= s.MaxRunnerPods) {
		allErrs = append(allErrs, field.Invalid(p.Child("maxRunnerPods"), s.MaxRunnerPods, "this value should be 0, or greater-than or equal-to replicas."))
	}

	_, err := time.ParseDuration(s.RecreateDeadline)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(p.Child("recreateDeadline"), s.RecreateDeadline, "this value should be able to parse using time.ParseDuration"))
	}

	for i, e := range s.Template.Env {
		if reservedEnvNames[e.Name] {
			allErrs = append(allErrs, field.Forbidden(p.Child("template").Child("env").Index(i),
				fmt.Sprintf("using the reserved environment variable %s in %s is forbidden", e.Name, constants.RunnerContainerName)))
		}
	}

	return allErrs
}

// GetRunnerDeploymentName returns the Deployment name for runners.
func (r *RunnerPool) GetRunnerDeploymentName() string {
	return r.Name
}

func (r *RunnerPool) GetRunnerSecretName() string {
	return "runner-token-" + r.Name
}

func (r *RunnerPool) IsOrgLevel() bool {
	return r.Spec.RepositoryName == ""
}
