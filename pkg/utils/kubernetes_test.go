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

package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestHttpsHealthProbe(t *testing.T) {
	t.Parallel()

	port := int32(6443)
	path := "/healthz"
	got := HttpsHealthProbe(port, path, 10, 5, 2, 3)

	if got == nil {
		t.Fatalf("expected probe, got nil")
	}
	if got.HTTPGet == nil {
		t.Fatalf("expected HTTPGet, got nil")
	}
	if got.HTTPGet.Scheme != corev1.URISchemeHTTPS {
		t.Fatalf("Scheme=%q want %q", got.HTTPGet.Scheme, corev1.URISchemeHTTPS)
	}
	if got.HTTPGet.Host != "127.0.0.1" {
		t.Fatalf("Host=%q want %q", got.HTTPGet.Host, "127.0.0.1")
	}
	if got.HTTPGet.Path != path {
		t.Fatalf("Path=%q want %q", got.HTTPGet.Path, path)
	}
	if got.HTTPGet.Port.Type != intstr.Int || got.HTTPGet.Port.IntValue() != int(port) {
		t.Fatalf("Port=%v want %v", got.HTTPGet.Port, intstr.FromInt(int(port)))
	}

	if got.InitialDelaySeconds != 10 {
		t.Fatalf("InitialDelaySeconds=%d want %d", got.InitialDelaySeconds, 10)
	}
	if got.PeriodSeconds != 5 {
		t.Fatalf("PeriodSeconds=%d want %d", got.PeriodSeconds, 5)
	}
	if got.TimeoutSeconds != 2 {
		t.Fatalf("TimeoutSeconds=%d want %d", got.TimeoutSeconds, 2)
	}
	if got.FailureThreshold != 3 {
		t.Fatalf("FailureThreshold=%d want %d", got.FailureThreshold, 3)
	}
	if got.SuccessThreshold != 1 {
		t.Fatalf("SuccessThreshold=%d want %d", got.SuccessThreshold, 1)
	}

	// Ensure other handler types aren't set
	if got.TCPSocket != nil || got.Exec != nil || got.GRPC != nil {
		t.Fatalf("expected only HTTPGet handler set, got %#v", got.ProbeHandler)
	}
}

func TestTcpProbe(t *testing.T) {
	t.Parallel()

	port := int32(2379)
	got := TcpProbe(port, 7, 11)

	if got == nil {
		t.Fatalf("expected probe, got nil")
	}
	if got.TCPSocket == nil {
		t.Fatalf("expected TCPSocket, got nil")
	}
	if got.TCPSocket.Port.Type != intstr.Int || got.TCPSocket.Port.IntValue() != int(port) {
		t.Fatalf("Port=%v want %v", got.TCPSocket.Port, intstr.FromInt(int(port)))
	}
	if got.InitialDelaySeconds != 7 {
		t.Fatalf("InitialDelaySeconds=%d want %d", got.InitialDelaySeconds, 7)
	}
	if got.PeriodSeconds != 11 {
		t.Fatalf("PeriodSeconds=%d want %d", got.PeriodSeconds, 11)
	}

	// Ensure other handler types aren't set
	if got.HTTPGet != nil || got.Exec != nil || got.GRPC != nil {
		t.Fatalf("expected only TCPSocket handler set, got %#v", got.ProbeHandler)
	}
}

func TestConfigMapVolume(t *testing.T) {
	t.Parallel()

	got := ConfigMapVolume("mount", "cm", "config.yaml", "config/config.yaml")

	if got.Name != "mount" {
		t.Fatalf("Name=%q want %q", got.Name, "mount")
	}
	if got.VolumeSource.ConfigMap == nil {
		t.Fatalf("expected ConfigMap volume source set")
	}
	cm := got.VolumeSource.ConfigMap
	if cm.Name != "cm" {
		t.Fatalf("ConfigMap name=%q want %q", cm.Name, "cm")
	}
	if len(cm.Items) != 1 {
		t.Fatalf("Items len=%d want %d", len(cm.Items), 1)
	}
	if cm.Items[0].Key != "config.yaml" {
		t.Fatalf("Item[0].Key=%q want %q", cm.Items[0].Key, "config.yaml")
	}
	if cm.Items[0].Path != "config/config.yaml" {
		t.Fatalf("Item[0].Path=%q want %q", cm.Items[0].Path, "config/config.yaml")
	}

	// Ensure no other source is accidentally set
	if got.VolumeSource.Secret != nil || got.VolumeSource.EmptyDir != nil {
		t.Fatalf("expected only ConfigMap volume source to be set: %#v", got.VolumeSource)
	}
}

