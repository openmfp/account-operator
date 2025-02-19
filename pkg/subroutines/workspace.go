package subroutines

import (
	"context"

	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NamespaceSubroutineName      = "WorkspaceSubroutine"
	NamespaceSubroutineFinalizer = "account.core.openmfp.org/finalizer"
	NamespaceNamePrefix          = "account-"
)

type WorkspaceSubroutine struct {
	client client.Client
}

func NewWorkspaceSubroutine(client client.Client) *WorkspaceSubroutine {
	return &WorkspaceSubroutine{client: client}
}

func (r *WorkspaceSubroutine) GetName() string {
	return NamespaceSubroutineName
}

func (r *WorkspaceSubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	//instance := runtimeObj.(*corev1alpha1.Account)
	//
	//if instance.Status.Namespace == nil {
	//	return ctrl.Result{}, nil
	//}
	//
	//ns := v1.Namespace{}
	//err := r.client.Get(ctx, client.ObjectKey{Name: *instance.Status.Namespace}, &ns)
	//if kerrors.IsNotFound(err) {
	//	return ctrl.Result{}, nil
	//}
	//if err != nil {
	//	return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	//}
	//
	//if ns.GetDeletionTimestamp() != nil {
	//	return ctrl.Result{Requeue: true}, nil
	//}
	//
	//err = r.client.Delete(ctx, &ns)
	//if err != nil {
	//	return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	//}
	//
	//return ctrl.Result{Requeue: true}, nil // we need to requeue to check if the namespace was deleted
	return ctrl.Result{}, nil
}

func (r *WorkspaceSubroutine) Finalizers() []string { // coverage-ignore
	return []string{"account.core.openmfp.org/finalizer"}
}

func (r *WorkspaceSubroutine) Process(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	//instance := runtimeObj.(*corev1alpha1.Account)
	//
	//// Test if namespace was already created based on status
	//createdNamespace := &v1.Namespace{}
	//if instance.Status.Namespace != nil {
	//	createdNamespace = generateNamespace(instance)
	//	_, err := controllerutil.CreateOrUpdate(ctx, r.client, createdNamespace, func() error {
	//		return setNamespaceLabels(createdNamespace, instance)
	//	})
	//	if err != nil {
	//		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	//	}
	//} else {
	//	if instance.Spec.Namespace != nil {
	//		createdNamespace = &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: *instance.Spec.Namespace}}
	//		_, err := controllerutil.CreateOrUpdate(ctx, r.client, createdNamespace, func() error {
	//			return setNamespaceLabels(createdNamespace, instance)
	//		})
	//		if err != nil {
	//			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	//		}
	//	} else {
	//		// Create New Namespace
	//		createdNamespace = generateNamespace(instance)
	//		err := r.client.Create(ctx, createdNamespace)
	//		if err != nil {
	//			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	//		}
	//	}
	//}
	//
	//instance.Status.Namespace = &createdNamespace.Name
	return ctrl.Result{}, nil
}

//
//var NamespaceOwnedByAnotherAccountErr = errors.New("Namespace already owned by another account")
//var NamespaceOwnedByAnAccountInAnotherNamespaceErr = errors.New("Namespace already owned by another account in another namespace")
//
//func setNamespaceLabels(ns *v1.Namespace, instance *corev1alpha1.Account) error {
//	accountOwner, hasOwnerLabel := ns.Labels[corev1alpha1.NamespaceAccountOwnerLabel]
//	accountOwnerNamespace, hasOwnerNamespaceLabel := ns.Labels[corev1alpha1.NamespaceAccountOwnerNamespaceLabel]
//
//	if hasOwnerLabel && accountOwner != instance.GetName() {
//		return NamespaceOwnedByAnotherAccountErr
//	}
//
//	if hasOwnerNamespaceLabel && accountOwnerNamespace != instance.GetNamespace() {
//		return NamespaceOwnedByAnAccountInAnotherNamespaceErr
//	}
//
//	if !hasOwnerLabel || !hasOwnerNamespaceLabel {
//		if ns.Labels == nil {
//			ns.Labels = make(map[string]string)
//		}
//		ns.Labels[corev1alpha1.NamespaceAccountOwnerLabel] = instance.GetName()
//		ns.Labels[corev1alpha1.NamespaceAccountOwnerNamespaceLabel] = instance.GetNamespace()
//	}
//
//	return nil
//}
//
//func generateNamespace(instance *corev1alpha1.Account) *v1.Namespace {
//	ns := &v1.Namespace{
//		ObjectMeta: metav1.ObjectMeta{
//			Labels: map[string]string{
//				corev1alpha1.NamespaceAccountOwnerLabel:          instance.GetName(),
//				corev1alpha1.NamespaceAccountOwnerNamespaceLabel: instance.GetNamespace(),
//			},
//		},
//	}
//
//	if instance.Status.Namespace != nil {
//		ns.Name = *instance.Status.Namespace
//	} else {
//		ns.ObjectMeta.GenerateName = NamespaceNamePrefix
//	}
//	return ns
//}
