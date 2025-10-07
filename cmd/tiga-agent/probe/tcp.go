package probe

import (
	"fmt"
	"net"
	"time"
)

// TCPProbe performs TCP port connectivity checks
type TCPProbe struct {
	timeout time.Duration
}

// NewTCPProbe creates a new TCP probe executor
func NewTCPProbe(timeout time.Duration) *TCPProbe {
	return &TCPProbe{
		timeout: timeout,
	}
}

// Execute performs a TCP probe to the target host:port
func (p *TCPProbe) Execute(target string) *ProbeResult {
	result := &ProbeResult{
		Target:    target,
		Type:      "TCP",
		Timestamp: time.Now(),
	}

	start := time.Now()

	// Attempt TCP connection
	conn, err := net.DialTimeout("tcp", target, p.timeout)
	result.Latency = time.Since(start).Milliseconds()

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Connection failed: %v", err)
		return result
	}

	// Connection successful
	conn.Close()
	result.Success = true

	return result
}

// ExecuteWithRetry performs TCP probe with retry logic
func (p *TCPProbe) ExecuteWithRetry(target string, retries int) *ProbeResult {
	var lastResult *ProbeResult

	for i := 0; i <= retries; i++ {
		lastResult = p.Execute(target)
		if lastResult.Success {
			return lastResult
		}

		// Wait before retry
		if i < retries {
			time.Sleep(time.Second)
		}
	}

	return lastResult
}

// ExecuteBatch performs TCP probes to multiple targets concurrently
func (p *TCPProbe) ExecuteBatch(targets []string) []*ProbeResult {
	results := make([]*ProbeResult, len(targets))
	done := make(chan struct {
		index  int
		result *ProbeResult
	}, len(targets))

	// Execute probes concurrently
	for i, target := range targets {
		go func(index int, tgt string) {
			result := p.Execute(tgt)
			done <- struct {
				index  int
				result *ProbeResult
			}{index, result}
		}(i, target)
	}

	// Collect results
	for i := 0; i < len(targets); i++ {
		item := <-done
		results[item.index] = item.result
	}

	return results
}
