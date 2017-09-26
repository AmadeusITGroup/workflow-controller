package v1

import (
	"fmt"
	"sort"

	batchv1 "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/validation"
	v1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateWorkflow validates Workflow
func ValidateWorkflow(workflow *Workflow) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&workflow.ObjectMeta, true, validation.NameIsDNSSubdomain, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateWorkflowSpec(&(workflow.Spec), field.NewPath("spec"))...)
	return allErrs
}

// ValidateWorkflowSpec validates WorkflowSpec
func ValidateWorkflowSpec(spec *WorkflowSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.ActiveDeadlineSeconds != nil {
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(*spec.ActiveDeadlineSeconds), fieldPath.Child("activeDeadlineSeconds"))...)
	}
	if spec.Selector == nil {
		allErrs = append(allErrs, field.Required(fieldPath.Child("selector"), ""))
	} else {
		allErrs = append(allErrs, v1validation.ValidateLabelSelector(spec.Selector, fieldPath.Child("selector"))...)
	}
	// TODO: workflow.spec.selector must be convertible to labels.Set
	// TODO: JobsTemplate must not have any label confliciting with workflow.spec.selector
	allErrs = append(allErrs, ValidateWorkflowSteps(spec.Steps, fieldPath.Child("steps"))...)
	return allErrs
}

func topologicalSort(steps map[string]WorkflowStep, fieldPath *field.Path) ([]string, *field.Error) {
	sorted := make([]string, len(steps))
	temporary := map[string]bool{}
	permanent := map[string]bool{}
	cycle := []string{}
	isCyclic := false
	cycleStart := ""
	var visit func(string) *field.Error
	visit = func(n string) *field.Error {
		if _, found := steps[n]; !found {
			return field.NotFound(fieldPath, n)
		}
		switch {
		case temporary[n]:
			isCyclic = true
			cycleStart = n
			return nil
		case permanent[n]:
			return nil
		}
		temporary[n] = true
		for _, m := range steps[n].Dependencies {
			if err := visit(m); err != nil {
				return err
			}
			if isCyclic {
				if len(cycleStart) != 0 {
					cycle = append(cycle, n)
					if n == cycleStart {
						cycleStart = ""
					}
				}
				return nil
			}
		}
		delete(temporary, n)
		permanent[n] = true
		sorted = append(sorted, n)
		return nil
	}
	var keys []string
	for k := range steps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if permanent[k] {
			continue
		}
		if err := visit(k); err != nil {
			return nil, err
		}
		if isCyclic {
			return nil, field.Forbidden(fieldPath, fmt.Sprintf("detected cycle %s", cycle))
		}
	}
	return sorted, nil
}

// ValidateWorkflowSteps validates steps. It detects cycles (with topological sort) and checks
// whether JobSpec are correct
func ValidateWorkflowSteps(steps []WorkflowStep, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	stepsMap := make(map[string]WorkflowStep, len(steps))
	for i := range steps {
		if _, ok := stepsMap[steps[i].Name]; ok {
			allErrs = append(allErrs, field.Duplicate(fieldPath.Child("steps"), steps[i].Name))
			continue
		}
		stepsMap[steps[i].Name] = steps[i]
	}

	if _, err := topologicalSort(stepsMap, fieldPath); err != nil {
		allErrs = append(allErrs, err)
	}
	for _, step := range steps {
		// TODO: sminonne handle ObjectReference
		if step.JobTemplate == nil && step.ExternalRef == nil {
			allErrs = append(allErrs, field.Required(fieldPath.Child("step"), "either jobTemplate or externalRef must be set")) // TODO: change the field qualifier as soon ObjectRef is handled
			continue
		}
		if step.JobTemplate != nil {
			allErrs = append(allErrs, ValidateJobTemplateSpec(step.JobTemplate, field.NewPath("jobTemplate"))...)
		}
		if step.ExternalRef != nil {
			allErrs = append(allErrs, ValidateExternalReference(step.ExternalRef, field.NewPath("externalRef"))...)
		}
	}
	return allErrs
}

// ValidateExternalReference validates external object reference
func ValidateExternalReference(externalReference *api.ObjectReference, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// TODO: fills It
	return allErrs
}

func getWorkflowUnmodifiableSteps(workflow *Workflow) (running, completed map[string]bool) {
	running = make(map[string]bool)
	completed = make(map[string]bool)
	if workflow.Status.Statuses == nil {
		return
	}
	for _, step := range workflow.Spec.Steps {
		key := step.Name
		stepStatus := GetStepStatusByName(workflow, key)
		if stepStatus != nil {
			if stepStatus.Complete {
				completed[key] = true
			} else {
				running[key] = true
			}
		}
	}
	return
}

