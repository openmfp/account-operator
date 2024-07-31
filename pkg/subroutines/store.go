package subroutines

import (
	"context"
	"path/filepath"
	"slices"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	language "github.com/openfga/language/pkg/go/transformer"
	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StoreSubroutine struct {
	Client    client.Client
	FGAClient openfgav1.OpenFGAServiceClient
}

func NewStoreSubroutine(client client.Client, fgaClient openfgav1.OpenFGAServiceClient) *StoreSubroutine {
	return &StoreSubroutine{
		Client:    client,
		FGAClient: fgaClient,
	}
}

func (r *StoreSubroutine) GetName() string {
	return "StoreSubroutine"
}

func (r *StoreSubroutine) Finalizers() []string {
	return []string{"store.core.openmfp.io/finalizer"}
}

func (r *StoreSubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	// TODO: should the store be deleted on finalization
	return ctrl.Result{}, nil
}

func (r *StoreSubroutine) Process(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {

	store := runtimeObj.(*corev1alpha1.Store)

	var coreModel corev1alpha1.AuthorizationModel
	err := r.Client.Get(ctx, client.ObjectKey{Name: store.Spec.CoreModule.Name, Namespace: store.Namespace}, &coreModel)
	if kerrors.IsNotFound(err) { // TODO: is this fine or should we handle this differently?
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, false, true)
	}

	if !meta.IsStatusConditionTrue(coreModel.Status.Conditions, lifecycle.ConditionReady) {
		return ctrl.Result{Requeue: true}, nil
	}

	if store.Status.StoreID == "" {
		stores, err := r.FGAClient.ListStores(ctx, &openfgav1.ListStoresRequest{})
		if err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}

		if idx := slices.IndexFunc(stores.Stores, func(s *openfgav1.Store) bool {
			return s.Name == store.Name
		}); idx != -1 {
			store.Status.StoreID = stores.Stores[idx].Id
		} else {
			fgaStore, err := r.FGAClient.CreateStore(ctx, &openfgav1.CreateStoreRequest{
				Name: store.Name,
			})
			if err != nil {
				return ctrl.Result{}, errors.NewOperatorError(err, true, true)
			}

			store.Status.StoreID = fgaStore.Id
		}
	}

	var extendingModules corev1alpha1.AuthorizationModelList
	if err := r.Client.List(ctx, &extendingModules, client.MatchingFields{".spec.storeRef.name": store.Name}); err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	moduleFiles := []language.ModuleFile{}
	for _, extendingModule := range extendingModules.Items {
		moduleFiles = append(moduleFiles, language.ModuleFile{
			Name:     filepath.Join(extendingModule.Namespace, extendingModule.Name+".fga"),
			Contents: extendingModule.Spec.Model,
		})
	}

	finalModel, err := language.TransformModuleFilesToModel(moduleFiles, "1.2")
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	_, err = r.FGAClient.WriteAuthorizationModel(ctx, &openfgav1.WriteAuthorizationModelRequest{
		StoreId:         store.Status.StoreID,
		TypeDefinitions: finalModel.TypeDefinitions,
		SchemaVersion:   finalModel.SchemaVersion,
		Conditions:      finalModel.Conditions,
	})
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, true)
	}

	return ctrl.Result{}, nil
}
