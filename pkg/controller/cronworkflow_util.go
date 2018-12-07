package controller

import (
	"fmt"
	"time"

	cwapi "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	"github.com/robfig/cron"
)

func getRecentUnmetScheduleTimes(cronWorkflow *cwapi.CronWorkflow, now time.Time) ([]time.Time, error) {
	times := []time.Time{}
	sched, err := cron.ParseStandard(cronWorkflow.Spec.Schedule)
	if err != nil {
		return times, fmt.Errorf("Unparsable schedule: %s : %s", cronWorkflow.Spec.Schedule, err)
	}
	earliestTime := cronWorkflow.ObjectMeta.CreationTimestamp.Time
	if cronWorkflow.Status.LastScheduleTime != nil {
		earliestTime = cronWorkflow.Status.LastScheduleTime.Time
	}
	if cronWorkflow.Spec.StartingDeadlineSeconds != nil {
		// Controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-time.Second * time.Duration(*cronWorkflow.Spec.StartingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}
	if earliestTime.After(now) {
		return []time.Time{}, nil
	}
	for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
		times = append(times, t)
		if len(times) > 100 {
			// We can't get the most recent times so just return an empty slice
			return []time.Time{}, fmt.Errorf("too many missed start time (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew")
		}
	}
	return times, nil
}
