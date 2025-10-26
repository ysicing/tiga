# API 契约：版本信息查询

**端点**：`GET /api/v1/version`
**功能**：返回服务端的版本号、构建时间和 commit ID

## HTTP 请求

### 请求方法
```
GET /api/v1/version HTTP/1.1
Host: localhost:12306
Accept: application/json
```

### 请求头
| Header | 值 | 必需 | 说明 |
|--------|------|------|------|
| Accept | application/json | 否 | 期望的响应格式 |

### 请求参数
无（此端点不接受任何查询参数）

### 认证要求
无（此端点无需认证，便于监控系统访问）

## HTTP 响应

### 成功响应（200 OK）

**响应头**：
```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
Content-Length: 112
```

**响应体**：
```json
{
  "version": "v1.2.3-a1b2c3d",
  "build_time": "2025-10-26T10:30:00Z",
  "commit_id": "a1b2c3d"
}
```

**响应 Schema**：
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "version": {
      "type": "string",
      "description": "版本号（tag+commit 或 date+commit）",
      "pattern": "^(v?\\d+\\.\\d+\\.\\d+|\\d{8}|dev|snapshot)-[0-9a-f]{7}$|^(dev|snapshot)$",
      "examples": ["v1.2.3-a1b2c3d", "20251026-a1b2c3d", "dev", "snapshot"]
    },
    "build_time": {
      "type": "string",
      "description": "构建时间（RFC3339 格式）",
      "format": "date-time",
      "examples": ["2025-10-26T10:30:00Z", "unknown"]
    },
    "commit_id": {
      "type": "string",
      "description": "Git commit 短 hash（7位）",
      "pattern": "^[0-9a-f]{7}$|^0000000$",
      "examples": ["a1b2c3d", "0000000"]
    }
  },
  "required": ["version", "build_time", "commit_id"],
  "additionalProperties": false
}
```

### 错误响应

**此端点无错误响应**，因为：
- 版本信息始终可用（编译时注入）
- 无需认证，无401错误
- 无参数验证，无400错误
- 无资源查询，无404/500错误

## OpenAPI 规范

```yaml
openapi: 3.0.3
info:
  title: Tiga Version API
  version: 1.0.0
  description: 获取服务端版本信息

paths:
  /api/v1/version:
    get:
      summary: 获取服务端版本信息
      description: 返回服务端的版本号、构建时间和 commit ID
      operationId: getVersion
      tags:
        - system
      responses:
        '200':
          description: 成功返回版本信息
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/VersionInfo'
              examples:
                production:
                  summary: 生产环境版本
                  value:
                    version: v1.2.3-a1b2c3d
                    build_time: "2025-10-26T10:30:00Z"
                    commit_id: a1b2c3d
                development:
                  summary: 开发环境版本
                  value:
                    version: dev
                    build_time: unknown
                    commit_id: "0000000"

components:
  schemas:
    VersionInfo:
      type: object
      required:
        - version
        - build_time
        - commit_id
      properties:
        version:
          type: string
          description: 版本号（tag+commit 或 date+commit）
          pattern: ^(v?\d+\.\d+\.\d+|\d{8}|dev|snapshot)-[0-9a-f]{7}$|^(dev|snapshot)$
          example: v1.2.3-a1b2c3d
        build_time:
          type: string
          format: date-time
          description: 构建时间（RFC3339 格式）
          example: "2025-10-26T10:30:00Z"
        commit_id:
          type: string
          pattern: ^[0-9a-f]{7}$|^0000000$
          description: Git commit 短 hash（7位）
          example: a1b2c3d
