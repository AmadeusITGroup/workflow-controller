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
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/pflag"
)

// Config contains configuration for workflow controller application
type Config struct {
	KubeConfigFile   string
	KubeMasterURL    string
	ResourceDomain   string
	ResourceName     string
	ResourceVersions []string
}

type kubeMasterURLVar struct {
	val *string
}

func (m kubeMasterURLVar) Set(v string) error {
	parsedURL, err := url.Parse(os.ExpandEnv(v))
	if err != nil {
		return fmt.Errorf("failed to parse kube-master-url")
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" || parsedURL.Host == ":" {
		return fmt.Errorf("invalid kube-master-url specified")
	}
	*m.val = v
	return nil
}

// String implements Stringer interface
func (m kubeMasterURLVar) String() string {
	return *m.val
}

// Type return the type of kubeMasterURLVar
func (m kubeMasterURLVar) Type() string {
	return "string"
}

// NewWorkflowControllerConfig builds and returns a workflow controller Config
func NewWorkflowControllerConfig() *Config {
	return &Config{}
}

// AddFlags add cobra flags to populate Config
func (c *Config) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.KubeConfigFile, "kubeconfig", c.KubeConfigFile, "Location of kubecfg file for access to kubernetes master service; --kube-master-url overrides the URL part of this; if neither this nor --kube-master-url are provided, defaults to service account tokens")
	fs.Var(kubeMasterURLVar{&c.KubeMasterURL}, "kube-master-url", "URL to reach kubernetes master. Env variables in this flag will be expanded.")
	fs.StringVar(&c.ResourceDomain, "domain", c.ResourceDomain, "Domain name where the resource will be installed.")
	fs.StringVar(&c.ResourceName, "name", c.ResourceName, "Resource name, it must not be already stored in kubernetes")
	fs.StringSliceVar(&c.ResourceVersions, "resource-versions", c.ResourceVersions, "List of resource versions")
}
