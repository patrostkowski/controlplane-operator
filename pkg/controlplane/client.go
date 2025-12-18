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

package controlplane

import (
	"context"
	"fmt"

	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControlPlaneClient struct {
	client.Client
	Discovery discovery.DiscoveryInterface
	REST      *rest.Config
}

func New(c client.Client, d discovery.DiscoveryInterface, r *rest.Config) *ControlPlaneClient {
	return &ControlPlaneClient{
		Client:    c,
		Discovery: d,
		REST:      r,
	}
}

func NewFromKubeconfigSecret(
	ctx context.Context,
	mgmt client.Reader,
	scheme *runtime.Scheme,
	secretNS string,
) (*ControlPlaneClient, error) {

	sec := &corev1.Secret{}
	if err := mgmt.Get(ctx, client.ObjectKey{Namespace: secretNS, Name: common.AdminConfigName}, sec); err != nil {
		return nil, err
	}

	kubeconfigBytes, ok := sec.Data[common.AdminConfigKubeconfigKey]
	if !ok || len(kubeconfigBytes) == 0 {
		return nil, fmt.Errorf("secret %s/%s missing key %q or it is empty", secretNS, common.AdminConfigName, common.AdminConfigKubeconfigKey)
	}

	restCfg, err := restConfigFromKubeconfigBytes(kubeconfigBytes)
	if err != nil {
		return nil, err
	}

	restCfg.QPS = 20
	restCfg.Burst = 40

	c, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	d, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	return New(c, d, restCfg), nil
}

func restConfigFromKubeconfigBytes(kubeconfig []byte) (*rest.Config, error) {
	// clientcmd handles clusters/users/contexts in kubeconfig properly.
	cfg, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}
	restCfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}
	return restCfg, nil
}
