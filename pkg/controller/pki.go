// Copyright 2025 mcpv1alpha1.patrostkowski.dev
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

package controller

import (
	"context"

	"github.com/go-logr/logr"
	mcpv1alpha1 "github.com/patrostkowski/controlplane-operator/pkg/apis/controlplane.patrostkowski.dev/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PKI struct {
	*Applier
	mcp *mcpv1alpha1.ManagedControlPlane
	log logr.Logger
}

func NewPKI(mcp *mcpv1alpha1.ManagedControlPlane, k8s client.Client, scheme *runtime.Scheme, log logr.Logger) *PKI {
	return &PKI{
		Applier: NewApplier(k8s, scheme, log, fieldOwner),
		mcp:     mcp,
		log:     log.WithName("pki"),
	}
}

func (a *PKI) Ensure(ctx context.Context, resources []client.Object) error {
	return a.Apply(ctx, a.mcp, resources...)
}
