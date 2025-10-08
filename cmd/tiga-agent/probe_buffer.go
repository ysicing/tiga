package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/ysicing/tiga/proto"
)

// ProbeResultBuffer accumulates probe results and sends them in batches
// to reduce gRPC overhead and improve performance
type ProbeResultBuffer struct {
	uuid          string
	client        proto.HostMonitorClient
	batchSize     int
	flushInterval time.Duration

	buffer   []*proto.ProbeResultItem
	bufferMu sync.Mutex

	resultChan chan *proto.ProbeResultItem
	stopChan   chan struct{}
	wg         sync.WaitGroup

	// Performance metrics
	totalReceived uint64
	totalReported uint64
	totalFailed   uint64
}

// NewProbeResultBuffer creates a new probe result buffer
func NewProbeResultBuffer(uuid string, client proto.HostMonitorClient) *ProbeResultBuffer {
	return &ProbeResultBuffer{
		uuid:          uuid,
		client:        client,
		batchSize:     10,                 // Flush after 10 results
		flushInterval: 10 * time.Second,   // Or flush every 10 seconds
		buffer:        make([]*proto.ProbeResultItem, 0, 10),
		resultChan:    make(chan *proto.ProbeResultItem, 100),
		stopChan:      make(chan struct{}),
	}
}

// Start starts the buffer worker
func (b *ProbeResultBuffer) Start() {
	b.wg.Add(1)
	go b.worker()
}

// Stop stops the buffer and flushes remaining results
func (b *ProbeResultBuffer) Stop() {
	close(b.stopChan)
	b.wg.Wait()

	// Final flush
	b.flush()

	logrus.Infof("[ProbeBuffer] Stopped. Stats: received=%d, reported=%d, failed=%d",
		atomic.LoadUint64(&b.totalReceived),
		atomic.LoadUint64(&b.totalReported),
		atomic.LoadUint64(&b.totalFailed))
}

// Add adds a probe result to the buffer (non-blocking)
func (b *ProbeResultBuffer) Add(serviceMonitorID string, result *proto.ProbeResult) {
	item := &proto.ProbeResultItem{
		ServiceMonitorId: serviceMonitorID,
		Result:           result,
	}

	atomic.AddUint64(&b.totalReceived, 1)

	select {
	case b.resultChan <- item:
		// Successfully queued
	case <-b.stopChan:
		// Buffer stopped, discard result
		logrus.Warnf("[ProbeBuffer] Buffer stopped, discarding result")
	default:
		// Channel full, drop result
		logrus.Warnf("[ProbeBuffer] Channel full, dropping probe result")
		atomic.AddUint64(&b.totalFailed, 1)
	}
}

// worker processes incoming results and triggers batch sends
func (b *ProbeResultBuffer) worker() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopChan:
			logrus.Debug("[ProbeBuffer] Worker stopping")
			return

		case result := <-b.resultChan:
			b.bufferMu.Lock()
			b.buffer = append(b.buffer, result)
			shouldFlush := len(b.buffer) >= b.batchSize
			b.bufferMu.Unlock()

			if shouldFlush {
				b.flush()
			}

		case <-ticker.C:
			b.flush()
		}
	}
}

// flush sends buffered results to server
func (b *ProbeResultBuffer) flush() {
	b.bufferMu.Lock()
	if len(b.buffer) == 0 {
		b.bufferMu.Unlock()
		return
	}

	// Swap buffer to avoid holding lock during network I/O
	toSend := b.buffer
	b.buffer = make([]*proto.ProbeResultItem, 0, b.batchSize)
	b.bufferMu.Unlock()

	// Send batch to server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &proto.ReportProbeResultBatchRequest{
		Uuid:    b.uuid,
		Results: toSend,
	}

	resp, err := b.client.ReportProbeResultBatch(ctx, req)
	if err != nil {
		logrus.Errorf("[ProbeBuffer] Failed to send batch: %v", err)
		atomic.AddUint64(&b.totalFailed, uint64(len(toSend)))
		return
	}

	if !resp.Success {
		logrus.Warnf("[ProbeBuffer] Server rejected batch: %s", resp.Message)
		atomic.AddUint64(&b.totalFailed, uint64(len(toSend)))
		return
	}

	// Update statistics
	atomic.AddUint64(&b.totalReported, uint64(resp.Processed))
	if resp.Failed > 0 {
		atomic.AddUint64(&b.totalFailed, uint64(resp.Failed))
	}

	logrus.Debugf("[ProbeBuffer] Batch sent successfully: total=%d, processed=%d, failed=%d",
		len(toSend), resp.Processed, resp.Failed)
}

// GetStats returns current buffer statistics
func (b *ProbeResultBuffer) GetStats() (received, reported, failed uint64) {
	return atomic.LoadUint64(&b.totalReceived),
		atomic.LoadUint64(&b.totalReported),
		atomic.LoadUint64(&b.totalFailed)
}
