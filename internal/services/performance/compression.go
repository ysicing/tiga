package performance

import (
	"compress/gzip"
	"io"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// CompressionConfig holds compression configuration
type CompressionConfig struct {
	Level         int      // Compression level (1-9)
	MinSize       int      // Minimum response size to compress (bytes)
	ContentTypes  []string // Content types to compress
	ExcludedPaths []string // Paths to exclude from compression
}

// DefaultCompressionConfig returns default compression configuration
func DefaultCompressionConfig() *CompressionConfig {
	return &CompressionConfig{
		Level:   gzip.DefaultCompression,
		MinSize: 1024, // 1KB
		ContentTypes: []string{
			"application/json",
			"application/javascript",
			"application/xml",
			"text/html",
			"text/plain",
			"text/css",
			"text/xml",
		},
		ExcludedPaths: []string{},
	}
}

// gzipWriterPool is a pool of gzip writers
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

// CompressionMiddleware creates a middleware for response compression
func CompressionMiddleware(config *CompressionConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCompressionConfig()
	}

	return func(c *gin.Context) {
		// Check if client accepts gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Check if path is excluded
		for _, path := range config.ExcludedPaths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		// Get gzip writer from pool
		gz := gzipWriterPool.Get().(*gzip.Writer)
		defer gzipWriterPool.Put(gz)

		// Reset writer
		gz.Reset(c.Writer)
		defer gz.Close()

		// Wrap response writer
		c.Writer = &gzipWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
			minSize:        config.MinSize,
			contentTypes:   config.ContentTypes,
		}

		// Set encoding header
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		c.Next()
	}
}

// gzipWriter wraps gin.ResponseWriter with gzip compression
type gzipWriter struct {
	gin.ResponseWriter
	writer         *gzip.Writer
	minSize        int
	contentTypes   []string
	written        int
	shouldCompress bool
}

// Write compresses data before writing
func (g *gzipWriter) Write(data []byte) (int, error) {
	// Check if we should compress based on content type
	if !g.shouldCompress && g.written == 0 {
		contentType := g.Header().Get("Content-Type")
		g.shouldCompress = g.isCompressible(contentType)
	}

	g.written += len(data)

	// Only compress if size threshold is met and content is compressible
	if g.shouldCompress && g.written >= g.minSize {
		return g.writer.Write(data)
	}

	// Write uncompressed
	return g.ResponseWriter.Write(data)
}

// WriteString writes string data
func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.Write([]byte(s))
}

// isCompressible checks if content type should be compressed
func (g *gzipWriter) isCompressible(contentType string) bool {
	for _, ct := range g.contentTypes {
		if strings.Contains(contentType, ct) {
			return true
		}
	}
	return false
}

// CompressionStats tracks compression statistics
type CompressionStats struct {
	TotalRequests      int64   `json:"total_requests"`
	CompressedRequests int64   `json:"compressed_requests"`
	CompressionRate    float64 `json:"compression_rate"`
	BytesSaved         int64   `json:"bytes_saved"`
	OriginalSize       int64   `json:"original_size"`
	CompressedSize     int64   `json:"compressed_size"`
	AverageRatio       float64 `json:"average_ratio"`
}

// CompressionMonitor monitors compression statistics
type CompressionMonitor struct {
	stats CompressionStats
	mu    sync.RWMutex
}

// NewCompressionMonitor creates a new compression monitor
func NewCompressionMonitor() *CompressionMonitor {
	return &CompressionMonitor{
		stats: CompressionStats{},
	}
}

// RecordCompression records compression statistics
func (cm *CompressionMonitor) RecordCompression(originalSize, compressedSize int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.stats.TotalRequests++
	if compressedSize < originalSize {
		cm.stats.CompressedRequests++
		cm.stats.BytesSaved += (originalSize - compressedSize)
	}

	cm.stats.OriginalSize += originalSize
	cm.stats.CompressedSize += compressedSize

	// Update rates
	if cm.stats.TotalRequests > 0 {
		cm.stats.CompressionRate = float64(cm.stats.CompressedRequests) / float64(cm.stats.TotalRequests)
	}

	if cm.stats.OriginalSize > 0 {
		cm.stats.AverageRatio = float64(cm.stats.CompressedSize) / float64(cm.stats.OriginalSize)
	}
}

