# API 契约：集群管理

**功能**：005-k8s-kite-k8s | **日期**：2025-10-17

## 概述

集群管理 API 提供多集群的 CRUD 操作、健康检查和上下文切换功能。

---

## C1: 获取集群列表

**端点**：`GET /api/v1/k8s/clusters`

**描述**：获取所有集群列表，包含健康状态和统计信息

**权限**：需要身份验证（JWT）

**请求参数**：无

**请求示例**：
```bash
curl -X GET http://localhost:12306/api/v1/k8s/clusters \
  -H "Authorization: Bearer <token>"
```

**响应**（200 OK）：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "clusters": [
      {
        "id": 1,
        "name": "production",
        "is_default": true,
        "enabled": true,
        "health_status": "healthy",
        "last_connected_at": "2025-10-17T10:30:00Z",
        "node_count": 5,
        "pod_count": 150,
        "prometheus_url": "http://prometheus-server.monitoring.svc.cluster.local:9090",
        "created_at": "2025-10-17T08:00:00Z",
        "updated_at": "2025-10-17T10:30:00Z"
      },
      {
        "id": 2,
        "name": "staging",
        "is_default": false,
        "enabled": true,
        "health_status": "warning",
        "last_connected_at": "2025-10-17T10:28:00Z",
        "node_count": 3,
        "pod_count": 80,
        "prometheus_url": "",
        "created_at": "2025-10-17T08:05:00Z",
        "updated_at": "2025-10-17T10:28:00Z"
      }
    ],
    "total": 2
  }
}
```

**错误响应**：
- `401 Unauthorized`：未提供有效的 JWT token
- `500 Internal Server Error`：数据库查询失败

---

## C2: 获取集群详情

**端点**：`GET /api/v1/k8s/clusters/:id`

**描述**：获取指定集群的详细信息

**权限**：需要身份验证（JWT）

**路径参数**：
- `id`：集群 ID（uint）

**请求示例**：
```bash
curl -X GET http://localhost:12306/api/v1/k8s/clusters/1 \
  -H "Authorization: Bearer <token>"
```

**响应**（200 OK）：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "name": "production",
    "kubeconfig": "YXBpVmVyc2lvbjog...",
    "is_default": true,
    "enabled": true,
    "health_status": "healthy",
    "last_connected_at": "2025-10-17T10:30:00Z",
    "node_count": 5,
    "pod_count": 150,
    "prometheus_url": "http://prometheus-server.monitoring.svc.cluster.local:9090",
    "created_at": "2025-10-17T08:00:00Z",
    "updated_at": "2025-10-17T10:30:00Z"
  }
}
```

**错误响应**：
- `401 Unauthorized`：未提供有效的 JWT token
- `404 Not Found`：集群不存在
- `500 Internal Server Error`：数据库查询失败

---

## C3: 创建集群

**端点**：`POST /api/v1/k8s/clusters`

**描述**：添加新集群配置

**权限**：需要管理员权限

**请求体**：
```json
{
  "name": "development",
  "kubeconfig": "YXBpVmVyc2lvbjog...",
  "is_default": false,
  "enabled": true
}
```

**字段说明**：
- `name`：集群名称（必填，1-100字符）
- `kubeconfig`：Kubeconfig 内容（必填，Base64编码）
- `is_default`：是否为默认集群（可选，默认 false）
- `enabled`：是否启用（可选，默认 true）

**请求示例**：
```bash
curl -X POST http://localhost:12306/api/v1/k8s/clusters \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "development",
    "kubeconfig": "YXBpVmVyc2lvbjog...",
    "is_default": false,
    "enabled": true
  }'
```

**响应**（201 Created）：
```json
{
  "code": 201,
  "message": "集群创建成功，Prometheus 自动发现任务已启动",
  "data": {
    "id": 3,
    "name": "development",
    "is_default": false,
    "enabled": true,
    "health_status": "unknown",
    "last_connected_at": null,
    "node_count": 0,
    "pod_count": 0,
    "prometheus_url": "",
    "created_at": "2025-10-17T11:00:00Z",
    "updated_at": "2025-10-17T11:00:00Z"
  }
}
```

**错误响应**：
- `400 Bad Request`：请求参数验证失败（如集群名称已存在、Kubeconfig 格式错误）
- `401 Unauthorized`：未提供有效的 JWT token
- `403 Forbidden`：非管理员用户
- `500 Internal Server Error`：数据库操作失败

**业务逻辑**：
1. 验证 Kubeconfig 格式（解析 YAML，检查必需字段）
2. 测试集群连接（调用 `GET /api/v1/namespaces`）
3. 保存集群配置到数据库
4. 启动 Prometheus 异步发现任务（如果启用自动发现）
5. 返回创建的集群信息

