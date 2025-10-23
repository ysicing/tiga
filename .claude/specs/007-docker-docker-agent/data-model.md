# 数据模型：Docker实例远程管理

**功能分支**：`007-docker-docker-agent`
**创建日期**：2025-10-22
**状态**：已完成
**输入**：来自 `spec.md` 功能需求和 `research.md` 技术决策

---

## 概述

Docker实例远程管理功能包含4个核心实体，其中2个持久化到数据库，2个从Agent实时获取。

**持久化实体**：
- `DockerInstance` - Docker实例元数据和健康状态
- `DockerOperation` - Docker操作审计记录

**非持久化实体**（从Agent实时获取）：
- `Container` - 容器信息
- `Image` - 镜像信息

**设计原则**：
- Agent为唯一数据源，容器/镜像数据不存储在数据库
- 操作历史持久化，包含目标对象快照
- 复用现有审计日志表，无需新建

---

## 1. DockerInstance（Docker实例）

**用途**：表示一个Docker守护进程实例，通过Agent管理

**存储**：数据库表 `docker_instances`

**Go模型定义**：
```go
package models

import (
    "time"
    "github.com/google/uuid"
)

// DockerInstance 表示一个Docker实例
type DockerInstance struct {
    BaseModel // ID, CreatedAt, UpdatedAt

    // 基本信息
    Name        string `gorm:"not null;index;uniqueIndex:idx_name_agent" json:"name"`
    Description string `gorm:"type:text" json:"description"`

    // Agent关联
    AgentID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_name_agent" json:"agent_id"`
    // 外键关联到现有 HostNode 表（002-nezha-webssh分支）
    // Agent 删除时，DockerInstance.HealthStatus 设置为 "archived"

    // 主机关联（冗余字段，便于查询）
    HostID   uuid.UUID `gorm:"type:uuid;not null;index" json:"host_id"`
    HostName string    `gorm:"index" json:"host_name"` // 主机名称快照

    // 健康状态
    HealthStatus     string    `gorm:"not null;index;default:'unknown'" json:"health_status"`
    // 可选值：unknown, online, offline, archived
    LastConnectedAt  time.Time `gorm:"index" json:"last_connected_at"`
    LastHealthCheck  time.Time `json:"last_health_check"`

    // Docker信息（从Agent上报，定期更新）
    DockerVersion  string `json:"docker_version"`  // e.g., "24.0.7"
    APIVersion     string `json:"api_version"`     // e.g., "1.43"
    StorageDriver  string `json:"storage_driver"`  // e.g., "overlay2"
    OperatingSystem string `json:"operating_system"` // e.g., "Ubuntu 22.04"
    Architecture   string `json:"architecture"`    // e.g., "x86_64"

    // 统计数据（从Agent上报，定期更新）
    ContainerCount int `gorm:"default:0" json:"container_count"`
    ImageCount     int `gorm:"default:0" json:"image_count"`
    VolumeCount    int `gorm:"default:0" json:"volume_count"`
    NetworkCount   int `gorm:"default:0" json:"network_count"`

    // 元数据
    Tags []string `gorm:"type:jsonb;serializer:json" json:"tags"` // 标签数组，支持搜索过滤
    // PostgreSQL使用jsonb，SQLite使用json，GORM自动处理
}

// TableName 指定表名
func (DockerInstance) TableName() string {
    return "docker_instances"
}

// 健康状态常量
const (
    DockerHealthStatusUnknown  = "unknown"  // 初始状态，未检查过
    DockerHealthStatusOnline   = "online"   // 在线，最近检查成功
    DockerHealthStatusOffline  = "offline"  // 离线，最近检查失败
    DockerHealthStatusArchived = "archived" // 已归档，Agent被删除
)

// IsOnline 判断实例是否在线
func (d *DockerInstance) IsOnline() bool {
    return d.HealthStatus == DockerHealthStatusOnline
}

