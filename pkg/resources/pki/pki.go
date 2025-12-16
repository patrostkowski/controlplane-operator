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
	"net"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type durations struct {
	tenYears   metav1.Duration
	thirtyDays metav1.Duration
}

func Resources(mcp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	ns := mcp.Namespace
	d := defaultDurations()

	objs := make([]client.Object, 0, 32)

	for _, is := range issuerSpecs(ns) {
		objs = append(objs, BuildIssuer(is))
	}
	for _, cs := range certificateSpecs(mcp, ns, d) {
		objs = append(objs, BuildCertificate(cs))
	}
	return objs
}

func defaultDurations() durations {
	return durations{
		tenYears:   metav1.Duration{Duration: 87600 * time.Hour}, // 10 years
		thirtyDays: metav1.Duration{Duration: 720 * time.Hour},   // 30 days
	}
}

func issuerSpecs(ns string) []PKIIssuerSpec {
	return []PKIIssuerSpec{
		{Name: issuerSelfSigned, Namespace: ns, SelfSigned: true},
		{Name: issuerCA, Namespace: ns, CASecret: secretManagedCA},

		{Name: issuerEtcdSelfSigned, Namespace: ns, SelfSigned: true},
		{Name: issuerEtcdCA, Namespace: ns, CASecret: secretEtcdCA},

		{Name: issuerFrontProxySelf, Namespace: ns, SelfSigned: true},
		{Name: issuerFrontProxyCA, Namespace: ns, CASecret: secretFrontProxyCA},
	}
}

func certificateSpecs(mcp *mcpv1alpha1.ManagedControlPlane, ns string, d durations) []PKICertificateSpec {
	specs := make([]PKICertificateSpec, 0, 32)
	specs = append(specs, rootCASpecs(ns)...)
	specs = append(specs, saSignerSpec(ns, d))
	specs = append(specs, apiserverServingSpec(mcp, ns, d))
	specs = append(specs, apiserverKubeletClientSpec(ns, d))
	specs = append(specs, etcdLeafSpecs(ns, d)...)
	specs = append(specs, frontProxySpecs(ns, d))
	specs = append(specs, componentClientSpecs(ns, d)...)
	specs = append(specs, adminClientSpec(ns, d))
	return specs
}

func rootCASpecs(ns string) []PKICertificateSpec {
	return []PKICertificateSpec{
		{
			Name:       secretManagedCA,
			Namespace:  ns,
			SecretName: secretManagedCA,
			CommonName: cnManagedCA,
			IsCA:       true,
			IssuerName: issuerSelfSigned,
		},
		{
			Name:       secretEtcdCA,
			Namespace:  ns,
			SecretName: secretEtcdCA,
			CommonName: cnEtcdCA,
			IsCA:       true,
			IssuerName: issuerEtcdSelfSigned,
		},
		{
			Name:       secretFrontProxyCA,
			Namespace:  ns,
			SecretName: secretFrontProxyCA,
			CommonName: cnFrontProxyCA,
			IsCA:       true,
			IssuerName: issuerFrontProxySelf,
		},
	}
}

func saSignerSpec(ns string, d durations) PKICertificateSpec {
	return PKICertificateSpec{
		Name:        secretSASigner,
		Namespace:   ns,
		SecretName:  secretSASigner,
		CommonName:  cnSASigner,
		IssuerName:  issuerCA,
		Duration:    &d.tenYears,
		RenewBefore: &d.thirtyDays,
		Usages: []certmanagerv1.KeyUsage{
			certmanagerv1.UsageDigitalSignature,
			certmanagerv1.UsageKeyEncipherment,
		},
	}
}

func apiserverServingSpec(mcp *mcpv1alpha1.ManagedControlPlane, ns string, d durations) PKICertificateSpec {
	dns, ips := apiserverSANs(mcp, ns)

	return PKICertificateSpec{
		Name:        secretAPIServerTLS,
		Namespace:   ns,
		SecretName:  secretAPIServerTLS,
		CommonName:  cnAPIServer,
		IssuerName:  issuerCA,
		Duration:    &d.tenYears,
		RenewBefore: &d.thirtyDays,
		Usages: []certmanagerv1.KeyUsage{
			certmanagerv1.UsageDigitalSignature,
			certmanagerv1.UsageKeyEncipherment,
			certmanagerv1.UsageServerAuth,
		},
		DNSNames:    dns,
		IPAddresses: ips,
	}
}

func apiserverKubeletClientSpec(ns string, d durations) PKICertificateSpec {
	return PKICertificateSpec{
		Name:          secretAPIServerKubelet,
		Namespace:     ns,
		SecretName:    secretAPIServerKubelet,
		CommonName:    cnAPIServerKubelet,
		IssuerName:    issuerCA,
		Duration:      &d.tenYears,
		RenewBefore:   &d.thirtyDays,
		Usages:        []certmanagerv1.KeyUsage{certmanagerv1.UsageDigitalSignature, certmanagerv1.UsageKeyEncipherment, certmanagerv1.UsageClientAuth},
		Organizations: []string{orgSystemMasters},
	}
}

