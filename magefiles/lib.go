package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/bwplotka/mimic/encoding"
	kghelpers "github.com/observatorium/observatorium/configuration_go/kubegen/helpers"
	"github.com/observatorium/observatorium/configuration_go/kubegen/openshift"
	"github.com/observatorium/observatorium/configuration_go/kubegen/workload"
	routev1 "github.com/openshift/api/route/v1"
	templatev1 "github.com/openshift/api/template/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/configuration/clusters"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/ptr"
)

const (
	servingCertSecretNameAnnotation = "service.alpha.openshift.io/serving-cert-secret-name"
	serviceRedirectAnnotation       = "serviceaccounts.openshift.io/oauth-redirectreference.application"

	serviceMonitorTemplate = "service-monitor-template.yaml"

	// openshiftCustomerMonitoringLabel and
	// openShiftClusterMonitoringLabelValue are label's key and value added to
	// all monitoring resources.
	openshiftCustomerMonitoringLabel     = "prometheus"
	openShiftClusterMonitoringLabelValue = "app-sre"

	// openshiftCustomerMonitoringNamespace is the namespace where monitoring resources should be deployed.
	openshiftCustomerMonitoringNamespace = "openshift-customer-monitoring"
)

var migratedClusters = []clusters.ClusterName{
	clusters.ClusterRHOBSUSWestIntegration,
	clusters.ClusterRHOBSUSEastOneStaging,
	clusters.ClusterRHOBSUSWestTwoStaging,
	clusters.ClusterRHOBSUSEastOneProduction,
	clusters.ClusterRHOBSSouthAmericaEastOneProduction,
	clusters.ClusterRHOBSEuropeWestOneProduction,
	clusters.ClusterRHOBSEuropeCentralOneProduction,
	clusters.ClusterRHOBSAsiaPacificNorthEastOneProduction,
	clusters.ClusterRHOBSUSEastOneShardTwoProduction,
}

func isMigratedCluster(config clusters.ClusterConfig) bool {
	for _, cluster := range migratedClusters {
		if config.Name == cluster {
			return true
		}
	}
	return false
}

type resourceRequirements struct {
	cpuRequest    string
	cpuLimit      string
	memoryRequest string
	memoryLimit   string
}

type manifestOptions struct {
	namespace string
	image     string
	imageTag  string
	resourceRequirements
}

