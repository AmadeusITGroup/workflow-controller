/*
Copyright 2016 The Kubernetes Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"testing"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/batch"
)

func TestDefaulting(t *testing.T) {
	workflow := &Workflow{
		ObjectMeta: kapi.ObjectMeta{
			Name:      "mydag",
			Namespace: kapi.NamespaceDefault,
		},
		Spec: WorkflowSpec{
			Steps: []WorkflowStep{
				{
					Name: "myJob",
					JobTemplate: &batch.JobTemplateSpec{
						ObjectMeta: kapi.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
						Spec: batch.JobSpec{
							Template: kapi.PodTemplateSpec{
								ObjectMeta: kapi.ObjectMeta{
									Labels: map[string]string{
										"foo": "bar",
									},
								},
								Spec: kapi.PodSpec{
									Containers: []kapi.Container{
										{
											Image: "foo/bar",
										},
									},
									RestartPolicy: "Never",
								},
							},
						},
					},
				},
			},
		},
	}
	d := NewBasicDefaulter()
	defaulted, err := d.Default(workflow)
	if err != nil {
		t.Errorf(err.Error())
	}
	for stepName, step := range defaulted.Spec.Steps {
		if step.JobTemplate != nil {
			spec := step.JobTemplate.Spec
			if spec.Completions == nil {
				t.Errorf("%s - Completion is nil", stepName)
				continue
			}
			if *(spec.Completions) != 1 {
				t.Errorf("%s - Completions should be 1", stepName)
				continue
			}
			if spec.Parallelism == nil {
				t.Errorf("%s - Parallelism is nil", stepName)
				continue
			}
			if *(spec.Parallelism) != 1 {
				t.Errorf("%s - Parallelism should be 1", stepName)
				continue
			}
			if len(spec.Template.Spec.Containers) == 0 {
				t.Errorf("%s - Missing containers", stepName)
			}
		}
	}
}
