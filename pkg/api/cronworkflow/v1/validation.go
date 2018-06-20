package v1

import (
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateCronWorkflow validates Workflow
func ValidateCronWorkflow(workflow *CronWorkflow) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&workflow.ObjectMeta, true, validation.NameIsDNSSubdomain, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateCronWorkflowSpec(&(workflow.Spec), field.NewPath("spec"))...)
	return allErrs
}

// ValidateCronWorkflowSpec validates WorkflowSpec
func ValidateCronWorkflowSpec(spec *CronWorkflowSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateExternalReference validates external object reference
func ValidateExternalReference(externalReference *api.ObjectReference, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// TODO: fills It
	return allErrs
}

// ValidateWorkflowUpdate validates Workflow during update
func ValidateWorkflowUpdate(workflow, oldWorkflow *CronWorkflow) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&workflow.ObjectMeta, &oldWorkflow.ObjectMeta, field.NewPath("metadata"))
	return allErrs
}

// ValidateCronWorkflowUpdateStatus valides CronWorkflow status update
func ValidateCronWorkflowUpdateStatus(workflow, oldWorkflow *CronWorkflow) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&oldWorkflow.ObjectMeta, &workflow.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateWorkflowStatusUpdate(workflow.Status, oldWorkflow.Status)...)
	return allErrs
}

// ValidateCronWorkflowSpecUpdate validates CronWorkfow spec update
func ValidateCronWorkflowSpecUpdate(spec, oldSpec *CronWorkflowSpec, running, completed map[string]bool, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateCronWorkflowSpec(spec, fieldPath)...)
	return allErrs
}

// ValidateWorkflowStatus validates status
func ValidateWorkflowStatus(status *CronWorkflowStatus, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	//TODO: @sdminonne add status validation
	return allErrs
}

// ValidateWorkflowStatusUpdate validates WorkflowStatus during update
func ValidateWorkflowStatusUpdate(status, oldStatus CronWorkflowStatus) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateWorkflowStatus(&status, field.NewPath("status"))...)
	return allErrs
}
