package v1alpha1

import (
	constants "github.com/cybozu-go/meows"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func (r *RunnerPool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups=meows.cybozu.com,resources=runnerpools,verbs=create,versions=v1alpha1,name=runnerpool-hook.meows.cybozu.com,path=/mutate-meows-cybozu-com-v1alpha1-runnerpool,mutating=true,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &RunnerPool{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *RunnerPool) Default() {
	controllerutil.AddFinalizer(r, constants.RunnerPoolFinalizer)
}

// +kubebuilder:webhook:failurePolicy=fail,matchPolicy=equivalent,groups=meows.cybozu.com,resources=runnerpools,verbs=create;update,versions=v1alpha1,name=runnerpool-hook.meows.cybozu.com,path=/validate-meows-cybozu-com-v1alpha1-runnerpool,mutating=false,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &RunnerPool{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *RunnerPool) ValidateCreate() (warnings admission.Warnings, err error) {
	errs := r.Spec.validateCreate()
	if len(errs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "RunnerPool"}, r.Name, errs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *RunnerPool) ValidateUpdate(old runtime.Object) (warnings admission.Warnings, err error) {
	errs := r.Spec.validateUpdate(old.(*RunnerPool).Spec)
	if len(errs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "RunnerPool"}, r.Name, errs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *RunnerPool) ValidateDelete() (warnings admission.Warnings, err error) {
	return nil, nil
}
