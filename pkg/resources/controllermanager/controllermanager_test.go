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

	"github.com/go-logr/logr"
	testutil "github.com/patrostkowski/controlplane-operator/internal/test"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildConfigMap_ControllerManager(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-cp", Namespace: "demo-ns"},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Version: "v1.34.0",
				Networking: &mcpv1alpha1.NetworkingSpec{
					PodCIDR:     "10.0.0.0/18",
					ServiceCIDR: "172.16.0.0/20",
				},
			},
		},
	}

	cc := cluster.NewClusterContext(mcp, logr.Logger{})
	cm := cc.ControllerManager()

	objs := NewBuilder(cc).Objects()
	cfg := testutil.MustFindConfigMap(t, objs)

	if cfg.Name != cm.KubeconfigConfigMapName() {
		t.Fatalf("configmap name=%q want %q", cfg.Name, cm.KubeconfigConfigMapName())
	}
	if cfg.Namespace != mcp.Namespace {
		t.Fatalf("configmap namespace=%q want %q", cfg.Namespace, mcp.Namespace)
	}

	val := strings.TrimSpace(cfg.Data[cmKubeconfigKey])
	if val == "" {
		t.Fatalf("expected configmap data[%q] to be non-empty", cmKubeconfigKey)
	}
	if !strings.Contains(val, "apiVersion: v1") {
		t.Fatalf("expected kubeconfig to contain apiVersion: v1, got:\n%s", val)
	}
}

func TestBuildDeployment_ControllerManager(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-cp", Namespace: "demo-ns"},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Version: "v1.34.0",
				Networking: &mcpv1alpha1.NetworkingSpec{
					PodCIDR: "10.244.0.0/16",
				},
			},
		},
	}
	cc := cluster.NewClusterContext(mcp, logr.Logger{})
	cm := cc.ControllerManager()

	objs := NewBuilder(cc).Objects()
	dep := testutil.MustFindDeployment(t, objs)

	if dep.Name != cm.DeploymentName() {
		t.Fatalf("deployment name=%q want %q", dep.Name, cm.DeploymentName())
	}
	if dep.Namespace != mcp.Namespace {
		t.Fatalf("deployment namespace=%q want %q", dep.Namespace, mcp.Namespace)
	}

	if len(dep.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected exactly 1 container, got %d", len(dep.Spec.Template.Spec.Containers))
	}
	c := dep.Spec.Template.Spec.Containers[0]
	if c.Name != containerName {
		t.Fatalf("container name=%q want %q", c.Name, containerName)
	}
	if !strings.Contains(c.Image, mcp.Spec.Kubernetes.Version) {
		t.Fatalf("expected image to include version %q, got %q", mcp.Spec.Kubernetes.Version, c.Image)
	}

	// Args should reference the correct PKI-derived paths.
	clusterCA := cm.ClusterCASecret()
	cmClient := cm.ClientCertSecret()
	saSigner := cm.SASignerSecret()

	testutil.MustContainArg(t, c.Args, "--kubeconfig="+kubeconfigPath)
	testutil.MustContainArg(t, c.Args, "--client-ca-file="+cc.CertPath(clusterCA))
	testutil.MustContainArg(t, c.Args, "--service-account-private-key-file="+cc.KeyPath(saSigner))
	testutil.MustContainArg(t, c.Args, "--cluster-cidr="+mcp.Spec.Kubernetes.Networking.PodCIDR)

	// Volumes must include the three PKI secrets.
	testutil.ExpectVolume(t, dep.Spec.Template.Spec.Volumes, clusterCA, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != clusterCA {
			t.Fatalf("volume %q expected secret %q, got %#v", clusterCA, clusterCA, v.Secret)
		}
	})
	testutil.ExpectVolume(t, dep.Spec.Template.Spec.Volumes, cmClient, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != cmClient {
			t.Fatalf("volume %q expected secret %q, got %#v", cmClient, cmClient, v.Secret)
		}
	})
	testutil.ExpectVolume(t, dep.Spec.Template.Spec.Volumes, saSigner, func(v corev1.Volume) {
		if v.Secret == nil || v.Secret.SecretName != saSigner {
			t.Fatalf("volume %q expected secret %q, got %#v", saSigner, saSigner, v.Secret)
		}
	})

	// And mounts should exist for those secrets.
	testutil.ExpectMount(t, c.VolumeMounts, clusterCA, cc.SecretMountDir(clusterCA), true)
	testutil.ExpectMount(t, c.VolumeMounts, cmClient, cc.SecretMountDir(cmClient), true)
	testutil.ExpectMount(t, c.VolumeMounts, saSigner, cc.SecretMountDir(saSigner), true)

	// Kubeconfig volume should mount to /etc/kubernetes.
	testutil.ExpectMount(t, c.VolumeMounts, kubeconfigVolumeName, kubeconfigMountDir, true)
}
