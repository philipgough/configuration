package main

import (
	"fmt"
	"os"

	"github.com/bwplotka/mimic"
	"github.com/go-kit/log"
	"github.com/magefile/mage/mg"
	"github.com/rhobs/configuration/clusters"
)

type (
	Stage      mg.Namespace
	Production mg.Namespace
	Unified    mg.Namespace

	Build mg.Namespace
	List  mg.Namespace
)

const (
	templatePath         = "resources"
	templateServicesPath = "services"
	templateClustersPath = "clusters"
	templateO11yPath     = "o11y"
)

// BuildStepFunctions maps build step names to their implementation functions
var BuildStepFunctions = map[string]func(Build, clusters.ClusterConfig) error{
	clusters.StepThanosOperatorCRDS: func(b Build, cfg clusters.ClusterConfig) error {
		return b.ThanosOperatorCRDS(cfg)
	},
	clusters.StepThanosOperator: func(b Build, cfg clusters.ClusterConfig) error {
		b.ThanosOperator(cfg)
		return nil
	},
	clusters.StepDefaultThanosStack: func(b Build, cfg clusters.ClusterConfig) error {
		b.DefaultThanosStack(cfg)
		return nil
	},
	clusters.StepLokiOperatorCRDS: func(b Build, cfg clusters.ClusterConfig) error {
		return b.LokiOperatorCRDS(cfg)
	},
	clusters.StepLokiOperator: func(b Build, cfg clusters.ClusterConfig) error {
		b.LokiOperator(cfg)
		return nil
	},
	clusters.StepDefaultLokiStack: func(b Build, cfg clusters.ClusterConfig) error {
		b.DefaultLokiStack(cfg)
		return nil
	},
	clusters.StepServiceMonitors: func(b Build, cfg clusters.ClusterConfig) error {
		b.ServiceMonitors(cfg)
		return nil
	},
	clusters.StepAlertmanager: func(b Build, cfg clusters.ClusterConfig) error {
		b.Alertmanager(cfg)
		return nil
	},
	clusters.StepSecrets: func(b Build, cfg clusters.ClusterConfig) error {
		b.Secrets(cfg)
		return nil
	},
	clusters.StepMemcached: func(b Build, cfg clusters.ClusterConfig) error {
		b.Cache(cfg)
		return nil
	},
	clusters.StepSyntheticsApi: func(b Build, cfg clusters.ClusterConfig) error {
		b.SyntheticsApi(cfg)
		return nil
	},
	clusters.StepGateway: func(b Build, cfg clusters.ClusterConfig) error {
		err := b.Gateway(cfg)
		if err != nil {
			return err
		}
		return nil
	},
}

// ExecuteSteps executes a list of build steps for a cluster
func (b Build) executeSteps(steps []string, cfg clusters.ClusterConfig) error {
	for _, step := range steps {
		if fn, exists := BuildStepFunctions[step]; exists {
			if err := fn(b, cfg); err != nil {
				return fmt.Errorf("build step '%s' failed for cluster %s: %w", step, cfg.Name, err)
			}
		} else {
			return fmt.Errorf("unknown build step '%s' for cluster %s", step, cfg.Name)
		}
	}
	return nil
}

// Clusters Builds manifests for all registered clusters
func (b Build) Clusters() error {
	clusterConfigs := clusters.GetClusters()
	if len(clusterConfigs) == 0 {
		return fmt.Errorf("no clusters registered")
	}

	for _, cfg := range clusterConfigs {
		if err := b.executeSteps(cfg.BuildSteps, cfg); err != nil {
			return err
		}
	}
	
	// Generate all monitoring bundles after all build steps complete
	return GenerateAllMonitoringBundles()
}

// Cluster Builds manifests for a specific cluster
func (b Build) Cluster(clusterName string) error {
	cluster, err := clusters.GetClusterByName(clusters.ClusterName(clusterName))
	if err != nil {
		return err
	}

	if err := b.executeSteps(cluster.BuildSteps, *cluster); err != nil {
		return err
	}
	
	// Generate monitoring bundle after build steps complete
	return GenerateAllMonitoringBundles()
}

