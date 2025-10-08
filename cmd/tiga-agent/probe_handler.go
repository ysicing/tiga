package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ysicing/tiga/cmd/tiga-agent/probe"
	"github.com/ysicing/tiga/proto"
)

// ProbeTaskHandler handles probe tasks sent by the server
type ProbeTaskHandler struct {
	httpProbe    *probe.HTTPProbe
	tcpProbe     *probe.TCPProbe
	icmpProbe    *probe.ICMPProbe
	resultBuffer *ProbeResultBuffer // Buffer for batch reporting
}

// NewProbeTaskHandler creates a new probe task handler
func NewProbeTaskHandler(uuid string, client proto.HostMonitorClient) *ProbeTaskHandler {
	return &ProbeTaskHandler{
		httpProbe:    probe.NewHTTPProbe(30 * time.Second),
		tcpProbe:     probe.NewTCPProbe(10 * time.Second),
		icmpProbe:    probe.NewICMPProbe(20*time.Second, 5),
		resultBuffer: NewProbeResultBuffer(uuid, client),
	}
}

// Start starts the probe handler and its buffer
func (h *ProbeTaskHandler) Start() {
	h.resultBuffer.Start()
}

// Stop stops the probe handler and flushes remaining results
func (h *ProbeTaskHandler) Stop() {
	h.resultBuffer.Stop()
}

// HandleProbeTask executes a probe task and reports the result
func (h *ProbeTaskHandler) HandleProbeTask(task *proto.AgentTask, config *Config) {
	logrus.Debugf("[ProbeTask] Handling probe task: %+v", task.Params)

	// Parse probe parameters
	probeType, ok := task.Params["type"]
	if !ok {
		logrus.Errorf("[ProbeTask] Missing 'type' parameter")
		return
	}

	target, ok := task.Params["target"]
	if !ok {
		logrus.Errorf("[ProbeTask] Missing 'target' parameter")
		return
	}

	monitorIDStr, ok := task.Params["monitor_id"]
	if !ok {
		logrus.Errorf("[ProbeTask] Missing 'monitor_id' parameter")
		return
	}

	// Execute probe based on type
	var result *probe.ProbeResult
	var err error

	switch probeType {
	case "http", "HTTP":
		method, _ := task.Params["method"]
		if method == "" {
			method = "GET"
		}
		result = h.executeHTTPProbe(target, method, task.Params)

	case "tcp", "TCP":
		result = h.executeTCPProbe(target)

	case "icmp", "ICMP":
		result = h.executeICMPProbe(target)

	default:
		logrus.Errorf("[ProbeTask] Unknown probe type: %s", probeType)
		return
	}

	if err != nil {
		logrus.Errorf("[ProbeTask] Probe execution failed: %v", err)
		return
	}

	// Report result back to server (using buffer for batch reporting)
	h.reportProbeResult(config.UUID, monitorIDStr, result)
}

// executeHTTPProbe executes HTTP probe with extended configuration
func (h *ProbeTaskHandler) executeHTTPProbe(target, method string, params map[string]string) *probe.ProbeResult {
	// Check if custom headers are provided
	if headersJSON, ok := params["headers"]; ok && headersJSON != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersJSON), &headers); err == nil && len(headers) > 0 {
			return h.httpProbe.ExecuteWithHeaders(target, headers)
		}
	}

	// Check if expected status is provided
	if expectedStatusStr, ok := params["expected_status"]; ok && expectedStatusStr != "" {
		if expectedStatus, err := strconv.Atoi(expectedStatusStr); err == nil {
			return h.httpProbe.ExecuteWithExpectedStatus(target, expectedStatus)
		}
	}

	// Standard HTTP probe
	return h.httpProbe.Execute(target, method)
}

// executeTCPProbe executes TCP probe with retry support
func (h *ProbeTaskHandler) executeTCPProbe(target string) *probe.ProbeResult {
	// Use retry logic for TCP probes
	return h.tcpProbe.ExecuteWithRetry(target, 2)
}

// executeICMPProbe executes ICMP probe
func (h *ProbeTaskHandler) executeICMPProbe(target string) *probe.ProbeResult {
	result, _ := h.icmpProbe.ExecuteWithPacketLoss(target)
	return result
}

// reportProbeResult adds probe result to buffer for batch reporting
func (h *ProbeTaskHandler) reportProbeResult(uuid, monitorID string, result *probe.ProbeResult) {
	// Convert probe.ProbeResult to proto.ProbeResult
	protoResult := &proto.ProbeResult{
		TaskId:    0, // Not used in this flow
		Timestamp: result.Timestamp.UnixMilli(),
		Success:   result.Success,
		Latency:   int32(result.Latency),
	}

	if result.Error != "" {
		protoResult.ErrorMessage = result.Error
	}

	// Add HTTP-specific fields
	if result.StatusCode > 0 {
		protoResult.HttpStatusCode = int32(result.StatusCode)
	}

	// Add to buffer (non-blocking, will be sent in batch)
	h.resultBuffer.Add(monitorID, protoResult)

	logrus.Debugf("[ProbeTask] Probe result buffered: monitor=%s, success=%v, latency=%dms",
		monitorID, result.Success, result.Latency)
}
