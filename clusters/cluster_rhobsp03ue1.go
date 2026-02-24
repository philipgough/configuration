package clusters

import (
	"github.com/observatorium/api/rbac"
	observatoriumapi "github.com/observatorium/observatorium/configuration_go/abstr/kubernetes/observatorium/api"
	cfgobservatorium "github.com/rhobs/configuration/configuration/observatorium"
)

const (
	ClusterRHOBSUSEastOneShardTwoProduction ClusterName = "rhobsp03ue1"
)

func init() {
	RegisterCluster(ClusterConfig{
		Name:        ClusterRHOBSUSEastOneShardTwoProduction,
		Environment: EnvironmentProduction,
		Namespace:   "rhobs-production",
		GatewayConfig: NewGatewayConfig(
			WithMetricsEnabled(),
			WithLoggingEnabled(),
			WithSyntheticsEnabled(),
			WithTracingEnabled(),
			WithTenants(rhobsp03ue1Tenants()),
			WithRBAC(rhobsp03ue1RBAC()),
			WithCustomRoute("us-east-1-2.rhobs.api.openshift.com"),
		),
		Templates:  rhobsp03ue1TemplateMaps(),
		BuildSteps: rhobsp03ue1BuildSteps(),
	})
}

func rhobsp03ue1Tenants() observatoriumapi.Tenants {
	return observatoriumapi.Tenants{
		Tenants: []observatoriumapi.Tenant{
			{
				Name: "hcp",
				ID:   "EFD08939-FE1D-41A1-A28A-BE9A9BC68003",
				OIDC: &observatoriumapi.TenantOIDC{
					ClientID:      "${CLIENT_ID}",
					ClientSecret:  "${CLIENT_SECRET}",
					IssuerURL:     "https://sso.redhat.com/auth/realms/redhat-external",
					RedirectURL:   "https://observatorium-mst.api.openshift.com/oidc/odfms/callback",
					UsernameClaim: "client_id",
				},
			},
		},
	}
}

func rhobsp03ue1RBAC() cfgobservatorium.ObservatoriumRBAC {
	opts := &cfgobservatorium.BindingOpts{}
	opts.WithServiceAccountName("cd54dce2-590e-4ea4-9b83-a83c58205962").
		WithTenant(cfgobservatorium.HcpTenant).
		WithSignals([]cfgobservatorium.Resource{cfgobservatorium.MetricsResource, cfgobservatorium.LogsResource, cfgobservatorium.ProbesResource}).
		WithPerms([]rbac.Permission{rbac.Read, rbac.Write}).
		WithRawSubjectName()

	config := cfgobservatorium.GenerateClusterRBAC(opts)
	return *config
}

func rhobsp03ue1BuildSteps() []string {
	return []string{
		StepGateway,
		StepDefaultThanosStack,
		StepDefaultLokiStack,
		StepSyntheticsApi,
		StepAlertmanager,
	}
}

// rhobsp03ue1TemplateMaps returns template mappings specific to the ClusterRHOBSUSEastOneShardTwoProduction production cluster
func rhobsp03ue1TemplateMaps() TemplateMaps {
	return DefaultBaseTemplate().Override()
}
