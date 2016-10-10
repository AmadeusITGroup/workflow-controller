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

package api

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"
)

// Workflow represents a DAG workflow
type Workflow struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	api.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behaviour of the Workflow.
	Spec WorkflowSpec `json:"spec,omitempty"`

	// Status contains the current status of the Workflow
	Status WorkflowStatus `json:"status,omitempty"`
}

// WorkflowList implements list of Workflow.
type WorkflowList struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	unversioned.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Workflow
	Items []Workflow `json:"items"`
}

// WorkflowSpec contains Workflow specification
type WorkflowSpec struct {
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`

	Steps []WorkflowStep `json:"steps,omitempty"`

	// Selector for created jobs (if any)
	Selector *unversioned.LabelSelector `json:"selector,omitempty"`
}

// WorkflowStep contains necessary information to identifiy the node of the workflow graph
type WorkflowStep struct {
	// Name of the step
	Name string `json:"name,omitempty"`

	// JobTemplate contains the job specificaton that should be run in this Workflow.
	// Only one between externalRef and JobTemplate can be set.
	JobTemplate *batch.JobTemplateSpec `json:"jobTemplate,omitempty"`

	// External contains a reference to another schedulable resource.
	// Only one between ExternalRef and JobTemplate can be set.
	ExternalRef *api.ObjectReference `json:"externalRef,omitempty"`

	// Dependecies represent dependecies of the current workflow step
	Dependencies []string `json:"dependencies,omitempty"`
}

// WorkflowConditionType is the type of WorkflowCondition
type WorkflowConditionType string

// These are valid conditions of a workflow.
const (
	// WorkflowComplete means the workflow has completed its execution.
	WorkflowComplete WorkflowConditionType = "Complete"
	// WorkflowFailed means the workflow has failed its execution.
	WorkflowFailed WorkflowConditionType = "Failed"
)

// WorkflowCondition represent the condition of the Workflow
type WorkflowCondition struct {
	// Type of workflow condition
	Type WorkflowConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status api.ConditionStatus `json:"status"`
	// Last time the condition was checked.
	LastProbeTime unversioned.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transited from one status to another.
	LastTransitionTime unversioned.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
}

// WorkflowStatus represents the status of Workflow
type WorkflowStatus struct {
	// Conditions represent the latest available observations of an object's current state.
	Conditions []WorkflowCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// StartTime represents time when the workflow was acknowledged by the Workflow controller
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	// StartTime doesn't consider startime of `ExternalReference`
	StartTime *unversioned.Time `json:"startTime,omitempty"`

	// CompletionTime represents time when the workflow was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *unversioned.Time `json:"completionTime,omitempty"`

	// Statuses represent status of different steps
	Statuses []WorkflowStepStatus `json:"statuses"`
}

// WorkflowStepStatus contains necessary information for the step status
type WorkflowStepStatus struct {
	// Name represents the Name of the Step
	Name string `json:"name,omitempty"`
	// Complete reports the completion of status`
	Complete bool `json:"complete"`
	// Reference contains a reference to the WorkflowStep
	Reference api.ObjectReference `json:"reference"`
}

//GetObjectKind implements minimum req fro runtimeObject
func (obj *Workflow) GetObjectKind() unversioned.ObjectKind { return &obj.TypeMeta }

//GetObjectKind implements minimum req fro runtimeObject
func (obj *WorkflowList) GetObjectKind() unversioned.ObjectKind { return &obj.TypeMeta }
