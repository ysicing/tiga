# 数据模型:主机管理子系统

**功能分支**:`002-nezha-webssh`
**创建日期**:2025-10-07
**相关规格**:[spec.md](./spec.md) | **研究**:[research.md](./research.md)

## 核心实体

### 1. HostNode(主机节点)

主机节点是被监控服务器的核心实体,包含主机的元数据和配置信息。

```go
type HostNode struct {
    gorm.Model

    // 基本信息
    UUID         string `gorm:"uniqueIndex;not null"` // 主机唯一标识
    Name         string `gorm:"not null"`             // 主机名称
    SecretKey    string `gorm:"not null"`             // Agent连接密钥(加密存储)

    // 显示配置
    Note         string `gorm:"type:text"`            // 管理员备注
    PublicNote   string `gorm:"type:text"`            // 公开备注
    DisplayIndex int    `gorm:"default:0"`            // 显示排序(越大越靠前)
    HideForGuest bool   `gorm:"default:false"`        // 对游客隐藏

    // WebSSH配置
    EnableWebSSH bool   `gorm:"default:false"`        // 是否启用WebSSH
    SSHPort      int    `gorm:"default:22"`           // SSH端口
    SSHUser      string `gorm:"default:root"`         // SSH用户名

    // 关联关系
    GroupIDs     string `gorm:"type:text"`            // 所属分组ID列表(JSON数组)

    // 运行时状态(不持久化)
    Online       bool      `gorm:"-"` // 在线状态
    LastActive   time.Time `gorm:"-"` // 最后活跃时间
}
```

**验证规则**:
- UUID必须唯一且不可变
- Name不能为空
- SecretKey使用AES-256加密存储
- DisplayIndex默认0,可为负数
- SSHPort范围1-65535

**索引**:
- `uuid`:唯一索引
- `created_at`:时间索引(用于排序)
- `display_index`:显示排序索引

---

### 2. HostInfo(主机信息)

主机的静态硬件和系统信息,由Agent上报一次。

```go
type HostInfo struct {
    gorm.Model

    HostNodeID uint   `gorm:"uniqueIndex;not null"` // 关联主机节点

    // 系统信息
    Platform        string `json:"platform"`         // 操作系统(linux/windows/darwin)
    PlatformVersion string `json:"platform_version"` // 系统版本
    Arch            string `json:"arch"`             // 架构(amd64/arm64)
    Virtualization  string `json:"virtualization"`   // 虚拟化类型(kvm/docker/none)

    // 硬件信息
    CPUModel    string `json:"cpu_model"`      // CPU型号
    CPUCores    int    `json:"cpu_cores"`      // CPU核心数
    MemTotal    uint64 `json:"mem_total"`      // 内存总量(字节)
    DiskTotal   uint64 `json:"disk_total"`     // 磁盘总量(字节)
    SwapTotal   uint64 `json:"swap_total"`     // 交换分区大小(字节)

    // Agent信息
    AgentVersion string `json:"agent_version"`  // Agent版本
    BootTime     uint64 `json:"boot_time"`      // 系统启动时间(Unix时间戳)

    // GPU信息(可选)
    GPUModel string `json:"gpu_model,omitempty"` // GPU型号
}
```

**关联关系**:
- 一对一关联HostNode
- Agent上报时自动创建或更新

---

### 3. HostState(主机状态)

主机的实时监控指标快照,高频更新(每30秒)。

