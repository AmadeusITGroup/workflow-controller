package controller

import (
	"reflect"
	"testing"
	"time"

	cwapi "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	testingutil "github.com/amadeusitgroup/workflow-controller/pkg/util/testing"
)

func Test_getRecentUnmetScheduleTimes(t *testing.T) {
	now, err := time.Parse(time.RFC3339, "2016-05-19T10:00:00Z")
	if err != nil {
		t.Errorf("Cannot initialize test: %v", err)
	}
	type args struct {
		cronWorkflow *cwapi.CronWorkflow
		now          time.Time
	}
	tests := []struct {
		name               string
		args               args
		want               []time.Time
		wantedErrorMessage string
	}{
		{
			name: "nominal case",
			args: args{
				cronWorkflow: testingutil.NewCronWorkflow(),
				now:          now,
			},
			want:               []time.Time{},
			wantedErrorMessage: `too many missed start time (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRecentUnmetScheduleTimes(tt.args.cronWorkflow, tt.args.now)
			if (err != nil) && err.Error() != tt.wantedErrorMessage {
				t.Errorf("getRecentUnmetScheduleTimes() got error message %q, wanted error message %q", err, tt.wantedErrorMessage)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRecentUnmetScheduleTimes() = %v, want %v", got, tt.want)
			}
		})
	}
}
