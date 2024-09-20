package service_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	"github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/pkg/service"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	pointer "k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

type serviceTest struct {
	suite.Suite
	testEnv    envtest.Environment
	testClient client.Client
}

func TestService(t *testing.T) {
	suite.Run(t, new(serviceTest))
}

func (s *serviceTest) SetupSuite() {

	s.testEnv = envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "chart", "charts", "crds", "crds")},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := s.testEnv.Start()

	s.Require().NoError(err)

	s.Require().NoError(v1alpha1.AddToScheme(scheme.Scheme))

	s.testClient, err = client.New(cfg, client.Options{
		Scheme: scheme.Scheme,
	})
	s.Require().NoError(err)

	err = s.testClient.Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "root-namespace"}})
	s.Require().NoError(err)
}

func (s *serviceTest) TearDownSuite() {
	s.Require().NoError(s.testEnv.Stop())
}

func (s *serviceTest) TestGetAccount() {
	tests := []struct {
		name        string
		mockObjects []client.Object
		objectKey   client.ObjectKey
	}{
		{
			name: "",
			mockObjects: []client.Object{
				&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-account",
						Namespace: "root-namespace",
					},
					Spec: v1alpha1.AccountSpec{
						Type: v1alpha1.AccountTypeAccount,
					},
				},
			},
			objectKey: types.NamespacedName{Namespace: "root-namespace", Name: "test-account"},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			svc := service.NewService(s.testClient, "root-namespace")

			for _, obj := range test.mockObjects {
				err := s.testClient.Create(context.Background(), obj)
				s.Require().NoError(err)
			}

			account, err := svc.GetAccount(context.Background(), test.objectKey)
			s.Require().NoError(err)

			s.Require().Equal(test.objectKey.Name, account.Name)
			s.Require().Equal(test.objectKey.Namespace, account.Namespace)

			for _, obj := range test.mockObjects {
				err := s.testClient.Delete(context.Background(), obj)
				s.Require().NoError(err)
			}
		})
	}
}

func (s *serviceTest) TestGetAccountForNamespace() {
	tests := []struct {
		name            string
		mockObjects     []client.Object
		namespace       string
		expectedAccount client.ObjectKey
		expectError     bool
	}{
		{
			name: "",
			mockObjects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "account-for-namespace",
						Labels: map[string]string{
							v1alpha1.NamespaceAccountOwnerNamespaceLabel: "root-namespace",
							v1alpha1.NamespaceAccountOwnerLabel:          "test-account",
						},
					},
				},
				&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-account",
						Namespace: "root-namespace",
					},
					Spec: v1alpha1.AccountSpec{
						Type: v1alpha1.AccountTypeAccount,
					},
				},
			},
			namespace: "account-for-namespace",
			expectedAccount: types.NamespacedName{
				Namespace: "root-namespace",
				Name:      "test-account",
			},
		},
		{
			name: "missing namespace",
			mockObjects: []client.Object{
				&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-account",
						Namespace: "root-namespace",
					},
					Spec: v1alpha1.AccountSpec{
						Type: v1alpha1.AccountTypeAccount,
					},
				},
			},
			namespace: "account-for-namespace-4",
			expectedAccount: types.NamespacedName{
				Namespace: "root-namespace",
				Name:      "test-account",
			},
			expectError: true,
		},
		{
			name: "return an error due to one missing owner-namespace label",
			mockObjects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "account-for-namespace-2",
						Labels: map[string]string{
							v1alpha1.NamespaceAccountOwnerNamespaceLabel: "root-namespace",
						},
					},
				},
				&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-account",
						Namespace: "root-namespace",
					},
					Spec: v1alpha1.AccountSpec{
						Type: v1alpha1.AccountTypeAccount,
					},
				},
			},
			namespace: "account-for-namespace-2",
			expectedAccount: types.NamespacedName{
				Namespace: "root-namespace",
				Name:      "test-account",
			},
			expectError: true,
		},
		{
			name: "return an error due to missing owner name label",
			mockObjects: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "account-for-namespace-3",
					},
				},
				&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-account",
						Namespace: "root-namespace",
					},
					Spec: v1alpha1.AccountSpec{
						Type: v1alpha1.AccountTypeAccount,
					},
				},
			},
			namespace: "account-for-namespace-3",
			expectedAccount: types.NamespacedName{
				Namespace: "root-namespace",
				Name:      "test-account",
			},
			expectError: true,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			svc := service.NewService(s.testClient, "root-namespace")

			for _, obj := range test.mockObjects {
				err := s.testClient.Create(context.Background(), obj)
				s.Require().NoError(err)
			}

			defer func() {
				for _, obj := range test.mockObjects {
					err := s.testClient.Delete(context.Background(), obj)
					s.Require().NoError(err)
				}
			}()

			account, err := svc.GetAccountForNamespace(context.Background(), test.namespace)
			if test.expectError {
				s.Require().Error(err)
				return
			} else {
				s.Require().NoError(err)
			}

			s.Require().Equal(test.expectedAccount.Name, account.Name)
			s.Require().Equal(test.expectedAccount.Namespace, account.Namespace)

		})
	}
}

func (s *serviceTest) TestGetFirstLevelAccount() {
	tests := []struct {
		name            string
		mockObjects     []client.Object
		expectedAccount client.ObjectKey
		namespace       string
	}{
		{
			name: "",
			mockObjects: []client.Object{
				&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "first-level-account",
						Namespace: "root-namespace",
					},
					Spec: v1alpha1.AccountSpec{
						Type: v1alpha1.AccountTypeFolder,
					},
					Status: v1alpha1.AccountStatus{
						Namespace: pointer.To("sub-namespace"),
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "first-level-sub-namespace",
						Labels: map[string]string{
							v1alpha1.NamespaceAccountOwnerNamespaceLabel: "root-namespace",
							v1alpha1.NamespaceAccountOwnerLabel:          "first-level-account",
						},
					},
				},
				&v1alpha1.Account{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sub-account",
						Namespace: "first-level-sub-namespace",
					},
					Spec: v1alpha1.AccountSpec{
						Type: v1alpha1.AccountTypeFolder,
					},
				},
			},
			namespace: "first-level-sub-namespace",
			expectedAccount: types.NamespacedName{
				Namespace: "root-namespace",
				Name:      "first-level-account",
			},
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			svc := service.NewService(s.testClient, "root-namespace")

			for _, obj := range test.mockObjects {
				err := s.testClient.Create(context.Background(), obj)
				s.Require().NoError(err)
			}

			account, err := svc.GetFirstLevelAccountForNamespace(context.Background(), test.namespace)
			s.Require().NoError(err)

			s.Require().Equal(test.expectedAccount.Name, account.Name)
			s.Require().Equal(test.expectedAccount.Namespace, account.Namespace)

			account, err = svc.GetFirstLevelAccountForAccount(context.Background(), types.NamespacedName{Namespace: test.namespace, Name: "sub-account"})
			s.Require().NoError(err)

			s.Require().Equal(test.expectedAccount.Name, account.Name)
			s.Require().Equal(test.expectedAccount.Namespace, account.Namespace)

			slices.Reverse(test.mockObjects)

			for _, obj := range test.mockObjects {
				err := s.testClient.Delete(context.Background(), obj)
				s.Require().NoError(err)
			}
		})
	}
}