// CanOperate 判断是否可以执行操作（在线且未归档）
func (d *DockerInstance) CanOperate() bool {
    return d.HealthStatus == DockerHealthStatusOnline
}

// MarkOffline 标记实例离线
func (d *DockerInstance) MarkOffline() {
    d.HealthStatus = DockerHealthStatusOffline
    d.LastHealthCheck = time.Now()
}

// MarkOnline 标记实例在线并更新统计数据
func (d *DockerInstance) MarkOnline(dockerInfo *DockerInfo) {
    d.HealthStatus = DockerHealthStatusOnline
    d.LastConnectedAt = time.Now()
    d.LastHealthCheck = time.Now()

    if dockerInfo != nil {
        d.DockerVersion = dockerInfo.Version
        d.APIVersion = dockerInfo.APIVersion
        d.StorageDriver = dockerInfo.StorageDriver
        d.OperatingSystem = dockerInfo.OperatingSystem
        d.Architecture = dockerInfo.Architecture
        d.ContainerCount = dockerInfo.Containers
        d.ImageCount = dockerInfo.Images
    }
}

// MarkArchived 标记实例为已归档（Agent删除时）
func (d *DockerInstance) MarkArchived() {
    d.HealthStatus = DockerHealthStatusArchived
    d.LastHealthCheck = time.Now()
}

// DockerInfo Agent上报的Docker信息
type DockerInfo struct {
    Version         string
    APIVersion      string
    StorageDriver   string
    OperatingSystem string
    Architecture    string
    Containers      int
    Images          int
}
```

**数据库索引**：
```sql
CREATE TABLE docker_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    name VARCHAR(255) NOT NULL,
    description TEXT,

    agent_id UUID NOT NULL,
    host_id UUID NOT NULL,
    host_name VARCHAR(255),

    health_status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    last_connected_at TIMESTAMP,
    last_health_check TIMESTAMP,

    docker_version VARCHAR(50),
    api_version VARCHAR(50),
    storage_driver VARCHAR(50),
    operating_system VARCHAR(100),
    architecture VARCHAR(50),

    container_count INT DEFAULT 0,
    image_count INT DEFAULT 0,
    volume_count INT DEFAULT 0,
    network_count INT DEFAULT 0,

    tags JSONB,

    CONSTRAINT uq_docker_instance_name_agent UNIQUE (name, agent_id)
);

CREATE INDEX idx_docker_instances_agent_id ON docker_instances(agent_id);
CREATE INDEX idx_docker_instances_host_id ON docker_instances(host_id);
CREATE INDEX idx_docker_instances_health_status ON docker_instances(health_status);
CREATE INDEX idx_docker_instances_last_connected_at ON docker_instances(last_connected_at);
CREATE INDEX idx_docker_instances_name ON docker_instances(name);
```

**状态转换图**：
```
           首次健康检查成功
unknown -----------------------> online
   |                                |
   | 首次健康检查失败                 | 健康检查失败
   |                                |
   v                                v
offline <----------------------- offline
   ^                                |
   |      健康检查成功                 |
   +----------------------------------+

任何状态 --Agent删除--> archived

archived --Agent重新注册--> online
```

**业务规则**：
1. `name + agent_id` 联合唯一约束（同一Agent下实例名称不可重复）
2. 实例创建时状态为 `unknown`，首次健康检查后变为 `online` 或 `offline`
3. Agent删除时，关联的所有DockerInstance自动设置为 `archived`
4. 归档实例保留所有数据，仅禁用操作功能
5. Agent重新注册时，可恢复归档实例为 `online`（需手动触发）

**关联关系**：
```go
// 在 HostNode 模型中（002-nezha-webssh分支）
type HostNode struct {
    BaseModel
    // ... 其他字段

    // 一对多关联
    DockerInstances []DockerInstance `gorm:"foreignKey:AgentID;references:ID" json:"docker_instances,omitempty"`
}

// 在 DockerOperation 模型中
type DockerOperation struct {
    // ... 字段定义见下文
    InstanceID uuid.UUID      `gorm:"type:uuid;not null;index" json:"instance_id"`
    Instance   *DockerInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}
