# REST API契约：Docker实例远程管理

**功能分支**：`007-docker-docker-agent`
**创建日期**：2025-10-22
**API版本**：v1
**状态**：草稿

---

## 概述

Docker实例远程管理功能通过RESTful API对外提供服务，所有Docker操作通过Agent转发执行。

**基础路径**：`/api/v1/docker`

**认证方式**：JWT Bearer Token
```http
Authorization: Bearer <token>
```

**权限级别**（FR-023）：
- **Viewer（只读）**：可查看Docker实例、容器、镜像信息和日志
- **Operator（操作员）**：可执行容器生命周期操作（启动、停止、重启），不可删除
- **Admin（管理员）**：可执行所有操作，包括删除容器、删除镜像、修改实例配置

---

## 1. Docker实例管理

### 1.1 获取Docker实例列表

**端点**：`GET /api/v1/docker/instances`

**权限**：Viewer

**查询参数**：
- `name` (string, optional): 名称模糊搜索
- `health_status` (string, optional): 健康状态过滤（online/offline/archived）
- `agent_id` (string, optional): Agent ID过滤
- `host_id` (string, optional): 主机ID过滤
- `tags` (string, optional): 标签过滤（逗号分隔）
- `page` (integer, optional): 页码，默认1
- `page_size` (integer, optional): 每页条数，默认20，最大100
- `sort_by` (string, optional): 排序字段（name/created_at/last_connected_at），默认created_at
- `sort_order` (string, optional): 排序方式（asc/desc），默认desc

**响应示例**：
```json
{
  "success": true,
  "data": {
    "instances": [
      {
        "id": "uuid",
        "name": "prod-docker-1",
        "description": "生产环境Docker实例",
        "agent_id": "agent-uuid",
        "host_id": "host-uuid",
        "host_name": "prod-server-1",
        "health_status": "online",
        "last_connected_at": "2025-10-22T10:30:00Z",
        "last_health_check": "2025-10-22T10:35:00Z",
        "docker_version": "24.0.7",
        "api_version": "1.43",
        "storage_driver": "overlay2",
        "operating_system": "Ubuntu 22.04",
        "architecture": "x86_64",
        "container_count": 15,
        "image_count": 8,
        "volume_count": 5,
        "network_count": 3,
        "tags": ["production", "web"],
        "created_at": "2025-10-01T08:00:00Z",
        "updated_at": "2025-10-22T10:35:00Z"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

**错误响应**：
```json
{
  "success": false,
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid page_size: maximum is 100"
  }
}
```

---

### 1.2 获取Docker实例详情

**端点**：`GET /api/v1/docker/instances/:id`

**权限**：Viewer

**路径参数**：
- `id` (string, required): Docker实例ID

**响应示例**：
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "prod-docker-1",
    "description": "生产环境Docker实例",
    "agent_id": "agent-uuid",
    "host_id": "host-uuid",
    "host_name": "prod-server-1",
    "health_status": "online",
    "last_connected_at": "2025-10-22T10:30:00Z",
    "last_health_check": "2025-10-22T10:35:00Z",
    "docker_version": "24.0.7",
    "api_version": "1.43",
    "storage_driver": "overlay2",
    "operating_system": "Ubuntu 22.04",
    "architecture": "x86_64",
    "kernel_version": "5.15.0-56-generic",
    "mem_total": 17179869184,
    "n_cpu": 8,
    "container_count": 15,
    "image_count": 8,
    "volume_count": 5,
    "network_count": 3,
    "tags": ["production", "web"],
    "created_at": "2025-10-01T08:00:00Z",
    "updated_at": "2025-10-22T10:35:00Z"
  }
}
```

**错误码**：
- `404 NOT_FOUND`: Docker实例不存在

---

### 1.3 创建Docker实例（手动注册）

**端点**：`POST /api/v1/docker/instances`

**权限**：Admin

**请求体**：
```json
{
  "name": "prod-docker-1",
  "description": "生产环境Docker实例",
  "agent_id": "agent-uuid",
  "tags": ["production", "web"]
}
```

**验证规则**：
- `name`: 必填，1-255字符，只允许字母、数字、下划线、中划线
- `agent_id`: 必填，有效的Agent UUID
- `tags`: 可选，每个标签1-50字符

**响应示例**：
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "prod-docker-1",
    "description": "生产环境Docker实例",
    "agent_id": "agent-uuid",
    "health_status": "unknown",
    "tags": ["production", "web"],
    "created_at": "2025-10-22T10:40:00Z",
    "updated_at": "2025-10-22T10:40:00Z"
  }
}
```

**错误码**：
- `400 INVALID_PARAMETER`: 参数验证失败
- `409 CONFLICT`: 实例名称已存在（同Agent下）
- `404 NOT_FOUND`: Agent不存在

---

### 1.4 更新Docker实例

**端点**：`PUT /api/v1/docker/instances/:id`

**权限**：Admin

**请求体**：
```json
{
  "name": "prod-docker-1-updated",
  "description": "更新后的描述",
  "tags": ["production", "web", "updated"]
}
```

**响应示例**：
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "prod-docker-1-updated",
    "description": "更新后的描述",
    "tags": ["production", "web", "updated"],
    "updated_at": "2025-10-22T10:45:00Z"
  }
}
```

