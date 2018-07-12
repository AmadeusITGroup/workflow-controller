package validation

import (
	"github.com/robfig/cron"

	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	cronworkflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
)

func validateScheduleFormat(schedule string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := cron.ParseStandard(schedule); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, schedule, err.Error()))
	}
	return allErrs
}

// ValidateCronWorkflow validates Workflow
func ValidateCronWorkflow(workflow *cronworkflowv1.CronWorkflow) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&workflow.ObjectMeta, true, validation.NameIsDNSSubdomain, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateCronWorkflowSpec(&(workflow.Spec), field.NewPath("spec"))...)
	return allErrs
}

func validateConcurrencyPolicy(concurrencyPolicy *batchv2alpha1.ConcurrencyPolicy, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	switch *concurrencyPolicy {
	case batchv2alpha1.AllowConcurrent, batchv2alpha1.ForbidConcurrent, batchv2alpha1.ReplaceConcurrent:
		break
	case "":
		allErrs = append(allErrs, field.Required(fldPath, ""))
	default:
		validValues := []string{string(batchv2alpha1.AllowConcurrent), string(batchv2alpha1.ForbidConcurrent), string(batchv2alpha1.ReplaceConcurrent)}
		allErrs = append(allErrs, field.NotSupported(fldPath, *concurrencyPolicy, validValues))
	}

	return allErrs
}

// ValidateCronWorkflowSpec validates WorkflowSpec
func ValidateCronWorkflowSpec(spec *cronworkflowv1.CronWorkflowSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateScheduleFormat(spec.Schedule, fieldPath.Child("schedule"))...)
	if spec.StartingDeadlineSeconds != nil {
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(*spec.StartingDeadlineSeconds), fieldPath.Child("startingDeadlineSeconds"))...)
	}
	allErrs = append(allErrs, validateConcurrencyPolicy(&spec.ConcurrencyPolicy, fieldPath.Child("concurrencyPolicy"))...)
	if spec.SuccessfulWorkflowsHistoryLimit != nil {
		// zero is a valid SuccessfulWorkflowsHistoryLimit
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(*spec.SuccessfulWorkflowsHistoryLimit), fieldPath.Child("successfulWorkflowsHistoryLimit"))...)
	}
	if spec.FailedWorkflowsHistoryLimit != nil {
		// zero is a valid FailedWorkflowsHistoryLimit
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(*spec.FailedWorkflowsHistoryLimit), fieldPath.Child("failedWorkflowsHistoryLimit"))...)
	}
	return allErrs
}

// ValidateCronWorkflowUpdate validates Workflow during update
func ValidateCronWorkflowUpdate(workflow, oldWorkflow *cronworkflowv1.CronWorkflow) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&workflow.ObjectMeta, &oldWorkflow.ObjectMeta, field.NewPath("metadata"))
	return allErrs
}

// ValidateCronWorkflowUpdateStatus valides CronWorkflow status update
func ValidateCronWorkflowUpdateStatus(workflow, oldWorkflow *cronworkflowv1.CronWorkflow) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&oldWorkflow.ObjectMeta, &workflow.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateCronWorkflowStatusUpdate(workflow.Status, oldWorkflow.Status)...)
	return allErrs
}

// ValidateCronWorkflowSpecUpdate validates CronWorkfow spec update
func ValidateCronWorkflowSpecUpdate(spec, oldSpec *cronworkflowv1.CronWorkflowSpec, running, completed map[string]bool, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateCronWorkflowSpec(spec, fieldPath)...)
	return allErrs
}

// ValidateCronWorkflowStatus validates status
func ValidateCronWorkflowStatus(status *cronworkflowv1.CronWorkflowStatus, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	//TODO: @sdminonne add status validation
	return allErrs
}

// ValidateCronWorkflowStatusUpdate validates WorkflowStatus during update
func ValidateCronWorkflowStatusUpdate(status, oldStatus cronworkflowv1.CronWorkflowStatus) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateCronWorkflowStatus(&status, field.NewPath("status"))...)
	return allErrs
}
