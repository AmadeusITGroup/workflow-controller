#!/bin/bash

if [ -z "$1" ]; then
    echo "please provide the version as parameter: ./helm-release.sh <version>\n"
    exit 1
fi

cd $(git rev-parse --show-toplevel)
helm package --version "$1" charts/workflow-controller
mv "helm-test-$1.tgz" docs/
helm repo index docs --url https://sdminonne.github.io/workflow-controller/ --merge docs/index.yaml
git add --all docs/
