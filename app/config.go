/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"github.com/spf13/pflag"
)

// Config contains configuration for workflow controller application
type Config struct {
	KubeConfigFile string
	ListenHTTPAddr string
	InstallCRDs    bool
}

// NewWorkflowControllerConfig builds and returns a workflow controller Config
func NewWorkflowControllerConfig() *Config {
	return &Config{}
}

// AddFlags add cobra flags to populate Config
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KubeConfigFile, "kubeconfig", c.KubeConfigFile, "Location of kubecfg file for access to kubernetes master service")
	fs.StringVar(&c.ListenHTTPAddr, "addr", "0.0.0.0:8086", "Listen address of the http server which serves kubernetes probes and prometheus endpoints")
	fs.BoolVar(&c.InstallCRDs, "install-crds", true, "install the CRDs used by the controller as part of startup") // to be compatible with kubebuilder https://github.com/kubernetes-sigs/kubebuilder) built controllers
}
