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
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type builder struct{ cc cluster.SchedulerSpec }

func NewBuilder(cc cluster.SchedulerSpec) cluster.ObjectProducer {
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
	s := cc.Scheduler()
	a := cc.APIServer()
	ns := cc.Namespace()

	clusterCA := s.ClusterCASecret()
	schedulerClient := s.ClientCertSecret()

	kcfg := utils.BuildComponentKubeconfig(
		ns,
		a.ServiceName(),
		apiserverSecurePort,
		"scheduler",
		cc.CertPath(clusterCA),
		cc.CertPath(schedulerClient),
		cc.KeyPath(schedulerClient),
	)

	kubeconfigData, err := clientcmd.Write(*kcfg)
	if err != nil {
		panic(err)
	}

	return builders.NewConfigMap().
		WithName(s.KubeconfigConfigMapName()).
		WithNamespace(ns).
		Put(schedulerKubeconfigKey, string(kubeconfigData)).
		Build()
}

func (b builder) buildDeployment() *appsv1.Deployment {
	cc := b.cc
	s := cc.Scheduler()
	ns := cc.Namespace()

	spec := cc.GetManagedControlPlaneSpec()
	version := spec.Kubernetes.Version

	labels := map[string]string{labelKeyApp: labelValApp}
	var replicas int32 = 1

	// PKI secret names via cc.Scheduler()
	clusterCA := s.ClusterCASecret()
	schedulerClient := s.ClientCertSecret()

	kubeconfigVol := utils.ConfigMapVolume(
		kubeconfigVolumeName,
		s.KubeconfigConfigMapName(),
		schedulerKubeconfigKey,
		schedulerKubeconfigFileName,
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
		LivenessProbe:  utils.HttpsHealthProbe(securePort, livezPath, 10, 10, 10, 10),
		ReadinessProbe: utils.HttpsHealthProbe(securePort, readyzPath, 5, 5, 5, 5),
	}

	secretVolumes := []corev1.Volume{
		cc.SecretVolume(clusterCA),
		cc.SecretVolume(schedulerClient),
	}
	secretMounts := []corev1.VolumeMount{
		cc.SecretMount(clusterCA, true),
		cc.SecretMount(schedulerClient, true),
	}

	return builders.NewDeployment().
		WithName(s.DeploymentName()).
		WithNamespace(ns).
		WithLabels(labels).
		WithSelector(labels).
		WithReplicas(replicas).
		WithContainer(c).
		AddVolumes(append([]corev1.Volume{kubeconfigVol}, secretVolumes...)...).
		AddVolumeMounts(c.Name,
			append([]corev1.VolumeMount{
				{
					Name:      kubeconfigVolumeName,
					MountPath: kubeconfigMountDir,
					ReadOnly:  true,
				},
			}, secretMounts...)...,
		).
		Build()
}
