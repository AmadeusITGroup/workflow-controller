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

package cronworkflow

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "k8s.io/client-go/kubernetes"

	"github.com/amadeusitgroup/workflow-controller/kubectl-plugin/utils"
	api "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	wfversioned "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
)

var kubeClient *kclient.Clientset
var workflowClient wfversioned.Interface

// NewCmd returns WorkflowCmd instance
func NewCmd(kc *kclient.Clientset, wfc wfversioned.Interface) *cobra.Command {
	kubeClient = kc
	workflowClient = wfc

	return &cobra.Command{
		Use:   "cronworkflow",
		Short: "cronworkflow shows cronworkflow custom resources",
		Long:  `cronworkflow shows cronworkflow custom resources`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return processGetWorkflow(cmd, args)
		},
	}
}

func processGetWorkflow(cmdC *cobra.Command, args []string) error {
	var err error
	namespace := os.Getenv("KUBECTL_PLUGINS_CURRENT_NAMESPACE")

	cronworkflowName := ""
	if len(args) > 0 {
		cronworkflowName = args[0]
	}

	cwfs := &api.CronWorkflowList{}
	if cronworkflowName == "" {
		cwfs, err = workflowClient.CronworkflowV1().CronWorkflows(namespace).List(meta_v1.ListOptions{})
		if err != nil {
			return fmt.Errorf("unable to list CronWorkflows err:%v", err)
		}
	} else {
		cwf, err := workflowClient.CronworkflowV1().CronWorkflows(namespace).Get(cronworkflowName, meta_v1.GetOptions{})
		if err == nil && cwf != nil {
			cwfs.Items = append(cwfs.Items, *cwf)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("unable to get CronWorkflow err:%v", err)
		}
	}

	utils.GenerateTable([]string{"Name", "Namespace", "schedule", "anti-schedule", "Active", "LastSchedule"}, computeTableData(kubeClient, cwfs))
	return nil
}

func computeTableData(kubeClient *kclient.Clientset, cwfs *api.CronWorkflowList) [][]string {
	data := [][]string{}
	for _, cwf := range cwfs.Items {
		nbActive := fmt.Sprintf("%d", len(cwf.Status.Active))
		data = append(data, []string{cwf.Name, cwf.Namespace, cwf.Spec.Schedule, cwf.Spec.AntiSchedule, nbActive, cwf.Status.LastScheduleTime.String()})
	}

	return data
}
