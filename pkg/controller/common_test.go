package controller

import (
	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// utility function to create a JobTemplateSpec
func newJobTemplateSpec() *batchv2.JobTemplateSpec {
	return &batchv2.JobTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: batch.JobSpec{
			Template: api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: api.PodSpec{
					RestartPolicy: "Never",
					Containers: []api.Container{
						{Image: "foo/bar"},
					},
				},
			},
		},
	}
}
