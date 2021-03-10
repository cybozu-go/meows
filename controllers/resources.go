package controllers

import (
	"errors"
	"fmt"

	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeDeployment(rp *actionsv1alpha1.RunnerPool) (*appsv1.Deployment, error) {
	rp2 := rp.DeepCopy()
	d := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        rp2.Name,
			Namespace:   rp2.Namespace,
			Labels:      rp2.Labels,
			Annotations: rp2.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: rp2.Spec.DeploymentSpec.Replicas,
			Selector: rp2.Spec.DeploymentSpec.Selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      rp2.Spec.DeploymentSpec.Template.ObjectMeta.Labels,
					Annotations: rp2.Spec.DeploymentSpec.Template.ObjectMeta.Annotations,
				},
				Spec: rp2.Spec.DeploymentSpec.Template.Spec,
			},
		},
	}

	ps := &d.Spec.Template.Spec
	if len(ps.Containers) == 0 {
		return nil, errors.New("spec.deploymentSpec.template.spec.containers should have 1 container")
	}

	// Add Secret volume to Volumes
	foundVolume := false
	for _, v := range ps.Volumes {
		if v.Name == actionsTokenVolumeName {
			foundVolume = true
			break
		}
	}
	if !foundVolume {
		ps.Volumes = append(ps.Volumes, corev1.Volume{
			Name: actionsTokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: actionsTokenSecretName,
				},
			},
		})
	}

	// Add Secret mount to VolumeMounts
	var container *corev1.Container
	for i := range ps.Containers {
		c := &ps.Containers[i]
		if c.Name == controllerContainerName {
			container = c
			break
		}
	}
	if container == nil {
		return nil, fmt.Errorf("%s should exist in one of spec.deploymentSpec.template.spec.containers", controllerContainerName)
	}

	foundMount := false
	for _, v := range container.VolumeMounts {
		if v.Name == actionsTokenMountPath {
			foundMount = true
			break
		}
	}
	if !foundMount {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      actionsTokenVolumeName,
			ReadOnly:  true,
			MountPath: actionsTokenMountPath,
		})
	}

	return &d, nil
}

func makeSecret(name, namespace, token string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: map[string]string{
			tokenSecretKey: token,
		},
		Type: corev1.SecretTypeOpaque,
	}
}