```go
type HostState struct {
    gorm.Model

    HostNodeID uint      `gorm:"index;not null"` // 关联主机节点
    Timestamp  time.Time `gorm:"index;not null"` // 数据采集时间

    // CPU和负载
    CPUUsage float64 `json:"cpu_usage"` // CPU使用率(%)
    Load1    float64 `json:"load_1"`    // 1分钟负载
    Load5    float64 `json:"load_5"`    // 5分钟负载
    Load15   float64 `json:"load_15"`   // 15分钟负载

    // 内存
    MemUsed  uint64 `json:"mem_used"`  // 已用内存(字节)
    MemUsage float64 `json:"mem_usage"` // 内存使用率(%)
    SwapUsed uint64 `json:"swap_used"` // 已用交换分区(字节)

    // 磁盘
    DiskUsed  uint64 `json:"disk_used"`  // 已用磁盘(字节)
    DiskUsage float64 `json:"disk_usage"` // 磁盘使用率(%)

    // 网络
    NetInTransfer  uint64 `json:"net_in_transfer"`  // 入站总流量(字节)
    NetOutTransfer uint64 `json:"net_out_transfer"` // 出站总流量(字节)
    NetInSpeed     uint64 `json:"net_in_speed"`     // 入站速率(字节/秒)
    NetOutSpeed    uint64 `json:"net_out_speed"`    // 出站速率(字节/秒)

    // 连接和进程
    TCPConnCount uint64 `json:"tcp_conn_count"` // TCP连接数
    UDPConnCount uint64 `json:"udp_conn_count"` // UDP连接数
    ProcessCount uint64 `json:"process_count"`  // 进程数

    // 系统运行时间
    Uptime uint64 `json:"uptime"` // 运行时间(秒)

    // 温度和GPU(可选)
    Temperatures string  `gorm:"type:text" json:"temperatures,omitempty"` // 温度传感器(JSON)
    GPUUsage     float64 `json:"gpu_usage,omitempty"`                    // GPU使用率(%)
}
```

**索引**:
- `host_node_id, timestamp`:复合索引(用于时间范围查询)
- `timestamp`:时间索引(用于数据清理)

**数据保留策略**:
- 实时数据:24小时,原始精度
- 短期数据:7天,每分钟聚合
- 长期数据:30天,每小时聚合

---

### 4. HostGroup(主机分组)

主机的分类组织,支持多对多关系。

```go
type HostGroup struct {
    gorm.Model

    Name        string `gorm:"uniqueIndex;not null"` // 分组名称
    Description string `gorm:"type:text"`            // 分组描述

    // 关联关系
    HostNodes []HostNode `gorm:"many2many:host_group_hosts;"` // 关联的主机
}
```

**关联关系**:
- 多对多关联HostNode
- 通过中间表`host_group_hosts`实现

---

### 5. ServiceMonitor(服务探测规则)

服务监控配置,定义探测任务的参数。

```go
type ServiceMonitor struct {
    gorm.Model

    // 基本信息
    Name   string `gorm:"not null"` // 服务名称
    Enable bool   `gorm:"default:true"` // 是否启用

    // 探测配置
    Type     uint8  `gorm:"not null"` // 探测类型(1=HTTP,2=TCP,3=ICMP)
    Target   string `gorm:"not null"` // 目标地址
    Duration uint64 `gorm:"not null"` // 探测频率(秒)
    Timeout  uint64 `gorm:"default:10"` // 超时时间(秒)
    Retry    int    `gorm:"default:3"` // 重试次数

    // 执行范围
    ExecuteOn    string `gorm:"type:text"` // 执行主机/分组(JSON配置)
    SkipHosts    string `gorm:"type:text"` // 跳过的主机ID(JSON数组)

    // 告警配置
    EnableAlert     bool   `gorm:"default:false"` // 启用告警
    FailThreshold   int    `gorm:"default:3"`     // 失败次数阈值
    NotificationID  uint   `gorm:"default:0"`     // 通知组ID

    // 延迟告警
    LatencyAlert    bool    `gorm:"default:false"` // 延迟告警
    MaxLatency      float32 `gorm:"default:0"`     // 最大延迟(秒)
}
```

**探测类型枚举**:
```go
const (
    ProbeTypeHTTP = 1 // HTTP GET请求
    ProbeTypeTCP  = 2 // TCP连接
    ProbeTypeICMP = 3 // ICMP Ping
)
```

