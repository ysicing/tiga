package managers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ysicing/tiga/internal/models"
)

// CaddyManager manages Caddy instances
type CaddyManager struct {
	*BaseManager
	apiURL     string
	httpClient *http.Client
}

// NewCaddyManager creates a new Caddy manager
func NewCaddyManager() *CaddyManager {
	return &CaddyManager{
		BaseManager: NewBaseManager(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Initialize initializes the Caddy manager
func (m *CaddyManager) Initialize(ctx context.Context, instance *models.Instance) error {
	if err := m.BaseManager.Initialize(ctx, instance); err != nil {
		return err
	}
	return nil
}

// Connect establishes connection to Caddy
func (m *CaddyManager) Connect(ctx context.Context) error {
	connection := m.instance.Connection
	host, _ := connection["host"].(string)

	// Caddy admin API URL
	adminPort := m.GetConfigValue("admin_port", 2019)
	var adminPortInt int
	switch v := adminPort.(type) {
	case int:
		adminPortInt = v
	case float64:
		adminPortInt = int(v)
	default:
		adminPortInt = 2019
	}

	useHTTPS := m.GetConfigValue("use_https", false).(bool)
	scheme := "http"
	if useHTTPS {
		scheme = "https"
	}

	m.apiURL = fmt.Sprintf("%s://%s:%d", scheme, host, adminPortInt)

	// Test connection
	req, err := http.NewRequestWithContext(ctx, "GET", m.apiURL+"/config/", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Add auth if configured
	apiKey := m.GetConfigValue("api_key", "").(string)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: status %d", ErrConnectionFailed, resp.StatusCode)
	}

	return nil
}

// Disconnect closes connection to Caddy
func (m *CaddyManager) Disconnect(ctx context.Context) error {
	m.apiURL = ""
	return nil
}

// HealthCheck checks Caddy health
func (m *CaddyManager) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if m.apiURL == "" {
		return nil, ErrNotConnected
	}

	start := time.Now()
	status := &HealthStatus{
		Healthy:       false,
		LastCheckTime: time.Now().Format(time.RFC3339),
		Details:       make(map[string]interface{}),
	}

	// Call /config/ endpoint to check health
	req, err := http.NewRequestWithContext(ctx, "GET", m.apiURL+"/config/", nil)
	if err != nil {
		status.Message = fmt.Sprintf("Health check failed: %v", err)
		status.ResponseTime = time.Since(start).Milliseconds()
		return status, nil
	}

	apiKey := m.GetConfigValue("api_key", "").(string)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		status.Message = fmt.Sprintf("Health check failed: %v", err)
		status.ResponseTime = time.Since(start).Milliseconds()
		return status, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		status.Message = fmt.Sprintf("Health check failed: status %d", resp.StatusCode)
		status.ResponseTime = time.Since(start).Milliseconds()
		return status, nil
	}

	// Parse config to get some details
	body, _ := io.ReadAll(resp.Body)
	var config map[string]interface{}
	if err := json.Unmarshal(body, &config); err == nil {
		if apps, ok := config["apps"].(map[string]interface{}); ok {
			status.Details["apps"] = len(apps)
		}
	}

	status.Healthy = true
	status.Message = "Caddy is healthy"
	status.ResponseTime = time.Since(start).Milliseconds()

	return status, nil
}

// CollectMetrics collects metrics from Caddy
func (m *CaddyManager) CollectMetrics(ctx context.Context) (*ServiceMetrics, error) {
	if m.apiURL == "" {
		return nil, ErrNotConnected
	}

	metrics := &ServiceMetrics{
		InstanceID: m.instance.ID,
		Timestamp:  time.Now().Format(time.RFC3339),
		Metrics:    make(map[string]interface{}),
	}

	// Get config
	req, err := http.NewRequestWithContext(ctx, "GET", m.apiURL+"/config/", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricsCollectionFailed, err)
	}

	apiKey := m.GetConfigValue("api_key", "").(string)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricsCollectionFailed, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var config map[string]interface{}
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetricsCollectionFailed, err)
	}

	// Extract metrics from config
	if apps, ok := config["apps"].(map[string]interface{}); ok {
		metrics.Metrics["app_count"] = len(apps)

		// HTTP app metrics
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				metrics.Metrics["server_count"] = len(servers)

				// Count routes
				totalRoutes := 0
				for _, server := range servers {
					if serverMap, ok := server.(map[string]interface{}); ok {
						if routes, ok := serverMap["routes"].([]interface{}); ok {
							totalRoutes += len(routes)
						}
					}
				}
				metrics.Metrics["total_routes"] = totalRoutes
			}
		}

		// TLS app metrics
		if tlsApp, ok := apps["tls"].(map[string]interface{}); ok {
			if automation, ok := tlsApp["automation"].(map[string]interface{}); ok {
				if policies, ok := automation["policies"].([]interface{}); ok {
					metrics.Metrics["tls_policy_count"] = len(policies)
				}
			}
		}
	}

	return metrics, nil
}

// GetInfo returns Caddy service information
func (m *CaddyManager) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	if m.apiURL == "" {
		return nil, ErrNotConnected
	}

	info := make(map[string]interface{})

	// Get config
	req, err := http.NewRequestWithContext(ctx, "GET", m.apiURL+"/config/", nil)
	if err != nil {
		return nil, err
	}

	apiKey := m.GetConfigValue("api_key", "").(string)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var config map[string]interface{}
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, err
	}

	// Get apps
	if apps, ok := config["apps"].(map[string]interface{}); ok {
		appList := make([]string, 0, len(apps))
		for appName := range apps {
			appList = append(appList, appName)
		}
		info["apps"] = appList

		// HTTP app details
		if httpApp, ok := apps["http"].(map[string]interface{}); ok {
			if servers, ok := httpApp["servers"].(map[string]interface{}); ok {
				serverList := make([]map[string]interface{}, 0)
				for serverName, server := range servers {
					serverInfo := map[string]interface{}{
						"name": serverName,
					}

					if serverMap, ok := server.(map[string]interface{}); ok {
						if listen, ok := serverMap["listen"].([]interface{}); ok {
							serverInfo["listen"] = listen
						}
						if routes, ok := serverMap["routes"].([]interface{}); ok {
							serverInfo["route_count"] = len(routes)
						}
					}

					serverList = append(serverList, serverInfo)
				}
				info["servers"] = serverList
			}
		}
	}

	// Get admin config
	if admin, ok := config["admin"].(map[string]interface{}); ok {
		info["admin"] = admin
	}

	return info, nil
}

// ValidateConfig validates Caddy configuration
func (m *CaddyManager) ValidateConfig(config map[string]interface{}) error {
	// Admin port is optional (defaults to 2019)
	// API key is optional
	return nil
}

// Type returns the service type
func (m *CaddyManager) Type() string {
	return "caddy"
}

// UpdateConfig updates Caddy configuration
func (m *CaddyManager) UpdateConfig(ctx context.Context, config map[string]interface{}) error {
	if m.apiURL == "" {
		return ErrNotConnected
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.apiURL+"/config/", bytes.NewReader(configJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	apiKey := m.GetConfigValue("api_key", "").(string)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update config: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
