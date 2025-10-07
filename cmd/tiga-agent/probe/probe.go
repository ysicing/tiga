package probe

import (
	"time"
)

// ProbeType represents the type of probe
type ProbeType string

const (
	ProbeTypeHTTP ProbeType = "HTTP"
	ProbeTypeTCP  ProbeType = "TCP"
	ProbeTypeICMP ProbeType = "ICMP"
)

// ProbeResult represents the result of a probe execution
type ProbeResult struct {
	Target     string    // Target URL/host
	Type       string    // Probe type (HTTP/TCP/ICMP)
	Timestamp  time.Time // Time of probe execution
	Success    bool      // Whether probe succeeded
	Latency    int64     // Response time in milliseconds
	StatusCode int       // HTTP status code (HTTP only)
	Error      string    // Error message if failed
}

// Executor defines the interface for probe executors
type Executor interface {
	Execute(target string) *ProbeResult
}

// ProbeExecutor aggregates all probe types
type ProbeExecutor struct {
	httpProbe *HTTPProbe
	tcpProbe  *TCPProbe
	icmpProbe *ICMPProbe
}

// NewProbeExecutor creates a new probe executor with default settings
func NewProbeExecutor() *ProbeExecutor {
	return &ProbeExecutor{
		httpProbe: NewHTTPProbe(10 * time.Second),
		tcpProbe:  NewTCPProbe(5 * time.Second),
		icmpProbe: NewICMPProbe(5*time.Second, 1),
	}
}

// NewProbeExecutorWithTimeout creates a new probe executor with custom timeout
func NewProbeExecutorWithTimeout(timeout time.Duration) *ProbeExecutor {
	return &ProbeExecutor{
		httpProbe: NewHTTPProbe(timeout),
		tcpProbe:  NewTCPProbe(timeout),
		icmpProbe: NewICMPProbe(timeout, 1),
	}
}

// Execute performs a probe based on the probe type
func (e *ProbeExecutor) Execute(probeType ProbeType, target string) *ProbeResult {
	switch probeType {
	case ProbeTypeHTTP:
		return e.httpProbe.Execute(target, "GET")
	case ProbeTypeTCP:
		return e.tcpProbe.Execute(target)
	case ProbeTypeICMP:
		return e.icmpProbe.Execute(target)
	default:
		return &ProbeResult{
			Target:    target,
			Type:      string(probeType),
			Timestamp: time.Now(),
			Success:   false,
			Error:     "Unknown probe type",
		}
	}
}

// ExecuteHTTP performs an HTTP probe
func (e *ProbeExecutor) ExecuteHTTP(target string) *ProbeResult {
	return e.httpProbe.Execute(target, "GET")
}

// ExecuteTCP performs a TCP probe
func (e *ProbeExecutor) ExecuteTCP(target string) *ProbeResult {
	return e.tcpProbe.Execute(target)
}

// ExecuteICMP performs an ICMP probe
func (e *ProbeExecutor) ExecuteICMP(target string) *ProbeResult {
	return e.icmpProbe.Execute(target)
}

// ExecuteWithRetry performs a probe with retry logic
func (e *ProbeExecutor) ExecuteWithRetry(probeType ProbeType, target string, retries int) *ProbeResult {
	var lastResult *ProbeResult

	for i := 0; i <= retries; i++ {
		lastResult = e.Execute(probeType, target)
		if lastResult.Success {
			return lastResult
		}

		// Wait before retry
		if i < retries {
			time.Sleep(2 * time.Second)
		}
	}

	return lastResult
}
