/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"crypto/tls"
	"os"

	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	openmfpcontext "github.com/openmfp/golang-commons/context"
	"github.com/openmfp/golang-commons/logger"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/kcp"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/openmfp/account-operator/api/v1alpha1"
	"github.com/openmfp/account-operator/internal/config"
	"github.com/openmfp/account-operator/internal/controller"
)

var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "operator to reconcile Accounts",
	Run:   RunController,
}

var (
	metricsAddr          string
	enableLeaderElection bool
	probeAddr            string
	loglevel             string
	logNoJson            bool
	secureMetrics        bool
	enableHTTP2          bool
	cfg                  config.Config
)

func init() { // coverage-ignore
	var err error
	cfg, err = config.NewFromEnv()
	if err != nil {
		setupLog.Error(err, "unable to load config")
		os.Exit(1)
	}
	operatorCmd.Flags().StringVar(&metricsAddr, "metrics-bind-address", cfg.Metrics.BindAddress,
		"The address the metric endpoint binds to.")
	operatorCmd.Flags().StringVar(&probeAddr, "health-probe-bind-address", cfg.Probes.BindAddress,
		"The address the probe endpoint binds to.")
	operatorCmd.Flags().StringVar(&loglevel, "log-level", cfg.Log.Level,
		"The log level for the application. Default is info.")
	operatorCmd.Flags().BoolVar(&logNoJson, "log-no-json", cfg.Log.NoJson,
		"Flag to disable JSON logging. Default is false.")
	operatorCmd.Flags().BoolVar(&enableLeaderElection, "leader-elect", cfg.LeaderElection.Enabled,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	operatorCmd.Flags().BoolVar(&secureMetrics, "metrics-secure", cfg.Metrics.Secure,
		"If set the metrics endpoint is served securely")
	operatorCmd.Flags().BoolVar(&enableHTTP2, "enable-http2", cfg.EnableHttp2,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
}

func RunController(_ *cobra.Command, _ []string) { // coverage-ignore
	log := initLog()
	ctrl.SetLogger(log.ComponentLogger("controller-runtime").Logr())

	ctx, _, shutdown := openmfpcontext.StartContext(log, cfg, cfg.ShutdownTimeout)
	defer shutdown()

	disableHTTP2 := func(c *tls.Config) {
		log.Info().Msg("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	var tlsOpts []func(*tls.Config)
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
		CertDir: cfg.Webhooks.CertDir,
	})
	restCfg := ctrl.GetConfigOrDie()
	opts := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		BaseContext:                   func() context.Context { return ctx },
		WebhookServer:                 webhookServer,
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              "8c290d9a.openmfp.org",
		LeaderElectionConfig:          restCfg,
		LeaderElectionReleaseOnCancel: true,
	}
	var mgr ctrl.Manager
	var err error
	mgrConfig := rest.CopyConfig(restCfg)
	if len(cfg.Kcp.ApiExportEndpointSliceName) > 0 {
		// Lookup API Endpointslice
		kclient, err := client.New(restCfg, client.Options{
			Scheme: scheme,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("unable to create client")
		}
		es := &apisv1alpha1.APIExportEndpointSlice{}
		err = kclient.Get(ctx, client.ObjectKey{Name: cfg.Kcp.ApiExportEndpointSliceName}, es)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to create client")
		}
		if len(es.Status.APIExportEndpoints) == 0 {
			log.Fatal().Msg("no APIExportEndpoints found")
		}
		log.Info().Str("host", es.Status.APIExportEndpoints[0].URL).Msg("using host")
		mgrConfig.Host = es.Status.APIExportEndpoints[0].URL
	}
	mgr, err = kcp.NewClusterAwareManager(mgrConfig, opts)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to start manager")
	}

	var fgaClient openfgav1.OpenFGAServiceClient
	if cfg.Subroutines.FGA.Enabled {
		log.Debug().Str("GrpcAddr", cfg.Subroutines.FGA.GrpcAddr).Msg("Creating FGA Client")
		conn, err := grpc.NewClient(cfg.Subroutines.FGA.GrpcAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		)
		if err != nil {

			log.Fatal().Err(err).Msg("error when creating the grpc client")
		}
		log.Debug().Msg("FGA client created")

		fgaClient = openfgav1.NewOpenFGAServiceClient(conn)
	}

	accountReconciler := controller.NewAccountReconciler(log, mgr, cfg, fgaClient)
	if err := accountReconciler.SetupWithManager(mgr, cfg, log); err != nil {
		log.Fatal().Err(err).Str("controller", "Account").Msg("unable to create controller")
	}

	if cfg.Webhooks.Enabled {
		if err := v1alpha1.SetupAccountWebhookWithManager(mgr); err != nil {
			log.Fatal().Err(err).Str("webhook", "Account").Msg("unable to create webhook")
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Fatal().Err(err).Msg("unable to set up health check")
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Fatal().Err(err).Msg("unable to set up ready check")
	}

	log.Info().Msg("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatal().Err(err).Msg("problem running manager")
	}
}

func initLog() *logger.Logger { // coverage-ignore
	cfg := logger.DefaultConfig()
	cfg.Level = loglevel
	cfg.NoJSON = logNoJson
	log, err := logger.New(cfg)
	if err != nil {
		setupLog.Error(err, "unable to create logger")
		os.Exit(1)
	}
	return log
}
