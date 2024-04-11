package subroutines

import (
	"context"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
	openmfpconfig "github.com/openmfp/golang-commons/config"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	"github.com/openmfp/golang-commons/logger"
)

const (
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
	instance := runtimeObj.(*corev1alpha1.Account)

	// Test if namespace was already created based on status
	createdNamespace := &v1.Namespace{}
	if instance.Status.Namespace != nil {

		// Test if namespace exists
		err := r.client.Get(ctx, types.NamespacedName{Name: *instance.Status.Namespace}, createdNamespace)
		if err != nil {
			if kerrors.IsNotFound(err) {

				// Namespace does not exist, create it
				createdNamespace = generateNamespace(instance)
				err = r.client.Create(ctx, createdNamespace)
				if err != nil {
					return ctrl.Result{}, errors.NewOperatorError(err, true, true)
				}
				// Processing completed
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}

		// Namespace exists, verify labels
		err = r.ensureNamespaceLabels(ctx, createdNamespace, instance)
		if err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, true)
		}
	} else {
		if instance.Spec.ExistingNamespace != nil {
			// Verify if namespace exists
			err := r.client.Get(ctx, types.NamespacedName{Name: *instance.Spec.ExistingNamespace}, createdNamespace)
			if err != nil {
				if kerrors.IsNotFound(err) {
					// Provided existing namespace does not exist
					return ctrl.Result{}, errors.NewOperatorError(err, false, false)
				}
				return ctrl.Result{}, errors.NewOperatorError(err, true, true)
			}

			// Namespace exists, ensure labels
			err = r.ensureNamespaceLabels(ctx, createdNamespace, instance)
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

func (r *NamespaceSubroutine) ensureNamespaceLabels(ctx context.Context, ns *v1.Namespace, instance *corev1alpha1.Account) error {
	hasOwnerLabel := verifyLabel(NamespaceAccountOwnerLabel, instance.GetName(), ns.Labels)
	hasOwnerNamespaceLabel := verifyLabel(NamespaceAccountOwnerNamespaceLabel, instance.GetNamespace(), ns.Labels)

	if !hasOwnerLabel || !hasOwnerNamespaceLabel {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels[NamespaceAccountOwnerLabel] = instance.GetName()
		ns.Labels[NamespaceAccountOwnerNamespaceLabel] = instance.GetNamespace()
		err := r.client.Update(ctx, ns)
		if err != nil {
			logger.LoadLoggerFromContext(ctx).Error().Err(err).Msg("Failed to update namespace labels")
			return err
		}
	}
	return nil
}

func verifyLabel(key string, value string, labels map[string]string) bool {
	if val, ok := labels[key]; ok {
		return val == value
	}
	return false
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
