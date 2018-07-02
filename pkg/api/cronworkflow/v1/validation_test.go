package v1

import (
	"fmt"
	"reflect"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	workflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
)

func negative64() *int64 {
	v := int64(-1)
	return &v
}

// shamless copied from .../pkg/api/workflow/v1 TODO: create a testin package
func goodJobTemplateSpec() *batchv2alpha1.JobTemplateSpec {
	return &batchv2alpha1.JobTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: batchv1.JobSpec{
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

func newWorkflowTemplateSpec() WorkflowTemplateSpec {
	return WorkflowTemplateSpec{
		Spec: workflowv1.WorkflowSpec{
			Steps: []workflowv1.WorkflowStep{
				{
					Name:        "one",
					JobTemplate: goodJobTemplateSpec(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"a": "b"},
			},
		},
	}
}

func newCronWorkflowSpec() *CronWorkflowSpec {
	return &CronWorkflowSpec{
		Schedule:          "* * * * *",
		ConcurrencyPolicy: batchv2alpha1.AllowConcurrent,
		WorkflowTemplate:  newWorkflowTemplateSpec(),
	}
}

func TestValidateCronWorkflowSpec(t *testing.T) {
	type args struct {
		spec *CronWorkflowSpec
	}
	tests := []struct {
		name               string
		args               args
		tweakSpec          func(*CronWorkflowSpec) *CronWorkflowSpec
		wantedErrorMessage string
	}{
		{
			name: "good spec",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "bad schedule",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "bad schedule"
				return s
			},
			wantedErrorMessage: `[spec.schedule: Invalid value: "bad schedule": Expected exactly 5 fields, found 2: bad schedule]`,
		},
		{
			name: "non-standard supported: @yearly",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@yearly"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard supported: @annually",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@annually"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard supported: @monthly",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@monthly"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard supported: @weekly",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@weekly"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard supported: @daily",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@daily"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard NOT supported: @reboot",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@reboot"
				return s
			},
			wantedErrorMessage: `[spec.schedule: Invalid value: "@reboot": Unrecognized descriptor: @reboot]`,
		},
		{
			name: "non-standard supported: @hourly",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@hourly"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "negative startingDeadlineSeconds",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.StartingDeadlineSeconds = negative64()
				return s
			},
			wantedErrorMessage: `[spec.startingDeadlineSeconds: Invalid value: -1: must be greater than or equal to 0]`,
		},
		{
			name: "bad concurrency policy",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.ConcurrencyPolicy = ""
				return s
			},
			wantedErrorMessage: `[spec.concurrencyPolicy: Required value]`,
		},
		{
			name: "bad concurrency policy 2",
			args: args{
				spec: newCronWorkflowSpec(),
			},
			tweakSpec: func(spec *CronWorkflowSpec) *CronWorkflowSpec {
				s := spec.DeepCopy()
				s.ConcurrencyPolicy = "mySpecialPolicy"
				return s
			},
			wantedErrorMessage: `[spec.concurrencyPolicy: Unsupported value: "mySpecialPolicy": supported values: Allow, Forbid, Replace]`,
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tweakSpec != nil {
				tt.args.spec = tt.tweakSpec(tt.args.spec)
			}
			got := ValidateCronWorkflowSpec(tt.args.spec, field.NewPath("spec"))
			gotErrorMessage := fmt.Sprintf("%s", got)
			if !reflect.DeepEqual(gotErrorMessage, tt.wantedErrorMessage) {
				t.Errorf("ValidateCronWorkflowSpec() = %v, want %v", gotErrorMessage, tt.wantedErrorMessage)
			}
		})
	}
}
