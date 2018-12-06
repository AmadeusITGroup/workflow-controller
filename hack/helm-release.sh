#!/bin/bash

if [ -z "$1" ]; then
    echo "please provide the version as parameter: ./helm-release.sh <version> <chart-name>\n"
    exit 1
fi

if [ -z "$2" ]; then
    echo "please provide the chart name as parameter: ./helm-release.sh <version> <chart-name>\n"
    exit 1
fi

cd $(git rev-parse --show-toplevel)
sed "s/tag: master/tag: $1/" chart/$2/values.yaml > charts/$2/values.tmp.yaml; mv charts/$2/values.tmp.yaml charts/$2/values.yaml
helm package --version "$1" charts/$2
mv "$2-$1.tgz" "docs/helm-$2-$1.tgz"
git checkout -- charts/$2/values.yaml
helm repo index docs --url https://amadeusitgroup.github.io/workflow-controller/ --merge docs/index.yaml
git add --all docs/