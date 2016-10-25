# workflow-controller
Kubernetes workflow controller

## Running workflow-controller

### locally

```shell
$ ./workflow-controller --kubeconfig=$HOME/.kube/config --resource-versions=v1 --domain=example.com --name=workflow
```

Now you can create a Workflow resource via

```shell
$ ./cluster/kubectl.sh create -f .../examples/hello_workflow/workflow.yaml
```

At this point the workflow-controller will start to handle the jobs.


### in a kubernetes pod

```shell
$ make container && make push
$ cd example/workflow-controller_in_pod
$ kubectl create -f workflow-controller-serviceaccount.yaml
serviceaccount "workflow-controller" created
$ kubectl create -f workflow-controller-deployment.yaml
deployment "workflow-controller-deployment" created
```


```shell
$ make container && make push
$ cd example/workflow-controller_in_pod
$ kubectl create -f workflow-controller-serviceaccount.yaml
serviceaccount "workflow-controller" created
$ kubectl create -f workflow-controller-deployment.yaml
deployment "workflow-controller-deployment" created
```
