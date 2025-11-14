package controlplane

import (
	"context"
	"fmt"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ConditionReady = "Ready"

// setCondition is the core helper: update a condition and write .status if changed.
func setCondition(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	conds *[]metav1.Condition,
	condType string,
	condStatus metav1.ConditionStatus,
	reason, msg string,
) error {
	cond := metav1.Condition{
		Type:               condType,
		Status:             condStatus,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: obj.GetGeneration(),
	}

	changed := apimeta.SetStatusCondition(conds, cond)
	if !changed {
		// nothing changed, skip API call
		return nil
	}

	if err := c.Status().Update(ctx, obj); err != nil {
		return fmt.Errorf("updating status for %T %s/%s: %w",
			obj, obj.GetNamespace(), obj.GetName(), err)
	}

	return nil
}

// MarkReady sets Ready=True.
func MarkReady(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	conds *[]metav1.Condition,
	reason, msg string,
) error {
	return setCondition(ctx, c, obj, conds, ConditionReady, metav1.ConditionTrue, reason, msg)
}

// MarkNotReady sets Ready=False.
func MarkNotReady(
	ctx context.Context,
	c client.Client,
	obj client.Object,
	conds *[]metav1.Condition,
	reason, msg string,
) error {
	return setCondition(ctx, c, obj, conds, ConditionReady, metav1.ConditionFalse, reason, msg)
}

// IsConditionTrue checks if given condition is true.
func IsConditionTrue(conds []metav1.Condition, condType string) bool {
	c := apimeta.FindStatusCondition(conds, condType)
	return c != nil && c.Status == metav1.ConditionTrue
}
