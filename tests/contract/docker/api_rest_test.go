package docker_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContractDockerInstancesListAPI tests the GET /api/v1/docker/instances endpoint contract
func TestContractDockerInstancesListAPI(t *testing.T) {
	t.Run("should return instances list with pagination", func(t *testing.T) {
		// Mock response according to api_rest.md contract
		mockResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"instances": []map[string]interface{}{
					{
						"id":                "test-uuid-1",
						"name":              "prod-docker-1",
						"description":       "Production Docker instance",
						"agent_id":          "agent-uuid-1",
						"host_id":           "host-uuid-1",
						"host_name":         "prod-server-1",
						"health_status":     "online",
						"last_connected_at": "2025-10-22T10:30:00Z",
						"last_health_check": "2025-10-22T10:35:00Z",
						"docker_version":    "24.0.7",
						"api_version":       "1.43",
						"storage_driver":    "overlay2",
						"operating_system":  "Ubuntu 22.04",
						"architecture":      "x86_64",
						"container_count":   15,
						"image_count":       8,
						"volume_count":      5,
						"network_count":     3,
						"tags":              []string{"production", "web"},
						"created_at":        "2025-10-01T08:00:00Z",
						"updated_at":        "2025-10-22T10:35:00Z",
					},
				},
				"total":     50,
				"page":      1,
				"page_size": 20,
			},
		}

		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/docker/instances", r.URL.Path)

			// Verify query parameters support
			query := r.URL.Query()
			if query.Get("page") != "" {
				assert.Equal(t, "1", query.Get("page"))
			}
			if query.Get("page_size") != "" {
				pageSize := query.Get("page_size")
				assert.LessOrEqual(t, len(pageSize), 3) // max 100
			}

			// Return mock response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		// Make request
		resp, err := http.Get(server.URL + "/api/v1/docker/instances?page=1&page_size=20")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Validate response structure
		assert.True(t, result["success"].(bool))
		assert.NotNil(t, result["data"])

		data := result["data"].(map[string]interface{})
		assert.NotNil(t, data["instances"])
		assert.Equal(t, float64(50), data["total"].(float64))
		assert.Equal(t, float64(1), data["page"].(float64))
		assert.Equal(t, float64(20), data["page_size"].(float64))

		// Validate instance structure
		instances := data["instances"].([]interface{})
		assert.NotEmpty(t, instances)
		instance := instances[0].(map[string]interface{})

		// Required fields validation
		requiredFields := []string{
			"id", "name", "agent_id", "host_id", "health_status",
			"docker_version", "api_version", "container_count",
			"image_count", "created_at", "updated_at",
		}
		for _, field := range requiredFields {
			assert.Contains(t, instance, field, "Missing required field: "+field)
		}
	})

	t.Run("should return error for invalid page_size", func(t *testing.T) {
		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_PARAMETER",
				"message": "Invalid page_size: maximum is 100",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		resp, err := http.Get(server.URL + "/api/v1/docker/instances?page_size=1000")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.False(t, result["success"].(bool))
		assert.NotNil(t, result["error"])
	})
}

// TestContractDockerInstanceDetailAPI tests the GET /api/v1/docker/instances/:id endpoint contract
func TestContractDockerInstanceDetailAPI(t *testing.T) {
	t.Run("should return instance detail", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"id":                "test-uuid-1",
				"name":              "prod-docker-1",
				"description":       "Production Docker instance",
				"agent_id":          "agent-uuid-1",
				"host_id":           "host-uuid-1",
				"host_name":         "prod-server-1",
				"health_status":     "online",
				"last_connected_at": "2025-10-22T10:30:00Z",
				"docker_version":    "24.0.7",
				"container_count":   15,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Path, "/api/v1/docker/instances/")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Get(server.URL + "/api/v1/docker/instances/test-uuid-1")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))
	})

	t.Run("should return 404 for non-existent instance", func(t *testing.T) {
		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "NOT_FOUND",
				"message": "Docker instance not found",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		resp, err := http.Get(server.URL + "/api/v1/docker/instances/nonexistent-id")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestContractDockerInstanceCreateAPI tests the POST /api/v1/docker/instances endpoint contract
func TestContractDockerInstanceCreateAPI(t *testing.T) {
	t.Run("should create instance successfully", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":        "prod-docker-1",
			"description": "Production Docker instance",
			"agent_id":    "agent-uuid-1",
			"tags":        []string{"production", "web"},
		}

		mockResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"id":            "new-uuid",
				"name":          "prod-docker-1",
				"description":   "Production Docker instance",
				"agent_id":      "agent-uuid-1",
				"health_status": "unknown",
				"tags":          []string{"production", "web"},
				"created_at":    "2025-10-22T10:40:00Z",
				"updated_at":    "2025-10-22T10:40:00Z",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/docker/instances", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Verify request body
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "prod-docker-1", body["name"])
			assert.NotEmpty(t, body["agent_id"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		// Make request
		bodyJSON, _ := json.Marshal(requestBody)
		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances",
			"application/json",
			strings.NewReader(string(bodyJSON)),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))
		data := result["data"].(map[string]interface{})
		assert.Equal(t, "prod-docker-1", data["name"])
		assert.Equal(t, "unknown", data["health_status"])
	})

	t.Run("should return validation error for invalid name", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":     "", // Invalid: empty name
			"agent_id": "agent-uuid-1",
		}

		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_PARAMETER",
				"message": "name is required and must be 1-255 characters",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		bodyJSON, _ := json.Marshal(requestBody)
		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances",
			"application/json",
			strings.NewReader(string(bodyJSON)),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestContractContainerListAPI tests the GET /api/v1/docker/instances/:id/containers endpoint contract
