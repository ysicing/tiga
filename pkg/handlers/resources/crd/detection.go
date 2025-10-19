package crd

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/ysicing/tiga/pkg/kube"
)

// CRDDetectionService detects installed CRDs in a cluster
type CRDDetectionService struct {
	// Supported CRD groups
	supportedGroups map[string]CRDGroup
}

// CRDGroup represents a group of related CRDs
type CRDGroup struct {
	Name         string   // Display name (e.g., "OpenKruise")
	Group        string   // API group (e.g., "apps.kruise.io")
	ExpectedCRDs []string // Expected CRD names
}

// DetectionResult represents the detection result for all supported CRD groups
type DetectionResult struct {
	Kruise     GroupResult `json:"kruise"`
	Tailscale  GroupResult `json:"tailscale"`
	Traefik    GroupResult `json:"traefik"`
	K3sUpgrade GroupResult `json:"k3s_upgrade"`
}

// GroupResult represents detection result for a single CRD group
type GroupResult struct {
	Installed bool     `json:"installed"`
	CRDs      []string `json:"crds"`
}

// NewCRDDetectionService creates a new CRD detection service
func NewCRDDetectionService() *CRDDetectionService {
	return &CRDDetectionService{
		supportedGroups: map[string]CRDGroup{
			"kruise": {
				Name:  "OpenKruise",
				Group: "apps.kruise.io",
				ExpectedCRDs: []string{
					"clonesets.apps.kruise.io",
					"daemonsets.apps.kruise.io",
					"statefulsets.apps.kruise.io",
					"broadcastjobs.apps.kruise.io",
					"advancedcronjobs.apps.kruise.io",
					"sidecarsets.apps.kruise.io",
				},
			},
			"tailscale": {
				Name:  "Tailscale",
				Group: "tailscale.com",
				ExpectedCRDs: []string{
					"connectors.tailscale.com",
					"proxyclasses.tailscale.com",
					"proxygroups.tailscale.com",
				},
			},
			"traefik": {
				Name:  "Traefik",
				Group: "traefik.io",
				ExpectedCRDs: []string{
					"ingressroutes.traefik.io",
					"middlewares.traefik.io",
					"ingressroutetcps.traefik.io",
					"ingressrouteudps.traefik.io",
				},
			},
			"k3s_upgrade": {
				Name:  "K3s Upgrade Controller",
				Group: "upgrade.cattle.io",
				ExpectedCRDs: []string{
					"plans.upgrade.cattle.io",
				},
			},
		},
	}
}

// DetectAll detects all supported CRD groups in the cluster
func (s *CRDDetectionService) DetectAll(ctx context.Context, client *kube.K8sClient) (*DetectionResult, error) {
	result := &DetectionResult{}

	// Detect OpenKruise
	kruiseResult, err := s.detectGroup(ctx, client, "kruise")
	if err != nil {
		logrus.Warnf("Failed to detect OpenKruise CRDs: %v", err)
	} else {
		result.Kruise = kruiseResult
	}

	// Detect Tailscale
	tailscaleResult, err := s.detectGroup(ctx, client, "tailscale")
	if err != nil {
		logrus.Warnf("Failed to detect Tailscale CRDs: %v", err)
	} else {
		result.Tailscale = tailscaleResult
	}

	// Detect Traefik
	traefikResult, err := s.detectGroup(ctx, client, "traefik")
	if err != nil {
		logrus.Warnf("Failed to detect Traefik CRDs: %v", err)
	} else {
		result.Traefik = traefikResult
	}

	// Detect K3s Upgrade Controller
	k3sResult, err := s.detectGroup(ctx, client, "k3s_upgrade")
	if err != nil {
		logrus.Warnf("Failed to detect K3s Upgrade CRDs: %v", err)
	} else {
		result.K3sUpgrade = k3sResult
	}

	return result, nil
}

// detectGroup detects CRDs for a specific group
func (s *CRDDetectionService) detectGroup(ctx context.Context, client *kube.K8sClient, groupKey string) (GroupResult, error) {
	group, exists := s.supportedGroups[groupKey]
	if !exists {
		return GroupResult{}, nil
	}

	// Get all CRDs in this API group
	installedCRDs, err := GetCRDsByGroup(ctx, client, group.Group)
	if err != nil {
		return GroupResult{}, err
	}

	// Check if any CRDs are installed
	installed := len(installedCRDs) > 0

	return GroupResult{
		Installed: installed,
		CRDs:      installedCRDs,
	}, nil
}
