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

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KonnectivityAgentResources returns only the Daemonset for konnectivity-agent
func KonnectivityAgentResources(mcp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildKonnectivityAgentDaemonSet(mcp),
	}
}

const (
	konnectivityAgentVolumeName = "agent-certs"
)

var KonnecivityAgentVersion = "v0.1.3"

func buildKonnectivityAgentDaemonSet(mcp *mcpv1alpha1.ManagedControlPlane) *appsv1.DaemonSet {
	p := pki.New(mcp).KonnectivityAgentView()
	policy := corev1.DNSClusterFirstWithHostNet
	labels := map[string]string{
		"app": konnectivityAgentName,
	}
	caPath := filepath.Join(p.KonnectivityAgent.MountDir, p.KonnectivityCA.CAFile)

	c := corev1.Container{
		Name: konnectivityAgentName,
		// TODO: make it compatible with k8s-api version
		Image: "registry.k8s.io/kas-network-proxy/proxy-agent:" + KonnecivityAgentVersion,
		Args: []string{
			"--proxy-server-host=" + mcp.Status.Address,
			"--proxy-server-port=" + utils.PortString(konnectivityServerPort),

			"--ca-cert=" + caPath,
			"--agent-cert=" + p.KonnectivityAgent.CertPath(),
			"--agent-key=" + p.KonnectivityAgent.KeyPath(),
			"--v=2",
		},
	}

	volume := corev1.Volume{
		Name: konnectivityAgentVolumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: common.KonnectivityAgentTLSSecretName,
							},
							Items: []corev1.KeyToPath{
								{
									Key:  p.KonnectivityAgent.CertFile,
									Path: p.KonnectivityAgent.CertFile,
								},
								{
									Key:  p.KonnectivityAgent.KeyFile,
									Path: p.KonnectivityAgent.KeyFile,
								},
							},
						},
					},
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: common.KonnectivityCASecretName,
							},
							Items: []corev1.KeyToPath{
								{
									Key:  p.KonnectivityCA.CAFile,
									Path: p.KonnectivityCA.CAFile,
								},
							},
						},
					},
				},
			},
		},
	}

	volumeMounts := corev1.VolumeMount{
		Name:      konnectivityAgentVolumeName,
		MountPath: p.KonnectivityAgent.MountDir,
	}

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