// makeOauthProxy creates a container for the oauth-proxy sidecar.
// It contains a template parameter OAUTH_PROXY_COOKIE_SECRET that must be added to the template parameters.
func makeOauthProxy(upstreamPort int32, namespace, serviceAccount, tlsSecret string) *workload.Container {
	const (
		name     = "oauth-proxy"
		image    = "registry.redhat.io/openshift4/ose-oauth-proxy"
		imageTag = "v4.14"
	)

	const (
		cpuRequest    = "100m"
		cpuLimit      = ""
		memoryRequest = "100Mi"
		memoryLimit   = ""
	)

	proxyPort := int32(8443)

	return &workload.Container{
		Name:     name,
		Image:    image,
		ImageTag: imageTag,
		Args: []string{
			"-provider=openshift",
			fmt.Sprintf("-https-address=:%d", proxyPort),
			"-http-address=",
			"-email-domain=*",
			fmt.Sprintf("-upstream=http://localhost:%d", upstreamPort),
			fmt.Sprintf("-openshift-service-account=%s", serviceAccount),
			fmt.Sprintf(`-openshift-sar={"resource": "namespaces", "verb": "get", "name": "%s", "namespace": "%s"}`, namespace, namespace),
			fmt.Sprintf(`-openshift-delegate-urls={"/": {"resource": "namespaces", "verb": "get", "name": "%s", "namespace": "%s"}}`, namespace, namespace),
			"-tls-cert=/etc/tls/private/tls.crt",
			"-tls-key=/etc/tls/private/tls.key",
			"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
			"-cookie-secret-file=/etc/oauth-cookie/cookie.txt",
			"-openshift-ca=/etc/pki/tls/cert.pem",
			"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		},
		Resources: kghelpers.NewResourcesRequirements(cpuRequest, cpuLimit, memoryRequest, memoryLimit),
		Ports: []corev1.ContainerPort{
			{
				Name:          "https",
				ContainerPort: proxyPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		ServicePorts: []corev1.ServicePort{
			kghelpers.NewServicePort("https", int(proxyPort), int(proxyPort)),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "tls",
				MountPath: "/etc/tls/private",
				ReadOnly:  true,
			},
			{
				Name:      "oauth-cookie",
				MountPath: "/etc/oauth-cookie",
				ReadOnly:  true,
			},
		},
		Volumes: []corev1.Volume{
			kghelpers.NewPodVolumeFromSecret("tls", tlsSecret),
			kghelpers.NewPodVolumeFromSecret("oauth-cookie", "oauth-cookie"),
		},
	}
}

// makeOauthProxyContainer creates a corev1.Container for the oauth-proxy sidecar with proper security context.
func makeOauthProxyContainer(upstreamPort int32, namespace, serviceAccount, tlsSecret string) corev1.Container {
	const (
		name     = "oauth-proxy"
		image    = "registry.redhat.io/openshift4/ose-oauth-proxy"
		imageTag = "v4.14"
	)

	const (
		cpuRequest    = "100m"
		memoryRequest = "100Mi"
	)

	proxyPort := int32(8443)

	return corev1.Container{
		Name:  name,
		Image: fmt.Sprintf("%s:%s", image, imageTag),
		Args: []string{
			"-provider=openshift",
			fmt.Sprintf("-https-address=:%d", proxyPort),
			"-http-address=",
			"-email-domain=*",
			fmt.Sprintf("-upstream=http://localhost:%d", upstreamPort),
			fmt.Sprintf("-openshift-service-account=%s", serviceAccount),
			fmt.Sprintf(`-openshift-sar={"resource": "namespaces", "verb": "get", "name": "%s", "namespace": "%s"}`, namespace, namespace),
			fmt.Sprintf(`-openshift-delegate-urls={"/": {"resource": "namespaces", "verb": "get", "name": "%s", "namespace": "%s"}}`, namespace, namespace),
			"-tls-cert=/etc/tls/private/tls.crt",
			"-tls-key=/etc/tls/private/tls.key",
			"-client-secret-file=/var/run/secrets/kubernetes.io/serviceaccount/token",
			"-cookie-secret-file=/etc/oauth-cookie/cookie.txt",
			"-openshift-ca=/etc/pki/tls/cert.pem",
			"-openshift-ca=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		},
		Resources: kghelpers.NewResourcesRequirements(cpuRequest, "", memoryRequest, ""),
		Ports: []corev1.ContainerPort{
			{
				Name:          "https",
				ContainerPort: proxyPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			RunAsNonRoot: ptr.To(true),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "tls",
				MountPath: "/etc/tls/private",
				ReadOnly:  true,
			},
			{
				Name:      "oauth-cookie",
				MountPath: "/etc/oauth-cookie",
				ReadOnly:  true,
			},
		},
	}
}

func getAndRemoveObject[T metav1.Object](objects []runtime.Object, name string) (T, []runtime.Object) {
	var ret T
	var atIndex int
	found := false

	for i, obj := range objects {
		typedObject, ok := obj.(T)
		if ok {
			if name != "" && typedObject.GetName() != name {
				continue
			}

			// Check if we already found an object of this type. If so, panic.
			if found {
				panic(fmt.Sprintf("found multiple objects of type %T", *new(T)))
			}

			ret = typedObject
			found = true
			atIndex = i
			break
		}
	}

	if !found {
		panic(fmt.Sprintf("could not find object of type %T", *new(T)))
	}
	var modifiedObjs []runtime.Object
	for i := range objects {
		if i != atIndex {
			modifiedObjs = append(modifiedObjs, objects[i])
		}
	}
	return ret, modifiedObjs
}

// postProcessServiceMonitor updates the service monitor to work with the app-sre prometheus.
func postProcessServiceMonitor(serviceMonitor *monv1.ServiceMonitor, namespaceSelector string) encoding.Encoder {
	serviceMonitor.ObjectMeta.Namespace = openshiftCustomerMonitoringNamespace
	serviceMonitor.Spec.NamespaceSelector.MatchNames = []string{namespaceSelector}
	serviceMonitor.ObjectMeta.Labels[openshiftCustomerMonitoringLabel] = openShiftClusterMonitoringLabelValue

	name := serviceMonitor.Name + "-service-monitor-" + namespaceSelector

	template := openshift.WrapInTemplate([]runtime.Object{serviceMonitor}, metav1.ObjectMeta{
		Name: name,
	}, nil)
	return encoding.GhodssYAML(template)
}

func sortTemplateParams(params []templatev1.Parameter) []templatev1.Parameter {
	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})
	return params
}

func createServiceAccount(name, namespace string, labels map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func deepCopyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func getCustomResourceDefinition(url string) (*v1.CustomResourceDefinition, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: %s", url, resp.Status)
	}

	var obj v1.CustomResourceDefinition
	decoder := yaml.NewYAMLOrJSONDecoder(resp.Body, 100000)
	err = decoder.Decode(&obj)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", url, err)
	}

	return &obj, nil
}

