package controller

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	kcptenancyv1alpha "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	openmfpcontext "github.com/openmfp/golang-commons/context"
	"github.com/openmfp/golang-commons/logger"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/kcp"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
	"github.com/openmfp/account-operator/pkg/subroutines"
	"github.com/openmfp/account-operator/pkg/testing/kcpenvtest"
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
	testEnv           *kcpenvtest.Environment

	cancel context.CancelCauseFunc
}

func (suite *AccountTestSuite) SetupSuite() {
	logConfig := logger.DefaultConfig()
	logConfig.NoJSON = true
	logConfig.Name = "AccountTestSuite"
	logConfig.Level = "debug"

	log, err := logger.New(logConfig)
	suite.Require().NoError(err)
	ctrl.SetLogger(log.Logr())

	cfg, err := config.NewFromEnv()
	suite.Require().NoError(err)

	testContext, cancel, _ := openmfpcontext.StartContext(log, cfg, cfg.ShutdownTimeout)
	suite.cancel = cancel

	testEnvLogger := log.ComponentLogger("kcpenvtest")

	suite.testEnv = kcpenvtest.NewEnvironment("core.openmfp.org", "openmfp-system", "../../", "bin", "test/setup", testEnvLogger)

	var k8sCfg *rest.Config
	var vsUrl string
	useExistingCluster := true
	if envValue, err := strconv.ParseBool(os.Getenv("USE_EXISTING_CLUSTER")); err != nil {
		useExistingCluster = envValue
	}
	k8sCfg, vsUrl, err = suite.testEnv.Start(useExistingCluster)
	if err != nil {
		err = suite.testEnv.Stop(useExistingCluster)
		suite.Require().NoError(err)
	}
	suite.Require().NoError(err)

	utilruntime.Must(corev1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1.AddToScheme(scheme.Scheme))

	managerCfg := rest.CopyConfig(k8sCfg)
	managerCfg.Host = vsUrl

	testDataConfig := rest.CopyConfig(k8sCfg)
	testDataConfig.Host = fmt.Sprintf("%s:%s", k8sCfg.Host, "openmfp:organizations:root-org")

	// +kubebuilder:scaffold:scheme
	suite.kubernetesClient, err = client.New(testDataConfig, client.Options{
		Scheme: scheme.Scheme,
	})
	suite.Require().NoError(err)

	suite.kubernetesManager, err = kcp.NewClusterAwareManager(managerCfg, ctrl.Options{
		Scheme:      scheme.Scheme,
		Logger:      log.Logr(),
		BaseContext: func() context.Context { return testContext },
	})
	suite.Require().NoError(err)

	accountReconciler := NewAccountReconciler(log, suite.kubernetesManager, cfg)
	err = accountReconciler.SetupWithManager(suite.kubernetesManager, cfg, log)
	suite.Require().NoError(err)

	go suite.startController(testContext)
}

func (suite *AccountTestSuite) TearDownSuite() {
	suite.cancel(fmt.Errorf("tearing down test suite"))
	useExistingCluster := true
	if envValue, err := strconv.ParseBool(os.Getenv("USE_EXISTING_CLUSTER")); err != nil {
		useExistingCluster = envValue
	}
	err := suite.testEnv.Stop(useExistingCluster)
	suite.Nil(err)
}

func (suite *AccountTestSuite) startController(ctx context.Context) {
	err := suite.kubernetesManager.Start(ctx)
	suite.Require().NoError(err)
}