```

---

## 2. Container（容器）

**用途**：表示Docker实例中的一个容器，从Agent实时获取

**存储**：不持久化到数据库，仅作为API响应对象

**Go模型定义**：
```go
package models

import (
    "time"
)

// Container 表示一个Docker容器（非持久化）
type Container struct {
    // 基本信息
    ID      string   `json:"id"`       // 容器ID（短格式，12字符）
    Name    string   `json:"name"`     // 容器名称（去除前导/）
    Image   string   `json:"image"`    // 镜像名称，如 "nginx:latest"
    ImageID string   `json:"image_id"` // 镜像ID

    // 状态信息
    State  string `json:"state"`  // created, running, paused, restarting, removing, exited, dead
    Status string `json:"status"` // 详细状态描述，如 "Up 2 hours"

    // 时间戳
    CreatedAt  time.Time `json:"created_at"`  // 容器创建时间
    StartedAt  time.Time `json:"started_at"`  // 最后启动时间
    FinishedAt time.Time `json:"finished_at"` // 最后停止时间

    // 网络配置
    Ports []ContainerPort `json:"ports"` // 端口映射列表

    // 存储配置
    Mounts []ContainerMount `json:"mounts"` // 挂载卷列表

    // 网络配置
    Networks map[string]ContainerNetwork `json:"networks"` // 网络配置映射

    // 环境变量
    Env []string `json:"env"` // 环境变量数组

    // 标签
    Labels map[string]string `json:"labels"` // 标签映射

    // 命令
    Command []string `json:"command"` // 启动命令

    // 资源限制
    CPULimit    int64 `json:"cpu_limit"`    // CPU限制（纳秒）
    MemoryLimit int64 `json:"memory_limit"` // 内存限制（字节）

    // 重启策略
    RestartCount  int    `json:"restart_count"`  // 重启次数
    RestartPolicy string `json:"restart_policy"` // 重启策略，如 "always"

    // 运行时信息（可选，按需加载）
    Stats *ContainerStats `json:"stats,omitempty"` // 资源使用统计
}

// ContainerPort 容器端口映射
type ContainerPort struct {
    IP            string `json:"ip"`             // 绑定IP，如 "0.0.0.0"
    PrivatePort   int    `json:"private_port"`   // 容器内端口
    PublicPort    int    `json:"public_port"`    // 主机端口
    Type          string `json:"type"`           // tcp/udp
    HostIP        string `json:"host_ip"`        // 主机IP
}

// ContainerMount 容器挂载卷
type ContainerMount struct {
    Type        string `json:"type"`        // bind, volume, tmpfs
    Source      string `json:"source"`      // 源路径/卷名
    Destination string `json:"destination"` // 容器内路径
    Mode        string `json:"mode"`        // rw, ro
    RW          bool   `json:"rw"`          // 是否可写
}

// ContainerNetwork 容器网络配置
type ContainerNetwork struct {
    NetworkID   string `json:"network_id"`   // 网络ID
    Gateway     string `json:"gateway"`      // 网关地址
    IPAddress   string `json:"ip_address"`   // 容器IP地址
    IPPrefixLen int    `json:"ip_prefix_len"` // IP前缀长度
    MacAddress  string `json:"mac_address"`  // MAC地址
}

// ContainerStats 容器资源使用统计（实时数据）
type ContainerStats struct {
    // CPU统计
    CPUUsagePercent float64 `json:"cpu_usage_percent"` // CPU使用率（百分比）
    CPUUsageNano    uint64  `json:"cpu_usage_nano"`    // CPU使用量（纳秒）

    // 内存统计
    MemoryUsage      uint64  `json:"memory_usage"`       // 内存使用量（字节）
    MemoryLimit      uint64  `json:"memory_limit"`       // 内存限制（字节）
    MemoryUsagePercent float64 `json:"memory_usage_percent"` // 内存使用率（百分比）

    // 网络统计
    NetworkRxBytes uint64 `json:"network_rx_bytes"` // 网络接收字节数
    NetworkTxBytes uint64 `json:"network_tx_bytes"` // 网络发送字节数

    // 磁盘IO统计
    BlockReadBytes  uint64 `json:"block_read_bytes"`  // 磁盘读取字节数
    BlockWriteBytes uint64 `json:"block_write_bytes"` // 磁盘写入字节数

    // PIDs
    PIDsCurrent uint64 `json:"pids_current"` // 当前进程数
}

