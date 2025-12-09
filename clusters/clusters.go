package clusters

import (
	"fmt"

	cfgobservatorium "github.com/rhobs/configuration/configuration/observatorium"

	observatoriumapi "github.com/observatorium/observatorium/configuration_go/abstr/kubernetes/observatorium/api"
)

// ClusterName represents a specific cluster identifier
type ClusterName string

// ClusterEnvironment represents the deployment environment
type ClusterEnvironment string

// Supported cluster environments
const (
	EnvironmentIntegration ClusterEnvironment = "integration"
	EnvironmentStaging     ClusterEnvironment = "staging"
	EnvironmentProduction  ClusterEnvironment = "production"
)

// ClusterConfig holds the configuration for a specific cluster deployment
type ClusterConfig struct {
	Name          ClusterName
	Environment   ClusterEnvironment
	Namespace     string
	Templates     TemplateMaps
	GatewayConfig *GatewayConfig
	BuildSteps    []string
}

type GatewayConfig struct {
	metricsEnabled    bool
	logsEnabled       bool
	syntheticsEnabled bool
	// tracing in this instance refers to internal tracing of the gateway itself
	tracingEnabled bool
	amsURL         string
	tenants        observatoriumapi.Tenants
	rbac           cfgobservatorium.ObservatoriumRBAC
}

// String returns the string representation of ClusterName
func (c ClusterName) String() string {
	return string(c)
}

// String returns the string representation of ClusterEnvironment
func (e ClusterEnvironment) String() string {
	return string(e)
}

// IsValid checks if the cluster environment is valid
func (e ClusterEnvironment) IsValid() bool {
	switch e {
	case EnvironmentIntegration, EnvironmentStaging, EnvironmentProduction:
		return true
	default:
		return false
	}
}

// Validate checks if the cluster configuration is valid
func (c ClusterConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}
	if !c.Environment.IsValid() {
		return fmt.Errorf("invalid environment: %s", c.Environment)
	}
	if c.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if len(c.BuildSteps) == 0 {
		return fmt.Errorf("cluster must have at least one build step")
	}
	return nil
}

// ClusterRegistry holds all registered clusters
var ClusterRegistry = make(map[ClusterName]ClusterConfig)

// BuildStep represents a string key for each template generation step
type BuildStep string

// Available build steps
const (
	StepThanosOperatorCRDS = "thanos-operator-crds"
	StepThanosOperator     = "thanos-operator"
	StepDefaultThanosStack = "default-thanos-stack"

	StepLokiOperatorCRDS = "loki-operator-crds"
	StepLokiOperator     = "loki-operator"
	StepDefaultLokiStack = "default-loki-stack"

	StepServiceMonitors = "servicemonitors"

	StepAlertmanager = "alertmanager"
	StepSecrets      = "secrets"
	StepGateway      = "gateway"
	StepMemcached    = "memcached"

	StepSyntheticsApi = "synthetics-api"
)

// DefaultBuildSteps returns the default build pipeline for clusters
func DefaultBuildSteps() []string {
	var steps []string
	steps = append(steps, DefaultMetricsBuildSteps()...)
	steps = append(steps, DefaultLoggingBuildSteps()...)
	steps = append(steps, DefaultSyntheticsBuildSteps()...)
	steps = append(steps, DefaultAlertingBuildSteps()...)

	steps = append(steps,
		StepServiceMonitors, // Monitoring setup
		StepSecrets,         // Secrets last
		StepMemcached,       // Memcached configuration
		StepGateway,         // Gateway configuration
	)
	return steps
}

func DefaultMetricsBuildSteps() []string {
	return []string{
		StepThanosOperatorCRDS, // Core components first
		StepThanosOperator,     // Custom Resource Definitions
		StepDefaultThanosStack, // ThanosOperator deployment
	}
}

func DefaultLoggingBuildSteps() []string {
	return []string{
		StepLokiOperatorCRDS,
		StepLokiOperator,
		StepDefaultLokiStack,
	}
}

func DefaultSyntheticsBuildSteps() []string {
	return []string{
		StepSyntheticsApi,
	}
}

func DefaultAlertingBuildSteps() []string {
	return []string{
		StepAlertmanager,
	}
}