func TestContractContainerListAPI(t *testing.T) {
	t.Run("should return containers list", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"containers": []map[string]interface{}{
					{
						"id":       "container-id-1",
						"names":    []string{"/nginx-web"},
						"image":    "nginx:latest",
						"image_id": "sha256:abc123",
						"state":    "running",
						"status":   "Up 2 hours",
						"ports": []map[string]interface{}{
							{
								"private_port": 80,
								"public_port":  8080,
								"type":         "tcp",
							},
						},
						"created_at": "2025-10-20T08:00:00Z",
					},
				},
				"total": 15,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Path, "/containers")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Get(server.URL + "/api/v1/docker/instances/test-uuid/containers")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		containers := data["containers"].([]interface{})
		assert.NotEmpty(t, containers)

		container := containers[0].(map[string]interface{})
		assert.Equal(t, "running", container["state"])
		assert.NotEmpty(t, container["names"])
	})
}

// TestContractContainerStartAPI tests the POST /api/v1/docker/instances/:id/containers/:cid/start endpoint contract
func TestContractContainerStartAPI(t *testing.T) {
	t.Run("should start container successfully", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"message": "Container started successfully",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/start")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/containers/container-id/start",
			"application/json",
			nil,
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))
	})

	t.Run("should return error when instance offline", func(t *testing.T) {
		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INSTANCE_OFFLINE",
				"message": "Docker instance is offline",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/offline-uuid/containers/container-id/start",
			"application/json",
			nil,
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})
}

// TestContractImageListAPI tests the GET /api/v1/docker/instances/:id/images endpoint contract
func TestContractImageListAPI(t *testing.T) {
	t.Run("should return images list", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"images": []map[string]interface{}{
					{
						"id":           "sha256:abc123def456",
						"repo_tags":    []string{"nginx:latest", "nginx:1.25"},
						"repo_digests": []string{"nginx@sha256:..."},
						"size":         142000000,
						"virtual_size": 142000000,
						"created":      "2025-10-15T10:00:00Z",
						"labels": map[string]string{
							"maintainer": "NGINX Docker Maintainers",
						},
					},
				},
				"total": 8,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Path, "/images")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Get(server.URL + "/api/v1/docker/instances/test-uuid/images")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		images := data["images"].([]interface{})
		assert.NotEmpty(t, images)

		image := images[0].(map[string]interface{})
		assert.NotEmpty(t, image["repo_tags"])
		assert.NotZero(t, image["size"])
	})
}

// TestContractAuditLogsAPI tests the GET /api/v1/docker/audit-logs endpoint contract
func TestContractAuditLogsAPI(t *testing.T) {
	t.Run("should return audit logs with pagination", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"logs": []map[string]interface{}{
					{
						"id":            "log-uuid-1",
						"user_id":       "user-uuid",
						"username":      "admin",
						"action":        "container_start",
						"resource_type": "docker_container",
						"resource_id":   "container-id",
						"resource_name": "nginx-web",
						"details": map[string]interface{}{
							"instance_id":   "instance-uuid",
							"instance_name": "prod-docker-1",
							"state_before":  "exited",
							"state_after":   "running",
							"success":       true,
							"duration":      1250,
						},
						"ip_address": "192.168.1.100",
						"timestamp":  "2025-10-22T10:30:00Z",
					},
				},
				"total":     100,
				"page":      1,
				"page_size": 20,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/docker/audit-logs", r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Get(server.URL + "/api/v1/docker/audit-logs?page=1&page_size=20")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		logs := data["logs"].([]interface{})
		assert.NotEmpty(t, logs)

		log := logs[0].(map[string]interface{})
		assert.Equal(t, "container_start", log["action"])
		assert.Equal(t, "docker_container", log["resource_type"])
		assert.NotNil(t, log["details"])
	})
}

