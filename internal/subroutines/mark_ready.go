package subroutines

import (
	"context"

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
	setStatusCondition(&instance.Status.Conditions, metav1.ConditionTrue, corev1alpha1.ConditionAccountReady, corev1alpha1.ConditionAccountReady, "The account is ready")
	return ctrl.Result{}, nil
}
