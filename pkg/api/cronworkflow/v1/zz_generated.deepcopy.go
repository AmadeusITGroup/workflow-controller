// +build !ignore_autogenerated

/*
   Copyright 2018 Amadeus SAS

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

// Generated file, do not modify manually!

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1

import (
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
	reflect "reflect"
)

// GetGeneratedDeepCopyFuncs returns the generated funcs, since we aren't registering them.
//
// Deprecated: deepcopy registration will go away when static deepcopy is fully implemented.
func GetGeneratedDeepCopyFuncs() []conversion.GeneratedDeepCopyFunc {
	return []conversion.GeneratedDeepCopyFunc{
		{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*CronWorkflow).DeepCopyInto(out.(*CronWorkflow))
			return nil
		}, InType: reflect.TypeOf(&CronWorkflow{})},
		{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*CronWorkflowList).DeepCopyInto(out.(*CronWorkflowList))
			return nil
		}, InType: reflect.TypeOf(&CronWorkflowList{})},
		{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*CronWorkflowSpec).DeepCopyInto(out.(*CronWorkflowSpec))
			return nil
		}, InType: reflect.TypeOf(&CronWorkflowSpec{})},
		{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*CronWorkflowStatus).DeepCopyInto(out.(*CronWorkflowStatus))
			return nil
		}, InType: reflect.TypeOf(&CronWorkflowStatus{})},
		{Fn: func(in interface{}, out interface{}, c *conversion.Cloner) error {
			in.(*WorkflowTemplateSpec).DeepCopyInto(out.(*WorkflowTemplateSpec))
			return nil
		}, InType: reflect.TypeOf(&WorkflowTemplateSpec{})},
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronWorkflow) DeepCopyInto(out *CronWorkflow) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronWorkflow.
func (in *CronWorkflow) DeepCopy() *CronWorkflow {
	if in == nil {
		return nil
	}
	out := new(CronWorkflow)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CronWorkflow) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronWorkflowList) DeepCopyInto(out *CronWorkflowList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CronWorkflow, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronWorkflowList.
func (in *CronWorkflowList) DeepCopy() *CronWorkflowList {
	if in == nil {
		return nil
	}
	out := new(CronWorkflowList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CronWorkflowList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	} else {
		return nil
	}
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronWorkflowSpec) DeepCopyInto(out *CronWorkflowSpec) {
	*out = *in
	if in.StartingDeadlineSeconds != nil {
		in, out := &in.StartingDeadlineSeconds, &out.StartingDeadlineSeconds
		if *in == nil {
			*out = nil
		} else {
			*out = new(int64)
			**out = **in
		}
	}
	if in.Suspend != nil {
		in, out := &in.Suspend, &out.Suspend
		if *in == nil {
			*out = nil
		} else {
			*out = new(bool)
			**out = **in
		}
	}
	in.WorkflowTemplate.DeepCopyInto(&out.WorkflowTemplate)
	if in.SuccessfulWorkflowsHistoryLimit != nil {
		in, out := &in.SuccessfulWorkflowsHistoryLimit, &out.SuccessfulWorkflowsHistoryLimit
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	if in.FailedWorkflowsHistoryLimit != nil {
		in, out := &in.FailedWorkflowsHistoryLimit, &out.FailedWorkflowsHistoryLimit
		if *in == nil {
			*out = nil
		} else {
			*out = new(int32)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronWorkflowSpec.
func (in *CronWorkflowSpec) DeepCopy() *CronWorkflowSpec {
	if in == nil {
		return nil
	}
	out := new(CronWorkflowSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CronWorkflowStatus) DeepCopyInto(out *CronWorkflowStatus) {
	*out = *in
	if in.Active != nil {
		in, out := &in.Active, &out.Active
		*out = make([]core_v1.ObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.LastScheduleTime != nil {
		in, out := &in.LastScheduleTime, &out.LastScheduleTime
		if *in == nil {
			*out = nil
		} else {
			*out = new(meta_v1.Time)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CronWorkflowStatus.
func (in *CronWorkflowStatus) DeepCopy() *CronWorkflowStatus {
	if in == nil {
		return nil
	}
	out := new(CronWorkflowStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowTemplateSpec) DeepCopyInto(out *WorkflowTemplateSpec) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowTemplateSpec.
func (in *WorkflowTemplateSpec) DeepCopy() *WorkflowTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(WorkflowTemplateSpec)
	in.DeepCopyInto(out)
	return out
}
