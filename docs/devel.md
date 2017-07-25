# How to generate deep-copy

First of all you need `deepcopy-gen`. You can get directy from `gengo` package via a simple:

```shell
$ go get go get k8s.io/gengo
```

Now you can generate the deepcopy functions running something like this...

```shell
$GOPATH/bin/deepcopy-gen  --v 4 --logtostderr -i github.com/sdminonne/workflow-controller/pkg/api/v1  --bounding-dirs "k8s.io/api"
```
