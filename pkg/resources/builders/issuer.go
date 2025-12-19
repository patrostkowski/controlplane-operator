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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IssuerTemplate struct {
	*certmanagerv1.Issuer
}

func NewIssuer(ns, name string) *IssuerTemplate {
	return &IssuerTemplate{
		Issuer: &certmanagerv1.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
		},
	}
}

func (i *IssuerTemplate) WithLabels(labels map[string]string) *IssuerTemplate {
	if i.Labels == nil {
		i.Labels = map[string]string{}
	}
	for k, v := range labels {
		i.Labels[k] = v
	}
	return i
}

func (i *IssuerTemplate) WithAnnotations(ann map[string]string) *IssuerTemplate {
	if i.Annotations == nil {
		i.Annotations = map[string]string{}
	}
	for k, v := range ann {
		i.Annotations[k] = v
	}
	return i
}

func (i *IssuerTemplate) SelfSigned() *IssuerTemplate {
	i.Spec = certmanagerv1.IssuerSpec{
		IssuerConfig: certmanagerv1.IssuerConfig{
			SelfSigned: &certmanagerv1.SelfSignedIssuer{},
		},
	}
	return i
}

func (i *IssuerTemplate) CA(secretName string) *IssuerTemplate {
	i.Spec = certmanagerv1.IssuerSpec{
		IssuerConfig: certmanagerv1.IssuerConfig{
			CA: &certmanagerv1.CAIssuer{
				SecretName: secretName,
			},
		},
	}
	return i
}

func (i *IssuerTemplate) Build() *certmanagerv1.Issuer {
	return i.Issuer.DeepCopy()
}
