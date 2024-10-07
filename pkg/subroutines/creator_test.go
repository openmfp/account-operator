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
	routine := subroutines.NewCreatorSubroutine(nil, nil, "")
	assert.Equal(t, "CreatorSubroutine", routine.GetName())
}

func TestCreatorSubroutine_Finalizers(t *testing.T) {
	routine := subroutines.NewCreatorSubroutine(nil, nil, "")
	assert.Equal(t, []string{}, routine.Finalizers())
}

func TestCreatorSubroutine_Process(t *testing.T) {
	namespace := "test-openmfp-namespace"
	creator := "test-creator"

	tests := []struct {
		name        string
		account     v1alpha1.Account
		in          *openfgav1.WriteRequest
		out         *openfgav1.WriteResponse
		ctx         context.Context
		setupMocks  func(context.Context, *openfgav1.WriteRequest, *openfgav1.WriteResponse, *mocks.OpenFGAServiceClient, *mocks.K8Service)
		expectError *struct {
			Retry  bool
			Sentry bool
		}
	}{
		{
			name: "should_write_owner",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "1",
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tenant1",
						Namespace: namespace,
					},
					Spec: v1alpha1.AccountSpec{
						Creator: &creator,
					},
					Status: v1alpha1.AccountStatus{
						Namespace: &namespace,
					},
				}, nil)
				openFGAServiceClientMock.EXPECT().ListStores(ctx, mock.Anything).Return(&openfgav1.ListStoresResponse{
					Stores: []*openfgav1.Store{{Id: "1", Name: "tenant-tenant1"}},
				}, nil)
				openFGAServiceClientMock.EXPECT().Write(ctx, in).
					Return(out, nil).
					Once()
			},
		},
		{
			name: "should_handle_duplicate_write",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "9",
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tenant9",
						Namespace: namespace,
					},
					Spec: v1alpha1.AccountSpec{
						Creator: &creator,
					},
					Status: v1alpha1.AccountStatus{
						Namespace: &namespace,
					},
				}, nil)
				openFGAServiceClientMock.EXPECT().ListStores(ctx, mock.Anything).Return(&openfgav1.ListStoresResponse{
					Stores: []*openfgav1.Store{{Id: "9", Name: "tenant-tenant9"}},
				}, nil)
				openFGAServiceClientMock.EXPECT().Write(ctx, in).
					Return(nil, newFgaError(openfgav1.ErrorCode_write_failed_due_to_invalid_input, "error")).
					Once()
			},
		},
		{
			name: "handle_write_error",
			expectError: &struct {
				Retry  bool
				Sentry bool
			}{Retry: true, Sentry: true},
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "2",
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tenant2",
						Namespace: namespace,
					},
					Spec: v1alpha1.AccountSpec{
						Creator: &creator,
					},
					Status: v1alpha1.AccountStatus{
						Namespace: &namespace,
					},
				}, nil)
				openFGAServiceClientMock.EXPECT().ListStores(ctx, mock.Anything).Return(&openfgav1.ListStoresResponse{
					Stores: []*openfgav1.Store{{Id: "2", Name: "tenant-tenant2"}},
				}, nil)
				openFGAServiceClientMock.EXPECT().Write(ctx, in).Return(out, assert.AnError).Once()
			},
		},
		{
			name: "should_not_write_owner_if_creator_is_nil",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tenant3",
					Namespace: namespace,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
			},
		},
		{
			name: "should_not_write_owner_if_condition_exists",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tenant",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
					Conditions: []metav1.Condition{
						{
							Type:    "CreatorSubroutine_Ready",
							Status:  metav1.ConditionTrue,
							Message: "CreatorSubroutine_Ready",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
			},
		},
		{
			expectError: &struct {
				Retry  bool
				Sentry bool
			}{Retry: true, Sentry: true},
			name: "should_handle_reading_account_err",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "7",
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tenant7",
						Namespace: namespace,
					},
					Spec: v1alpha1.AccountSpec{
						Creator: &creator,
					},
					Status: v1alpha1.AccountStatus{
						Namespace: &namespace,
					},
				}, nil)
				openFGAServiceClientMock.EXPECT().ListStores(ctx, mock.Anything).Return(nil, assert.AnError)
			},
		},
		{
			expectError: &struct {
				Retry  bool
				Sentry bool
			}{Retry: true, Sentry: true},
			name: "should_handle_reading_store_err",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "7",
				Writes: &openfgav1.WriteRequestWrites{
					TupleKeys: []*openfgav1.TupleKey{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(nil, assert.AnError)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// setup mocks
			openFGAServiceClientMock := mocks.NewOpenFGAServiceClient(t)
			k8ServiceMock := mocks.NewK8Service(t)
			if test.setupMocks != nil {
				out := test.out
				if out == nil {
					out = &openfgav1.WriteResponse{}
				}
				test.setupMocks(test.ctx, test.in, out, openFGAServiceClientMock, k8ServiceMock)
			}

			routine := subroutines.NewCreatorSubroutine(openFGAServiceClientMock, k8ServiceMock, namespace)
			_, err := routine.Process(test.ctx, &test.account)
			if test.expectError != nil {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectError.Retry, err.Retry())
				assert.Equal(t, test.expectError.Sentry, err.Sentry())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCreatorSubroutine_Finalize(t *testing.T) {
	namespace := "test-openmfp-namespace"
	creator := "test-creator"

	tests := []struct {
		name        string
		account     v1alpha1.Account
		in          *openfgav1.WriteRequest
		out         *openfgav1.WriteResponse
		ctx         context.Context
		setupMocks  func(context.Context, *openfgav1.WriteRequest, *openfgav1.WriteResponse, *mocks.OpenFGAServiceClient, *mocks.K8Service)
		expectError *struct {
			Retry  bool
			Sentry bool
		}
	}{
		{
			name: "should_delete_owner",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "4",
				Deletes: &openfgav1.WriteRequestDeletes{
					TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tenant4",
						Namespace: namespace,
					},
					Spec: v1alpha1.AccountSpec{
						Creator: &creator,
					},
					Status: v1alpha1.AccountStatus{
						Namespace: &namespace,
					},
				}, nil)
				openFGAServiceClientMock.EXPECT().ListStores(ctx, mock.Anything).Return(&openfgav1.ListStoresResponse{
					Stores: []*openfgav1.Store{{Id: "4", Name: "tenant-tenant4"}},
				}, nil).Once()
				openFGAServiceClientMock.EXPECT().Write(ctx, in).
					Return(out, nil).
					Once()
			},
		},
		{
			name: "should_not_delete_owner_if_creator_is_nil",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "1",
				Deletes: &openfgav1.WriteRequestDeletes{
					TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
			},
		},
		{
			name: "handle_write_error",
			expectError: &struct {
				Retry  bool
				Sentry bool
			}{Retry: true, Sentry: true},
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "5",
				Deletes: &openfgav1.WriteRequestDeletes{
					TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tenant5",
						Namespace: namespace,
					},
					Spec: v1alpha1.AccountSpec{
						Creator: &creator,
					},
					Status: v1alpha1.AccountStatus{
						Namespace: &namespace,
					},
				}, nil)
				openFGAServiceClientMock.EXPECT().ListStores(ctx, mock.Anything).Return(&openfgav1.ListStoresResponse{
					Stores: []*openfgav1.Store{{Id: "5", Name: "tenant-tenant5"}},
				}, nil).Once()
				openFGAServiceClientMock.EXPECT().Write(ctx, in).
					Return(nil, assert.AnError).
					Once()
			},
		},
		{
			name:        "handle_write_error_with_non_existing_entry",
			expectError: nil,
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "5",
				Deletes: &openfgav1.WriteRequestDeletes{
					TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tenant5",
						Namespace: namespace,
					},
					Spec: v1alpha1.AccountSpec{
						Creator: &creator,
					},
					Status: v1alpha1.AccountStatus{
						Namespace: &namespace,
					},
				}, nil)
				openFGAServiceClientMock.EXPECT().ListStores(ctx, mock.Anything).Return(&openfgav1.ListStoresResponse{
					Stores: []*openfgav1.Store{{Id: "5", Name: "tenant-tenant5"}},
				}, nil).Once()
				openFGAServiceClientMock.EXPECT().Write(ctx, in).
					Return(nil, newFgaError(openfgav1.ErrorCode_write_failed_due_to_invalid_input, "error")).
					Once()
			},
		},
		{
			expectError: &struct {
				Retry  bool
				Sentry bool
			}{Retry: true, Sentry: true},
			name: "should_handle_read_user_err",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "4",
				Deletes: &openfgav1.WriteRequestDeletes{
					TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(nil, assert.AnError)
			},
		},
		{
			expectError: &struct {
				Retry  bool
				Sentry bool
			}{Retry: true, Sentry: true},
			name: "should_handle_read_stores_err",
			account: v1alpha1.Account{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-account",
					Namespace: namespace,
				},
				Spec: v1alpha1.AccountSpec{
					Creator: &creator,
				},
				Status: v1alpha1.AccountStatus{
					Namespace: &namespace,
				},
			},
			in: &openfgav1.WriteRequest{
				StoreId: "4",
				Deletes: &openfgav1.WriteRequestDeletes{
					TupleKeys: []*openfgav1.TupleKeyWithoutCondition{
						{
							Object:   "account:test-account",
							Relation: "owner",
							User:     "user:test-creator",
						},
					},
				},
			},
			setupMocks: func(ctx context.Context, in *openfgav1.WriteRequest, out *openfgav1.WriteResponse, openFGAServiceClientMock *mocks.OpenFGAServiceClient, k8ServiceMock *mocks.K8Service) {
				k8ServiceMock.EXPECT().GetFirstLevelAccountForNamespace(ctx, namespace).Return(&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "tenant8",
						Namespace: namespace,
					},
					Spec: v1alpha1.AccountSpec{
						Creator: &creator,
					},
					Status: v1alpha1.AccountStatus{
						Namespace: &namespace,
					},
				}, nil)
				openFGAServiceClientMock.EXPECT().ListStores(ctx, mock.Anything).Return(nil, assert.AnError).Once()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			openFGAServiceClientMock := &mocks.OpenFGAServiceClient{}
			k8ServiceMock := mocks.NewK8Service(t)
			if test.setupMocks != nil {
				out := test.out
				if out == nil {
					out = &openfgav1.WriteResponse{}
				}
				test.setupMocks(test.ctx, test.in, out, openFGAServiceClientMock, k8ServiceMock)
			}

			routine := subroutines.NewCreatorSubroutine(openFGAServiceClientMock, k8ServiceMock, namespace)
			_, err := routine.Finalize(test.ctx, &test.account)
			if test.expectError != nil {
				assert.NotNil(t, err)
				assert.Equal(t, test.expectError.Retry, err.Retry())
				assert.Equal(t, test.expectError.Sentry, err.Sentry())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
