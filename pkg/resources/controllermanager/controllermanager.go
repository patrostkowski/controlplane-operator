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
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources returns ConfigMap + Deployment for kube-controller-manager.
func Resources(cm *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildConfigMap(cm),
		buildDeployment(cm),
	}
}

func buildConfigMap(cm *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	p := pki.New(cm).ControllerManager()

	ns := cm.Namespace
	kcfg := utils.BuildComponentKubeconfig(
		ns,
		common.KubeAPIServerName,
		common.KubeAPISecurePort,
		"cm",
		p.ClientCA,
		p.Client,
	)

	kubeconfigData, err := clientcmd.Write(*kcfg)
	if err != nil {
		panic(err) // should never happen
	}

	return builders.NewConfigMap().
		WithName(cmKubeconfigName).
		WithNamespace(ns).
		Put(cmKubeconfigKey, string(kubeconfigData)).
		Build()
}

func buildDeployment(cm *mcpv1alpha1.ManagedControlPlane) *appsv1.Deployment {
	p := pki.New(cm).ControllerManager()

	ns := cm.Namespace
	version := cm.Spec.Version

	labels := map[string]string{common.LabelKeyApp: labelValApp}

	var replicas int32 = 1

	kubeconfigVol := utils.ConfigMapVolume(
		common.KubeconfigVolumeName,
		cmKubeconfigName,
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

			"--cluster-name=" + cm.GetObjectMeta().GetName(),

			// kubeconfig wiring
			"--kubeconfig=" + kubeconfigPath,
			"--authentication-kubeconfig=" + kubeconfigPath,
			"--authorization-kubeconfig=" + kubeconfigPath,

			// leader election + sa creds
			"--leader-elect=true",
			"--use-service-account-credentials=true",
			"--controllers=*,bootstrapsigner,tokencleaner",

			// service account signing key
			"--service-account-private-key-file=" + p.ServiceAccountSigner.KeyPath(),

			// cluster signing
			"--cluster-signing-cert-file=" + p.ClientCA.CertPath(),
			"--cluster-signing-key-file=" + p.ClientCA.KeyPath(),
			// CA wiring
			"--client-ca-file=" + p.ClientCA.CertPath(),
			"--root-ca-file=" + p.ClientCA.CertPath(),

			// networking
			"--cluster-cidr=" + cm.Spec.Networking.PodCIDR,
			"--allocate-node-cidrs=true",

			"--logging-format=json",
		},
		Ports: []corev1.ContainerPort{
			{Name: "https", ContainerPort: securePort},
		},
		StartupProbe:   utils.HttpsHealthProbe(securePort, common.HealthPath, 10, 10, 15, 24),
		LivenessProbe:  utils.HttpsHealthProbe(securePort, common.HealthPath, 10, 10, 15, 8),
		ReadinessProbe: utils.HttpsHealthProbe(securePort, common.HealthPath, 5, 5, 15, 3),
	}

	return builders.NewDeployment().
		WithName(componentName).
		WithNamespace(ns).
		WithLabels(labels).
		WithSelector(labels).
		WithReplicas(replicas).
		WithContainer(c).
		AddVolumes(append([]corev1.Volume{kubeconfigVol}, p.Volumes()...)...).
		AddVolumeMounts(c.Name,
			append([]corev1.VolumeMount{
				{
					Name:      common.KubeconfigVolumeName,
					MountPath: kubeconfigMountDir,
					ReadOnly:  true,
				},
			}, p.Mounts(true)...)...,
		).
		Build()
}
