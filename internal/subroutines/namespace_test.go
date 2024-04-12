package subroutines

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/subroutines/mocks"
)

const defaultExpectedTestNamespace = "account-test"

type NamespaceSubroutineTestSuite struct {
	suite.Suite

	// Tested Object(s)
	testObj *NamespaceSubroutine

	// Mocks
	clientMock *mocks.Client
}

func (suite *NamespaceSubroutineTestSuite) SetupTest() {
	// Setup Mocks
	suite.clientMock = new(mocks.Client)

	// Initialize Tested Object(s)
	suite.testObj = NewNamespaceSubroutine(suite.clientMock)
}

func (suite *NamespaceSubroutineTestSuite) TestGetName_OK() {
	// When
	result := suite.testObj.GetName()

	// Then
	suite.Equal(NamespaceSubroutineName, result)
}

func (suite *NamespaceSubroutineTestSuite) TestFinalize_OK() {
	// Given
	testAccount := &corev1alpha1.Account{}

	// When
	res, err := suite.testObj.Finalize(context.Background(), testAccount)

	// Then
	suite.False(res.Requeue)
	suite.Assert().Zero(res.RequeueAfter)
	suite.Nil(err)
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingNamespace_NoFinalizer_OK() {
	// Given
	testAccount := &corev1alpha1.Account{}
	mockNewNamespaceCreateCall(suite, defaultExpectedTestNamespace)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Namespace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Namespace)

	suite.Nil(err)
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingNamespace_NoFinalizer_CreateError() {
	// Given
	testAccount := &corev1alpha1.Account{}
	suite.clientMock.EXPECT().
		Create(mock.Anything, mock.Anything).
		Return(errors.NewBadRequest(""))

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Nil(testAccount.Status.Namespace)
	suite.NotNil(err)
	suite.True(err.Retry())
	suite.True(err.Sentry())
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithNamespaceInStatus() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Namespace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallWithLabels(suite, defaultExpectedTestNamespace, map[string]string{
		NamespaceAccountOwnerLabel:          testAccount.Name,
		NamespaceAccountOwnerNamespaceLabel: testAccount.Namespace,
	})

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Namespace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Namespace)

	suite.Nil(err)
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithNamespaceInStatus_LookupError() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Namespace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(errors.NewBadRequest(""))

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.NotNil(err)
	suite.True(err.Retry())
	suite.True(err.Sentry())
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusMissingLabels() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Namespace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallWithLabels(suite, defaultExpectedTestNamespace, map[string]string{
		NamespaceAccountOwnerLabel: testAccount.Name,
	})
	mockNewNamespaceUpdateCall(suite)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Namespace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Namespace)

	suite.Nil(err)
}

// Test like TestProcessingWithNamespaceInStatusMissingLabels but the update call fails unexpectedly
func (suite *NamespaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusMissingLabels_UpdateError() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Namespace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallWithLabels(suite, defaultExpectedTestNamespace, map[string]string{
		NamespaceAccountOwnerLabel: testAccount.Name,
	})
	suite.clientMock.EXPECT().
		Update(mock.Anything, mock.Anything).
		Return(errors.NewBadRequest(""))

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.NotNil(err)
	suite.True(err.Retry())
	suite.True(err.Sentry())
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusMissingLabels2() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Namespace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallWithLabels(suite, defaultExpectedTestNamespace, nil)
	mockNewNamespaceUpdateCall(suite)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Namespace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Namespace)

	suite.Nil(err)
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusAndNotFound() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Namespace: ptr.To(defaultExpectedTestNamespace),
		},
	}
	mockGetNamespaceCallNotFound(suite)
	mockNewNamespaceCreateCall(suite, defaultExpectedTestNamespace)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Namespace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Namespace)

	suite.Nil(err)
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithDeclaredNamespace_OK() {
	// Given
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		Spec: corev1alpha1.AccountSpec{
			Namespace: &namespaceName,
		},
	}
	mockGetNamespaceCallWithLabels(suite, namespaceName, map[string]string{
		NamespaceAccountOwnerLabel:          testAccount.Name,
		NamespaceAccountOwnerNamespaceLabel: testAccount.Namespace,
	})

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Namespace)
	suite.Equal(namespaceName, *testAccount.Status.Namespace)

	suite.Nil(err)

}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithDeclaredNamespaceNotFound() {
	// Given
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		Spec: corev1alpha1.AccountSpec{
			Namespace: &namespaceName,
		},
	}
	mockGetNamespaceCallNotFound(suite)

	mockNewNamespaceCreateCall(suite, namespaceName)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Equal(namespaceName, *testAccount.Status.Namespace)
	suite.Nil(err)
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithDeclaredNamespaceLookupError() {
	// Given
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		Spec: corev1alpha1.AccountSpec{
			Namespace: &namespaceName,
		},
	}
	suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(errors.NewBadRequest(""))

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Nil(testAccount.Status.Namespace)
	suite.NotNil(err)
	suite.True(err.Retry())
	suite.True(err.Sentry())
}

// Test finalize function and expect no error
func (suite *NamespaceSubroutineTestSuite) TestFinalizeNamespace_OK() {
	// Given
	testAccount := &corev1alpha1.Account{}

	// When
	res, err := suite.testObj.Finalize(context.Background(), testAccount)

	// Then
	suite.False(res.Requeue)
	suite.Assert().Zero(res.RequeueAfter)
	suite.Nil(err)
}

func TestNamespaceSubroutineTestSuite(t *testing.T) {
	suite.Run(t, new(NamespaceSubroutineTestSuite))
}

//nolint:golint,unparam
func mockNewNamespaceCreateCall(suite *NamespaceSubroutineTestSuite, generatedName string) *mocks.Client_Create_Call {
	return suite.clientMock.EXPECT().
		Create(mock.Anything, mock.Anything).
		Run(func(ctx context.Context, obj client.Object, opts ...client.CreateOption) {
			actual, _ := obj.(*v1.Namespace)
			actual.Name = generatedName
		}).
		Return(nil)
}

//nolint:golint,unparam
func mockNewNamespaceUpdateCall(suite *NamespaceSubroutineTestSuite) *mocks.Client_Update_Call {
	return suite.clientMock.EXPECT().
		Update(mock.Anything, mock.Anything).
		Return(nil)
}

//nolint:golint,unparam
func mockGetNamespaceCallWithLabels(suite *NamespaceSubroutineTestSuite, name string, labels map[string]string) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			actual, _ := obj.(*v1.Namespace)
			actual.Name = name
			actual.Labels = labels
		}).
		Return(nil)
}

func mockGetNamespaceCallNotFound(
	suite *NamespaceSubroutineTestSuite) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(errors.NewNotFound(schema.GroupResource{}, ""))
}
