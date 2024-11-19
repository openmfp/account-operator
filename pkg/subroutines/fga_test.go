package subroutines_test

import (
	"context"
	"testing"

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

func TestCreatorSubroutine_GetName(t *testing.T) {
	routine := subroutines.NewFGASubroutine(nil, nil, "", "", "", "")
	assert.Equal(t, "CreatorSubroutine", routine.GetName())
}

func TestCreatorSubroutine_Finalizers(t *testing.T) {
	routine := subroutines.NewFGASubroutine(nil, nil, "", "", "", "")
	assert.Equal(t, []string{}, routine.Finalizers())
}

func getStoreMocks(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
	k8ServiceMock.EXPECT().
		GetFirstLevelAccountForNamespace(context.Background(), mock.Anything).
		Return(&v1alpha1.Account{
			ObjectMeta: metav1.ObjectMeta{
				Name: "tenant1",
			},
		}, nil)

	openFGAServiceClientMock.EXPECT().
		ListStores(context.Background(), mock.Anything).
		Return(&openfgav1.ListStoresResponse{
			Stores: []*openfgav1.Store{{Id: "1", Name: "tenant-tenant1"}},
		}, nil).Maybe()
}

func TestCreatorSubroutine_Process(t *testing.T) {
	namespace := "test-openmfp-namespace"
	creator := "test-creator"

	testCases := []struct {
		name          string
		expectedError bool
		account       *v1alpha1.Account
		setupMocks    func(*mocks.OpenFGAServiceClient, *mocks.K8Service)
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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(mock.Anything, mock.Anything).Return(nil, assert.AnError)
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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)
				k8ServiceMock.EXPECT().GetAccountForNamespace(mock.Anything, mock.Anything).Return(nil, assert.AnError)
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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)

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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)

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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)

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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)

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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)
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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)

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

			if test.setupMocks != nil {
				test.setupMocks(openFGAClient, accountClient)
			}

			routine := subroutines.NewFGASubroutine(openFGAClient, accountClient, namespace, "owner", "parent", "account")

			_, err := routine.Process(context.Background(), test.account)
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
		setupMocks    func(*mocks.OpenFGAServiceClient, *mocks.K8Service)
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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(mock.Anything, mock.Anything).Return(nil, assert.AnError)
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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)
				k8ServiceMock.EXPECT().GetAccountForNamespace(mock.Anything, mock.Anything).Return(nil, assert.AnError)
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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)

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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)

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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)

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
			setupMocks: func(openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				getStoreMocks(openFGAServiceClientMock, k8ServiceMock)

				k8ServiceMock.EXPECT().
					GetAccountForNamespace(mock.Anything, mock.Anything).
					Return(&v1alpha1.Account{
						ObjectMeta: metav1.ObjectMeta{
							Name: "parent-account",
						},
					}, nil)

				openFGAServiceClientMock.EXPECT().
					Write(mock.Anything, mock.Anything).
					Return(&openfgav1.WriteResponse{}, nil).Twice()
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			openFGAClient := mocks.NewOpenFGAServiceClient(t)
			accountClient := mocks.NewK8Service(t)

			if test.setupMocks != nil {
				test.setupMocks(openFGAClient, accountClient)
			}

			routine := subroutines.NewFGASubroutine(openFGAClient, accountClient, namespace, "owner", "parent", "account")

			_, err := routine.Finalize(context.Background(), test.account)
			if test.expectedError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

		})
	}
}
