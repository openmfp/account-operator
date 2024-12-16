package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	openmfpcontext "github.com/openmfp/golang-commons/context"
	"github.com/openmfp/golang-commons/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
	"github.com/openmfp/account-operator/pkg/subroutines"
)

const (
	defaultTestTimeout  = 5 * time.Second
	defaultTickInterval = 250 * time.Millisecond
	defaultNamespace    = "default"
)

type AccountTestSuite struct {
	suite.Suite

	kubernetesClient  client.Client
	kubernetesManager ctrl.Manager
	testEnv           *envtest.Environment

	cancel context.CancelFunc
}

func (suite *AccountTestSuite) SetupSuite() {
	logConfig := logger.DefaultConfig()
	logConfig.NoJSON = true
	logConfig.Name = "AccountTestSuite"
	logConfig.Level = "debug"
	// Disable color logging as vs-code does not support color logging in the test output
	logConfig.Output = &zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}
	log, err := logger.New(logConfig)
	suite.Require().NoError(err)

	cfg, err := config.NewFromEnv()
	suite.Require().NoError(err)

	testContext, _, _ := openmfpcontext.StartContext(log, cfg, cfg.ShutdownTimeout)

	testContext = logger.SetLoggerInContext(testContext, log.ComponentLogger("TestSuite"))

	suite.testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	k8sCfg, err := suite.testEnv.Start()
	suite.Require().NoError(err)

	utilruntime.Must(corev1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1.AddToScheme(scheme.Scheme))

	// +kubebuilder:scaffold:scheme

	suite.kubernetesClient, err = client.New(k8sCfg, client.Options{
		Scheme: scheme.Scheme,
	})
	suite.Require().NoError(err)
	ctrl.SetLogger(log.Logr())
	suite.kubernetesManager, err = ctrl.NewManager(k8sCfg, ctrl.Options{
		Scheme:      scheme.Scheme,
		BaseContext: func() context.Context { return testContext },
	})
	suite.Require().NoError(err)

	accountReconciler := NewAccountReconciler(log, suite.kubernetesManager, cfg)
	err = accountReconciler.SetupWithManager(suite.kubernetesManager, cfg, log)
	suite.Require().NoError(err)

	go suite.startController()
}

func (suite *AccountTestSuite) TearDownSuite() {
	suite.cancel()
	err := suite.testEnv.Stop()
	suite.Nil(err)
}

func (suite *AccountTestSuite) startController() {
	var controllerContext context.Context
	controllerContext, suite.cancel = context.WithCancel(context.Background())
	err := suite.kubernetesManager.Start(controllerContext)
	suite.Require().NoError(err)
}

func (suite *AccountTestSuite) TestAddingFinalizer() {
	// Given
	testContext := context.Background()
	accountName := "test-account-finalizer"

	account := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accountName,
			Namespace: defaultNamespace,
		},
		Spec: corev1alpha1.AccountSpec{
			Type: corev1alpha1.AccountTypeFolder,
		}}

	// When
	err := suite.kubernetesClient.Create(testContext, account)
	suite.Nil(err)

	// Then
	createdAccount := corev1alpha1.Account{}
	suite.Assert().Eventually(func() bool {
		err := suite.kubernetesClient.Get(testContext, types.NamespacedName{
			Name:      accountName,
			Namespace: defaultNamespace,
		}, &createdAccount)
		return err == nil && createdAccount.Finalizers != nil
	}, defaultTestTimeout, defaultTickInterval)

	suite.Equal(createdAccount.ObjectMeta.Finalizers, []string{subroutines.NamespaceSubroutineFinalizer, "account.core.openmfp.io/fga"})
}

func (suite *AccountTestSuite) TestNamespaceCreation() {
	// Given
	testContext := context.Background()
	accountName := "test-account-ns-creation"
	account := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accountName,
			Namespace: defaultNamespace,
		},
		Spec: corev1alpha1.AccountSpec{
			Type: corev1alpha1.AccountTypeFolder,
		}}

	// When
	err := suite.kubernetesClient.Create(testContext, account)
	suite.Nil(err)

	// Then
	createdAccount := corev1alpha1.Account{}
	suite.Assert().Eventually(func() bool {
		err := suite.kubernetesClient.Get(testContext, types.NamespacedName{
			Name:      accountName,
			Namespace: defaultNamespace,
		}, &createdAccount)
		return err == nil && createdAccount.Status.Namespace != nil
	}, defaultTestTimeout, defaultTickInterval)

	// Test if Namespace exists
	suite.verifyNamespace(testContext, accountName, defaultNamespace, createdAccount.Status.Namespace)
}