**验证规则**:
- Type必须在1-3范围内
- Duration>=10秒(避免过于频繁)
- Timeout<Duration
- FailThreshold>=1

---

### 6. ServiceProbeResult(服务探测结果)

每次服务探测的执行结果记录。

```go
type ServiceProbeResult struct {
    gorm.Model

    ServiceMonitorID uint      `gorm:"index;not null"` // 关联探测规则
    HostNodeID       uint      `gorm:"index;not null"` // 执行探测的主机
    Timestamp        time.Time `gorm:"index;not null"` // 探测时间

    // 结果
    Success      bool    `json:"success"`       // 是否成功
    Latency      float32 `json:"latency"`       // 响应时间(秒)
    StatusCode   int     `json:"status_code,omitempty"`   // HTTP状态码
    ErrorMessage string  `gorm:"type:text" json:"error_message,omitempty"` // 错误信息
}
```

**索引**:
- `service_monitor_id, timestamp`:复合索引
- `host_node_id, timestamp`:复合索引

**数据保留策略**:
- 保留7天详细记录
- 聚合为ServiceAvailability统计数据

---

### 7. ServiceAvailability(服务可用性统计)

聚合服务的可用性数据,用于趋势分析。

```go
type ServiceAvailability struct {
    gorm.Model

    ServiceMonitorID uint      `gorm:"index;not null"` // 关联探测规则
    Period           string    `gorm:"index;not null"` // 统计周期(hour/day/week/month)
    StartTime        time.Time `gorm:"index;not null"` // 周期开始时间

    // 统计数据
    TotalProbes   int     `json:"total_probes"`   // 总探测次数
    SuccessProbes int     `json:"success_probes"` // 成功次数
    FailProbes    int     `json:"fail_probes"`    // 失败次数
    Availability  float64 `json:"availability"`   // 可用率(%)

    // 延迟统计
    AvgLatency float32 `json:"avg_latency"` // 平均延迟(秒)
    MinLatency float32 `json:"min_latency"` // 最小延迟(秒)
    MaxLatency float32 `json:"max_latency"` // 最大延迟(秒)
}
```

**索引**:
- `service_monitor_id, period, start_time`:唯一索引

---

### 8. WebSSHSession(WebSSH会话)

Web终端会话的生命周期管理。

```go
type WebSSHSession struct {
    gorm.Model

    SessionID  string    `gorm:"uniqueIndex;not null"` // 会话ID(UUID)
    UserID     uint      `gorm:"index;not null"`       // 用户ID
    HostNodeID uint      `gorm:"index;not null"`       // 主机ID

    // 会话状态
    Status      string    `gorm:"index;not null"` // 状态(active/closed)
    StartTime   time.Time `gorm:"not null"`       // 开始时间
    LastActive  time.Time `gorm:"not null"`       // 最后活跃时间
    EndTime     *time.Time                        // 结束时间

    // 客户端信息
    ClientIP    string    `json:"client_ip"`    // 客户端IP
    UserAgent   string    `gorm:"type:text" json:"user_agent"` // User-Agent
}
```

**状态枚举**:
```go
const (
    SessionStatusActive = "active" // 连接中
    SessionStatusClosed = "closed" // 已关闭
)
```

**生命周期**:
1. 用户请求WebSSH → 创建Session(status=active)
2. 定期更新LastActive(心跳)
3. 超时或主动关闭 → 更新status=closed,设置EndTime

---

### 9. MonitorAlertRule(告警规则)

定义监控告警的触发条件。

```go
type MonitorAlertRule struct {
    gorm.Model

    // 基本信息
    Name        string `gorm:"not null"` // 规则名称
    Enable      bool   `gorm:"default:true"` // 是否启用
    Type        string `gorm:"not null"` // 规则类型(host_monitor/service_probe)

    // 触发条件
    TargetIDs   string `gorm:"type:text"` // 关联的主机/服务ID(JSON数组)
    Condition   string `gorm:"type:text"` // 条件表达式(如"cpu > 90")
    Duration    int    `gorm:"default:300"` // 持续时长(秒)

    // 通知配置
    NotificationGroupID uint   `gorm:"default:0"` // 通知组ID
    AlertLevel          string `gorm:"default:warning"` // 告警级别(info/warning/critical)

    // 抑制配置
    SilencePeriod int `gorm:"default:300"` // 静默期(秒,避免重复告警)
}
```

