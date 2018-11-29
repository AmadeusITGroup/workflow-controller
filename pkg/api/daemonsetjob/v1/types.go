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

package v1

import (
	batch "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DaemonSetJob represents a DaemonSet Job
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DaemonSetJob struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behaviour of the DaemonSetJob.
	Spec DaemonSetJobSpec `json:"spec,omitempty"`

	// Status contains the current status off the DaemonSetJob
	Status DaemonSetJobStatus `json:"status,omitempty"`
}

// DaemonSetJobList implements list of DaemonSetJob.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DaemonSetJobList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of DaemonSetJob
	Items []DaemonSetJob `json:"items"`
}

// DaemonSetJobSpec contains DaemonSetJob specification
type DaemonSetJobSpec struct {
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`

	// JobTemplate contains the job specificaton that should be run in this Workflow.
	// Only one between externalRef and JobTemplate can be set.
	JobTemplate *batch.JobTemplateSpec `json:"jobTemplate,omitempty"`

	// NodeSelector use to create DaemonSetJobs only on selected nodes
	NodeSelector *metav1.LabelSelector `json:"nodeSelector,omitempty"`

	// Selector for created jobs (if any)
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// DaemonSetJobStatus represents the status of DaemonSetJob
type DaemonSetJobStatus struct {
	// Conditions represent the latest available observations of an object's current state.
	Conditions []DaemonSetJobCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// StartTime represents time when the workflow was acknowledged by the DaemonSetJob controller
	// It is not guaranteed to be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	// StartTime doesn't consider startime of `ExternalReference`
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime represents time when the workflow was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// The number of actively running jobs.
	// +optional
	Active int32 `json:"active,omitempty" protobuf:"varint,4,opt,name=active"`

	// The number of jobs which reached phase Succeeded.
	// +optional
	Succeeded int32 `json:"succeeded,omitempty" protobuf:"varint,5,opt,name=succeeded"`

	// The number of jobs which reached phase Failed.
	// +optional
	Failed int32 `json:"failed,omitempty" protobuf:"varint,6,opt,name=failed"`
}

// DaemonSetJobNodeJobStatus contains necessary information for the step status
type DaemonSetJobNodeJobStatus struct {
	// Name represents the Name of the Step
	Name string `json:"name,omitempty"`
	// Complete reports the completion of status`
	Complete bool `json:"complete"`
	// Reference contains a reference to the DaemonSetJob JobNode
	Reference api.ObjectReference `json:"reference"`
}

// DaemonSetJobCondition represent the condition of the DaemonSetJob
type DaemonSetJobCondition struct {
	// Type of workflow condition
	Type DaemonSetJobConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status api.ConditionStatus `json:"status"`
	// Last time the condition was checked.
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transited from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
}

// DaemonSetJobConditionType is the type of DaemonSetJobCondition
type DaemonSetJobConditionType string

// These are valid conditions of a workflow.
const (
	// DaemonSetJobComplete means the workflow has completed its execution.
	DaemonSetJobComplete DaemonSetJobConditionType = "Complete"
	// DaemonSetJobFailed means the workflow has failed its execution.
	DaemonSetJobFailed DaemonSetJobConditionType = "Failed"
)
