#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -x

#deep copy
$GOPATH/bin/deepcopy-gen --input-dirs "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1" -p github.com/amadeusitgroup/workflow-controller/pkg/api/ --bounding-dirs github.com/amadeusitgroup/workflow-controller/pkg/api -h $GOPATH/src/github.com/amadeusitgroup/workflow-controller/hack/boilerplate.go.txt

#client
$GOPATH/bin/client-gen --input-base "github.com/amadeusitgroup/workflow-controller/pkg/api" --input "workflow/v1" --go-header-file "hack/boilerplate.go.txt"   --clientset-path "github.com/amadeusitgroup/workflow-controller/pkg/client" --clientset-name "versioned"

#listers
${GOPATH}/bin/lister-gen --input-dirs "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1" --output-package "github.com/amadeusitgroup/workflow-controller/pkg/client/listers" -h "hack/boilerplate.go.txt"


#informers
${GOPATH}/bin/informer-gen --input-dirs "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1" --versioned-clientset-package "github.com/amadeusitgroup/workflow-controller/pkg/client/versioned" --listers-package "github.com/amadeusitgroup/workflow-controller/pkg/client/listers" --output-package "github.com/amadeusitgroup/workflow-controller/pkg/client/informers"   -h "hack/boilerplate.go.txt"
