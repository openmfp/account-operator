package subroutines

import (
	"context"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
	instance := runtimeObj.(*corev1alpha1.Account)

	if instance.Status.Namespace == nil {
		return ctrl.Result{}, nil
	}

	ns := v1.Namespace{}
	err := r.client.Get(ctx, client.ObjectKey{Name: *instance.Status.Namespace}, &ns)
	if kerrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	err = r.client.Delete(ctx, &ns)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	return ctrl.Result{Requeue: true}, nil // we need to requeue to check if the namespace was deleted
}

func (r *NamespaceSubroutine) Finalizers() []string { // coverage-ignore
	return []string{"account.core.openmfp.io/finalizer"}
}

func (r *NamespaceSubroutine) Process(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	instance := runtimeObj.(*corev1alpha1.Account)

	// Test if namespace was already created based on status
	createdNamespace := &v1.Namespace{}
	if instance.Status.Namespace != nil {
		createdNamespace = generateNamespace(instance)
		_, err := controllerutil.CreateOrUpdate(ctx, r.client, createdNamespace, func() error {
			return setNamespaceLabels(createdNamespace, instance)
		})
		if err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}
	} else {
		if instance.Spec.Namespace != nil {
			createdNamespace = &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: *instance.Spec.Namespace}}
			_, err := controllerutil.CreateOrUpdate(ctx, r.client, createdNamespace, func() error {
				return setNamespaceLabels(createdNamespace, instance)
			})
			if err != nil {
				return ctrl.Result{}, errors.NewOperatorError(err, true, true)
			}
		} else {
			// Create New Namespace
			createdNamespace = generateNamespace(instance)
			err := r.client.Create(ctx, createdNamespace)
			if err != nil {
				return ctrl.Result{}, errors.NewOperatorError(err, true, true)
			}
		}
	}

	instance.Status.Namespace = &createdNamespace.Name
	return ctrl.Result{}, nil
}

var NamespaceOwnedByAnotherAccountErr = errors.New("Namespace already owned by another account")
var NamespaceOwnedByAnAccountInAnotherNamespaceErr = errors.New("Namespace already owned by another account in another namespace")

func setNamespaceLabels(ns *v1.Namespace, instance *corev1alpha1.Account) error {
	accountOwner, hasOwnerLabel := ns.Labels[NamespaceAccountOwnerLabel]
	accountOwnerNamespace, hasOwnerNamespaceLabel := ns.Labels[NamespaceAccountOwnerNamespaceLabel]

	if hasOwnerLabel && accountOwner != instance.GetName() {
		return NamespaceOwnedByAnotherAccountErr
	}

	if hasOwnerNamespaceLabel && accountOwnerNamespace != instance.GetNamespace() {
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

func generateNamespace(instance *corev1alpha1.Account) *v1.Namespace {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				NamespaceAccountOwnerLabel:          instance.GetName(),
				NamespaceAccountOwnerNamespaceLabel: instance.GetNamespace(),
			},
		},
	}

	if instance.Status.Namespace != nil {
		ns.Name = *instance.Status.Namespace
	} else {
		ns.ObjectMeta.GenerateName = NamespaceNamePrefix
	}
	return ns
}
