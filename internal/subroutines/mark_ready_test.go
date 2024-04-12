package subroutines

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/subroutines/mocks"
)

type MarkReadySubroutineSuite struct {
	suite.Suite

	// Tested Object(s)
	testObj *MarkReadySubroutine

	// Mocks
	clientMock *mocks.Client
}

func (suite *MarkReadySubroutineSuite) SetupTest() {
	// Setup Mocks
	suite.clientMock = new(mocks.Client)

	// Initialize Tested Object(s)
	suite.testObj = NewMarkReadySubroutine()
}

func TestMarkReadySubroutineSuite(t *testing.T) {
	suite.Run(t, new(MarkReadySubroutineSuite))
}

// Test Process and verify that the Ready condition was set to the instance
func (s *MarkReadySubroutineSuite) TestProcess_OK() {
	// Given
	instance := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
	}
	s.clientMock.On("Update", mock.Anything, instance).Return(nil)

	// When
	result, err := s.testObj.Process(context.Background(), instance)

	// Then
	s.Nil(err)
	s.Equal(result, ctrl.Result{})

	s.Len(instance.Status.Conditions, 1)
	s.Equal(metav1.ConditionTrue, instance.Status.Conditions[0].Status)
	s.Equal(corev1alpha1.ConditionAccountReady, instance.Status.Conditions[0].Type)
	s.Equal(corev1alpha1.ConditionAccountReady, instance.Status.Conditions[0].Reason)
	s.Equal("The account is ready", instance.Status.Conditions[0].Message)
}
