package probe

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPProbe performs HTTP/HTTPS health checks
type HTTPProbe struct {
	client *http.Client
}

// NewHTTPProbe creates a new HTTP probe executor
func NewHTTPProbe(timeout time.Duration) *HTTPProbe {
	return &HTTPProbe{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // Allow self-signed certificates
				},
			},
		},
	}
}

// Execute performs an HTTP probe to the target URL
func (p *HTTPProbe) Execute(target string, method string) *ProbeResult {
	result := &ProbeResult{
		Target:    target,
		Type:      "HTTP",
		Timestamp: time.Now(),
	}

	if method == "" {
		method = "GET"
	}

	start := time.Now()

	// Create request
	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	// Set user agent
	req.Header.Set("User-Agent", "Tiga-Agent-Probe/1.0")

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Request failed: %v", err)
		result.Latency = time.Since(start).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	// Read response body (for content validation)
	_, _ = io.Copy(io.Discard, resp.Body)

	result.Latency = time.Since(start).Milliseconds()
	result.StatusCode = resp.StatusCode

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Success = true
	} else {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

// ExecuteWithExpectedStatus performs HTTP probe with expected status code
func (p *HTTPProbe) ExecuteWithExpectedStatus(target string, expectedStatus int) *ProbeResult {
	result := p.Execute(target, "GET")

	if result.Success && result.StatusCode != expectedStatus {
		result.Success = false
		result.Error = fmt.Sprintf("Expected status %d, got %d", expectedStatus, result.StatusCode)
	}

	return result
}

// ExecuteWithHeaders performs HTTP probe with custom headers
func (p *HTTPProbe) ExecuteWithHeaders(target string, headers map[string]string) *ProbeResult {
	result := &ProbeResult{
		Target:    target,
		Type:      "HTTP",
		Timestamp: time.Now(),
	}

	start := time.Now()

	// Create request
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Request failed: %v", err)
		result.Latency = time.Since(start).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	// Read response
	_, _ = io.Copy(io.Discard, resp.Body)

	result.Latency = time.Since(start).Milliseconds()
	result.StatusCode = resp.StatusCode
	result.Success = (resp.StatusCode >= 200 && resp.StatusCode < 400)

	if !result.Success {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}
