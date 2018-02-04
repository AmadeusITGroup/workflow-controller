# How to vendor

Vendoring is performed with `glide`. To avoid `k8s.io/kubernetes`vendoring, some code is copied `.../workflow-controller/pkg/api/v1/default.go`. More particulary the function `func SetDefaults_Job`. While the code should not change a lot (since `batch/v1` version is frozen), one may double check this in case of strange behaviors.

# How to generate code

```shell
$ ./hack/update-codegen.sh

```
