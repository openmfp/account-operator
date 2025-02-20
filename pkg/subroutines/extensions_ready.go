package subroutines

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmfp/golang-commons/controller/lifecycle"
	"github.com/openmfp/golang-commons/errors"

	"github.com/openmfp/account-operator/api/v1alpha1"
)

type ExtensionReadySubroutine struct {
	client client.Client
}

func NewExtensionReadySubroutine(cl client.Client) *ExtensionReadySubroutine {
	return &ExtensionReadySubroutine{client: cl}
}

func (e *ExtensionReadySubroutine) GetName() string { return "ExtensionReadySubroutine" }

func (e *ExtensionReadySubroutine) Finalizers() []string { return []string{} }

func (e *ExtensionReadySubroutine) Process(ctx context.Context, instance lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {

	extensions, err := collectExtensions(ctx, e.client, instance.GetNamespace())
	if err != nil {
		return ctrl.Result{}, errors.NewOperatorError(err, true, false)
	}

	account := instance.(*v1alpha1.Account)

	for _, extension := range append(extensions, account.Spec.Extensions...) {
		if extension.ReadyConditionType == nil {
			continue
		}

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
			us.SetNamespace(*account.Status.Workspace)
		}

		err = e.client.Get(ctx, client.ObjectKeyFromObject(&us), &us)
		if kerrors.IsNotFound(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		if err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, false)
		}

		conditions, hasField, err := unstructured.NestedSlice(us.Object, "status", "conditions")
		if !hasField || err != nil {
			return ctrl.Result{}, errors.NewOperatorError(err, true, false)
		}

		parsedConditions := make([]metav1.Condition, len(conditions))
		for i, cond := range conditions {

			intermediate, err := json.Marshal(cond)
			if err != nil { // coverage-ignore
				return ctrl.Result{}, errors.NewOperatorError(err, true, false)
			}

			var parsed metav1.Condition
			err = json.NewDecoder(bytes.NewReader(intermediate)).Decode(&parsed)
			if err != nil { // coverage-ignore
				return ctrl.Result{}, errors.NewOperatorError(err, true, false)
			}

			parsedConditions[i] = parsed
		}

		if meta.IsStatusConditionFalse(parsedConditions, *extension.ReadyConditionType) {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (e *ExtensionReadySubroutine) Finalize(_ context.Context, _ lifecycle.RuntimeObject) (ctrl.Result, errors.OperatorError) {
	return ctrl.Result{}, nil
}