func (suite *AccountTestSuite) TestNamespaceUsingExisitingNamespace() {
	// Given
	testContext := context.Background()
	accountName := "test-account-existing-namespace"
	existingNamespaceName := "existing-namespace"

	account := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accountName,
			Namespace: defaultNamespace,
		},
		Spec: corev1alpha1.AccountSpec{
			Type:      corev1alpha1.AccountTypeFolder,
			Namespace: &existingNamespaceName,
		},
	}

	nsToCreate := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: existingNamespaceName}}
	err := suite.kubernetesClient.Create(testContext, nsToCreate)
	suite.Nil(err)

	// When
	err = suite.kubernetesClient.Create(testContext, account)
	suite.Nil(err)

	// Then
	createdAccount := corev1alpha1.Account{}
	suite.Assert().Eventually(func() bool {
		err := suite.kubernetesClient.Get(testContext, types.NamespacedName{
			Name:      accountName,
			Namespace: defaultNamespace,
		}, &createdAccount)
		return err == nil && createdAccount.Status.Namespace != nil
	}, defaultTestTimeout, defaultTickInterval)

	suite.Assert().Equal(existingNamespaceName, *createdAccount.Status.Namespace)
	// Test if Namespace exists
	suite.verifyNamespace(testContext, accountName, defaultNamespace, createdAccount.Status.Namespace)
}

func (suite *AccountTestSuite) TestExtensionProcessing() {

	accountName := "test-account-extension-creation"

	testExtensionResource := `{
		"podSelector": {
			"matchLabels": {
				"openmfp-owner": "{{ .Account.ObjectMeta.Name }}"
			}
		}
	}`

	account := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accountName,
			Namespace: defaultNamespace,
		},
		Spec: corev1alpha1.AccountSpec{
			Type: corev1alpha1.AccountTypeAccount,
			Extensions: []corev1alpha1.Extension{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "networking.k8s.io/v1",
						Kind:       "NetworkPolicy",
					},
					SpecGoTemplate: apiextensionsv1.JSON{
						Raw: []byte(testExtensionResource),
					},
				},
			},
		},
	}

	err := suite.kubernetesClient.Create(context.Background(), account)
	suite.Assert().NoError(err)

	// Then
	createdAccount := corev1alpha1.Account{}
	createdNetworkPolicy := networkv1.NetworkPolicy{}
	suite.Assert().Eventually(func() bool {
		err := suite.kubernetesClient.Get(context.Background(), types.NamespacedName{
			Name:      accountName,
			Namespace: defaultNamespace,
		}, &createdAccount)
		if err != nil || createdAccount.Status.Namespace == nil {
			return false
		}

		err = suite.kubernetesClient.Get(context.Background(), types.NamespacedName{
			Name:      "networkpolicy",
			Namespace: *createdAccount.Status.Namespace,
		}, &createdNetworkPolicy)

		return err == nil && createdNetworkPolicy.Spec.PodSelector.MatchLabels["openmfp-owner"] == accountName
	}, time.Second*30, time.Millisecond*250)

}

func (suite *AccountTestSuite) verifyNamespace(
	ctx context.Context, accName string, accNamespace string, nsName *string) {

	suite.Require().NotNil(nsName, "failed to verify namespace name")
	ns := &v1.Namespace{}
	err := suite.kubernetesClient.Get(ctx, types.NamespacedName{Name: *nsName}, ns)
	suite.Nil(err)

	suite.Assert().Contains(ns.GetLabels(), corev1alpha1.NamespaceAccountOwnerLabel,
		"failed to verify account label on namespace")
	suite.Assert().Contains(ns.GetLabels(), corev1alpha1.NamespaceAccountOwnerNamespaceLabel,
		"failed to verify account namespace label on namespace")

	suite.Assert().Equal(ns.GetLabels()[corev1alpha1.NamespaceAccountOwnerLabel], accName,
		"failed to verify account label on namespace")
	suite.Assert().Contains(ns.GetLabels()[corev1alpha1.NamespaceAccountOwnerNamespaceLabel], accNamespace,
		"failed to verify account namespace label on namespace")
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
