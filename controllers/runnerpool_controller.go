/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
)

const (
	runnerPoolFinalizer = "actions.cybozu.com/runnerpool"

	controllerContainerName = "controller"

	actionsTokenSecretName = "github-actions-token"
	actionsTokenSecretKey  = "GITHUB_ACTIONS_TOKEN"
	actionsTokenVolumeName = "github-actions-token"
	actionsTokenMountPath  = "/etc/github/token.json"
)

// RunnerPoolReconciler reconciles a RunnerPool object
type RunnerPoolReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=actions.cybozu.com,resources=runnerpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=actions.cybozu.com,resources=runnerpools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RunnerPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("runnerpool", req.NamespacedName)

	log.Info("debug0")
	rp := &actionsv1alpha1.RunnerPool{}
	if err := r.Get(ctx, req.NamespacedName, rp); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to get RunnerPool", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	log.Info("debug1")
	if rp.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(rp, runnerPoolFinalizer) {
			rp2 := rp.DeepCopy()
			controllerutil.AddFinalizer(rp2, runnerPoolFinalizer)
			patch := client.MergeFrom(rp)
			if err := r.Patch(ctx, rp2, patch); err != nil {
				log.Error(err, "failed to add finalizer")
				return ctrl.Result{}, err
			}
		}
	} else {
		targetDeployment := &appsv1.Deployment{}
		err := r.Get(ctx, req.NamespacedName, targetDeployment)
		if err == nil {
			return ctrl.Result{}, errors.New("deployment is not deleted yet")
		}
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		rp2 := rp.DeepCopy()
		controllerutil.RemoveFinalizer(rp2, runnerPoolFinalizer)
		patch := client.MergeFrom(rp)
		if err := r.Patch(ctx, rp2, patch); err != nil {
			log.Error(err, "failed to remove finalizer")
			return ctrl.Result{}, err
		}
	}

	log.Info("debug2")
	d, err := r.makeDeployment(rp)
	if err != nil {
		log.Error(err, "failed to make Deployment definition")
		return ctrl.Result{}, err
	}

	log.Info("debug3")
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, d, func() error {
		return ctrl.SetControllerReference(rp, d, r.Scheme)
	})
	if err != nil {
		log.Error(err, "unable to create-or-update Deployment")
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		log.Info("reconcile Deployment successfully", "op", op)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&actionsv1alpha1.RunnerPool{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RunnerPoolReconciler) makeDeployment(rp *actionsv1alpha1.RunnerPool) (*appsv1.Deployment, error) {
	rp2 := rp.DeepCopy()
	d := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        rp2.Name,
			Namespace:   rp2.Namespace,
			Labels:      rp2.Labels,
			Annotations: rp2.Annotations,
		},
		Spec: rp2.Spec.DeploymentSpec,
	}

	ps := d.Spec.Template.Spec
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
	for _, c := range ps.Containers {
		if c.Name == controllerContainerName {
			container = &c
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
