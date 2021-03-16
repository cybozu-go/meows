package controllers

import (
	"context"
	"fmt"

	actionscontroller "github.com/cybozu-go/github-actions-controller"
	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
//+kubebuilder:rbac:groups=actions.cybozu.com,resources=runnerpools/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RunnerPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("runnerpool", req.NamespacedName)
	log.Info("start reconciliation loop", "name", req.NamespacedName)

	rp := &actionsv1alpha1.RunnerPool{}
	if err := r.Get(ctx, req.NamespacedName, rp); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("runnerpool is not found", "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to get RunnerPool", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if rp.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(rp, runnerPoolFinalizer) {
			err := r.addFinalizer(ctx, rp)
			if err != nil {
				log.Error(err, "failed to add finalizer", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			// Result does not change even if not requeued here.
			// This just breaks down one reconciliation loop into small steps for simplicity.
			log.Info("added finalizer", "name", req.NamespacedName)
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(rp, runnerPoolFinalizer) {
			err := r.cleanUpOwnedResources(ctx, req.NamespacedName)
			if err != nil {
				log.Error(err, "failed to clean up deployment", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}

			err = r.removeFinalizer(ctx, rp)
			if err != nil {
				log.Error(err, "failed to remove finalizer", "name", req.NamespacedName)
				return ctrl.Result{}, err
			}
			log.Info("removed finalizer", "name", req.NamespacedName)
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
		log.Error(err, "unable to create-or-update Deployment")
		return ctrl.Result{}, err
	}

	log.Info("finished reconciliation successfully", "op", op)
	return ctrl.Result{}, nil
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

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&actionsv1alpha1.RunnerPool{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RunnerPoolReconciler) updateDeploymentWithRunnerPool(rp *actionsv1alpha1.RunnerPool, d *appsv1.Deployment) error {
	rp2 := rp.DeepCopy()

	depLabels := rp2.GetLabels()
	if depLabels == nil {
		depLabels = make(map[string]string)
	}
	depLabels[actionscontroller.RunnerOrgLabelKey] = r.organizationName
	depLabels[actionscontroller.RunnerRepoLabelKey] = rp.Spec.RepositoryName
	d.ObjectMeta.Labels = depLabels

	d.Spec.Replicas = rp2.Spec.Replicas
	d.Spec.Selector = rp2.Spec.Selector
	d.Spec.Strategy = rp2.Spec.Strategy

	podLabels := rp2.Spec.Template.ObjectMeta.Labels
	if podLabels == nil {
		podLabels = make(map[string]string)
	}
	podLabels[actionscontroller.RunnerOrgLabelKey] = r.organizationName
	podLabels[actionscontroller.RunnerRepoLabelKey] = rp.Spec.RepositoryName
	d.Spec.Template.ObjectMeta.Labels = podLabels

	d.Spec.Template.ObjectMeta.Annotations = rp2.Spec.Template.ObjectMeta.Annotations
	d.Spec.Template.Spec = rp2.Spec.Template.Spec

	var container *corev1.Container
	for i := range d.Spec.Template.Spec.Containers {
		c := &d.Spec.Template.Spec.Containers[i]
		if c.Name == actionscontroller.RunnerContainerName {
			container = c
			break
		}
	}
	if container == nil {
		return fmt.Errorf("container with name %s should exist in the manifest", actionscontroller.RunnerContainerName)
	}

	var envMap map[string]struct{}
	for _, v := range container.Env {
		envMap[v.Name] = struct{}{}
	}

	if _, ok := envMap[actionscontroller.RunnerNameEnvName]; !ok {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name: actionscontroller.RunnerNameEnvName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		)
	}
	if _, ok := envMap[actionscontroller.RunnerOrgEnvName]; !ok {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name:  actionscontroller.RunnerOrgEnvName,
				Value: r.organizationName,
			},
		)
	}
	if _, ok := envMap[actionscontroller.RunnerRepoEnvName]; !ok {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name:  actionscontroller.RunnerRepoEnvName,
				Value: rp.Spec.RepositoryName,
			},
		)
	}
	return ctrl.SetControllerReference(rp, d, r.Scheme)
}
