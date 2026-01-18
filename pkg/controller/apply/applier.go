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

package applier

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Applier struct {
	k8s         client.Client
	scheme      *runtime.Scheme
	fieldOwner  string
	log         logr.Logger
	setOwnerRef bool
}

type ApplierOption func(*Applier)

func WithOwnerRef(enabled bool) ApplierOption {
	return func(a *Applier) { a.setOwnerRef = enabled }
}

func NewApplier(k8s client.Client, scheme *runtime.Scheme, log logr.Logger, fieldOwner string, opts ...ApplierOption) *Applier {
	a := &Applier{
		k8s:         k8s,
		scheme:      scheme,
		fieldOwner:  fieldOwner,
		log:         log.WithName("applier"),
		setOwnerRef: true,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

func (a *Applier) Apply(ctx context.Context, owner client.Object, objs ...client.Object) error {
	for _, obj := range objs {
		if a.setOwnerRef {
			if err := controllerutil.SetControllerReference(owner, obj, a.scheme); err != nil {
				return fmt.Errorf("set owner ref for %T/%s: %w", obj, obj.GetName(), err)
			}
		}

		gvk, err := apiutil.GVKForObject(obj, a.scheme)
		if err != nil {
			return fmt.Errorf("resolve gvk for %T/%s: %w", obj, obj.GetName(), err)
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)

		a.log.Info("Applying",
			"gvk", gvk.String(),
			"ns", obj.GetNamespace(),
			"name", obj.GetName(),
			"fieldOwner", a.fieldOwner,
		)

		if err := a.k8s.Patch(
			ctx,
			obj,
			client.Apply,
			client.FieldOwner(a.fieldOwner),
			client.ForceOwnership,
		); err != nil {
			return fmt.Errorf("apply %s %s/%s: %w", gvk.Kind, obj.GetNamespace(), obj.GetName(), err)
		}
	}
	return nil
}

func (a *Applier) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return a.k8s.Get(ctx, key, obj)
}