**错误码**：
- `404 NOT_FOUND`: Docker实例不存在
- `409 CONFLICT`: 实例名称已存在

---

### 1.5 删除Docker实例

**端点**：`DELETE /api/v1/docker/instances/:id`

**权限**：Admin

**响应示例**：
```json
{
  "success": true,
  "message": "Docker实例已删除"
}
```

**错误码**：
- `404 NOT_FOUND`: Docker实例不存在

---

### 1.6 测试Docker实例连接

**端点**：`POST /api/v1/docker/instances/:id/test-connection`

**权限**：Viewer

**响应示例**（成功）：
```json
{
  "success": true,
  "data": {
    "connected": true,
    "docker_version": "24.0.7",
    "api_version": "1.43",
    "latency": 120
  }
}
```

**响应示例**（失败）：
```json
{
  "success": false,
  "error": {
    "code": "CONNECTION_FAILED",
    "message": "Failed to connect to Docker daemon: connection timeout"
  }
}
```

---

## 2. 容器管理

### 2.1 获取容器列表

**端点**：`GET /api/v1/docker/instances/:id/containers`

**权限**：Viewer

**查询参数**：
- `all` (boolean, optional): 是否包含已停止容器，默认true
- `page` (integer, optional): 页码，默认1
- `page_size` (integer, optional): 每页条数，默认50，最大1000
- `filter` (string, optional): Docker原生filter（name=nginx, status=running）
- `sort_by` (string, optional): 排序字段（created/name/status）
- `sort_order` (string, optional): 排序方式（asc/desc）

**响应示例**：
```json
{
  "success": true,
  "data": {
    "containers": [
      {
        "id": "abc123def456",
        "name": "nginx-web",
        "image": "nginx:latest",
        "image_id": "sha256:...",
        "state": "running",
        "status": "Up 2 hours",
        "created_at": "2025-10-22T08:00:00Z",
        "started_at": "2025-10-22T08:00:30Z",
        "ports": [
          {
            "ip": "0.0.0.0",
            "private_port": 80,
            "public_port": 8080,
            "type": "tcp"
          }
        ],
        "mounts": [
          {
            "type": "volume",
            "source": "nginx-data",
            "destination": "/usr/share/nginx/html",
            "mode": "rw",
            "rw": true
          }
        ],
        "networks": {
          "bridge": {
            "network_id": "network-id",
            "ip_address": "172.17.0.2",
            "gateway": "172.17.0.1"
          }
        },
        "labels": {
          "com.example.app": "web"
        },
        "restart_count": 0,
        "restart_policy": "always"
      }
    ],
    "total": 15,
    "page": 1,
    "page_size": 50
  }
}
```

**错误码**：
- `404 NOT_FOUND`: Docker实例不存在
- `503 SERVICE_UNAVAILABLE`: Docker实例离线

---

### 2.2 获取容器详情

**端点**：`GET /api/v1/docker/instances/:id/containers/:container_id`

**权限**：Viewer

**响应示例**：包含完整容器信息（同列表，但包含env、command等完整字段）

---

### 2.3 启动容器

**端点**：`POST /api/v1/docker/instances/:id/containers/:container_id/start`

**权限**：Operator

**响应示例**：
```json
{
  "success": true,
  "message": "容器启动成功",
  "data": {
    "container_id": "abc123def456",
    "duration": 1250
  }
}
```

**错误码**：
- `404 NOT_FOUND`: 容器不存在
- `400 BAD_REQUEST`: 容器已在运行
- `503 SERVICE_UNAVAILABLE`: Docker实例离线

---

### 2.4 停止容器

**端点**：`POST /api/v1/docker/instances/:id/containers/:container_id/stop`

**权限**：Operator

**请求体**（可选）：
```json
{
  "timeout": 10
}
```

**响应示例**：同启动容器

---

### 2.5 重启容器

**端点**：`POST /api/v1/docker/instances/:id/containers/:container_id/restart`

**权限**：Operator

**请求体**（可选）：
```json
{
  "timeout": 10
}
```

---

### 2.6 暂停容器

**端点**：`POST /api/v1/docker/instances/:id/containers/:container_id/pause`

**权限**：Operator

---

### 2.7 恢复容器

**端点**：`POST /api/v1/docker/instances/:id/containers/:container_id/unpause`

