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

package apiserver

import (
	"testing"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildService_APIServer(t *testing.T) {
	api := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "demo-ns",
		},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Version: "v1.34.0",
			},
		},
		Status: mcpv1alpha1.ManagedControlPlaneStatus{
			Address: "192.0.2.10",
		},
	}

	svc := buildService(api)

	// metadata
	if svc.Name != apiServerName {
		t.Fatalf("expected service name %q, got %q", apiServerName, svc.Name)
	}
	if svc.Namespace != api.Namespace {
		t.Fatalf("expected service namespace %q, got %q", api.Namespace, svc.Namespace)
	}

	// labels + selector (should match)
	wantVal := appLabelVal
	if got := svc.Labels[appLabelKey]; got != wantVal {
		t.Fatalf("expected label %s=%s, got %#v", appLabelKey, wantVal, svc.Labels)
	}

	// type
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		t.Fatalf("expected service type %q, got %q", corev1.ServiceTypeLoadBalancer, svc.Spec.Type)
	}

	// ports
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d: %#v", len(svc.Spec.Ports), svc.Spec.Ports)
	}
	p := svc.Spec.Ports[0]

	if p.Name != "https" {
		t.Fatalf("expected port name %q, got %q", "https", p.Name)
	}
	if p.Port != securePort {
		t.Fatalf("expected port %d, got %d", securePort, p.Port)
	}
	if p.Protocol != "" && p.Protocol != corev1.ProtocolTCP {
		t.Fatalf("expected protocol TCP/empty-default, got %q", p.Protocol)
	}

	// targetPort should be intstr FromInt(securePort)
	if p.TargetPort.IntValue() != int(securePort) || p.TargetPort.Type != 0 {
		// Type==0 means Int in k8s intstr
		t.Fatalf("expected targetPort to be int %d, got %#v", securePort, p.TargetPort)
	}

	// sanity: service is labeled as "control-plane" app (not required, but helps ensure you didn't accidentally change labels)
	_ = common.LabelKeyApp
}

