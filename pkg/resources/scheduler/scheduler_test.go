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
	"strings"
	"testing"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"github.com/patrostkowski/controlplane-operator/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestBuildConfigMap_Scheduler(t *testing.T) {
	cm := &mcpv1alpha1.ManagedControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-cp",
			Namespace: "demo-ns",
		},
		Spec: mcpv1alpha1.ManagedControlPlaneSpec{
			Kubernetes: mcpv1alpha1.KubernetesSpec{
				Version: "v1.34.0",
			},
		},
	}

	cc := cluster.NewClusterContext(cm, logr.Logger{})
	objs := NewBuilder(cc).Objects()

	cfg := mustFindConfigMap(t, objs)

	if cfg.Name != cmKubeconfigName {
		t.Fatalf("expected configmap name %q, got %q", cmKubeconfigName, cfg.Name)
	}
	if cfg.Namespace != cm.Namespace {
		t.Fatalf("expected configmap namespace %q, got %q", cm.Namespace, cfg.Namespace)
	}

	val, ok := cfg.Data[cmKubeconfigKey]
	if !ok {
		t.Fatalf("expected configmap Data to contain key %q; keys=%v", cmKubeconfigKey, keys(cfg.Data))
	}
	if strings.TrimSpace(val) == "" {
		t.Fatalf("expected configmap Data[%q] to be non-empty", cmKubeconfigKey)
	}

	if !strings.Contains(val, "apiVersion: v1") {
		t.Fatalf("expected kubeconfig to contain %q, got:\n%s", "apiVersion: v1", val)
	}
	if !strings.Contains(val, "clusters:") || !strings.Contains(val, "users:") {
		t.Fatalf("expected kubeconfig to contain clusters/users sections, got:\n%s", val)
	}
}

func TestBuildDeployment_KubeScheduler(t *testing.T) {
	ms := &mcpv1alpha1.ManagedControlPlane{}
	ms.Namespace = "mcp"
	ms.Spec.Kubernetes.Version = "v1.34.1"

	cc := cluster.NewClusterContext(ms, logr.Logger{})
	objs := NewBuilder(cc).Objects()

	clusterCA := cc.Names.SecretManagedCAName()
	schedulerClient := cc.Names.SecretSchedulerClientName()

	dep := mustFindDeployment(t, objs)

	if dep.Name != componentName {
		t.Fatalf("metadata.name: got %q, want %q", dep.Name, componentName)
	}
	if dep.Namespace != ms.Namespace {
		t.Fatalf("metadata.namespace: got %q, want %q", dep.Namespace, ms.Namespace)
	}
	if dep.Spec.Selector == nil {
		t.Fatalf("spec.selector is nil")
	}
	if dep.Spec.Selector.MatchLabels[cc.Keys.LabelKeyApp] != labelValApp {
		t.Fatalf("spec.selector.matchLabels[%q]: got %q, want %q",
			cc.Keys.LabelKeyApp,
			dep.Spec.Selector.MatchLabels[cc.Keys.LabelKeyApp],
			labelValApp,
		)
	}

	if dep.Spec.Replicas == nil {
		t.Fatalf("spec.replicas is nil")
	}
	if *dep.Spec.Replicas != 1 {
		t.Fatalf("spec.replicas: got %d, want 1", *dep.Spec.Replicas)
	}

	if dep.Spec.Template.Labels[cc.Keys.LabelKeyApp] != labelValApp {
		t.Fatalf("template.metadata.labels[%q]: got %q, want %q",
			cc.Keys.LabelKeyApp,
			dep.Spec.Template.Labels[cc.Keys.LabelKeyApp],
			labelValApp,
		)
	}

	pod := dep.Spec.Template.Spec
	if len(pod.Containers) != 1 {
		t.Fatalf("pod.spec.containers length: got %d, want 1", len(pod.Containers))
	}
	c := pod.Containers[0]

	if c.Name != containerName {
		t.Fatalf("container.name: got %q, want %q", c.Name, containerName)
	}
	wantImage := "registry.k8s.io/kube-scheduler:" + ms.Spec.Kubernetes.Version
	if c.Image != wantImage {
		t.Fatalf("container.image: got %q, want %q", c.Image, wantImage)
	}
	if c.ImagePullPolicy != corev1.PullIfNotPresent {
		t.Fatalf("container.imagePullPolicy: got %q, want %q", c.ImagePullPolicy, corev1.PullIfNotPresent)
	}

	// Command
	if len(c.Command) != 1 || c.Command[0] != "kube-scheduler" {
		t.Fatalf("container.command: got %#v, want [\"kube-scheduler\"]", c.Command)
	}

	wantArgs := []string{
		"--bind-address=0.0.0.0",
		"--kubeconfig=" + kubeconfigPath,
		"--authentication-kubeconfig=" + kubeconfigPath,
		"--authorization-kubeconfig=" + kubeconfigPath,
		"--leader-elect=true",
		"--logging-format=json",
	}
	assertStringSliceEqual(t, c.Args, wantArgs, "container.args")

	if len(c.Ports) != 1 {
		t.Fatalf("container.ports length: got %d, want 1", len(c.Ports))
	}
	if c.Ports[0].Name != "https" || c.Ports[0].ContainerPort != securePort {
		t.Fatalf("container.ports[0]: got %+v, want name=https port=%d", c.Ports[0], securePort)
	}

	wantLive := utils.HttpsHealthProbe(securePort, cc.Keys.LivezPath, 10, 10, 10, 10)
	wantReady := utils.HttpsHealthProbe(securePort, cc.Keys.ReadyzPath, 5, 5, 5, 5)
	assertProbeEqual(t, c.LivenessProbe, wantLive, "livenessProbe")
	assertProbeEqual(t, c.ReadinessProbe, wantReady, "readinessProbe")

	// mounts: kubeconfig + scheduler client + cluster CA
	if len(c.VolumeMounts) != 3 {
		t.Fatalf("container.volumeMounts length: got %d, want 3", len(c.VolumeMounts))
	}

	vmKubeconfig, ok := findVolumeMount(c.VolumeMounts, cc.Volumes.Kubeconfig)
	if !ok {
		t.Fatalf("missing volumeMount %q", cc.Volumes.Kubeconfig)
	}
	if vmKubeconfig.MountPath != kubeconfigMountDir || !vmKubeconfig.ReadOnly {
		t.Fatalf("kubeconfig volumeMount: got %+v, want mountPath=%q readOnly=true", vmKubeconfig, kubeconfigMountDir)
	}

	if _, ok := findVolumeMount(c.VolumeMounts, schedulerClient); !ok {
		t.Fatalf("missing volumeMount %q (scheduler client)", schedulerClient)
	}
	if _, ok := findVolumeMount(c.VolumeMounts, clusterCA); !ok {
		t.Fatalf("missing volumeMount %q (cluster CA)", clusterCA)
	}

	// volumes: kubeconfig configmap + two secrets
	vKubeconfig, ok := findVolume(pod.Volumes, cc.Volumes.Kubeconfig)
	if !ok {
		t.Fatalf("missing volume %q", cc.Volumes.Kubeconfig)
	}
	if vKubeconfig.ConfigMap == nil {
		t.Fatalf("volume %q: expected configMap, got nil", cc.Volumes.Kubeconfig)
	}
	if vKubeconfig.ConfigMap.Name != cmKubeconfigName {
		t.Fatalf("kubeconfig configMap.name: got %q, want %q", vKubeconfig.ConfigMap.Name, cmKubeconfigName)
	}
	if len(vKubeconfig.ConfigMap.Items) != 1 {
		t.Fatalf("kubeconfig configMap.items length: got %d, want 1", len(vKubeconfig.ConfigMap.Items))
	}
	if vKubeconfig.ConfigMap.Items[0].Key != cmKubeconfigKey || vKubeconfig.ConfigMap.Items[0].Path != cmKubeconfigFileName {
		t.Fatalf("kubeconfig configMap.items[0]: got %+v, want key=%q path=%q",
			vKubeconfig.ConfigMap.Items[0], cmKubeconfigKey, cmKubeconfigFileName)
	}

	assertSecretVolume(t, pod.Volumes, schedulerClient, "scheduler client secret volume")
	assertSecretVolume(t, pod.Volumes, clusterCA, "cluster CA secret volume")
}

