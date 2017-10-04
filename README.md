# workflow-controller [![Build Status](https://travis-ci.org/sdminonne/workflow-controller.svg?branch=master)](https://travis-ci.org/sdminonne/workflow-controller)
Kubernetes workflow controller

## Running workflow-controller

### locally

```shell
$ ./workflow-controller --kubeconfig=$HOME/.kube/config
```

Now you can create a Workflow resource via

```shell
$ kubectl create -f .../examples/hello_workflow/workflow.yaml
```

At this point the workflow-controller will start to handle the jobs.


### in a kubernetes pod


To run `workflow-controller` in a Kubernetes pod you shoud run these commands

```shell
$ kubectl create -f .../deployment/k8s/workflow-controller_serviceaccount.yaml
$ kubectl create -f .../deployment/k8s/workflow-controller_deployment.yaml
```
Then you may want to test a workflow example like this:

```shell
$ kubectl create -f  .../examples/hello_workflow/workflow.yaml
```

### in an openshift cluster
TODO
