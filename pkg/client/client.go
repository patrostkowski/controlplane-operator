// Copyright 2025 Patryk Rostkowski
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

package client

import (
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	Kubeconfig string
	Context    string
	client     client.Client
}

func NewClient(ctx, kubeconfig string) (client.Client, error) {
	c := Config{
		Kubeconfig: kubeconfig,
		Context:    ctx,
	}

	client, err := c.buildConfig()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Config) buildConfig() (client.Client, error) {
	if c.client != nil {
		return c.client, nil
	}

	// build kube cfg
	cfg, err := c.buildRESTConfig()
	if err != nil {
		return nil, err
	}

	// add core objects
	scheme := runtime.NewScheme()

	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}

	if err := mcpv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	kClient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	c.client = kClient

	return c.client, nil
}

func (c *Config) buildRESTConfig() (*rest.Config, error) {
	loading := clientcmd.NewDefaultClientConfigLoadingRules()

	if c.Kubeconfig != "" {
		loading.ExplicitPath = c.Kubeconfig
	}

	overrides := &clientcmd.ConfigOverrides{}
	if c.Context != "" {
		overrides.CurrentContext = c.Context
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loading,
		overrides,
	).ClientConfig()
}
