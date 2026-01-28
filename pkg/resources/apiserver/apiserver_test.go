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

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEndpointBuilder_Service(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
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

	cc := cluster.NewClusterContext(mcp, logr.Logger{})

	objs := NewEndpointBuilder(cc).Objects()
	if len(objs) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objs))
	}

	svc, ok := objs[0].(*corev1.Service)
	if !ok {
		t.Fatalf("expected *corev1.Service, got %T", objs[0])
	}

	// metadata
	if svc.Name != cc.Names.APIServerServiceName() {
		t.Fatalf("expected service name %q, got %q", cc.Names.APIServerServiceName(), svc.Name)
	}
	if svc.Namespace != mcp.Namespace {
		t.Fatalf("expected service namespace %q, got %q", mcp.Namespace, svc.Namespace)
	}

	// labels + selector (should match)
	wantVal := cc.Contract.APIServer.AppLabelVal
	if got := svc.Labels[cc.Contract.APIServer.AppLabelKey]; got != wantVal {
		t.Fatalf("expected label %s=%s, got %#v", cc.Contract.APIServer.AppLabelKey, wantVal, svc.Labels)
	}

	// type
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		t.Fatalf("expected service type %q, got %q", corev1.ServiceTypeLoadBalancer, svc.Spec.Type)
	}

	// ports: https + grpc
	if len(svc.Spec.Ports) != 2 {
		t.Fatalf("expected 2 ports, got %d: %#v", len(svc.Spec.Ports), svc.Spec.Ports)
	}

	https := findServicePort(t, svc, "https")
	if https.Port != 6443 {
		t.Fatalf("expected https port %d, got %d", 6443, https.Port)
	}
	if https.TargetPort.IntValue() != 6443 {
		t.Fatalf("expected https targetPort int %d, got %#v", 6443, https.TargetPort)
	}

	grpc := findServicePort(t, svc, "grpc")
	if grpc.Port != 8132 {
		t.Fatalf("expected grpc port %d, got %d", 8132, grpc.Port)
	}
	if grpc.TargetPort.IntValue() != 8132 {
		t.Fatalf("expected grpc targetPort int %d, got %#v", 8132, grpc.TargetPort)
	}
}

