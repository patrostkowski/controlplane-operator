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

package addons

import (
	"path/filepath"

	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KonnectivityAgentResources returns only the Daemonset for konnectivity-agent
func (e addonsBuilder) buildKonnectivityAgent() []client.Object {
	return []client.Object{
		e.buildKonnectivityAgentDaemonSet(),
	}
}

const (
	konnectivityAgentVolumeName = "agent-certs"
)

var KonnecivityAgentVersion = "v0.1.3"

func (e addonsBuilder) buildKonnectivityAgentDaemonSet() *appsv1.DaemonSet {
	cc := e.cc
	mcp := cc.MCP

	k := cc.Contract.Konnectivity

	agentName := k.AgentName
	volumeName := konnectivityAgentVolumeName
	mountDir := "/var/run/konnectivity"

	// secrets copied into managed cluster namespace (names must match what you copy)
	agentTLS := cc.Names.SecretKonnectivityAgentTLSName()
	konCA := cc.Names.SecretKonnectivityCAName()

	labels := map[string]string{
		"app": agentName,
	}

	// file names / keys from cc.Keys
	caPath := filepath.Join(mountDir, cc.Keys.CACrt)
	agentCertPath := filepath.Join(mountDir, cc.Keys.TLSCrt)
	agentKeyPath := filepath.Join(mountDir, cc.Keys.TLSKey)

	c := corev1.Container{
		Name: konnectivityAgentName,
		// TODO: make it compatible with k8s-api version
		Image: "registry.k8s.io/kas-network-proxy/proxy-agent:" + KonnecivityAgentVersion,
		Args: []string{
			"--proxy-server-host=" + mcp.Status.Address,
			"--proxy-server-port=" + utils.PortString(konnectivityServerPort),

			"--ca-cert=" + caPath,
			"--agent-cert=" + agentCertPath,
			"--agent-key=" + agentKeyPath,
			"--v=2",
		},
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: agentTLS},
							Items: []corev1.KeyToPath{
								{Key: cc.Keys.TLSCrt, Path: cc.Keys.TLSCrt},
								{Key: cc.Keys.TLSKey, Path: cc.Keys.TLSKey},
							},
						},
					},
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: konCA},
							Items: []corev1.KeyToPath{
								{Key: cc.Keys.CACrt, Path: cc.Keys.CACrt},
							},
						},
					},
				},
			},
		},
	}

	volumeMounts := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountDir,
		ReadOnly:  true,
	}

	policy := corev1.DNSClusterFirstWithHostNet

	return builders.NewDaemonSet().
		WithName(konnectivityAgentDSName).
		WithNamespace(konnectivityAgentNamespace).
		WithLabels(labels).
		WithSelector(labels).
		WithPodLabels(labels).
		WithContainer(c).
		WithDNSPolicy(policy).
		WithHostNetwork().
		AddVolumes(volume).
		AddVolumeMounts(c.Name, volumeMounts).
		Build()
}