// 容器状态常量
const (
    ContainerStateCreated    = "created"
    ContainerStateRunning    = "running"
    ContainerStatePaused     = "paused"
    ContainerStateRestarting = "restarting"
    ContainerStateRemoving   = "removing"
    ContainerStateExited     = "exited"
    ContainerStateDead       = "dead"
)

// IsRunning 判断容器是否正在运行
func (c *Container) IsRunning() bool {
    return c.State == ContainerStateRunning
}

// CanStart 判断是否可以启动
func (c *Container) CanStart() bool {
    return c.State == ContainerStateCreated ||
           c.State == ContainerStateExited ||
           c.State == ContainerStateDead
}

// CanStop 判断是否可以停止
func (c *Container) CanStop() bool {
    return c.State == ContainerStateRunning ||
           c.State == ContainerStatePaused ||
           c.State == ContainerStateRestarting
}

// CanDelete 判断是否可以删除
func (c *Container) CanDelete() bool {
    return c.State != ContainerStateRemoving
}
```

**数据来源**：通过Agent gRPC调用 `docker ps` 实时获取

**缓存策略**（研究任务2决策）：
- TTL：5分钟
- 缓存键：`instanceID:containers`
- 失效条件：实例健康状态变化、用户手动刷新

---

## 3. Image（镜像）

**用途**：表示Docker实例上的容器镜像，从Agent实时获取

**存储**：不持久化到数据库，仅作为API响应对象

**Go模型定义**：
```go
package models

import (
    "time"
)

// Image 表示一个Docker镜像（非持久化）
type Image struct {
    // 基本信息
    ID       string   `json:"id"`        // 镜像ID（短格式，12字符）
    RepoTags []string `json:"repo_tags"` // 标签列表，如 ["nginx:latest", "nginx:1.25"]
    RepoDigests []string `json:"repo_digests"` // 摘要列表

    // 大小信息
    Size        int64 `json:"size"`         // 镜像大小（字节）
    VirtualSize int64 `json:"virtual_size"` // 虚拟大小（包含共享层）

    // 时间戳
    Created time.Time `json:"created"` // 镜像创建时间

    // 标签
    Labels map[string]string `json:"labels"` // 标签映射

    // 层信息
    Layers []string `json:"layers"` // 镜像层SHA256列表

    // 元数据（可选，详情页加载）
    Comment       string            `json:"comment,omitempty"`
    Author        string            `json:"author,omitempty"`
    Architecture  string            `json:"architecture,omitempty"`
    OS            string            `json:"os,omitempty"`
    Config        *ImageConfig      `json:"config,omitempty"`
    History       []ImageHistory    `json:"history,omitempty"`
}

// ImageConfig 镜像配置
type ImageConfig struct {
    Env         []string          `json:"env"`          // 环境变量
    Cmd         []string          `json:"cmd"`          // 默认命令
    Entrypoint  []string          `json:"entrypoint"`   // 入口点
    WorkingDir  string            `json:"working_dir"`  // 工作目录
    ExposedPorts map[string]struct{} `json:"exposed_ports"` // 暴露端口
    Volumes     map[string]struct{} `json:"volumes"`      // 数据卷
    Labels      map[string]string `json:"labels"`       // 标签
}

// ImageHistory 镜像历史记录
type ImageHistory struct {
    Created    time.Time `json:"created"`
    CreatedBy  string    `json:"created_by"`  // 创建命令
    Size       int64     `json:"size"`        // 该层大小
    Comment    string    `json:"comment"`
    EmptyLayer bool      `json:"empty_layer"` // 是否为空层
}

