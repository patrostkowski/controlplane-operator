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
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubeProxyName          = "kube-proxy"
	kubeProxyNamespaceName = "kube-system"
)

var labels = map[string]string{"k8s-app": "kube-proxy"}

func buildKubeproxy(ma *mcpv1alpha1.ManagedControlPlane) []client.Object {
	return []client.Object{
		buildKubeproxyServiceAccount(),
		buildKubeproxyClusterRoleBinding(),
		buildKubeproxyConfigMap(ma),
		buildKubeproxyDaemonSet(ma),
	}
}

func buildKubeproxyServiceAccount() *corev1.ServiceAccount {
	return builders.NewServiceAccount().
		WithName(kubeProxyName).
		WithNamespace(kubeProxyNamespaceName).
		WithLabels(labels).
		Build()
}

func buildKubeproxyClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return builders.NewClusterRoleBinding().
		WithName(kubeProxyName).
		WithRefs(
			rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "system:node-proxier",
			},
			rbacv1.Subject{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      "kube-proxy",
				Namespace: "kube-system",
			},
		).
		Build()
}

// for now hardcode clusterCIDR
// TODO: make it configurable depends
// on CNI that was set
func buildKubeproxyConfigMap(ma *mcpv1alpha1.ManagedControlPlane) *corev1.ConfigMap {
	server := "https://" + ma.Status.Address + ":6443"
	configConf := `apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "iptables"
clusterCIDR: "` + ma.Spec.Kubernetes.Networking.ServiceCIDR + `"
bindAddress: "0.0.0.0"
metricsBindAddress: "127.0.0.1:10249"
healthzBindAddress: "0.0.0.0:10256"
clientConnection:
  kubeconfig: /var/lib/kube-proxy/kubeconfig.conf
iptables:
  masqueradeAll: false
conntrack:
  maxPerCore: 32768
  min: 131072
`

	kubeconfigConf := `apiVersion: v1
kind: Config
clusters:
- name: default
  cluster:
    server: ` + server + `
    certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
users:
- name: kube-proxy
  user:
    tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
contexts:
- name: default
  context:
    cluster: default
    user: kube-proxy
current-context: default
`

	return builders.NewConfigMap().
		WithName(kubeProxyName).
		WithNamespace(kubeProxyNamespaceName).
		Put("config.conf", configConf).
		Put("kubeconfig.conf", kubeconfigConf).
		Build()
}

func buildKubeproxyDaemonSet(ma *mcpv1alpha1.ManagedControlPlane) *appsv1.DaemonSet {
	version := ma.Spec.Kubernetes.Version
	priv := true
	fileOrCreate := corev1.HostPathFileOrCreate
	dir := corev1.HostPathDirectory
	nodeSelector := map[string]string{"kubernetes.io/os": "linux"}
	priorityClass := "system-node-critical"
	updStrategy := appsv1.RollingUpdateDaemonSetStrategyType

	tolerations := []corev1.Toleration{
		{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
		{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute},
	}

	c := corev1.Container{
		Name:  "kube-proxy",
		Image: "registry.k8s.io/kube-proxy:" + version,
		Command: []string{
			"/usr/local/bin/kube-proxy",
			"--config=/var/lib/kube-proxy/config.conf",
			"--hostname-override=$(NODE_NAME)",
		},
		Env: []corev1.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "spec.nodeName",
					},
				},
			},
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: &priv,
		},
		Ports: []corev1.ContainerPort{
			{Name: "metrics", ContainerPort: 10249, HostPort: 10249, Protocol: corev1.ProtocolTCP},
			{Name: "healthz", ContainerPort: 10256, HostPort: 10256, Protocol: corev1.ProtocolTCP},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(10256),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       10,
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{Name: "kube-proxy", MountPath: "/var/lib/kube-proxy"},
		{Name: "xtables-lock", MountPath: "/run/xtables.lock"},
		{Name: "lib-modules", MountPath: "/lib/modules", ReadOnly: true},
	}

	volumes := []corev1.Volume{
		{
			Name: "kube-proxy",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "kube-proxy"},
					Items: []corev1.KeyToPath{
						{Key: "config.conf", Path: "config.conf"},
						{Key: "kubeconfig.conf", Path: "kubeconfig.conf"},
					},
				},
			},
		},
		{
			Name: "xtables-lock",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/run/xtables.lock",
					Type: &fileOrCreate,
				},
			},
		},
		{
			Name: "lib-modules",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/lib/modules",
					Type: &dir,
				},
			},
		},
	}

	return builders.NewDaemonSet().
		WithName(kubeProxyName).
		WithNamespace(kubeProxyNamespaceName).
		WithLabels(labels).
		WithSelector(labels).
		WithPodLabels(labels).
		WithServiceAccount(kubeProxyName).
		WithUpdateStrategy(updStrategy).
		WithHostNetwork().
		WithPriorityClass(priorityClass).
		WithTolerations(tolerations...).
		WithNodeSelector(nodeSelector).
		WithPriorityClass(priorityClass).
		WithContainer(c).
		AddVolumes(volumes...).
		AddVolumeMounts(c.Name, volumeMounts...).
		Build()
}
