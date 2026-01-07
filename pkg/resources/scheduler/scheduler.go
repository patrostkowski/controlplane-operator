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

package scheduler

import (
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/apiserver"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources returns ConfigMap + Deployment for kube-scheduler.
func Resources(mcp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildConfigMap(mcp),
		buildDeployment(mcp),
	}
}

func buildConfigMap(mcp *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	p := pki.New(mcp).Scheduler()

	ns := mcp.Namespace
	kcfg := utils.BuildComponentKubeconfig(
		ns,
		apiserver.KubeAPIServerSvcName,
		apiserver.KubeAPIServerSecurePort,
		"scheduler",
		p.ClientCA,
		p.Client,
	)

	kubeconfigData, err := clientcmd.Write(*kcfg)
	if err != nil {
		panic(err)
	}

	return builders.NewConfigMap().
		WithName(cmKubeconfigName).
		WithNamespace(ns).
		Put(cmKubeconfigKey, string(kubeconfigData)).
		Build()
}

func buildDeployment(mcp *mcpv1alpha1.ManagedControlPlane) *appsv1.Deployment {
	p := pki.New(mcp).Scheduler()

	ns := mcp.Namespace
	version := mcp.Spec.Version

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
		Image:           "registry.k8s.io/kube-scheduler:" + version,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"kube-scheduler"},
		Args: []string{
			"--bind-address=0.0.0.0",
			"--kubeconfig=" + kubeconfigPath,
			"--authentication-kubeconfig=" + kubeconfigPath,
			"--authorization-kubeconfig=" + kubeconfigPath,
			"--leader-elect=true",
			"--logging-format=json",
		},
		Ports: []corev1.ContainerPort{
			{Name: "https", ContainerPort: securePort},
		},
		LivenessProbe:  utils.HttpsHealthProbe(securePort, common.LivezPath, 10, 10, 10, 10),
		ReadinessProbe: utils.HttpsHealthProbe(securePort, common.ReadyzPath, 5, 5, 5, 5),
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
			append(p.Mounts(true),
				corev1.VolumeMount{
					Name:      common.KubeconfigVolumeName,
					MountPath: kubeconfigMountDir,
					ReadOnly:  true,
				},
			)...,
		).
		Build()
}