// GetMainTag 获取主标签（第一个）
func (i *Image) GetMainTag() string {
    if len(i.RepoTags) > 0 {
        return i.RepoTags[0]
    }
    return "<none>:<none>"
}

// IsUntagged 判断是否为未标记镜像
func (i *Image) IsUntagged() bool {
    return len(i.RepoTags) == 0 || i.RepoTags[0] == "<none>:<none>"
}

// CanDelete 判断是否可以删除（未被容器使用）
// 注意：此方法仅判断标签状态，实际删除需检查容器依赖
func (i *Image) CanDelete() bool {
    return true // Docker API会检查容器依赖并返回错误
}

// FormatSize 格式化大小显示
func (i *Image) FormatSize() string {
    const (
        KB = 1024
        MB = 1024 * KB
        GB = 1024 * MB
    )

    size := float64(i.Size)
    switch {
    case size >= GB:
        return fmt.Sprintf("%.2f GB", size/GB)
    case size >= MB:
        return fmt.Sprintf("%.2f MB", size/MB)
    case size >= KB:
        return fmt.Sprintf("%.2f KB", size/KB)
    default:
        return fmt.Sprintf("%d B", i.Size)
    }
}
```

**数据来源**：通过Agent gRPC调用 `docker images` 实时获取

**缓存策略**（研究任务2决策）：
- TTL：5分钟
- 缓存键：`instanceID:images`
- 失效条件：实例健康状态变化、用户手动刷新、拉取/删除镜像操作后

---

## 4. DockerOperation（Docker操作记录）

**用途**：记录所有Docker操作，用于审计和问题追溯

**存储**：复用现有审计日志表 `audit_logs`

**Go模型定义**：
```go
// 复用现有 AuditLog 模型（internal/models/audit_log.go）
// 无需新建 DockerOperation 表

package models

import (
    "encoding/json"
    "time"
    "github.com/google/uuid"
)

// AuditLog 审计日志（现有模型）
type AuditLog struct {
    ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
    UserID       uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
    Username     string    `json:"username"`
    Action       string    `gorm:"not null;index" json:"action"` // Docker操作类型
    ResourceType string    `gorm:"not null;index" json:"resource_type"` // docker_container, docker_image, docker_instance
    ResourceID   string    `gorm:"index" json:"resource_id"` // 容器ID/镜像ID/实例ID
    ResourceName string    `json:"resource_name"` // 容器名称/镜像名称/实例名称
    Details      string    `gorm:"type:text" json:"details"` // JSON格式详细信息
    IPAddress    string    `json:"ip_address"`
    UserAgent    string    `json:"user_agent"`
    Timestamp    time.Time `gorm:"not null;index" json:"timestamp"`
}

// Docker操作类型常量
const (
    // 容器操作
    ActionContainerStart   = "container_start"
    ActionContainerStop    = "container_stop"
    ActionContainerRestart = "container_restart"
    ActionContainerPause   = "container_pause"
    ActionContainerUnpause = "container_unpause"
    ActionContainerDelete  = "container_delete"
    ActionContainerCreate  = "container_create"
    ActionContainerExec    = "container_exec" // 进入终端

    // 镜像操作
    ActionImagePull   = "image_pull"
    ActionImageDelete = "image_delete"
    ActionImageTag    = "image_tag"

    // 实例操作
    ActionInstanceCreate = "instance_create"
    ActionInstanceUpdate = "instance_update"
    ActionInstanceDelete = "instance_delete"
    ActionInstanceArchive = "instance_archive" // 归档
    ActionInstanceRestore = "instance_restore" // 恢复
)

// 资源类型常量
const (
    ResourceTypeDockerContainer = "docker_container"
    ResourceTypeDockerImage     = "docker_image"
    ResourceTypeDockerInstance  = "docker_instance"
)

