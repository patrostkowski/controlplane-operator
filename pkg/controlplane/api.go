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

package controlplane

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *ControlPlaneClient) ApplySSA(ctx context.Context, fieldOwner string, objs ...client.Object) error {
	for _, obj := range objs {
		if err := c.Patch(
			ctx,
			obj,
			client.Apply,
			client.FieldOwner(fieldOwner),
			client.ForceOwnership,
		); err != nil {
			return err
		}
	}
	return nil
}
