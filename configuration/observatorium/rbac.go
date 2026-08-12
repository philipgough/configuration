package cfgobservatorium

import (
	"fmt"
	"strings"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/encoding"
	"github.com/observatorium/api/rbac"
)

type TenantID string

const (
	cnvqeTenant     TenantID = "cnvqe"
	telemeterTenant TenantID = "telemeter"
	rhobsTenant     TenantID = "rhobs"
	psiocpTenant    TenantID = "psiocp"
	rhodsTenant     TenantID = "rhods"
	rhacsTenant     TenantID = "rhacs"
	odfmsTenant     TenantID = "odfms"
	refAddonTenant  TenantID = "reference-addon"
	rhtapTenant     TenantID = "rhtap"
	rhelTenant      TenantID = "rhel"

	HcpTenant TenantID = "hcp"
)

type Resource string

const (
	MetricsResource Resource = "metrics"
	LogsResource    Resource = "logs"
	ProbesResource  Resource = "probes"
)

type env string

const (
	testingEnv    env = "testing"
	stagingEnv    env = "staging"
	productionEnv env = "production"
)

func GenerateRBACFile(gen *mimic.Generator) {
	gen.Add("rbac.json", encoding.JSON(GenerateRBAC()))
}

// GenerateRBAC generates rbac.json that is meant to be consumed by observatorium.libsonnet
// and put into config map consumed by observatorium-api.
//
// RBAC defines roles and role binding for each tenant and matching subject names that will be validated
// against 'user' field in the incoming JWT token that contains service account.
//
// TODO(bwplotka): Generate tenants.yaml (without secrets) using the same tenant definitions.
func GenerateRBAC() *ObservatoriumRBAC {
	obsRBAC := ObservatoriumRBAC{
		mappedRoleNames: map[RoleMapKey]string{},
	}

	// CNV-QE
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-cnv-qe",
		tenant:  cnvqeTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write, rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})

	// RHODS
	// Starbust write-only
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-starburst-isv-write",
		tenant:  rhodsTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write},
		envs:    []env{stagingEnv},
	})
	// Starbust read-only
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-starburst-isv-read",
		tenant:  rhodsTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Read},
		envs:    []env{stagingEnv},
	})

	// RHACS
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-rhacs-metrics",
		tenant:  rhacsTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write, rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-rhacs-grafana",
		tenant:  rhacsTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})

	// RHOBS
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-rhobs",
		tenant:  rhobsTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write, rbac.Read},
		envs:    []env{testingEnv, stagingEnv, productionEnv},
	})
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-rhobs-mst",
		tenant:  rhobsTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write, rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})
	// Special admin role.
	obsRBAC.RoleBindings = append(obsRBAC.RoleBindings, rbac.RoleBinding{
		Name: "rhobs-admin",
		Roles: []string{
			getOrCreateRoleName(&obsRBAC, telemeterTenant, MetricsResource, rbac.Read),
			getOrCreateRoleName(&obsRBAC, rhobsTenant, MetricsResource, rbac.Read),
		},
		Subjects: []rbac.Subject{{Name: "team-monitoring@redhat.com", Kind: rbac.Group}},
	})

	// Telemeter
	attachBinding(&obsRBAC, BindingOpts{
		name:    "telemeter-service",
		tenant:  telemeterTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write, rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})

	// CCX Processing
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-ccx-processing",
		tenant:  telemeterTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})

	// SD TCS (App-interface progressive delivery feature)
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-sdtcs",
		tenant:  telemeterTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})

	// Subwatch
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-subwatch",
		tenant:  telemeterTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})

	// PSIOCP
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-psiocp",
		tenant:  psiocpTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write, rbac.Read},
		envs:    []env{stagingEnv},
	})

	// ODFMS
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-odfms-write",
		tenant:  odfmsTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write}, // Write only.
		envs:    []env{productionEnv},
	})
	// Special request of extra read account.
	// Ref: https://issues.redhat.com/browse/MON-2536?focusedCommentId=20492830&page=com.atlassian.jira.plugin.system.issuetabpanels:comment-tabpanel#comment-20492830
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-odfms-read",
		tenant:  odfmsTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Read}, // Read only.
		envs:    []env{productionEnv},
	})

	// ODFMS has one set of staging credentials that has read & write permissions
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-odfms",
		tenant:  odfmsTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Read, rbac.Write},
		envs:    []env{stagingEnv},
	})

	// reference-addon
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-reference-addon",
		tenant:  refAddonTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write, rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})

	// placeholder read only prod
	// Special request of extra read account.
	// https://issues.redhat.com/browse/RHOBS-1116
	attachBinding(&obsRBAC, BindingOpts{
		name:                "7f7f912e-0429-4639-8e70-609ecf65b280",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// analytics read only prod
	// Special request of extra read account.
	// https://issues.redhat.com/browse/RHOBS-1116
	attachBinding(&obsRBAC, BindingOpts{
		name:                "8f7aa5e1-aa08-493d-82eb-cf24834fc08f",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// data foundation pms read only prod
	// Special request of extra read account.
	// https://issues.redhat.com/browse/RHOBS-1116
	attachBinding(&obsRBAC, BindingOpts{
		name:                "4bfe1a9f-e875-4d37-9c6a-d2faff2a69dc",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// observability pms read only prod
	// Special request of extra read account.
	attachBinding(&obsRBAC, BindingOpts{
		name:                "f6b3e12c-bb50-4bfc-89fe-330a28820fa9",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// hybrid-platforms pms read only prod
	// Special request of extra read account.
	attachBinding(&obsRBAC, BindingOpts{
		name:                "1a45eb31-bcc6-4bb7-8a38-88f00aa718ee",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// cnv read only prod
	// Special request of extra read account.
	// https://issues.redhat.com/browse/RHOBS-1116
	attachBinding(&obsRBAC, BindingOpts{
		name:                "e7c2f772-e418-4ef3-9568-ea09b1acb929",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// dev-spaces read only prod
	// Special request of extra read account.
	// https://issues.redhat.com/browse/RHOBS-1116
	attachBinding(&obsRBAC, BindingOpts{
		name:                "e07f5b10-e62b-47a2-9698-e245d1198a3b",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// plmshift read only prod
	// Special request of extra read account.
	attachBinding(&obsRBAC, BindingOpts{
		name:                "8a5cc14c-570c-4106-9a3b-cb2fcf4e3de4",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// plmshift second read only prod
	// Special request of extra read account.
	attachBinding(&obsRBAC, BindingOpts{
		name:                "plmshift",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// partner-accelerator-tools read only prod
	// Special request of extra read account.
	attachBinding(&obsRBAC, BindingOpts{
		name:                "9baf25c1-f61e-4b0d-b3a5-41802dbc061e",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// ai-bu-pms read only prod
	// Special request of extra read account.
	attachBinding(&obsRBAC, BindingOpts{
		name:                "cefb23fb-d0a2-4c8f-9180-d95c259e79a3",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// fedramp write only stage
	// Special request of extra read account.
	attachBinding(&obsRBAC, BindingOpts{
		name:                "875c08bc-d313-417f-a044-295212338e81",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Write}, // Write only
		envs:                []env{stagingEnv, productionEnv},
		skipConventionCheck: true,
	})

	// fedramp write only prod
	// Special request of extra read account.
	attachBinding(&obsRBAC, BindingOpts{
		name:                "4cbd24b0-3aed-4b03-839a-f4515b199a5d",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Write}, // Write only
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// fedramp HCP billing read only prod
	// Special request of extra read account.
	// https://redhat.atlassian.net/browse/RHOBS-1477
	attachBinding(&obsRBAC, BindingOpts{
		name:                "bff5b6de-e6ef-4f83-8f5d-a1db8370c995",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// operator runtime read only prod
	// Special request of extra read account.
	// https://redhat.atlassian.net/browse/RHOBS-1476
	attachBinding(&obsRBAC, BindingOpts{
		name:                "d13bc04a-2763-4b3b-9ccb-a264f5d64aa9",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// obsint analytics read only prod
	// Special request of extra read account.
	// https://redhat.atlassian.net/browse/RHOBS-1703
	attachBinding(&obsRBAC, BindingOpts{
		name:                "745a997b-24eb-4b5f-9f53-58e94c6bbc53",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read}, // Read only.
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// rosa-core read/write
	// Special request of extra read account.
	attachBinding(&obsRBAC, BindingOpts{
		name:                "0174b0a8-649a-4a95-bdff-9592f41b0de4",
		tenant:              telemeterTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read},
		envs:                []env{productionEnv},
		skipConventionCheck: true,
	})

	// RHTAP
	// Reader and Writer serviceaccount
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-rhtap",
		tenant:  rhtapTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Read, rbac.Write},
		envs:    []env{stagingEnv, productionEnv},
	})

	// RHTAP - SREP -special access request
	// Reader and Writer serviceaccount
	attachBinding(&obsRBAC, BindingOpts{
		name:                "aed46b58-abb5-4b1e-831f-a5678de691e0",
		tenant:              rhtapTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read, rbac.Write},
		envs:                []env{stagingEnv},
		skipConventionCheck: true,
		withConcreteName:    true,
	})

	// RHTAP - SPRE Alert staging special read access request
	attachBinding(&obsRBAC, BindingOpts{
		name:                "1fc0f182-dfc0-40e2-a147-b768927d9c20",
		tenant:              rhtapTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read},
		envs:                []env{stagingEnv},
		skipConventionCheck: true,
		withConcreteName:    true,
	})

	// RHTAP - SPRE Alert production special read access request
	attachBinding(&obsRBAC, BindingOpts{
		name:                "4819e9be-e3e0-42de-b4a6-231443107d8e",
		tenant:              rhtapTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read},
		envs:                []env{productionEnv},
		skipConventionCheck: true,
		withConcreteName:    true,
	})

	// RHTAP - SPRE Alert production special write access request
	attachBinding(&obsRBAC, BindingOpts{
		name:                "44d9cf7d-69a9-4acd-88c2-dd66b338c1eb",
		tenant:              rhtapTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Write},
		envs:                []env{productionEnv},
		skipConventionCheck: true,
		withConcreteName:    true,
	})

	// RHTAP - Konflux Devprod staging special read access request
	// https://redhat.atlassian.net/browse/RHOBS-1699
	attachBinding(&obsRBAC, BindingOpts{
		name:                "7b8b11a7-17fa-4411-aef5-23096e130184",
		tenant:              rhtapTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read},
		envs:                []env{stagingEnv},
		skipConventionCheck: true,
		withConcreteName:    true,
	})

	// RHTAP - SPRE Alert production special read access request
	// https://redhat.atlassian.net/browse/RHOBS-1699
	attachBinding(&obsRBAC, BindingOpts{
		name:                "7a0665fe-cdc7-4bed-9453-4f97211130c2",
		tenant:              rhtapTenant,
		signals:             []Resource{MetricsResource},
		perms:               []rbac.Permission{rbac.Read},
		envs:                []env{productionEnv},
		skipConventionCheck: true,
		withConcreteName:    true,
	})

	// RHEL
	// Reader serviceaccount
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-rhel-read",
		tenant:  rhelTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Read},
		envs:    []env{stagingEnv, productionEnv},
	})
	// RHEL
	// Writer serviceaccount
	attachBinding(&obsRBAC, BindingOpts{
		name:    "observatorium-rhel-write",
		tenant:  rhelTenant,
		signals: []Resource{MetricsResource},
		perms:   []rbac.Permission{rbac.Write},
		envs:    []env{stagingEnv, productionEnv},
	})

	// Use JSON because we want to have jsonnet using that in configmaps/secrets.
	return &obsRBAC
}

// GenerateClusterRBAC generates rbac.json for the cluster
// RBAC defines roles and role binding for each tenant and matching subject names that will be validated
// against 'user' field in the incoming JWT token that contains service account.
func GenerateClusterRBAC(opts ...*BindingOpts) *ObservatoriumRBAC {
	obsRBAC := ObservatoriumRBAC{
		mappedRoleNames: map[RoleMapKey]string{},
	}

	for _, o := range opts {
		o.skipConventionCheck = true
		o.envs = []env{productionEnv}
		attachBinding(&obsRBAC, *o)
	}

	// Use JSON because we want to have jsonnet using that in configmaps/secrets.
	return &obsRBAC
}

type RoleMapKey struct {
	tenant TenantID
	signal Resource
	perm   rbac.Permission
}

// observatoriumRBAC represents the structure that is sued to parse RBAC configuration
// in Observatorium API: https://github.com/observatorium/api/blob/078b7ce75837bb03984f5ed99d2b69a512b696b5/rbac/rbac.go#L181.
type ObservatoriumRBAC struct {
	// mappedRoleNames is used for deduplication logic.
	mappedRoleNames map[RoleMapKey]string

	Roles        []rbac.Role        `json:"roles"`
	RoleBindings []rbac.RoleBinding `json:"roleBindings"`
}

type BindingOpts struct {
	// NOTE(bwplotka): Name is strongly correlated to subject name that corresponds to the service account username (it has to match it)/
	// Any change, require changes on tenant side, so be careful.
	name                string
	tenant              TenantID
	signals             []Resource
	perms               []rbac.Permission
	envs                []env
	skipConventionCheck bool
	// withConcreteName is used to bypass name generation logic and use the name as is.
	withConcreteName bool
	// withRawSubjectName skips the "service-account-" prefix, using the name exactly as provided.
	// Use this for OIDC client credentials flow where the token's client_id claim is the raw UUID.
	withRawSubjectName bool
}

func (bo *BindingOpts) WithServiceAccountName(n string) *BindingOpts {
	bo.name = n
	return bo
}

func (bo *BindingOpts) WithTenant(t TenantID) *BindingOpts {
	bo.tenant = t
	return bo
}

func (bo *BindingOpts) WithSignals(signals []Resource) *BindingOpts {
	bo.signals = signals
	return bo
}

func (bo *BindingOpts) WithPerms(perms []rbac.Permission) *BindingOpts {
	bo.perms = perms
	return bo
}

// WithRawSubjectName configures the binding to use the service account name exactly as provided,
// without adding the "service-account-" prefix. Use this for OIDC client credentials flow
// where the gateway's usernameClaim is set to "client_id" and the token contains the raw UUID.
func (bo *BindingOpts) WithRawSubjectName() *BindingOpts {
	bo.withRawSubjectName = true
	return bo
}

func getOrCreateRoleName(o *ObservatoriumRBAC, tenant TenantID, s Resource, p rbac.Permission) string {
	k := RoleMapKey{tenant: tenant, signal: s, perm: p}

	n, ok := o.mappedRoleNames[k]
	if !ok {
		n = fmt.Sprintf("%s-%s-%s", k.tenant, k.signal, k.perm)
		o.Roles = append(o.Roles, rbac.Role{
			Name:        n,
			Permissions: []rbac.Permission{k.perm},
			Resources:   []string{string(k.signal)},
			Tenants:     []string{string(k.tenant)},
		})
		o.mappedRoleNames[k] = n
	}
	return n
}

func tenantNameFollowsConvention(name string) (string, bool) {
	var envs = []env{stagingEnv, productionEnv, testingEnv}

	for _, e := range envs {
		if strings.HasSuffix(name, string(e)) {
			err := fmt.Sprintf(
				"found name breaking conventions with environment suffix: %s, should be: %s",
				name,
				strings.TrimRight(strings.TrimSuffix(name, string(e)), "-"),
			)
			return err, false
		}
	}

	return "", true
}

func attachBinding(o *ObservatoriumRBAC, opts BindingOpts) {
	for _, b := range o.RoleBindings {
		if b.Name == opts.name {
			mimic.Panicf("found duplicate binding name", opts.name)

		}
	}

	// Is there role that satisfy this already? If not, create.
	var roles []string
	for _, s := range opts.signals {
		for _, p := range opts.perms {
			roles = append(roles, getOrCreateRoleName(o, opts.tenant, s, p))
		}
	}

	var subs []rbac.Subject
	for _, e := range opts.envs {
		errMsg, ok := tenantNameFollowsConvention(opts.name)
		if !ok && !opts.skipConventionCheck {
			mimic.Panicf(errMsg)
		}

		var n string
		if opts.withRawSubjectName {
			// Use the name exactly as provided (for OIDC client credentials flow with client_id claim)
			n = opts.name
		} else if e == productionEnv || opts.withConcreteName {
			n = fmt.Sprintf("service-account-%s", opts.name)
		} else {
			n = fmt.Sprintf("service-account-%s-%s", opts.name, e)
		}

		subs = append(subs, rbac.Subject{Name: n, Kind: rbac.User})
	}

	o.RoleBindings = append(o.RoleBindings, rbac.RoleBinding{
		Name:     opts.name,
		Roles:    roles,
		Subjects: subs,
	})
}
