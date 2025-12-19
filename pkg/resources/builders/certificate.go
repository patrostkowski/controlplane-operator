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
}

func NewCertificate(ns, name string) *CertificateTemplate {
	return &CertificateTemplate{
		Certificate: &certmanagerv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: certmanagerv1.CertificateSpec{
				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.RSAKeyAlgorithm,
					Size:      2048,
				},
			},
		},
	}
}

func (c *CertificateTemplate) WithLabels(labels map[string]string) *CertificateTemplate {
	if c.Labels == nil {
		c.Labels = map[string]string{}
	}
	for k, v := range labels {
		c.Labels[k] = v
	}
	return c
}

func (c *CertificateTemplate) WithAnnotations(ann map[string]string) *CertificateTemplate {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	for k, v := range ann {
		c.Annotations[k] = v
	}
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
