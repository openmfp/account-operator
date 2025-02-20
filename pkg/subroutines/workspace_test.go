package subroutines_test

import (
	"context"
	"testing"

	kcptenancyv1alpha "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmfp/golang-commons/errors"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/account-operator/pkg/subroutines/mocks"
)

const defaultExpectedTestNamespace = "account-test"

type WorkspaceSubroutineTestSuite struct {
	suite.Suite

	// Tested Object(s)
	testObj *subroutines.WorkspaceSubroutine

	// Mocks
	clientMock *mocks.Client
}

func (suite *WorkspaceSubroutineTestSuite) SetupTest() {
	// Setup Mocks
	suite.clientMock = new(mocks.Client)

	// Initialize Tested Object(s)
	suite.testObj = subroutines.NewWorkspaceSubroutine(suite.clientMock)
}

func (suite *WorkspaceSubroutineTestSuite) TestGetName_OK() {
	// When
	result := suite.testObj.GetName()

	// Then
	suite.Equal(subroutines.WorkspaceSubroutineName, result)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalize_OK() {
	// Given
	testAccount := &corev1alpha1.Account{}

	// When
	res, err := suite.testObj.Finalize(context.Background(), testAccount)

	// Then
	suite.False(res.Requeue)
	suite.Assert().Zero(res.RequeueAfter)
	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingWorkspace_NoFinalizer_OK() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockNewWorkspaceCreateCall(suite, defaultExpectedTestNamespace)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Workspace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Workspace)

	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingNamespace_NoFinalizer_CreateError() {
	// Given
	testAccount := &corev1alpha1.Account{}
	suite.clientMock.EXPECT().
		Create(mock.Anything, mock.Anything).
		Return(kerrors.NewBadRequest(""))

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Nil(testAccount.Status.Workspace)
	suite.NotNil(err)
	suite.True(err.Retry())
	suite.True(err.Sentry())
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithNamespaceInStatus() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Workspace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallWithLabels(suite, defaultExpectedTestNamespace, map[string]string{
		corev1alpha1.NamespaceAccountOwnerLabel:          testAccount.Name,
		corev1alpha1.NamespaceAccountOwnerNamespaceLabel: testAccount.Namespace,
	})

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Workspace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Workspace)

	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithNamespaceInStatus_LookupError() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Workspace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(kerrors.NewBadRequest(""))

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.NotNil(err)
	suite.True(err.Retry())
	suite.True(err.Sentry())
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusMissingLabels() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Workspace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallWithLabels(suite, defaultExpectedTestNamespace, map[string]string{
		corev1alpha1.NamespaceAccountOwnerLabel: testAccount.Name,
	})
	mockNewNamespaceUpdateCall(suite)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Workspace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Workspace)

	suite.Nil(err)
}

// Test like TestProcessingWithNamespaceInStatusMissingLabels but the update call fails unexpectedly
func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusMissingLabels_UpdateError() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Workspace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallWithLabels(suite, defaultExpectedTestNamespace, map[string]string{
		corev1alpha1.NamespaceAccountOwnerLabel: testAccount.Name,
	})
	suite.clientMock.EXPECT().
		Update(mock.Anything, mock.Anything).
		Return(kerrors.NewBadRequest(""))

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.NotNil(err)
	suite.True(err.Retry())
	suite.True(err.Sentry())
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusMissingLabels2() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Workspace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallWithLabels(suite, defaultExpectedTestNamespace, nil)
	mockNewNamespaceUpdateCall(suite)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Workspace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Workspace)

	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusAndNotFound() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Workspace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallNotFound(suite)
	mockNewWorkspaceCreateCall(suite, defaultExpectedTestNamespace)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Workspace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Workspace)

	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithDeclaredNamespace_OK() {
	// Given
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		Spec: corev1alpha1.AccountSpec{
			Workspace: &namespaceName,
		},
	}
	mockGetNamespaceCallWithLabels(suite, namespaceName, map[string]string{
		corev1alpha1.NamespaceAccountOwnerLabel:          testAccount.Name,
		corev1alpha1.NamespaceAccountOwnerNamespaceLabel: testAccount.Namespace,
	})

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Workspace)
	suite.Equal(namespaceName, *testAccount.Status.Workspace)

	suite.Nil(err)

}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithDeclaredNamespaceNotFound() {
	// Given
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		Spec: corev1alpha1.AccountSpec{
			Workspace: &namespaceName,
		},
	}
	mockGetNamespaceCallNotFound(suite)

	mockNewWorkspaceCreateCall(suite, namespaceName)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Equal(namespaceName, *testAccount.Status.Workspace)
	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithDeclaredNamespaceLookupError() {
	// Given
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		Spec: corev1alpha1.AccountSpec{
			Workspace: &namespaceName,
		},
	}
	suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(kerrors.NewBadRequest(""))

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Nil(testAccount.Status.Workspace)
	suite.NotNil(err)
	suite.True(err.Retry())
	suite.True(err.Sentry())
}

// Test finalize function and expect no error
func (suite *WorkspaceSubroutineTestSuite) TestFinalizeNamespace_OK() {
	// Given
	testAccount := &corev1alpha1.Account{}

	// When
	res, err := suite.testObj.Finalize(context.Background(), testAccount)

	// Then
	suite.False(res.Requeue)
	suite.Assert().Zero(res.RequeueAfter)
	suite.Nil(err)
}

