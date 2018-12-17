package framework

import (
	"fmt"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"

	clientset "k8s.io/client-go/kubernetes"

	daemonsetjobv1 "github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/workflow-controller/pkg/controller"
)

// NewDaemonSetJob return new DaemonSetJob instance
func NewDaemonSetJob(group, version, name, namespace string, nodeSelector map[string]string) *daemonsetjobv1.DaemonSetJob {
	s := &daemonsetjobv1.DaemonSetJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSetJob",
			APIVersion: group + "/" + version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: daemonsetjobv1.DaemonSetJobSpec{
			JobTemplate: &batchv2.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: batch.JobSpec{
					Template: api.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{},
						Spec: api.PodSpec{
							Containers: []api.Container{
								{
									Name:            "sleep",
									Image:           "gcr.io/google_containers/busybox",
									Command:         []string{"sh", "-c", "echo Starting on: $(date); sleep 5; echo Goodbye cruel world at: $(date)"},
									ImagePullPolicy: "IfNotPresent",
								},
							},
							RestartPolicy: "Never",
							DNSPolicy:     "Default",
						},
					},
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"daemonsetjob": "myjob",
				},
			},
		},
	}
	if nodeSelector != nil {
		s.Spec.NodeSelector = &metav1.LabelSelector{
			MatchLabels: nodeSelector,
		}
	}
	return s
}

// HOCheckDaemonSetJobRegistration check wether DaemonSetJob is defined
func HOCheckDaemonSetJobRegistration(extensionClient apiextensionsclient.Clientset) func() error {
	return func() error {
		deamonSetJobResourceName := daemonsetjobv1.ResourcePlural + "." + workflow.GroupName
		_, err := extensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(deamonSetJobResourceName, metav1.GetOptions{})
		return err
	}
}

// HOCreateDaemonSetJob is an higher order func that returns the func to create a DaemonSetJob
func HOCreateDaemonSetJob(workflowClient versioned.Interface, daemonsetjob *daemonsetjobv1.DaemonSetJob) func() error {
	return func() error {
		if _, err := workflowClient.DaemonsetjobV1().DaemonSetJobs(daemonsetjob.Namespace).Create(daemonsetjob); err != nil {
			Logf("cannot create DaemonSetJob %s/%s: %v", daemonsetjob.Namespace, daemonsetjob.Name, err)
			return err
		}
		Logf("DaemonSetJob created")
		return nil
	}
}

// HOUpdateDaemonSetJob is an higher order func that returns the func to update a DaemonSetJob
func HOUpdateDaemonSetJob(workflowClient versioned.Interface, daemonsetjob *daemonsetjobv1.DaemonSetJob) func() error {
	return func() error {
		d, err := workflowClient.DaemonsetjobV1().DaemonSetJobs(daemonsetjob.Namespace).Get(daemonsetjob.Name, metav1.GetOptions{})
		if err != nil {
			Logf("Cannot get daemonsetjob:%v", err)
			return err
		}
		// add a volume to update the JobSpec.
		d.Spec.JobTemplate.Spec.Template.Spec.Volumes = append(d.Spec.JobTemplate.Spec.Template.Spec.Volumes, api.Volume{Name: "emptyvol", VolumeSource: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}})
		// add label to easily the new job version.
		if d.Spec.JobTemplate.Labels == nil {
			d.Spec.JobTemplate.Labels = map[string]string{}
		}
		d.Spec.JobTemplate.Labels["test-update"] = "true"

		if _, err := workflowClient.DaemonsetjobV1().DaemonSetJobs(daemonsetjob.Namespace).Update(d); err != nil {
			Logf("cannot update DaemonSetJob %s/%s: %v", d.Namespace, d.Name, err)
			return err
		}
		Logf("DaemonSetJob updated")
		return nil
	}
}

// HOIsDaemonSetJobStarted is an higher order func that returns the func that checks whether DaemonSetJob is started
func HOIsDaemonSetJobStarted(workflowClient versioned.Interface, daemonsetjob *daemonsetjobv1.DaemonSetJob) func() error {
	return func() error {
		daemonSetJobs, err := workflowClient.DaemonsetjobV1().DaemonSetJobs(daemonsetjob.Namespace).List(metav1.ListOptions{})
		if err != nil {
			Logf("Cannot list daemonsetjob:%v", err)
			return err
		}
		if len(daemonSetJobs.Items) != 1 {
			return fmt.Errorf("Expected only 1 DaemonSetJob got %d, %s/%s", len(daemonSetJobs.Items), daemonsetjob.Namespace, daemonsetjob.Name)
		}
		if daemonSetJobs.Items[0].Status.StartTime != nil {
			Logf("DaemonSetJob started")
			return nil
		}
		Logf("DaemonSetJob %s not updated", daemonsetjob.Name)
		return fmt.Errorf("daemonSetJob %s not updated", daemonsetjob.Name)
	}
}

