// Copyright 2025 controlplane.patrostkowski.dev
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
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	v1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	mcpv1alpha1 "github.com/patrostkowski/operator-template/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resources returns all cert-manager objects that should exist for this ManagedPKI.
// The reconciler will loop over this and CreateOrUpdate each one.
func Resources(pki *mcpv1alpha1.ManagedPKI) []client.Object {
	ns := pki.Namespace
	// For now, keep it simple and just reuse the static names.
	return []client.Object{
		selfSignedIssuer(ns),
		caCertificate(ns),
		caIssuer(ns),
	}
}

func selfSignedIssuer(namespace string) client.Object {
	return &certmanagerv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selfsigned",
			Namespace: namespace,
		},
		Spec: certmanagerv1.IssuerSpec{
			IssuerConfig: certmanagerv1.IssuerConfig{
				SelfSigned: &certmanagerv1.SelfSignedIssuer{},
			},
		},
	}
}

func caCertificate(namespace string) client.Object {
	return &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managed-ca",
			Namespace: namespace,
		},
		Spec: certmanagerv1.CertificateSpec{
			SecretName: "managed-ca",
			IsCA:       true,
			CommonName: "managed-ca",
			PrivateKey: &certmanagerv1.CertificatePrivateKey{
				Algorithm: certmanagerv1.RSAKeyAlgorithm,
				Size:      2048,
			},
			IssuerRef: v1.IssuerReference{
				Name: "selfsigned",
				Kind: "Issuer",
			},
		},
	}
}

func caIssuer(namespace string) client.Object {
	return &certmanagerv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ca-issuer",
			Namespace: namespace,
		},
		Spec: certmanagerv1.IssuerSpec{
			IssuerConfig: certmanagerv1.IssuerConfig{
				CA: &certmanagerv1.CAIssuer{
					SecretName: "managed-ca",
				},
			},
		},
	}
}
