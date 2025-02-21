package subroutines_test

import (
	"context"
	"fmt"
	"testing"

	kcptenancyv1alpha "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	utilruntime.Must(corev1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1.AddToScheme(scheme.Scheme))
	suite.clientMock.On("Scheme").Return(scheme.Scheme)
}

func (suite *WorkspaceSubroutineTestSuite) TestGetName_OK() {
	// When
	result := suite.testObj.GetName()

	// Then
	suite.Equal(subroutines.WorkspaceSubroutineName, result)
}

func (suite *WorkspaceSubroutineTestSuite) TestGetFinalizerName() {
	// When
	finalizers := suite.testObj.Finalizers()

	// Then
	suite.Contains(finalizers, subroutines.WorkspaceSubroutineFinalizer)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalize_OK_Workspace_NotExisting() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockGetWorkspaceCallNotFound(suite)

	// When
	res, err := suite.testObj.Finalize(context.Background(), testAccount)

	// Then
	suite.False(res.Requeue)
	suite.Assert().Zero(res.RequeueAfter)
	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalize_OK_Workspace_ExistingButInDeletion() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockGetWorkspaceByNameInDeletion(suite)

	// When
	res, err := suite.testObj.Finalize(context.Background(), testAccount)

	// Then
	suite.True(res.Requeue)
	suite.Assert().Zero(res.RequeueAfter)
	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalize_OK_Workspace_Existing() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockGetWorkspaceByName(suite)
	mockDeleteWorkspaceCall(suite)

	// When
	res, err := suite.testObj.Finalize(context.Background(), testAccount)

	// Then
	suite.True(res.Requeue)
	suite.Assert().Zero(res.RequeueAfter)
	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalize_Error_On_Deletion() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockGetWorkspaceByName(suite)
	mockDeleteWorkspaceCallFailed(suite)

	// When
	_, err := suite.testObj.Finalize(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(err)
	suite.Error(err.Err())

	suite.True(err.Sentry())
	suite.True(err.Retry())
}

func (suite *WorkspaceSubroutineTestSuite) TestFinalize_Error_On_Get() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockGetWorkspaceFailed(suite)

	// When
	_, err := suite.testObj.Finalize(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(err)
	suite.Error(err.Err())

	suite.True(err.Sentry())
	suite.True(err.Retry())
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessing_OK() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockGetWorkspaceCallNotFound(suite)
	mockNewWorkspaceCreateCall(suite, defaultExpectedTestNamespace)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Nil(err)
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessing_Error_On_Get() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockGetWorkspaceFailed(suite)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(err)
	suite.Error(err.Err())
	suite.True(err.Sentry())
	suite.True(err.Retry())
}

func (suite *WorkspaceSubroutineTestSuite) TestProcessing_CreateError() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockGetWorkspaceCallNotFound(suite)
	suite.clientMock.EXPECT().
		Create(mock.Anything, mock.Anything).
		Return(kerrors.NewBadRequest(""))

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.NotNil(err)
	suite.True(err.Retry())
	suite.True(err.Sentry())
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
func mockGetWorkspaceCallNotFound(suite *WorkspaceSubroutineTestSuite) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(kerrors.NewNotFound(schema.GroupResource{}, ""))
}

func mockGetWorkspaceByName(suite *WorkspaceSubroutineTestSuite) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, types.NamespacedName{}, mock.Anything).
		Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			actual, _ := obj.(*kcptenancyv1alpha.Workspace)
			actual.Name = key.Name
		}).
		Return(nil)
}

func mockGetWorkspaceFailed(suite *WorkspaceSubroutineTestSuite) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, types.NamespacedName{}, mock.Anything).
		Return(kerrors.NewInternalError(fmt.Errorf("failed")))
}

func mockGetWorkspaceByNameInDeletion(suite *WorkspaceSubroutineTestSuite) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, types.NamespacedName{}, mock.Anything).
		Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			actual, _ := obj.(*kcptenancyv1alpha.Workspace)
			actual.Name = key.Name
			actual.DeletionTimestamp = &metav1.Time{}
		}).
		Return(nil)
}

//nolint:golint,unparam
func mockDeleteWorkspaceCall(suite *WorkspaceSubroutineTestSuite) *mocks.Client_Delete_Call {
	return suite.clientMock.EXPECT().
		Delete(mock.Anything, mock.Anything).
		Return(nil)
}

func mockDeleteWorkspaceCallFailed(suite *WorkspaceSubroutineTestSuite) *mocks.Client_Delete_Call {
	return suite.clientMock.EXPECT().
		Delete(mock.Anything, mock.Anything).
		Return(kerrors.NewInternalError(fmt.Errorf("failed")))
}
