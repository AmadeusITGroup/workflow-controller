/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package v1

import (
	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronWorkflow represents a Cron Workflow
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CronWorkflow struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behaviour of the Workflow.
	Spec CronWorkflowSpec `json:"spec,omitempty"`

	// Status contains the current status off the Workflow
	Status CronWorkflowStatus `json:"status,omitempty"`
}

// CronWorkflowList implements list of CronWorkflow.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CronWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Workflow
	Items []CronWorkflow `json:"items"`
}

// CronWorkflowSpec contains a cron Workflow specification
type CronWorkflowSpec struct {
	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`

	// The anti schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	AntiSchedule string `json:"antischedule,omitempty"`

	// Optional deadline in seconds for starting the workflow if it misses scheduled
	// time for any reason.  Missed workflows executions will be counted as failed ones.
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// Specifies how to treat concurrent executions of a Workflow.
	// Defaults to Allow.
	ConcurrencyPolicy batchv2alpha1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	Suspend *bool `json:"suspend,omitempty"`

	// Specifies the job that will be created when executing a CronJob.
	WorkflowTemplate WorkflowTemplateSpec `json:"workflowTemplate"`

	// The number of successful finished workflows to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	SuccessfulWorkflowsHistoryLimit *int32 `json:"successfulWorkflowsHistoryLimit,omitempty"`

	// The number of failed finished workflows to retain.
	// This is a pointer to distinguish between explicit zero and not specified.
	// +optional
	FailedWorkflowsHistoryLimit *int32 `json:"failedWorkflowsHistoryLimit,omitempty"`
}

// CronWorkflowStatus contains the status of a cron Workflow
type CronWorkflowStatus struct {
	// A list of pointers to currently running workflows.
	Active []v1.ObjectReference `json:"active,omitempty"`

	// Information when was the last time the job was successfully scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

// WorkflowTemplateSpec represents a WorkflowTemplate
type WorkflowTemplateSpec struct {
	// Standard object's metadata of the workflows created from this template.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the workflow.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status
	// +optional
	Spec workflowv1.WorkflowSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}
