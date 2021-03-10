package controllers

import (
	"errors"
	"fmt"

	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	tokenSecretKey = "GITHUB_ACTIONS_TOKEN"
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
	return &d, nil
}

var _ = makeSecret

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
