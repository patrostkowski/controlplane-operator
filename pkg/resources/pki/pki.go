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

// --- issuers (now via builders) ---

func issuerResources(ns string) []client.Object {
	return []client.Object{
		// managed
		builders.NewIssuer(ns, issuerSelfSigned).SelfSigned().Build(),
		builders.NewIssuer(ns, issuerCA).CA(secretManagedCA).Build(),

		// etcd
		builders.NewIssuer(ns, issuerEtcdSelfSigned).SelfSigned().Build(),
		builders.NewIssuer(ns, issuerEtcdCA).CA(secretEtcdCA).Build(),

		// front-proxy
		builders.NewIssuer(ns, issuerFrontProxySelf).SelfSigned().Build(),
		builders.NewIssuer(ns, issuerFrontProxyCA).CA(secretFrontProxyCA).Build(),
	}
}

// --- certificates (now via builders) ---

func certificateResources(mcp *mcpv1alpha1.ManagedControlPlane, ns string, d durations) []client.Object {
	objs := make([]client.Object, 0, 32)

	// root CAs
	objs = append(objs,
		builders.NewCertificate(ns, secretManagedCA).
			WithSecretName(secretManagedCA).
			WithCommonName(cnManagedCA).
			IsCA(true).
			Issuer(issuerSelfSigned).
			Build(),

		builders.NewCertificate(ns, secretEtcdCA).
			WithSecretName(secretEtcdCA).
			WithCommonName(cnEtcdCA).
			IsCA(true).
			Issuer(issuerEtcdSelfSigned).
			Build(),

		builders.NewCertificate(ns, secretFrontProxyCA).
			WithSecretName(secretFrontProxyCA).
			WithCommonName(cnFrontProxyCA).
			IsCA(true).
			Issuer(issuerFrontProxySelf).
			Build(),
	)

	// service account signer
	objs = append(objs,
		builders.NewCertificate(ns, secretSASigner).
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
		builders.NewCertificate(ns, secretAPIServerTLS).
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

	// apiserver -> kubelet client
	objs = append(objs,
		builders.NewCertificate(ns, secretAPIServerKubelet).
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

	// etcd leaf certs
	etcd0 := "etcd-0.etcd." + ns + ".svc"
	etcdSvc := "etcd." + ns + ".svc"

	objs = append(objs,
		// etcd server cert
		builders.NewCertificate(ns, "etcd-server").
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
		builders.NewCertificate(ns, "etcd-peer").
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
		builders.NewCertificate(ns, secretEtcdHealthClient).
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
		builders.NewCertificate(ns, secretAPIServerEtcd).
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
		builders.NewCertificate(ns, secretFrontProxyClient).
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
		builders.NewCertificate(ns, secretCMClient).
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

		builders.NewCertificate(ns, secretSchedulerClient).
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
		builders.NewCertificate(ns, secretAdminClient).
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

// --- SAN logic (unchanged) ---

func apiserverSANs(mcp *mcpv1alpha1.ManagedControlPlane, ns string) (dns []string, ips []string) {
	dns = []string{
		"kube-apiserver." + ns + ".svc",
		"kube-apiserver." + ns + ".svc.cluster.local",
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster.local",
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
