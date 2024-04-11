package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
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

	// +kubebuilder:scaffold:scheme

	suite.kubernetesClient, err = client.New(k8scfg, client.Options{
		Scheme: scheme.Scheme,
	})
	suite.Nil(err)

	suite.kubernetesManager, err = ctrl.NewManager(k8scfg, ctrl.Options{
		Scheme:      scheme.Scheme,
		BaseContext: func() context.Context { return testContext },
	})
	suite.Nil(err)

	accountReconciler := NewAccountReconciler(testContext, suite.kubernetesManager, cfg)
	err = accountReconciler.SetupWithManager(suite.kubernetesManager, cfg)
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

func (suite *AccountTestSuite) TestAccountReconciler() {
	testContext := context.Background()
	accountName := "test-account"
	accountNamespace := "default"
	account := &corev1alpha1.Account{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      accountName,
			Namespace: accountNamespace,
		},
		Spec: corev1alpha1.AccountSpec{
			AccountRole: corev1alpha1.AccountRoleFolder,
		}}

	err := suite.kubernetesClient.Create(testContext, account)
	suite.Nil(err)

	createdAccount := corev1alpha1.Account{}

	suite.Assert().Eventually(func() bool {
		err := suite.kubernetesClient.Get(testContext, types.NamespacedName{
			Name:      accountName,
			Namespace: accountNamespace,
		}, &createdAccount)
		return err == nil
	}, time.Second*30, time.Millisecond*250)

	suite.Assert().Eventually(func() bool {
		err := suite.kubernetesClient.Update(testContext, &createdAccount)
		return err == nil
	}, time.Second*30, time.Millisecond*250)

	suite.Equal(createdAccount.ObjectMeta.Finalizers, []string{subroutines.NamespaceSubroutineFinalizer})
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
