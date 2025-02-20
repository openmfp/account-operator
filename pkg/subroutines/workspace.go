package subroutines

import (
	"context"

	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	WorkspaceSubroutineName      = "WorkspaceSubroutine"
	WorkspaceSubroutineFinalizer = "account.core.openmfp.org/finalizer"
)

type WorkspaceSubroutine struct {
	client client.Client
}

func NewWorkspaceSubroutine(client client.Client) *WorkspaceSubroutine {
	return &WorkspaceSubroutine{client: client}
}

func (r *WorkspaceSubroutine) GetName() string {
	return WorkspaceSubroutineName
}

func (r *WorkspaceSubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	//instance := runtimeObj.(*corev1alpha1.Account)
	//
	//if instance.Status.Workspace == nil {
	//	return ctrl.Result{}, nil
	//}
	//
	//ns := v1.Workspace{}
	//err := r.client.Get(ctx, client.ObjectKey{Name: *instance.Status.Workspace}, &ns)
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
	//createdNamespace := &v1.Workspace{}
	//if instance.Status.Workspace != nil {
	//	createdNamespace = generateNamespace(instance)
	//	_, err := controllerutil.CreateOrUpdate(ctx, r.client, createdNamespace, func() error {
	//		return setNamespaceLabels(createdNamespace, instance)
	//	})
	//	if err != nil {
	//		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	//	}
	//} else {
	//	if instance.Spec.Workspace != nil {
	//		createdNamespace = &v1.Workspace{ObjectMeta: metav1.ObjectMeta{Name: *instance.Spec.Workspace}}
	//		_, err := controllerutil.CreateOrUpdate(ctx, r.client, createdNamespace, func() error {
	//			return setNamespaceLabels(createdNamespace, instance)
	//		})
	//		if err != nil {
	//			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	//		}
	//	} else {
	//		// Create New Workspace
	//		createdNamespace = generateNamespace(instance)
	//		err := r.client.Create(ctx, createdNamespace)
	//		if err != nil {
	//			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	//		}
	//	}
	//}
	//
	//instance.Status.Workspace = &createdNamespace.Name
	return ctrl.Result{}, nil
}

//
//var NamespaceOwnedByAnotherAccountErr = errors.New("Workspace already owned by another account")
//var NamespaceOwnedByAnAccountInAnotherNamespaceErr = errors.New("Workspace already owned by another account in another namespace")
//
//func setNamespaceLabels(ns *v1.Workspace, instance *corev1alpha1.Account) error {
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
//func generateNamespace(instance *corev1alpha1.Account) *v1.Workspace {
//	ns := &v1.Workspace{
//		ObjectMeta: metav1.ObjectMeta{
//			Labels: map[string]string{
//				corev1alpha1.NamespaceAccountOwnerLabel:          instance.GetName(),
//				corev1alpha1.NamespaceAccountOwnerNamespaceLabel: instance.GetNamespace(),
//			},
//		},
//	}
//
//	if instance.Status.Workspace != nil {
//		ns.Name = *instance.Status.Workspace
//	} else {
//		ns.ObjectMeta.GenerateName = NamespaceNamePrefix
//	}
//	return ns
//}
