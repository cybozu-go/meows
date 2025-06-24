package v1alpha1

import (
	"context"
	"fmt"

	constants "github.com/cybozu-go/meows"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&RunnerPool{}).
		WithDefaulter(&RunnerPoolDefaulter{}).
		WithValidator(&RunnerPoolValidator{}).
		Complete()
}

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups=meows.cybozu.com,resources=runnerpools,verbs=create,versions=v1alpha1,name=runnerpool-hook.meows.cybozu.com,path=/mutate-meows-cybozu-com-v1alpha1-runnerpool,mutating=true,sideEffects=none,admissionReviewVersions=v1

type RunnerPoolDefaulter struct{}

var _ webhook.CustomDefaulter = &RunnerPoolDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *RunnerPoolDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	rp, ok := obj.(*RunnerPool)
	if !ok {
		return fmt.Errorf("expected a RunnerPool object but got %T", rp)
	}

	controllerutil.AddFinalizer(rp, constants.RunnerPoolFinalizer)
	return nil
}

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups=meows.cybozu.com,resources=runnerpools,verbs=create;update,versions=v1alpha1,name=runnerpool-hook.meows.cybozu.com,path=/validate-meows-cybozu-com-v1alpha1-runnerpool,mutating=false,sideEffects=none,admissionReviewVersions=v1

type RunnerPoolValidator struct{}

var _ webhook.CustomValidator = &RunnerPoolValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *RunnerPoolValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	rp, ok := obj.(*RunnerPool)
	if !ok {
		return nil, fmt.Errorf("expected a RunnerPool object but got %T", rp)
	}

	errs := rp.Spec.validateCreate()
	if len(errs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "RunnerPool"}, rp.Name, errs)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *RunnerPoolValidator) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) (warnings admission.Warnings, err error) {
	oldRp, ok := oldObj.(*RunnerPool)
	if !ok {
		return nil, fmt.Errorf("expected a RunnerPool object for the oldRp but got %T", oldRp)
	}
	newRp, ok := newObj.(*RunnerPool)
	if !ok {
		return nil, fmt.Errorf("expected a RunnerPool object for the newRp but got %T", newRp)
	}

	errs := newRp.Spec.validateUpdate(oldRp.Spec)
	if len(errs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "RunnerPool"}, newRp.Name, errs)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *RunnerPoolValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}
