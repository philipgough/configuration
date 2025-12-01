package main

import (
	"fmt"

	"github.com/bwplotka/mimic"
	"github.com/bwplotka/mimic/encoding"
	"github.com/rhobs/configuration/clusters"
	"k8s.io/apimachinery/pkg/runtime"
)

// Metrics generates all metrics-related resources in a bundled service directory
// This includes: CRDs, Operator, Thanos components, and ServiceMonitors
func (b Build) Metrics(config clusters.ClusterConfig) error {
	gen := b.generator(config, "metrics")

	// Generate CRDs as individual files
	if err := b.generateMetricsCRDs(gen); err != nil {
		return fmt.Errorf("failed to generate metrics CRDs: %w", err)
	}

	// Generate Operator resources as individual files
	if err := b.generateMetricsOperator(gen, config); err != nil {
		return fmt.Errorf("failed to generate metrics operator: %w", err)
	}

	// Generate Thanos stack components as individual files
	if err := b.generateMetricsStack(gen, config); err != nil {
		return fmt.Errorf("failed to generate metrics stack: %w", err)
	}

	// Generate consolidated ServiceMonitors
	if err := b.generateMetricsServiceMonitors(gen, config); err != nil {
		return fmt.Errorf("failed to generate metrics service monitors: %w", err)
	}

	gen.Generate()
	return nil
}

func (b Build) generateMetricsCRDs(gen *mimic.Generator) error {
	// Use existing CRD generation logic but output individual files
	const (
		compact   = "thanoscompacts.yaml"
		queries   = "thanosqueries.yaml"
		receivers = "thanosreceives.yaml"
		rulers    = "thanosrulers.yaml"
		stores    = "thanosstores.yaml"
		base      = "https://raw.githubusercontent.com/thanos-community/thanos-operator/" + thanosOperatorCRDRef + "/config/crd/bases/monitoring.thanos.io_"
	)

	components := []struct {
		file      string
		component string
	}{
		{compact, "thanos-compact"},
		{queries, "thanos-query"},
		{receivers, "thanos-receive"},
		{rulers, "thanos-ruler"},
		{stores, "thanos-store"},
	}

	for _, comp := range components {
		crd, err := getCustomResourceDefinition(base + comp.file)
		if err != nil {
			return err
		}

		// Generate pure Kubernetes resource (no template wrapping)
		fileName := fmt.Sprintf("%s-CustomResourceDefinition.yaml", comp.component)
		gen.Add(fileName, encoding.GhodssYAML(crd))
	}

	return nil
}

func (b Build) generateMetricsOperator(gen *mimic.Generator, config clusters.ClusterConfig) error {
	operatorObjs := operatorResources(config.Namespace, config.Templates)

	// Group resources by type and generate individual files
	resourceGroups := groupResourcesByType(operatorObjs)

	for resourceType, objs := range resourceGroups {
		if len(objs) == 1 {
			// Single resource gets individual file (pure Kubernetes resource)
			obj := objs[0]
			name := getResourceName(obj)
			fileName := fmt.Sprintf("thanos-operator-%s-%s.yaml", name, resourceType)
			gen.Add(fileName, encoding.GhodssYAML(obj))
		} else {
			// Multiple resources of same type get combined file (pure Kubernetes resources)
			fileName := fmt.Sprintf("thanos-operator-%s.yaml", resourceType)
			gen.Add(fileName, encoding.GhodssYAML(objs))
		}
	}

	return nil
}

func (b Build) generateMetricsStack(gen *mimic.Generator, config clusters.ClusterConfig) error {
	// Generate Thanos Query (pure Kubernetes resources)
	queryResourcePairs := defaultQueryCR(config.Namespace, config.Templates, true)
	for _, obj := range queryResourcePairs.GetCoreResources() {
		resourceType := obj.GetObjectKind().GroupVersionKind().Kind
		name := getResourceName(obj)
		fileName := fmt.Sprintf("%s-%s.yaml", name, resourceType)
		gen.Add(fileName, encoding.GhodssYAML(obj))
	}

	// Generate other Thanos components (pure Kubernetes resources)
	components := []struct {
		name string
		obj  runtime.Object
	}{
		{"thanos-receive-default", defaultReceiveCR(config.Namespace, config.Templates)},
		{"thanos-ruler", defaultRulerCR(config.Namespace, config.Templates)},
		{"thanos-store", defaultStoreCR(config.Namespace, config.Templates)},
	}

	// Add compact components (pure Kubernetes resources)
	compactObjs := defaultCompactCR(config.Namespace, config.Templates, true)
	for _, obj := range compactObjs {
		resourceType := obj.GetObjectKind().GroupVersionKind().Kind
		name := getResourceName(obj)
		fileName := fmt.Sprintf("%s-%s.yaml", name, resourceType)
		gen.Add(fileName, encoding.GhodssYAML(obj))
	}

	for _, comp := range components {
		resourceType := comp.obj.GetObjectKind().GroupVersionKind().Kind
		fileName := fmt.Sprintf("%s-%s.yaml", comp.name, resourceType)
		gen.Add(fileName, encoding.GhodssYAML(comp.obj))
	}

	return nil
}

func (b Build) generateMetricsServiceMonitors(gen *mimic.Generator, config clusters.ClusterConfig) error {
	var allServiceMonitors []runtime.Object

	// Get ServiceMonitors from Thanos query
	queryResourcePairs := defaultQueryCR(config.Namespace, config.Templates, true)
	allServiceMonitors = append(allServiceMonitors, queryResourcePairs.GetServiceMonitors()...)

	// Get ServiceMonitors from other components (if any exist)
	allServiceMonitors = append(allServiceMonitors, createThanosServiceMonitors(config.Namespace)...)

	// Generate consolidated ServiceMonitor file (pure Kubernetes resources)
	gen.Add("metrics-ServiceMonitor.yaml", encoding.GhodssYAML(allServiceMonitors))

	return nil
}

// Helper functions
func groupResourcesByType(objs []runtime.Object) map[string][]runtime.Object {
	groups := make(map[string][]runtime.Object)
	
	for _, obj := range objs {
		resourceType := obj.GetObjectKind().GroupVersionKind().Kind
		groups[resourceType] = append(groups[resourceType], obj)
	}
	
	return groups
}

func getResourceName(obj runtime.Object) string {
	if metaObj, ok := obj.(interface{ GetName() string }); ok {
		return metaObj.GetName()
	}
	return "unknown"
}