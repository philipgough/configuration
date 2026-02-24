package clusters

import (
	"fmt"

	"github.com/thanos-community/thanos-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

// ParamMap is a map of parameters to config values.
type ParamMap[T any] map[string]T

// TemplateMaps is a map of parameters to config value maps of different types.
type TemplateMaps struct {
	Images               ParamMap[string]
	Versions             ParamMap[string]
	LogLevels            ParamMap[string]
	StorageSize          ParamMap[v1alpha1.StorageSize]
	Replicas             ParamMap[int32]
	ResourceRequirements ParamMap[corev1.ResourceRequirements]
	ObjectStorageBucket  ParamMap[v1alpha1.ObjectStorageConfig]
	LokiOverrides        ParamMap[LokiOverrides]
}

type LokiOverrides struct {
	LokiLimitOverrides
	Router        LokiComponentSpec
	Ingest        LokiComponentSpec
	Query         LokiComponentSpec
	QueryFrontend LokiComponentSpec
}

type LokiLimitOverrides struct {
	IngestionRateLimitMB int32
	IngestionBurstSizeMB int32
	MaxLineSize          int32

	PerStreamRateLimitMB int32
	PerStreamBurstSizeMB int32

	QueryTimeout string
}

type LokiComponentSpec struct {
	Replicas int32
}

// Override applies overrides to a TemplateMaps and returns a new instance
func (t TemplateMaps) Override(overrides ...TemplateOverride) TemplateMaps {
	result := t
	for _, override := range overrides {
		result = override.Apply(result)
	}
	return result
}

// TemplateOverride represents a set of overrides to apply to a TemplateMaps
type TemplateOverride interface {
	Apply(TemplateMaps) TemplateMaps
}

// Images override
type Images map[string]string

func (i Images) Apply(t TemplateMaps) TemplateMaps {
	if t.Images == nil {
		t.Images = make(ParamMap[string])
	}
	for k, v := range i {
		t.Images[k] = v
	}
	return t
}

// Replicas override
type Replicas map[string]int32

func (r Replicas) Apply(t TemplateMaps) TemplateMaps {
	if t.Replicas == nil {
		t.Replicas = make(ParamMap[int32])
	}
	for k, v := range r {
		t.Replicas[k] = v
	}
	return t
}

// StorageSizes override
type StorageSizes map[string]v1alpha1.StorageSize

func (s StorageSizes) Apply(t TemplateMaps) TemplateMaps {
	if t.StorageSize == nil {
		t.StorageSize = make(ParamMap[v1alpha1.StorageSize])
	}
	for k, v := range s {
		t.StorageSize[k] = v
	}
	return t
}

// Resources override
type Resources map[string]corev1.ResourceRequirements

func (r Resources) Apply(t TemplateMaps) TemplateMaps {
	if t.ResourceRequirements == nil {
		t.ResourceRequirements = make(ParamMap[corev1.ResourceRequirements])
	}
	for k, v := range r {
		t.ResourceRequirements[k] = v
	}
	return t
}

// LogLevels override
type LogLevels map[string]string

func (l LogLevels) Apply(t TemplateMaps) TemplateMaps {
	if t.LogLevels == nil {
		t.LogLevels = make(ParamMap[string])
	}
	for k, v := range l {
		t.LogLevels[k] = v
	}
	return t
}

// Versions override
type Versions map[string]string

func (v Versions) Apply(t TemplateMaps) TemplateMaps {
	if t.Versions == nil {
		t.Versions = make(ParamMap[string])
	}
	for k, val := range v {
		t.Versions[k] = val
	}
	return t
}

// LokiOverridesMap override
type LokiOverridesMap map[string]LokiOverrides

func (l LokiOverridesMap) Apply(t TemplateMaps) TemplateMaps {
	if t.LokiOverrides == nil {
		t.LokiOverrides = make(ParamMap[LokiOverrides])
	}
	for k, v := range l {
		// Get existing config or create empty one
		existing := t.LokiOverrides[k]

		// Merge the override with existing values
		merged := LokiOverrides{
			LokiLimitOverrides: mergeLokiLimitOverrides(existing.LokiLimitOverrides, v.LokiLimitOverrides),
			Router:             mergeComponentSpec(existing.Router, v.Router),
			Ingest:             mergeComponentSpec(existing.Ingest, v.Ingest),
			Query:              mergeComponentSpec(existing.Query, v.Query),
			QueryFrontend:      mergeComponentSpec(existing.QueryFrontend, v.QueryFrontend),
		}

		t.LokiOverrides[k] = merged
	}
	return t
}

// mergeComponentSpec merges two LokiComponentSpec, using override values when non-zero
func mergeComponentSpec(existing, override LokiComponentSpec) LokiComponentSpec {
	result := existing
	if override.Replicas != 0 {
		result.Replicas = override.Replicas
	}
	return result
}

// mergeLokiLimitOverrides merges two LokiLimitOverrides, using override values when non-zero/non-empty
func mergeLokiLimitOverrides(existing, override LokiLimitOverrides) LokiLimitOverrides {
	result := existing
	if override.IngestionRateLimitMB != 0 {
		result.IngestionRateLimitMB = override.IngestionRateLimitMB
	}
	if override.IngestionBurstSizeMB != 0 {
		result.IngestionBurstSizeMB = override.IngestionBurstSizeMB
	}
	if override.PerStreamRateLimitMB != 0 {
		result.PerStreamRateLimitMB = override.PerStreamRateLimitMB
	}
	if override.PerStreamBurstSizeMB != 0 {
		result.PerStreamBurstSizeMB = override.PerStreamBurstSizeMB
	}
	if override.QueryTimeout != "" {
		result.QueryTimeout = override.QueryTimeout
	}
	return result
}

// TemplateFn is a function that returns a value from a map.
// It panics if the param is not found in the map.
// It returns the value of the param.
func TemplateFn[T any](param string, m ParamMap[T]) T {
	v, ok := m[param]
	// TODO moadz: We should surface the error, so that we can track the build chain for debug
	if !ok {
		panic(fmt.Sprintf("param %s not found %v", param, m))
	}
	return v
}

const (
	thanosImage        = "quay.io/redhat-services-prod/rhobs-mco-tenant/rhobs-thanos"
	thanosVersionStage = "0dcacf2a5108328d537dbcf59fcabc9b69f08a2b"
	thanosVersionProd  = "0dcacf2a5108328d537dbcf59fcabc9b69f08a2b"

	ThanosOperatorImage = "quay.io/redhat-services-prod/rhobs-mco-tenant/rhobs-thanos-operator"
	// ThanosOperatorVersion is the image version (git sha) of the Thanos Operator and is matched to a commit in
	// the rhobs-konflux-thanos-operator repository.
	// This is generated by mage sync:operator and should not be modified manually in normal circumstances.
	// To hot patch for individual environments, update the corresponding thanosOperatorVersion{Env} constant below.
	ThanosOperatorVersion = "4b2242f42d204be12ab3350a7b3bbb02e0be399e"

	thanosOperatorVersionStage = ThanosOperatorVersion
	thanosOperatorVersionProd  = ThanosOperatorVersion
)

const (
	memcachedTag           = "1.5-316"
	memcachedImage         = "registry.redhat.io/rhel8/memcached" + ":" + memcachedTag
	memcachedExporterImage = "quay.io/prometheus/memcached-exporter:v0.15.0"
)

const (
	syntheticsApiImage        = "quay.io/redhat-services-prod/openshift/rhobs-synthetics-api"
	syntheticsApiVersionStage = "84bd137"
	syntheticsApiVersionProd  = "cea7d4656cd0ad338e580cc6ba266264a9938e5c"

	syntheticsAgentImage        = "quay.io/redhat-services-prod/openshift/rhobs-synthetics-agent"
	syntheticsAgentVersionStage = "3012046"
	syntheticsAgentVersionProd  = "3012046"
)

const (
	ObservatoriumImage   = "quay.io/redhat-services-prod/rhobs-mco-tenant/rhobs-observatorium-api"
	ObservatoriumVersion = "1fb4dafb58f6158c832f307d8e37729f390f4f5a"
)

// Template key constants - exportable template parameter names
const (
	// Service and component keys
	MemcachedExporter = "MEMCACHED_EXPORTER"
	ApiCache          = "API_CACHE"
	Jaeger            = "JAEGER_AGENT"
	ObservatoriumAPI  = "OBSERVATORIUM_API"
	OpaAMS            = "OPA_AMS"
	SyntheticsAPI     = "SYNTHETICS_API"
	SyntheticsAgent   = "SYNTHETICS_AGENT"

	// Thanos component keys
	ThanosOperator         = "THANOS_OPERATOR"
	KubeRbacProxy          = "KUBE_RBAC_PROXY"
	StoreDefault           = "STORE_DEFAULT"
	ReceiveRouter          = "RECEIVE_ROUTER"
	ReceiveIngestorDefault = "RECEIVE_INGESTOR_DEFAULT"
	Ruler                  = "RULER"
	CompactDefault         = "COMPACT_DEFAULT"
	Query                  = "QUERY"
	QueryFrontend          = "QUERY_FRONTEND"
	Manager                = "MANAGER"

	LokiConfig = "LOKI_CONFIG"

	// Object storage keys
	DefaultBucket = "DEFAULT_BUCKET"
)

var logLevels = []string{"debug", "info", "warn", "error"}

// DefaultBaseTemplate Base default template for all instances
func DefaultBaseTemplate() TemplateMaps {
	return TemplateMaps{
		Images: ParamMap[string]{
			ThanosOperator:         fmt.Sprintf("%s:%s", ThanosOperatorImage, thanosOperatorVersionStage),
			KubeRbacProxy:          "registry.redhat.io/openshift4/ose-kube-rbac-proxy@sha256:98455d503b797b6b02edcfd37045c8fab0796b95ee5cf4cfe73b221a07e805f0",
			StoreDefault:           thanosImage,
			ReceiveRouter:          thanosImage,
			ReceiveIngestorDefault: thanosImage,
			Ruler:                  thanosImage,
			CompactDefault:         thanosImage,
			Query:                  thanosImage,
			QueryFrontend:          thanosImage,
			Jaeger:                 "registry.redhat.io/rhosdt/jaeger-agent-rhel8:1.57.0-10",
			ApiCache:               memcachedImage,
			MemcachedExporter:      memcachedExporterImage,
			ObservatoriumAPI:       fmt.Sprintf("%s:%s", ObservatoriumImage, ObservatoriumVersion),
			SyntheticsAPI:          syntheticsApiImage,
			SyntheticsAgent:        syntheticsAgentImage,
		},
		Versions: ParamMap[string]{
			StoreDefault:           thanosVersionProd,
			ReceiveRouter:          thanosVersionProd,
			ReceiveIngestorDefault: thanosVersionProd,
			Ruler:                  thanosVersionProd,
			CompactDefault:         thanosVersionProd,
			Query:                  thanosVersionProd,
			QueryFrontend:          thanosVersionProd,
			ApiCache:               memcachedTag,
			SyntheticsAPI:          syntheticsApiVersionProd,
			SyntheticsAgent:        syntheticsAgentVersionProd,
			ObservatoriumAPI:       ObservatoriumVersion,
		},
		LogLevels: ParamMap[string]{
			StoreDefault:           logLevels[1],
			ReceiveRouter:          logLevels[1],
			ReceiveIngestorDefault: logLevels[1],
			Ruler:                  logLevels[1],
			CompactDefault:         logLevels[1],
			Query:                  logLevels[1],
			QueryFrontend:          logLevels[1],
			SyntheticsAPI:          logLevels[1],
			SyntheticsAgent:        logLevels[1],
			ObservatoriumAPI:       logLevels[1],
		},
		ResourceRequirements: ParamMap[corev1.ResourceRequirements]{
			StoreDefault: {
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("20Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			ReceiveRouter: {
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("5Gi"),
				},
			},
			ReceiveIngestorDefault: {
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("15Gi"),
				},
			},
			Ruler: {
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("5Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			CompactDefault: {
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("20Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			Query: {
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("20Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			QueryFrontend: {
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("20Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			Manager: {
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("600m"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
			},
			KubeRbacProxy: {
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("25m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
			},
		},
		Replicas: ParamMap[int32]{
			StoreDefault:           3,
			ReceiveRouter:          3,
			ReceiveIngestorDefault: 3,
			Ruler:                  1,
			Query:                  3,
			QueryFrontend:          3,
			CompactDefault:         1,
			ObservatoriumAPI:       2,
		},
		StorageSize: ParamMap[v1alpha1.StorageSize]{
			StoreDefault:           "10Gi",
			ReceiveIngestorDefault: "10Gi",
			CompactDefault:         "50Gi",
			Ruler:                  "10Gi",
		},
		ObjectStorageBucket: ParamMap[v1alpha1.ObjectStorageConfig]{
			DefaultBucket: v1alpha1.ObjectStorageConfig{
				Key: "thanos.yaml",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "default-thanos-bucket",
				},
				Optional: ptr.To(false),
			},
		},
		LokiOverrides: ParamMap[LokiOverrides]{
			LokiConfig: LokiOverrides{
				LokiLimitOverrides: LokiLimitOverrides{
					IngestionRateLimitMB: 12,
					IngestionBurstSizeMB: 256, // workaround for LOG-6817, OTLP configuration appears to be sending unbounded-large requests
					MaxLineSize:          2097152,
					PerStreamRateLimitMB: 5,
					PerStreamBurstSizeMB: 15,
					QueryTimeout:         "3m",
				},
				Router: LokiComponentSpec{
					Replicas: 3,
				},
				Ingest: LokiComponentSpec{
					Replicas: 3,
				},
				Query: LokiComponentSpec{
					Replicas: 2,
				},
				QueryFrontend: LokiComponentSpec{
					Replicas: 2,
				},
			},
		},
	}
}

// Stage images.
var StageImages = ParamMap[string]{
	"STORE02W":                   thanosImage,
	"STORE2W90D":                 thanosImage,
	"STORE90D+":                  thanosImage,
	"STORE_ROS":                  thanosImage,
	"STORE_DEFAULT":              thanosImage,
	"RECEIVE_ROUTER":             thanosImage,
	"RECEIVE_INGESTOR_TELEMETER": thanosImage,
	"RECEIVE_INGESTOR_DEFAULT":   thanosImage,
	"RULER":                      thanosImage,
	"COMPACT_DEFAULT":            thanosImage,
	"COMPACT_ROS":                thanosImage,
	"COMPACT_TELEMETER":          thanosImage,
	"QUERY":                      thanosImage,
	"QUERY_FRONTEND":             thanosImage,
	Jaeger:                       "registry.redhat.io/rhosdt/jaeger-agent-rhel8:1.57.0-10",
	"THANOS_OPERATOR":            fmt.Sprintf("%s:%s", ThanosOperatorImage, thanosOperatorVersionStage),
	"KUBE_RBAC_PROXY":            "registry.redhat.io/openshift4/ose-kube-rbac-proxy@sha256:98455d503b797b6b02edcfd37045c8fab0796b95ee5cf4cfe73b221a07e805f0",
	ApiCache:                     memcachedImage,
	MemcachedExporter:            memcachedExporterImage,
	ObservatoriumAPI:             fmt.Sprintf("%s:%s", ObservatoriumImage, ObservatoriumVersion),
	SyntheticsAPI:                fmt.Sprintf("%s:%s", syntheticsApiImage, syntheticsApiVersionStage),
	SyntheticsAgent:              fmt.Sprintf("%s:%s", syntheticsAgentImage, syntheticsAgentVersionStage),
	OpaAMS:                       "quay.io/redhat-services-prod/rhobs-mco-tenant/rhobs-opa-ams:da95dce02a3be7dade09199ce5a9d91a5ac6a878",
}

// Stage images.
var StageVersions = ParamMap[string]{
	"STORE02W":                   thanosVersionStage,
	"STORE2W90D":                 thanosVersionStage,
	"STORE90D+":                  thanosVersionStage,
	"STORE_ROS":                  thanosVersionStage,
	"STORE_DEFAULT":              thanosVersionStage,
	"RECEIVE_ROUTER":             thanosVersionStage,
	"RECEIVE_INGESTOR_TELEMETER": thanosVersionStage,
	"RECEIVE_INGESTOR_DEFAULT":   thanosVersionStage,
	"RULER":                      thanosVersionStage,
	"COMPACT_DEFAULT":            thanosVersionStage,
	"COMPACT_ROS":                thanosVersionStage,
	"COMPACT_TELEMETER":          thanosVersionStage,
	"QUERY":                      thanosVersionStage,
	"QUERY_FRONTEND":             thanosVersionStage,
	ApiCache:                     memcachedTag,
	ObservatoriumAPI:             "9aada65247a07782465beb500323a0e18d7e3d05",
	SyntheticsAPI:                syntheticsApiVersionStage,
	SyntheticsAgent:              syntheticsAgentVersionStage,
}

// Stage log levels.
var StageLogLevels = ParamMap[string]{
	"STORE02W":                   logLevels[1],
	"STORE2W90D":                 logLevels[1],
	"STORE90D+":                  logLevels[1],
	"STORE_ROS":                  logLevels[1],
	"STORE_DEFAULT":              logLevels[1],
	"RECEIVE_ROUTER":             logLevels[1],
	"RECEIVE_INGESTOR_TELEMETER": logLevels[1],
	"RECEIVE_INGESTOR_DEFAULT":   logLevels[1],
	"RULER":                      logLevels[1],
	"COMPACT_DEFAULT":            logLevels[1],
	"COMPACT_ROS":                logLevels[1],
	"COMPACT_TELEMETER":          logLevels[1],
	"QUERY":                      logLevels[1],
	"QUERY_FRONTEND":             logLevels[1],
	ObservatoriumAPI:             logLevels[0],
	SyntheticsAPI:                logLevels[0],
	SyntheticsAgent:              logLevels[0],
}

// Stage PV storage sizes.
var StageStorageSize = ParamMap[v1alpha1.StorageSize]{
	"STORE02W":          "512Mi",
	"STORE2W90D":        "512Mi",
	"STORE90D+":         "512Mi",
	"STORE_ROS":         "512Mi",
	"STORE_DEFAULT":     "3Gi",
	"RECEIVE_TELEMETER": "3Gi",
	"RECEIVE_DEFAULT":   "3Gi",
	"RULER":             "512Mi",
	"COMPACT_DEFAULT":   "512Mi",
	"COMPACT_ROS":       "512Mi",
	"COMPACT_TELEMETER": "512Mi",
}

// Stage replicas.
var StageReplicas = ParamMap[int32]{
	"STORE02W":                   3,
	"STORE2W90D":                 3,
	"STORE90D+":                  3,
	"STORE_ROS":                  3,
	"STORE_DEFAULT":              3,
	"RECEIVE_ROUTER":             3,
	"RECEIVE_INGESTOR_TELEMETER": 6,
	"RECEIVE_INGESTOR_DEFAULT":   3,
	"RULER":                      2,
	"QUERY":                      6,
	"QUERY_FRONTEND":             3,
	ApiCache:                     1,
	ObservatoriumAPI:             2,
	SyntheticsAPI:                2,
}

// Stage resource requirements.
var StageResourceRequirements = ParamMap[corev1.ResourceRequirements]{
	"STORE02W": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	},
	"STORE2W90D": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	},
	"STORE90D+": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	},
	"STORE_ROS": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	},
	"STORE_DEFAULT": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	},
	"RECEIVE_ROUTER": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("700m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	},
	"RECEIVE_INGESTOR_TELEMETER": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("700m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	},
	"RECEIVE_INGESTOR_DEFAULT": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("700m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	},
	"RULER": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("700m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("3Gi"),
		},
	},
	"COMPACT_DEFAULT": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	},
	"COMPACT_ROS": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	},
	"COMPACT_TELEMETER": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	},
	"QUERY": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("300m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	},
	"QUERY_FRONTEND": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("500Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("3Gi"),
		},
	},
	"MANAGER": corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	},
	"KUBE_RBAC_PROXY": corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("5m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		},
	},
	ApiCache: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("3"),
			corev1.ResourceMemory: resource.MustParse("1844Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
	},
	MemcachedExporter: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
	},
	ObservatoriumAPI: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
	},
	SyntheticsAPI: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
	},
	SyntheticsAgent: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("50Mi"),
		},
	},
}

