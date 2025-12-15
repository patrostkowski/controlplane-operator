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

package pki

import (
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PKIIssuerSpec struct {
	Name      string
	Namespace string

	SelfSigned bool
	CASecret   string
}

type PKICertificateSpec struct {
	Name      string
	Namespace string

	SecretName string
	CommonName string
	IsCA       bool

	IssuerName string

	Duration      *metav1.Duration
	RenewBefore   *metav1.Duration
	Usages        []certmanagerv1.KeyUsage
	DNSNames      []string
	IPAddresses   []string
	Organizations []string
}

// Resources returns all cert-manager objects that should exist for this ManagedPKI.
func Resources(pkiObj *mcpv1alpha1.ManagedPKI) []client.Object {
	ns := pkiObj.Namespace

	tenYears := metav1.Duration{Duration: 87600 * time.Hour} // 10 years
	thirtyDays := metav1.Duration{Duration: 720 * time.Hour} // 30 days

	var objs []client.Object

	// ========= Issuers =========
	issuerSpecs := []PKIIssuerSpec{
		// Root CA
		{Name: "selfsigned", Namespace: ns, SelfSigned: true},
		{Name: "ca-issuer", Namespace: ns, CASecret: "managed-ca"},

		// Etcd CA
		{Name: "etcd-selfsigned", Namespace: ns, SelfSigned: true},
		{Name: "etcd-ca-issuer", Namespace: ns, CASecret: "etcd-ca"},

		// Front-proxy CA
		{Name: "front-proxy-selfsigned", Namespace: ns, SelfSigned: true},
		{Name: "front-proxy-ca-issuer", Namespace: ns, CASecret: "front-proxy-ca"},
	}

	for _, is := range issuerSpecs {
		objs = append(objs, BuildIssuer(is))
	}

	// ========= Certificates =========
	certSpecs := []PKICertificateSpec{
		// Root CA
		{
			Name:       "managed-ca",
			Namespace:  ns,
			SecretName: "managed-ca",
			CommonName: "managed-ca",
			IsCA:       true,
			IssuerName: "selfsigned",
		},

		// Etcd CA
		{
			Name:       "etcd-ca",
			Namespace:  ns,
			SecretName: "etcd-ca",
			CommonName: "etcd-ca",
			IsCA:       true,
			IssuerName: "etcd-selfsigned",
		},

		// Front-proxy CA
		{
			Name:       "front-proxy-ca",
			Namespace:  ns,
			SecretName: "front-proxy-ca",
			CommonName: "kubernetes-front-proxy-ca",
			IsCA:       true,
			IssuerName: "front-proxy-selfsigned",
		},

		// =========================
		// SERVICE ACCOUNT SIGNER (uses ca-issuer)
		// =========================
		{
			Name:        "sa-signer",
			Namespace:   ns,
			SecretName:  "sa-signer",
			CommonName:  "sa-signer",
			IssuerName:  "ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
			},
		},

		// =========================
		// KUBERNETES LEAF CERTS (signed by ca-issuer)
		// =========================

		// kube-apiserver serving cert
		{
			Name:        "apiserver-tls",
			Namespace:   ns,
			SecretName:  "apiserver-tls",
			CommonName:  "kube-apiserver",
			IssuerName:  "ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageServerAuth,
			},
			DNSNames: []string{
				"kube-apiserver." + ns + ".svc",
				"kube-apiserver." + ns + ".svc.cluster.local",
				"kubernetes",
				"kubernetes.default",
				"kubernetes.default.svc",
				"kubernetes.default.svc.cluster.local",
				"localhost",
			},
			IPAddresses: []string{
				"172.30.0.250", // todo use dynamic LB address instead
				"10.200.0.1",   // use first addr from cluster IP
				"127.0.0.1",
			},
		},

		// kube-apiserver -> kubelet client cert (O=system:masters)
		{
			Name:        "apiserver-kubelet-client",
			Namespace:   ns,
			SecretName:  "apiserver-kubelet-client",
			CommonName:  "kube-apiserver-kubelet-client",
			IssuerName:  "ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
			Organizations: []string{"system:masters"},
		},

		// =========================
		// ETCD LEAF CERTS (signed by etcd-ca-issuer)
		// =========================

		// etcd server
		{
			Name:        "etcd-server",
			Namespace:   ns,
			SecretName:  "etcd-server-tls",
			CommonName:  "etcd-0.etcd." + ns + ".svc",
			IssuerName:  "etcd-ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageServerAuth,
				certmanagerv1.UsageClientAuth,
			},
			DNSNames: []string{
				"etcd-0.etcd." + ns + ".svc",
				"etcd." + ns + ".svc",
				"localhost",
			},
		},

		// etcd peer
		{
			Name:        "etcd-peer",
			Namespace:   ns,
			SecretName:  "etcd-peer-tls",
			CommonName:  "etcd-0.etcd." + ns + ".svc",
			IssuerName:  "etcd-ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageServerAuth,
				certmanagerv1.UsageClientAuth,
			},
			DNSNames: []string{
				"etcd-0.etcd." + ns + ".svc",
				"localhost",
			},
		},

		// etcd healthcheck client
		{
			Name:        "etcd-healthcheck-client",
			Namespace:   ns,
			SecretName:  "etcd-healthcheck-client",
			CommonName:  "kube-etcd-healthcheck-client",
			IssuerName:  "etcd-ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},

		// apiserver -> etcd client
		{
			Name:        "apiserver-etcd-client",
			Namespace:   ns,
			SecretName:  "apiserver-etcd-client",
			CommonName:  "kube-apiserver-etcd-client",
			IssuerName:  "etcd-ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},

		// =========================
		// FRONT-PROXY client
		// =========================
		{
			Name:        "front-proxy-client",
			Namespace:   ns,
			SecretName:  "front-proxy-client",
			CommonName:  "front-proxy-client",
			IssuerName:  "front-proxy-ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},

		// =========================
		// Controller-manager / Scheduler client certs
		// =========================
		{
			Name:        "cm-client",
			Namespace:   ns,
			SecretName:  "cm-client",
			CommonName:  "system:kube-controller-manager",
			IssuerName:  "ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},
		{
			Name:        "scheduler-client",
			Namespace:   ns,
			SecretName:  "scheduler-client",
			CommonName:  "system:kube-scheduler",
			IssuerName:  "ca-issuer",
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},

		// =========================
		// Admin Client Cert
		// =========================
		{
			Name:       "admin-client",
			Namespace:  ns,
			SecretName: "admin-client",
			CommonName: "kubernetes-admin",
			IssuerName: "ca-issuer",
			Organizations: []string{
				"system:masters",
			},
			Duration:    &tenYears,
			RenewBefore: &thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},
	}

	for _, cs := range certSpecs {
		objs = append(objs, BuildCertificate(cs))
	}

	objs = append(objs, BuildConfigMap(pkiObj))

	return objs
}

func BuildIssuer(spec PKIIssuerSpec) *certmanagerv1.Issuer {
	issuer := &certmanagerv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
		},
	}

	switch {
	case spec.SelfSigned:
		issuer.Spec = certmanagerv1.IssuerSpec{
			IssuerConfig: certmanagerv1.IssuerConfig{
				SelfSigned: &certmanagerv1.SelfSignedIssuer{},
			},
		}
	case spec.CASecret != "":
		issuer.Spec = certmanagerv1.IssuerSpec{
			IssuerConfig: certmanagerv1.IssuerConfig{
				CA: &certmanagerv1.CAIssuer{
					SecretName: spec.CASecret,
				},
			},
		}
	}

	return issuer
}

