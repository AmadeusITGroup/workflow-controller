package v1

import (
	"k8s.io/apimachinery/pkg/api/validation"
	v1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateDaemonSetJob validates DaemonSetJob
func ValidateDaemonSetJob(daemonsetjob *DaemonSetJob) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&daemonsetjob.ObjectMeta, true, validation.NameIsDNSSubdomain, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateDaemonSetJobSpec(&(daemonsetjob.Spec), field.NewPath("spec"))...)
	return allErrs
}

// ValidateDaemonSetJobSpec validates DaemonSetJobSpec
func ValidateDaemonSetJobSpec(spec *DaemonSetJobSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.ActiveDeadlineSeconds != nil {
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(*spec.ActiveDeadlineSeconds), fieldPath.Child("activeDeadlineSeconds"))...)
	}
	if spec.Selector == nil {
		allErrs = append(allErrs, field.Required(fieldPath.Child("selector"), ""))
	} else {
		allErrs = append(allErrs, v1validation.ValidateLabelSelector(spec.Selector, fieldPath.Child("selector"))...)
	}
	// TODO: daemonsetjob.spec.selector must be convertible to labels.Set
	// TODO: JobsTemplate must not have any label confliciting with daemonsetjob.spec.selector
	return allErrs
}
