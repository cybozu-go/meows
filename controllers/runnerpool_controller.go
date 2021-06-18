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

// RunnerPoolReconciler reconciles a RunnerPool object
type RunnerPoolReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	repositoryNames  []string
	organizationName string
}

// NewRunnerPoolReconciler creates RunnerPoolReconciler
func NewRunnerPoolReconciler(
	client client.Client,
	log logr.Logger,
	scheme *runtime.Scheme,

	repositoryNames []string,
	organizationName string,
) *RunnerPoolReconciler {
	return &RunnerPoolReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,

		repositoryNames:  repositoryNames,
		organizationName: organizationName,
	}
}

//+kubebuilder:rbac:groups=actions.cybozu.com,resources=runnerpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=actions.cybozu.com,resources=runnerpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RunnerPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("name", req.NamespacedName)
	log.Info("start reconciliation loop")

	rp := &actionsv1alpha1.RunnerPool{}
	if err := r.Get(ctx, req.NamespacedName, rp); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("runnerpool is not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to get RunnerPool")
		return ctrl.Result{}, err
	}

	if rp.ObjectMeta.DeletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(rp, constants.RunnerPoolFinalizer) {
			err := r.cleanUpOwnedResources(ctx, req.NamespacedName)
			if err != nil {
				log.Error(err, "failed to clean up deployment")
				return ctrl.Result{}, err
			}

			err = r.removeFinalizer(ctx, rp)
			if err != nil {
				log.Error(err, "failed to remove finalizer")
				return ctrl.Result{}, err
			}
			log.Info("removed finalizer")
		}
		return ctrl.Result{}, nil
	}

	found := false
	for _, n := range r.repositoryNames {
		if n == rp.Spec.RepositoryName {
			found = true
			break
		}
	}
	if !found {
		return ctrl.Result{}, fmt.Errorf("found the invalid repository name %v. Valid repository names are %v", rp.Spec.RepositoryName, r.repositoryNames)
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
	log.Info("completed create-or-update successfully", "result", op)

	rp.Status.Bound = true
	err = r.Status().Update(ctx, rp)
	if err != nil {
		log.Error(err, "failed to update status")
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

func (r *RunnerPoolReconciler) removeFinalizer(ctx context.Context, rp *actionsv1alpha1.RunnerPool) error {
	rp2 := rp.DeepCopy()
	controllerutil.RemoveFinalizer(rp2, constants.RunnerPoolFinalizer)
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

	if _, ok := envMap[constants.PodNameEnvName]; !ok {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name: constants.PodNameEnvName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		)
	}
	if _, ok := envMap[constants.PodNamespaceEnvName]; !ok {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name: constants.PodNamespaceEnvName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
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
				Value: rp2.Spec.RepositoryName,
			},
		)
	}
	if _, ok := envMap[constants.RunnerPoolNameEnvName]; !ok {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name:  constants.RunnerPoolNameEnvName,
				Value: rp2.ObjectMeta.Name,
			},
		)
	}
	if _, ok := envMap[constants.SlackAgentEnvName]; !ok && rp2.Spec.SlackAgentServiceName != nil {
		container.Env = append(
			container.Env,
			corev1.EnvVar{
				Name:  constants.SlackAgentEnvName,
				Value: *rp2.Spec.SlackAgentServiceName,
			},
		)
	}

	if !equality.Semantic.DeepDerivative(&rp2.Spec.Template.Spec, &d.Spec.Template.Spec) {
		d.Spec.Template.Spec = rp2.Spec.Template.Spec
	}
	return ctrl.SetControllerReference(rp, d, r.Scheme)
}