---

## C4: 更新集群

**端点**：`PUT /api/v1/k8s/clusters/:id`

**描述**：更新集群配置

**权限**：需要管理员权限

**路径参数**：
- `id`：集群 ID（uint）

**请求体**：
```json
{
  "name": "production-updated",
  "kubeconfig": "YXBpVmVyc2lvbjog...",
  "is_default": true,
  "enabled": true,
  "prometheus_url": "https://prometheus.prod.example.com"
}
```

**字段说明**（所有字段可选）：
- `name`：集群名称
- `kubeconfig`：Kubeconfig 内容
- `is_default`：是否为默认集群
- `enabled`：是否启用
- `prometheus_url`：Prometheus URL（手动配置）

**响应**（200 OK）：
```json
{
  "code": 200,
  "message": "集群更新成功",
  "data": {
    "id": 1,
    "name": "production-updated",
    "is_default": true,
    "enabled": true,
    "health_status": "healthy",
    "prometheus_url": "https://prometheus.prod.example.com",
    "updated_at": "2025-10-17T11:05:00Z"
  }
}
```

**错误响应**：
- `400 Bad Request`：请求参数验证失败
- `401 Unauthorized`：未提供有效的 JWT token
- `403 Forbidden`：非管理员用户
- `404 Not Found`：集群不存在
- `500 Internal Server Error`：数据库操作失败

**业务逻辑**：
1. 如果更新 Kubeconfig，清除集群的 Client 缓存
2. 如果手动设置 `prometheus_url`，停止该集群的自动发现任务
3. 保存更新到数据库

---

## C5: 删除集群

**端点**：`DELETE /api/v1/k8s/clusters/:id`

**描述**：删除集群配置（软删除）

**权限**：需要管理员权限

**路径参数**：
- `id`：集群 ID（uint）

**请求示例**：
```bash
curl -X DELETE http://localhost:12306/api/v1/k8s/clusters/3 \
  -H "Authorization: Bearer <token>"
```

**响应**（200 OK）：
```json
{
  "code": 200,
  "message": "集群删除成功",
  "data": null
}
```

**错误响应**：
- `401 Unauthorized`：未提供有效的 JWT token
- `403 Forbidden`：非管理员用户
- `404 Not Found`：集群不存在
- `500 Internal Server Error`：数据库操作失败

**业务逻辑**：
1. 软删除集群记录（设置 `deleted_at`）
2. 清除集群的 Client 缓存
3. 停止集群的 Prometheus 自动发现任务

---

## C6: 测试集群连接

**端点**：`POST /api/v1/k8s/clusters/:id/test-connection`

**描述**：测试集群连接是否正常

**权限**：需要身份验证（JWT）

**路径参数**：
- `id`：集群 ID（uint）

**请求示例**：
```bash
curl -X POST http://localhost:12306/api/v1/k8s/clusters/1/test-connection \
  -H "Authorization: Bearer <token>"
```

**响应**（200 OK）：
```json
{
  "code": 200,
  "message": "集群连接成功",
  "data": {
    "cluster_id": 1,
    "cluster_name": "production",
    "connected": true,
    "kubernetes_version": "v1.31.4",
    "node_count": 5,
    "tested_at": "2025-10-17T11:10:00Z"
  }
}
```

**错误响应**：
- `401 Unauthorized`：未提供有效的 JWT token
- `404 Not Found`：集群不存在
- `503 Service Unavailable`：无法连接到集群

---

## C7: 手动触发 Prometheus 重新检测

**端点**：`POST /api/v1/k8s/clusters/:id/prometheus/rediscover`

**描述**：手动触发 Prometheus 自动发现任务

**权限**：需要身份验证（JWT）

**路径参数**：
- `id`：集群 ID（uint）

**请求示例**：
```bash
curl -X POST http://localhost:12306/api/v1/k8s/clusters/1/prometheus/rediscover \
  -H "Authorization: Bearer <token>"
```

**响应**（202 Accepted）：
```json
{
  "code": 202,
  "message": "Prometheus 重新检测任务已启动，请稍后刷新查看结果",
  "data": {
    "cluster_id": 1,
    "task_started_at": "2025-10-17T11:15:00Z"
  }
}
```

**错误响应**：
- `401 Unauthorized`：未提供有效的 JWT token
- `404 Not Found`：集群不存在
- `409 Conflict`：已有正在运行的发现任务

---

**文档版本**：v1.0
**最后更新**：2025-10-17
**作者**：Claude Code
