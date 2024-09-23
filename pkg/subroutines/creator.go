package subroutines

import (
	"context"
	"fmt"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	"github.com/openmfp/golang-commons/fga/helpers"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"

	v1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/service"
)

type CreatorSubroutine struct {
	client        openfgav1.OpenFGAServiceClient
	srv           service.Service
	rootNamespace string
}

func NewCreatorSubroutine(cl openfgav1.OpenFGAServiceClient, s service.Service, rootNamespace string) *CreatorSubroutine {
	return &CreatorSubroutine{client: cl, srv: s, rootNamespace: rootNamespace}
}

func (e *CreatorSubroutine) Process(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	account := runtimeObj.(*v1alpha1.Account)

	if account.Spec.Creator == nil {
		return ctrl.Result{}, nil
	}

	if meta.IsStatusConditionTrue(account.Status.Conditions, "CreatorSubroutine_Ready") {
		return ctrl.Result{}, nil
	}

	storeId, err := e.storeId(ctx)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	_, err = e.client.Write(ctx, &openfgav1.WriteRequest{
		StoreId: storeId,
		Writes: &openfgav1.WriteRequestWrites{
			TupleKeys: []*openfgav1.TupleKey{
				{
					Object:   fmt.Sprintf("account:%s", account.Name),
					Relation: "owner",
					User:     fmt.Sprintf("user:%s", *account.Spec.Creator),
				},
			},
		},
	})

	if helpers.IsDuplicateWriteError(err) {
		err = nil
	}

	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	return ctrl.Result{}, nil
}

func (e *CreatorSubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	account := runtimeObj.(*v1alpha1.Account)

	if account.Spec.Creator == nil {
		return ctrl.Result{}, nil
	}

	storeId, err := e.storeId(ctx)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	_, err = e.client.Write(ctx, &openfgav1.WriteRequest{
		StoreId: storeId,
		Deletes: &openfgav1.WriteRequestDeletes{
			TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
				{
					Object:   fmt.Sprintf("account:%s", account.Name),
					Relation: "owner",
					User:     fmt.Sprintf("user:%s", *account.Spec.Creator),
				},
			},
		},
	})

	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	return ctrl.Result{}, nil
}

func (e *CreatorSubroutine) storeId(ctx context.Context) (string, error) {
	a, err := e.srv.GetFirstLevelAccountForNamespace(ctx, e.rootNamespace)
	if err != nil {
		return "", err
	}

	storeId, err := helpers.GetStoreIDForTenant(ctx, e.client, a.Name)
	if err != nil {
		return "", err
	}

	return storeId, nil
}

func (e *CreatorSubroutine) GetName() string { return "CreatorSubroutine" }

func (e *CreatorSubroutine) Finalizers() []string { return []string{} }