func etcdLeafSpecs(ns string, d durations) []PKICertificateSpec {
	etcd0 := "etcd-0.etcd." + ns + ".svc"
	etcdSvc := "etcd." + ns + ".svc"

	return []PKICertificateSpec{
		// etcd server cert
		{
			Name:        "etcd-server",
			Namespace:   ns,
			SecretName:  secretEtcdServerTLS,
			CommonName:  etcd0,
			IssuerName:  issuerEtcdCA,
			Duration:    &d.tenYears,
			RenewBefore: &d.thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageServerAuth,
				certmanagerv1.UsageClientAuth,
			},
			DNSNames: []string{etcd0, etcdSvc, "localhost"},
		},
		// etcd peer cert
		{
			Name:        "etcd-peer",
			Namespace:   ns,
			SecretName:  secretEtcdPeerTLS,
			CommonName:  etcd0,
			IssuerName:  issuerEtcdCA,
			Duration:    &d.tenYears,
			RenewBefore: &d.thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageServerAuth,
				certmanagerv1.UsageClientAuth,
			},
			DNSNames: []string{etcd0, "localhost"},
		},
		// etcd healthcheck client
		{
			Name:        secretEtcdHealthClient,
			Namespace:   ns,
			SecretName:  secretEtcdHealthClient,
			CommonName:  cnEtcdHealth,
			IssuerName:  issuerEtcdCA,
			Duration:    &d.tenYears,
			RenewBefore: &d.thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},
		// apiserver -> etcd client
		{
			Name:        secretAPIServerEtcd,
			Namespace:   ns,
			SecretName:  secretAPIServerEtcd,
			CommonName:  cnAPIServerEtcd,
			IssuerName:  issuerEtcdCA,
			Duration:    &d.tenYears,
			RenewBefore: &d.thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},
	}
}

func frontProxySpecs(ns string, d durations) PKICertificateSpec {
	return PKICertificateSpec{
		Name:        secretFrontProxyClient,
		Namespace:   ns,
		SecretName:  secretFrontProxyClient,
		CommonName:  cnFrontProxyClient,
		IssuerName:  issuerFrontProxyCA,
		Duration:    &d.tenYears,
		RenewBefore: &d.thirtyDays,
		Usages: []certmanagerv1.KeyUsage{
			certmanagerv1.UsageDigitalSignature,
			certmanagerv1.UsageKeyEncipherment,
			certmanagerv1.UsageClientAuth,
		},
	}
}

func componentClientSpecs(ns string, d durations) []PKICertificateSpec {
	return []PKICertificateSpec{
		{
			Name:        secretCMClient,
			Namespace:   ns,
			SecretName:  secretCMClient,
			CommonName:  cnCMClient,
			IssuerName:  issuerCA,
			Duration:    &d.tenYears,
			RenewBefore: &d.thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},
		{
			Name:        secretSchedulerClient,
			Namespace:   ns,
			SecretName:  secretSchedulerClient,
			CommonName:  cnSchedulerClient,
			IssuerName:  issuerCA,
			Duration:    &d.tenYears,
			RenewBefore: &d.thirtyDays,
			Usages: []certmanagerv1.KeyUsage{
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			},
		},
	}
}

func adminClientSpec(ns string, d durations) PKICertificateSpec {
	return PKICertificateSpec{
		Name:       secretAdminClient,
		Namespace:  ns,
		SecretName: secretAdminClient,
		CommonName: cnAdminClient,
		IssuerName: issuerCA,
		Organizations: []string{
			orgSystemMasters,
		},
		Duration:    &d.tenYears,
		RenewBefore: &d.thirtyDays,
		Usages: []certmanagerv1.KeyUsage{
			certmanagerv1.UsageDigitalSignature,
			certmanagerv1.UsageKeyEncipherment,
			certmanagerv1.UsageClientAuth,
		},
	}
}

func apiserverSANs(mcp *mcpv1alpha1.ManagedControlPlane, ns string) (dns []string, ips []string) {
	// Stable DNS SANs
	dns = []string{
		"kube-apiserver." + ns + ".svc",
		"kube-apiserver." + ns + ".svc.cluster.local",
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster.local",
		"localhost",
	}

	// Service CIDR -> kubernetes.default service IP (typically first usable)
	if svcIP, ok := firstServiceIP(mcp.Spec.Networking.ServiceCIDR); ok {
		ips = append(ips, svcIP)
	}

	// LB address in status (might be IP or hostname)
	addr := mcp.Status.Address
	if addr != "" {
		if net.ParseIP(addr) != nil {
			ips = append(ips, addr)
		} else {
			dns = append(dns, addr)
		}
	}

	// Optional: local loopback
	ips = append(ips, "127.0.0.1")
	return dns, ips
}

func firstServiceIP(cidr string) (string, bool) {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", false
	}

	ip := make(net.IP, len(n.IP))
	copy(ip, n.IP)

	// first usable is network + 1
	ip = addIP(ip, 1)
	return ip.String(), true
}

func addIP(ip net.IP, add uint) net.IP {
	// Work on a copy
	out := make(net.IP, len(ip))
	copy(out, ip)

	// Ensure we operate on 16-byte form for IPv4 too
	out16 := out.To16()
	if out16 == nil {
		return out
	}
	out = out16

	for i := len(out) - 1; i >= 0 && add > 0; i-- {
		sum := uint(out[i]) + (add & 0xff)
		out[i] = byte(sum & 0xff)
		add = (add >> 8) + (sum >> 8)
	}

	// If original was IPv4, return in 4-byte form
	if v4 := ip.To4(); v4 != nil {
		return out.To4()
	}
	return out
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
			Kind: kindIssuer,
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
