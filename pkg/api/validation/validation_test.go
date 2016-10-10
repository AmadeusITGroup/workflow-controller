/*
Copyright 2016 The Kubernetes Authors

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
	"strings"
	"testing"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/types"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
)

func goodJobTemplateSpec() *batch.JobTemplateSpec {
	return &batch.JobTemplateSpec{
		ObjectMeta: api.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: batch.JobSpec{
			Template: api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:            "baz",
							Image:           "foo/bar",
							ImagePullPolicy: "IfNotPresent",
						},
					},
					RestartPolicy: "OnFailure",
					DNSPolicy:     "Default",
				},
			},
		},
	}
}

func TestValidateWorkflowSpec(t *testing.T) {
	successCases := map[string]wapi.Workflow{
		"K1": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec(),
					},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"2K1": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec(),
					},
					{
						Name:        "two",
						JobTemplate: goodJobTemplateSpec(),
					},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"K2": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{Name: "one", JobTemplate: goodJobTemplateSpec()},
					{
						Name:         "two",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"one"},
					},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"2K2": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{Name: "one", JobTemplate: goodJobTemplateSpec()},
					{Name: "two", JobTemplate: goodJobTemplateSpec(),
						Dependencies: []string{"one"},
					},
					{Name: "three", JobTemplate: goodJobTemplateSpec()},
					{Name: "four", JobTemplate: goodJobTemplateSpec(),
						Dependencies: []string{"three"},
					},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"K3": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec(),
					},
					{
						Name:         "two",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"one"},
					},
					{
						Name:         "three",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"one", "two"},
					},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
	}
	for k, v := range successCases {
		errs := ValidateWorkflow(&v)
		if len(errs) != 0 {
			t.Errorf("%s unexpected error %v", k, errs)
		}
	}
	negative64 := int64(-42)
	errorCases := map[string]wapi.Workflow{
		"spec.steps: Forbidden: detected cycle [one]": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{
						Name:         "one",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"one"},
					},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.steps: Forbidden: detected cycle [two one]": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{
						Name:         "one",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"two"},
					},
					{
						Name:         "two",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"one"},
					},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.steps: Forbidden: detected cycle [three four five]": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec()},
					{
						Name:         "two",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"one"},
					},
					{
						Name:         "three",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"two", "five"},
					},
					{
						Name:         "four",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"three"},
					},
					{
						Name:         "five",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"four"},
					},
					{
						Name:         "six",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"five"},
					},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.steps: Not found: \"three\"": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec()},
					{
						Name:         "two",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"three"},
					},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.activeDeadlineSeconds: Invalid value: -42: must be greater than or equal to 0": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec(),
					},
				},
				ActiveDeadlineSeconds: &negative64,
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.selector: Required value": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec(),
					},
				},
			},
		},
		"spec.steps.step: Required value: either jobTemplate or externalRef must be set": {
			ObjectMeta: api.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: wapi.WorkflowSpec{
				Steps: []wapi.WorkflowStep{
					{Name: "one"},
				},
				Selector: &unversioned.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
	}

	for k, v := range errorCases {
		errs := ValidateWorkflow(&v)
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		} else {
			s := strings.Split(k, ":")
			err := errs[0]
			if err.Field != s[0] || !strings.Contains(err.Error(), s[1]) {
				t.Errorf("unexpected error: %v, expected: %s", err, k)
			}
		}
	}
}

func NewWorkflow() wapi.Workflow {
	return wapi.Workflow{
		ObjectMeta: api.ObjectMeta{
			Name:            "mydag",
			Namespace:       api.NamespaceDefault,
			UID:             types.UID("1uid1cafe"),
			ResourceVersion: "42",
		},
		Spec: wapi.WorkflowSpec{
			Steps: []wapi.WorkflowStep{
				{
					Name:        "one",
					JobTemplate: goodJobTemplateSpec(),
				},
				{
					Name:        "two",
					JobTemplate: goodJobTemplateSpec(),
				},
			},
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{"a": "b"},
			},
		},
		Status: wapi.WorkflowStatus{
			StartTime: &unversioned.Time{time.Date(2009, time.January, 1, 27, 6, 25, 0, time.UTC)},
			Statuses: []wapi.WorkflowStepStatus{
				{
					Name:     "one",
					Complete: false,
				},
				{
					Name:     "two",
					Complete: false,
				},
			},
		},
	}
}

func deleteStepStatusByName(statuses *[]wapi.WorkflowStepStatus, stepName string) {
	for i := range *statuses {
		if (*statuses)[i].Name == stepName {
			*statuses = (*statuses)[:i+copy((*statuses)[i:], (*statuses)[i+1:])]
			return
		}
	}
	panic("cannot find stepStatus")
}

func TestValidateWorkflowUpdate(t *testing.T) {
	type WorkflowPair struct {
		current      wapi.Workflow
		patchCurrent func(*wapi.Workflow)
		update       wapi.Workflow
		patchUpdate  func(*wapi.Workflow)
	}
	errorCases := map[string]WorkflowPair{

		"metadata.resourceVersion: Invalid value: \"\": must be specified for an update": {
			current: NewWorkflow(),
			update:  NewWorkflow(),
			patchUpdate: func(w *wapi.Workflow) {
				w.ObjectMeta.ResourceVersion = ""
			},
		},
		"workflow: Forbidden: cannot update completed workflow": {
			current: NewWorkflow(),
			patchCurrent: func(w *wapi.Workflow) {
				w.GetStepStatusByName("one").Complete = true
				w.GetStepStatusByName("two").Complete = true
			},
			update: NewWorkflow(),
		},
		"spec.steps: Forbidden: cannot delete running step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *wapi.Workflow) {
				deleteStepStatusByName(&(w.Status.Statuses), "two") // one is running
			},
			update: NewWorkflow(),
			patchUpdate: func(w *wapi.Workflow) {
				w.RemoveStepFromSpec("one")
				deleteStepStatusByName(&(w.Status.Statuses), "two")
			},
		},
		"spec.steps: Forbidden: cannot modify running step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *wapi.Workflow) {
				deleteStepStatusByName(&(w.Status.Statuses), "two") // one is running
			},
			update: NewWorkflow(),
			patchUpdate: func(w *wapi.Workflow) {
				// modify "one"
				s := w.GetStepByName("one")
				if s == nil {
					t.Fatalf("cannot update workflow")
				}
				s.JobTemplate.Labels = make(map[string]string)
				s.JobTemplate.Labels["foo"] = "bar2"
				//w.Spec.Steps["one"] = s
				deleteStepStatusByName(&(w.Status.Statuses), "two")
				//delete(w.Status.Statuses, "two")
			},
		},

		"spec.steps: Forbidden: cannot delete completed step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *wapi.Workflow) {
				w.GetStepStatusByName("one").Complete = true
				deleteStepStatusByName(&(w.Status.Statuses), "two")
			},
			update: NewWorkflow(),
			patchUpdate: func(w *wapi.Workflow) {
				w.RemoveStepFromSpec("one")
				deleteStepStatusByName(&(w.Status.Statuses), "two")
			},
		},
		"spec.steps: Forbidden: cannot modify completed step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *wapi.Workflow) {
				s := w.GetStepStatusByName("one")
				s.Complete = true // one is complete
				//w.Status.Statuses["one"] = s
				deleteStepStatusByName(&(w.Status.Statuses), "two") // two is running
			},
			update: NewWorkflow(),
			patchUpdate: func(w *wapi.Workflow) {
				// modify "one"
				s := w.GetStepByName("one")
				if s == nil {
					t.Fatalf("unable to patch workflow")
				}
				s.JobTemplate.Labels = make(map[string]string)
				s.JobTemplate.Labels["foo"] = "bar2"
				deleteStepStatusByName(&(w.Status.Statuses), "two") // two always running
			},
		},
	}
	for k, v := range errorCases {
		if v.patchUpdate != nil {
			v.patchUpdate(&v.update)
		}
		if v.patchCurrent != nil {
			v.patchCurrent(&v.current)
		}
		errs := ValidateWorkflowUpdate(&v.update, &v.current)
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
			continue
		}
		if errs.ToAggregate().Error() != k {
			t.Errorf("unexpected error: %v, expected: %s", errs, k)
		}
	}
}
