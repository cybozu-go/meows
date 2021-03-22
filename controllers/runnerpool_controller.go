package controllers

import (
	"context"
	"fmt"

	constants "github.com/cybozu-go/github-actions-controller"
	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
//+kubebuilder:rbac:groups=actions.cybozu.com,resources=runnerpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RunnerPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("start reconciliation loop", "name", req.NamespacedName)

	rp := &actionsv1alpha1.RunnerPool{}
	if err := r.Get(ctx, req.NamespacedName, rp); err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("runnerpool is not found")
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "unable to get RunnerPool", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if rp.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(rp, runnerPoolFinalizer) {
			err := r.addFinalizer(ctx, rp)
			if err != nil {
				r.Log.Error(err, "failed to add finalizer", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			// Result does not change even if not requeued here.
			// This just breaks down one reconciliation loop into small steps for simplicity.
			r.Log.Info("added finalizer", "name", req.NamespacedName)
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(rp, runnerPoolFinalizer) {
			err := r.cleanUpOwnedResources(ctx, req.NamespacedName)
			if err != nil {
				r.Log.Error(err, "failed to clean up deployment", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}

			err = r.removeFinalizer(ctx, rp)
			if err != nil {
				r.Log.Error(err, "failed to remove finalizer", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			r.Log.Info("removed finalizer", "name", req.NamespacedName)
		}
		return ctrl.Result{}, nil
	}

	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	}
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, d, func() error {
		return r.updateDeploymentWithRunnerPool(rp, d)
	})
	if err != nil {
		r.Log.Error(err, "unable to create-or-update Deployment", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}
	r.Log.Info("completed create-or-update successfully with status "+string(op), "name", req.NamespacedName)

	rp.Status.Bound = true
	err = r.Status().Update(ctx, rp)
	if err != nil {
		r.Log.Error(err, "failed to update status", "name", req.NamespacedName)
		return ctrl.Result{}, err
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

func (r *RunnerPoolReconciler) addFinalizer(ctx context.Context, rp *actionsv1alpha1.RunnerPool) error {
	rp2 := rp.DeepCopy()
	controllerutil.AddFinalizer(rp2, runnerPoolFinalizer)
	patch := client.MergeFrom(rp)
	return r.Patch(ctx, rp2, patch)
}

func (r *RunnerPoolReconciler) removeFinalizer(ctx context.Context, rp *actionsv1alpha1.RunnerPool) error {
	rp2 := rp.DeepCopy()
	controllerutil.RemoveFinalizer(rp2, runnerPoolFinalizer)
	patch := client.MergeFrom(rp)
	return r.Patch(ctx, rp2, patch)
}

func (r *RunnerPoolReconciler) cleanUpOwnedResources(ctx context.Context, namespacedName types.NamespacedName) error {
	d := &appsv1.Deployment{}
	err := r.Get(ctx, namespacedName, d)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.Delete(ctx, d)
}

func (r *RunnerPoolReconciler) updateDeploymentWithRunnerPool(rp *actionsv1alpha1.RunnerPool, d *appsv1.Deployment) error {
	rp2 := rp.DeepCopy()

	depLabels := rp2.GetLabels()
	if depLabels == nil {
		depLabels = make(map[string]string)
	}
	depLabels[constants.RunnerOrgLabelKey] = r.organizationName
	depLabels[constants.RunnerRepoLabelKey] = rp2.Spec.RepositoryName
	d.ObjectMeta.Labels = depLabels

	d.Spec.Replicas = rp2.Spec.Replicas
	d.Spec.Selector = rp2.Spec.Selector
	d.Spec.Strategy = rp2.Spec.Strategy

	podLabels := rp2.Spec.Template.ObjectMeta.Labels
	if podLabels == nil {
		podLabels = make(map[string]string)
	}
	podLabels[constants.RunnerOrgLabelKey] = r.organizationName
	podLabels[constants.RunnerRepoLabelKey] = rp2.Spec.RepositoryName
	d.Spec.Template.ObjectMeta.Labels = podLabels

	d.Spec.Template.ObjectMeta.Annotations = rp2.Spec.Template.ObjectMeta.Annotations

	var container *corev1.Container
	for i := range rp2.Spec.Template.Spec.Containers {
		c := &rp2.Spec.Template.Spec.Containers[i]
		if c.Name == constants.RunnerContainerName {
			container = c
			break
		}
	}
	if container == nil {
		return fmt.Errorf("container with name %s should exist in the manifest", constants.RunnerContainerName)
	}

	envMap := make(map[string]struct{})
	for _, v := range container.Env {
		envMap[v.Name] = struct{}{}
	}

	if _, ok := envMap[constants.RunnerNameEnvName]; !ok {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name: constants.RunnerNameEnvName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		)
	}
	if _, ok := envMap[constants.RunnerOrgEnvName]; !ok {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name:  constants.RunnerOrgEnvName,
				Value: r.organizationName,
			},
		)
	}
	if _, ok := envMap[constants.RunnerRepoEnvName]; !ok {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name:  constants.RunnerRepoEnvName,
				Value: rp.Spec.RepositoryName,
			},
		)
	}

	if !equality.Semantic.DeepDerivative(&rp2.Spec.Template.Spec, &d.Spec.Template.Spec) {
		d.Spec.Template.Spec = rp2.Spec.Template.Spec
	}
	return ctrl.SetControllerReference(rp, d, r.Scheme)
}