func BuildCertificate(spec PKICertificateSpec) *certmanagerv1.Certificate {
	c := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
		},
	}

	certSpec := certmanagerv1.CertificateSpec{
		SecretName: spec.SecretName,
		CommonName: spec.CommonName,
		IsCA:       spec.IsCA,
		IssuerRef: certmanagermeta.IssuerReference{
			Name: spec.IssuerName,
			Kind: "Issuer",
		},
		PrivateKey: &certmanagerv1.CertificatePrivateKey{
			Algorithm: certmanagerv1.RSAKeyAlgorithm,
			Size:      2048,
		},
	}

	if spec.Duration != nil {
		certSpec.Duration = spec.Duration
	}
	if spec.RenewBefore != nil {
		certSpec.RenewBefore = spec.RenewBefore
	}
	if len(spec.Usages) > 0 {
		certSpec.Usages = spec.Usages
	}
	if len(spec.DNSNames) > 0 {
		certSpec.DNSNames = spec.DNSNames
	}
	if len(spec.IPAddresses) > 0 {
		certSpec.IPAddresses = spec.IPAddresses
	}
	if len(spec.Organizations) > 0 {
		certSpec.Subject = &certmanagerv1.X509Subject{
			Organizations: spec.Organizations,
		}
	}

	c.Spec = certSpec
	return c
}

func BuildConfigMap(pki *mcpv1alpha1.ManagedPKI) *corev1.ConfigMap {
	ns := pki.Namespace

	kcfg := BuildAdminKubeconfig(ns)

	kubeconfigData, err := clientcmd.Write(*kcfg)
	if err != nil {
		panic(err) // should never happen
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-kubeconfig",
			Namespace: ns,
		},
		Data: map[string]string{
			"config": string(kubeconfigData),
		},
	}
}

func BuildAdminKubeconfig(namespace string) *clientcmdapi.Config {
	serverURL := "https://kube-apiserver." + namespace + ".svc:6443"

	cfg := clientcmdapi.NewConfig()

	// --- Cluster ---
	cfg.Clusters["local"] = &clientcmdapi.Cluster{
		Server:                   serverURL,
		CertificateAuthorityData: []byte{},
	}

	// --- User ---
	cfg.AuthInfos["local"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: []byte{},
		ClientKeyData:         []byte{},
	}

	// --- Context ---
	cfg.Contexts["local"] = &clientcmdapi.Context{
		Cluster:  "local",
		AuthInfo: "local",
	}

	cfg.CurrentContext = "local"

	return cfg
}
