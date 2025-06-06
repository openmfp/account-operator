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

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/logger"
	ctrl "sigs.k8s.io/controller-runtime"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
	"github.com/openmfp/account-operator/pkg/subroutines"
)

var (
	operatorName          = "account-operator"
	accountReconcilerName = "AccountReconciler"
)

// AccountReconciler reconciles a Account object
type AccountReconciler struct {
	lifecycle *lifecycle.LifecycleManager
}

func NewAccountReconciler(log *logger.Logger, mgr mcmanager.Manager, cfg config.OperatorConfig, fgaClient openfgav1.OpenFGAServiceClient) mcreconcile.Func {
	localMgr := mgr.GetLocalManager()

	var subs []lifecycle.Subroutine
	if cfg.Subroutines.Workspace.Enabled {
		subs = append(subs, subroutines.NewWorkspaceSubroutine(localMgr.GetClient()))
	}
	if cfg.Subroutines.AccountInfo.Enabled {
		subs = append(subs, subroutines.NewAccountInfoSubroutine(localMgr.GetClient(), string(localMgr.GetConfig().CAData)))
	}
	if cfg.Subroutines.FGA.Enabled {
		subs = append(subs, subroutines.NewFGASubroutine(localMgr.GetClient(), fgaClient, cfg.Subroutines.FGA.CreatorRelation, cfg.Subroutines.FGA.ParentRelation, cfg.Subroutines.FGA.ObjectType))
	}

	reconciler := &AccountReconciler{
		lifecycle: lifecycle.NewLifecycleManager(log, operatorName, accountReconcilerName, localMgr.GetClient(), subs).WithConditionManagement(),
	}

	return mcreconcile.Func(func(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
		log.Info().Str("cluster", req.ClusterName).Str("name", req.Name).Msg("Reconciling Account resource")
		// DEBUG: Log the incoming request
		log.Debug().Interface("request", req).Msg("Received reconcile request")
		cluster, err := mgr.GetCluster(ctx, req.ClusterName)
		if err != nil {
			log.Error().Err(err).Str("cluster", req.ClusterName).Msg("Failed to get cluster in reconcile")
			return ctrl.Result{}, err
		}

		reconciler.lifecycle = lifecycle.NewLifecycleManager(log, operatorName, accountReconcilerName, cluster.GetClient(), subs).WithConditionManagement()

		result, err := reconciler.lifecycle.Reconcile(ctx, req.Request, &corev1alpha1.Account{})
		if err != nil {
			log.Error().Err(err).Str("cluster", req.ClusterName).Str("name", req.Name).Msg("Reconcile error")
		} else {
			log.Debug().Str("cluster", req.ClusterName).Str("name", req.Name).Msg("Reconcile successful")
		}
		return result, err
	})
}
