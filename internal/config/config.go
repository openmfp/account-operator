package config

// OperatorConfig struct to hold the app config
type OperatorConfig struct {
	Webhooks struct {
		Enabled bool   `mapstructure:"webhooks--enabled"`
		CertDir string `mapstructure:"webhooks-cert-dir"`
	} `mapstructure:",squash"`
	Subroutines struct {
		Workspace struct {
			Enabled bool `mapstructure:"subroutines-workspace-enabled"`
		} `mapstructure:",squash"`
		AccountInfo struct {
			Enabled bool `mapstructure:"subroutines-account-info-enabled"`
		} `mapstructure:",squash"`
		FGA struct {
			Enabled         bool   `mapstructure:"subroutines-fga-enabled"`
			RootNamespace   string `mapstructure:"subroutines-fga-root-namespace"`
			GrpcAddr        string `mapstructure:"subroutines-fga-grpc-addr"`
			ObjectType      string `mapstructure:"subroutines-fga-object-type"`
			ParentRelation  string `mapstructure:"subroutines-fga-parent-relation"`
			CreatorRelation string `mapstructure:"subroutines-fga-creator-relation"`
		} `mapstructure:",squash"`
	} `mapstructure:",squash"`
	Kcp struct {
		ApiExportEndpointSliceName string `mapstructure:"kcp-api-export-endpoint-slice-name"`
		ProviderWorkspace          string `mapstructure:"kcp-provider-workspace"`
	} `mapstructure:",squash"`
}
