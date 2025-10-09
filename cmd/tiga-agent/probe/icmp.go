package probe

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ICMPProbe performs ICMP ping checks
type ICMPProbe struct {
	timeout time.Duration
	count   int
}

// NewICMPProbe creates a new ICMP probe executor
func NewICMPProbe(timeout time.Duration, count int) *ICMPProbe {
	if count <= 0 {
		count = 1
	}
	return &ICMPProbe{
		timeout: timeout,
		count:   count,
	}
}

// Execute performs an ICMP ping to the target host
// Uses system ping command for cross-platform compatibility
func (p *ICMPProbe) Execute(target string) *ProbeResult {
	result := &ProbeResult{
		Target:    target,
		Type:      "ICMP",
		Timestamp: time.Now(),
	}

	start := time.Now()

	// Build ping command based on OS
	cmd := p.buildPingCommand(target)

	// Execute ping
	output, err := cmd.CombinedOutput()
	result.Latency = time.Since(start).Milliseconds()

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Ping failed: %v", err)
		return result
	}

	// Parse ping output to extract RTT
	rtt := p.parseRTT(string(output))
	if rtt > 0 {
		result.Latency = rtt
	}

	result.Success = true
	return result
}

// buildPingCommand builds the appropriate ping command for the OS
func (p *ICMPProbe) buildPingCommand(target string) *exec.Cmd {
	count := strconv.Itoa(p.count)
	timeout := strconv.Itoa(int(p.timeout.Seconds()))

	switch runtime.GOOS {
	case "windows":
		// ping -n count -w timeout_ms target
		return exec.Command("ping", "-n", count, "-w", strconv.Itoa(int(p.timeout.Milliseconds())), target)
	case "darwin":
		// ping -c count -W timeout_ms target
		return exec.Command("ping", "-c", count, "-W", strconv.Itoa(int(p.timeout.Milliseconds())), target)
	default: // linux
		// ping -c count -W timeout_sec target
		return exec.Command("ping", "-c", count, "-W", timeout, target)
	}
}

// parseRTT extracts RTT (Round Trip Time) from ping output
func (p *ICMPProbe) parseRTT(output string) int64 {
	// Common patterns across different OS ping outputs
	patterns := []string{
		"time=",     // Linux/macOS
		"Average =", // Windows
	}

	for _, pattern := range patterns {
		if idx := strings.Index(output, pattern); idx != -1 {
			// Extract the number after the pattern
			start := idx + len(pattern)
			end := start
			for end < len(output) && (output[end] >= '0' && output[end] <= '9' || output[end] == '.') {
				end++
			}
			if end > start {
				if rtt, err := strconv.ParseFloat(output[start:end], 64); err == nil {
					return int64(rtt)
				}
			}
		}
	}

	return 0
}

// ExecuteWithPacketLoss performs ping and returns packet loss percentage
func (p *ICMPProbe) ExecuteWithPacketLoss(target string) (*ProbeResult, int) {
	result := p.Execute(target)

	// Simple packet loss calculation (requires multiple pings)
	// For now, return 0 for success, 100 for failure
	packetLoss := 0
	if !result.Success {
		packetLoss = 100
	}

	return result, packetLoss
}