// DockerOperationDetails 操作详细信息（存储在Details字段的JSON）
type DockerOperationDetails struct {
    // 实例信息
    InstanceID   string `json:"instance_id"`
    InstanceName string `json:"instance_name"`

    // 操作前状态（快照）
    StateBefore string `json:"state_before,omitempty"` // 容器状态
    ImageBefore string `json:"image_before,omitempty"` // 容器镜像

    // 操作后状态
    StateAfter string `json:"state_after,omitempty"`

    // 操作参数
    Parameters map[string]interface{} `json:"parameters,omitempty"`

    // 操作结果
    Success      bool   `json:"success"`
    ErrorMessage string `json:"error_message,omitempty"`
    Duration     int64  `json:"duration"` // 操作耗时（毫秒）
}

// NewDockerAuditLog 创建Docker操作审计日志
func NewDockerAuditLog(
    userID uuid.UUID,
    username string,
    action string,
    resourceType string,
    resourceID string,
    resourceName string,
    details *DockerOperationDetails,
    ipAddress string,
    userAgent string,
) *AuditLog {
    detailsJSON, _ := json.Marshal(details)

    return &AuditLog{
        ID:           uuid.New(),
        UserID:       userID,
        Username:     username,
        Action:       action,
        ResourceType: resourceType,
        ResourceID:   resourceID,
        ResourceName: resourceName,
        Details:      string(detailsJSON),
        IPAddress:    ipAddress,
        UserAgent:    userAgent,
        Timestamp:    time.Now(),
    }
}

// ParseDockerDetails 解析Details字段为DockerOperationDetails
func (a *AuditLog) ParseDockerDetails() (*DockerOperationDetails, error) {
    var details DockerOperationDetails
    if err := json.Unmarshal([]byte(a.Details), &details); err != nil {
        return nil, err
    }
    return &details, nil
}
```

**审计日志示例**：
```json
{
    "id": "uuid",
    "user_id": "user-uuid",
    "username": "admin",
    "action": "container_stop",
    "resource_type": "docker_container",
    "resource_id": "abc123def456",
    "resource_name": "nginx-web",
    "details": {
        "instance_id": "docker-instance-uuid",
        "instance_name": "prod-server-1",
        "state_before": "running",
        "state_after": "exited",
        "image_before": "nginx:latest",
        "parameters": {
            "force": false,
            "timeout": 10
        },
        "success": true,
        "duration": 1250
    },
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0...",
    "timestamp": "2025-10-22T10:30:00Z"
}
```

**查询示例**：
```sql
-- 查询某用户的所有Docker操作
SELECT * FROM audit_logs
WHERE user_id = 'uuid'
  AND resource_type IN ('docker_container', 'docker_image', 'docker_instance')
ORDER BY timestamp DESC
LIMIT 50;

-- 查询某实例的所有容器操作
SELECT * FROM audit_logs
WHERE resource_type = 'docker_container'
  AND details::jsonb->>'instance_id' = 'instance-uuid'
ORDER BY timestamp DESC;

-- 查询失败的操作
SELECT * FROM audit_logs
WHERE resource_type LIKE 'docker_%'
  AND details::jsonb->>'success' = 'false'
ORDER BY timestamp DESC;
```

**保留策略**（研究任务11决策）：
- 保留时长：90天
- 清理任务：定时任务每天2AM执行，删除90天前的日志
- 审计日志永不物理删除关键操作（可选：标记为archived）

---

## 数据关系图

```
┌─────────────────────────────────────────────────────────────────┐
│                        HostNode（主机节点）                       │
│                     (002-nezha-webssh分支)                       │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           │ 1:N
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    DockerInstance（Docker实例）                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ - ID, Name, AgentID, HostID                                │ │
│  │ - HealthStatus (online/offline/archived)                   │ │
│  │ - DockerVersion, ContainerCount, ImageCount                │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           │ 实时查询（非关联）
                           │
           ┌───────────────┴───────────────┐
           │                               │
           ▼                               ▼
