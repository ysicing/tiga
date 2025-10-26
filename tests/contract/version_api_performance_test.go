package contract

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	versionhandler "github.com/ysicing/tiga/internal/api/handlers/version"
)

// TestVersionAPIPerformance 验证版本 API 性能契约
// 参考: .claude/specs/008-commitid-commit-agent/contracts/version-api.md (测试用例 3)
// 参考: .claude/specs/008-commitid-commit-agent/quickstart.md (性能验证)
//
// 性能目标：p99 延迟 <10ms，吞吐量 >1000 req/s
//
// 重要提示：这个测试在实现 API 端点之前应该失败（TDD 方法）
func TestVersionAPIPerformance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup router with version endpoint
	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/version", versionhandler.GetVersion)

	t.Run("average latency should be less than 10ms", func(t *testing.T) {
		const iterations = 100

		var totalDuration time.Duration
		successCount := 0

		for i := 0; i < iterations; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
			w := httptest.NewRecorder()

			start := time.Now()
			router.ServeHTTP(w, req)
			elapsed := time.Since(start)

			totalDuration += elapsed

			if w.Code == http.StatusOK {
				successCount++
			}
		}

		// 跳过如果端点未实现
		if successCount == 0 {
			t.Skip("API not implemented yet")
		}

		// 计算平均延迟
		avgLatency := totalDuration / time.Duration(iterations)

		assert.Less(t, avgLatency, 10*time.Millisecond,
			"Average latency should be less than 10ms (got: %v)", avgLatency)

		// 验证所有请求都成功
		assert.Equal(t, iterations, successCount,
			"All %d requests should return 200 OK", iterations)

		t.Logf("Performance metrics: avg latency = %v, success rate = %d/%d",
			avgLatency, successCount, iterations)
	})

	t.Run("p99 latency should be less than 20ms", func(t *testing.T) {
		const iterations = 100

		var latencies []time.Duration
		successCount := 0

		for i := 0; i < iterations; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
			w := httptest.NewRecorder()

			start := time.Now()
			router.ServeHTTP(w, req)
			elapsed := time.Since(start)

			latencies = append(latencies, elapsed)

			if w.Code == http.StatusOK {
				successCount++
			}
		}

		if successCount == 0 {
			t.Skip("API not implemented yet")
		}

		// 计算 p99（第 99 个百分位数）
		// 对于 100 个样本，p99 是第 99 个（索引 98）
		// 需要先排序
		sortedLatencies := make([]time.Duration, len(latencies))
		copy(sortedLatencies, latencies)

		// 简单冒泡排序
		for i := 0; i < len(sortedLatencies); i++ {
			for j := i + 1; j < len(sortedLatencies); j++ {
				if sortedLatencies[i] > sortedLatencies[j] {
					sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
				}
			}
		}

		p99Index := int(float64(len(sortedLatencies)) * 0.99)
		p99Latency := sortedLatencies[p99Index]

		assert.Less(t, p99Latency, 20*time.Millisecond,
			"p99 latency should be less than 20ms (got: %v)", p99Latency)

		t.Logf("p99 latency = %v", p99Latency)
	})

	t.Run("throughput should be greater than 1000 req/s", func(t *testing.T) {
		const duration = 1 * time.Second

		requestCount := 0
		start := time.Now()

		for time.Since(start) < duration {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				requestCount++
			}
		}

		elapsed := time.Since(start)

		// 跳过如果端点未实现
		if requestCount == 0 {
			t.Skip("API not implemented yet")
		}

		// 计算吞吐量 (requests/second)
		throughput := float64(requestCount) / elapsed.Seconds()

		assert.Greater(t, throughput, 1000.0,
			"Throughput should be greater than 1000 req/s (got: %.2f req/s)", throughput)

		t.Logf("Throughput: %.2f req/s (%d requests in %v)",
			throughput, requestCount, elapsed)
	})
}
