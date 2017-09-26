# How to vendor

Vendoring is performed with `glide`. To avoid `k8s.io/kubernetes`vendoring, some code is copied `.../workflow-controller/pkg/api/v1/default.go`. More particulary the function `func SetDefaults_Job`. While the code should not change a lot (since `batch/v1` version is frozen), one may double check this in case of strange behaviors.

# How to generate deep-copy

First of all you need `deepcopy-gen`. You can retrieve directy from `gengo` package via a simple:

```shell
$ go get go get k8s.io/gengo
```

Now you can generate the deepcopy functions running something like this...

```shell
$GOPATH/bin/deepcopy-gen  --v 4 --logtostderr -i github.com/sdminonne/workflow-controller/pkg/api/v1  --bounding-dirs "k8s.io/api"
```
