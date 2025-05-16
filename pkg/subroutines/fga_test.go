package subroutines_test

import (
	"context"
	"testing"

	kcpcorev1alpha1 "github.com/kcp-dev/kcp/sdk/apis/core/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/kontext"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/openmfp/account-operator/api/v1alpha1"
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

func TestFGASubroutine_GetName(t *testing.T) {
	routine := subroutines.NewFGASubroutine(nil, nil, "", "", "")
	assert.Equal(t, "FGASubroutine", routine.GetName())
}

func TestFGASubroutine_Finalizers(t *testing.T) {
	routine := subroutines.NewFGASubroutine(nil, nil, "", "", "")
	assert.Equal(t, []string{"account.core.openmfp.org/fga"}, routine.Finalizers())
}

func TestFGASubroutine_Process(t *testing.T) {
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
				Spec: v1alpha1.AccountSpec{
					Type: v1alpha1.AccountTypeOrg,
				},
				Status: v1alpha1.AccountStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "FGASubroutine_Ready",
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				mockGetWorkspaceByName(clientMock, kcpcorev1alpha1.LogicalClusterPhaseReady, "root:openmfp:orgs:root-org").Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()
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
				mockGetWorkspaceByName(clientMock, kcpcorev1alpha1.LogicalClusterPhaseReady, "root:openmfp:orgs:root-org")
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {
					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
						},
					}

					return nil
				}).Once()
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
				mockGetWorkspaceByName(clientMock, kcpcorev1alpha1.LogicalClusterPhaseReady, "root:openmfp:orgs:root-org")
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
				Spec: v1alpha1.AccountSpec{
					Type: v1alpha1.AccountTypeAccount,
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				mockGetWorkspaceByName(clientMock, kcpcorev1alpha1.LogicalClusterPhaseReady, "root:openmfp:orgs:root-org").Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name:      "test-account",
								ClusterId: "test-account-id",
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()
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
				Spec: v1alpha1.AccountSpec{
					Type: v1alpha1.AccountTypeAccount,
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				mockGetWorkspaceByName(clientMock, kcpcorev1alpha1.LogicalClusterPhaseReady, "root:openmfp:orgs:root-org").Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name:      "test-account",
								ClusterId: "test-account-id",
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()
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
				Spec: v1alpha1.AccountSpec{
					Type: v1alpha1.AccountTypeAccount,
				}},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				mockGetWorkspaceByName(clientMock, kcpcorev1alpha1.LogicalClusterPhaseReady, "root:openmfp:orgs:root-org").Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name:      "test-account",
								ClusterId: "test-account-id",
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()
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
					Type:    v1alpha1.AccountTypeAccount,
					Creator: ptr.To("system:serviceaccount:some-namespace:some-service-account"),
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				mockGetWorkspaceByName(clientMock, kcpcorev1alpha1.LogicalClusterPhaseReady, "root:openmfp:orgs:root-org").Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name:      "test-account",
								ClusterId: "test-account-id",
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()
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
				mockGetWorkspaceByName(clientMock, kcpcorev1alpha1.LogicalClusterPhaseReady, "root:openmfp:orgs:root-org").Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
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
					Type:    v1alpha1.AccountTypeOrg,
					Creator: &creator,
				},
			},
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service, clientMock *mocks.Client) {
				mockGetWorkspaceByName(clientMock, kcpcorev1alpha1.LogicalClusterPhaseReady, "root:openmfp:orgs:root-org").Once()
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name:      "root-org",
								ClusterId: "root-org",
								Path:      "root:openmfp:org:root-org",
								URL:       "http://example.com/clusters/root:openmfp:org:root-org",
								Type:      v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name:      "test-account",
								ClusterId: "test-account-id",
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()

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

			routine := subroutines.NewFGASubroutine(clientMock, openFGAClient, "owner", "parent", "account")
			ctx := kontext.WithCluster(context.Background(), "abcdefghi")
			_, err := routine.Process(ctx, test.account)
			if test.expectedError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			clientMock.AssertExpectations(t)

		})
	}
}

func TestCreatorSubroutine_Finalize(t *testing.T) {
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
					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
						},
					}

					return nil
				}).Once()
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()
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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()

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
				clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn types.NamespacedName, o client.Object, opts ...client.GetOption) error {

					account := o.(*v1alpha1.AccountInfo)

					*account = v1alpha1.AccountInfo{
						ObjectMeta: metav1.ObjectMeta{
							Name: "root-org",
						},
						Spec: v1alpha1.AccountInfoSpec{
							Organization: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							ParentAccount: &v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							Account: v1alpha1.AccountLocation{
								Name: "root-org",
								Path: "root:openmfp:org:root-org",
								URL:  "http://example.com/clusters/root:openmfp:org:root-org",
								Type: v1alpha1.AccountTypeOrg,
							},
							FGA: v1alpha1.FGAInfo{Store: v1alpha1.StoreInfo{Id: "123123"}},
						},
					}

					return nil
				}).Once()

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

			routine := subroutines.NewFGASubroutine(k8sClient, openFGAClient, "owner", "parent", "account")
			ctx := kontext.WithCluster(context.Background(), "abcdefghi")
			_, err := routine.Finalize(ctx, test.account)
			if test.expectedError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

		})
	}
}
