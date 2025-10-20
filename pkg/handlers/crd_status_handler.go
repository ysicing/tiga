package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ysicing/tiga/pkg/cluster"

	appsv1 "k8s.io/api/apps/v1"
)

// CRDWorkload represents a CRD workload type
type CRDWorkload struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	APIVersion  string `json:"apiVersion"`
	Available   bool   `json:"available"`
	Count       int    `json:"count"`
	Description string `json:"description"`
}

// CRDStatus represents the status of a CRD operator installation
type CRDStatus struct {
	Installed bool          `json:"installed"`
	Version   string        `json:"version,omitempty"`
	Workloads []CRDWorkload `json:"workloads"`
}

// DeploymentInfo for version detection
type DeploymentInfo struct {
	Name          string
	Namespace     string
	ContainerName string
}

// CRDStatusHandler handles CRD status check requests
type CRDStatusHandler struct{}

// NewCRDStatusHandler creates a new CRDStatusHandler
func NewCRDStatusHandler() *CRDStatusHandler {
	return &CRDStatusHandler{}
}

// getCRDTypeConfig returns workload definitions and deployment info for a given CRD type
func getCRDTypeConfig(crdType string) ([]CRDWorkload, []DeploymentInfo, bool) {
	configs := map[string]struct {
		workloads   []CRDWorkload
		deployments []DeploymentInfo
	}{
		"openkruise": {
			workloads: []CRDWorkload{
				{Name: "clonesets", Kind: "CloneSet", APIVersion: "apps.kruise.io/v1alpha1", Description: "CloneSet provides enhanced deployment capabilities"},
				{Name: "advancedstatefulsets", Kind: "AdvancedStatefulSet", APIVersion: "apps.kruise.io/v1beta1", Description: "AdvancedStatefulSet provides enhanced StatefulSet capabilities"},
				{Name: "advanceddaemonsets", Kind: "AdvancedDaemonSet", APIVersion: "apps.kruise.io/v1alpha1", Description: "AdvancedDaemonSet provides enhanced DaemonSet capabilities"},
				{Name: "broadcastjobs", Kind: "BroadcastJob", APIVersion: "apps.kruise.io/v1alpha1", Description: "BroadcastJob runs pods on all or selected nodes"},
				{Name: "advancedcronjobs", Kind: "AdvancedCronJob", APIVersion: "apps.kruise.io/v1alpha1", Description: "AdvancedCronJob provides enhanced CronJob capabilities"},
				{Name: "sidecarsets", Kind: "SidecarSet", APIVersion: "apps.kruise.io/v1alpha1", Description: "SidecarSet injects sidecar containers into pods"},
				{Name: "uniteddeployments", Kind: "UnitedDeployment", APIVersion: "apps.kruise.io/v1alpha1", Description: "UnitedDeployment manages multi-domain workloads"},
				{Name: "workloadspreads", Kind: "WorkloadSpread", APIVersion: "apps.kruise.io/v1alpha1", Description: "WorkloadSpread controls workload distribution across domains"},
				{Name: "imagepulljobs", Kind: "ImagePullJob", APIVersion: "apps.kruise.io/v1alpha1", Description: "ImagePullJob pre-pulls images on nodes"},
				{Name: "containerrecreaterequests", Kind: "ContainerRecreateRequest", APIVersion: "apps.kruise.io/v1alpha1", Description: "ContainerRecreateRequest recreates containers in place"},
				{Name: "resourcedistributions", Kind: "ResourceDistribution", APIVersion: "apps.kruise.io/v1alpha1", Description: "ResourceDistribution distributes resources across namespaces"},
				{Name: "persistentpodstates", Kind: "PersistentPodState", APIVersion: "apps.kruise.io/v1alpha1", Description: "PersistentPodState preserves pod state information"},
				{Name: "podprobemarkers", Kind: "PodProbeMarker", APIVersion: "apps.kruise.io/v1alpha1", Description: "PodProbeMarker custom pod health checks"},
				{Name: "nodeimages", Kind: "NodeImage", APIVersion: "apps.kruise.io/v1alpha1", Description: "NodeImage manages node image lists"},
				{Name: "podunavailablebudgets", Kind: "PodUnavailableBudget", APIVersion: "policy.kruise.io/v1alpha1", Description: "PodUnavailableBudget controls pod disruptions"},
			},
			deployments: []DeploymentInfo{
				{Name: "kruise-controller-manager", Namespace: "kruise-system", ContainerName: "manager"},
			},
		},
		"tailscale": {
			workloads: []CRDWorkload{
				{Name: "connectors", Kind: "Connector", APIVersion: "tailscale.com/v1alpha1", Description: "Connector manages subnet routers, exit nodes, and app connectors"},
				{Name: "proxyclasses", Kind: "ProxyClass", APIVersion: "tailscale.com/v1alpha1", Description: "ProxyClass customizes proxy configuration"},
			},
			deployments: []DeploymentInfo{
				{Name: "operator", Namespace: "tailscale", ContainerName: "operator"},
				{Name: "tailscale-operator", Namespace: "tailscale", ContainerName: "tailscale-operator"},
				{Name: "tailscale-operator", Namespace: "kube-system", ContainerName: "operator"},
				{Name: "operator", Namespace: "kube-system", ContainerName: "tailscale-operator"},
			},
		},
		"traefik": {
			workloads: []CRDWorkload{
				{Name: "ingressroutes", Kind: "IngressRoute", APIVersion: "traefik.io/v1alpha1", Description: "IngressRoute manages HTTP/HTTPS routing rules"},
				{Name: "middlewares", Kind: "Middleware", APIVersion: "traefik.io/v1alpha1", Description: "Middleware defines request/response processing rules"},
				{Name: "ingressroutetcps", Kind: "IngressRouteTCP", APIVersion: "traefik.io/v1alpha1", Description: "IngressRouteTCP manages TCP routing rules"},
				{Name: "ingressrouteudps", Kind: "IngressRouteUDP", APIVersion: "traefik.io/v1alpha1", Description: "IngressRouteUDP manages UDP routing rules"},
				{Name: "tlsoptions", Kind: "TLSOption", APIVersion: "traefik.io/v1alpha1", Description: "TLSOption defines TLS configuration options"},
				{Name: "tlsstores", Kind: "TLSStore", APIVersion: "traefik.io/v1alpha1", Description: "TLSStore defines TLS certificate stores"},
				{Name: "traefikservices", Kind: "TraefikService", APIVersion: "traefik.io/v1alpha1", Description: "TraefikService defines load balancing configuration"},
				{Name: "serverstransports", Kind: "ServersTransport", APIVersion: "traefik.io/v1alpha1", Description: "ServersTransport configures backend server connections"},
			},
			deployments: []DeploymentInfo{
				{Name: "traefik", Namespace: "kube-system", ContainerName: "traefik"},
				{Name: "traefik", Namespace: "traefik", ContainerName: "traefik"},
				{Name: "traefik-controller", Namespace: "kube-system", ContainerName: "traefik"},
			},
		},
		"systemupgrade": {
			workloads: []CRDWorkload{
				{Name: "plans", Kind: "Plan", APIVersion: "upgrade.cattle.io/v1", Description: "Plan defines upgrade specifications for nodes"},
			},
			deployments: []DeploymentInfo{
				{Name: "system-upgrade-controller", Namespace: "system-upgrade", ContainerName: "system-upgrade-controller"},
				{Name: "system-upgrade-controller", Namespace: "cattle-system", ContainerName: "system-upgrade-controller"},
				{Name: "system-upgrade-controller", Namespace: "kube-system", ContainerName: "system-upgrade-controller"},
			},
		},
	}

	config, exists := configs[crdType]
	if !exists {
		return nil, nil, false
	}
	return config.workloads, config.deployments, true
}