**权限**：Operator

---

### 2.8 删除容器

**端点**：`DELETE /api/v1/docker/instances/:id/containers/:container_id`

**权限**：Admin

**查询参数**：
- `force` (boolean, optional): 强制删除（停止并删除），默认false
- `remove_volumes` (boolean, optional): 删除关联的匿名卷，默认false

**响应示例**：
```json
{
  "success": true,
  "message": "容器已删除",
  "data": {
    "container_id": "abc123def456"
  }
}
```

---

### 2.9 获取容器日志

**端点**：`GET /api/v1/docker/instances/:id/containers/:container_id/logs`

**权限**：Viewer

**查询参数**：
- `follow` (boolean, optional): 流式跟踪日志，默认false
- `tail` (integer, optional): 最后N行，默认100，最大10000
- `since` (integer, optional): Unix时间戳，获取该时间之后的日志
- `timestamps` (boolean, optional): 包含时间戳，默认true

**响应类型**：
- `follow=false`: JSON响应
- `follow=true`: Server-Sent Events（SSE）流

**JSON响应示例**：
```json
{
  "success": true,
  "data": {
    "logs": [
      "2025-10-22T10:00:00Z [INFO] Server started on port 8080",
      "2025-10-22T10:00:01Z [INFO] Connected to database",
      "2025-10-22T10:00:02Z [INFO] Ready to accept connections"
    ],
    "total_lines": 3
  }
}
```

**SSE流示例**：
```
Content-Type: text/event-stream

data: 2025-10-22T10:00:00Z [INFO] Server started on port 8080

data: 2025-10-22T10:00:01Z [INFO] Connected to database

data: 2025-10-22T10:00:02Z [INFO] Ready to accept connections
```

---

### 2.10 获取容器统计信息

**端点**：`GET /api/v1/docker/instances/:id/containers/:container_id/stats`

**权限**：Viewer

**查询参数**：
- `stream` (boolean, optional): 流式推送，默认false

**响应类型**：
- `stream=false`: JSON响应（单次查询）
- `stream=true`: SSE流（每秒推送一次）

**JSON响应示例**：
```json
{
  "success": true,
  "data": {
    "container_id": "abc123def456",
    "timestamp": 1729590000,
    "cpu_usage_percent": 12.5,
    "cpu_usage_nano": 1250000000,
    "memory_usage": 134217728,
    "memory_limit": 536870912,
    "memory_usage_percent": 25.0,
    "network_rx_bytes": 1048576,
    "network_tx_bytes": 524288,
    "block_read_bytes": 2097152,
    "block_write_bytes": 1048576,
    "pids_current": 5
  }
}
```

---

## 3. 镜像管理

### 3.1 获取镜像列表

**端点**：`GET /api/v1/docker/instances/:id/images`

**权限**：Viewer

**查询参数**：
- `all` (boolean, optional): 包含中间层镜像，默认false
- `filter` (string, optional): Docker原生filter（dangling=true, reference=nginx）

**响应示例**：
```json
{
  "success": true,
  "data": {
    "images": [
      {
        "id": "sha256:abc123",
        "repo_tags": ["nginx:latest", "nginx:1.25"],
        "repo_digests": ["nginx@sha256:..."],
        "size": 142606336,
        "virtual_size": 142606336,
        "created_at": "2025-09-15T10:00:00Z",
        "labels": {
          "maintainer": "NGINX Docker Maintainers"
        },
        "layers": ["sha256:layer1", "sha256:layer2"]
      }
    ],
    "total": 8
  }
}
```

---

### 3.2 获取镜像详情

**端点**：`GET /api/v1/docker/instances/:id/images/:image_id`

**权限**：Viewer

**响应示例**：包含完整镜像信息（config、history等）

---

### 3.3 删除镜像

**端点**：`DELETE /api/v1/docker/instances/:id/images/:image_id`

**权限**：Admin

**查询参数**：
- `force` (boolean, optional): 强制删除，默认false
- `no_prune` (boolean, optional): 不删除未标记的父镜像，默认false

**响应示例**：
```json
{
  "success": true,
  "message": "镜像已删除",
  "data": {
    "deleted": ["sha256:abc123"],
    "untagged": ["nginx:latest"]
  }
}
```

**错误码**：
- `404 NOT_FOUND`: 镜像不存在
- `409 CONFLICT`: 镜像被容器使用，无法删除

---

### 3.4 拉取镜像

**端点**：`POST /api/v1/docker/instances/:id/images/pull`

**权限**：Admin

**请求体**：
```json
{
  "image_name": "nginx:latest",
  "registry_auth": {
    "username": "user",
    "password": "pass"
  },
  "platform": "linux/amd64"
}
```

**响应类型**：SSE流（实时推送拉取进度）

