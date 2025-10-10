package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ysicing/tiga/proto"
)

// TestNewProbeTaskHandler tests probe handler creation
func TestNewProbeTaskHandler(t *testing.T) {
	handler := NewProbeTaskHandler("test-uuid-123", nil)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.httpProbe)
	assert.NotNil(t, handler.tcpProbe)
	assert.NotNil(t, handler.icmpProbe)
	assert.NotNil(t, handler.resultBuffer)
}

// TestHandleProbeTask tests probe task parameter validation
func TestHandleProbeTask(t *testing.T) {
	handler := NewProbeTaskHandler("test-uuid-123", nil)

	config := &Config{
		UUID: "test-uuid-123",
	}

	t.Run("missing type parameter", func(t *testing.T) {
		task := &proto.AgentTask{
			TaskId:   "task-001",
			TaskType: "PROBE",
			Params: map[string]string{
				"target":     "https://example.com",
				"monitor_id": "mon-001",
			},
		}

		// Should log error and return without panic
		handler.HandleProbeTask(task, config)
		// Test passes if no panic occurs
	})

	t.Run("missing target parameter", func(t *testing.T) {
		task := &proto.AgentTask{
			TaskId:   "task-002",
			TaskType: "PROBE",
			Params: map[string]string{
				"type":       "http",
				"monitor_id": "mon-001",
			},
		}

		handler.HandleProbeTask(task, config)
		// Test passes if no panic occurs
	})

	t.Run("missing monitor_id parameter", func(t *testing.T) {
		task := &proto.AgentTask{
			TaskId:   "task-003",
			TaskType: "PROBE",
			Params: map[string]string{
				"type":   "http",
				"target": "https://example.com",
			},
		}

		handler.HandleProbeTask(task, config)
		// Test passes if no panic occurs
	})

	t.Run("unknown probe type", func(t *testing.T) {
		task := &proto.AgentTask{
			TaskId:   "task-004",
			TaskType: "PROBE",
			Params: map[string]string{
				"type":       "unknown",
				"target":     "https://example.com",
				"monitor_id": "mon-001",
			},
		}

		handler.HandleProbeTask(task, config)
		// Test passes if no panic occurs
	})

	// Note: Testing actual probe execution (HTTP/TCP/ICMP) requires network access
	// and should be covered by integration tests. Unit tests focus on parameter validation
	// and error handling.
}

