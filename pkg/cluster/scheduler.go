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

// SchedulerSpec defines the inputs required to render scheduler resources for a control plane.
type SchedulerSpec interface {
	Namespacer
	MountLayout

	GetManagedControlPlaneSpec() mcpv1alpha1.ManagedControlPlaneSpec

	Scheduler() SchedulerConfig
	APIServer() APIServerConfig
}

// SchedulerConfig exposes stable names of scheduler resources and secrets.
type SchedulerConfig interface {
	DeploymentName() string
	KubeconfigConfigMapName() string
	ClusterCASecret() string
	ClientCertSecret() string
}

var _ SchedulerConfig = scheduler{}

// Scheduler returns a names-only facade for scheduler-related resources.
func (cc ClusterContext) Scheduler() SchedulerConfig {
	return scheduler{cc: cc}
}

// scheduler is an internal struct that implements the SchedulerConfig interface.
type scheduler struct {
	cc ClusterContext
}

// DeploymentName returns the name of the kube-scheduler Deployment.
func (s scheduler) DeploymentName() string {
	return "kube-scheduler"
}

// KubeconfigConfigMapName returns the name of the ConfigMap containing the scheduler kubeconfig.
func (s scheduler) KubeconfigConfigMapName() string {
	return "kube-scheduler-kubeconfig"
}

// ClusterCASecret returns the Secret name holding the cluster CA certificate for the scheduler.
func (s scheduler) ClusterCASecret() string {
	return s.cc.PKI().Certificate().ManagedCA()
}

// ClientCertSecret returns the Secret name holding the scheduler client certificate.
func (s scheduler) ClientCertSecret() string {
	return s.cc.PKI().Certificate().SchedulerClient()
}
