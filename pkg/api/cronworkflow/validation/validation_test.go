package validation

import (
	"fmt"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"

	cronworkflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	testingutil "github.com/amadeusitgroup/workflow-controller/pkg/util/testing"
)

func negative64() *int64 {
	v := int64(-1)
	return &v
}

func TestValidateCronWorkflowSpec(t *testing.T) {
	type args struct {
		spec *cronworkflowv1.CronWorkflowSpec
	}
	tests := []struct {
		name               string
		args               args
		tweakSpec          func(*cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec
		wantedErrorMessage string
	}{
		{
			name: "good spec",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "bad schedule",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "bad schedule"
				return s
			},
			wantedErrorMessage: `[spec.schedule: Invalid value: "bad schedule": Expected exactly 5 fields, found 2: bad schedule]`,
		},
		{
			name: "special characters",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "0,1  * * * *"
				return s
			},
			wantedErrorMessage: `[]`,
		},
		{
			name: "non-standard supported: @yearly",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@yearly"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard supported: @annually",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@annually"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard supported: @monthly",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@monthly"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard supported: @weekly",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@weekly"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard supported: @daily",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@daily"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "non-standard NOT supported: @reboot",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@reboot"
				return s
			},
			wantedErrorMessage: `[spec.schedule: Invalid value: "@reboot": Unrecognized descriptor: @reboot]`,
		},
		{
			name: "non-standard supported: @hourly",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.Schedule = "@hourly"
				return s
			},
			wantedErrorMessage: "[]",
		},
		{
			name: "negative startingDeadlineSeconds",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.StartingDeadlineSeconds = negative64()
				return s
			},
			wantedErrorMessage: `[spec.startingDeadlineSeconds: Invalid value: -1: must be greater than or equal to 0]`,
		},
		{
			name: "bad concurrency policy",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
				s := spec.DeepCopy()
				s.ConcurrencyPolicy = ""
				return s
			},
			wantedErrorMessage: `[spec.concurrencyPolicy: Required value]`,
		},
		{
			name: "bad concurrency policy 2",
			args: args{
				spec: testingutil.NewCronWorkflowSpec(),
			},
			tweakSpec: func(spec *cronworkflowv1.CronWorkflowSpec) *cronworkflowv1.CronWorkflowSpec {
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