func mustFindConfigMap(t *testing.T, objs []client.Object) *corev1.ConfigMap {
	t.Helper()
	for _, o := range objs {
		if cm, ok := o.(*corev1.ConfigMap); ok {
			return cm
		}
	}
	t.Fatalf("expected *corev1.ConfigMap in objects, got %#v", objs)
	return nil
}

func mustFindDeployment(t *testing.T, objs []client.Object) *appsv1.Deployment {
	t.Helper()
	for _, o := range objs {
		if d, ok := o.(*appsv1.Deployment); ok {
			return d
		}
	}
	t.Fatalf("expected *appsv1.Deployment in objects, got %#v", objs)
	return nil
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func assertStringSliceEqual(t *testing.T, got, want []string, field string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length: got %d, want %d\n got=%#v\nwant=%#v", field, len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d]: got %q, want %q\n got=%#v\nwant=%#v", field, i, got[i], want[i], got, want)
		}
	}
}

func assertProbeEqual(t *testing.T, got, want *corev1.Probe, field string) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("%s: got %v, want %v", field, got, want)
		}
		return
	}

	if got.InitialDelaySeconds != want.InitialDelaySeconds ||
		got.PeriodSeconds != want.PeriodSeconds ||
		got.TimeoutSeconds != want.TimeoutSeconds ||
		got.FailureThreshold != want.FailureThreshold ||
		got.SuccessThreshold != want.SuccessThreshold {
		t.Fatalf("%s timing/thresholds mismatch:\n got=%+v\nwant=%+v", field, got, want)
	}
	if got.HTTPGet == nil || want.HTTPGet == nil {
		t.Fatalf("%s httpGet: got=%v want=%v", field, got.HTTPGet, want.HTTPGet)
	}
	if got.HTTPGet.Path != want.HTTPGet.Path ||
		got.HTTPGet.Port != want.HTTPGet.Port ||
		got.HTTPGet.Scheme != want.HTTPGet.Scheme ||
		got.HTTPGet.Host != want.HTTPGet.Host {
		t.Fatalf("%s httpGet mismatch:\n got=%+v\nwant=%+v", field, got.HTTPGet, want.HTTPGet)
	}
}

func findVolumeMount(vms []corev1.VolumeMount, name string) (corev1.VolumeMount, bool) {
	for _, vm := range vms {
		if vm.Name == name {
			return vm, true
		}
	}
	return corev1.VolumeMount{}, false
}

func findVolume(vs []corev1.Volume, name string) (corev1.Volume, bool) {
	for _, v := range vs {
		if v.Name == name {
			return v, true
		}
	}
	return corev1.Volume{}, false
}

func assertSecretVolume(t *testing.T, vs []corev1.Volume, name, msg string) {
	t.Helper()
	v, ok := findVolume(vs, name)
	if !ok {
		t.Fatalf("missing %s: %q", msg, name)
	}
	if v.Secret == nil {
		t.Fatalf("%s %q: expected secret volume, got nil", msg, name)
	}
	if v.Secret.SecretName != name {
		t.Fatalf("%s secretName: got %q, want %q", msg, v.Secret.SecretName, name)
	}
}