// TestContractContainerStopAPI tests the POST /api/v1/docker/instances/:id/containers/:cid/stop endpoint contract
func TestContractContainerStopAPI(t *testing.T) {
	t.Run("should stop container successfully", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"message": "容器停止成功",
			"data": map[string]interface{}{
				"container_id": "abc123def456",
				"duration":     850,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/stop")

			// Verify optional timeout parameter
			var body map[string]interface{}
			if r.Body != nil {
				json.NewDecoder(r.Body).Decode(&body)
				if timeout, ok := body["timeout"]; ok {
					assert.IsType(t, float64(0), timeout)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		reqBody := map[string]interface{}{"timeout": 10}
		bodyJSON, _ := json.Marshal(reqBody)
		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/containers/container-id/stop",
			"application/json",
			strings.NewReader(string(bodyJSON)),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))
		assert.Equal(t, "容器停止成功", result["message"])
	})

	t.Run("should return error when container not running", func(t *testing.T) {
		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "BAD_REQUEST",
				"message": "容器未在运行",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/containers/exited-container/stop",
			"application/json",
			nil,
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestContractContainerRestartAPI tests the POST /api/v1/docker/instances/:id/containers/:cid/restart endpoint contract
func TestContractContainerRestartAPI(t *testing.T) {
	t.Run("should restart container successfully", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"message": "容器重启成功",
			"data": map[string]interface{}{
				"container_id": "abc123def456",
				"duration":     2100,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/restart")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/containers/container-id/restart",
			"application/json",
			nil,
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))
	})
}

// TestContractContainerPauseAPI tests the POST /api/v1/docker/instances/:id/containers/:cid/pause endpoint contract
func TestContractContainerPauseAPI(t *testing.T) {
	t.Run("should pause container successfully", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"message": "容器已暂停",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/pause")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/containers/container-id/pause",
			"application/json",
			nil,
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestContractContainerUnpauseAPI tests the POST /api/v1/docker/instances/:id/containers/:cid/unpause endpoint contract
func TestContractContainerUnpauseAPI(t *testing.T) {
	t.Run("should unpause container successfully", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"message": "容器已恢复",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/unpause")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/containers/container-id/unpause",
			"application/json",
			nil,
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestContractContainerDeleteAPI tests the DELETE /api/v1/docker/instances/:id/containers/:cid endpoint contract
func TestContractContainerDeleteAPI(t *testing.T) {
	t.Run("should delete container successfully", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"message": "容器已删除",
			"data": map[string]interface{}{
				"container_id": "abc123def456",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Contains(t, r.URL.Path, "/containers/")

			// Verify query parameters
			query := r.URL.Query()
			if force := query.Get("force"); force != "" {
				assert.Contains(t, []string{"true", "false"}, force)
			}
			if removeVolumes := query.Get("remove_volumes"); removeVolumes != "" {
				assert.Contains(t, []string{"true", "false"}, removeVolumes)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client := &http.Client{}
		req, _ := http.NewRequest(
			http.MethodDelete,
			server.URL+"/api/v1/docker/instances/test-uuid/containers/container-id?force=true&remove_volumes=false",
			nil,
		)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))
	})

	t.Run("should return error when container in use", func(t *testing.T) {
		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "CONFLICT",
				"message": "容器正在运行，无法删除",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		client := &http.Client{}
		req, _ := http.NewRequest(
			http.MethodDelete,
			server.URL+"/api/v1/docker/instances/test-uuid/containers/running-container",
			nil,
		)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

// TestContractContainerLogsAPI tests the GET /api/v1/docker/instances/:id/containers/:cid/logs endpoint contract
func TestContractContainerLogsAPI(t *testing.T) {
	t.Run("should return container logs in JSON format", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"logs": []string{
					"2025-10-22T10:00:00Z [INFO] Server started on port 8080",
					"2025-10-22T10:00:01Z [INFO] Connected to database",
					"2025-10-22T10:00:02Z [INFO] Ready to accept connections",
				},
				"total_lines": 3,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Path, "/logs")

			// Verify query parameters
			query := r.URL.Query()
			assert.Equal(t, "false", query.Get("follow"))
			if tail := query.Get("tail"); tail != "" {
				assert.Equal(t, "100", tail)
			}
			if timestamps := query.Get("timestamps"); timestamps != "" {
				assert.Equal(t, "true", timestamps)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Get(server.URL + "/api/v1/docker/instances/test-uuid/containers/container-id/logs?follow=false&tail=100&timestamps=true")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		logs := data["logs"].([]interface{})
		assert.Len(t, logs, 3)
		assert.Equal(t, float64(3), data["total_lines"].(float64))
	})
}

// TestContractContainerStatsAPI tests the GET /api/v1/docker/instances/:id/containers/:cid/stats endpoint contract
func TestContractContainerStatsAPI(t *testing.T) {
	t.Run("should return container stats in JSON format", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"container_id":         "abc123def456",
				"timestamp":            1729590000,
				"cpu_usage_percent":    12.5,
				"cpu_usage_nano":       1250000000,
				"memory_usage":         134217728,
				"memory_limit":         536870912,
				"memory_usage_percent": 25.0,
				"network_rx_bytes":     1048576,
				"network_tx_bytes":     524288,
				"block_read_bytes":     2097152,
				"block_write_bytes":    1048576,
				"pids_current":         5,
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Contains(t, r.URL.Path, "/stats")

			// Verify query parameters
			query := r.URL.Query()
			assert.Equal(t, "false", query.Get("stream"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		resp, err := http.Get(server.URL + "/api/v1/docker/instances/test-uuid/containers/container-id/stats?stream=false")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		// Validate required fields
		assert.NotNil(t, data["cpu_usage_percent"])
		assert.NotNil(t, data["memory_usage"])
		assert.NotNil(t, data["memory_limit"])
		assert.NotNil(t, data["memory_usage_percent"])
	})
}

// TestContractImageDeleteAPI tests the DELETE /api/v1/docker/instances/:id/images/:image_id endpoint contract
func TestContractImageDeleteAPI(t *testing.T) {
	t.Run("should delete image successfully", func(t *testing.T) {
		mockResponse := map[string]interface{}{
			"success": true,
			"message": "镜像已删除",
			"data": map[string]interface{}{
				"deleted":  []string{"sha256:abc123"},
				"untagged": []string{"nginx:latest"},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Contains(t, r.URL.Path, "/images/")

			// Verify query parameters
			query := r.URL.Query()
			if force := query.Get("force"); force != "" {
				assert.Contains(t, []string{"true", "false"}, force)
			}
			if noPrune := query.Get("no_prune"); noPrune != "" {
				assert.Contains(t, []string{"true", "false"}, noPrune)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		client := &http.Client{}
		req, _ := http.NewRequest(
			http.MethodDelete,
			server.URL+"/api/v1/docker/instances/test-uuid/images/sha256:abc123?force=false&no_prune=false",
			nil,
		)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		assert.NotEmpty(t, data["deleted"])
	})

	t.Run("should return error when image in use", func(t *testing.T) {
		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "CONFLICT",
				"message": "镜像被容器使用，无法删除",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		client := &http.Client{}
		req, _ := http.NewRequest(
			http.MethodDelete,
			server.URL+"/api/v1/docker/instances/test-uuid/images/sha256:inuse",
			nil,
		)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

// TestContractImageTagAPI tests the POST /api/v1/docker/instances/:id/images/:image_id/tag endpoint contract
func TestContractImageTagAPI(t *testing.T) {
	t.Run("should tag image successfully", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"target_repo": "myregistry.com/nginx",
			"target_tag":  "v1.0",
		}

		mockResponse := map[string]interface{}{
			"success": true,
			"message": "镜像标签已创建",
			"data": map[string]interface{}{
				"source_image": "nginx:latest",
				"target_image": "myregistry.com/nginx:v1.0",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Contains(t, r.URL.Path, "/tag")

			// Verify request body
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.NotEmpty(t, body["target_repo"])
			assert.NotEmpty(t, body["target_tag"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		bodyJSON, _ := json.Marshal(requestBody)
		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/images/sha256:abc123/tag",
			"application/json",
			strings.NewReader(string(bodyJSON)),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.True(t, result["success"].(bool))

		data := result["data"].(map[string]interface{})
		assert.NotEmpty(t, data["target_image"])
	})

	t.Run("should return error for invalid tag format", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"target_repo": "invalid tag format",
			"target_tag":  "",
		}

		mockError := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "INVALID_PARAMETER",
				"message": "标签格式无效",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(mockError)
		}))
		defer server.Close()

		bodyJSON, _ := json.Marshal(requestBody)
		resp, err := http.Post(
			server.URL+"/api/v1/docker/instances/test-uuid/images/sha256:abc123/tag",
			"application/json",
			strings.NewReader(string(bodyJSON)),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
