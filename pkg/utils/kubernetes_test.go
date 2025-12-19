package utils

import (
	"path/filepath"
	"testing"

	"github.com/patrostkowski/controlplane-operator/pkg/resources/common"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/pki"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestSecretMount(t *testing.T) {
	t.Parallel()

	got := SecretMount("vol", "/mnt/secret")
	if got.Name != "vol" {
		t.Fatalf("Name=%q want %q", got.Name, "vol")
	}
	if got.MountPath != "/mnt/secret" {
		t.Fatalf("MountPath=%q want %q", got.MountPath, "/mnt/secret")
	}
	if !got.ReadOnly {
		t.Fatalf("ReadOnly=false want true")
	}
}

func TestSecretVolume(t *testing.T) {
	t.Parallel()

	got := SecretVolume("vol", "my-secret")

	if got.Name != "vol" {
		t.Fatalf("Name=%q want %q", got.Name, "vol")
	}
	if got.VolumeSource.Secret == nil {
		t.Fatalf("expected VolumeSource.Secret to be set")
	}
	if got.VolumeSource.Secret.SecretName != "my-secret" {
		t.Fatalf("SecretName=%q want %q", got.VolumeSource.Secret.SecretName, "my-secret")
	}

	// Ensure no other source is accidentally set
	if got.VolumeSource.ConfigMap != nil || got.VolumeSource.EmptyDir != nil {
		t.Fatalf("expected only Secret volume source to be set: %#v", got.VolumeSource)
	}
}

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
	clientSecret := "kcm-client"

	cfg := BuildComponentKubeconfig(ns, svc, port, user, clientSecret)
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
	cluster, ok := cfg.Clusters["local"]
	if !ok || cluster == nil {
		t.Fatalf("expected Clusters[local] to exist")
	}
	wantServer := "https://" + svc + "." + ns + ".svc:6443"
	if cluster.Server != wantServer {
		t.Fatalf("Clusters[local].Server=%q want %q", cluster.Server, wantServer)
	}

	caDir := filepath.Join(common.PKIMountRoot, pki.SecretManagedCA)
	wantCA := filepath.Join(caDir, common.TLSCrtKey)
	if cluster.CertificateAuthority != wantCA {
		t.Fatalf("Clusters[local].CertificateAuthority=%q want %q", cluster.CertificateAuthority, wantCA)
	}

	// User auth info
	auth, ok := cfg.AuthInfos[user]
	if !ok || auth == nil {
		t.Fatalf("expected AuthInfos[%s] to exist", user)
	}
	clientDir := filepath.Join(common.PKIMountRoot, clientSecret)
	wantCert := filepath.Join(clientDir, common.TLSCrtKey)
	wantKey := filepath.Join(clientDir, common.TLSKeyKey)

	if auth.ClientCertificate != wantCert {
		t.Fatalf("AuthInfos[%s].ClientCertificate=%q want %q", user, auth.ClientCertificate, wantCert)
	}
	if auth.ClientKey != wantKey {
		t.Fatalf("AuthInfos[%s].ClientKey=%q want %q", user, auth.ClientKey, wantKey)
	}
}
