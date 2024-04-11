package subroutines

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
	openmfpconfig "github.com/openmfp/golang-commons/config"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	"github.com/openmfp/golang-commons/logger"
)

const NamespaceSubroutineFinalizer = "account.core.openmfp.io/finalizer"

type NamespaceSubroutine struct {
	client client.Client
}

func NewNamespaceSubroutine(mgr ctrl.Manager) *NamespaceSubroutine {
	return &NamespaceSubroutine{client: mgr.GetClient()}
}

func (r *NamespaceSubroutine) GetName() string {
	return NamespaceSubroutineFinalizer
}

func (r *NamespaceSubroutine) Finalize(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	log := logger.LoadLoggerFromContext(ctx)
	cfg := openmfpconfig.LoadConfigFromContext(ctx).(config.Config)
	instance := runtimeObj.(*corev1alpha1.Account)
	log.Info().Bool("enabled", cfg.Subroutines.Namespace.Enabled).Str("name", instance.GetName()).Msg("Finalizing NamespaceSubroutine")
	return ctrl.Result{}, nil
}

func (r *NamespaceSubroutine) Finalizers() []string {
	return []string{"account.core.openmfp.io/finalizer"}
}

func (r *NamespaceSubroutine) Process(ctx context.Context, runtimeObj lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	log := logger.LoadLoggerFromContext(ctx)
	cfg := openmfpconfig.LoadConfigFromContext(ctx).(config.Config)
	instance := runtimeObj.(*corev1alpha1.Account)
	log.Info().Bool("enabled", cfg.Subroutines.Namespace.Enabled).Str("name", instance.GetName()).Msg("Processing NamespaceSubroutine")
	return ctrl.Result{}, nil

}
