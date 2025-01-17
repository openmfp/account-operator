package subroutines_test

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openmfp/account-operator/api/v1alpha1"
	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/account-operator/pkg/subroutines/mocks"
)

type fgaError struct {
	code codes.Code
	msg  string
}

func (e fgaError) Error() string { return e.msg }
func (e fgaError) GRPCStatus() *status.Status {
	return status.New(e.code, e.msg)
}

func newFgaError(c openfgav1.ErrorCode, m string) *fgaError {
	return &fgaError{
		code: codes.Code(c),
		msg:  m,
	}
}

func TestCreatorSubroutine_GetName(t *testing.T) {
	routine := subroutines.NewFGASubroutine(nil, nil, nil, "", "", "", "")
	assert.Equal(t, "CreatorSubroutine", routine.GetName())
}

func TestCreatorSubroutine_Finalizers(t *testing.T) {
	routine := subroutines.NewFGASubroutine(nil, nil, nil, "", "", "", "")
	assert.Equal(t, []string{"account.core.openmfp.io/fga"}, routine.Finalizers())
}

func getStoreMocks(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {

	clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
			ns := o.(*corev1.Namespace)

			*ns = corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
						corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
					},
				},
			}
			return nil
		}).Once()
	clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
			account := o.(*v1alpha1.Account)

			*account = v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "first-level",
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
	clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
			return nil
		}).Once()

	openFGAServiceClientMock.EXPECT().ListStores(context.Background(), mock.Anything).Return(&openfgav1.ListStoresResponse{Stores: []*openfgav1.Store{{Id: "1", Name: "tenant-first-level"}}}, nil).Maybe()
}

func TestCreatorSubroutine_Process(t *testing.T) {
	namespace := "test-openmfp-namespace"
	creator := "test-creator"

	testCases := []struct {
		name          string
		expectedError bool
		account       *v1alpha1.Account
		setupMocks    func(*mocks.OpenFGAServiceClient, *mocks.K8Service, *mocks.Client)
	}{
		{
			name: "should_skip_processing_if_subroutine_ran_before",
			account: &v1alpha1.Account{
				Status: v1alpha1.AccountStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "CreatorSubroutine_Ready",
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name:          "should_fail_if_get_store_id_fails",
			expectedError: true,
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)

					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				openFGAServiceClientMock.EXPECT().ListStores(mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
		},
		{
			name:          "should_fail_if_get_parent_account_fails",
			expectedError: true,
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)

					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
			},
		},
		{
			name:          "should_fail_if_write_fails",
			expectedError: true,
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.Anything).
					Return(nil, assert.AnError)

			},
		},
		{
			name: "should_ignore_error_if_duplicate_write_error",
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.Anything).
					Return(nil, newFgaError(openfgav1.ErrorCode_write_failed_due_to_invalid_input, "error"))
			},
		},
		{
			name: "should_succeed",
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.Anything).
					Return(&openfgav1.WriteResponse{}, nil)
			},
		},
		{
			name: "should_succeed_with_creator_for_sa",
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Creator: ptr.To("system:serviceaccount:some-namespace:some-service-account"),
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.Anything).
					Return(&openfgav1.WriteResponse{}, nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.MatchedBy(func(req *openfgav1.WriteRequest) bool {
						// Check for partial match
						return len(req.Writes.TupleKeys) == 1 && req.Writes.TupleKeys[0].User == "user:system.serviceaccount.some-namespace.some-service-account"
					})).
					Return(&openfgav1.WriteResponse{}, nil)
			},
		},
		{
			name:          "should_fail_with_creator_in_sa_range",
			expectedError: true,
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Creator: ptr.To("system.serviceaccount.some-namespace.some-service-account"),
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "should_succeed_with_creator",
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.Anything).
					Return(&openfgav1.WriteResponse{}, nil)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			openFGAClient := mocks.NewOpenFGAServiceClient(t)
			accountClient := mocks.NewK8Service(t)
			clientMock := mocks.NewClient(t)

			if test.setupMocks != nil {
				test.setupMocks(openFGAClient, accountClient, clientMock)
			}

			routine := subroutines.NewFGASubroutine(clientMock, openFGAClient, accountClient, namespace, "owner", "parent", "account")
			ctx := context.Background()
			_, err := routine.Process(ctx, test.account)
			if test.expectedError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

		})
	}
}

func TestCreatorSubroutine_Finalize(t *testing.T) {
	namespace := "test-openmfp-namespace"
	creator := "test-creator"

	testCases := []struct {
		name          string
		expectedError bool
		account       *v1alpha1.Account
		setupMocks    func(*mocks.OpenFGAServiceClient, *mocks.K8Service, *mocks.Client)
	}{
		{
			name:          "should_fail_if_get_store_id_fails",
			expectedError: true,
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)

					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				openFGAServiceClientMock.EXPECT().ListStores(mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
		},
		{
			name:          "should_fail_if_get_parent_account_fails",
			expectedError: true,
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)

					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
			},
		},
		{
			name:          "should_fail_if_write_fails",
			expectedError: true,
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {

				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				openFGAServiceClientMock.EXPECT().Write(mock.Anything, mock.Anything).Return(nil, assert.AnError)

			},
		},
		{
			name: "should_ignore_error_if_duplicate_write_error",
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.Anything).
					Return(nil, newFgaError(openfgav1.ErrorCode_write_failed_due_to_invalid_input, "error"))
			},
		},
		{
			name: "should_succeed",
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.Anything).
					Return(&openfgav1.WriteResponse{}, nil)
			},
		},
		{
			name: "should_succeed_with_creator",
			account: &v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: "test-namespace",
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock, clientMock)

				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					ns := o.(*corev1.Namespace)
					*ns = corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								corev1alpha1.NamespaceAccountOwnerLabel:          "first-level",
								corev1alpha1.NamespaceAccountOwnerNamespaceLabel: "first-level",
							},
						},
					}
					return nil
				}).Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.Anything).
					Return(&openfgav1.WriteResponse{}, nil).Times(3)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			openFGAClient := mocks.NewOpenFGAServiceClient(t)
			accountClient := mocks.NewK8Service(t)
			k8sClient := mocks.NewClient(t)

			if test.setupMocks != nil {
				test.setupMocks(openFGAClient, accountClient, k8sClient)
			}

			routine := subroutines.NewFGASubroutine(k8sClient, openFGAClient, accountClient, namespace, "owner", "parent", "account")
			ctx := context.Background()
			_, err := routine.Finalize(ctx, test.account)
			if test.expectedError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

		})
	}
}
