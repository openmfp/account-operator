package cmd

import (
	tenancyv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/tenancy/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openmfp/account-operator/api/v1alpha1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var rootCmd = &cobra.Command{
	Use:   "account-operator",
	Short: "operator to reconcile Accounts",
}

func init() { // coverage-ignore
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(tenancyv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
	rootCmd.AddCommand(operatorCmd)

}

func Execute() { // coverage-ignore
	cobra.CheckErr(rootCmd.Execute())
}
