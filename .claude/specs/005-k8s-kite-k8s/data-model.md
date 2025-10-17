# 数据模型：K8s子系统

**功能**：005-k8s-kite-k8s | **日期**：2025-10-17

## 概述

本文档定义 K8s 子系统的数据模型，包括扩展现有实体和新增配置项。所有模型使用 GORM 进行 ORM 映射。

---

## 实体定义

### E1: Cluster（扩展现有实体）

**位置**：`internal/models/cluster.go`

**描述**：K8s 集群配置，扩展健康状态和统计信息

**字段**：

| 字段名 | 类型 | 必填 | 说明 | 验证规则 |
|--------|------|------|------|---------|
| `ID` | `uint` | 是 | 主键（数据库自增ID） | 唯一 |
| `Name` | `string` | 是 | 集群名称 | 长度 1-100 |
| `Kubeconfig` | `string` | 是 | Kubeconfig 内容（Base64编码） | 非空 |
| `IsDefault` | `bool` | 否 | 是否为默认集群 | 默认 false |
| `Enabled` | `bool` | 否 | 是否启用 | 默认 true |
| `HealthStatus` | `string` | 否 | **新增**：健康状态 | 枚举：healthy/warning/error/unavailable，默认 unknown |
| `LastConnectedAt` | `time.Time` | 否 | **新增**：最后连接时间 | - |
| `NodeCount` | `int` | 否 | **新增**：节点数量 | >= 0 |
| `PodCount` | `int` | 否 | **新增**：Pod数量 | >= 0 |
| `PrometheusURL` | `string` | 否 | **新增**：Prometheus URL | 必须是有效的 HTTP/HTTPS URL |
| `CreatedAt` | `time.Time` | 是 | 创建时间 | 自动设置 |
| `UpdatedAt` | `time.Time` | 是 | 更新时间 | 自动更新 |
| `DeletedAt` | `*time.Time` | 否 | 删除时间（软删除） | - |

**索引**：
- 唯一索引：`name`（支持软删除，仅对 deleted_at IS NULL 的记录唯一）
- 普通索引：`enabled`、`health_status`

**关系**：
- 无外键关系（独立实体）

**示例**：
```json
{
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
```

---

### E2: ConfigExtension（配置扩展）

**位置**：`internal/config/config.go`

**描述**：扩展现有配置结构，添加 K8s 和 Prometheus 相关配置

#### KubernetesConfig

**字段**：

| 字段名 | 类型 | 必填 | 说明 | 默认值 |
|--------|------|------|------|--------|
| `NodeTerminalImage` | `string` | 否 | 节点终端镜像 | `gcr.io/google-containers/ubuntu:latest` |
| `EnableKruise` | `bool` | 否 | 启用 OpenKruise 支持 | `true` |
| `EnableTailscale` | `bool` | 否 | 启用 Tailscale 支持 | `false` |
| `EnableTraefik` | `bool` | 否 | 启用 Traefik 支持 | `true` |
| `EnableK3sUpgrade` | `bool` | 否 | 启用 K3s Upgrade Controller 支持 | `false` |

#### PrometheusConfig

**字段**：

| 字段名 | 类型 | 必填 | 说明 | 默认值 |
|--------|------|------|------|--------|
| `AutoDiscovery` | `bool` | 否 | 启用自动发现 | `true` |
| `DiscoveryTimeout` | `int` | 否 | 发现超时（秒） | `30` |
| `ClusterURLs` | `map[string]string` | 否 | 集群特定URL映射（key: cluster name, value: URL） | `{}` |

#### FeaturesConfig

**字段**：

| 字段名 | 类型 | 必填 | 说明 | 默认值 |
|--------|------|------|------|--------|
| `ReadonlyMode` | `bool` | 否 | 只读模式 | `false` |

**示例配置（YAML）**：
```yaml
kubernetes:
  node_terminal_image: gcr.io/google-containers/ubuntu:latest
  enable_kruise: true
  enable_tailscale: false
  enable_traefik: true
  enable_k3s_upgrade: false

prometheus:
  auto_discovery: true
  discovery_timeout: 30
  cluster_urls:
    production: https://prometheus.prod.example.com
    staging: http://prometheus-server.monitoring.svc.cluster.local:9090

features:
  readonly_mode: false
```

