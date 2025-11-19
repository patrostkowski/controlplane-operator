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

package utils

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func EnsureCreatedAndOwned(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner client.Object,
	template client.Object,
	log logr.Logger,
	mutate func(obj client.Object) error,
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		obj := template.DeepCopyObject().(client.Object)
		_, err := controllerutil.CreateOrUpdate(ctx, c, obj, func() error {
			if err := controllerutil.SetControllerReference(owner, obj, scheme); err != nil {
				return err
			}
			if mutate != nil {
				return mutate(obj)
			}
			return nil
		})
		return err
	})
}
