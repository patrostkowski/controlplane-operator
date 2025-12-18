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