// Test an account with a namspace in the spec, where the already existing namespace has different owner labels
func (suite *WorkspaceSubroutineTestSuite) TestProcessingWithDeclaredNamespaceMismatchedOwnerLabels() {
	// Given
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Spec: corev1alpha1.AccountSpec{
			Workspace: &namespaceName,
		},
	}
	mockGetNamespaceCallWithLabels(suite, namespaceName, map[string]string{
		corev1alpha1.NamespaceAccountOwnerLabel: "different-owner",
	})

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().Nil(testAccount.Status.Workspace)
	suite.NotNil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalizationWithNamespaceInStatus() {
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Status: corev1alpha1.AccountStatus{
			Workspace: &namespaceName,
		},
	}

	mockGetNamespaceCallWithName(suite, namespaceName)
	mockDeleteNamespaceCall(suite)

	result, err := suite.testObj.Finalize(context.Background(), testAccount)
	suite.Require().Nil(err)
	suite.Require().True(result.Requeue)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalizationWithNamespaceInStatus_Error() {
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Status: corev1alpha1.AccountStatus{
			Workspace: &namespaceName,
		},
	}

	mockGetNamespaceCallWithError(suite, errors.New("error"))

	_, err := suite.testObj.Finalize(context.Background(), testAccount)
	suite.Require().NotNil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalizationWithNamespaceInStatus_DeletionError() {
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Status: corev1alpha1.AccountStatus{
			Workspace: &namespaceName,
		},
	}

	mockGetNamespaceCallWithName(suite, namespaceName)
	mockDeleteNamespaceCallWithError(suite, errors.New("error"))

	_, err := suite.testObj.Finalize(context.Background(), testAccount)
	suite.Require().NotNil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalizationWithNamespaceInStatus_DeletionTimestampSet() {
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Status: corev1alpha1.AccountStatus{
			Workspace: &namespaceName,
		},
	}

	mockGetNamespaceCallWithNameAndDeletionTimestamp(suite, namespaceName)

	result, err := suite.testObj.Finalize(context.Background(), testAccount)
	suite.Require().Nil(err)
	suite.Require().True(result.Requeue)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalizationWithNamespaceInStatus_NamespaceGone() {
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Status: corev1alpha1.AccountStatus{
			Workspace: &namespaceName,
		},
	}

	mockGetNamespaceCallNotFound(suite)

	result, err := suite.testObj.Finalize(context.Background(), testAccount)
	suite.Require().Nil(err)
	suite.Require().False(result.Requeue)
}

func TestNamespaceSubroutineTestSuite(t *testing.T) {
	suite.Run(t, new(WorkspaceSubroutineTestSuite))
}

//nolint:golint,unparam
func mockNewWorkspaceCreateCall(suite *WorkspaceSubroutineTestSuite, name string) *mocks.Client_Create_Call {
	return suite.clientMock.EXPECT().
		Create(mock.Anything, mock.Anything).
		Run(func(ctx context.Context, obj client.Object, opts ...client.CreateOption) {
			actual, _ := obj.(*kcptenancyv1alpha.Workspace)
			actual.Name = name
		}).
		Return(nil)
}

//nolint:golint,unparam
func mockNewNamespaceUpdateCall(suite *WorkspaceSubroutineTestSuite) *mocks.Client_Update_Call {
	return suite.clientMock.EXPECT().
		Update(mock.Anything, mock.Anything).
		Return(nil)
}

//nolint:golint,unparam
func mockGetNamespaceCallWithLabels(suite *WorkspaceSubroutineTestSuite, name string, labels map[string]string) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			actual, _ := obj.(*v1.Namespace)
			actual.Name = name
			actual.Labels = labels
		}).
		Return(nil)
}

//nolint:golint,unparam
func mockGetNamespaceCallWithName(suite *WorkspaceSubroutineTestSuite, name string) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			actual, _ := obj.(*v1.Namespace)
			actual.Name = name
		}).
		Return(nil)
}

//nolint:golint,unparam
func mockGetNamespaceCallWithNameAndDeletionTimestamp(suite *WorkspaceSubroutineTestSuite, name string) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			actual, _ := obj.(*v1.Namespace)
			actual.Name = name
			actual.DeletionTimestamp = &metav1.Time{}
		}).
		Return(nil)
}

//nolint:golint,unparam
func mockDeleteNamespaceCall(suite *WorkspaceSubroutineTestSuite) *mocks.Client_Delete_Call {
	return suite.clientMock.EXPECT().
		Delete(mock.Anything, mock.Anything).
		Return(nil)
}

//nolint:golint,unparam
func mockDeleteNamespaceCallWithError(suite *WorkspaceSubroutineTestSuite, err error) *mocks.Client_Delete_Call {
	return suite.clientMock.EXPECT().
		Delete(mock.Anything, mock.Anything).
		Return(err)
}

func mockGetNamespaceCallNotFound(
	suite *WorkspaceSubroutineTestSuite) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(kerrors.NewNotFound(schema.GroupResource{}, ""))
}

func mockGetNamespaceCallWithError(suite *WorkspaceSubroutineTestSuite, err error) {
	suite.clientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).
		Return(err)
}
