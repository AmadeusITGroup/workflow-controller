# How to vendor

Vendoring is performed with `glide`. To avoid `k8s.io/kubernetes`vendoring, some code is copied `.../workflow-controller/pkg/api/v1/default.go`. More particulary the function `func SetDefaults_Job`. While the code should not change a lot (since `batch/v1` version is frozen), one may double check this in case of strange behaviors.

# How to generate deep-copy

First of all you need kubernetes `code-generator` executables. To install in your `$GOPATH` run:

```shell
$ go get k8s.io/code-generator/...

```

Now you can generate the deepcopy functions running something like this...

```shell
$GOPATH/bin/deepcopy-gen  --v 4 --logtostderr -i github.com/sdminonne/workflow-controller/pkg/api/v1  --bounding-dirs "k8s.io/api"
```

and the go client

```shell
$GOPATH/bin/client-gen --input-base "github.com/sdminonne/workflow-controller/pkg/api" --input "workflow/v1" --go-header-file "hack/boilerplate.go.txt"   --clientset-path "github.com/sdminonne/workflow-controller/pkg/client" --clientset-name "versioned"
```

now the listers:

```shell
${GOPATH}/bin/lister-gen --input-dirs "github.com/sdminonne/workflow-controller/pkg/api/workflow/v1" --output-package "github.com/sdminonne/workflow-controller/pkg/client/listers" -h "hack/boilerplate.go.txt"
```

now the informers:
```shell
 ${GOPATH}/bin/informer-gen --input-dirs "github.com/sdminonne/workflow-controller/pkg/api/workflow/v1" --versioned-clientset-package "github.com/sdminonne/workflow-controller/pkg/client/versioned" --listers-package "github.com/sdminonne/workflow-controller/pkg/client/listers" --output-package "github.com/sdminonne/workflow-controller/pkg/client/informers"   -h "hack/boilerplate.go.txt"
```