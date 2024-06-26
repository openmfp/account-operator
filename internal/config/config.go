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
		Extension struct {
			Enabled bool `envconfig:"default=true"`
		}
		ExtensionReady struct {
			Enabled bool `envconfig:"default=true"`
		}
	}
	MaxConcurrentReconciles int `envconfig:"default=10"`
}

// NewFromEnv creates a Config from environment values
func NewFromEnv() (Config, error) {
	appConfig := Config{}
	err := envconfig.Init(&appConfig)
	return appConfig, err
}
