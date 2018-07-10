// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	kapiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "k8s.io/client-go/kubernetes"

	"github.com/amadeusitgroup/workflow-controller/kubectl-plugin/utils"
	api "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	wfversioned "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
)

var kubeClient *kclient.Clientset
var workflowClient wfversioned.Interface

// NewCmd returns WorkflowCmd instance
func NewCmd(kc *kclient.Clientset, wfc wfversioned.Interface) *cobra.Command {
	kubeClient = kc
	workflowClient = wfc

	return &cobra.Command{
		Use:   "workflow",
		Short: "workflow shows workflow custom resources",
		Long:  `workflow shows workflow custom resources`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return processGetWorkflow(cmd, args)
		},
	}
}

func processGetWorkflow(cmdC *cobra.Command, args []string) error {
	var err error
	namespace := os.Getenv("KUBECTL_PLUGINS_CURRENT_NAMESPACE")

	workflowName := ""
	if len(args) > 0 {
		workflowName = args[0]
	}

	wfs := &api.WorkflowList{}
	if workflowName == "" {
		wfs, err = workflowClient.WorkflowV1().Workflows(namespace).List(meta_v1.ListOptions{})
		if err != nil {
			return fmt.Errorf("unable to list Workflows err:%v", err)
		}
	} else {
		wf, err := workflowClient.WorkflowV1().Workflows(namespace).Get(workflowName, meta_v1.GetOptions{})
		if err == nil && wf != nil {
			wfs.Items = append(wfs.Items, *wf)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("unable to get Workflow err:%v", err)
		}
	}

	utils.GenerateTable([]string{"Name", "Namespace", "Status", "Steps", "Current_Step(s)", "Job(s)_Status"}, computeTableData(kubeClient, wfs))
	return nil
}

func hasStatus(wf *api.Workflow, conditionType api.WorkflowConditionType, status kapiv1.ConditionStatus) bool {
	for _, cond := range wf.Status.Conditions {
		if cond.Type == conditionType && cond.Status == status {
			return true
		}
	}
	return false
}

func buildWorkflowStatus(wf *api.Workflow) string {
	status := "Created"
	if wf.Status.StartTime != nil {
		status = "Running"
	}
	if hasStatus(wf, api.WorkflowComplete, kapiv1.ConditionTrue) {
		status = string(api.WorkflowComplete)
	} else if hasStatus(wf, api.WorkflowFailed, kapiv1.ConditionTrue) {
		status = string(api.WorkflowFailed)
	}

	return status
}

func computeTableData(kubeClient *kclient.Clientset, wfs *api.WorkflowList) [][]string {
	data := [][]string{}
	for _, wf := range wfs.Items {
		status := buildWorkflowStatus(&wf)
		nbSteps := len(wf.Spec.Steps)
		currentStepID := len(wf.Status.Statuses)
		currentStepNames := []string{}
		currentJobName := "-"
		jobsStatus := []string{}
		for _, s := range wf.Status.Statuses {
			if s.Complete != true {
				currentStepNames = append(currentStepNames, s.Name)
			}
			if s.Reference.Name != "" {
				jobStatus := ""
				currentJobName = s.Reference.Name
				job, err := kubeClient.BatchV1().Jobs(wf.Namespace).Get(s.Reference.Name, meta_v1.GetOptions{})
				if err == nil && job != nil {
					jobStatus = utils.BuildjobStatus(job)
					currentJobName = fmt.Sprintf("%s(%s)", currentJobName, jobStatus)
				}
				jobsStatus = append(jobsStatus, currentJobName)
			}
		}
		if len(jobsStatus) == 0 {
			jobsStatus = append(jobsStatus, "-")
		}
		if len(currentStepNames) == 0 {
			currentStepNames = append(currentStepNames, "-")
		}
		data = append(data, []string{wf.Name, wf.Namespace, status, fmt.Sprintf("%d/%d", currentStepID, nbSteps), strings.Join(currentStepNames, ","), strings.Join(jobsStatus, ",")})
	}

	return data
}
