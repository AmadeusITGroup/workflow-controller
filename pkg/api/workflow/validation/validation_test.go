package validation

import (
	"strings"
	"testing"
	"time"

	api "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	workflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/util"
	testingutil "github.com/amadeusitgroup/workflow-controller/pkg/util/testing"
)

func TestValidateWorkflowSpec(t *testing.T) {
	successCases := map[string]workflowv1.Workflow{
		"K1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: metav1.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: testingutil.ValidFakeTemplateSpec(),
					},
					{
						Name:        "two",
						JobTemplate: testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{Name: "one", JobTemplate: testingutil.ValidFakeTemplateSpec()},
					{
						Name:         "two",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{Name: "one", JobTemplate: testingutil.ValidFakeTemplateSpec()},
					{Name: "two", JobTemplate: testingutil.ValidFakeTemplateSpec(),
						Dependencies: []string{"one"},
					},
					{Name: "three", JobTemplate: testingutil.ValidFakeTemplateSpec()},
					{Name: "four", JobTemplate: testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: testingutil.ValidFakeTemplateSpec(),
					},
					{
						Name:         "two",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
						Dependencies: []string{"one"},
					},
					{
						Name:         "three",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
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
	errorCases := map[string]workflowv1.Workflow{
		"spec.steps: Forbidden: detected cycle [one]": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydag",
				Namespace: api.NamespaceDefault,
				UID:       types.UID("1uid1cafe"),
			},
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{
						Name:         "one",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{
						Name:         "one",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
						Dependencies: []string{"two"},
					},
					{
						Name:         "two",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: testingutil.ValidFakeTemplateSpec()},
					{
						Name:         "two",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
						Dependencies: []string{"one"},
					},
					{
						Name:         "three",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
						Dependencies: []string{"two", "five"},
					},
					{
						Name:         "four",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
						Dependencies: []string{"three"},
					},
					{
						Name:         "five",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
						Dependencies: []string{"four"},
					},
					{
						Name:         "six",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: testingutil.ValidFakeTemplateSpec()},
					{
						Name:         "two",
						JobTemplate:  testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
					{
						Name:        "one",
						JobTemplate: testingutil.ValidFakeTemplateSpec(),
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
			Spec: workflowv1.WorkflowSpec{
				Steps: []workflowv1.WorkflowStep{
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

func NewWorkflow() workflowv1.Workflow {
	return workflowv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "mydag",
			Namespace:       api.NamespaceDefault,
			UID:             types.UID("1uid1cafe"),
			ResourceVersion: "42",
		},
		Spec: workflowv1.WorkflowSpec{
			Steps: []workflowv1.WorkflowStep{
				{
					Name:        "one",
					JobTemplate: testingutil.ValidFakeTemplateSpec(),
				},
				{
					Name:        "two",
					JobTemplate: testingutil.ValidFakeTemplateSpec(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"a": "b"},
			},
		},
		Status: workflowv1.WorkflowStatus{
			StartTime: &metav1.Time{time.Date(2009, time.January, 1, 27, 6, 25, 0, time.UTC)},
			Statuses: []workflowv1.WorkflowStepStatus{
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

func TestValidateWorkflowUpdate(t *testing.T) {
	type WorkflowPair struct {
		current      workflowv1.Workflow
		patchCurrent func(*workflowv1.Workflow)
		update       workflowv1.Workflow
		patchUpdate  func(*workflowv1.Workflow)
	}
	errorCases := map[string]WorkflowPair{

		"metadata.resourceVersion: Invalid value: \"\": must be specified for an update": {
			current: NewWorkflow(),
			update:  NewWorkflow(),
			patchUpdate: func(w *workflowv1.Workflow) {
				w.ObjectMeta.ResourceVersion = ""
			},
		},
		"workflow: Forbidden: cannot update completed workflow": {
			current: NewWorkflow(),
			patchCurrent: func(w *workflowv1.Workflow) {
				util.GetStepStatusByName(w, "one").Complete = true
				util.GetStepStatusByName(w, "two").Complete = true
			},
			update: NewWorkflow(),
		},
		"spec.steps: Forbidden: cannot delete running step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *workflowv1.Workflow) {
				testingutil.DeleteStepStatusByName(&(w.Status.Statuses), "two") // one is running
			},
			update: NewWorkflow(),
			patchUpdate: func(w *workflowv1.Workflow) {
				util.RemoveStepFromSpec(w, "one")
				testingutil.DeleteStepStatusByName(&(w.Status.Statuses), "two")
			},
		},
		"spec.steps: Forbidden: cannot modify running step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *workflowv1.Workflow) {
				testingutil.DeleteStepStatusByName(&(w.Status.Statuses), "two") // one is running
			},
			update: NewWorkflow(),
			patchUpdate: func(w *workflowv1.Workflow) {
				// modify "one"
				s := util.GetStepByName(w, "one")
				if s == nil {
					t.Fatalf("cannot update workflow")
				}
				s.JobTemplate.Labels = make(map[string]string)
				s.JobTemplate.Labels["foo"] = "bar2"
				//w.Spec.Steps["one"] = s
				testingutil.DeleteStepStatusByName(&(w.Status.Statuses), "two")
				//delete(w.Status.Statuses, "two")
			},
		},

		"spec.steps: Forbidden: cannot delete completed step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *workflowv1.Workflow) {
				util.GetStepStatusByName(w, "one").Complete = true
				testingutil.DeleteStepStatusByName(&(w.Status.Statuses), "two")
			},
			update: NewWorkflow(),
			patchUpdate: func(w *workflowv1.Workflow) {
				util.RemoveStepFromSpec(w, "one")
				testingutil.DeleteStepStatusByName(&(w.Status.Statuses), "two")
			},
		},
		"spec.steps: Forbidden: cannot modify completed step \"one\"": {
			current: NewWorkflow(),
			patchCurrent: func(w *workflowv1.Workflow) {
				s := util.GetStepStatusByName(w, "one")
				s.Complete = true // one is complete
				//w.Status.Statuses["one"] = s
				testingutil.DeleteStepStatusByName(&(w.Status.Statuses), "two") // two is running
			},
			update: NewWorkflow(),
			patchUpdate: func(w *workflowv1.Workflow) {
				// modify "one"
				s := util.GetStepByName(w, "one")
				if s == nil {
					t.Fatalf("unable to patch workflow")
				}
				s.JobTemplate.Labels = make(map[string]string)
				s.JobTemplate.Labels["foo"] = "bar2"
				testingutil.DeleteStepStatusByName(&(w.Status.Statuses), "two") // two always running
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
