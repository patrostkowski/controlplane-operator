package utils

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateCondition updates (or creates) a condition on the given object's status.
// It returns error only if the Status().Update fails.
func UpdateCondition(
	ctx context.Context,
	c client.StatusWriter,
	obj client.Object,
	conditions *[]metav1.Condition,
	condType string,
	conditionStatus metav1.ConditionStatus,
	reason, message string,
) error {
	cond := metav1.Condition{
		Type:               condType,
		Status:             conditionStatus,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: obj.GetGeneration(),
	}

	// SetStatusCondition returns true if it changed anything
	if meta.SetStatusCondition(conditions, cond) {
		return c.Update(ctx, obj)
	}
	return nil
}