// checkWorkloadAvailability checks if a workload CRD exists and counts instances
func checkWorkloadAvailability(ctx context.Context, cs *cluster.ClientSet, workload CRDWorkload) (bool, int) {
	// Parse APIVersion to get group and version
	parts := strings.Split(workload.APIVersion, "/")
	if len(parts) != 2 {
		return false, 0
	}
	group := parts[0]
	version := parts[1]

	// Create GVK for listing
	gvk := schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    workload.Kind,
	}

	// Create an unstructured list
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)

	// Try to list resources (limit to 1 for quick detection)
	err := cs.K8sClient.List(ctx, list, &client.ListOptions{Limit: 1})
	if err != nil {
		// CRD doesn't exist or no permission
		return false, 0
	}

	// CRD exists, now count all instances
	fullList := &unstructured.UnstructuredList{}
	fullList.SetGroupVersionKind(gvk)
	if err := cs.K8sClient.List(ctx, fullList); err != nil {
		// Can detect but can't list - return available but count 0
		return true, 0
	}

	return true, len(fullList.Items)
}

// extractVersionFromImage extracts version from container image tag
func extractVersionFromImage(image string) string {
	// Split by ":" to get tag
	parts := strings.Split(image, ":")
	if len(parts) < 2 {
		return "unknown"
	}
	tag := parts[len(parts)-1]

	// Remove common prefixes like "v"
	tag = strings.TrimPrefix(tag, "v")

	// Ignore "latest" tag
	if tag == "latest" {
		return "unknown"
	}

	return tag
}

