package config

import (
	"time"

	"github.com/vrischmann/envconfig"
)

// Config struct to hold the app config
type Config struct {
	DebugLabelValue string `envconfig:"optional"`
	Log             struct {
		Level  string `envconfig:"default=info"`
		NoJson bool   `envconfig:"default=false"`
	}
	ShutdownTimeout time.Duration `envconfig:"default=1s"`
	EnableHttp2     bool          `envconfig:"default=false"`
	Metrics         struct {
		BindAddress string `envconfig:"default=:8080"`
		Secure      bool   `envconfig:"default=false"`
	}
	Webhooks struct {
		Enabled bool   `envconfig:"default=false"`
		CertDir string `envconfig:"default=certs,optional"`
	}
	Probes struct {
		BindAddress string `envconfig:"default=:8081"`
	}
	LeaderElection struct {
		Enabled bool `envconfig:"default=false"`
	}
	Subroutines struct {
		Namespace struct {
			Enabled bool `envconfig:"default=true"`
		}
		FGA struct {
			Enabled         bool   `envconfig:"default=true"`
			RootNamespace   string `envconfig:"default=openmfp-root"`
			GrpcAddr        string `envconfig:"default=localhost:8081"`
			ObjectType      string `envconfig:"default=account"`
			ParentRelation  string `envconfig:"default=parent"`
			CreatorRelation string `envconfig:"default=owner"`
		}
		Extension struct {
			Enabled bool `envconfig:"default=true"`
		}
		ExtensionReady struct {
			Enabled bool `envconfig:"default=true"`
		}
	}
	MaxConcurrentReconciles int `envconfig:"default=10"`
	Kcp                     struct {
		Enabled             bool   `envconfig:"default=false"`
		VirtualWorkspaceUrl string `envconfig:"optional"`
	}
}

// NewFromEnv creates a Config from environment values
func NewFromEnv() (Config, error) {
	appConfig := Config{}
	err := envconfig.Init(&appConfig)
	return appConfig, err
}
