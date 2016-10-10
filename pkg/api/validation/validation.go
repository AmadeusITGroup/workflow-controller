/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"fmt"
	"sort"

	kapi "k8s.io/kubernetes/pkg/api"
	unversionedvalidation "k8s.io/kubernetes/pkg/api/unversioned/validation"
	kapivalidation "k8s.io/kubernetes/pkg/api/validation"
	batchvalidation "k8s.io/kubernetes/pkg/apis/batch/validation"
	"k8s.io/kubernetes/pkg/util/sets"
	"k8s.io/kubernetes/pkg/util/validation/field"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	wapi "github.com/sdminonne/workflow-controller/pkg/api"
)

// ValidateWorkflow validates a workflow
func ValidateWorkflow(workflow *wapi.Workflow) field.ErrorList {
	s := spew.ConfigState{
		Indent: " ",
		// Extra deep spew.
		DisableMethods: true,
	}
	glog.V(6).Infof("Workflow %s\n", s.Sdump(workflow))

	allErrs := kapivalidation.ValidateObjectMeta(&workflow.ObjectMeta, true, kapivalidation.ValidateReplicationControllerName, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateWorkflowSpec(&(workflow.Spec), field.NewPath("spec"))...)
	return allErrs
}

// ValidateWorkflowSpec validate WorkflowSpec
func ValidateWorkflowSpec(spec *wapi.WorkflowSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.ActiveDeadlineSeconds != nil {
		allErrs = append(allErrs, kapivalidation.ValidateNonnegativeField(int64(*spec.ActiveDeadlineSeconds), fieldPath.Child("activeDeadlineSeconds"))...)
	}
	if spec.Selector == nil {
		allErrs = append(allErrs, field.Required(fieldPath.Child("selector"), ""))
	} else {
		allErrs = append(allErrs, unversionedvalidation.ValidateLabelSelector(spec.Selector, fieldPath.Child("selector"))...)
	}
	allErrs = append(allErrs, ValidateWorkflowSteps(spec.Steps, fieldPath.Child("steps"))...)
	return allErrs
}

func topologicalSort(steps map[string]wapi.WorkflowStep, fieldPath *field.Path) ([]string, *field.Error) {
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
func ValidateWorkflowSteps(steps []wapi.WorkflowStep, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	stepsMap := make(map[string]wapi.WorkflowStep, len(steps))
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
			allErrs = append(allErrs, batchvalidation.ValidateJobTemplateSpec(step.JobTemplate, field.NewPath("jobTemplate"))...)
		}
		if step.ExternalRef != nil {
			allErrs = append(allErrs, ValidateExternalReference(step.ExternalRef, field.NewPath("externalRef"))...)
		}
	}
	return allErrs
}

// ValidateExternalReference validates external object reference
func ValidateExternalReference(externalReference *kapi.ObjectReference, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// TODO: fills It
	return allErrs
}

// ValidateWorkflowStatus validates statuses update
func ValidateWorkflowStatus(status *wapi.WorkflowStatus, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	glog.V(6).Infof("Validating WorkflowStatus %v\n", spew.Sdump(*status))
	//TODO: @sdminonne add status validation
	return allErrs
}

func getWorkflowUnmodifiableSteps(workflow *wapi.Workflow) (running, completed map[string]bool) {
	running = make(map[string]bool)
	completed = make(map[string]bool)
	if workflow.Status.Statuses == nil {
		return
	}
	for _, step := range workflow.Spec.Steps {
		key := step.Name
		stepStatus := workflow.GetStepStatusByName(key)
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
func ValidateWorkflowUpdate(workflow, oldWorkflow *wapi.Workflow) field.ErrorList {
	allErrs := kapivalidation.ValidateObjectMetaUpdate(&workflow.ObjectMeta, &oldWorkflow.ObjectMeta, field.NewPath("metadata"))
	runningSteps, completedSteps := getWorkflowUnmodifiableSteps(oldWorkflow)
	allCompleted := (len(completedSteps) == len(oldWorkflow.Spec.Steps)) // comparing size of maps
	if allCompleted {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("workflow"), "cannot update completed workflow"))
		return allErrs
	}
	allErrs = append(allErrs, ValidateWorkflowSpecUpdate(&workflow.Spec, &oldWorkflow.Spec, runningSteps, completedSteps, field.NewPath("spec"))...)
	return allErrs
}

func ValidateWorkflowUpdateStatus(workflow, oldWorkflow *wapi.Workflow) field.ErrorList {
	allErrs := kapivalidation.ValidateObjectMetaUpdate(&oldWorkflow.ObjectMeta, &workflow.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateWorkflowStatusUpdate(workflow.Status, oldWorkflow.Status)...)
	return allErrs
}

func ValidateWorkflowSpecUpdate(spec, oldSpec *wapi.WorkflowSpec, running, completed map[string]bool, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateWorkflowSpec(spec, fieldPath)...)
	allErrs = append(allErrs, kapivalidation.ValidateImmutableField(spec.Selector, oldSpec.Selector, fieldPath.Child("selector"))...)

	for i, step := range spec.Steps {
		k := step.Name
		if !kapi.Semantic.DeepEqual(step, oldSpec.Steps[i]) {
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
		//spec.Steps[i]newSteps = append(newSteps, spec.Steps[i])
		newSteps.Insert(spec.Steps[i].Name)
	}

	oldSteps := sets.NewString()
	for i := range oldSpec.Steps {
		//oldSteps = append(oldSteps, oldSpec.Steps[i])
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

func ValidateWorkflowStatusUpdate(status, oldStatus wapi.WorkflowStatus) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateWorkflowStatus(&status, field.NewPath("status"))...)
	return allErrs
}