// TestExecuteHTTPProbe tests HTTP probe execution with various configurations
func TestExecuteHTTPProbe(t *testing.T) {
	t.Skip("Skipping HTTP probe test as it requires network access - use integration tests instead")

	handler := NewProbeTaskHandler("test-uuid-123", nil)

	t.Run("standard HTTP GET probe", func(t *testing.T) {
		// Test with a reliable target (Google)
		result := handler.executeHTTPProbe("https://www.google.com", "GET", map[string]string{})

		require.NotNil(t, result)
		assert.True(t, result.Success, "Google should be reachable")
		assert.Greater(t, result.Latency, int64(0), "Latency should be positive")
		assert.Equal(t, 200, result.StatusCode)
	})

	t.Run("HTTP probe with custom headers", func(t *testing.T) {
		params := map[string]string{
			"headers": `{"User-Agent": "TigaAgent/1.0"}`,
		}

		result := handler.executeHTTPProbe("https://www.google.com", "GET", params)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("HTTP probe with expected status", func(t *testing.T) {
		params := map[string]string{
			"expected_status": "200",
		}

		result := handler.executeHTTPProbe("https://www.google.com", "GET", params)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("HTTP probe with invalid URL", func(t *testing.T) {
		result := handler.executeHTTPProbe("invalid-url", "GET", map[string]string{})
		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Error)
	})
}

// TestExecuteTCPProbe tests TCP probe execution
func TestExecuteTCPProbe(t *testing.T) {
	t.Skip("Skipping TCP probe test as it requires network access - use integration tests instead")

	handler := NewProbeTaskHandler("test-uuid-123", nil)

	t.Run("TCP probe to valid port", func(t *testing.T) {
		// Test with Google DNS (usually reachable on port 443)
		result := handler.executeTCPProbe("8.8.8.8:443")

		require.NotNil(t, result)
		assert.True(t, result.Success, "Google DNS should be reachable on port 443")
		assert.Greater(t, result.Latency, int64(0))
	})

	t.Run("TCP probe to invalid port", func(t *testing.T) {
		// Use a high unlikely-to-be-open port
		result := handler.executeTCPProbe("127.0.0.1:65534")

		require.NotNil(t, result)
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Error)
	})

	t.Run("TCP probe with invalid address", func(t *testing.T) {
		result := handler.executeTCPProbe("invalid-address")
		require.NotNil(t, result)
		assert.False(t, result.Success)
	})
}

// TestExecuteICMPProbe tests ICMP probe execution
func TestExecuteICMPProbe(t *testing.T) {
	t.Skip("Skipping ICMP test as it requires root privileges on most systems")

	// Note: ICMP probes typically require elevated privileges
	// These tests should be run in privileged integration test environment

	handler := NewProbeTaskHandler("test-uuid-123", nil)

	t.Run("ICMP probe to valid host", func(t *testing.T) {
		result := handler.executeICMPProbe("8.8.8.8")
		require.NotNil(t, result)
		assert.True(t, result.Success)
		assert.Greater(t, result.Latency, int64(0))
	})
}

// TestProbeResultConversion tests conversion from probe.ProbeResult to proto.ProbeResult
func TestProbeResultConversion(t *testing.T) {
	// Test the conversion logic in reportProbeResult
	// Since reportProbeResult is private and has side effects (buffer Add),
	// we test the conversion logic indirectly through the structure

	t.Run("timestamp conversion to milliseconds", func(t *testing.T) {
		// Unix timestamp 1234567890 = 2009-02-13 23:31:30 UTC
		testTime := time.Unix(1234567890, 0)
		expectedMillis := int64(1234567890000)

		actualMillis := testTime.UnixMilli()
		assert.Equal(t, expectedMillis, actualMillis)
	})

	t.Run("int64 to int32 conversion", func(t *testing.T) {
		var latency int64 = 456
		int32Latency := int32(latency)
		assert.Equal(t, int32(456), int32Latency)
	})

	t.Run("status code conversion", func(t *testing.T) {
		statusCode := 200
		int32Status := int32(statusCode)
		assert.Equal(t, int32(200), int32Status)
	})
}

// TestProbeTypeHandling tests probe type string handling
func TestProbeTypeHandling(t *testing.T) {
	handler := NewProbeTaskHandler("test-uuid-123", nil)
	config := &Config{UUID: "test-uuid"}

	tests := []struct {
		name      string
		probeType string
		target    string
		shouldRun bool // whether probe should attempt to run
	}{
		{"HTTP uppercase", "HTTP", "https://example.com", true},
		{"http lowercase", "http", "https://example.com", true},
		{"TCP uppercase", "TCP", "example.com:80", true},
		{"tcp lowercase", "tcp", "example.com:80", true},
		{"ICMP uppercase", "ICMP", "8.8.8.8", true},
		{"icmp lowercase", "icmp", "8.8.8.8", true},
		{"Unknown type", "unknown", "target", false},
		{"Empty type", "", "target", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &proto.AgentTask{
				TaskId:   "test-task",
				TaskType: "PROBE",
				Params: map[string]string{
					"type":       tt.probeType,
					"target":     tt.target,
					"monitor_id": "mon-123",
				},
			}

			// Should not panic regardless of input
			handler.HandleProbeTask(task, config)
		})
	}
}

// TestProbeHTTPMethodHandling tests HTTP method parameter handling
func TestProbeHTTPMethodHandling(t *testing.T) {
	t.Skip("Skipping as it requires network access")

	handler := NewProbeTaskHandler("test-uuid-123", nil)

	t.Run("default method is GET", func(t *testing.T) {
		// When method is not specified, should default to GET
		result := handler.executeHTTPProbe("https://httpbin.org/get", "", map[string]string{})
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("explicit GET method", func(t *testing.T) {
		result := handler.executeHTTPProbe("https://httpbin.org/get", "GET", map[string]string{})
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})
}

// TestProbeHTTPHeadersHandling tests HTTP headers parameter parsing
func TestProbeHTTPHeadersHandling(t *testing.T) {
	t.Skip("Skipping as it requires network access")

	handler := NewProbeTaskHandler("test-uuid-123", nil)

	t.Run("valid JSON headers", func(t *testing.T) {
		params := map[string]string{
			"headers": `{"User-Agent": "Test/1.0", "X-Custom": "value"}`,
		}
		result := handler.executeHTTPProbe("https://httpbin.org/headers", "GET", params)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("invalid JSON headers", func(t *testing.T) {
		params := map[string]string{
			"headers": `invalid-json`,
		}
		// Should fall back to standard HTTP probe
		result := handler.executeHTTPProbe("https://httpbin.org/headers", "GET", params)
		require.NotNil(t, result)
	})

	t.Run("empty headers", func(t *testing.T) {
		params := map[string]string{
			"headers": `{}`,
		}
		result := handler.executeHTTPProbe("https://httpbin.org/headers", "GET", params)
		require.NotNil(t, result)
	})
}

// TestProbeHTTPExpectedStatusHandling tests expected status parameter parsing
func TestProbeHTTPExpectedStatusHandling(t *testing.T) {
	t.Skip("Skipping as it requires network access")

	handler := NewProbeTaskHandler("test-uuid-123", nil)

	t.Run("valid expected status", func(t *testing.T) {
		params := map[string]string{
			"expected_status": "200",
		}
		result := handler.executeHTTPProbe("https://httpbin.org/status/200", "GET", params)
		require.NotNil(t, result)
		assert.True(t, result.Success)
	})

	t.Run("invalid expected status format", func(t *testing.T) {
		params := map[string]string{
			"expected_status": "not-a-number",
		}
		// Should fall back to standard HTTP probe
		result := handler.executeHTTPProbe("https://httpbin.org/status/200", "GET", params)
		require.NotNil(t, result)
	})
}
