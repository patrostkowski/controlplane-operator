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

package cluster

import (
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
)

// ControllerManagerSpec describes the inputs required to render controller-manager resources for a control plane.
type ControllerManagerSpec interface {
	Namespacer
	MountLayout

	GetManagedControlPlaneSpec() mcpv1alpha1.ManagedControlPlaneSpec

	ControllerManager() ControllerManagerConfig
	APIServer() APIServerConfig
}

// ControllerManagerConfig exposes stable names of controller-manager resources and secrets.
type ControllerManagerConfig interface {
	DeploymentName() string
	KubeconfigConfigMapName() string
	ClusterCASecret() string
	ClientCertSecret() string
	SASignerSecret() string
}

// Compile-time check ensuring controllerManager implements ControllerManagerConfig.
var _ ControllerManagerConfig = controllerManager{}

// ControllerManager returns a names-only facade for controller-manager-related resources.
func (cc ClusterContext) ControllerManager() ControllerManagerConfig {
	return controllerManager{cc: cc}
}

// controllerManager is an internal implementation of ControllerManagerConfig backed by ClusterContext.
type controllerManager struct {
	cc ClusterContext
}

// DeploymentName returns the name of the kube-controller-manager Deployment.
func (cm controllerManager) DeploymentName() string {
	return "kube-controller-manager"
}

// KubeconfigConfigMapName returns the name of the ConfigMap containing the controller-manager kubeconfig.
func (cm controllerManager) KubeconfigConfigMapName() string {
	return "kube-controller-manager-kubeconfig"
}

// ClusterCASecret returns the Secret name holding the cluster CA certificate.
func (cm controllerManager) ClusterCASecret() string {
	return cm.cc.PKI().Certificate().ManagedCA()
}

// ClientCertSecret returns the Secret name holding the controller-manager client certificate.
func (cm controllerManager) ClientCertSecret() string {
	// this is the CM client certificate secret you create in PKI
	return cm.cc.PKI().Certificate().CMClient()
}

// SASignerSecret returns the Secret name holding the service-account signing key pair.
func (cm controllerManager) SASignerSecret() string {
	return cm.cc.PKI().Certificate().SASigner()
}
