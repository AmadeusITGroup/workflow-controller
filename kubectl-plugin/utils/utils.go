package utils

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"

	kbatchv1 "k8s.io/api/batch/v1"
	kapiv1 "k8s.io/api/core/v1"
)

// NewTable returns new Table instance
func NewTable(headers []string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)

	return table
}

// BuildjobStatus return job status as string
func BuildjobStatus(job *kbatchv1.Job) string {
	status := "Created"
	if job.Status.StartTime != nil {
		status = "Running"
	}
	if hasJobStatus(job, kbatchv1.JobComplete, kapiv1.ConditionTrue) {
		status = string(kbatchv1.JobComplete)
	} else if hasJobStatus(job, kbatchv1.JobFailed, kapiv1.ConditionTrue) {
		status = string(kbatchv1.JobFailed)
	}
	return status
}

func hasJobStatus(job *kbatchv1.Job, conditionType kbatchv1.JobConditionType, status kapiv1.ConditionStatus) bool {
	for _, cond := range job.Status.Conditions {
		if cond.Type == conditionType && cond.Status == status {
			return true
		}
	}
	return false
}

// GenerateTable used to generate the ouput table
func GenerateTable(headers []string, data [][]string) {
	if len(data) == 0 {
		resourcesNotFound()
		return
	}

	table := NewTable(headers)
	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output
}

func resourcesNotFound() {
	fmt.Println("No resources found.")
}
