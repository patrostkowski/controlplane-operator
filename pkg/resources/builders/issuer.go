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
	meta MetaMutator
}

func NewIssuer() *IssuerTemplate {
	iss := &certmanagerv1.Issuer{}
	newISS := &IssuerTemplate{Issuer: iss}
	newISS.meta = MetaMutator{obj: iss}
	return newISS
}

func (i *IssuerTemplate) GetMeta() *metav1.ObjectMeta {
	return &i.Issuer.ObjectMeta
}

func (i *IssuerTemplate) WithLabels(labels map[string]string) *IssuerTemplate {
	i.meta.WithLabels(labels)
	return i
}

func (i *IssuerTemplate) WithAnnotations(ann map[string]string) *IssuerTemplate {
	i.meta.WithAnnotations(ann)
	return i
}

func (i *IssuerTemplate) WithName(name string) *IssuerTemplate {
	i.meta.WithName(name)
	return i
}

func (i *IssuerTemplate) WithNamespace(ns string) *IssuerTemplate {
	i.meta.WithNamespace(ns)
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
