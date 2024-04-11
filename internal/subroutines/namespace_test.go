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

func (suite *NamespaceSubroutineTestSuite) TestProcessingNamespace_NoFinalizer_OK() {
	// Given
	testAccount := &corev1alpha1.Account{}

	expectNewNamespaceCreateCall(suite, defaultExpectedTestNamespace)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Namespace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Namespace)

	suite.Nil(err)
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithNamespaceInStatus() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Namespace: ptr.To(defaultExpectedTestNamespace),
		},
	}

	expectGetNamespaceCallReturningNamespaceWithLabels(suite, defaultExpectedTestNamespace, map[string]string{
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

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusMissingLabels() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Namespace: ptr.To(defaultExpectedTestNamespace),
		},
	}

	expectGetNamespaceCallReturningNamespaceWithLabels(suite, defaultExpectedTestNamespace, map[string]string{
		NamespaceAccountOwnerLabel: testAccount.Name,
	})
	expectNewNamespaceUpdateCall(suite)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Namespace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Namespace)

	suite.Nil(err)
}

func (suite *NamespaceSubroutineTestSuite) TestProcessingWithNamespaceInStatusMissingLabels2() {
	// Given
	testAccount := &corev1alpha1.Account{
		Status: corev1alpha1.AccountStatus{
			Namespace: ptr.To(defaultExpectedTestNamespace),
		},
	}

	expectGetNamespaceCallReturningNamespaceWithLabels(suite, defaultExpectedTestNamespace, nil)
	expectNewNamespaceUpdateCall(suite)

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

	expectGetNamespaceCallNotFound(suite)
	expectNewNamespaceCreateCall(suite, defaultExpectedTestNamespace)

	// When
	_, err := suite.testObj.Process(context.Background(), testAccount)

	// Then
	suite.Require().NotNil(testAccount.Status.Namespace)
	suite.Equal(defaultExpectedTestNamespace, *testAccount.Status.Namespace)

	suite.Nil(err)
}

func TestNamespaceSubroutineTestSuite(t *testing.T) {
	suite.Run(t, new(NamespaceSubroutineTestSuite))
}

func expectNewNamespaceCreateCall(suite *NamespaceSubroutineTestSuite, generatedName string) *mocks.Client_Create_Call {
	return suite.clientMock.EXPECT().
		Create(mock.Anything, mock.Anything).
		Run(func(ctx context.Context, obj client.Object, opts ...client.CreateOption) {
			actual, _ := obj.(*v1.Namespace)
			actual.Name = generatedName
		}).
		Return(nil)
}

func expectNewNamespaceUpdateCall(suite *NamespaceSubroutineTestSuite) *mocks.Client_Update_Call {
	return suite.clientMock.EXPECT().
		Update(mock.Anything, mock.Anything).
		Return(nil)
}

func expectGetNamespaceCallReturningNamespaceWithLabels(
	suite *NamespaceSubroutineTestSuite, name string, labels map[string]string) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Run(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) {
			actual, _ := obj.(*v1.Namespace)
			actual.Name = name
			actual.Labels = labels
		}).
		Return(nil)
}

func expectGetNamespaceCallNotFound(
	suite *NamespaceSubroutineTestSuite) *mocks.Client_Get_Call {
	return suite.clientMock.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(errors.NewNotFound(schema.GroupResource{}, ""))
}