// ValidateWorkflowUpdate validates Workflow during update
func ValidateWorkflowUpdate(workflow, oldWorkflow *Workflow) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&workflow.ObjectMeta, &oldWorkflow.ObjectMeta, field.NewPath("metadata"))
	runningSteps, completedSteps := getWorkflowUnmodifiableSteps(oldWorkflow)
	allCompleted := (len(completedSteps) == len(oldWorkflow.Spec.Steps)) // comparing size of maps
	if allCompleted {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("workflow"), "cannot update completed workflow"))
		return allErrs
	}
	allErrs = append(allErrs, ValidateWorkflowSpecUpdate(&workflow.Spec, &oldWorkflow.Spec, runningSteps, completedSteps, field.NewPath("spec"))...)
	return allErrs
}

func ValidateWorkflowUpdateStatus(workflow, oldWorkflow *Workflow) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&oldWorkflow.ObjectMeta, &workflow.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateWorkflowStatusUpdate(workflow.Status, oldWorkflow.Status)...)
	return allErrs
}

func ValidateWorkflowSpecUpdate(spec, oldSpec *WorkflowSpec, running, completed map[string]bool, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateWorkflowSpec(spec, fieldPath)...)
	allErrs = append(allErrs, validation.ValidateImmutableField(spec.Selector, oldSpec.Selector, fieldPath.Child("selector"))...)

	for i, step := range spec.Steps {
		k := step.Name
		if !apiequality.Semantic.DeepEqual(step, oldSpec.Steps[i]) {
			if running[k] {
				allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "steps"), "cannot modify running step \""+k+"\""))
			}
			if completed[k] {
				allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "steps"), "cannot modify completed step \""+k+"\""))
			}
		}
	}

	newSteps := sets.NewString()
	for i := range spec.Steps {
		newSteps.Insert(spec.Steps[i].Name)
	}

	oldSteps := sets.NewString()
	for i := range oldSpec.Steps {
		oldSteps.Insert(oldSpec.Steps[i].Name)
	}

	removedSteps := oldSteps.Difference(newSteps)
	for _, s := range removedSteps.List() {
		if running[s] {
			allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "steps"), "cannot delete running step \""+s+"\""))
			return allErrs
		}
		if completed[s] {
			allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "steps"), "cannot delete completed step \""+s+"\""))
			return allErrs
		}
	}

	return allErrs
}

// ValidateWorkflowStatus validates status
func ValidateWorkflowStatus(status *WorkflowStatus, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	//TODO: @sdminonne add status validation
	return allErrs
}

// ValidateWorkflowStatusUpdate validates WorkflowStatus during update
func ValidateWorkflowStatusUpdate(status, oldStatus WorkflowStatus) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateWorkflowStatus(&status, field.NewPath("status"))...)
	return allErrs
}

func validateJobSpec(spec *batchv1.JobSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.Parallelism != nil {
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(*spec.Parallelism), fldPath.Child("parallelism"))...)
	}

	if spec.Completions != nil {
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(*spec.Completions), fldPath.Child("completions"))...)
	}

	if spec.ActiveDeadlineSeconds != nil {
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(*spec.ActiveDeadlineSeconds), fldPath.Child("activeDeadlineSeconds"))...)
	}

	// TODO: copy&paste ValidatePodTemplateSpec
	//allErrs = append(allErrs, validation.ValidatePodTemplateSpec(&spec.Template, fldPath.Child("template"))...)

	if spec.Template.Spec.RestartPolicy != api.RestartPolicyOnFailure &&
		spec.Template.Spec.RestartPolicy != api.RestartPolicyNever {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("template", "spec", "restartPolicy"),
			spec.Template.Spec.RestartPolicy, []string{string(api.RestartPolicyOnFailure), string(api.RestartPolicyNever)}))
	}

	return allErrs
}

func ValidateJobTemplateSpec(spec *batchv2.JobTemplateSpec, fldPath *field.Path) field.ErrorList {
	allErrs := validateJobSpec(&spec.Spec, fldPath.Child("spec"))

	// jobtemplate will always have the selector automatically generated
	if spec.Spec.Selector != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("spec", "selector"), spec.Spec.Selector, "`selector` will be auto-generated"))
	}
	if spec.Spec.ManualSelector != nil && *spec.Spec.ManualSelector {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("spec", "manualSelector"), spec.Spec.ManualSelector, []string{"nil", "false"}))
	}
	return allErrs
}
