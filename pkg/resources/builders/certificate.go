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

package builders

import (
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const IssuerKind = "Issuer"

type CertificateTemplate struct {
	*certmanagerv1.Certificate
	meta MetaMutator
}

func NewCertificate() *CertificateTemplate {
	obj := &certmanagerv1.Certificate{}
	b := &CertificateTemplate{Certificate: obj}
	b.meta = MetaMutator{obj: obj}
	b.Spec = certmanagerv1.CertificateSpec{
		PrivateKey: &certmanagerv1.CertificatePrivateKey{
			Algorithm: certmanagerv1.RSAKeyAlgorithm,
			Size:      2048,
		},
	}
	return b
}

func (c *CertificateTemplate) GetMeta() *metav1.ObjectMeta {
	return &c.Certificate.ObjectMeta
}

func (c *CertificateTemplate) WithLabels(labels map[string]string) *CertificateTemplate {
	c.meta.WithLabels(labels)
	return c
}

func (c *CertificateTemplate) WithAnnotations(ann map[string]string) *CertificateTemplate {
	c.meta.WithAnnotations(ann)
	return c
}

func (c *CertificateTemplate) WithName(name string) *CertificateTemplate {
	c.meta.WithName(name)
	return c
}

func (c *CertificateTemplate) WithNamespace(ns string) *CertificateTemplate {
	c.meta.WithNamespace(ns)
	return c
}

func (c *CertificateTemplate) WithSecretName(secret string) *CertificateTemplate {
	c.Spec.SecretName = secret
	return c
}

func (c *CertificateTemplate) WithCommonName(cn string) *CertificateTemplate {
	c.Spec.CommonName = cn
	return c
}

func (c *CertificateTemplate) IsCA(v bool) *CertificateTemplate {
	c.Spec.IsCA = v
	return c
}

func (c *CertificateTemplate) Issuer(name string) *CertificateTemplate {
	c.Spec.IssuerRef = certmanagermeta.IssuerReference{
		Name: name,
		Kind: IssuerKind,
	}
	return c
}

func (c *CertificateTemplate) WithDuration(d *metav1.Duration) *CertificateTemplate {
	c.Spec.Duration = d
	return c
}

func (c *CertificateTemplate) WithRenewBefore(d *metav1.Duration) *CertificateTemplate {
	c.Spec.RenewBefore = d
	return c
}

func (c *CertificateTemplate) WithUsages(usages ...certmanagerv1.KeyUsage) *CertificateTemplate {
	c.Spec.Usages = append([]certmanagerv1.KeyUsage{}, usages...)
	return c
}

func (c *CertificateTemplate) WithDNSNames(dns ...string) *CertificateTemplate {
	c.Spec.DNSNames = append([]string{}, dns...)
	return c
}

func (c *CertificateTemplate) WithIPAddresses(ips ...string) *CertificateTemplate {
	c.Spec.IPAddresses = append([]string{}, ips...)
	return c
}

func (c *CertificateTemplate) WithOrganizations(orgs ...string) *CertificateTemplate {
	c.Spec.Subject = &certmanagerv1.X509Subject{
		Organizations: append([]string{}, orgs...),
	}
	return c
}

func (c *CertificateTemplate) Build() *certmanagerv1.Certificate {
	return c.Certificate.DeepCopy()
}
