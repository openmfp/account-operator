package kcpenvtest

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/openmfp/golang-commons/logger"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/rest"

	"github.com/openmfp/account-operator/pkg/testing/kcpenvtest/process"
)

const (
	kcpEnvStartTimeout      = "KCP_SERVER_START_TIMEOUT"
	kcpEnvStopTimeout       = "KCP_SERVER_STOP_TIMEOUT"
	defaulKCPServerTimeout  = 20 * time.Second
	defaultKCPServerTimeout = 20 * time.Second
)

type Environment struct {
	kcpServer             *KCPServer
	BinaryAssetsDirectory string
	// ControlPlaneStartTimeout is the maximum duration each controlplane component
	// may take to start. It defaults to the KUBEBUILDER_CONTROLPLANE_START_TIMEOUT
	// environment variable or 20 seconds if unspecified
	ControlPlaneStartTimeout time.Duration

	// ControlPlaneStopTimeout is the maximum duration each controlplane component
	// may take to stop. It defaults to the KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT
	// environment variable or 20 seconds if unspecified
	ControlPlaneStopTimeout time.Duration

	log *logger.Logger
}

func (te *Environment) Start() (*rest.Config, error) {

	//apiServer := te.ControlPlane.GetAPIServer()

	te.kcpServer.Path = process.BinPathFinder("kcp", te.BinaryAssetsDirectory)

	if err := te.defaultTimeouts(); err != nil {
		return nil, fmt.Errorf("failed to default controlplane timeouts: %w", err)
	}
	te.kcpServer.StartTimeout = te.ControlPlaneStartTimeout
	te.kcpServer.StopTimeout = te.ControlPlaneStopTimeout

	log.Info().Msg("starting control plane")
	if err := te.kcpServer.Start(); err != nil {
		return nil, fmt.Errorf("unable to start control plane itself: %w", err)
	}

	// Create the *rest.Config for creating new clients
	//baseConfig := &rest.Config{
	//	// gotta go fast during tests -- we don't really care about overwhelming our test API server
	//	QPS:   1000.0,
	//	Burst: 2000.0,
	//}

	//adminInfo := User{Name: "admin", Groups: []string{"system:masters"}}
	//adminUser, err := te.ControlPlane.AddUser(adminInfo, baseConfig)
	//if err != nil {
	//	return te.Config, fmt.Errorf("unable to provision admin user: %w", err)
	//}
	//te.Config = adminUser.Config()
	//
	//// Set the default scheme if nil.
	//if te.Scheme == nil {
	//	te.Scheme = scheme.Scheme
	//}
	//
	//// If we are bringing etcd up for the first time, it can take some time for the
	//// default namespace to actually be created and seen as available to the apiserver
	//if err := te.waitForDefaultNamespace(te.Config); err != nil {
	//	return nil, fmt.Errorf("default namespace didn't register within deadline: %w", err)
	//}
	//
	//// Call PrepWithoutInstalling to setup certificates first
	//// and have them available to patch CRD conversion webhook as well.
	//if err := te.WebhookInstallOptions.PrepWithoutInstalling(); err != nil {
	//	return nil, err
	//}
	//
	//log.V(1).Info("installing CRDs")
	//if te.CRDInstallOptions.Scheme == nil {
	//	te.CRDInstallOptions.Scheme = te.Scheme
	//}
	//te.CRDInstallOptions.CRDs = mergeCRDs(te.CRDInstallOptions.CRDs, te.CRDs)
	//te.CRDInstallOptions.Paths = mergePaths(te.CRDInstallOptions.Paths, te.CRDDirectoryPaths)
	//te.CRDInstallOptions.ErrorIfPathMissing = te.ErrorIfCRDPathMissing
	//te.CRDInstallOptions.WebhookOptions = te.WebhookInstallOptions
	//crds, err := InstallCRDs(te.Config, te.CRDInstallOptions)
	//if err != nil {
	//	return te.Config, fmt.Errorf("unable to install CRDs onto control plane: %w", err)
	//}
	//te.CRDs = crds
	//
	//log.V(1).Info("installing webhooks")
	//if err := te.WebhookInstallOptions.Install(te.Config); err != nil {
	//	return nil, fmt.Errorf("unable to install webhooks onto control plane: %w", err)
	//}
	return nil, nil
}

func (te *Environment) defaultTimeouts() error {
	var err error
	if te.ControlPlaneStartTimeout == 0 {
		if envVal := os.Getenv(kcpEnvStartTimeout); envVal != "" {
			te.ControlPlaneStartTimeout, err = time.ParseDuration(envVal)
			if err != nil {
				return err
			}
		} else {
			te.kcpServer.StartTimeout = defaulKCPServerTimeout
		}
	}

	if te.ControlPlaneStopTimeout == 0 {
		if envVal := os.Getenv(kcpEnvStopTimeout); envVal != "" {
			te.ControlPlaneStopTimeout, err = time.ParseDuration(envVal)
			if err != nil {
				return err
			}
		} else {
			te.ControlPlaneStopTimeout = defaultKCPServerTimeout
		}
	}
	return nil
}

type KCPServer struct {
	processState *process.State
	Out          io.Writer
	Err          io.Writer
	StartTimeout time.Duration
	StopTimeout  time.Duration
	Path         string
	Args         []string

	args *process.Arguments
}

func (s *KCPServer) Start() error {
	if err := s.prepare(); err != nil {
		return err
	}
	return s.processState.Start(s.Out, s.Err)
}

func (s *KCPServer) prepare() error {
	if err := s.setProcessState(); err != nil {
		return err
	}
	return nil
}

func (s *KCPServer) setProcessState() error {
	var err error

	// unconditionally re-set this so we can successfully restart
	// TODO(directxman12): we supported this in the past, but do we actually
	// want to support re-using an API server object to restart?  The loss
	// of provisioned users is surprising to say the least.
	s.processState = &process.State{
		Path:         s.Path,
		StartTimeout: s.StartTimeout,
		StopTimeout:  s.StopTimeout,
	}
	if err := s.processState.Init("kcp-server"); err != nil {
		return err
	}

	s.Path = s.processState.Path
	s.StartTimeout = s.processState.StartTimeout
	s.StopTimeout = s.processState.StopTimeout

	s.processState.Args, s.Args, err = process.TemplateAndArguments(s.Args, s.Configure(), process.TemplateDefaults{ //nolint:staticcheck
		Data:     s,
		Defaults: s.defaultArgs(),
		MinimalDefaults: map[string][]string{
			// as per kubernetes-sigs/controller-runtime#641, we need this (we
			// probably need other stuff too, but this is the only thing that was
			// previously considered a "minimal default")
			"service-cluster-ip-range": {"10.0.0.0/24"},

			// we need *some* authorization mode for health checks on the secure port,
			// so default to RBAC unless the user set something else (in which case
			// this'll be ignored due to SliceToArguments using AppendNoDefaults).
			"authorization-mode": {"RBAC"},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *KCPServer) defaultArgs() map[string][]string {
	args := map[string][]string{}
	return args
}

func (s *KCPServer) Configure() *process.Arguments {
	if s.args == nil {
		s.args = process.EmptyArguments()
	}
	return s.args
}

func (s *KCPServer) Stop() error {
	if s.processState != nil {
		if err := s.processState.Stop(); err != nil {
			return err
		}
	}
	return nil
}
