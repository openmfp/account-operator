package subroutines_test

import (
	"context"
	"errors"
	"testing"

	"github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/account-operator/pkg/subroutines/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetName(t *testing.T) {
	routine := subroutines.NewExtensionSubroutine(nil)
	assert.Equal(t, "ExtensionSubroutine", routine.GetName())
}

func TestFinalizers(t *testing.T) {
	routine := subroutines.NewExtensionSubroutine(nil)
	assert.Equal(t, []string{}, routine.Finalizers())
}

func TestExtensionSubroutine_Process(t *testing.T) {
	namespace := "namespace"

	tests := []struct {
		name        string
		account     v1alpha1.Account
		k8sMocks    func(*mocks.Client)
		expectError bool
	}{
		{
			name: "should work without parent accounts",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "AccountExtension"))

				c.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "should work without parent accounts and extension spec",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{
								Raw: []byte(`{"foo":"bar"}`),
							},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "AccountExtension"))

				c.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "should fail without parent accounts due to random error",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(errors.New(""))
			},
			expectError: true,
		},
		{
			name: "should work with 1 level parent accounts",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)

					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								subroutines.NamespaceAccountOwnerLabel:          "first-level",
								subroutines.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					account := o.(*v1alpha1.Account)

					*account = v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fist-level",
							Namespace: "first-level",
						},
						Spec: v1alpha1.AccountSpec{
							Extensions: []v1alpha1.Extension{
								{
									TypeMeta: metav1.TypeMeta{
										Kind:       "AccountExtension",
										APIVersion: "core.openmfp.io/v1alpha1",
									},
									SpecGoTemplate: apiextensionsv1.JSON{},
								},
							},
						},
					}

					return nil
				}).Once()

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "AccountExtension"))
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "AccountExtension"))

				c.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
				c.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "should work with 1 level parent accounts but missing namespace owner label",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)

					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								subroutines.NamespaceAccountOwnerLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "AccountExtension"))

				c.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "should work with 1 level parent accounts but missing namespace namespace label",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "AccountExtension"))

				c.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "should work with 1 level parent accounts but missing namespace namespace label with random error",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(errors.New(""))
			},
			expectError: true,
		},
		{
			name: "should work with 1 level parent accounts and account not found",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)

					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								subroutines.NamespaceAccountOwnerLabel:          "first-level",
								subroutines.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Account"))

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "AccountExtension"))

				c.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
				c.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k8sClient := mocks.NewClient(t)
			if test.k8sMocks != nil {
				test.k8sMocks(k8sClient)
			}

			routine := subroutines.NewExtensionSubroutine(k8sClient)
			_, err := routine.Process(context.Background(), &test.account)
			if test.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestExtensionSubroutine_Finalize(t *testing.T) {
	namespace := "namespace"

	tests := []struct {
		name        string
		account     v1alpha1.Account
		k8sMocks    func(*mocks.Client)
		expectError bool
	}{
		{
			name: "should work without parent accounts",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))

				c.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "should fail without parent accounts due to random deletion error",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))

				c.EXPECT().Delete(mock.Anything, mock.Anything).Return(errors.New(""))
			},
			expectError: true,
		},
		{
			name: "should work without parent accounts and already deleted extension",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))

				c.EXPECT().Delete(mock.Anything, mock.Anything).Return(kerrors.NewNotFound(schema.GroupResource{}, "AccountExtension"))
			},
		},
		{
			name: "should work with 1 level parent accounts",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-account-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Extensions: []v1alpha1.Extension{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "AccountExtension",
								APIVersion: "core.openmfp.io/v1alpha1",
							},
							SpecGoTemplate: apiextensionsv1.JSON{},
						},
					},
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			k8sMocks: func(c *mocks.Client) {
				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)

					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								subroutines.NamespaceAccountOwnerLabel:          "first-level",
								subroutines.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					account := o.(*v1alpha1.Account)

					*account = v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fist-level",
							Namespace: "first-level",
						},
						Spec: v1alpha1.AccountSpec{
							Extensions: []v1alpha1.Extension{
								{
									TypeMeta: metav1.TypeMeta{
										Kind:       "AccountExtension",
										APIVersion: "core.openmfp.io/v1alpha1",
									},
									SpecGoTemplate: apiextensionsv1.JSON{},
								},
							},
						},
					}

					return nil
				}).Once()

				c.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Once().Return(kerrors.NewNotFound(schema.GroupResource{}, "Namespace"))

				c.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
				c.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k8sClient := mocks.NewClient(t)
			if test.k8sMocks != nil {
				test.k8sMocks(k8sClient)
			}

			routine := subroutines.NewExtensionSubroutine(k8sClient)
			_, err := routine.Finalize(context.Background(), &test.account)
			if test.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestRenderExtensionSpec(t *testing.T) {
	creator := "user"
	us := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	err := subroutines.RenderExtensionSpec(context.Background(), map[string]any{
		"foo":    "bar",
		"number": int64(1),
		"bool":   true,
		"nested": map[string]any{
			"value": "{{.Account.Spec.Creator}}",
		},
	}, &v1alpha1.Account{
		Spec: v1alpha1.AccountSpec{
			Creator: &creator,
		},
	}, &us, []string{"spec"})
	assert.NoError(t, err)

	us = unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	err = subroutines.RenderExtensionSpec(context.Background(), map[string]any{
		"foo":    "bar",
		"number": int64(1),
		"bool":   true,
		"nested": map[string]any{
			"value": "{{ .Account.Spec.Creator | upper }}",
		},
	}, &v1alpha1.Account{
		Spec: v1alpha1.AccountSpec{
			Creator: &creator,
		},
	}, &us, []string{"spec"})
	assert.NoError(t, err)
}

func TestRenderExtensionSpecInvalidTemplate(t *testing.T) {
	creator := ""
	us := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	err := subroutines.RenderExtensionSpec(context.Background(), map[string]any{
		"foo": "{{ .Account }",
	}, &v1alpha1.Account{
		Spec: v1alpha1.AccountSpec{
			Creator: &creator,
		},
	}, &us, []string{"spec"})
	assert.Error(t, err)
}
