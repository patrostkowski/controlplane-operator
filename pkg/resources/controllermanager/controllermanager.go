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

package controllermanager

import (
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type builder struct{ cc *cluster.ClusterContext }

func NewBuilder(cc *cluster.ClusterContext) cluster.ObjectProducer {
	return builder{cc: cc}
}

func (b builder) Objects() []client.Object {
	return []client.Object{
		b.buildConfigMap(),
		b.buildDeployment(),
	}
}

func (b builder) buildConfigMap() *corev1.ConfigMap {
	cc := b.cc
	mcp := cc.MCP
	ns := mcp.Namespace

	clusterCA := cc.Names.SecretManagedCAName()
	cmClient := cc.Names.SecretControllerManagerClientName()

	kcfg := utils.BuildComponentKubeconfig(
		ns,
		cc.Names.APIServerServiceName(),
		cc.Contract.APIServer.SecurePort,
		"cm",
		cc.CertPath(clusterCA),
		cc.CertPath(cmClient),
		cc.KeyPath(cmClient),
	)

	kubeconfigData, err := clientcmd.Write(*kcfg)
	if err != nil {
		panic(err)
	}

	return builders.NewConfigMap().
		WithName(cc.Names.ControllerManagerKubeconfigConfigMapName()).
		WithNamespace(ns).
		Put(cmKubeconfigKey, string(kubeconfigData)).
		Build()
}

func (b builder) buildDeployment() *appsv1.Deployment {
	cc := b.cc
	mcp := cc.MCP
	ns := mcp.Namespace
	version := mcp.Spec.Kubernetes.Version

	labels := map[string]string{cc.Keys.LabelKeyApp: labelValApp}
	var replicas int32 = 1

	// PKI secret names
	clusterCA := cc.Names.SecretManagedCAName()
	cmClient := cc.Names.SecretControllerManagerClientName()
	saSigner := cc.Names.SecretServiceAccountSignerName()

	kubeconfigVol := utils.ConfigMapVolume(
		cc.Volumes.Kubeconfig,
		cc.Names.ControllerManagerKubeconfigConfigMapName(),
		cmKubeconfigKey,
		cmKubeconfigFileName,
	)

	c := corev1.Container{
		Name:            containerName,
		Image:           "registry.k8s.io/kube-controller-manager:" + version,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"kube-controller-manager"},
		Args: []string{
			"--bind-address=0.0.0.0",

			"--cluster-name=" + mcp.GetObjectMeta().GetName(),

			// kubeconfig wiring
			"--kubeconfig=" + kubeconfigPath,
			"--authentication-kubeconfig=" + kubeconfigPath,
			"--authorization-kubeconfig=" + kubeconfigPath,

			// leader election + sa creds
			"--leader-elect=true",
			"--use-service-account-credentials=true",
			"--controllers=*,bootstrapsigner,tokencleaner",

			// service account signing key
			"--service-account-private-key-file=" + cc.KeyPath(saSigner),

			// cluster signing
			"--cluster-signing-cert-file=" + cc.CertPath(clusterCA),
			"--cluster-signing-key-file=" + cc.KeyPath(clusterCA),
			"--feature-gates=RotateKubeletServerCertificate=true",

			// CA wiring
			"--client-ca-file=" + cc.CertPath(clusterCA),
			"--root-ca-file=" + cc.CertPath(clusterCA),

			// networking
			"--cluster-cidr=" + mcp.Spec.Kubernetes.Networking.PodCIDR,
			"--allocate-node-cidrs=true",

			"--logging-format=json",
		},
		Ports: []corev1.ContainerPort{
			{Name: "https", ContainerPort: securePort},
		},
		StartupProbe:   utils.HttpsHealthProbe(securePort, cc.Keys.HealthPath, 10, 10, 15, 24),
		LivenessProbe:  utils.HttpsHealthProbe(securePort, cc.Keys.HealthPath, 10, 10, 15, 8),
		ReadinessProbe: utils.HttpsHealthProbe(securePort, cc.Keys.HealthPath, 5, 5, 15, 3),
	}

	secretVolumes := []corev1.Volume{
		cc.SecretVolume(clusterCA),
		cc.SecretVolume(cmClient),
		cc.SecretVolume(saSigner),
	}

	secretMounts := []corev1.VolumeMount{
		cc.SecretMount(clusterCA, true),
		cc.SecretMount(cmClient, true),
		cc.SecretMount(saSigner, true),
	}

	return builders.NewDeployment().
		WithName(cc.Names.ControllerManagerDeploymentName()).
		WithNamespace(ns).
		WithLabels(labels).
		WithSelector(labels).
		WithReplicas(replicas).
		WithContainer(c).
		AddVolumes(append([]corev1.Volume{kubeconfigVol}, secretVolumes...)...).
		AddVolumeMounts(c.Name,
			append([]corev1.VolumeMount{
				{
					Name:      cc.Volumes.Kubeconfig,
					MountPath: kubeconfigMountDir,
					ReadOnly:  true,
				},
			}, secretMounts...)...,
		).
		Build()
}