**SSE流示例**：
```
data: {"status":"Pulling from library/nginx","progress":""}

data: {"status":"Pulling fs layer","id":"sha256:layer1"}

data: {"status":"Downloading","id":"sha256:layer1","progress":"50%","current":1048576,"total":2097152}

data: {"status":"Download complete","id":"sha256:layer1"}

data: {"status":"Pull complete"}
```

---

### 3.5 给镜像打标签

**端点**：`POST /api/v1/docker/instances/:id/images/:image_id/tag`

**权限**：Admin

**请求体**：
```json
{
  "target_repo": "myregistry.com/nginx",
  "target_tag": "v1.0"
}
```

**响应示例**：
```json
{
  "success": true,
  "message": "镜像标签已创建",
  "data": {
    "source_image": "nginx:latest",
    "target_image": "myregistry.com/nginx:v1.0"
  }
}
```

---

## 4. 审计日志

### 4.1 查询Docker操作日志

**端点**：`GET /api/v1/docker/audit-logs`

**权限**：Viewer（仅查看自己的日志），Admin（查看所有日志）

**查询参数**：
- `user_id` (string, optional): 用户ID过滤
- `action` (string, optional): 操作类型过滤（container_start, container_stop等）
- `resource_type` (string, optional): 资源类型（docker_container, docker_image）
- `instance_id` (string, optional): 实例ID过滤
- `start_time` (string, optional): 开始时间（ISO8601格式）
- `end_time` (string, optional): 结束时间
- `page` (integer, optional): 页码
- `page_size` (integer, optional): 每页条数，默认50

**响应示例**：
```json
{
  "success": true,
  "data": {
    "logs": [
      {
        "id": "uuid",
        "user_id": "user-uuid",
        "username": "admin",
        "action": "container_stop",
        "resource_type": "docker_container",
        "resource_id": "abc123def456",
        "resource_name": "nginx-web",
        "details": {
          "instance_id": "instance-uuid",
          "instance_name": "prod-docker-1",
          "state_before": "running",
          "state_after": "exited",
          "success": true,
          "duration": 1250
        },
        "ip_address": "192.168.1.100",
        "timestamp": "2025-10-22T10:30:00Z"
      }
    ],
    "total": 150,
    "page": 1,
    "page_size": 50
  }
}
```

---

## 5. WebSocket终端（详见websocket.md）

**端点**：`WS /api/v1/docker/terminal/:session_id`

详细协议见 `contracts/websocket.md`

---

## 通用错误响应

所有API遵循统一的错误响应格式：

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "错误描述",
    "details": {
      "field": "field_name",
      "reason": "详细原因"
    }
  }
}
```

**错误码列表**：

| 错误码 | HTTP状态码 | 描述 |
|--------|-----------|------|
| INVALID_PARAMETER | 400 | 参数验证失败 |
| UNAUTHORIZED | 401 | 未认证 |
| FORBIDDEN | 403 | 权限不足 |
| NOT_FOUND | 404 | 资源不存在 |
| CONFLICT | 409 | 资源冲突 |
| INSTANCE_OFFLINE | 503 | Docker实例离线 |
| AGENT_UNAVAILABLE | 503 | Agent不可用 |
| OPERATION_TIMEOUT | 504 | 操作超时 |
| INTERNAL_ERROR | 500 | 内部错误 |

---

## 只读模式

当系统处于只读模式时（`features.readonly_mode=true`），所有修改操作返回403错误：

```json
{
  "success": false,
  "error": {
    "code": "READONLY_MODE",
    "message": "系统当前处于只读模式，无法执行修改操作"
  }
}
```

**只读模式下允许的操作**：
- 所有GET请求
- 连接测试（POST /api/v1/docker/instances/:id/test-connection）

**只读模式下禁止的操作**：
- POST/PUT/DELETE请求（除连接测试外）

---

## Swagger文档

**生成命令**：
```bash
./scripts/generate-swagger.sh
```

**访问地址**：`http://localhost:12306/swagger/index.html`

**Swagger注解示例**：
```go
// @Summary 获取Docker实例列表
// @Description 获取所有Docker实例，支持分页和过滤
// @Tags Docker实例
// @Accept json
// @Produce json
// @Param name query string false "名称搜索"
// @Param health_status query string false "健康状态" Enums(online, offline, archived)
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} handlers.SuccessResponse{data=[]models.DockerInstance}
// @Failure 400 {object} handlers.ErrorResponse
// @Failure 401 {object} handlers.ErrorResponse
// @Router /api/v1/docker/instances [get]
// @Security BearerAuth
func (h *InstanceHandler) GetInstances(c *gin.Context) {
    // ...
}
```

---

**API版本**：v1.0.0
**创建时间**：2025-10-22
**状态**：草稿，待契约测试验证