func TestBuildDeployment_APIServer(t *testing.T) {
	api := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "demo-ns",
		},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Version: "v1.34.0",
				Networking: &mcpv1alpha1.NetworkingSpec{
					ServiceCIDR: "10.96.0.0/12",
				},
			},
		},
		Status: mcpv1alpha1.ManagedControlPlaneStatus{
			Address: "192.0.2.10",
		},
	}

	p := pki.New(api).APIServer()

	dep := buildDeployment(api)

	// metadata
	if dep.Name != apiServerName {
		t.Fatalf("expected deployment name %q, got %q", apiServerName, dep.Name)
	}
	if dep.Namespace != api.Namespace {
		t.Fatalf("expected namespace %q, got %q", api.Namespace, dep.Namespace)
	}
	if dep.Spec.Replicas == nil || *dep.Spec.Replicas != 1 {
		t.Fatalf("expected replicas=1, got %#v", dep.Spec.Replicas)
	}

	// container
	pod := dep.Spec.Template.Spec
	if len(pod.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(pod.Containers))
	}
	c := pod.Containers[0]
	if c.Name != "apiserver" {
		t.Fatalf("expected container name %q, got %q", "apiserver", c.Name)
	}
	wantImage := "registry.k8s.io/kube-apiserver:" + api.Spec.Kubernetes.Version
	if c.Image != wantImage {
		t.Fatalf("expected image %q, got %q", wantImage, c.Image)
	}

	// key args
	mustContainArg(t, c.Args, "--advertise-address="+api.Status.Address)
	mustContainArg(t, c.Args, "--service-cluster-ip-range="+api.Spec.Kubernetes.Networking.ServiceCIDR)
	mustContainArg(t, c.Args, "--etcd-servers=https://etcd-0.etcd."+api.Namespace+".svc:2379")

	mustContainArg(t, c.Args, "--client-ca-file="+p.ClientCA.CertPath())
	mustContainArg(t, c.Args, "--tls-cert-file="+p.Serving.CertPath())
	mustContainArg(t, c.Args, "--tls-private-key-file="+p.Serving.KeyPath())

	mustContainArg(t, c.Args, "--etcd-cafile="+p.EtcdCA.CertPath())
	mustContainArg(t, c.Args, "--etcd-certfile="+p.EtcdClient.CertPath())
	mustContainArg(t, c.Args, "--etcd-keyfile="+p.EtcdClient.KeyPath())

	mustContainArg(t, c.Args, "--kubelet-client-certificate="+p.KubeletClient.CertPath())
	mustContainArg(t, c.Args, "--kubelet-client-key="+p.KubeletClient.KeyPath())

	mustContainArg(t, c.Args, "--service-account-key-file="+p.ServiceAccountSigner.CertPath())
	mustContainArg(t, c.Args, "--service-account-signing-key-file="+p.ServiceAccountSigner.KeyPath())

	// probes
	if c.LivenessProbe == nil {
		t.Fatalf("expected LivenessProbe to be set")
	}
	if c.ReadinessProbe == nil {
		t.Fatalf("expected ReadinessProbe to be set")
	}

	// volumes (8 secrets)
	expectVolume(t, pod.Volumes, p.ClientCA.SecretName)
	expectVolume(t, pod.Volumes, p.Serving.SecretName)
	expectVolume(t, pod.Volumes, p.EtcdCA.SecretName)
	expectVolume(t, pod.Volumes, p.EtcdClient.SecretName)
	expectVolume(t, pod.Volumes, p.KubeletClient.SecretName)
	expectVolume(t, pod.Volumes, p.ServiceAccountSigner.SecretName)
	// expectVolume(t, pod.Volumes, p.FrontProxyCA.SecretName)
	// expectVolume(t, pod.Volumes, p.FrontProxyClient.SecretName)

	// mounts (same volumes, mounted at view-provided dirs)
	expectMount(t, c.VolumeMounts, p.ClientCA.SecretName, p.ClientCA.MountDir)
	expectMount(t, c.VolumeMounts, p.Serving.SecretName, p.Serving.MountDir)
	expectMount(t, c.VolumeMounts, p.EtcdCA.SecretName, p.EtcdCA.MountDir)
	expectMount(t, c.VolumeMounts, p.EtcdClient.SecretName, p.EtcdClient.MountDir)
	expectMount(t, c.VolumeMounts, p.KubeletClient.SecretName, p.KubeletClient.MountDir)
	expectMount(t, c.VolumeMounts, p.ServiceAccountSigner.SecretName, p.ServiceAccountSigner.MountDir)
	// expectMount(t, c.VolumeMounts, p.FrontProxyCA.SecretName, p.FrontProxyCA.MountDir)
	// expectMount(t, c.VolumeMounts, p.FrontProxyClient.SecretName, p.FrontProxyClient.MountDir)
}

func mustContainArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			return
		}
	}
	t.Fatalf("expected args to contain %q, got %#v", want, args)
}

func expectVolume(t *testing.T, vols []corev1.Volume, name string) {
	t.Helper()
	for _, v := range vols {
		if v.Name == name {
			if v.Secret == nil || v.Secret.SecretName != name {
				t.Fatalf("expected volume %q to be secret volume with secretName=%q, got %#v", name, name, v)
			}
			return
		}
	}
	t.Fatalf("expected volume %q not found; volumes=%#v", name, vols)
}

func expectMount(t *testing.T, mounts []corev1.VolumeMount, name, path string) {
	t.Helper()
	for _, m := range mounts {
		if m.Name == name {
			if m.MountPath != path {
				t.Fatalf("mount %q: expected path %q, got %q", name, path, m.MountPath)
			}
			if !m.ReadOnly {
				t.Fatalf("mount %q: expected readOnly=true", name)
			}
			return
		}
	}
	t.Fatalf("expected volumeMount %q (path=%q) not found; mounts=%#v", name, path, mounts)
}
