package main

import (
	"fmt"
	"os"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/encoding"
	"github.com/go-kit/log"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/configuration/clusters"
	"k8s.io/apimachinery/pkg/runtime"
)

// MonitoringBundle collects ServiceMonitors from all build steps
type MonitoringBundle struct {
	config          clusters.ClusterConfig
	serviceMonitors []runtime.Object
}

// NewMonitoringBundle creates a new monitoring bundle for a cluster
func NewMonitoringBundle(config clusters.ClusterConfig) *MonitoringBundle {
	return &MonitoringBundle{
		config:          config,
		serviceMonitors: make([]runtime.Object, 0),
	}
}

// AddServiceMonitor adds a ServiceMonitor to the monitoring bundle
func (mb *MonitoringBundle) AddServiceMonitor(sm *monv1.ServiceMonitor) {
	if sm != nil {
		mb.serviceMonitors = append(mb.serviceMonitors, sm)
	}
}

// Generate creates the monitoring bundle files
func (mb *MonitoringBundle) Generate() error {
	if len(mb.serviceMonitors) == 0 {
		return nil // No ServiceMonitors to generate
	}

	gen := &mimic.Generator{}
	gen = gen.With("resources", "clusters", string(mb.config.Environment), string(mb.config.Name), "monitoring")
	gen.Logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))

	// Generate individual ServiceMonitor files as pure Kubernetes resources
	for i, sm := range mb.serviceMonitors {
		if sm == nil {
			continue // Skip nil ServiceMonitors
		}

		if smObj, ok := sm.(*monv1.ServiceMonitor); ok && smObj != nil {
			// Process ServiceMonitor to ensure it has proper values (no template variables)
			processedSM := mb.processServiceMonitorForBundle(smObj)

			name := processedSM.Name
			if name == "" {
				name = fmt.Sprintf("unnamed-%d", i)
			}
			fileName := fmt.Sprintf("%s-ServiceMonitor.yaml", name)
			// Generate pure Kubernetes resource (no template wrapping)
			gen.Add(fileName, encoding.GhodssYAML(processedSM))
		} else {
			// Fallback for non-ServiceMonitor objects
			fileName := fmt.Sprintf("service-monitor-%d.yaml", i)
			gen.Add(fileName, encoding.GhodssYAML(sm))
		}
	}

	gen.Generate()
	return nil
}

// processServiceMonitorForBundle processes a ServiceMonitor to ensure it's a pure Kubernetes resource
func (mb *MonitoringBundle) processServiceMonitorForBundle(sm *monv1.ServiceMonitor) *monv1.ServiceMonitor {
	// Create a copy to avoid modifying the original
	processed := sm.DeepCopy()

	// Ensure namespace is set to the monitoring namespace (not template variable)
	processed.Namespace = "openshift-customer-monitoring"

	// Ensure NamespaceSelector points to the actual cluster namespace (not template variable)
	if processed.Spec.NamespaceSelector.MatchNames != nil {
		for i, ns := range processed.Spec.NamespaceSelector.MatchNames {
			// Replace template variables with actual namespace
			if ns == "${NAMESPACE}" || ns == "" {
				processed.Spec.NamespaceSelector.MatchNames[i] = mb.config.Namespace
			}
		}
	} else {
		// Set namespace selector if not present
		processed.Spec.NamespaceSelector.MatchNames = []string{mb.config.Namespace}
	}

	// Add required prometheus label if not present
	if processed.Labels == nil {
		processed.Labels = make(map[string]string)
	}
	processed.Labels["prometheus"] = "app-sre"

	return processed
}

// Global monitoring bundle registry per cluster
var monitoringBundles = make(map[string]*MonitoringBundle)

// GetMonitoringBundle gets or creates a monitoring bundle for a cluster
func GetMonitoringBundle(config clusters.ClusterConfig) *MonitoringBundle {
	key := fmt.Sprintf("%s-%s", config.Environment, config.Name)
	if bundle, exists := monitoringBundles[key]; exists {
		return bundle
	}

	bundle := NewMonitoringBundle(config)
	monitoringBundles[key] = bundle
	return bundle
}

// GenerateAllMonitoringBundles generates all monitoring bundles that have been collected
func GenerateAllMonitoringBundles() error {
	for key, bundle := range monitoringBundles {
		if bundle == nil {
			continue
		}
		if err := bundle.Generate(); err != nil {
			return fmt.Errorf("failed to generate monitoring bundle for %s (%s): %w",
				key, bundle.config.Name, err)
		}
	}
	// Clear the registry after generation
	monitoringBundles = make(map[string]*MonitoringBundle)
	return nil
}
