package v1

import (
	"strings"
	"testing"
	"time"

	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func goodJobTemplateSpec() *batchv2.JobTemplateSpec {
	return &batchv2.JobTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: batch.JobSpec{
			Template: api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
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
	successCases := map[string]Workflow{
		"K1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: metav1.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec(),
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"2K1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec(),
					},
					{
						Name:        "two",
						JobTemplate: goodJobTemplateSpec(),
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"K2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
					{Name: "one", JobTemplate: goodJobTemplateSpec()},
					{
						Name:         "two",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"one"},
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"2K2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
					{Name: "one", JobTemplate: goodJobTemplateSpec()},
					{Name: "two", JobTemplate: goodJobTemplateSpec(),
						Dependencies: []string{"one"},
					},
					{Name: "three", JobTemplate: goodJobTemplateSpec()},
					{Name: "four", JobTemplate: goodJobTemplateSpec(),
						Dependencies: []string{"three"},
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"K3": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
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
				Selector: &metav1.LabelSelector{
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
	errorCases := map[string]Workflow{
		"spec.steps: Forbidden: detected cycle [one]": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
					{
						Name:         "one",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"one"},
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.steps: Forbidden: detected cycle [two one]": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
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
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.steps: Forbidden: detected cycle [three four five]": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
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
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.steps: Not found: \"three\"": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec()},
					{
						Name:         "two",
						JobTemplate:  goodJobTemplateSpec(),
						Dependencies: []string{"three"},
					},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.activeDeadlineSeconds: Invalid value: -42: must be greater than or equal to 0": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec(),
					},
				},
				ActiveDeadlineSeconds: &negative64,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "b"},
				},
			},
		},
		"spec.selector: Required value": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
					{
						Name:        "one",
						JobTemplate: goodJobTemplateSpec(),
					},
				},
			},
		},
		"spec.steps.step: Required value: either jobTemplate or externalRef must be set": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: WorkflowSpec{
				Steps: []WorkflowStep{
					{Name: "one"},
				},
				Selector: &metav1.LabelSelector{
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

func NewWorkflow() Workflow {
	return Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "mydag",
			Namespace:       api.NamespaceDefault,
			UID:             types.UID("1uid1cafe"),
			ResourceVersion: "42",
		},
		Spec: WorkflowSpec{
			Steps: []WorkflowStep{
				{
					Name:        "one",
					JobTemplate: goodJobTemplateSpec(),
				},
				{
					Name:        "two",
					JobTemplate: goodJobTemplateSpec(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"a": "b"},
			},
		},
		Status: WorkflowStatus{
			StartTime: &metav1.Time{time.Date(2009, time.January, 1, 27, 6, 25, 0, time.UTC)},
			Statuses: []WorkflowStepStatus{
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

func deleteStepStatusByName(statuses *[]WorkflowStepStatus, stepName string) {
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
		current      Workflow
		patchCurrent func(*Workflow)
		update       Workflow
		patchUpdate  func(*Workflow)
	}
	errorCases := map[string]WorkflowPair{

		"metadata.resourceVersion: Invalid value: \"\": must be specified for an update": {
			current: NewWorkflow(),
			update:  NewWorkflow(),
			patchUpdate: func(w *Workflow) {
				w.ObjectMeta.ResourceVersion = ""
			},
		},
		"workflow: Forbidden: cannot update completed workflow": {
			current: NewWorkflow(),
			patchCurrent: func(w *Workflow) {
				GetStepStatusByName(w, "one").Complete = true
				GetStepStatusByName(w, "two").Complete = true
			},
			update: NewWorkflow(),
		},
		"spec.steps: Forbidden: cannot delete running step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *Workflow) {
				deleteStepStatusByName(&(w.Status.Statuses), "two") // one is running
			},
			update: NewWorkflow(),
			patchUpdate: func(w *Workflow) {
				RemoveStepFromSpec(w, "one")
				deleteStepStatusByName(&(w.Status.Statuses), "two")
			},
		},
		"spec.steps: Forbidden: cannot modify running step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *Workflow) {
				deleteStepStatusByName(&(w.Status.Statuses), "two") // one is running
			},
			update: NewWorkflow(),
			patchUpdate: func(w *Workflow) {
				// modify "one"
				s := GetStepByName(w, "one")
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
			patchCurrent: func(w *Workflow) {
				GetStepStatusByName(w, "one").Complete = true
				deleteStepStatusByName(&(w.Status.Statuses), "two")
			},
			update: NewWorkflow(),
			patchUpdate: func(w *Workflow) {
				RemoveStepFromSpec(w, "one")
				deleteStepStatusByName(&(w.Status.Statuses), "two")
			},
		},
		"spec.steps: Forbidden: cannot modify completed step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *Workflow) {
				s := GetStepStatusByName(w, "one")
				s.Complete = true // one is complete
				//w.Status.Statuses["one"] = s
				deleteStepStatusByName(&(w.Status.Statuses), "two") // two is running
			},
			update: NewWorkflow(),
			patchUpdate: func(w *Workflow) {
				// modify "one"
				s := GetStepByName(w, "one")
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
