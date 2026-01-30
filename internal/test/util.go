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

package test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MustFindService(t *testing.T, objs []client.Object) *corev1.Service {
	t.Helper()
	for _, o := range objs {
		if s, ok := o.(*corev1.Service); ok {
			return s
		}
	}
	t.Fatalf("expected *corev1.Service in objects, got %#v", objs)
	return nil
}

func MustFindConfigMap(t *testing.T, objs []client.Object) *corev1.ConfigMap {
	t.Helper()
	for _, o := range objs {
		if cm, ok := o.(*corev1.ConfigMap); ok {
			return cm
		}
	}
	t.Fatalf("expected *corev1.ConfigMap in objects, got %#v", objs)
	return nil
}

func MustFindDeployment(t *testing.T, objs []client.Object) *appsv1.Deployment {
	t.Helper()
	for _, o := range objs {
		if d, ok := o.(*appsv1.Deployment); ok {
			return d
		}
	}
	t.Fatalf("expected *appsv1.Deployment in objects, got %#v", objs)
	return nil
}

func MustFindStatefulSet(t *testing.T, objs []client.Object) *appsv1.StatefulSet {
	t.Helper()
	for _, o := range objs {
		if s, ok := o.(*appsv1.StatefulSet); ok {
			return s
		}
	}
	t.Fatalf("expected *appsv1.StatefulSet in objects, got %#v", objs)
	return nil
}

func MustContainArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			return
		}
	}
	t.Fatalf("expected args to contain %q, got %#v", want, args)
}

func ExpectMount(t *testing.T, mounts []corev1.VolumeMount, name, path string, readOnly bool) {
	t.Helper()
	for _, m := range mounts {
		if m.Name != name {
			continue
		}
		if m.MountPath != path {
			t.Fatalf("mount %q: expected path %q, got %q", name, path, m.MountPath)
		}
		if m.ReadOnly != readOnly {
			t.Fatalf("mount %q: expected readOnly=%v, got %v", name, readOnly, m.ReadOnly)
		}
		return
	}
	t.Fatalf("expected volumeMount %q (path=%q) not found; mounts=%#v", name, path, mounts)
}

func ExpectVolume(t *testing.T, vols []corev1.Volume, name string, validate func(corev1.Volume)) {
	t.Helper()
	for _, v := range vols {
		if v.Name == name {
			validate(v)
			return
		}
	}
	t.Fatalf("expected volume %q not found; volumes=%#v", name, vols)
}

func ExpectPort(t *testing.T, ports []corev1.ServicePort, name string, port int32) {
	t.Helper()
	for _, p := range ports {
		if p.Name == name {
			if p.Port != port {
				t.Fatalf("port %q: expected %d, got %d", name, port, p.Port)
			}
			return
		}
	}
	t.Fatalf("expected port %q not found; ports=%#v", name, ports)
}