**规则类型枚举**:
```go
const (
    AlertTypeHostMonitor  = "host_monitor"  // 主机监控告警
    AlertTypeServiceProbe = "service_probe" // 服务探测告警
)
```

**告警级别枚举**:
```go
const (
    AlertLevelInfo     = "info"     // 信息
    AlertLevelWarning  = "warning"  // 警告
    AlertLevelCritical = "critical" // 严重
)
```

---

### 10. AlertEvent(告警事件)

触发的告警实例记录。

```go
type AlertEvent struct {
    gorm.Model

    AlertRuleID uint      `gorm:"index;not null"` // 关联告警规则
    TargetID    uint      `gorm:"index;not null"` // 触发对象ID(主机/服务)
    TargetType  string    `gorm:"not null"`       // 对象类型(host/service)

    // 事件信息
    TriggerTime time.Time `gorm:"index;not null"` // 触发时间
    RecoverTime *time.Time                        // 恢复时间
    Status      string    `gorm:"index;not null"` // 状态(pending/firing/resolved/silenced)
    Level       string    `gorm:"not null"`       // 告警级别

    // 告警内容
    Title       string `gorm:"not null"`     // 告警标题
    Message     string `gorm:"type:text"`    // 告警消息
    CurrentValue string `gorm:"type:text"`   // 当前值(JSON)

    // 处理状态
    Acknowledged bool      `gorm:"default:false"` // 是否已确认
    AckUser      uint      `gorm:"default:0"`     // 确认用户ID
    AckTime      *time.Time                       // 确认时间
    AckNote      string    `gorm:"type:text"`     // 确认备注
}
```

**状态枚举**:
```go
const (
    AlertStatusPending  = "pending"  // 待触发(条件满足但未达到持续时长)
    AlertStatusFiring   = "firing"   // 告警中
    AlertStatusResolved = "resolved" // 已恢复
    AlertStatusSilenced = "silenced" // 已静默
)
```

---

### 11. AgentConnection(Agent连接状态)

Agent的连接信息和状态追踪。

```go
type AgentConnection struct {
    gorm.Model

    HostNodeID uint      `gorm:"uniqueIndex;not null"` // 关联主机节点

    // 连接信息
    ConnectedAt time.Time `gorm:"not null"` // 连接建立时间
    LastHeartbeat time.Time `gorm:"index;not null"` // 最后心跳时间
    Status      string    `gorm:"index;not null"` // 状态(online/offline)

    // Agent信息
    AgentVersion string `json:"agent_version"` // Agent版本
    IPAddress    string `json:"ip_address"`    // Agent IP地址
    GeoLocation  string `gorm:"type:text" json:"geo_location,omitempty"` // 地理位置(JSON)

    // 流ID(运行时)
    StreamID string `gorm:"-" json:"-"` // gRPC Stream ID(不持久化)
}
```

**状态枚举**:
```go
const (
    AgentStatusOnline  = "online"  // 在线
    AgentStatusOffline = "offline" // 离线
)
```

**心跳机制**:
- Agent每10秒发送心跳
- Server更新LastHeartbeat
- 超过60秒未收到心跳 → 标记offline

---

## 实体关系图

```
HostNode (1) ←→ (1) HostInfo
    ↓ (1)
    ↓
    ↓ (N) HostState
    ↓
    ↓ (1)
    ↓
    ↓ (1) AgentConnection
    ↓
    ↓ (N)
    ↓
HostGroup (N) ←→ (N) HostNode

HostNode (1) → (N) WebSSHSession
User (1) → (N) WebSSHSession

ServiceMonitor (1) → (N) ServiceProbeResult
HostNode (1) → (N) ServiceProbeResult

ServiceMonitor (1) → (N) ServiceAvailability

MonitorAlertRule (1) → (N) AlertEvent
HostNode/ServiceMonitor → AlertEvent (多态关联)
```