func (suite *AccountTestSuite) TestAddingFinalizer() {
	// Given
	testContext := context.Background()
	accountName := "test-account-finalizer"

	account := &corev1alpha1.Account{
		ObjectMeta: metav1.ObjectMeta{
			Name: accountName,
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

	suite.Equal([]string{subroutines.WorkspaceSubroutineFinalizer, subroutines.ExtensionSubroutineFinalizer, "account.core.openmfp.org/fga"}, createdAccount.ObjectMeta.Finalizers)
}

func (suite *AccountTestSuite) TestWorkspaceCreation() {
	// Given
	testContext := context.Background()
	accountName := "test-account-ws-creation"
	//account := &corev1alpha1.Account{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name: accountName,
	//	},
	//	Spec: corev1alpha1.AccountSpec{
	//		Type: corev1alpha1.AccountTypeFolder,
	//	}}
	//
	//// When
	//err := suite.kubernetesClient.Create(testContext, account)
	//suite.Nil(err)

	// Then
	createdAccount := corev1alpha1.Account{}
	suite.Assert().Eventually(func() bool {
		err := suite.kubernetesClient.Get(testContext, types.NamespacedName{
			Name:      accountName,
			Namespace: defaultNamespace,
		}, &createdAccount)
		return err == nil && createdAccount.Status.Workspace != nil
	}, defaultTestTimeout, defaultTickInterval)

	// Test if Workspace exists
	suite.verifyWorkspace(testContext, accountName, accountName)
}

//	func (suite *AccountTestSuite) TestNamespaceUsingExistingNamespace() {
//		// Given
//		testContext := context.Background()
//		accountName := "test-account-existing-namespace"
//		existingNamespaceName := "existing-namespace"
//
//		account := &corev1alpha1.Account{
//			ObjectMeta: metav1.ObjectMeta{
//				Name:      accountName,
//				Workspace: defaultNamespace,
//			},
//			Spec: corev1alpha1.AccountSpec{
//				Type:      corev1alpha1.AccountTypeFolder,
//				Workspace: &existingNamespaceName,
//			},
//		}
//
//		nsToCreate := &v1.Workspace{ObjectMeta: metav1.ObjectMeta{Name: existingNamespaceName}}
//		err := suite.kubernetesClient.Create(testContext, nsToCreate)
//		suite.Nil(err)
//
//		// When
//		err = suite.kubernetesClient.Create(testContext, account)
//		suite.Nil(err)
//
//		// Then
//		createdAccount := corev1alpha1.Account{}
//		suite.Assert().Eventually(func() bool {
//			err := suite.kubernetesClient.Get(testContext, types.NamespacedName{
//				Name:      accountName,
//				Workspace: defaultNamespace,
//			}, &createdAccount)
//			return err == nil && createdAccount.Status.Workspace != nil
//		}, defaultTestTimeout, defaultTickInterval)
//
//		suite.Assert().Equal(existingNamespaceName, *createdAccount.Status.Workspace)
//		// Test if Workspace exists
//		suite.verifyWorkspace(testContext, accountName, defaultNamespace, createdAccount.Status.Workspace)
//	}
//
// func (suite *AccountTestSuite) TestExtensionProcessing() {
//
//	accountName := "test-account-extension-creation"
//
//	testExtensionResource := `{
//		"podSelector": {
//			"matchLabels": {
//				"openmfp-owner": "{{ .Account.metadata.name }}"
//			}
//		}
//	}`
//
//	account := &corev1alpha1.Account{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      accountName,
//			Workspace: defaultNamespace,
//		},
//		Spec: corev1alpha1.AccountSpec{
//			Type: corev1alpha1.AccountTypeAccount,
//			Extensions: []corev1alpha1.Extension{
//				{
//					TypeMeta: metav1.TypeMeta{
//						APIVersion: "networking.k8s.io/v1",
//						Kind:       "NetworkPolicy",
//					},
//					SpecGoTemplate: apiextensionsv1.JSON{
//						Raw: []byte(testExtensionResource),
//					},
//				},
//			},
//		},
//	}
//
//	err := suite.kubernetesClient.Create(context.Background(), account)
//	suite.Assert().NoError(err)
//
//	// Then
//	createdAccount := corev1alpha1.Account{}
//	createdNetworkPolicy := networkv1.NetworkPolicy{}
//	suite.Assert().Eventually(func() bool {
//		err := suite.kubernetesClient.Get(context.Background(), types.NamespacedName{
//			Name:      accountName,
//			Workspace: defaultNamespace,
//		}, &createdAccount)
//		if err != nil || createdAccount.Status.Workspace == nil {
//			return false
//		}
//
//		err = suite.kubernetesClient.Get(context.Background(), types.NamespacedName{
//			Name:      "networkpolicy",
//			Workspace: *createdAccount.Status.Workspace,
//		}, &createdNetworkPolicy)
//
//		return err == nil && createdNetworkPolicy.Spec.PodSelector.MatchLabels["openmfp-owner"] == accountName
//	}, time.Second*30, time.Millisecond*250)
//
// }
func (suite *AccountTestSuite) verifyWorkspace(ctx context.Context, accName string, name string) {

	suite.Require().NotNil(name, "failed to verify namespace name")
	ns := &kcptenancyv1alpha.Workspace{}
	err := suite.kubernetesClient.Get(ctx, types.NamespacedName{Name: name}, ns)
	suite.Nil(err)

	suite.Assert().Len(ns.GetOwnerReferences(), 1, "failed to verify owner reference on workspace")
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
