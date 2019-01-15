// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	kclient "k8s.io/client-go/kubernetes"
	// import for auth providers plugin
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/amadeusitgroup/workflow-controller/kubectl-plugin/cmd/cronworkflow"
	"github.com/amadeusitgroup/workflow-controller/kubectl-plugin/cmd/daemonsetjob"
	"github.com/amadeusitgroup/workflow-controller/kubectl-plugin/cmd/workflow"
	wfclient "github.com/amadeusitgroup/workflow-controller/pkg/client"
	wfversioned "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
)

var cfgFile string
var kubeClient *kclient.Clientset
var workflowClient wfversioned.Interface

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "workflow",
	Short: "This plugin show Custom Resources information handled by Workflow-Controller",
	Long:  `This plugin show Custom Resources information handled by Workflow-Controller".`,
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	if err := initClients(); err != nil {
		fmt.Println("Unable to init Kubernetes API clients, err:", err)
		os.Exit(1)
	}

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kubectl-plugin.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// Register sub commands
	RootCmd.AddCommand(workflow.NewCmd(kubeClient, workflowClient))
	RootCmd.AddCommand(cronworkflow.NewCmd(kubeClient, workflowClient))
	RootCmd.AddCommand(daemonsetjob.NewCmd(kubeClient, workflowClient))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".kubectl-plugin") // name of config file (without extension)
	viper.AddConfigPath("$HOME")           // adding home directory as first search path
	viper.AutomaticEnv()                   // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func initClients() error {
	kubeconfigFilePath := getKubeConfigDefaultPath(getHomePath())
	if len(kubeconfigFilePath) == 0 {
		return fmt.Errorf("error initializing config. The KUBECONFIG environment variable must be defined")
	}

	config, err := configFromPath(kubeconfigFilePath)
	if err != nil {
		return fmt.Errorf("error obtaining kubectl config: %v", err)
	}

	rest, err := config.ClientConfig()
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	workflowClient, err = wfclient.NewWorkflowClient(rest)
	if err != nil {
		return fmt.Errorf("Unable to init workflow.clientset from kubeconfig:%v", err)
	}

	kubeClient, err = kclient.NewForConfig(rest)
	if err != nil {
		return fmt.Errorf("Unable to initialize KubeClient:%v", err)
	}

	return nil
}

func configFromPath(path string) (clientcmd.ClientConfig, error) {
	rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: path}
	credentials, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("the provided credentials %q could not be loaded: %v", path, err)
	}

	overrides := &clientcmd.ConfigOverrides{
		Context: clientcmdapi.Context{
			Namespace: os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_NAMESPACE"),
		},
	}

	context := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONTEXT")
	if len(context) > 0 {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		return clientcmd.NewNonInteractiveClientConfig(*credentials, context, overrides, rules), nil
	}
	return clientcmd.NewDefaultClientConfig(*credentials, overrides), nil
}

func getHomePath() string {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
	}

	return home
}

func getKubeConfigDefaultPath(home string) string {
	kubeconfig := filepath.Join(home, ".kube", "config")

	kubeconfigEnv := os.Getenv("KUBECONFIG")
	if len(kubeconfigEnv) > 0 {
		kubeconfig = kubeconfigEnv
	}

	configFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONFIG")
	kubeConfigFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_KUBECONFIG")
	if len(configFile) > 0 {
		kubeconfig = configFile
	} else if len(kubeConfigFile) > 0 {
		kubeconfig = kubeConfigFile
	}

	return kubeconfig
}