func getClusterRole(url string) (*rbacv1.ClusterRole, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: %s", url, resp.Status)
	}

	var obj rbacv1.ClusterRole
	decoder := yaml.NewYAMLOrJSONDecoder(resp.Body, 100000)
	err = decoder.Decode(&obj)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", url, err)
	}

	return &obj, nil
}

// getResourceKind returns the Kind field from a Kubernetes object
func getResourceKind(obj runtime.Object) string {
	switch obj.(type) {
	case *appsv1.Deployment:
		return "Deployment"
	case *corev1.Service:
		return "Service"
	case *corev1.ServiceAccount:
		return "ServiceAccount"
	case *corev1.ConfigMap:
		return "ConfigMap"
	case *corev1.Secret:
		return "Secret"
	case *appsv1.StatefulSet:
		return "StatefulSet"
	case *rbacv1.ClusterRole:
		return "ClusterRole"
	case *rbacv1.ClusterRoleBinding:
		return "ClusterRoleBinding"
	case *rbacv1.Role:
		return "Role"
	case *rbacv1.RoleBinding:
		return "RoleBinding"
	case *routev1.Route:
		return "Route"
	case *monv1.Alertmanager:
		return "Alertmanager"
	case *monv1.ServiceMonitor:
		return "ServiceMonitor"
	default:
		// Try to get the kind from TypeMeta as a fallback
		if gvk := obj.GetObjectKind().GroupVersionKind(); gvk.Kind != "" {
			return gvk.Kind
		}
		// If TypeMeta doesn't have Kind, try to infer from type name
		objType := fmt.Sprintf("%T", obj)
		if objType != "" && len(objType) > 1 {
			// Extract the last part after the dot and asterisk (e.g., "*v1.LokiStack" -> "LokiStack")
			parts := strings.Split(objType, ".")
			if len(parts) > 0 {
				typeName := parts[len(parts)-1]
				return typeName
			}
		}
		return "Unknown"
	}
}

// getKubernetesResourceName extracts a meaningful name from a Kubernetes object
func getKubernetesResourceName(obj runtime.Object) string {
	if obj == nil {
		return "unknown"
	}

	switch o := obj.(type) {
	case metav1.Object:
		name := o.GetName()
		if name != "" {
			return name
		}
	}

	// Fallback to the object type
	return "unnamed"
}

// createServiceSelectorLabels creates labels for service selectors, removing version labels that go stale
func createServiceSelectorLabels(labels map[string]string) map[string]string {
	selectorLabels := deepCopyMap(labels)
	// Remove version label as it goes stale
	delete(selectorLabels, "app.kubernetes.io/version")
	return selectorLabels
}
