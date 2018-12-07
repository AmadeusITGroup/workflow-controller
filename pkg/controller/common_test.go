package controller

import (
	"testing"

	batch "k8s.io/api/batch/v1"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_compareJobSpecMD5Hash(t *testing.T) {
	jobSpec1 := batch.JobSpec{Template: api.PodTemplateSpec{Spec: api.PodSpec{Containers: []api.Container{{Name: "foo", Image: "foo:3.0.3"}}}}}
	jobSpec2 := batch.JobSpec{Template: api.PodTemplateSpec{Spec: api.PodSpec{Containers: []api.Container{{Name: "foo", Image: "foo:4.0.8"}}}}}
	jobSpec3 := batch.JobSpec{Template: api.PodTemplateSpec{Spec: api.PodSpec{Containers: []api.Container{{Name: "foo", Image: "foo:3.0.3"}, {Name: "bar", Image: "bar:latest"}}}}}
	hashspec1, _ := generateMD5JobSpec(&jobSpec1)
	t.Logf("hashspec1:%s", hashspec1)
	hashspec2, _ := generateMD5JobSpec(&jobSpec2)
	t.Logf("hashspec2:%s", hashspec2)
	hashspec3, _ := generateMD5JobSpec(&jobSpec3)
	t.Logf("hashspec3:%s", hashspec3)

	type args struct {
		hash string
		job  *batch.Job
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "similar JobSpec",
			args: args{
				hash: hashspec1,
				job: &batch.Job{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{DaemonSetJobMD5AnnotationKey: hashspec1},
					},
					Spec: jobSpec1,
				},
			},
			want: true,
		},
		{
			name: "different JobSpec",
			args: args{
				hash: hashspec1,
				job: &batch.Job{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{DaemonSetJobMD5AnnotationKey: hashspec2},
					},
					Spec: jobSpec2,
				},
			},
			want: false,
		},
		{
			name: "PodTemplate changed",
			args: args{
				hash: hashspec1,
				job: &batch.Job{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{DaemonSetJobMD5AnnotationKey: string(hashspec3)},
					},
					Spec: jobSpec2,
				},
			},
			want: false,
		},
		{
			name: "missing annotation",
			args: args{
				hash: hashspec2,
				job: &batch.Job{
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       jobSpec2,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareJobSpecMD5Hash(tt.args.hash, tt.args.job); got != tt.want {
				t.Errorf("compareJobSpecMD5Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}
