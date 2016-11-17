package testing

import (
	"fmt"
	"io/ioutil"

	"github.com/davecgh/go-spew/spew"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
)

// DumpWorkflow dumps a workflow in a tmp file
func DumpWorkflowInTmpFile(w *wapi.Workflow) (string, error) {
	tmpfile, err := ioutil.TempFile("", "workflow")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()
	s := spew.ConfigState{
		Indent: " ",
		// Extra deep spew.
		DisableMethods: true,
	}
	if _, err := tmpfile.Write([]byte(s.Sdump(w))); err != nil {
		return "", fmt.Errorf("cannot write in tmp file: %v", err)
	}
	return tmpfile.Name(), nil
}

// NewJobTemplatespec is an utility function to create a JobTemplateSpec
func NewJobTemplateSpec() *batch.JobTemplateSpec {
	return &batch.JobTemplateSpec{
		ObjectMeta: api.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: batch.JobSpec{
			Template: api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:            "baz",
							Image:           "foo/bar",
							Command:         []string{"sh", "-c", "echo Starting on: $(date); sleep 5; echo Goodbye cruel world at: $(date)"},
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

func newJobTemplateStatus() wapi.WorkflowStepStatus {
	return wapi.WorkflowStepStatus{
		Name:     "one",
		Complete: false,
		Reference: api.ObjectReference{
			Kind:      "Job",
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
	}
}

// NewWorkflow returns an initialized Worfklow for testing
func NewWorkflow(group, version, name, namespace string, labelSelector map[string]string) *wapi.Workflow {
	return &wapi.Workflow{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "Workflow",
			APIVersion: group + "/" + version,
		},

		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: wapi.WorkflowSpec{
			Steps: []wapi.WorkflowStep{
				{
					Name:        "one",
					JobTemplate: NewJobTemplateSpec(),
				},
				{
					Name:        "two",
					JobTemplate: NewJobTemplateSpec(),
				},
			},
			Selector: &unversioned.LabelSelector{
				MatchLabels: labelSelector,
			},
		},
		Status: wapi.WorkflowStatus{
			Conditions: []wapi.WorkflowCondition{},
			Statuses: []wapi.WorkflowStepStatus{
				newJobTemplateStatus(),
			},
		},
	}
}

// NewWorkflowList returns a list of workflows
func NewWorkflowList(group, version string) *wapi.WorkflowList {
	return &wapi.WorkflowList{
		TypeMeta: unversioned.TypeMeta{
			Kind: "WorkflowList",
		},
		Items: []wapi.Workflow{
			*NewWorkflow(group, version, "dag2", api.NamespaceDefault, nil),
			*NewWorkflow(group, version, "dag1", api.NamespaceDefault, nil),
			*NewWorkflow(group, version, "dag3", api.NamespaceDefault, nil),
		},
	}
}
