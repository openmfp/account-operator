package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1alpha1 "github.com/openmfp/account-operator/api/v1alpha1"
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
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corev1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	rootCmd.AddCommand(operatorCmd)

}

func Execute() { // coverage-ignore
	cobra.CheckErr(rootCmd.Execute())
}
