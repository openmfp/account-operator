package subroutines_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/account-operator/pkg/subroutines/mocks"
)

func TestExtensionReadyInterfaceFunction(t *testing.T) {
	routine := subroutines.NewExtensionReadySubroutine(nil)
	assert.Equal(t, "ExtensionReadySubroutine", routine.GetName())
	assert.Equal(t, []string{}, routine.Finalizers())
	_, err := routine.Finalize(context.Background(), nil)
	assert.Nil(t, err)
}

func TestExtensionReadySubroutine(t *testing.T) {
	readyCondition := "Ready"
	defaultNamespace := "default"

	tests := []struct {
		name           string
		k8sMocks       func(*mocks.Client)
		account        v1alpha1.Account
		expectError    bool
		expectedResult ctrl.Result
	}{
		{
			name: "should respect ready condition and return sucessfully",
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					us := o.(*unstructured.Unstructured)

					cond := []metav1.Condition{
						{
							Type:   readyCondition,
							Status: metav1.ConditionTrue,
						},
					}

					out, err := json.Marshal(cond)
					assert.NoError(t, err)

					var conditionMap []interface{}
					err = json.Unmarshal(out, &conditionMap)
					assert.NoError(t, err)

					us.Object["status"] = map[string]any{
						"conditions": conditionMap,
					}

					return nil
				}).Once()
			},
			account: v1alpha1.Account{
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							ReadyConditionType: &readyCondition,
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &defaultNamespace,
				},
			},
		},
		{
			name: "should respect ready condition and requeue in case the extension is not found",
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(kerrors.NewNotFound(schema.GroupResource{}, "AccountExtension"))
			},
			account: v1alpha1.Account{
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							ReadyConditionType: &readyCondition,
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &defaultNamespace,
				},
			},
			expectError:    false,
			expectedResult: ctrl.Result{Requeue: true},
		},
		{
			name: "should respect ready condition and requeue in case the extension is not yet ready",
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					us := o.(*unstructured.Unstructured)

					cond := []metav1.Condition{
						{
							Type:   readyCondition,
							Status: metav1.ConditionFalse,
						},
					}

					out, err := json.Marshal(cond)
					assert.NoError(t, err)

					var conditionMap []interface{}
					err = json.Unmarshal(out, &conditionMap)
					assert.NoError(t, err)

					us.Object["status"] = map[string]any{
						"conditions": conditionMap,
					}

					return nil
				}).Once()
			},
			account: v1alpha1.Account{
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							ReadyConditionType: &readyCondition,
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &defaultNamespace,
				},
			},
			expectError:    false,
			expectedResult: ctrl.Result{Requeue: true},
		},
		{
			name: "should respect ready condition and fail in case the namespace cannot be retrived",
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(errors.New("some error"))
			},
			account: v1alpha1.Account{
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							ReadyConditionType: &readyCondition,
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &defaultNamespace,
				},
			},
			expectError: true,
		},
		{
			name: "should respect ready condition and fail in case the extension retrieval failed",
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("some error"))
			},
			account: v1alpha1.Account{
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							ReadyConditionType: &readyCondition,
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &defaultNamespace,
				},
			},
			expectError: true,
		},
		{
			name: "should respect ready condition and fail for wrong format",
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					us := o.(*unstructured.Unstructured)

					us.Object["status"] = map[string]any{
						"wrong-key": "",
					}

					return nil
				})
			},
			account: v1alpha1.Account{
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							ReadyConditionType: &readyCondition,
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &defaultNamespace,
				},
			},
			expectError: true,
		},
		{
			name: "should skip processing of subroutine for extension if no readyConditionType is procided",
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))
			},
			account: v1alpha1.Account{
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &defaultNamespace,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k8sClient := mocks.NewClient(t)
			if test.k8sMocks != nil {
				test.k8sMocks(k8sClient)
			}

			routine := subroutines.NewExtensionReadySubroutine(k8sClient)

			result, err := routine.Process(context.Background(), &test.account)
			if test.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			if (test.expectedResult != ctrl.Result{}) {
				assert.Equal(t, test.expectedResult, result)
			}
		})
	}
}