// Prune is a utility function to remove specified steps from a list
func Prune(from []string, prune ...[]string) []string {
	pruneMap := make(map[string]struct{})
	for _, p := range prune {
		for _, step := range p {
			pruneMap[step] = struct{}{}
		}
	}

	var result []string
	for _, step := range from {
		if _, shouldPrune := pruneMap[step]; !shouldPrune {
			result = append(result, step)
		}
	}
	return result
}

// RegisterCluster registers a cluster configuration with validation
func RegisterCluster(config ClusterConfig) {
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("Invalid cluster %s: %v", config.Name, err))
	}
	ClusterRegistry[config.Name] = config
}

// GetClusters returns all registered clusters
func GetClusters() []ClusterConfig {
	var clusters []ClusterConfig
	for _, cluster := range ClusterRegistry {
		clusters = append(clusters, cluster)
	}
	return clusters
}

// GetClusterByName finds a cluster configuration by name
func GetClusterByName(name ClusterName) (*ClusterConfig, error) {
	if cluster, exists := ClusterRegistry[name]; exists {
		return &cluster, nil
	}
	return nil, fmt.Errorf("cluster not found: %s", name)
}

// GetClustersByEnvironment returns all clusters for a specific environment
func GetClustersByEnvironment(env ClusterEnvironment) []ClusterConfig {
	var clusters []ClusterConfig
	for _, cluster := range ClusterRegistry {
		if cluster.Environment == env {
			clusters = append(clusters, cluster)
		}
	}
	return clusters
}

func NewGatewayConfig(options ...func(*GatewayConfig)) *GatewayConfig {
	g := &GatewayConfig{}
	for _, o := range options {
		o(g)
	}
	return g
}

// WithMetricsEnabled enables metrics functionality for the gateway
func WithMetricsEnabled() func(*GatewayConfig) {
	return func(g *GatewayConfig) {
		g.metricsEnabled = true
	}
}

// WithLoggingEnabled enables logging functionality for the gateway
func WithLoggingEnabled() func(*GatewayConfig) {
	return func(g *GatewayConfig) {
		g.logsEnabled = true
	}
}

// WithSyntheticsEnabled enables synthetics functionality for the gateway
func WithSyntheticsEnabled() func(*GatewayConfig) {
	return func(g *GatewayConfig) {
		g.syntheticsEnabled = true
	}
}

// WithTracingEnabled enables internal tracing for the gateway itself
func WithTracingEnabled() func(*GatewayConfig) {
	return func(g *GatewayConfig) {
		g.tracingEnabled = true
	}
}

// WithAMS configures the Account Management Service URL for the gateway
func WithAMS(url string) func(*GatewayConfig) {
	return func(g *GatewayConfig) {
		g.amsURL = url
	}
}

// WithTenants configures the tenant definitions for multi-tenancy support
func WithTenants(tenants observatoriumapi.Tenants) func(*GatewayConfig) {
	return func(g *GatewayConfig) {
		g.tenants = tenants
	}
}

// WithRBAC configures role-based access control settings for the gateway
func WithRBAC(rbac cfgobservatorium.ObservatoriumRBAC) func(*GatewayConfig) {
	return func(g *GatewayConfig) {
		g.rbac = rbac
	}
}

// Getter methods for GatewayConfig fields

// MetricsEnabled returns whether metrics are enabled for the gateway
func (g *GatewayConfig) MetricsEnabled() bool {
	return g.metricsEnabled
}

// LogsEnabled returns whether logs are enabled for the gateway
func (g *GatewayConfig) LogsEnabled() bool {
	return g.logsEnabled
}

// SyntheticsEnabled returns whether synthetics are enabled for the gateway
func (g *GatewayConfig) SyntheticsEnabled() bool {
	return g.syntheticsEnabled
}

// TracingEnabled returns whether tracing is enabled for the gateway
func (g *GatewayConfig) TracingEnabled() bool {
	return g.tracingEnabled
}

// AMSURL returns the AMS URL for the gateway
func (g *GatewayConfig) AMSURL() string {
	return g.amsURL
}

// Tenants returns the tenants configuration for the gateway
func (g *GatewayConfig) Tenants() observatoriumapi.Tenants {
	return g.tenants
}

// RBAC returns the RBAC configuration for the gateway
func (g *GatewayConfig) RBAC() cfgobservatorium.ObservatoriumRBAC {
	return g.rbac
}
