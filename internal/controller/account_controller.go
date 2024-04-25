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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/logger"
)

var (
	operatorName          = "account-operator"
	accountReconcilerName = "AccountReconciler"
)

// AccountReconciler reconciles a Account object
type AccountReconciler struct {
	lifecycle *lifecycle.LifecycleManager
}

func NewAccountReconciler(log *logger.Logger, mgr ctrl.Manager, cfg config.Config) *AccountReconciler {
	subs := []lifecycle.Subroutine{}
	if cfg.Subroutines.Namespace.Enabled {
		subs = append(subs, subroutines.NewNamespaceSubroutine(mgr.GetClient()))
	}
	if cfg.Subroutines.Extension.Enabled {
		subs = append(subs, subroutines.NewExtensionSubroutine(mgr.GetClient()))
	}
	if cfg.Subroutines.ExtensionReady.Enabled {
		subs = append(subs, subroutines.NewExtensionReadySubroutine(mgr.GetClient()))
	}
	return &AccountReconciler{
		lifecycle: lifecycle.NewLifecycleManager(log, operatorName, accountReconcilerName, mgr.GetClient(), subs).WithSpreadingReconciles().WithConditionManagement(),
	}
}

func (r *AccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.lifecycle.Reconcile(ctx, req, &corev1alpha1.Account{})
}

func (r *AccountReconciler) SetupWithManager(mgr ctrl.Manager, cfg config.Config, log *logger.Logger, eventPredicates ...predicate.Predicate) error {
	return r.lifecycle.SetupWithManager(mgr, cfg.MaxConcurrentReconciles, accountReconcilerName, &corev1alpha1.Account{}, cfg.DebugLabelValue, r, log, eventPredicates...)
}