// GetStats retrieves compression statistics
func (cm *CompressionMonitor) GetStats() CompressionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.stats
}

// Reset resets compression statistics
func (cm *CompressionMonitor) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.stats = CompressionStats{}
}

// CompressResponse manually compresses a response
func CompressResponse(data []byte, level int) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	var buf strings.Builder
	writer, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// DecompressResponse manually decompresses a response
func DecompressResponse(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// AdaptiveCompressionMiddleware creates middleware with adaptive compression
func AdaptiveCompressionMiddleware() gin.HandlerFunc {
	monitor := NewCompressionMonitor()

	return func(c *gin.Context) {
		// Check if client accepts gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Capture original response
		originalWriter := c.Writer
		buf := &responseBuffer{ResponseWriter: c.Writer}
		c.Writer = buf

		c.Next()

		// Get response data
		data := []byte(buf.body.String())
		originalSize := int64(len(data))

		// Decide whether to compress based on size and content type
		contentType := c.Writer.Header().Get("Content-Type")
		shouldCompress := originalSize > 1024 && isCompressibleContentType(contentType)

		if shouldCompress {
			// Compress response
			compressed, err := CompressResponse(data, gzip.DefaultCompression)
			if err == nil && len(compressed) < len(data) {
				// Use compressed version
				c.Writer = originalWriter
				c.Header("Content-Encoding", "gzip")
				c.Header("Vary", "Accept-Encoding")
				c.Writer.Write(compressed)

				// Record statistics
				monitor.RecordCompression(originalSize, int64(len(compressed)))
				return
			}
		}

		// Use original uncompressed version
		c.Writer = originalWriter
		c.Writer.Write(data)
		monitor.RecordCompression(originalSize, originalSize)
	}
}

// responseBuffer buffers response for compression decision
type responseBuffer struct {
	gin.ResponseWriter
	body   strings.Builder
	status int
}

func (rb *responseBuffer) Write(data []byte) (int, error) {
	return rb.body.Write(data)
}

func (rb *responseBuffer) WriteHeader(status int) {
	rb.status = status
	rb.ResponseWriter.WriteHeader(status)
}

// isCompressibleContentType checks if content type is compressible
func isCompressibleContentType(contentType string) bool {
	compressible := []string{
		"application/json",
		"application/javascript",
		"application/xml",
		"text/",
	}

	for _, ct := range compressible {
		if strings.Contains(contentType, ct) {
			return true
		}
	}

	return false
}

// SelectiveCompressionMiddleware compresses only specific endpoints
func SelectiveCompressionMiddleware(endpoints map[string]bool) gin.HandlerFunc {
	config := DefaultCompressionConfig()

	return func(c *gin.Context) {
		endpoint := c.FullPath()

		// Check if endpoint should be compressed
		if shouldCompress, exists := endpoints[endpoint]; !exists || !shouldCompress {
			c.Next()
			return
		}

		// Apply compression
		CompressionMiddleware(config)(c)
	}
}

// BrotliMiddleware creates a middleware for Brotli compression (placeholder)
// Note: Requires github.com/andybalholm/brotli for actual implementation
func BrotliMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client accepts br (Brotli)
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "br") {
			c.Next()
			return
		}

		// Placeholder for Brotli compression
		// Real implementation would use brotli encoder
		c.Next()
	}
}

// MultiCompressionMiddleware supports multiple compression algorithms
func MultiCompressionMiddleware(config *CompressionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptEncoding := c.GetHeader("Accept-Encoding")

		// Prefer Brotli if available
		if strings.Contains(acceptEncoding, "br") {
			// Use Brotli (placeholder)
			c.Next()
			return
		}

		// Fall back to gzip
		if strings.Contains(acceptEncoding, "gzip") {
			CompressionMiddleware(config)(c)
			return
		}

		// No compression
		c.Next()
	}
}