func TestWorkloadBuilder_ConfigMapAndDeployment(t *testing.T) {
	mcp := &mcpv1alpha1.ManagedControlPlane{
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

	cc := cluster.NewClusterContext(mcp, logr.Logger{})

	// PKI secret names (no pki package)
	clientCA := cc.Names.SecretManagedCAName()
	serving := cc.Names.SecretAPIServerServingTLSName()
	etcdCA := cc.Names.SecretEtcdCAName()
	etcdClient := cc.Names.SecretAPIServerEtcdClientName()
	kubeletClient := cc.Names.SecretAPIServerKubeletClientName()
	saSigner := cc.Names.SecretServiceAccountSignerName()
	fpCA := cc.Names.SecretFrontProxyCAName()
	fpClient := cc.Names.SecretFrontProxyClientName()
	konCA := cc.Names.SecretKonnectivityCAName()
	konSrv := cc.Names.SecretKonnectivityServerTLSName()

	objs := NewWorkloadBuilder(cc).Objects()
	if len(objs) != 2 {
		t.Fatalf("expected 2 objects (Deployment + ConfigMap), got %d: %#v", len(objs), objs)
	}

	var dep *appsv1.Deployment
	var cm *corev1.ConfigMap
	for _, o := range objs {
		switch x := o.(type) {
		case *appsv1.Deployment:
			dep = x
		case *corev1.ConfigMap:
			cm = x
		default:
			t.Fatalf("unexpected object type %T", o)
		}
	}
	if dep == nil {
		t.Fatalf("expected Deployment in objects")
	}
	if cm == nil {
		t.Fatalf("expected ConfigMap in objects")
	}

	// ConfigMap
	if cm.Name != cc.Names.KonnectivityConfigMapName() {
		t.Fatalf("expected konnectivity configmap name %q, got %q", cc.Names.KonnectivityConfigMapName(), cm.Name)
	}
	if cm.Namespace != mcp.Namespace {
		t.Fatalf("expected configmap namespace %q, got %q", mcp.Namespace, cm.Namespace)
	}

	// Deployment metadata
	if dep.Name != cc.Names.APIServerDeploymentName() {
		t.Fatalf("expected deployment name %q, got %q", cc.Names.APIServerDeploymentName(), dep.Name)
	}
	if dep.Namespace != mcp.Namespace {
		t.Fatalf("expected namespace %q, got %q", mcp.Namespace, dep.Namespace)
	}
	if dep.Spec.Replicas == nil || *dep.Spec.Replicas != 1 {
		t.Fatalf("expected replicas=1, got %#v", dep.Spec.Replicas)
	}

	// containers
	pod := dep.Spec.Template.Spec
	if len(pod.Containers) != 2 {
		t.Fatalf("expected 2 containers (apiserver + konnectivity), got %d", len(pod.Containers))
	}

	apiC := findContainer(t, pod.Containers, "apiserver")
	wantImage := "registry.k8s.io/kube-apiserver:" + mcp.Spec.Kubernetes.Version
	if apiC.Image != wantImage {
		t.Fatalf("expected image %q, got %q", wantImage, apiC.Image)
	}

	// key args
	mustContainArg(t, apiC.Args, "--advertise-address="+mcp.Status.Address)
	mustContainArg(t, apiC.Args, "--service-cluster-ip-range="+mcp.Spec.Kubernetes.Networking.ServiceCIDR)

	mustContainArg(t, apiC.Args, "--etcd-servers=https://etcd-0."+cc.Names.EtcdServiceName()+"."+mcp.Namespace+".svc.cluster.local:2379")

	// args: PKI paths derived from cc helpers
	mustContainArg(t, apiC.Args, "--client-ca-file="+cc.CertPath(clientCA))
	mustContainArg(t, apiC.Args, "--tls-cert-file="+cc.CertPath(serving))
	mustContainArg(t, apiC.Args, "--tls-private-key-file="+cc.KeyPath(serving))

	mustContainArg(t, apiC.Args, "--etcd-cafile="+cc.CAPath(etcdCA))
	mustContainArg(t, apiC.Args, "--etcd-certfile="+cc.CertPath(etcdClient))
	mustContainArg(t, apiC.Args, "--etcd-keyfile="+cc.KeyPath(etcdClient))

	mustContainArg(t, apiC.Args, "--kubelet-client-certificate="+cc.CertPath(kubeletClient))
	mustContainArg(t, apiC.Args, "--kubelet-client-key="+cc.KeyPath(kubeletClient))

	mustContainArg(t, apiC.Args, "--service-account-key-file="+cc.CertPath(saSigner))
	mustContainArg(t, apiC.Args, "--service-account-signing-key-file="+cc.KeyPath(saSigner))

	// probes
	if apiC.LivenessProbe == nil {
		t.Fatalf("expected LivenessProbe to be set")
	}
	if apiC.ReadinessProbe == nil {
		t.Fatalf("expected ReadinessProbe to be set")
	}

	// volumes: include konnectivity configmap + uds + secrets
	expectConfigMapVolume(t, pod.Volumes, konnectivityConfigVolumeName, cc.Names.KonnectivityConfigMapName())
	expectEmptyDirVolume(t, pod.Volumes, konnectivityServerUDS)

	expectSecretVolume(t, pod.Volumes, clientCA)
	expectSecretVolume(t, pod.Volumes, serving)
	expectSecretVolume(t, pod.Volumes, etcdCA)
	expectSecretVolume(t, pod.Volumes, etcdClient)
	expectSecretVolume(t, pod.Volumes, kubeletClient)
	expectSecretVolume(t, pod.Volumes, saSigner)
	expectSecretVolume(t, pod.Volumes, fpCA)
	expectSecretVolume(t, pod.Volumes, fpClient)
	expectSecretVolume(t, pod.Volumes, konSrv)
	expectSecretVolume(t, pod.Volumes, konCA)

	// mounts: apiserver container includes config + uds + PKI mounts
	expectMount(t, apiC.VolumeMounts, konnectivityConfigVolumeName, konnectivityServerMountDir)
	expectMountRW(t, apiC.VolumeMounts, konnectivityServerUDS, filepathForUDSDir(cc))

	expectMount(t, apiC.VolumeMounts, clientCA, cc.SecretMountDir(clientCA))
	expectMount(t, apiC.VolumeMounts, serving, cc.SecretMountDir(serving))
	expectMount(t, apiC.VolumeMounts, etcdCA, cc.SecretMountDir(etcdCA))
	expectMount(t, apiC.VolumeMounts, etcdClient, cc.SecretMountDir(etcdClient))
	expectMount(t, apiC.VolumeMounts, kubeletClient, cc.SecretMountDir(kubeletClient))
	expectMount(t, apiC.VolumeMounts, saSigner, cc.SecretMountDir(saSigner))
	expectMount(t, apiC.VolumeMounts, fpCA, cc.SecretMountDir(fpCA))
	expectMount(t, apiC.VolumeMounts, fpClient, cc.SecretMountDir(fpClient))

	// konnectivity container exists (and optionally assert it mounts TLS)
	konC := findContainer(t, pod.Containers, konnectivityServerName)

	// If your migrated deployment mounts konnectivity TLS secrets into the konnectivity container (recommended):
	expectMount(t, konC.VolumeMounts, konSrv, cc.SecretMountDir(konSrv))
	expectMount(t, konC.VolumeMounts, konCA, cc.SecretMountDir(konCA))
}

// ---- helpers ----

func findServicePort(t *testing.T, svc *corev1.Service, name string) corev1.ServicePort {
	t.Helper()
	for _, p := range svc.Spec.Ports {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("expected service port %q not found; ports=%#v", name, svc.Spec.Ports)
	return corev1.ServicePort{}
}

func findContainer(t *testing.T, cs []corev1.Container, name string) corev1.Container {
	t.Helper()
	for _, c := range cs {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("expected container %q not found; containers=%#v", name, cs)
	return corev1.Container{}
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

func expectSecretVolume(t *testing.T, vols []corev1.Volume, name string) {
	t.Helper()
	for _, v := range vols {
		if v.Name == name {
			if v.Secret == nil || v.Secret.SecretName != name {
				t.Fatalf("expected volume %q to be secret volume with secretName=%q, got %#v", name, name, v)
			}
			return
		}
	}
	t.Fatalf("expected secret volume %q not found; volumes=%#v", name, vols)
}

func expectConfigMapVolume(t *testing.T, vols []corev1.Volume, volName, cmName string) {
	t.Helper()
	for _, v := range vols {
		if v.Name == volName {
			if v.ConfigMap == nil || v.ConfigMap.Name != cmName {
				t.Fatalf("expected volume %q to be configmap volume with name=%q, got %#v", volName, cmName, v)
			}
			return
		}
	}
	t.Fatalf("expected configmap volume %q not found; volumes=%#v", volName, vols)
}

func expectEmptyDirVolume(t *testing.T, vols []corev1.Volume, volName string) {
	t.Helper()
	for _, v := range vols {
		if v.Name == volName {
			if v.EmptyDir == nil {
				t.Fatalf("expected volume %q to be emptyDir, got %#v", volName, v)
			}
			return
		}
	}
	t.Fatalf("expected emptyDir volume %q not found; volumes=%#v", volName, vols)
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

func expectMountRW(t *testing.T, mounts []corev1.VolumeMount, name, path string) {
	t.Helper()
	for _, m := range mounts {
		if m.Name == name {
			if m.MountPath != path {
				t.Fatalf("mount %q: expected path %q, got %q", name, path, m.MountPath)
			}
			if m.ReadOnly {
				t.Fatalf("mount %q: expected readOnly=false", name)
			}
			return
		}
	}
	t.Fatalf("expected volumeMount %q (path=%q) not found; mounts=%#v", name, path, mounts)
}

func filepathForUDSDir(cc *cluster.ClusterContext) string {
	return cc.Layout.PKIRoot + "/" + konnectivityServerUDS
}
