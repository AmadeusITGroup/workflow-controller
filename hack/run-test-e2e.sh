#!/usr/bin/env bash
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail
set -x

export ROOT=$(dirname "${BASH_SOURCE}")/..
command -v kubectl >/dev/null 2>&1 || { echo >&2 "kubectl not found in path. Aborting."; exit 1; }

pushd ${ROOT}/test/e2e
echo "compile tests..."
go test -c .
popd

#TODO:
#  - check if kubeconfig is there
#  - check if cluster is up

./test/e2e/e2e.test --kubeconfig=$HOME/.kube/config --ginkgo.slowSpecThreshold 60
