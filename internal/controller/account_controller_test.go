package controller

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
	"github.com/openmfp/account-operator/internal/subroutines"
	openmfpcontext "github.com/openmfp/golang-commons/context"
	"github.com/openmfp/golang-commons/logger"
)

const (
	defaultTestTimeout  = 10 * time.Second
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
	log, err := logger.New(logConfig)
	suite.Nil(err)
	// Disable color logging as vs-code does not support color logging in the test output
	log = logger.NewFromZerolog(log.Output(&zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}))

	cfg, err := config.NewFromEnv()
	suite.Nil(err)

	testContext, _, _ := openmfpcontext.StartContext(log, cfg, cfg.ShutdownTimeout)

	testContext = logger.SetLoggerInContext(testContext, log.ComponentLogger("TestSuite"))

	suite.testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "chart", "crds")},
		ErrorIfCRDPathMissing: true,
	}

	k8scfg, err := suite.testEnv.Start()
	suite.Nil(err)

	utilruntime.Must(corev1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1.AddToScheme(scheme.Scheme))

	// +kubebuilder:scaffold:scheme

	suite.kubernetesClient, err = client.New(k8scfg, client.Options{
		Scheme: scheme.Scheme,
	})
	suite.Nil(err)
	ctrl.SetLogger(log.Logr())
	suite.kubernetesManager, err = ctrl.NewManager(k8scfg, ctrl.Options{
		Scheme:      scheme.Scheme,
		BaseContext: func() context.Context { return testContext },
	})
	suite.Nil(err)

	accountReconciler := NewAccountReconciler(log, suite.kubernetesManager, cfg)
	err = accountReconciler.SetupWithManager(suite.kubernetesManager, cfg, log)
	suite.Nil(err)

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
	suite.Nil(err)
}

func (suite *AccountTestSuite) TestAddingFinalizer() {
	// Given
	testContext := context.Background()
	accountName := "test-account-finalizer"

	account := &corev1alpha1.Account{
		ObjectMeta: metaV1.ObjectMeta{
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

	suite.Equal(createdAccount.ObjectMeta.Finalizers, []string{subroutines.NamespaceSubroutineFinalizer})
}

func (suite *AccountTestSuite) TestNamespaceCreation() {
	// Given
	testContext := context.Background()
	accountName := "test-account-ns-creation"
	account := &corev1alpha1.Account{
		ObjectMeta: metaV1.ObjectMeta{
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
		ObjectMeta: metaV1.ObjectMeta{
			Name:      accountName,
			Namespace: defaultNamespace,
		},
		Spec: corev1alpha1.AccountSpec{
			Type:      corev1alpha1.AccountTypeFolder,
			Namespace: &existingNamespaceName,
		}}

	nsToCreate := &v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: existingNamespaceName}}
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
	}, time.Second*30, time.Millisecond*250)

	suite.Assert().Equal(existingNamespaceName, *createdAccount.Status.Namespace)
	// Test if Namespace exists
	suite.verifyNamespace(testContext, accountName, defaultNamespace, createdAccount.Status.Namespace)
}

func (suite *AccountTestSuite) verifyNamespace(
	ctx context.Context, accName string, accNamespace string, nsName *string) {

	suite.Require().NotNil(nsName, "failed to verify namespace name")
	ns := &v1.Namespace{}
	err := suite.kubernetesClient.Get(ctx, types.NamespacedName{Name: *nsName}, ns)
	suite.Nil(err)

	suite.Assert().Contains(ns.GetLabels(), subroutines.NamespaceAccountOwnerLabel,
		"failed to verify account label on namespace")
	suite.Assert().Contains(ns.GetLabels(), subroutines.NamespaceAccountOwnerNamespaceLabel,
		"failed to verify account namespace label on namespace")

	suite.Assert().Equal(ns.GetLabels()[subroutines.NamespaceAccountOwnerLabel], accName,
		"failed to verify account label on namespace")
	suite.Assert().Contains(ns.GetLabels()[subroutines.NamespaceAccountOwnerNamespaceLabel], accNamespace,
		"failed to verify account namespace label on namespace")
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
