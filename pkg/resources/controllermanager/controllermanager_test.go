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
	"strings"
	"testing"

	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildConfigMap_ControllerManager(t *testing.T) {
	cm := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-cp",
			Namespace: "demo-ns",
		},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Version: "v1.34.0",
		},
	}

	got := buildConfigMap(cm)

	if got.Name != cmKubeconfigName {
		t.Fatalf("expected configmap name %q, got %q", cmKubeconfigName, got.Name)
	}
	if got.Namespace != cm.Namespace {
		t.Fatalf("expected configmap namespace %q, got %q", cm.Namespace, got.Namespace)
	}

	val, ok := got.Data[cmKubeconfigKey]
	if !ok {
		t.Fatalf("expected configmap Data to contain key %q; keys=%v", cmKubeconfigKey, keys(got.Data))
	}
	if strings.TrimSpace(val) == "" {
		t.Fatalf("expected configmap Data[%q] to be non-empty", cmKubeconfigKey)
	}

	// Optional sanity checks: kubeconfig should look like YAML and contain cluster/user stanzas.
	// Keep these loose to avoid brittle tests.
	if !strings.Contains(val, "apiVersion: v1") {
		t.Fatalf("expected kubeconfig to contain %q, got:\n%s", "apiVersion: v1", val)
	}
	if !strings.Contains(val, "clusters:") || !strings.Contains(val, "users:") {
		t.Fatalf("expected kubeconfig to contain clusters/users sections, got:\n%s", val)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestBuildDeployment_ControllerManager(t *testing.T) {
	// Arrange
	cm := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-cp",
			Namespace: "demo-ns",
		},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Version: "v1.34.0",
			Networking: &mcpv1alpha1.NetworkingSpec{
				PodCIDR: "10.244.0.0/16",
			},
		},
	}

	p := pki.New(cm).ControllerManager()

	// Act
	dep := buildDeployment(cm)

	// Assert: basic metadata
	if dep.Name != componentName {
		t.Fatalf("expected deployment name %q, got %q", componentName, dep.Name)
	}
	if dep.Namespace != cm.Namespace {
		t.Fatalf("expected namespace %q, got %q", cm.Namespace, dep.Namespace)
	}

	wantLabels := map[string]string{common.LabelKeyApp: labelValApp}
	if dep.Labels[common.LabelKeyApp] != labelValApp {
		t.Fatalf("expected label %s=%s, got %#v", common.LabelKeyApp, labelValApp, dep.Labels)
	}
	if dep.Spec.Selector == nil || dep.Spec.Selector.MatchLabels[common.LabelKeyApp] != labelValApp {
		t.Fatalf("expected selector matchLabels %#v, got %#v", wantLabels, dep.Spec.Selector)
	}
	if dep.Spec.Replicas == nil || *dep.Spec.Replicas != 1 {
		t.Fatalf("expected replicas=1, got %#v", dep.Spec.Replicas)
	}

	// Assert: container
	pod := dep.Spec.Template.Spec
	if len(pod.Containers) != 1 {
		t.Fatalf("expected exactly 1 container, got %d", len(pod.Containers))
	}
	c := pod.Containers[0]
	if c.Name != containerName {
		t.Fatalf("expected container name %q, got %q", containerName, c.Name)
	}
	wantImage := "registry.k8s.io/kube-controller-manager:" + cm.Spec.Version
	if c.Image != wantImage {
		t.Fatalf("expected image %q, got %q", wantImage, c.Image)
	}
	if len(c.Command) != 1 || c.Command[0] != "kube-controller-manager" {
		t.Fatalf("expected command [kube-controller-manager], got %#v", c.Command)
	}

	// Assert: key args exist
	mustContainArg(t, c.Args, "--bind-address=0.0.0.0")
	mustContainArg(t, c.Args, "--cluster-name="+cm.GetObjectMeta().GetName())

	// kubeconfig wiring
	mustContainArg(t, c.Args, "--kubeconfig="+kubeconfigPath)
	mustContainArg(t, c.Args, "--authentication-kubeconfig="+kubeconfigPath)
	mustContainArg(t, c.Args, "--authorization-kubeconfig="+kubeconfigPath)

	// leader election / controllers
	mustContainArg(t, c.Args, "--leader-elect=true")
	mustContainArg(t, c.Args, "--use-service-account-credentials=true")
	mustContainArg(t, c.Args, "--controllers=*,bootstrapsigner,tokencleaner")

	mustContainArg(t, c.Args, "--service-account-private-key-file="+p.ServiceAccountSigner.KeyPath())
	mustContainArg(t, c.Args, "--cluster-signing-cert-file="+p.ClientCA.CertPath())
	mustContainArg(t, c.Args, "--cluster-signing-key-file="+p.ClientCA.KeyPath())
	mustContainArg(t, c.Args, "--client-ca-file="+p.ClientCA.CertPath())
	mustContainArg(t, c.Args, "--root-ca-file="+p.ClientCA.CertPath())

	// networking
	mustContainArg(t, c.Args, "--cluster-cidr="+cm.Spec.Networking.PodCIDR)
	mustContainArg(t, c.Args, "--allocate-node-cidrs=true")

	// probes should be set
	if c.StartupProbe == nil {
		t.Fatalf("expected StartupProbe to be set")
	}
	if c.LivenessProbe == nil {
		t.Fatalf("expected LivenessProbe to be set")
	}
	if c.ReadinessProbe == nil {
		t.Fatalf("expected ReadinessProbe to be set")
	}

	// Assert: mounts
	expectMount(t, c.VolumeMounts, common.KubeconfigVolumeName, kubeconfigMountDir, true)

	// secret mounts (name + path)
	expectMount(t, c.VolumeMounts, p.Client.SecretName, p.Client.MountDir, true)
	expectMount(t, c.VolumeMounts, p.ServiceAccountSigner.SecretName, p.ServiceAccountSigner.MountDir, true)
	expectMount(t, c.VolumeMounts, p.ClientCA.SecretName, p.ClientCA.MountDir, true)

	// Assert: volumes
	expectVolume(t, pod.Volumes, common.KubeconfigVolumeName, func(v corev1.Volume) {
		if v.ConfigMap == nil {
			t.Fatalf("expected kubeconfig volume %q to be a ConfigMap volume, got %#v", common.KubeconfigVolumeName, v)
		}
		if v.ConfigMap.Name != cmKubeconfigName {
			t.Fatalf("expected kubeconfig volume CM name %q, got %q", cmKubeconfigName, v.ConfigMap.Name)
		}
	})
	expectVolume(t, pod.Volumes, p.Client.SecretName, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != p.Client.SecretName {
			t.Fatalf("expected secret volume %q -> secretName %q, got %#v", p.Client.SecretName, p.Client.SecretName, v)
		}
	})
	expectVolume(t, pod.Volumes, p.ServiceAccountSigner.SecretName, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != p.ServiceAccountSigner.SecretName {
			t.Fatalf("expected secret volume %q -> secretName %q, got %#v", p.ServiceAccountSigner.SecretName, p.ServiceAccountSigner.SecretName, v)
		}
	})
	expectVolume(t, pod.Volumes, p.ClientCA.SecretName, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != p.ClientCA.SecretName {
			t.Fatalf("expected secret volume %q -> secretName %q, got %#v", p.ClientCA.SecretName, p.ClientCA.SecretName, v)
		}
	})
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

func expectMount(t *testing.T, mounts []corev1.VolumeMount, name, path string, readOnly bool) {
	t.Helper()
	for _, m := range mounts {
		if m.Name == name {
			if m.MountPath != path {
				t.Fatalf("mount %q: expected path %q, got %q", name, path, m.MountPath)
			}
			if m.ReadOnly != readOnly {
				t.Fatalf("mount %q: expected readOnly=%v, got %v", name, readOnly, m.ReadOnly)
			}
			return
		}
	}
	t.Fatalf("expected volumeMount %q (path=%q) not found; mounts=%#v", name, path, mounts)
}

func expectVolume(t *testing.T, vols []corev1.Volume, name string, validate func(corev1.Volume)) {
	t.Helper()
	for _, v := range vols {
		if v.Name == name {
			validate(v)
			return
		}
	}
	t.Fatalf("expected volume %q not found; volumes=%#v", name, vols)
}