// HOIsDaemonSetJobJobsStarted is an higher order func that returns the func that checks whether Jobs linked to a DaemonSetJob are started
func HOIsDaemonSetJobJobsStarted(kubeclient clientset.Interface, daemonsetjob *daemonsetjobv1.DaemonSetJob) func() error {
	return func() error {
		jobs, err := kubeclient.BatchV1().Jobs(daemonsetjob.Namespace).List(metav1.ListOptions{
			LabelSelector: labels.Set(daemonsetjob.Labels).AsSelector().String(),
		})
		if err != nil {
			Logf("Cannot list jobs:%v", err)
			return err
		}
		if len(jobs.Items) != 1 {
			return fmt.Errorf("Expected only 1 Job got %d, %s/%s", len(jobs.Items), daemonsetjob.Namespace, daemonsetjob.Name)
		}
		if jobs.Items[0].Status.StartTime != nil {
			Logf("Job started")
			return nil
		}
		Logf("Job associated to %s not created", daemonsetjob.Name)
		return fmt.Errorf("Job associated to %s not created", daemonsetjob.Name)
	}
}

// HOIsDaemonSetJobFinished is an higher order func that returns the func that checks whether a Workflow is finished
func HOIsDaemonSetJobFinished(workflowClient versioned.Interface, daemonsetjob *daemonsetjobv1.DaemonSetJob) func() error {
	return func() error {
		daemonSetJobs, err := workflowClient.DaemonsetjobV1().DaemonSetJobs(daemonsetjob.Namespace).List(metav1.ListOptions{})
		if err != nil {
			Logf("Cannot list DaemonSetJobs:%v", err)
			return err
		}
		if len(daemonSetJobs.Items) != 1 {
			return fmt.Errorf("Expected only 1 daemonSetJobs got %d", len(daemonSetJobs.Items))
		}
		if controller.IsDaemonSetJobFinished(&daemonSetJobs.Items[0]) {
			Logf("DaemonSetJob %s finished", daemonsetjob.Name)
			return nil
		}
		Logf("DaemonSetJob %s not finished", daemonsetjob.Name)
		return fmt.Errorf("daemonsetjob %s not finished", daemonsetjob.Name)
	}
}

// HOCheckAllDaemonSetJobJobsFinished is an higher order func that returns func that checks
// whether all jobs are finished
func HOCheckAllDaemonSetJobJobsFinished(workflowClient versioned.Interface, daemonsetjob *daemonsetjobv1.DaemonSetJob) func() error {
	return func() error {
		d, err := workflowClient.DaemonsetjobV1().DaemonSetJobs(daemonsetjob.Namespace).Get(daemonsetjob.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("unable to get %s/%s:%v", daemonsetjob.Namespace, daemonsetjob.Name, err)
		}
		if d.Status.CompletionTime == nil {
			return fmt.Errorf("daemonsetjob %s/%s not fully completed", d.Namespace, d.Name)
		}
		if d.Status.Active != 0 {
			return fmt.Errorf("daemonsetjob %s/%s not fully completed, jobs still active", d.Namespace, d.Name)
		}
		return nil
	}
}

// HOCheckZeroDaemonSetJobJobsWasCreated is an higher order func that returns func that checks
// that no job was created
func HOCheckZeroDaemonSetJobJobsWasCreated(workflowClient versioned.Interface, daemonsetjob *daemonsetjobv1.DaemonSetJob) func() error {
	return func() error {
		d, err := workflowClient.DaemonsetjobV1().DaemonSetJobs(daemonsetjob.Namespace).Get(daemonsetjob.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("unable to get %s/%s:%v", daemonsetjob.Namespace, daemonsetjob.Name, err)
		}

		if d.Status.Active != 0 {
			return fmt.Errorf("daemonsetjob %s/%s have jobs still active", d.Namespace, d.Name)
		}
		if d.Status.Failed != 0 {
			return fmt.Errorf("daemonsetjob %s/%s have jobs with status failed", d.Namespace, d.Name)
		}
		if d.Status.Succeeded != 0 {
			return fmt.Errorf("daemonsetjob %s/%s have jobs with status success", d.Namespace, d.Name)
		}
		return nil
	}
}
