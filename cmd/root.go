package cmd

import (
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	tenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	openmfpconfig "github.com/openmfp/golang-commons/config"
	"github.com/openmfp/golang-commons/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
)

var (
	scheme      = runtime.NewScheme()
	setupLog    = ctrl.Log.WithName("setup")
	operatorCfg config.OperatorConfig
	defaultCfg  *openmfpconfig.CommonServiceConfig
	v           *viper.Viper
	log         *logger.Logger
)

var rootCmd = &cobra.Command{
	Use:   "account-operator",
	Short: "operator to reconcile Accounts",
}

func init() {
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(tenancyv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apisv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	rootCmd.AddCommand(operatorCmd)

	var err error
	v, defaultCfg, err = openmfpconfig.NewDefaultConfig(rootCmd)
	if err != nil {
		panic(err)
	}

	cobra.OnInitialize(initConfig)

	err = openmfpconfig.BindConfigToFlags(v, operatorCmd, &operatorCfg)
	if err != nil {
		panic(err)
	}

	cobra.OnInitialize(initLog)
}

func initConfig() {
	v.SetDefault("subroutines-workspace-enabled", true)
	v.SetDefault("subroutines-account-info-enabled", true)
	v.SetDefault("subroutines-fga-enabled", true)
	v.SetDefault("subroutines-fga-root-namespace", "openmfp-root")
	v.SetDefault("subroutines-fga-grpc-addr", "localhost:8081")
	v.SetDefault("subroutines-fga-object-type", "account")
	v.SetDefault("subroutines-fga-parent-relation", "parent")
	v.SetDefault("subroutines-fga-creator-relation", "owner")
	v.SetDefault("kcp-provider-workspace", "root")
	v.SetDefault("webhooks-enabled", "false")
	v.SetDefault("webhooks-cert-dir", "certs")
}

func initLog() { // coverage-ignore
	logcfg := logger.DefaultConfig()
	logcfg.Level = defaultCfg.Log.Level
	logcfg.NoJSON = defaultCfg.Log.NoJson

	var err error
	log, err = logger.New(logcfg)
	if err != nil {
		panic(err)
	}
	ctrl.SetLogger(log.Logr())
	setupLog = ctrl.Log.WithName("setup") // coverage-ignore
}

func Execute() { // coverage-ignore
	cobra.CheckErr(rootCmd.Execute())
}
