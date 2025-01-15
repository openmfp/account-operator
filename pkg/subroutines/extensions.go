package subroutines

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	tenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	"github.com/kcp-dev/logicalcluster/v3"
	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/kontext"

	"github.com/openmfp/account-operator/api/v1alpha1"
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

func (e *ExtensionSubroutine) Process(ctx context.Context, instance lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	account := instance.(*v1alpha1.Account)

	lookupNamespace := account.GetNamespace()

	extensionsToApply, err := collectExtensions(ctx, e.client, lookupNamespace)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, false)
	}

	for _, extension := range append(extensionsToApply, account.Spec.Extensions...) {
		us := unstructured.Unstructured{}
		us.SetGroupVersionKind(extension.GroupVersionKind())

		if len(extension.MetadataGoTemplate.Raw) > 0 {
			var metadataKeyValues map[string]any
			err := json.NewDecoder(bytes.NewReader(extension.MetadataGoTemplate.Raw)).Decode(&metadataKeyValues)
			if err != nil {
				return ctrl.Result{}, errors.NewOperatorError(err, true, false)
			}

			err = RenderExtensionSpec(ctx, metadataKeyValues, account, &us, []string{"metadata"})
			if err != nil {
				return ctrl.Result{}, errors.NewOperatorError(err, true, false)
			}
		}

		if us.GetName() == "" {
			us.SetName(strings.ToLower(extension.Kind))
		}
		if namespaced, err := e.client.IsObjectNamespaced(&us); err == nil && namespaced {
			us.SetNamespace(*account.Status.Namespace)
		}

		_, err = controllerutil.CreateOrUpdate(ctx, e.client, &us, func() error {
			if len(extension.SpecGoTemplate.Raw) == 0 {
				return nil
			}
			var keyValues map[string]any
			err := json.NewDecoder(bytes.NewReader(extension.SpecGoTemplate.Raw)).Decode(&keyValues)
			if err != nil {
				return err
			}

			path := []string{"spec"}
			return RenderExtensionSpec(ctx, keyValues, account, &us, path)
		})
		if err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, false)
		}
	}

	return ctrl.Result{}, nil
}

func RenderExtensionSpec(ctx context.Context, keyValues map[string]any, account *v1alpha1.Account, us *unstructured.Unstructured, path []string) error {
	for key, value := range keyValues {
		switch val := value.(type) {
		case string: // render string values
			t, err := template.New("field").Funcs(sprig.FuncMap()).Parse(val)
			if err != nil {
				return err
			}

			var intermediate bytes.Buffer
			err = json.NewEncoder(&intermediate).Encode(account)
			if err != nil {
				return err
			}

			var renderAccount map[string]any
			err = json.NewDecoder(&intermediate).Decode(&renderAccount)
			if err != nil {
				return err
			}

			var rendered bytes.Buffer
			err = t.Execute(&rendered, map[string]any{
				"Account": renderAccount,
			})
			if err != nil {
				return err
			}

			err = unstructured.SetNestedField(us.Object, rendered.String(), append(path, key)...)
			if err != nil {
				return err
			}
		case map[string]any:
			return RenderExtensionSpec(ctx, val, account, us, append(path, key))
		default: // any other primitive type
			err := unstructured.SetNestedField(us.Object, val, append(path, key)...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *ExtensionSubroutine) Finalize(ctx context.Context, instance lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	account := instance.(*v1alpha1.Account)

	lookupNamespace := account.GetNamespace()

	extensionsToRemove, err := collectExtensions(ctx, e.client, lookupNamespace)
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, false)
	}

	for _, extension := range append(extensionsToRemove, account.Spec.Extensions...) {
		us := unstructured.Unstructured{}
		us.SetGroupVersionKind(extension.GroupVersionKind())

		us.SetName(strings.ToLower(extension.Kind))
		if namespaced, err := e.client.IsObjectNamespaced(&us); err == nil && namespaced {
			us.SetNamespace(*account.Status.Namespace)
		}

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

func collectExtensions(ctx context.Context, cl client.Client, lookupNamespace string) ([]v1alpha1.Extension, error) {
	var extensions []v1alpha1.Extension
	for {
		parentAccount, newClusterContext, err := getParentAccount(ctx, cl, lookupNamespace)
		if errors.Is(err, ErrNoParentAvailable) {
			break
		}
		if err != nil {
			return nil, err
		}

		if newClusterContext != nil {
			ctx = kontext.WithCluster(ctx, logicalcluster.Name(*newClusterContext))
		}

		lookupNamespace = parentAccount.GetNamespace()

		extensions = append(extensions, parentAccount.Spec.Extensions...)
	}

	return extensions, nil
}

func getParentAccount(ctx context.Context, cl client.Client, ns string) (*v1alpha1.Account, *string, error) {

	if cluster, ok := kontext.ClusterFrom(ctx); ok && !cluster.Empty() {

		wsCtx := kontext.WithCluster(ctx, logicalcluster.Name(""))
		list := &tenancyv1alpha1.WorkspaceList{}

		err := cl.List(wsCtx, list)
		if err != nil {
			return nil, nil, err
		}

		for _, ws := range list.Items {
			if ws.Spec.Cluster != cluster.String() {
				continue
			}

			clusterName := ws.Annotations[logicalcluster.AnnotationKey]

			parentCtx := kontext.WithCluster(ctx, logicalcluster.Name(clusterName))

			parentAccount := v1alpha1.Account{}
			err = cl.Get(parentCtx, types.NamespacedName{
				Name:      ws.Annotations[v1alpha1.NamespaceAccountOwnerLabel],
				Namespace: ns,
			}, &parentAccount)
			if err != nil {
				return nil, nil, err
			}

			return &parentAccount, &clusterName, nil
		}

		return nil, nil, ErrNoParentAvailable
	}

	var namespace v1.Namespace
	err := cl.Get(ctx, types.NamespacedName{Name: ns}, &namespace)
	if kerrors.IsNotFound(err) {
		return nil, nil, ErrNoParentAvailable
	}
	if err != nil {
		return nil, nil, err
	}

	accountName, ok := namespace.GetLabels()[v1alpha1.NamespaceAccountOwnerLabel]
	if !ok || accountName == "" {
		return nil, nil, ErrNoParentAvailable
	}

	accountNamespace, ok := namespace.GetLabels()[v1alpha1.NamespaceAccountOwnerNamespaceLabel]
	if !ok || accountNamespace == "" {
		return nil, nil, ErrNoParentAvailable
	}

	var account v1alpha1.Account
	err = cl.Get(ctx, types.NamespacedName{Name: accountName, Namespace: accountNamespace}, &account)
	if kerrors.IsNotFound(err) {
		return nil, nil, ErrNoParentAvailable
	}
	if err != nil {
		return nil, nil, err
	}

	return &account, nil, nil
}
