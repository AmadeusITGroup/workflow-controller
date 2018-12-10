# Copyright 2015 The Kubernetes Authors. All rights reserved.
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

ARTIFACT=workflow-controller

PLUGIN_ARTIFACT=kubectl-plugin
PLUGIN_PATH=./kubectl-plugin

# 0.0 shouldn't clobber any released builds
TAG= latest
PREFIX =  workflowctrl/${ARTIFACT}

SOURCES := $(shell find $(SOURCEDIR) ! -name "*_test.go" -name '*.go')

all: build


build: ${ARTIFACT} ${PLUGIN_ARTIFACT}

${ARTIFACT}: ${SOURCES}
	CGO_ENABLED=0  GOOS=linux go build -i -installsuffix cgo -ldflags '-w' -o ${ARTIFACT} ./main.go

${PLUGIN_ARTIFACT}: ${SOURCES}
	CGO_ENABLED=0 go build -i -installsuffix cgo -ldflags '-w' -o ${PLUGIN_PATH}/${PLUGIN_ARTIFACT} ${PLUGIN_PATH}/main.go

plugin: ${PLUGIN_ARTIFACT} install-plugin

install-plugin:
	./hack/install-plugin.sh

container: build
	docker build -t $(PREFIX):$(TAG) .

test:
	./go.test.sh

push: container
	docker push $(PREFIX):$(TAG)

clean:
	rm -f ${ARTIFACT}
	rm -f ${PLUGIN_PATH}/${PLUGIN_ARTIFACT}

.PHONY: build push clean test
