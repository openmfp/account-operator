package subroutines

import (
	"context"
	"strings"

	v1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ExtensionSubroutine struct {
	client client.Client
}

func NewExtensionSubroutine(cl client.Client) *ExtensionSubroutine {
	return &ExtensionSubroutine{client: cl}
}

var (
	ErrNoParentAvailable = errors.New("no parent namespace available")
)

func (e *ExtensionSubroutine) collectExtensions(ctx context.Context, lookupNamespace string) ([]v1alpha1.Extension, error) {
	var extensions []v1alpha1.Extension
	for {
		parentAccount, err := e.getParentAccount(ctx, lookupNamespace)
		if errors.Is(err, ErrNoParentAvailable) {
			break
		}
		if err != nil {
			return nil, err
		}

		lookupNamespace = parentAccount.GetNamespace()

		extensions = append(extensions, parentAccount.Spec.Extensions...)
	}

	return extensions, nil
}

func (e *ExtensionSubroutine) Process(ctx context.Context, instance lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	account := instance.(*v1alpha1.Account)

	lookupNamespace := account.GetNamespace()

	extensionsToApply, err := e.collectExtensions(ctx, lookupNamespace)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, false)
	}

	for _, extension := range append(extensionsToApply, account.Spec.Extensions...) {
		us := unstructured.Unstructured{}
		us.SetGroupVersionKind(extension.GroupVersionKind())

		us.SetName(strings.ToLower(extension.GroupVersionKind().Kind))
		us.SetNamespace(*account.Status.Namespace)

		_, err := controllerutil.CreateOrUpdate(ctx, e.client, &us, func() error {
			c := us.UnstructuredContent()
			c["spec"] = extension.SpecGoTemplate
			return nil
		})
		if err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, false)
		}
	}

	return ctrl.Result{}, nil
}

func (e *ExtensionSubroutine) Finalize(ctx context.Context, instance lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	account := instance.(*v1alpha1.Account)

	lookupNamespace := account.GetNamespace()

	extensionsToRemove, err := e.collectExtensions(ctx, lookupNamespace)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, false)
	}

	for _, extension := range append(extensionsToRemove, account.Spec.Extensions...) {
		us := unstructured.Unstructured{}
		us.SetGroupVersionKind(extension.GroupVersionKind())

		us.SetName(strings.ToLower(extension.GroupVersionKind().Kind))
		us.SetNamespace(*account.Status.Namespace)

		err := e.client.Delete(ctx, &us)
		if kerrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, false)
		}
	}

	return ctrl.Result{}, nil
}

func (e *ExtensionSubroutine) GetName() string { return "ExtensionSubroutine" }

func (e *ExtensionSubroutine) Finalizers() []string { return []string{} }

func (e *ExtensionSubroutine) getParentAccount(ctx context.Context, ns string) (*v1alpha1.Account, error) {
	var namespace v1.Namespace
	err := e.client.Get(ctx, types.NamespacedName{Name: ns}, &namespace)
	if kerrors.IsNotFound(err) {
		return nil, ErrNoParentAvailable
	}
	if err != nil {
		return nil, err
	}

	accountName, ok := namespace.GetLabels()[NamespaceAccountOwnerLabel]
	if !ok || accountName == "" {
		return nil, ErrNoParentAvailable
	}

	accountNamespace, ok := namespace.GetLabels()[NamespaceAccountOwnerNamespaceLabel]
	if !ok || accountNamespace == "" {
		return nil, ErrNoParentAvailable
	}

	var account v1alpha1.Account
	err = e.client.Get(ctx, types.NamespacedName{Name: accountName, Namespace: accountNamespace}, &account)
	if kerrors.IsNotFound(err) {
		return nil, ErrNoParentAvailable
	}
	if err != nil {
		return nil, err
	}

	return &account, nil
}
