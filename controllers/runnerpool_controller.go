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

	actionscontroller "github.com/cybozu-go/github-actions-controller"
	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
)

const runnerPoolFinalizer = "actions.cybozu.com/runnerpool"

// RunnerPoolReconciler reconciles a RunnerPool object
type RunnerPoolReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	organizationName string
}

// NewRunnerPoolReconciler creates RunnerPoolReconciler
func NewRunnerPoolReconciler(
	client client.Client,
	log logr.Logger,
	scheme *runtime.Scheme,
	organizationName string,
) *RunnerPoolReconciler {
	return &RunnerPoolReconciler{
		Client:           client,
		Log:              log,
		Scheme:           scheme,
		organizationName: organizationName,
	}
}

//+kubebuilder:rbac:groups=actions.cybozu.com,resources=runnerpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=actions.cybozu.com,resources=runnerpools/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RunnerPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("runnerpool", req.NamespacedName)

	rp := &actionsv1alpha1.RunnerPool{}
	if err := r.Get(ctx, req.NamespacedName, rp); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to get RunnerPool", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

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

	d, err := r.makeDeployment(rp)
	if err != nil {
		log.Error(err, "failed to make Deployment definition")
		return ctrl.Result{}, err
	}

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

	depLabels := rp2.GetLabels()
	if depLabels == nil {
		depLabels = make(map[string]string)
	}
	depLabels[actionscontroller.RunnerOrgLabelKey] = r.organizationName
	depLabels[actionscontroller.RunnerRepoLabelKey] = rp.Spec.RepositoryName

	podLabels := rp2.Spec.DeploymentSpec.Template.ObjectMeta.Labels
	if podLabels == nil {
		podLabels = make(map[string]string)
	}
	podLabels[actionscontroller.RunnerOrgLabelKey] = r.organizationName
	podLabels[actionscontroller.RunnerRepoLabelKey] = rp.Spec.RepositoryName

	d := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        rp2.Name,
			Namespace:   rp2.Namespace,
			Labels:      depLabels,
			Annotations: rp2.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: rp2.Spec.DeploymentSpec.Replicas,
			Selector: rp2.Spec.DeploymentSpec.Selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: rp2.Spec.DeploymentSpec.Template.ObjectMeta.Annotations,
				},
				Spec: rp2.Spec.DeploymentSpec.Template.Spec,
			},
		},
	}

	var container *corev1.Container
	for i := range d.Spec.Template.Spec.Containers {
		c := &d.Spec.Template.Spec.Containers[i]
		if c.Name == actionscontroller.RunnerContainerName {
			container = c
			break
		}
	}
	if container == nil {
		return nil, fmt.Errorf("container with name %s should exist in the manifest", actionscontroller.RunnerContainerName)
	}

	container.Env = append(container.Env,
		corev1.EnvVar{
			Name: actionscontroller.RunnerNameEnvName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		corev1.EnvVar{
			Name:  actionscontroller.RunnerOrgEnvName,
			Value: r.organizationName,
		},
		corev1.EnvVar{
			Name:  actionscontroller.RunnerRepoEnvName,
			Value: rp.Spec.RepositoryName,
		},
	)
	return &d, nil
}