// Stage object storage bucket.
var StageObjectStorageBucket = ParamMap[v1alpha1.ObjectStorageConfig]{
	"DEFAULT": v1alpha1.ObjectStorageConfig{
		Key: "thanos.yaml",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "observatorium-mst-thanos-objectstorage",
		},
		Optional: ptr.To(false),
	},
	"TELEMETER": v1alpha1.ObjectStorageConfig{
		Key: "thanos.yaml",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "thanos-objectstorage",
		},
		Optional: ptr.To(false),
	},
	"ROS": v1alpha1.ObjectStorageConfig{
		Key: "thanos.yaml",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "ros-thanos-objstore",
		},
		Optional: ptr.To(false),
	},
}

// ProductionImages is a map of production images.
var ProductionImages = ParamMap[string]{
	"STORE02W":                 thanosImage,
	"STORE2W90D":               thanosImage,
	"STORE90D+":                thanosImage,
	"STORE_ROS":                thanosImage,
	"STORE_DEFAULT":            thanosImage,
	"QUERY":                    thanosImage,
	"QUERY_FRONTEND":           thanosImage,
	"RECEIVE_ROUTER":           thanosImage,
	"RECEIVE_INGESTOR_DEFAULT": thanosImage,
	"RULER":                    thanosImage,
	"COMPACT":                  thanosImage,
	"THANOS_OPERATOR":          fmt.Sprintf("%s:%s", ThanosOperatorImage, thanosOperatorVersionProd),
	"KUBE_RBAC_PROXY":          "registry.redhat.io/openshift4/ose-kube-rbac-proxy@sha256:98455d503b797b6b02edcfd37045c8fab0796b95ee5cf4cfe73b221a07e805f0",
	ApiCache:                   memcachedImage,
	MemcachedExporter:          memcachedExporterImage,
	ObservatoriumAPI:           fmt.Sprintf("%s:%s", ObservatoriumImage, ObservatoriumVersion),
	SyntheticsAPI:              fmt.Sprintf("%s:%s", syntheticsApiImage, syntheticsApiVersionProd),
	SyntheticsAgent:            fmt.Sprintf("%s:%s", syntheticsAgentImage, syntheticsAgentVersionProd),
	Jaeger:                     "registry.redhat.io/rhosdt/jaeger-agent-rhel8:1.57.0-10",
}