## 数据迁移

使用GORM AutoMigrate自动创建表结构:

```go
// internal/db/migrate.go
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.HostNode{},
        &models.HostInfo{},
        &models.HostState{},
        &models.HostGroup{},
        &models.ServiceMonitor{},
        &models.ServiceProbeResult{},
        &models.ServiceAvailability{},
        &models.WebSSHSession{},
        &models.MonitorAlertRule{},
        &models.AlertEvent{},
        &models.AgentConnection{},
    )
}
```

## 数据访问层

使用仓储模式封装数据访问:

```go
// internal/repository/host_repository.go
type HostRepository interface {
    Create(ctx context.Context, host *models.HostNode) error
    GetByID(ctx context.Context, id uint) (*models.HostNode, error)
    GetByUUID(ctx context.Context, uuid string) (*models.HostNode, error)
    List(ctx context.Context, filter HostFilter) ([]*models.HostNode, int64, error)
    Update(ctx context.Context, host *models.HostNode) error
    Delete(ctx context.Context, id uint) error

    // 状态相关
    SaveState(ctx context.Context, state *models.HostState) error
    GetLatestStates(ctx context.Context, hostID uint, limit int) ([]*models.HostState, error)
    GetStatesByTimeRange(ctx context.Context, hostID uint, start, end time.Time) ([]*models.HostState, error)
}

// internal/repository/service_repository.go
type ServiceRepository interface {
    Create(ctx context.Context, service *models.ServiceMonitor) error
    GetByID(ctx context.Context, id uint) (*models.ServiceMonitor, error)
    List(ctx context.Context, filter ServiceFilter) ([]*models.ServiceMonitor, int64, error)
    Update(ctx context.Context, service *models.ServiceMonitor) error
    Delete(ctx context.Context, id uint) error

    // 探测结果
    SaveProbeResult(ctx context.Context, result *models.ServiceProbeResult) error
    GetProbeHistory(ctx context.Context, serviceID uint, limit int) ([]*models.ServiceProbeResult, error)
    GetAvailability(ctx context.Context, serviceID uint, period string, start time.Time) (*models.ServiceAvailability, error)
}

// internal/repository/alert_repository.go
type AlertRepository interface {
    CreateRule(ctx context.Context, rule *models.MonitorAlertRule) error
    GetRuleByID(ctx context.Context, id uint) (*models.MonitorAlertRule, error)
    ListRules(ctx context.Context, filter AlertRuleFilter) ([]*models.MonitorAlertRule, int64, error)
    UpdateRule(ctx context.Context, rule *models.MonitorAlertRule) error
    DeleteRule(ctx context.Context, id uint) error

    // 告警事件
    CreateEvent(ctx context.Context, event *models.AlertEvent) error
    GetEventByID(ctx context.Context, id uint) (*models.AlertEvent, error)
    ListEvents(ctx context.Context, filter AlertEventFilter) ([]*models.AlertEvent, int64, error)
    UpdateEvent(ctx context.Context, event *models.AlertEvent) error
    AcknowledgeEvent(ctx context.Context, eventID, userID uint, note string) error
}
```

## 总结

数据模型设计遵循以下原则:
1. **关注点分离**:元数据(HostNode)与监控数据(HostState)分离
2. **时序数据优化**:HostState和ProbeResult使用时间索引,支持高效范围查询
3. **聚合统计**:ServiceAvailability预聚合数据,减少实时计算
4. **审计追溯**:WebSSHSession和AlertEvent记录完整生命周期
5. **扩展性**:JSON字段存储灵活配置(如分组、执行范围)
6. **性能考虑**:合理索引设计,支持分页查询和批量操作
