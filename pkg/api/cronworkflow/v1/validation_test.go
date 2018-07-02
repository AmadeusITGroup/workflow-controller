package v1

import (
	"fmt"
	"reflect"
	"testing"

	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func negative64() *int64 {
	v := int64(-1)
	return &v
}

func newCronWorkflowSpec() *CronWorkflowSpec {
	return &CronWorkflowSpec{
		Schedule:          "* * * * ?",
		ConcurrencyPolicy: batchv2alpha1.AllowConcurrent,
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
