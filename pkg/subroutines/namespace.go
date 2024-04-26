package subroutines

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
)

const (
	NamespaceSubroutineName             = "NamespaceSubroutine"
	NamespaceSubroutineFinalizer        = "account.core.openmfp.io/finalizer"
	NamespaceAccountOwnerLabel          = "account.core.openmfp.io/owner"
	NamespaceAccountOwnerNamespaceLabel = "account.core.openmfp.io/owner-namespace"
	NamespaceNamePrefix                 = "account-"
)

type NamespaceSubroutine struct {
	client client.Client
}

func NewNamespaceSubroutine(client client.Client) *NamespaceSubroutine {
	return &NamespaceSubroutine{client: client}
}

func (r *NamespaceSubroutine) GetName() string {
	return NamespaceSubroutineName
}

func (r *NamespaceSubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	return ctrl.Result{}, nil
}

func (r *NamespaceSubroutine) Finalizers() []string { // coverage-ignore
	return []string{"account.core.openmfp.io/finalizer"}
}

func (r *NamespaceSubroutine) Process(ctx context.Context, instance lifecycle.RuntimeObject, specObject interface{}, statusObject interface{}) (ctrl.Result, errors.OperatorError) {
	spec := specObject.(corev1alpha1.AccountSpec)
	status := statusObject.(corev1alpha1.AccountStatus)

	// Test if namespace was already created based on status
	createdNamespace := &v1.Namespace{}
	if status.Namespace != nil {
		createdNamespace = generateNamespace(instance, status)
		_, err := controllerutil.CreateOrUpdate(ctx, r.client, createdNamespace, func() error {
			return setNamespaceLabels(createdNamespace, instance)
		})
		if err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}
	} else {
		if spec.Namespace != nil {
			createdNamespace = &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: *spec.Namespace}}
			_, err := controllerutil.CreateOrUpdate(ctx, r.client, createdNamespace, func() error {
				return setNamespaceLabels(createdNamespace, instance)
			})
			if err != nil {
				return ctrl.Result{}, errors.NewOperatorError(err, true, true)
			}
		} else {
			// Create New Namespace
			createdNamespace = generateNamespace(instance, status)
			err := r.client.Create(ctx, createdNamespace)
			if err != nil {
				return ctrl.Result{}, errors.NewOperatorError(err, true, true)
			}
		}
	}

	status.Namespace = &createdNamespace.Name
	return ctrl.Result{}, nil
}

var NamespaceOwnedByAnotherAccountErr = errors.New("Namespace already owned by another account")
var NamespaceOwnedByAnAccountInAnotherNamespaceErr = errors.New("Namespace already owned by another account in another namespace")

func setNamespaceLabels(ns *v1.Namespace, instance lifecycle.RuntimeObject) error {
	hasOwnerLabel := hasLabel(NamespaceAccountOwnerLabel, ns.Labels)
	hasOwnerNamespaceLabel := hasLabel(NamespaceAccountOwnerNamespaceLabel, ns.Labels)
	if hasOwnerLabel && ns.Labels[NamespaceAccountOwnerLabel] != instance.GetName() {
		return NamespaceOwnedByAnotherAccountErr
	}
	if hasOwnerNamespaceLabel && ns.Labels[NamespaceAccountOwnerNamespaceLabel] != instance.GetNamespace() {
		return NamespaceOwnedByAnAccountInAnotherNamespaceErr
	}

	if !hasOwnerLabel || !hasOwnerNamespaceLabel {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels[NamespaceAccountOwnerLabel] = instance.GetName()
		ns.Labels[NamespaceAccountOwnerNamespaceLabel] = instance.GetNamespace()
	}
	return nil
}

func hasLabel(key string, labels map[string]string) bool {
	_, ok := labels[key]
	return ok
}

func generateNamespace(instance lifecycle.RuntimeObject, status corev1alpha1.AccountStatus) *v1.Namespace {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				NamespaceAccountOwnerLabel:          instance.GetName(),
				NamespaceAccountOwnerNamespaceLabel: instance.GetNamespace(),
			},
		},
	}

	if status.Namespace != nil {
		ns.Name = *status.Namespace
	} else {
		ns.ObjectMeta.GenerateName = NamespaceNamePrefix
	}
	return ns
}
