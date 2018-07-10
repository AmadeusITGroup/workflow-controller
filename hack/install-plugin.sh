#!/usr/bin/env bash

KUBE_CONFIG_PATH=$HOME"/.kube"
if [ -n "$1" ]
then KUBE_CONFIG_PATH=$1
fi

WORKFLOW_PLUGIN_BIN_PATH="kubectl-plugin"
WORKFLOW_PLUGIN_BIN_NAME="kubectl-plugin"
WORKFLOW_PLUGIN_PATH=$KUBE_CONFIG_PATH/plugins/workflow

mkdir -p $WORKFLOW_PLUGIN_PATH

GIT_ROOT=$(git rev-parse --show-toplevel)
cp $GIT_ROOT/$WORKFLOW_PLUGIN_BIN_PATH/$WORKFLOW_PLUGIN_BIN_NAME $WORKFLOW_PLUGIN_PATH/$WORKFLOW_PLUGIN_BIN_NAME

cat > $WORKFLOW_PLUGIN_PATH/plugin.yaml << EOF1
name: "workflow"
shortDesc: "workflow shows workflow custom resources"
longDesc: >
  workflow shows workflow custom resources
command: ./kubectl-plugin
EOF1