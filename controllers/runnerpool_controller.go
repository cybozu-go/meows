package controllers

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	actionsv1alpha1 "github.com/cybozu-go/github-actions-controller/api/v1alpha1"
	"github.com/cybozu-go/github-actions-controller/github"
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
	Log          logr.Logger
	Scheme       *runtime.Scheme
	githubClient github.RegistrationTokenGenerator
}

// NewRunnerPoolReconciler creates RunnerPoolReconciler
func NewRunnerPoolReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme, githubClient github.RegistrationTokenGenerator) *RunnerPoolReconciler {
	return &RunnerPoolReconciler{
		Client:       client,
		Log:          log,
		Scheme:       scheme,
		githubClient: githubClient,
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

	targetSecret := &corev1.Secret{}
	err := r.Get(ctx, req.NamespacedName, targetSecret)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "failed to get secret")
			return ctrl.Result{}, err
		}
		// TODO
		_, err := r.githubClient.CreateOrganizationRegistrationToken(ctx)
		if err != nil {
			log.Error(err, "failed to generate token")
			return ctrl.Result{}, err
		}
	}

	d, err := makeDeployment(rp)
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
