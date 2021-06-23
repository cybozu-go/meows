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
	"k8s.io/apimachinery/pkg/runtime"
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
func NewRunnerPoolReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme, repositoryNames []string, organizationName string) *RunnerPoolReconciler {
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
		if !controllerutil.ContainsFinalizer(rp, constants.RunnerPoolFinalizer) {
			return ctrl.Result{}, nil
		}

		log.Info("start finalizing RunnerPool")

		if err := r.finalize(ctx, log, rp); err != nil {
			log.Error(err, "failed to finalize")
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(rp, constants.RunnerPoolFinalizer)
		if err := r.Update(ctx, rp); err != nil {
			log.Error(err, "failed to remove finalizer")
			return ctrl.Result{}, err
		}

		log.Info("finalizing RunnerPool is completed")
		return ctrl.Result{}, nil
	}

	if err := r.validateRepositoryName(rp); err != nil {
		log.Error(err, "failed to validate repository name")
		return ctrl.Result{}, err
	}

	if err := r.reconcileDeployment(ctx, log, rp); err != nil {
		log.Error(err, "failed to reconcile deployment")
		return ctrl.Result{}, err
	}

	rp.Status.Bound = true
	if err := r.Status().Update(ctx, rp); err != nil {
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

func (r *RunnerPoolReconciler) validateRepositoryName(rp *actionsv1alpha1.RunnerPool) error {
	for _, n := range r.repositoryNames {
		if n == rp.Spec.RepositoryName {
			return nil
		}
	}
	return fmt.Errorf("found the invalid repository name %v. Valid repository names are %v", rp.Spec.RepositoryName, r.repositoryNames)
}

func (r *RunnerPoolReconciler) finalize(ctx context.Context, log logr.Logger, rp *actionsv1alpha1.RunnerPool) error {
	d := &appsv1.Deployment{}
	d.SetNamespace(rp.GetNamespace())
	d.SetName(rp.GetRunnerDeploymentName())
	if err := r.Delete(ctx, d); err != nil {
		log.Error(err, "failed to delete deployment")
		return err
	}
	return nil
}

func (r *RunnerPoolReconciler) reconcileDeployment(ctx context.Context, log logr.Logger, rp *actionsv1alpha1.RunnerPool) error {
	d := &appsv1.Deployment{}
	d.SetNamespace(rp.GetNamespace())
	d.SetName(rp.GetRunnerDeploymentName())
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, d, func() error {
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
			// NOTE:This error should not occur.
			// Because the existence of the runner container is validated by an admission webhook.
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
					Value: rp.Spec.RepositoryName,
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
					Value: *rp.Spec.SlackAgentServiceName,
				},
			)
		}

		if !equality.Semantic.DeepDerivative(&rp2.Spec.Template.Spec, &d.Spec.Template.Spec) {
			d.Spec.Template.Spec = rp2.Spec.Template.Spec
		}
		return ctrl.SetControllerReference(rp, d, r.Scheme)
	})

	if err != nil {
		log.Error(err, "failed to reconcile deployment")
		return err
	}
	if op != controllerutil.OperationResultNone {
		log.Info("reconciled stateful set", "operation", string(op))
	}

	return nil
}