// Environment Builds manifests for all clusters in a specific environment
func (b Build) Environment(environment string) error {
	env := clusters.ClusterEnvironment(environment)
	if !env.IsValid() {
		return fmt.Errorf("invalid environment: %s", environment)
	}

	clusterConfigs := clusters.GetClustersByEnvironment(env)
	if len(clusterConfigs) == 0 {
		return fmt.Errorf("no clusters found for environment: %s", environment)
	}

	for _, cfg := range clusterConfigs {
		if err := b.executeSteps(cfg.BuildSteps, cfg); err != nil {
			return err
		}
	}
	
	// Generate all monitoring bundles after all build steps complete
	return GenerateAllMonitoringBundles()
}

// Steps Shows all available build steps
func (l List) Steps() {
	fmt.Fprintln(os.Stdout, "Available build steps:")
	for step := range BuildStepFunctions {
		fmt.Fprintf(os.Stdout, "  - %s\n", step)
	}
}

// Clusters lists all registered clusters and their build steps
func (l List) Clusters() {
	clusterConfigs := clusters.GetClusters()
	if len(clusterConfigs) == 0 {
		fmt.Fprintln(os.Stdout, "No clusters registered")
		return
	}

	for _, cluster := range clusterConfigs {
		fmt.Fprintf(os.Stdout, "Cluster: %s (%s)\n", cluster.Name, cluster.Environment)
		fmt.Fprintf(os.Stdout, "  Steps: %v\n", cluster.BuildSteps)
	}
}

// Build Builds the manifests for the stage environment.
func (Stage) Build() {
	mg.SerialDeps(Stage.Alertmanager, Stage.CRDS, Stage.Operator, Stage.Thanos, Stage.ServiceMonitors, Stage.Secrets)
}

func (Build) generator(config clusters.ClusterConfig, component string) *mimic.Generator {
	gen := &mimic.Generator{}
	gen = gen.With(templatePath, templateClustersPath, string(config.Environment), string(config.Name), component)
	gen.Logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	return gen
}

func (Build) o11yGenerator(component string) *mimic.Generator {
	gen := &mimic.Generator{}
	gen = gen.With(templatePath, templateO11yPath, component)
	gen.Logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	return gen
}

func (Stage) generator(component string) *mimic.Generator {
	gen := &mimic.Generator{}
	gen = gen.With(templatePath, templateServicesPath, component, "staging")
	gen.Logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	return gen
}

func (Production) generator(component string) *mimic.Generator {
	gen := &mimic.Generator{}
	gen = gen.With(templatePath, templateServicesPath, component, "production")
	gen.Logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	return gen
}

func (Unified) generator(component string) *mimic.Generator {
	gen := &mimic.Generator{}
	gen = gen.With(templatePath, templateServicesPath)
	gen.Logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	return gen
}

const (
	stageNamespace = "rhobs-stage"
	prodNamespace  = "rhobs-production"
)

func (Stage) namespace() string {
	return stageNamespace
}

func (Production) namespace() string {
	return prodNamespace
}

// Build Builds the manifests for the production environment.
func (Production) Build() {
	mg.Deps(Production.Alertmanager)
}

// SyntheticsApi generates a single, environment-agnostic synthetics-api template
func (Unified) SyntheticsApi() {
	generateUnifiedSyntheticsApi()
}

// All generates all available unified templates
func (Unified) All() {
	mg.Deps(Unified.SyntheticsApi)
}

// List shows all available unified template targets
func (Unified) List() {
	fmt.Fprintln(os.Stdout, "Available unified template targets:")
	fmt.Fprintln(os.Stdout, "  unified:syntheticsApi - Generate synthetics-api template")
	fmt.Fprintln(os.Stdout, "  unified:all           - Generate all unified templates")
}
