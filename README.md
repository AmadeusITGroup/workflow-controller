# workflow-controller
[![Build Status](https://travis-ci.org/sdminonne/workflow-controller.svg?branch=master)](https://travis-ci.org/sdminonne/workflow-controller)
[![Go Report Card](https://goreportcard.com/badge/github.com/sdminonne/workflow-controller)](https://goreportcard.com/report/github.com/sdminonne/workflow-controller)
[![codecov](https://codecov.io/gh/sdminonne/workflow-controller/branch/master/graph/badge.svg)](https://codecov.io/gh/sdminonne/workflow-controller)
![DopeBadge](https://img.shields.io/badge/Hightower-dope-C0C0C0.svg)

A simple Kubernetes workflow controller. TODO: add more explanations.

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


To run `workflow-controller` in a Kubernetes pod you should run this command

```shell
$ kubectl create -f .../deployment/k8s/workflow-controller_allinone.yaml
```
Then you may want to test a workflow example like this:

```shell
$ kubectl create -f  .../examples/hello_workflow/workflow.yaml
```

### in an openshift cluster
TODO
