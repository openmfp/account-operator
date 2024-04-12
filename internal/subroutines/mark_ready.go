package subroutines

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
)

const (
	MarkReadySubroutineName = "MarkReadySubroutine"
)

type MarkReadySubroutine struct{}

func NewMarkReadySubroutine() *MarkReadySubroutine {
	return &MarkReadySubroutine{}
}

func (r *MarkReadySubroutine) GetName() string { // coverage-ignore
	return MarkReadySubroutineName
}

func (r *MarkReadySubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) { // coverage-ignore
	return ctrl.Result{}, nil
}

func (r *MarkReadySubroutine) Finalizers() []string { // coverage-ignore
	return []string{}
}

func (r *MarkReadySubroutine) Process(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	instance := runtimeObj.(*corev1alpha1.Account)
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = []metav1.Condition{}
	}

	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    corev1alpha1.ConditionAccountReady,
		Status:  metav1.ConditionTrue,
		Message: "The account is ready",
		Reason:  corev1alpha1.ConditionAccountReady,
	})

	return ctrl.Result{}, nil
}
