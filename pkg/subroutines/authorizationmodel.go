package subroutines

import (
	"context"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type AuthorizationModelSubroutine struct {
	Client client.Client
}

func NewAuthorizationModelSubroutine(client client.Client) *AuthorizationModelSubroutine {
	return &AuthorizationModelSubroutine{
		Client: client,
	}
}

func (r *AuthorizationModelSubroutine) GetName() string {
	return "AuthorizationModelSubroutine"
}

func (r *AuthorizationModelSubroutine) Finalizers() []string {
	return []string{"authorizationmodel.core.openmfp.io/finalizer"}
}

func (r *AuthorizationModelSubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	authorizationModel := runtimeObj.(*corev1alpha1.AuthorizationModel)

	var owningStore corev1alpha1.Store
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: authorizationModel.Namespace,
		Name:      authorizationModel.Spec.StoreRef.Name,
	}, &owningStore); err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	err := controllerutil.RemoveOwnerReference(&owningStore, authorizationModel, r.Client.Scheme())
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	return ctrl.Result{}, nil
}

func (r *AuthorizationModelSubroutine) Process(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {

	authorizationModel := runtimeObj.(*corev1alpha1.AuthorizationModel)

	var owningStore corev1alpha1.Store
	if err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: authorizationModel.Namespace,
		Name:      authorizationModel.Spec.StoreRef.Name,
	}, &owningStore); err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	err := controllerutil.SetOwnerReference(&owningStore, authorizationModel, r.Client.Scheme())
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	return ctrl.Result{}, nil
}
