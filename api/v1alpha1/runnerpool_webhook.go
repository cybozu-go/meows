package v1alpha1

import (
	"context"

	constants "github.com/cybozu-go/meows"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &RunnerPool{}).
		WithDefaulter(&RunnerPoolDefaulter{}).
		WithValidator(&RunnerPoolValidator{}).
		Complete()
}

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups=meows.cybozu.com,resources=runnerpools,verbs=create,versions=v1alpha1,name=runnerpool-hook.meows.cybozu.com,path=/mutate-meows-cybozu-com-v1alpha1-runnerpool,mutating=true,sideEffects=none,admissionReviewVersions=v1

type RunnerPoolDefaulter struct{}

// Default implements admission.Defaulter so a webhook will be registered for the type
func (r *RunnerPoolDefaulter) Default(ctx context.Context, rp *RunnerPool) error {
	controllerutil.AddFinalizer(rp, constants.RunnerPoolFinalizer)
	return nil
}

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups=meows.cybozu.com,resources=runnerpools,verbs=create;update,versions=v1alpha1,name=runnerpool-hook.meows.cybozu.com,path=/validate-meows-cybozu-com-v1alpha1-runnerpool,mutating=false,sideEffects=none,admissionReviewVersions=v1

type RunnerPoolValidator struct{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (r *RunnerPoolValidator) ValidateCreate(ctx context.Context, rp *RunnerPool) (warnings admission.Warnings, err error) {
	errs := rp.Spec.validateCreate()
	if len(errs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "RunnerPool"}, rp.Name, errs)
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (r *RunnerPoolValidator) ValidateUpdate(ctx context.Context, oldRp *RunnerPool, newRp *RunnerPool) (warnings admission.Warnings, err error) {
	errs := newRp.Spec.validateUpdate(oldRp.Spec)
	if len(errs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "RunnerPool"}, newRp.Name, errs)
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (r *RunnerPoolValidator) ValidateDelete(ctx context.Context, rp *RunnerPool) (warnings admission.Warnings, err error) {
	return nil, nil
}