```

## 契约测试

### 测试场景

**位置**：`tests/contract/version_api_test.go`

**测试用例**：

#### 1. 成功场景：返回版本信息
```go
func TestVersionAPI_Success(t *testing.T) {
    // Arrange: 启动测试服务器
    router := setupTestRouter()
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/version", nil)
    req.Header.Set("Accept", "application/json")

    // Act: 执行请求
    router.ServeHTTP(w, req)

    // Assert: 验证响应
    assert.Equal(t, 200, w.Code)
    assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

    var response map[string]string
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)

    // 验证必需字段存在
    assert.Contains(t, response, "version")
    assert.Contains(t, response, "build_time")
    assert.Contains(t, response, "commit_id")

    // 验证字段非空
    assert.NotEmpty(t, response["version"])
    assert.NotEmpty(t, response["build_time"])
    assert.NotEmpty(t, response["commit_id"])
}
```

#### 2. Schema 验证
```go
func TestVersionAPI_SchemaValidation(t *testing.T) {
    // Arrange & Act
    router := setupTestRouter()
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/version", nil)
    router.ServeHTTP(w, req)

    // Assert: 验证 JSON Schema
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)

    // 验证 version 格式
    version := response["version"].(string)
    versionPattern := regexp.MustCompile(`^(v?\d+\.\d+\.\d+|\d{8}|dev|snapshot)(-[0-9a-f]{7})?$`)
    assert.True(t, versionPattern.MatchString(version), "version格式不符合预期")

    // 验证 build_time 格式（RFC3339 或 "unknown"）
    buildTime := response["build_time"].(string)
    if buildTime != "unknown" {
        _, err := time.Parse(time.RFC3339, buildTime)
        assert.NoError(t, err, "build_time不是有效的RFC3339格式")
    }

    // 验证 commit_id 格式（7位十六进制或"0000000"）
    commitID := response["commit_id"].(string)
    commitPattern := regexp.MustCompile(`^[0-9a-f]{7}$`)
    assert.True(t, commitPattern.MatchString(commitID), "commit_id格式不符合预期")
}
```

#### 3. 性能测试
```go
func TestVersionAPI_Performance(t *testing.T) {
    router := setupTestRouter()

    // 测试延迟 <10ms
    start := time.Now()
    for i := 0; i < 100; i++ {
        w := httptest.NewRecorder()
        req, _ := http.NewRequest("GET", "/api/v1/version", nil)
        router.ServeHTTP(w, req)
        assert.Equal(t, 200, w.Code)
    }
    elapsed := time.Since(start)
    avgLatency := elapsed / 100

    assert.Less(t, avgLatency, 10*time.Millisecond, "平均延迟应小于10ms")
}
```

#### 4. 响应体大小
```go
func TestVersionAPI_ResponseSize(t *testing.T) {
    router := setupTestRouter()
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/version", nil)
    router.ServeHTTP(w, req)

    // 验证响应体大小 <500 bytes（包含headers）
    responseSize := w.Body.Len()
    assert.Less(t, responseSize, 500, "响应体应小于500 bytes")
}
```

## 前端集成示例

### TypeScript 类型定义

**位置**：`ui/src/types/version.ts`

```typescript
export interface VersionInfo {
  version: string;      // v1.2.3-a1b2c3d | 20251026-a1b2c3d | dev
  build_time: string;   // RFC3339 格式 | "unknown"
  commit_id: string;    // 7位十六进制
}
```

### API 客户端

**位置**：`ui/src/services/version.ts`

```typescript
import axios from 'axios';
import { VersionInfo } from '../types/version';

export const versionAPI = {
  /**
   * 获取服务端版本信息
   * @returns Promise<VersionInfo>
   */
  async getVersion(): Promise<VersionInfo> {
    const response = await axios.get<VersionInfo>('/api/v1/version');
    return response.data;
  }
};
```

### React 组件使用

```typescript
import { useQuery } from '@tanstack/react-query';
import { versionAPI } from '../services/version';

export function VersionDisplay() {
  const { data: version, isLoading } = useQuery({
    queryKey: ['version'],
    queryFn: versionAPI.getVersion,
    staleTime: 5 * 60 * 1000, // 5分钟缓存
  });

  if (isLoading) return <div>Loading...</div>;

  return (
    <div className="version-info">
      <p>Version: {version?.version}</p>
      <p>Build Time: {new Date(version?.build_time).toLocaleString()}</p>
      <p>Commit: {version?.commit_id}</p>
    </div>
  );
}
```

## 监控集成

### cURL 示例
```bash
# 获取版本信息
curl -X GET http://localhost:12306/api/v1/version

# 提取特定字段
curl -s http://localhost:12306/api/v1/version | jq -r '.version'
```

### Prometheus 监控（可选）

虽然版本信息本身不需要监控，但可以暴露为 Prometheus metric：

```go
// 可选：将版本信息暴露为 Prometheus metric
var versionInfo = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "tiga_build_info",
        Help: "Tiga build information",
    },
    []string{"version", "build_time", "commit_id"},
)

func init() {
    versionInfo.WithLabelValues(
        version.Version,
        version.BuildTime,
        version.CommitID,
    ).Set(1)
}
```

## 变更历史

| 日期 | 版本 | 描述 |
|------|------|------|
| 2025-10-26 | 1.0.0 | 初始版本 - 添加版本信息API |

## 注意事项

1. **无需认证**：此端点无需认证，便于监控系统访问
2. **缓存策略**：前端建议缓存5分钟，因为版本信息不会频繁变化
3. **错误处理**：此端点无错误响应，始终返回 200 OK
4. **性能**：目标 <10ms p99 延迟（实际远低于此值）