// ProductionVersions is a map of production versions.
var ProductionVersions = ParamMap[string]{
	"STORE02W":                 thanosVersionProd,
	"STORE2W90D":               thanosVersionProd,
	"STORE90D+":                thanosVersionProd,
	"STORE_ROS":                thanosVersionProd,
	"STORE_DEFAULT":            thanosVersionProd,
	"RECEIVE_ROUTER":           thanosVersionProd,
	"RECEIVE_INGESTOR_DEFAULT": thanosVersionProd,
	"QUERY":                    thanosVersionProd,
	"QUERY_FRONTEND":           thanosVersionProd,
	"RULER":                    thanosVersionProd,
	ApiCache:                   memcachedTag,
	ObservatoriumAPI:           "9aada65247a07782465beb500323a0e18d7e3d05",
	SyntheticsAPI:              syntheticsApiVersionProd,
	SyntheticsAgent:            syntheticsAgentVersionProd,
}

// ProductionLogLevels is a map of production log levels.
var ProductionLogLevels = ParamMap[string]{
	"STORE02W":                 logLevels[0],
	"STORE2W90D":               logLevels[0],
	"STORE90D+":                logLevels[0],
	"STORE_ROS":                logLevels[0],
	"STORE_DEFAULT":            logLevels[0],
	"RECEIVE_ROUTER":           logLevels[0],
	"RECEIVE_INGESTOR_DEFAULT": logLevels[0],
	"QUERY":                    logLevels[0],
	"QUERY_FRONTEND":           logLevels[0],
	"RULER":                    logLevels[0],
	ObservatoriumAPI:           logLevels[0],
	SyntheticsAPI:              logLevels[0],
	SyntheticsAgent:            logLevels[0],
}

