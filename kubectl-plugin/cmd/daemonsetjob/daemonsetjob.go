package daemonsetjob

import (
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	kapiv1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kclient "k8s.io/client-go/kubernetes"

	"github.com/amadeusitgroup/workflow-controller/kubectl-plugin/utils"
	api "github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	wfversioned "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
)

var kubeClient *kclient.Clientset
var workflowClient wfversioned.Interface

// NewCmd returns WorkflowCmd instance
func NewCmd(kc *kclient.Clientset, wfc wfversioned.Interface) *cobra.Command {
	kubeClient = kc
	workflowClient = wfc

	return &cobra.Command{
		Use:   "daemonsetjob",
		Short: "daemonsetjob shows daemonsetjob custom resources",
		Long:  `daemonsetjob shows daemonsetjob custom resources`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return processGetDaemonSetJob(cmd, args)
		},
	}
}

func processGetDaemonSetJob(cmdC *cobra.Command, args []string) error {
	var err error
	namespace := os.Getenv("KUBECTL_PLUGINS_CURRENT_NAMESPACE")

	daemonSetJobName := ""
	if len(args) > 0 {
		daemonSetJobName = args[0]
	}

	dsjs := &api.DaemonSetJobList{}
	if daemonSetJobName == "" {
		dsjs, err = workflowClient.DaemonsetjobV1().DaemonSetJobs(namespace).List(meta_v1.ListOptions{})
		if err != nil {
			return fmt.Errorf("unable to list Workflows err:%v", err)
		}
	} else {
		dsj, err := workflowClient.DaemonsetjobV1().DaemonSetJobs(namespace).Get(daemonSetJobName, meta_v1.GetOptions{})
		if err == nil && dsj != nil {
			dsjs.Items = append(dsjs.Items, *dsj)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("unable to get Workflow err:%v", err)
		}
	}

	utils.GenerateTable([]string{"Namespace", "Name", "Status", "Active", "Succeeded", "Failed", "Node Selector", "Age"}, computeTableData(kubeClient, dsjs))
	return nil
}

func hasStatus(dsj *api.DaemonSetJob, conditionType api.DaemonSetJobConditionType, status kapiv1.ConditionStatus) bool {
	for _, cond := range dsj.Status.Conditions {
		if cond.Type == conditionType && cond.Status == status {
			return true
		}
	}
	return false
}

func buildDaemonSetJobStatus(dsj *api.DaemonSetJob) string {
	status := "Created"
	if dsj.Status.StartTime != nil {
		status = "Running"
	}
	if hasStatus(dsj, api.DaemonSetJobComplete, kapiv1.ConditionTrue) {
		status = string(api.DaemonSetJobComplete)
	} else if hasStatus(dsj, api.DaemonSetJobFailed, kapiv1.ConditionTrue) {
		status = string(api.DaemonSetJobFailed)
	}

	return status
}

func computeTableData(kubeClient *kclient.Clientset, dsjs *api.DaemonSetJobList) [][]string {
	data := [][]string{}

	for _, dsj := range dsjs.Items {
		status := buildDaemonSetJobStatus(&dsj)
		active := fmt.Sprintf("%d", dsj.Status.Active)
		succeeded := fmt.Sprintf("%d", dsj.Status.Succeeded)
		failed := fmt.Sprintf("%d", dsj.Status.Failed)
		nodeSelector := "<none>"
		if dsj.Spec.NodeSelector != nil {
			nodeSelector = dsj.Spec.NodeSelector.String()
		}
		age := translateTimestampSince(dsj.CreationTimestamp)

		data = append(data, []string{dsj.Namespace, dsj.Name, status, active, succeeded, failed, nodeSelector, age})
	}

	return data
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp meta_v1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return humanize.RelTime(timestamp.Time, time.Now(), "", "")
}