// getVersionFromDeployments tries to get version from a list of possible deployments
func getVersionFromDeployments(ctx context.Context, cs *cluster.ClientSet, deploymentInfos []DeploymentInfo) string {
	for _, info := range deploymentInfos {
		var deployment appsv1.Deployment
		err := cs.K8sClient.Get(ctx, types.NamespacedName{
			Name:      info.Name,
			Namespace: info.Namespace,
		}, &deployment)

		if err == nil {
			// Extract version from container image
			for _, container := range deployment.Spec.Template.Spec.Containers {
				if container.Name == info.ContainerName {
					version := extractVersionFromImage(container.Image)
					if version != "unknown" {
						return version
					}
				}
			}
		}
	}
	return ""
}

// GetCRDStatus checks the status of a specific CRD operator
// @Summary Get CRD operator status
// @Description Get the installation status and available workloads of a CRD operator (openkruise, tailscale, traefik, systemupgrade)
// @Tags kubernetes
// @Produce json
// @Param type path string true "CRD type (openkruise|tailscale|traefik|systemupgrade)"
// @Success 200 {object} CRDStatus
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/k8s/crd-status/{type} [get]
func (h *CRDStatusHandler) GetCRDStatus(c *gin.Context) {
	crdType := c.Param("type")
	if crdType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "CRD type parameter is required",
		})
		return
	}

	// Normalize type to lowercase
	crdType = strings.ToLower(crdType)

	// Get configuration for this CRD type
	workloadDefs, deploymentInfos, exists := getCRDTypeConfig(crdType)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Unsupported CRD type. Supported types: openkruise, tailscale, traefik, systemupgrade",
		})
		return
	}

	cs := c.MustGet("cluster").(*cluster.ClientSet)
	ctx := context.Background()

	status := CRDStatus{
		Installed: false,
		Workloads: make([]CRDWorkload, 0, len(workloadDefs)),
	}

	// Check each workload
	for _, workloadDef := range workloadDefs {
		available, count := checkWorkloadAvailability(ctx, cs, workloadDef)
		workload := workloadDef // Copy
		workload.Available = available
		workload.Count = count

		status.Workloads = append(status.Workloads, workload)

		if available {
			status.Installed = true
		}
	}

	// Try to get version from deployments if installed
	if status.Installed {
		status.Version = getVersionFromDeployments(ctx, cs, deploymentInfos)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    status,
	})
}
