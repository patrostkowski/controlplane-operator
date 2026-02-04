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

package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DataFromSecret(ctx context.Context, c client.Reader, obj client.ObjectKey, key string) ([]byte, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: obj.Namespace,
		Name:      obj.Name,
	}

	if err := c.Get(ctx, secretKey, secret); err != nil {
		return nil, err
	}

	return toBytes(secret, key)
}

func toBytes(out *corev1.Secret, key string) ([]byte, error) {
	data, ok := out.Data[key]
	if !ok {
		return nil, errors.Errorf("missing key %q in secret data", key)
	}
	return data, nil
}