---

## 状态转换

### 集群健康状态

```
unknown → healthy    # 首次连接成功
healthy → warning    # 部分节点异常
warning → healthy    # 所有节点恢复
healthy → error      # 集群 API 返回错误
error → unavailable  # 连接超时
unavailable → healthy # 连接恢复
```

**状态定义**：
- `unknown`：初始状态，未进行健康检查
- `healthy`：所有节点正常，API 可达
- `warning`：部分节点异常（NotReady），但 API 可达
- `error`：API 可达，但返回错误（如权限不足）
- `unavailable`：无法连接到集群 API

**健康检查逻辑**：
1. 每 60 秒执行一次健康检查（后台 Goroutine）
2. 调用 `GET /api/v1/nodes` 获取节点列表
3. 统计 Ready 节点数量
4. 更新 `health_status`、`last_connected_at`、`node_count`

---

## 验证规则

### Cluster 验证

1. **Kubeconfig 验证**：
   - 必须是有效的 YAML 格式
   - 必须包含 `clusters`、`contexts`、`users` 字段
   - 至少包含一个 context

2. **Prometheus URL 验证**：
   - 如果提供，必须是有效的 HTTP/HTTPS URL
   - 支持 ClusterIP、NodePort、LoadBalancer、Ingress

3. **集群名称唯一性**：
   - 在未删除的集群中必须唯一
   - 支持软删除后重新创建同名集群

### 配置验证

1. **节点终端镜像**：
   - 必须是有效的 Docker 镜像格式（`registry/repository:tag`）

2. **Prometheus 发现超时**：
   - 必须在 5-300 秒范围内

---

## 数据库迁移

### 迁移脚本（GORM AutoMigrate）

```go
// internal/models/cluster.go

// 添加新字段
type Cluster struct {
    // 现有字段...

    // 新增字段
    HealthStatus     string     `gorm:"column:health_status;type:varchar(20);default:'unknown';index" json:"health_status"`
    LastConnectedAt  *time.Time `gorm:"column:last_connected_at" json:"last_connected_at,omitempty"`
    NodeCount        int        `gorm:"column:node_count;default:0" json:"node_count"`
    PodCount         int        `gorm:"column:pod_count;default:0" json:"pod_count"`
    PrometheusURL    string     `gorm:"column:prometheus_url;type:varchar(512)" json:"prometheus_url,omitempty"`
}
```

**迁移步骤**：
1. 启动应用时自动执行 `db.AutoMigrate(&models.Cluster{})`
2. 为现有集群设置默认值：
   - `health_status = 'unknown'`
   - `node_count = 0`
   - `pod_count = 0`

---

## 缓存策略

### K8s Client 实例缓存

**缓存键**：`cluster_id`（uint）
**缓存值**：`*K8sClient`
**过期策略**：无过期时间（手动清理）
**清理时机**：
- 集群被删除时
- 集群 Kubeconfig 被更新时

### 工作负载资源缓存

**缓存键**：`cluster_id:resource_type`（如 `1:pods`）
**缓存值**：资源列表（JSON）
**过期时间**：5 分钟
**清理时机**：
- 手动刷新
- 资源被修改后（通过 ResourceVersion 检测）

### 搜索结果缓存

**缓存键**：`search:cluster_id:term`（如 `search:1:nginx`）
**缓存值**：搜索结果列表（JSON）
**过期时间**：5 分钟

---

## 并发控制

### ResourceVersion 冲突检测

**机制**：依赖 Kubernetes API Server 的 ResourceVersion 验证

**流程**：
1. 用户 A 读取资源（ResourceVersion=1000）
2. 用户 B 读取资源（ResourceVersion=1000）
3. 用户 A 修改并保存（ResourceVersion 更新为 1001）
4. 用户 B 尝试保存（携带 ResourceVersion=1000）
5. API Server 返回 409 Conflict 错误
6. 系统提示用户 B："资源已被修改，请刷新后重试"

**实施建议**：
- 不在 Tiga 层实现锁机制
- 直接使用 client-go 的默认行为
- 在 API Handler 中捕获 409 错误并返回友好提示

---

**文档版本**：v1.0
**最后更新**：2025-10-17
**作者**：Claude Code
