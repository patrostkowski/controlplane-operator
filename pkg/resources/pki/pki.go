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
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	"github.com/patrostkowski/controlplane-operator/pkg/resources/builders"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type durations struct {
	tenYears   metav1.Duration
	thirtyDays metav1.Duration
}

func Resources(mcp *mcpv1alpha1.ManagedControlPlane) []client.Object {
	ns := mcp.Namespace
	d := defaultDurations()

	objs := make([]client.Object, 0, 32)
	objs = append(objs, issuerResources(ns)...)
	objs = append(objs, certificateResources(mcp, ns, d)...)
	return objs
}

func defaultDurations() durations {
	return durations{
		tenYears:   metav1.Duration{Duration: 87600 * time.Hour}, // 10 years
		thirtyDays: metav1.Duration{Duration: 720 * time.Hour},   // 30 days
	}
}

func issuerResources(ns string) []client.Object {
	return []client.Object{
		// managed
		builders.NewIssuer().
			WithName(issuerSelfSigned).
			WithNamespace(ns).
			SelfSigned().
			Build(),
		builders.NewIssuer().
			WithName(issuerCA).
			WithNamespace(ns).
			CA(secretManagedCA).
			Build(),

		// etcd
		builders.NewIssuer().
			WithName(issuerEtcdSelfSigned).
			WithNamespace(ns).
			SelfSigned().
			Build(),
		builders.NewIssuer().
			WithName(issuerEtcdCA).
			WithNamespace(ns).
			CA(secretEtcdCA).
			Build(),

		// front-proxy
		builders.NewIssuer().
			WithName(issuerFrontProxySelf).
			WithNamespace(ns).
			SelfSigned().
			Build(),
		builders.NewIssuer().
			WithName(issuerFrontProxyCA).
			WithNamespace(ns).
			CA(secretFrontProxyCA).
			Build(),

		// konnectivity
		builders.NewIssuer().
			WithName(issuerKonnectivitySelf).
			WithNamespace(ns).
			SelfSigned().
			Build(),
		builders.NewIssuer().
			WithName(issuerKonnectivityCA).
			WithNamespace(ns).
			CA(secretKonnectivityCA).
			Build(),
	}
}

