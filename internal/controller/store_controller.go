/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/logger"
)

// StoreReconciler reconciles a Store object
type StoreReconciler struct {
	lifecycle *lifecycle.LifecycleManager
}

//+kubebuilder:rbac:groups=core.openmfp.io,resources=stores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openmfp.io,resources=stores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.openmfp.io,resources=stores/finalizers,verbs=update

func NewStoreReconciler(mgr ctrl.Manager, log *logger.Logger, cfg config.Config) (*StoreReconciler, error) {

	conn, err := grpc.NewClient(cfg.Subroutines.Store.OpenFGAURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	fgaClient := openfgav1.NewOpenFGAServiceClient(conn)

	subs := []lifecycle.Subroutine{
		subroutines.NewStoreSubroutine(mgr.GetClient(), fgaClient),
	}

	return &StoreReconciler{
		lifecycle: lifecycle.NewLifecycleManager(log, operatorName, "StoreReconciler", mgr.GetClient(), subs).
			WithConditionManagement(),
	}, nil
}

func (r *StoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.lifecycle.Reconcile(ctx, req, &corev1alpha1.Store{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *StoreReconciler) SetupWithManager(mgr ctrl.Manager) error {

	err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1alpha1.AuthorizationModel{}, ".spec.storeRef.name", func(o client.Object) []string {
		store := o.(*corev1alpha1.AuthorizationModel).Spec.StoreRef.Name
		return []string{store}
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Store{}).
		Watches(&corev1alpha1.AuthorizationModel{}, handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &corev1alpha1.Store{})).
		Complete(r)
}
