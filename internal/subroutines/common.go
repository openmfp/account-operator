package subroutines

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setStatusCondition(conditions *[]metav1.Condition, status metav1.ConditionStatus, conditionType, reason string, message string) {
	condition := meta.FindStatusCondition(*conditions, conditionType)
	if condition == nil || condition.Status != status || condition.Reason != reason || condition.Message != message {
		newCond := metav1.Condition{
			Type:    conditionType,
			Status:  status,
			Message: message,
			Reason:  reason,
		}

		meta.SetStatusCondition(conditions, newCond)
	}
}