┌─────────────────────┐          ┌─────────────────────┐
│  Container（容器）   │          │   Image（镜像）      │
│  ┌─────────────────┐│          │  ┌─────────────────┐│
│  │ - ID, Name      ││          │  │ - ID, RepoTags  ││
│  │ - State, Image  ││          │  │ - Size, Layers  ││
│  │ - Ports, Mounts ││          │  │ - Created       ││
│  └─────────────────┘│          │  └─────────────────┘│
│  非持久化，Agent获取 │          │  非持久化，Agent获取 │
└─────────────────────┘          └─────────────────────┘
           │                               │
           │                               │
           └───────────────┬───────────────┘
                           │
                           │ 操作记录（快照）
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    AuditLog（审计日志）                          │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ - UserID, Action, ResourceType, ResourceID                 │ │
│  │ - Details (JSON: InstanceID, StateBefore, Parameters)      │ │
│  │ - Timestamp, IPAddress                                     │ │
│  └────────────────────────────────────────────────────────────┘ │
│  复用现有表，ResourceType=docker_*                               │
└─────────────────────────────────────────────────────────────────┘
           ▲
           │
           │ N:1
           │
┌─────────────────────┐
│   User（用户）       │
│  ┌─────────────────┐│
│  │ - ID, Username  ││
│  │ - Role, Perms   ││
│  └─────────────────┘│
│  现有用户表          │
└─────────────────────┘
```

---

## 数据库迁移脚本

**GORM自动迁移代码**（internal/db/migrate.go）：
```go
package db

import (
    "github.com/ysicing/tiga/internal/models"
    "gorm.io/gorm"
)

// AutoMigrateDockerTables Docker子系统数据库迁移
func AutoMigrateDockerTables(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.DockerInstance{},
        // AuditLog已在现有迁移中，无需重复
    )
}
```

**PostgreSQL DDL示例**（仅供参考，实际使用GORM自动迁移）：
```sql
-- DockerInstance 表
CREATE TABLE IF NOT EXISTS docker_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    name VARCHAR(255) NOT NULL,
    description TEXT,

    agent_id UUID NOT NULL,
    host_id UUID NOT NULL,
    host_name VARCHAR(255),

    health_status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    last_connected_at TIMESTAMP,
    last_health_check TIMESTAMP,

    docker_version VARCHAR(50),
    api_version VARCHAR(50),
    storage_driver VARCHAR(50),
    operating_system VARCHAR(100),
    architecture VARCHAR(50),

    container_count INT DEFAULT 0,
    image_count INT DEFAULT 0,
    volume_count INT DEFAULT 0,
    network_count INT DEFAULT 0,

    tags JSONB,

    CONSTRAINT fk_docker_instance_agent FOREIGN KEY (agent_id) REFERENCES host_nodes(id) ON DELETE SET NULL,
    CONSTRAINT uq_docker_instance_name_agent UNIQUE (name, agent_id)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_docker_instances_agent_id ON docker_instances(agent_id);
CREATE INDEX IF NOT EXISTS idx_docker_instances_host_id ON docker_instances(host_id);
CREATE INDEX IF NOT EXISTS idx_docker_instances_health_status ON docker_instances(health_status);
CREATE INDEX IF NOT EXISTS idx_docker_instances_last_connected_at ON docker_instances(last_connected_at);
CREATE INDEX IF NOT EXISTS idx_docker_instances_name ON docker_instances(name);

-- AuditLog 表已存在，无需创建
-- 添加Docker相关索引（可选优化）
CREATE INDEX IF NOT EXISTS idx_audit_logs_docker_resource_type ON audit_logs(resource_type) WHERE resource_type LIKE 'docker_%';
```

---

## 数据访问层（Repository接口）

**DockerInstanceRepository接口定义**（internal/repository/interfaces.go）：
```go
package repository

import (
    "context"
    "github.com/google/uuid"
    "github.com/ysicing/tiga/internal/models"
)