func certificateResources(mcp *mcpv1alpha1.ManagedControlPlane, ns string, d durations) []client.Object {
	objs := make([]client.Object, 0, 32)

	// root CAs
	objs = append(objs,
		builders.NewCertificate().
			WithName(secretManagedCA).
			WithNamespace(ns).
			WithSecretName(secretManagedCA).
			WithCommonName(cnManagedCA).
			IsCA(true).
			Issuer(issuerSelfSigned).
			Build(),

		builders.NewCertificate().
			WithName(secretEtcdCA).
			WithNamespace(ns).
			WithSecretName(secretEtcdCA).
			WithCommonName(cnEtcdCA).
			IsCA(true).
			Issuer(issuerEtcdSelfSigned).
			Build(),

		builders.NewCertificate().
			WithName(secretFrontProxyCA).
			WithNamespace(ns).
			WithSecretName(secretFrontProxyCA).
			WithCommonName(cnFrontProxyCA).
			IsCA(true).
			Issuer(issuerFrontProxySelf).
			Build(),

		builders.NewCertificate().
			WithName(secretKonnectivityCA).
			WithNamespace(ns).
			WithSecretName(secretKonnectivityCA).
			WithCommonName(cnKonnectivityCA).
			IsCA(true).
			Issuer(issuerKonnectivitySelf).
			Build(),
	)

	// service account signer
	objs = append(objs,
		builders.NewCertificate().
			WithName(secretSASigner).
			WithNamespace(ns).
			WithSecretName(secretSASigner).
			WithCommonName(cnSASigner).
			Issuer(issuerCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
			).
			Build(),
	)

	// apiserver serving cert
	dns, ips := apiserverSANs(mcp, ns)
	objs = append(objs,
		builders.NewCertificate().
			WithName(secretAPIServerTLS).
			WithNamespace(ns).
			WithSecretName(secretAPIServerTLS).
			WithCommonName(cnAPIServer).
			Issuer(issuerCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageServerAuth,
			).
			WithDNSNames(dns...).
			WithIPAddresses(ips...).
			Build(),
	)

	// konnectivity-server cert
	konnDNS, konnIPs := konnectivityServerSANs(mcp, ns)
	objs = append(objs,
		builders.NewCertificate().
			WithName(secretKonnectivityTLS).
			WithNamespace(ns).
			WithSecretName(secretKonnectivityTLS).
			WithCommonName(cnKonnectivityServer).
			Issuer(issuerKonnectivityCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageServerAuth,
			).
			WithDNSNames(konnDNS...).
			WithIPAddresses(konnIPs...).
			Build(),
	)

	// apiserver -> kubelet client
	objs = append(objs,
		builders.NewCertificate().
			WithName(secretAPIServerKubelet).
			WithNamespace(ns).
			WithSecretName(secretAPIServerKubelet).
			WithCommonName(cnAPIServerKubelet).
			Issuer(issuerCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			).
			WithOrganizations(orgSystemMasters).
			Build(),
	)

	// konnectivity-agent client cert
	objs = append(objs,
		builders.NewCertificate().
			WithName(secretKonnectivityAgentTLS).
			WithNamespace(ns).
			WithSecretName(secretKonnectivityAgentTLS).
			WithCommonName(cnKonnectivityAgent).
			Issuer(issuerKonnectivityCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			).
			Build(),
	)

	// etcd leaf certs
	etcd0 := "etcd-0.etcd." + ns + ".svc"
	etcdSvc := "etcd." + ns + ".svc"

	objs = append(objs,
		// etcd server cert
		builders.NewCertificate().
			WithName("etcd-server").
			WithNamespace(ns).
			WithSecretName(secretEtcdServerTLS).
			WithCommonName(etcd0).
			Issuer(issuerEtcdCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageServerAuth,
				certmanagerv1.UsageClientAuth,
			).
			WithDNSNames(etcd0, etcdSvc, "localhost").
			Build(),

		// etcd peer cert
		builders.NewCertificate().
			WithName("etcd-peer").
			WithNamespace(ns).
			WithSecretName(secretEtcdPeerTLS).
			WithCommonName(etcd0).
			Issuer(issuerEtcdCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageServerAuth,
				certmanagerv1.UsageClientAuth,
			).
			WithDNSNames(etcd0, "localhost").
			Build(),

		// etcd healthcheck client
		builders.NewCertificate().
			WithName(secretEtcdHealthClient).
			WithNamespace(ns).
			WithSecretName(secretEtcdHealthClient).
			WithCommonName(cnEtcdHealth).
			Issuer(issuerEtcdCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			).
			Build(),

		// apiserver -> etcd client
		builders.NewCertificate().
			WithName(secretAPIServerEtcd).
			WithNamespace(ns).
			WithSecretName(secretAPIServerEtcd).
			WithCommonName(cnAPIServerEtcd).
			Issuer(issuerEtcdCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			).
			Build(),
	)

	// front-proxy client
	objs = append(objs,
		builders.NewCertificate().
			WithName(secretFrontProxyClient).
			WithNamespace(ns).
			WithSecretName(secretFrontProxyClient).
			WithCommonName(cnFrontProxyClient).
			Issuer(issuerFrontProxyCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			).
			Build(),
	)

	// controller-manager & scheduler clients
	objs = append(objs,
		builders.NewCertificate().
			WithName(secretCMClient).
			WithNamespace(ns).
			WithSecretName(secretCMClient).
			WithCommonName(cnCMClient).
			Issuer(issuerCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			).
			Build(),

		builders.NewCertificate().
			WithName(secretSchedulerClient).
			WithNamespace(ns).
			WithSecretName(secretSchedulerClient).
			WithCommonName(cnSchedulerClient).
			Issuer(issuerCA).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			).
			Build(),
	)

	// admin client (system:masters)
	objs = append(objs,
		builders.NewCertificate().
			WithName(secretAdminClient).
			WithNamespace(ns).
			WithSecretName(secretAdminClient).
			WithCommonName(cnAdminClient).
			Issuer(issuerCA).
			WithOrganizations(orgSystemMasters).
			WithDuration(&d.tenYears).
			WithRenewBefore(&d.thirtyDays).
			WithUsages(
				certmanagerv1.UsageDigitalSignature,
				certmanagerv1.UsageKeyEncipherment,
				certmanagerv1.UsageClientAuth,
			).
			Build(),
	)

	return objs
}

func konnectivityServerSANs(mcp *mcpv1alpha1.ManagedControlPlane, ns string) (dns []string, ips []string) {
	dns = []string{
		"konnectivity-server",
		"konnectivity-server." + ns,
		"konnectivity-server." + ns + ".svc",
	}

	if mcp.Spec.Networking != nil {
		if svcIP, ok := firstServiceIP(mcp.Spec.Networking.ServiceCIDR); ok {
			ips = append(ips, svcIP)
		}
	}

	addr := mcp.Status.Address
	if addr != "" {
		if net.ParseIP(addr) != nil {
			ips = append(ips, addr)
		} else {
			dns = append(dns, addr)
		}
	}

	ips = append(ips, "127.0.0.1")
	return dns, ips
}

func apiserverSANs(mcp *mcpv1alpha1.ManagedControlPlane, ns string) (dns []string, ips []string) {
	dns = []string{
		"kube-apiserver." + ns + ".svc",
		"kube-apiserver." + ns + ".svc.cluster.local",
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster.local",
		"konnectivity-server",
		"konnectivity-server." + ns,
		"konnectivity-server." + ns + ".svc",
		"localhost",
	}

	if mcp.Spec.Networking != nil {
		if svcIP, ok := firstServiceIP(mcp.Spec.Networking.ServiceCIDR); ok {
			ips = append(ips, svcIP)
		}
	}

	addr := mcp.Status.Address
	if addr != "" {
		if net.ParseIP(addr) != nil {
			ips = append(ips, addr)
		} else {
			dns = append(dns, addr)
		}
	}

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

	ip = addIP(ip, 1)
	return ip.String(), true
}

func addIP(ip net.IP, add uint) net.IP {
	out := make(net.IP, len(ip))
	copy(out, ip)

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

	if v4 := ip.To4(); v4 != nil {
		return out.To4()
	}
	return out
}