// ProductionStorageSize is a map of production PV storage sizes.
var ProductionStorageSize = ParamMap[v1alpha1.StorageSize]{
	"STORE02W":        "300Gi",
	"STORE2W90D":      "300Gi",
	"STORE90D+":       "300Gi",
	"STORE_ROS":       "300Gi",
	"STORE_DEFAULT":   "300Gi",
	"RECEIVE_DEFAULT": "3Gi",
	"RULER":           "3Gi",
}

// ProductionReplicas is a map of production replicas.
var ProductionReplicas = ParamMap[int32]{
	"STORE02W":                 2,
	"STORE2W90D":               2,
	"STORE90D+":                1,
	"STORE_ROS":                0, //TODO @moadz RHOBS-904: Temporary stage-only configuration for ROS disabled in Production.
	"STORE_DEFAULT":            2,
	"RECEIVE_ROUTER":           2,
	"RECEIVE_INGESTOR_DEFAULT": 3,
	"QUERY":                    3,
	"QUERY_FRONTEND":           3,
	"RULER":                    1,
	ApiCache:                   1,
	ObservatoriumAPI:           2,
	SyntheticsAPI:              2,
}

// ProductionResourceRequirements is a map of production resource requirements.
var ProductionResourceRequirements = ParamMap[corev1.ResourceRequirements]{
	"STORE02W": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	},
	"STORE2W90D": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	},
	"STORE90D+": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	},
	"STORE_ROS": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	},
	"STORE_DEFAULT": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	},
	"QUERY": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("300m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	},
	"QUERY_FRONTEND": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("500Mi"),
		},
	},
	"RECEIVE_ROUTER": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("700m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	},
	"RECEIVE_INGESTOR_DEFAULT": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("700m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("5Gi"),
		},
	},
	"RULER": corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("300m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	},
	"MANAGER": corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	},
	"KUBE_RBAC_PROXY": corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("5m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		},
	},
	ApiCache: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("3"),
			corev1.ResourceMemory: resource.MustParse("1844Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
	},
	MemcachedExporter: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
	},
	ObservatoriumAPI: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
	},
	SyntheticsAPI: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
	},
	SyntheticsAgent: corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("50Mi"),
		},
	},
}