// DockerInstanceRepositoryInterface Docker实例仓储接口
type DockerInstanceRepositoryInterface interface {
    // 基本CRUD
    Create(ctx context.Context, instance *models.DockerInstance) error
    Update(ctx context.Context, instance *models.DockerInstance) error
    Delete(ctx context.Context, id uuid.UUID) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.DockerInstance, error)
    GetAll(ctx context.Context) ([]*models.DockerInstance, error)

    // 查询方法
    GetByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DockerInstance, error)
    GetByHostID(ctx context.Context, hostID uuid.UUID) ([]*models.DockerInstance, error)
    GetByHealthStatus(ctx context.Context, status string) ([]*models.DockerInstance, error)
    GetOnlineInstances(ctx context.Context) ([]*models.DockerInstance, error)

    // 批量操作
    UpdateHealthStatus(ctx context.Context, id uuid.UUID, status string) error
    MarkAllInstancesOfflineByAgentID(ctx context.Context, agentID uuid.UUID) error
    MarkInstanceArchived(ctx context.Context, id uuid.UUID) error

    // 搜索
    Search(ctx context.Context, filters DockerInstanceFilters) ([]*models.DockerInstance, int64, error)
}

// DockerInstanceFilters 查询过滤器
type DockerInstanceFilters struct {
    Name         string
    HealthStatus string
    AgentID      *uuid.UUID
    HostID       *uuid.UUID
    Tags         []string
    Page         int
    PageSize     int
    SortBy       string // name, created_at, last_connected_at
    SortOrder    string // asc, desc
}
```

**实现示例**（internal/repository/docker/instance_repository.go）：
```go
package docker

import (
    "context"
    "github.com/google/uuid"
    "github.com/ysicing/tiga/internal/models"
    "github.com/ysicing/tiga/internal/repository"
    "gorm.io/gorm"
)

type dockerInstanceRepository struct {
    db *gorm.DB
}

func NewDockerInstanceRepository(db *gorm.DB) repository.DockerInstanceRepositoryInterface {
    return &dockerInstanceRepository{db: db}
}

func (r *dockerInstanceRepository) Create(ctx context.Context, instance *models.DockerInstance) error {
    return r.db.WithContext(ctx).Create(instance).Error
}

func (r *dockerInstanceRepository) GetOnlineInstances(ctx context.Context) ([]*models.DockerInstance, error) {
    var instances []*models.DockerInstance
    err := r.db.WithContext(ctx).
        Where("health_status = ?", models.DockerHealthStatusOnline).
        Find(&instances).Error
    return instances, err
}

// ... 其他方法实现
```

---

## 验证规则

**数据验证**（使用 `github.com/go-playground/validator`）：
```go
// DockerInstance 验证规则
type DockerInstanceCreateRequest struct {
    Name        string   `json:"name" validate:"required,min=1,max=255"`
    Description string   `json:"description" validate:"max=1000"`
    AgentID     string   `json:"agent_id" validate:"required,uuid"`
    Tags        []string `json:"tags" validate:"dive,min=1,max=50"`
}

// 自定义验证函数
func ValidateDockerInstanceName(fl validator.FieldLevel) bool {
    name := fl.Field().String()
    // 只允许字母、数字、下划线、中划线
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
    return matched
}
```

---

## 性能优化建议

1. **索引优化**：
   - `health_status` 索引用于健康检查任务过滤
   - `agent_id` 索引用于Agent关联查询
   - `last_connected_at` 索引用于时间范围查询

2. **查询优化**：
   - 容器列表查询：使用缓存（5分钟TTL）
   - 健康检查：批量查询所有实例，10并发更新
   - 审计日志查询：添加时间范围限制避免全表扫描

3. **数据清理**：
   - 审计日志：定时清理90天前数据
   - 归档实例：可选手动清理（提供批量删除API）

---

**数据模型设计完成时间**：2025-10-22
**设计版本**：v1.0.0
**基于规格**：`spec.md` v1.0.0、`research.md` v1.0.0
