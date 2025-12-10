package clusters

import (
	"github.com/observatorium/api/rbac"
	observatoriumapi "github.com/observatorium/observatorium/configuration_go/abstr/kubernetes/observatorium/api"
	cfgobservatorium "github.com/rhobs/configuration/configuration/observatorium"
)

const (
	ClusterRHOBSUSEastOneStaging ClusterName = "rhobss01ue1"
)

func init() {
	RegisterCluster(ClusterConfig{
		Name:        ClusterRHOBSUSEastOneStaging,
		Environment: EnvironmentStaging,
		Namespace:   "rhobs-stage",
		GatewayConfig: NewGatewayConfig(
			WithMetricsEnabled(),
			WithLoggingEnabled(),
			WithSyntheticsEnabled(),
			WithTracingEnabled(),
			WithTenants(rhobss01ue1Tenants()),
			WithRBAC(rhobss01ue1RBAC()),
		),
		Templates:  rhobss01ue1TemplateMaps(),
		BuildSteps: rhobss01ue1sBuildSteps(),
	})
}

func rhobss01ue1Tenants() observatoriumapi.Tenants {
	return observatoriumapi.Tenants{
		Tenants: []observatoriumapi.Tenant{
			{
				Name: "hcp",
				ID:   "EFD08939-FE1D-41A1-A28A-BE9A9BC68003",
				OIDC: &observatoriumapi.TenantOIDC{
					ClientID:      "${CLIENT_ID}",
					ClientSecret:  "${CLIENT_SECRET}",
					IssuerURL:     "https://sso.redhat.com/auth/realms/redhat-external",
					RedirectURL:   "https://observatorium-mst.api.stage.openshift.com/oidc/odfms/callback",
					UsernameClaim: "preferred_username",
				},
			},
		},
	}
}

func rhobss01ue1RBAC() cfgobservatorium.ObservatoriumRBAC {
	opts := &cfgobservatorium.BindingOpts{}
	opts.WithServiceAccountName("d4045e4b-7b9c-46fc-8af0-5d483d9d205b").
		WithTenant(cfgobservatorium.HcpTenant).
		WithSignals([]cfgobservatorium.Resource{cfgobservatorium.MetricsResource, cfgobservatorium.LogsResource, cfgobservatorium.ProbesResource}).
		WithPerms([]rbac.Permission{rbac.Read, rbac.Write})

	config := cfgobservatorium.GenerateClusterRBAC(opts)
	return *config
}

func rhobss01ue1sBuildSteps() []string {
	return DefaultGatewayBuildSteps()
}

// rhobss01ue1TemplateMaps returns template mappings specific to the rhobss01ue1 integration cluster
func rhobss01ue1TemplateMaps() TemplateMaps {
	// Start with integration base template and override only what's different
	lokiOverrides := LokiOverridesMap{
		LokiConfig: LokiOverrides{
			LokiLimitOverrides: LokiLimitOverrides{
				IngestionRateLimitMB: 20,
				PerStreamRateLimitMB: 15,
				PerStreamBurstSizeMB: 30,
				QueryTimeout:         "5m",
			},
			Ingest: LokiComponentSpec{
				Replicas: 3,
			},
		},
	}
	return DefaultBaseTemplate().Override(lokiOverrides)
}
