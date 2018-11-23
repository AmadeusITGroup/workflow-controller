#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ${GOPATH}/src/k8s.io/code-generator)}


vendor/k8s.io/code-generator/generate-groups.sh all \
  github.com/amadeusitgroup/workflow-controller/pkg/client github.com/amadeusitgroup/workflow-controller/pkg/api \
  "workflow:v1 cronworkflow:v1 daemonsetjob:v1" \
  --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.go.txt