// ProductionObjectStorageBucket is a map of production object storage buckets.
var ProductionObjectStorageBucket = ParamMap[v1alpha1.ObjectStorageConfig]{
	"DEFAULT": v1alpha1.ObjectStorageConfig{
		Key: "thanos.yaml",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "observatorium-mst-thanos-objectstorage",
		},
		Optional: ptr.To(false),
	},
	"TELEMETER": v1alpha1.ObjectStorageConfig{
		Key: "thanos.yaml",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "thanos-objectstorage",
		},
		Optional: ptr.To(false),
	},
	"ROS": v1alpha1.ObjectStorageConfig{
		Key: "thanos.yaml",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "ros-thanos-objstore",
		},
		Optional: ptr.To(false),
	},
}

var StageMaps = TemplateMaps{
	Images:               StageImages,
	Versions:             StageVersions,
	LogLevels:            StageLogLevels,
	StorageSize:          StageStorageSize,
	Replicas:             StageReplicas,
	ResourceRequirements: StageResourceRequirements,
	ObjectStorageBucket:  StageObjectStorageBucket,
}

var ProductionMaps = TemplateMaps{
	Images:               ProductionImages,
	Versions:             ProductionVersions,
	LogLevels:            ProductionLogLevels,
	StorageSize:          ProductionStorageSize,
	Replicas:             ProductionReplicas,
	ResourceRequirements: ProductionResourceRequirements,
	ObjectStorageBucket:  ProductionObjectStorageBucket,
}
