package subroutines_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/account-operator/pkg/subroutines/mocks"
)

const defaultExpectedTestNamespace = "account-test"

type NamespaceSubroutineTestSuite struct {
	suite.Suite

	// Tested Object(s)
	testObj *subroutines.NamespaceSubroutine

	// Mocks
	clientMock *mocks.Client
}

func (suite *NamespaceSubroutineTestSuite) SetupTest() {
	// Setup Mocks
	suite.clientMock = new(mocks.Client)

	// Initialize Tested Object(s)
	suite.testObj = subroutines.NewNamespaceSubroutine(suite.clientMock)
}

func (suite *NamespaceSubroutineTestSuite) TestGetName_OK() {
	// When
	result := suite.testObj.GetName()

	// Then
	suite.Equal(subroutines.NamespaceSubroutineName, result)
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
		subroutines.NamespaceAccountOwnerLabel:          testAccount.Name,
		subroutines.NamespaceAccountOwnerNamespaceLabel: testAccount.Namespace,
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
		subroutines.NamespaceAccountOwnerLabel: testAccount.Name,
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
		subroutines.NamespaceAccountOwnerLabel: testAccount.Name,
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
		subroutines.NamespaceAccountOwnerLabel:          testAccount.Name,
		subroutines.NamespaceAccountOwnerNamespaceLabel: testAccount.Namespace,
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

// Test an account with a namspace in the spec, where the already existing namespace has different owner labels
func (suite *NamespaceSubroutineTestSuite) TestProcessingWithDeclaredNamespaceMismatchedOwnerLabels() {
	// Given
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Spec: corev1alpha1.AccountSpec{
			Namespace: &namespaceName,
		},
	}
	mockGetNamespaceCallWithLabels(suite, namespaceName, map[string]string{
		subroutines.NamespaceAccountOwnerLabel: "different-owner",
	})

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().Nil(testAccount.Status.Namespace)
	suite.NotNil(err)
}

func (suite *NamespaceSubroutineTestSuite) TestFinalizationWithNamespaceInStatus() {
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Status: corev1alpha1.AccountStatus{
			Namespace: &namespaceName,
		},
	}

	mockGetNamespaceCallWithName(suite, namespaceName)
	mockDeleteNamespaceCall(suite)

	result, err := suite.testObj.Finalize(context.Background(), testAccount)
	suite.Require().Nil(err)
	suite.Require().True(result.Requeue)
}

func (suite *NamespaceSubroutineTestSuite) TestFinalizationWithNamespaceInStatus_DeletionTimestampSet() {
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Status: corev1alpha1.AccountStatus{
			Namespace: &namespaceName,
		},
	}

	mockGetNamespaceCallWithNameAndDeletionTimestamp(suite, namespaceName)

	result, err := suite.testObj.Finalize(context.Background(), testAccount)
	suite.Require().Nil(err)
	suite.Require().True(result.Requeue)
}

func (suite *NamespaceSubroutineTestSuite) TestFinalizationWithNamespaceInStatus_NamespaceGone() {
	namespaceName := "a-names-space"
	testAccount := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "test-account"},
		Status: corev1alpha1.AccountStatus{
			Namespace: &namespaceName,
		},
	}

	mockGetNamespaceCallNotFound(suite)

	result, err := suite.testObj.Finalize(context.Background(), testAccount)
	suite.Require().Nil(err)
	suite.Require().False(result.Requeue)
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

//nolint:golint,unparam
func mockGetNamespaceCallWithName(suite *NamespaceSubroutineTestSuite, name string) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			actual, _ := obj.(*v1.Namespace)
			actual.Name = name
		}).
		Return(nil)
}

//nolint:golint,unparam
func mockGetNamespaceCallWithNameAndDeletionTimestamp(suite *NamespaceSubroutineTestSuite, name string) *mocks.Client_Get_Call {
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
func mockDeleteNamespaceCall(suite *NamespaceSubroutineTestSuite) *mocks.Client_Delete_Call {
	return suite.clientMock.EXPECT().
		Delete(mock.Anything, mock.Anything).
		Return(nil)
}

func mockGetNamespaceCallNotFound(
	suite *NamespaceSubroutineTestSuite) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(errors.NewNotFound(schema.GroupResource{}, ""))
}