func TestBuildComponentKubeconfig(t *testing.T) {
	t.Parallel()

	ns := "ns"
	svc := "apiserver"
	port := int32(6443)
	user := "kube-controller-manager"

	// Instead of SecretMount, provide explicit paths
	caCrtPath := "/pki/cluster-ca/tls.crt"
	clientCrtPath := "/pki/cm-client/tls.crt"
	clientKeyPath := "/pki/cm-client/tls.key"

	cfg := BuildComponentKubeconfig(ns, svc, port, user, caCrtPath, clientCrtPath, clientKeyPath)
	if cfg == nil {
		t.Fatalf("expected config, got nil")
	}

	// Current context + context wiring
	if cfg.CurrentContext != "local" {
		t.Fatalf("CurrentContext=%q want %q", cfg.CurrentContext, "local")
	}
	ctx, ok := cfg.Contexts["local"]
	if !ok || ctx == nil {
		t.Fatalf("expected Contexts[local] to exist")
	}
	if ctx.Cluster != "local" {
		t.Fatalf("Contexts[local].Cluster=%q want %q", ctx.Cluster, "local")
	}
	if ctx.AuthInfo != user {
		t.Fatalf("Contexts[local].AuthInfo=%q want %q", ctx.AuthInfo, user)
	}

	// Cluster
	cl, ok := cfg.Clusters["local"]
	if !ok || cl == nil {
		t.Fatalf("expected Clusters[local] to exist")
	}
	wantServer := "https://" + svc + "." + ns + ".svc:6443"
	if cl.Server != wantServer {
		t.Fatalf("Clusters[local].Server=%q want %q", cl.Server, wantServer)
	}
	if cl.CertificateAuthority != caCrtPath {
		t.Fatalf(
			"Clusters[local].CertificateAuthority=%q want %q",
			cl.CertificateAuthority,
			caCrtPath,
		)
	}

	// User auth info
	auth, ok := cfg.AuthInfos[user]
	if !ok || auth == nil {
		t.Fatalf("expected AuthInfos[%s] to exist", user)
	}
	if auth.ClientCertificate != clientCrtPath {
		t.Fatalf(
			"AuthInfos[%s].ClientCertificate=%q want %q",
			user,
			auth.ClientCertificate,
			clientCrtPath,
		)
	}
	if auth.ClientKey != clientKeyPath {
		t.Fatalf(
			"AuthInfos[%s].ClientKey=%q want %q",
			user,
			auth.ClientKey,
			clientKeyPath,
		)
	}
}

func TestBuildKubeconfigWithCertData(t *testing.T) {
	t.Parallel()

	cfg := BuildKubeconfigWithCertData(
		"https://1.2.3.4:6443",
		"local",
		[]byte("ca"),
		[]byte("crt"),
		[]byte("key"),
	)

	if cfg.CurrentContext != DefaultContextName {
		t.Fatalf("CurrentContext=%q want %q", cfg.CurrentContext, DefaultContextName)
	}

	c := cfg.Clusters[DefaultContextName]
	if c == nil || c.Server != "https://1.2.3.4:6443" {
		t.Fatalf("cluster server wrong: %#v", c)
	}

	a := cfg.AuthInfos["local"]
	if a == nil || string(a.ClientCertificateData) != "crt" || string(a.ClientKeyData) != "key" {
		t.Fatalf("auth wrong: %#v", a)
	}
}
