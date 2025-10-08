# Tiga Development TODO List

> Last Updated: 2025-10-07
> Branch: 002-nezha-webssh
> Priority: WebSSH Terminal & Service Probe Features

## 🎯 Current Sprint Focus

### 🖥️ WebSSH Terminal Implementation (6 days total)

#### Phase 1: Backend Enhancement (2 days)

##### WebSocket Handler (`internal/api/handlers/webssh_handler.go`)
- [x] Complete HandleWebSocket method implementation
  - [x] Define WebSocket message protocol (JSON format)
  - [x] Handle resize events from frontend
  - [x] Implement ping/pong for connection keep-alive
  - [x] Add graceful connection close handling
- [ ] Implement connection pool management
  - [ ] Maximum connections per user limit
  - [ ] Connection timeout (default: 30 minutes)
  - [ ] Connection cleanup on disconnect
- [ ] Add error handling and reconnection mechanism
  - [ ] Handle network interruptions
  - [ ] Automatic reconnection with backoff
  - [ ] Error message propagation to frontend
- [ ] Add session authentication and authorization
  - [ ] Verify user permissions for host access
  - [ ] Session token validation
  - [ ] Rate limiting for connections

##### Terminal Manager (`internal/services/host/terminal_manager.go`)
- [ ] Complete PTY session management
  - [ ] Create PTY with proper size
  - [ ] Handle PTY resize events
  - [ ] Manage PTY lifecycle
- [ ] Implement input/output stream processing
  - [ ] Handle stdin from WebSocket
  - [ ] Stream stdout/stderr to WebSocket
  - [ ] Buffer management for large outputs
- [ ] Add session timeout handling
  - [ ] Idle timeout detection (configurable)
  - [ ] Warning before timeout
  - [ ] Graceful session termination
- [ ] Implement session recording (optional)
  - [x] Record terminal sessions to file
  - [x] Playback capability
  - [ ] Storage management

##### Agent Manager (`internal/services/host/agent_manager.go`)
- [ ] Implement Agent SSH proxy protocol
  - [ ] Define gRPC service for SSH proxy
  - [ ] Message format for SSH commands
  - [ ] Stream handling for terminal I/O
- [ ] Forward SSH connections to Agent
  - [ ] Connection request routing
  - [ ] Bidirectional stream forwarding
  - [ ] Connection state tracking
- [ ] Add security authentication mechanism
  - [ ] Agent authentication
  - [ ] End-to-end encryption
  - [ ] Command whitelisting/blacklisting

#### Phase 2: Agent Implementation (2 days)

##### SSH Proxy Service (`cmd/tiga-agent/ssh_proxy.go`) - NEW FILE
- [ ] Create SSH proxy service structure
  ```go
  type SSHProxy struct {
    maxConnections int
    connections    map[string]*SSHConnection
    mu            sync.RWMutex
  }
  ```
- [ ] Implement SSH connection handling
  - [ ] Receive SSH requests from server
  - [ ] Create local SSH client connection
  - [ ] Handle authentication (key/password)
  - [ ] Manage connection lifecycle
- [ ] Implement bidirectional data forwarding
  - [ ] Forward stdin to SSH session
  - [ ] Stream stdout/stderr back to server
  - [ ] Handle control sequences properly
- [ ] Add security mechanisms
  - [ ] Command execution restrictions
  - [ ] Audit logging for all commands
  - [ ] Connection encryption using TLS
  - [ ] Rate limiting per connection
- [ ] Implement error handling
  - [ ] Connection failure recovery
  - [ ] Graceful disconnection
  - [ ] Error reporting to server

##### Agent Integration (`cmd/tiga-agent/main.go`)
- [ ] Integrate SSH proxy into agent main loop
- [ ] Add configuration for SSH proxy
  - [ ] Enable/disable SSH proxy
  - [ ] Maximum concurrent connections
  - [ ] Allowed command whitelist
- [ ] Add health checks for SSH service
- [ ] Implement metrics collection for SSH sessions

#### Phase 3: Frontend Optimization (1 day)

##### Terminal UI (`ui/src/pages/hosts/host-ssh-page.tsx`)
- [ ] 适配新的 WebSSH JSON/Base64 消息协议（输入、输出、ping/pong 等）
- [ ] Optimize xterm.js configuration
  - [ ] Fine-tune scrollback buffer size
  - [ ] Configure cursor style and blinking
  - [ ] Set appropriate font family and size
  - [ ] Enable/disable sound
- [ ] Implement adaptive terminal sizing
  - [ ] Detect container size changes
  - [ ] Calculate rows/cols based on container
  - [ ] Send resize events to backend
  - [ ] Handle mobile responsive layout
- [ ] Add copy/paste support
  - [ ] Right-click context menu
  - [ ] Ctrl+C/Ctrl+V handling
  - [ ] Selection highlighting
  - [ ] Clipboard API integration
- [ ] Implement terminal themes
  - [ ] Predefined themes (dark, light, solarized, etc.)
  - [ ] Custom theme configuration
  - [ ] Theme persistence in localStorage
  - [ ] Real-time theme switching

##### Connection Management UI
- [ ] Add connection status indicator
  - [ ] Connected/Disconnected states
  - [ ] Reconnecting animation
  - [ ] Latency display
  - [ ] Data transfer indicators
- [ ] Implement automatic reconnection
  - [ ] Exponential backoff strategy
  - [ ] Maximum retry attempts
  - [ ] User notification on failure
- [ ] Enhance error notifications
  - [ ] User-friendly error messages
  - [ ] Actionable error suggestions
  - [ ] Error log viewer
- [ ] Add session management features
  - [ ] Multiple tab support
  - [ ] Session history
  - [ ] Quick connect shortcuts

#### Phase 4: Integration Testing (1 day)

##### End-to-End Tests
- [ ] Connection establishment test
  - [ ] Successful connection flow
  - [ ] Authentication failure handling
  - [ ] Permission denied scenarios
- [ ] Multi-session concurrency test
  - [ ] Open 100+ concurrent sessions
  - [ ] Memory usage monitoring
  - [ ] CPU usage monitoring
- [ ] Network disruption recovery test
  - [ ] Simulate network interruption
  - [ ] Verify automatic reconnection
  - [ ] Check session state preservation
- [ ] Performance stress test
  - [ ] Large output handling (e.g., cat large file)
  - [ ] Rapid input testing
  - [ ] Long-running session stability

##### Security Tests
- [ ] Command injection prevention
- [ ] Session hijacking prevention
- [ ] Rate limiting verification
- [ ] Authentication bypass attempts

---

### 📊 Service Probe Implementation (9 days total)

#### Phase 1: Backend Enhancement (2 days)

##### Probe Scheduler (`internal/services/monitor/probe_scheduler.go`)
- [ ] Implement HTTP/HTTPS probe
  - [ ] GET/POST/HEAD methods
  - [ ] Custom headers support
  - [ ] Response code validation
  - [ ] Response time measurement
  - [ ] Content validation (regex)
  - [ ] SSL certificate validation
- [ ] Implement TCP port probe
  - [ ] Connection establishment test
  - [ ] Custom payload sending
  - [ ] Response validation
  - [ ] Connection timeout handling
- [ ] Implement ICMP ping probe
  - [ ] Packet loss calculation
  - [ ] RTT measurement
  - [ ] Jitter calculation
  - [ ] Multi-packet statistics
- [ ] Add custom script probe
  - [ ] Script execution framework
  - [ ] Timeout enforcement
  - [ ] Output parsing
  - [ ] Exit code validation

##### Probe Result Model (`internal/models/service_probe_result.go`)
- [ ] Enhance data model
  ```go
  type ServiceProbeResult struct {
    ID           uuid.UUID
    MonitorID    uuid.UUID
    AgentID      uuid.UUID
    Status       string // up/down/degraded
    ResponseTime int64  // microseconds
    StatusCode   int
    ErrorMessage string
    Metadata     JSONB // additional probe-specific data
    ProbeAt      time.Time
  }
  ```
- [ ] Add response time statistics
  - [ ] P50/P95/P99 percentiles
  - [ ] Average response time
  - [ ] Standard deviation
- [ ] Implement availability calculation
  - [ ] Uptime percentage
  - [ ] MTBF (Mean Time Between Failures)
  - [ ] MTTR (Mean Time To Recovery)
- [ ] Add trend analysis
  - [ ] Response time trends
  - [ ] Availability trends
  - [ ] Anomaly detection

##### Service Repository (`internal/repository/service_repository.go`)
- [ ] Optimize probe result storage
  - [ ] Batch insert for results
  - [ ] Data retention policies
  - [ ] Automatic old data cleanup
- [ ] Implement time-series queries
  - [ ] Range queries with aggregation
  - [ ] Downsampling for long ranges
  - [ ] Moving averages
- [ ] Add statistical aggregation
  - [ ] Hourly/daily/weekly summaries
  - [ ] Service group aggregations
  - [ ] Cross-service comparisons
- [ ] Create availability reports
  - [ ] SLA compliance calculation
  - [ ] Incident detection
  - [ ] Outage duration tracking

#### Phase 2: Agent Probe Executor (2 days)

##### Probe Executor (`cmd/tiga-agent/prober/executor.go`) - NEW FILE
- [ ] Create probe executor framework
  ```go
  type ProbeExecutor struct {
    httpClient  *http.Client
    maxWorkers  int
    workQueue   chan ProbeTask
    results     chan ProbeResult
  }
  ```
- [ ] Implement distributed probe execution
  - [ ] Receive probe tasks from server
  - [ ] Queue management with priorities
  - [ ] Concurrent execution with worker pool
  - [ ] Result batching for efficiency
- [ ] Report probe results to server
  - [ ] Real-time result streaming
  - [ ] Batch upload for efficiency
  - [ ] Retry on failure
  - [ ] Result compression

##### HTTP Probe (`cmd/tiga-agent/prober/http.go`) - NEW FILE
- [ ] Implement HTTP/HTTPS probing
  - [ ] Support all HTTP methods
  - [ ] Custom headers and body
  - [ ] Follow redirects (configurable)
  - [ ] Cookie jar support
- [ ] Add authentication support
  - [ ] Basic authentication
  - [ ] Bearer token
  - [ ] Custom authentication headers
- [ ] Implement response validation
  - [ ] Status code checks
  - [ ] Response body regex matching
  - [ ] JSON path validation
  - [ ] Response time thresholds

##### TCP Probe (`cmd/tiga-agent/prober/tcp.go`) - NEW FILE
- [ ] Implement TCP connection testing
  - [ ] Connection establishment
  - [ ] TLS support
  - [ ] Custom handshake protocols
- [ ] Add protocol-specific probes
  - [ ] Redis PING
  - [ ] MySQL connection test
  - [ ] PostgreSQL connection test
  - [ ] MongoDB connection test

##### DNS Probe (`cmd/tiga-agent/prober/dns.go`) - NEW FILE
- [ ] Implement DNS resolution testing
  - [ ] A/AAAA record lookup
  - [ ] MX/TXT/CNAME queries
  - [ ] Response time measurement
  - [ ] Multiple DNS server support

##### Script Probe (`cmd/tiga-agent/prober/script.go`) - NEW FILE
- [ ] Implement custom script execution
  - [ ] Shell script support
  - [ ] Python script support
  - [ ] Output parsing
  - [ ] Resource limits (CPU/Memory)

#### Phase 3: Frontend Development (3 days)

##### Service Monitor List (`ui/src/pages/hosts/service-monitor-list.tsx`) - NEW FILE
- [ ] Create list page component
  - [ ] Table view with sorting/filtering
  - [ ] Card view for overview
  - [ ] Status indicators (up/down/degraded)
  - [ ] Quick actions menu
- [ ] Implement CRUD operations
  - [ ] Create monitor dialog
  - [ ] Edit monitor form
  - [ ] Delete with confirmation
  - [ ] Bulk operations support
- [ ] Add search and filters
  - [ ] Search by name/URL/host
  - [ ] Filter by status
  - [ ] Filter by type (HTTP/TCP/etc)
  - [ ] Filter by tag/group
- [ ] Implement batch operations
  - [ ] Enable/disable multiple monitors
  - [ ] Delete multiple monitors
  - [ ] Export/import configurations

##### Service Monitor Detail (`ui/src/pages/hosts/service-monitor-detail.tsx`) - NEW FILE
- [ ] Create detail page layout
  - [ ] Monitor configuration display
  - [ ] Current status card
  - [ ] Recent probe results table
  - [ ] Alert configuration section
- [ ] Add availability visualization
  - [ ] Availability percentage gauge
  - [ ] Uptime/downtime timeline
  - [ ] Incident list with duration
  - [ ] SLA compliance indicator
- [ ] Implement response time charts
  - [ ] Line chart for response times
  - [ ] Histogram distribution
  - [ ] Percentile trends
  - [ ] Comparison with baseline
- [ ] Add probe result logs
  - [ ] Paginated log viewer
  - [ ] Log filtering by status
  - [ ] Export logs feature
  - [ ] Real-time log streaming

##### Service Monitor Form (`ui/src/components/hosts/service-monitor-form.tsx`) - NEW FILE
- [ ] Create comprehensive form
  - [ ] Basic information (name, description)
  - [ ] Probe type selection
  - [ ] Target configuration
  - [ ] Schedule configuration
- [ ] Add probe-specific fields
  - [ ] HTTP: URL, method, headers, body
  - [ ] TCP: host, port, timeout
  - [ ] ICMP: host, packet count
  - [ ] Script: script content, interpreter
- [ ] Implement validation rules
  - [ ] URL format validation
  - [ ] Port range validation
  - [ ] Cron expression validation
  - [ ] Script syntax checking
- [ ] Add advanced options
  - [ ] Retry configuration
  - [ ] Timeout settings
  - [ ] Alert thresholds
  - [ ] Notification channels

##### Availability Chart (`ui/src/components/hosts/availability-chart.tsx`) - NEW FILE
- [ ] Create chart component
  - [ ] Availability heatmap (calendar view)
  - [ ] Response time line chart
  - [ ] Status timeline visualization
  - [ ] Statistical summary cards
- [ ] Add interactivity
  - [ ] Zoom and pan
  - [ ] Tooltip with details
  - [ ] Click to view incident
  - [ ] Export chart as image
- [ ] Implement real-time updates
  - [ ] WebSocket subscription
  - [ ] Smooth animations
  - [ ] Auto-refresh toggle
  - [ ] Notification on status change

#### Phase 4: Visualization & Alerting (2 days)

##### Monitoring Dashboard (`ui/src/pages/hosts/monitoring-dashboard.tsx`) - NEW FILE
- [ ] Create dashboard layout
  - [ ] Service status overview grid
  - [ ] Real-time alert feed
  - [ ] Top issues widget
  - [ ] Performance metrics summary
- [ ] Implement service topology
  - [ ] Interactive service map
  - [ ] Dependency visualization
  - [ ] Impact analysis view
  - [ ] Group by tags/categories
- [ ] Add real-time updates
  - [ ] WebSocket connection for live data
  - [ ] Status change animations
  - [ ] Alert notifications
  - [ ] Sound alerts (optional)

##### Alert Configuration (`ui/src/pages/hosts/alert-rules.tsx`)
- [ ] Enhance alert rule builder
  - [ ] Visual rule builder
  - [ ] Condition templates
  - [ ] Multi-condition support
  - [ ] Schedule-based muting
- [ ] Configure notification channels
  - [ ] Email configuration
  - [ ] Webhook setup
  - [ ] Slack integration
  - [ ] DingTalk integration
- [ ] Implement alert testing
  - [ ] Test rule execution
  - [ ] Preview notifications
  - [ ] Dry run mode
- [ ] Add alert history
  - [ ] Alert timeline
  - [ ] Alert analytics
  - [ ] False positive tracking
  - [ ] Alert acknowledgment

##### Alert Event Handler (`internal/services/alert/probe_alert_handler.go`) - NEW FILE
- [ ] Create alert evaluation engine
  - [ ] Rule condition evaluation
  - [ ] Alert state machine
  - [ ] Flapping detection
  - [ ] Alert deduplication
- [ ] Implement notification dispatcher
  - [ ] Channel selection logic
  - [ ] Retry with backoff
  - [ ] Notification templating
  - [ ] Rate limiting

---

## 🔧 Technical Debt & Improvements

### Code Quality
- [ ] Add unit tests for WebSSH components
  - [ ] Session manager tests
  - [ ] Terminal manager tests
  - [ ] WebSocket handler tests
- [ ] Add unit tests for probe components
  - [ ] Probe executor tests
  - [ ] Scheduler tests
  - [ ] Alert handler tests
- [ ] Add integration tests
  - [ ] End-to-end WebSSH flow
  - [ ] Probe execution flow
  - [ ] Alert triggering flow
- [ ] Improve error handling
  - [ ] Consistent error types
  - [ ] Error wrapping with context
  - [ ] User-friendly error messages

### Performance Optimization
- [ ] Optimize WebSocket connection pooling
- [ ] Implement probe result data compression
- [ ] Add caching for frequently accessed data
- [ ] Optimize database queries with indexes
- [ ] Implement connection multiplexing for Agent

### Security Enhancements
- [ ] Add rate limiting for WebSSH connections
- [ ] Implement session recording encryption
- [ ] Add probe result data encryption at rest
- [ ] Implement API request signing for Agent
- [ ] Add security headers for all endpoints

### Documentation
- [ ] Write WebSSH user guide
- [ ] Create service probe configuration guide
- [ ] Document Agent installation process
- [ ] Add API documentation for new endpoints
- [ ] Create troubleshooting guide

---

## 📈 Success Metrics

### WebSSH Terminal
- [ ] Support 100+ concurrent sessions
- [ ] Connection latency < 100ms
- [ ] 99.9% connection stability
- [ ] Support for all common terminal operations
- [ ] Zero security vulnerabilities

### Service Probe
- [ ] Support 1000+ monitored services
- [ ] Probe execution accuracy > 99.99%
- [ ] Alert false positive rate < 1%
- [ ] Data retention for 90 days
- [ ] Query response time < 500ms

---

## 🚀 Quick Start Tasks (Do First!)

### Today's Focus
1. [ ] Complete WebSocket message protocol design
2. [ ] Implement basic HTTP probe functionality
3. [ ] Create frontend page skeletons

### Tomorrow's Focus
1. [ ] Implement Agent SSH proxy basic structure
2. [ ] Add probe result storage optimization
3. [ ] Design UI components for service monitor

### This Week's Goals
1. [ ] Working WebSSH prototype
2. [ ] Basic probe execution with HTTP support
3. [ ] Frontend pages with mock data

---

## 📝 Notes

### Dependencies
- xterm.js (already installed)
- React Query for data fetching
- Recharts for visualization
- golang.org/x/crypto/ssh for SSH client
- robfig/cron for probe scheduling

### Known Issues
- WebSocket connection drops on network change
- Agent reconnection sometimes fails
- Probe results table needs indexing

### References
- [xterm.js documentation](https://xtermjs.org/docs/)
- [SSH Protocol RFC](https://tools.ietf.org/html/rfc4254)
- [Prometheus metric types](https://prometheus.io/docs/concepts/metric_types/)

---

## 🎯 Definition of Done

### For Each Feature
- [ ] Code implemented and reviewed
- [ ] Unit tests written and passing
- [ ] Integration tests passing
- [ ] Documentation updated
- [ ] UI/UX reviewed and approved
- [ ] Performance benchmarks met
- [ ] Security review completed
- [ ] Deployed to staging environment

---

*Last updated: 2025-10-07 by Claude*
*Next review: After Phase 1 completion*

---

# 🔍 服务监控增强计划（基于 Nezha 实现参考）

> 参考源码分析: nezha/service/singleton/servicesentinel.go
> 目标: 实现分布式探测 + 30天可用性热力图 + 节点间网络拓扑

## 📊 核心特性

### 1. 三种探测类型
- **ICMP Ping**: 使用 `prometheus-community/pro-bing`，发送 5 个包计算平均 RTT
- **TCP Ping**: TCP 连接测试，记录连接建立耗时
- **HTTP GET**: HTTP 请求测试 + TLS 证书信息提取

### 2. 分布式探测架构
- Agent 端执行探测任务并上报结果（含 HostID）
- Server 端按 `(ServiceID, HostID)` 分别存储
- 支持构建节点间 N×N 延迟矩阵

### 3. 30 天可用性展示
- 使用 `[30]数组` 存储每日数据（索引 29 是今天）
- 每天凌晨自动左移数据
- 状态码: Good(>95%)、LowAvailability(80-95%)、Down(<80%)

### 4. 智能告警
- 状态变更告警（Good ↔ Low ↔ Down）
- 延迟异常告警（超出 MinLatency/MaxLatency）
- TLS 证书过期提醒（7 天内）
- 静音标签防止重复告警

---

## 🚀 实施阶段

### Phase 1: 数据模型与基础框架 (Week 1-2)

#### 数据模型设计

```go
// 服务探测任务配置 (复用并增强 ServiceMonitor)
type ServiceMonitor struct {
    Common
    Name                string
    Type                uint8  // 1=HTTP, 2=ICMP, 3=TCP
    Target              string // URL/IP/Host:Port
    Duration            uint64 // 探测间隔（秒）
    SkipHostsRaw        string `json:"-"` // JSON: map[uint64]bool
    Cover               uint8  // 0=监控所有节点, 1=忽略指定节点

    // 告警配置
    Notify              bool
    NotificationGroupID uint64
    MinLatency          float32
    MaxLatency          float32
    LatencyNotify       bool

    // 触发任务
    EnableTriggerTask      bool
    EnableShowInService    bool
    FailTriggerTasksRaw    string // JSON: []uint64
    RecoverTriggerTasksRaw string // JSON: []uint64

    // 运行时
    SkipHosts        map[uint64]bool `gorm:"-"`
    FailTriggerTasks []uint64        `gorm:"-"`
    RecoverTriggerTasks []uint64     `gorm:"-"`
    CronJobID        cron.EntryID    `gorm:"-"`
}

// 探测历史数据 (复用并增强 ServiceHistory)
type ServiceHistory struct {
    ID        uint64
    CreatedAt time.Time `gorm:"index:idx_service_host_time"`
    ServiceID uint64    `gorm:"index:idx_service_host_time"`
    HostID    uint64    `gorm:"index:idx_service_host_time"` // 0=汇总数据
    AvgDelay  float32   `gorm:"index:idx_service_host_time"` // 毫秒
    Up        uint64
    Down      uint64
    Data      string    // 额外信息（TLS证书、错误信息）
}

// 30天聚合数据（内存缓存）
type ServiceResponseItem struct {
    ServiceID   uint64
    ServiceName string
    Delay       *[30]float32 // 每日平均延迟
    Up          *[30]uint64  // 每日成功次数
    Down        *[30]uint64  // 每日失败次数
    TotalUp     uint64       // 30天总成功
    TotalDown   uint64       // 30天总失败
    CurrentUp   uint64       // 当前15分钟成功
    CurrentDown uint64       // 当前15分钟失败
}
```

#### 任务清单
- [ ] 创建/更新数据库迁移
- [ ] 实现 ServiceMonitor 模型的 BeforeSave/AfterFind
- [ ] 添加 ServiceHistory 的复合索引
- [ ] 实现 Repository 层查询方法
- [ ] 添加数据保留策略（90天自动清理）

---

### Phase 2: Server 端 ServiceSentinel (Week 2-3)

#### 核心组件结构

```go
// internal/services/monitor/service_sentinel.go
type ServiceSentinel struct {
    // 通道
    reportChannel chan ReportData        // Agent 上报通道 (buffer: 200)
    dispatchBus   chan<- *ServiceMonitor // 任务下发通道

    // 数据存储（需加锁访问）
    services         map[uint64]*ServiceMonitor      // 任务列表
    serviceList      []*ServiceMonitor               // 排序列表（用于遍历）
    monthlyStatus    map[uint64]*serviceResponseItem // 30天数据
    statusToday      map[uint64]*todayStats          // 今日统计
    currentStatus    map[uint64]*taskStatus          // 15分钟滑动窗口
    responsePing     map[uint64]map[uint64]*pingStore // Ping聚合缓存
    tlsCertCache     map[uint64]string               // TLS证书缓存

    // 锁
    servicesLock         sync.RWMutex
    serviceListLock      sync.RWMutex
    monthlyStatusLock    sync.Mutex
    responseDataStoreLock sync.RWMutex
}

// 今日统计
type todayStats struct {
    Up    uint64
    Down  uint64
    Delay float32
}

// 任务状态（15分钟窗口）
type taskStatus struct {
    lastStatus uint8
    t          time.Time
    result     []*pb.TaskResult // 最多30个样本
}

// Ping 聚合（减少写入频率）
type pingStore struct {
    count int
    ping  float32
}
```

#### 关键方法

```go
// 初始化
func NewServiceSentinel(dispatchBus chan<- *ServiceMonitor) (*ServiceSentinel, error)

// 从数据库加载历史数据
func (ss *ServiceSentinel) loadServiceHistory() error

// 接收 Agent 上报
func (ss *ServiceSentinel) Dispatch(r ReportData)

// 后台 worker 处理上报
func (ss *ServiceSentinel) worker()

// 每日凌晨左移数据
func (ss *ServiceSentinel) refreshMonthlyServiceStatus()

// CRUD 操作
func (ss *ServiceSentinel) Update(m *ServiceMonitor) error
func (ss *ServiceSentinel) Delete(ids []uint64)
func (ss *ServiceSentinel) Get(id uint64) (*ServiceMonitor, bool)
func (ss *ServiceSentinel) GetList() map[uint64]*ServiceMonitor

// 加载统计数据
func (ss *ServiceSentinel) LoadStats() map[uint64]*serviceResponseItem
func (ss *ServiceSentinel) CopyStats() map[uint64]ServiceResponseItem
```

#### 实施任务
- [ ] 实现 ServiceSentinel 结构体和初始化
- [ ] 集成 `robfig/cron` 定时任务调度
  ```go
  cronID, _ := cron.AddFunc(m.CronSpec(), func() {
      dispatchBus <- m
  })
  ```
- [ ] 实现 worker goroutine（消费 reportChannel）
- [ ] 实现 Ping 数据批量聚合逻辑
  ```go
  if ts.count == Conf.AvgPingCount {
      DB.Create(&ServiceHistory{...})
      ts.count = 0
  }
  ```
- [ ] 实现每日凌晨数据左移逻辑（cron: `0 0 0 * * *`）
- [ ] 实现启动时加载30天历史数据
- [ ] 处理状态变更（计算在线率并判断状态码）
  ```go
  upPercent := up * 100 / (up + down)
  stateCode := GetStatusCode(upPercent)
  ```

---

### Phase 3: Agent 端探测执行器 (Week 3-4)

#### gRPC Proto 定义

```protobuf
// proto/service.proto
message Task {
    uint64 id = 1;
    uint64 type = 2;  // 1=HTTP, 2=ICMP, 3=TCP
    string data = 3;  // URL/IP/Host:Port
}

message TaskResult {
    uint64 id = 1;
    uint64 type = 2;
    bool successful = 3;
    float delay = 4;    // 毫秒
    string data = 5;    // 错误信息或证书信息
}

service AgentService {
    rpc ExecuteTask(Task) returns (TaskResult);
}
```

#### 探测实现

```go
// cmd/tiga-agent/prober/icmp.go
func handleIcmpPing(target string) (delay float32, success bool, data string) {
    pinger, err := ping.NewPinger(target)
    if err != nil {
        return 0, false, err.Error()
    }

    pinger.SetPrivileged(true)
    pinger.Count = 5
    pinger.Timeout = 20 * time.Second

    if err := pinger.Run(); err != nil {
        return 0, false, err.Error()
    }

    stat := pinger.Statistics()
    if stat.PacketsRecv == 0 {
        return 0, false, "no packets received"
    }

    return float32(stat.AvgRtt.Milliseconds()), true, ""
}

// cmd/tiga-agent/prober/tcp.go
func handleTcpPing(target string) (delay float32, success bool, data string) {
    start := time.Now()
    conn, err := net.DialTimeout("tcp", target, 10*time.Second)
    if err != nil {
        return 0, false, err.Error()
    }
    defer conn.Close()

    return float32(time.Since(start).Milliseconds()), true, ""
}

// cmd/tiga-agent/prober/http.go
func handleHttpGet(url string) (delay float32, success bool, data string) {
    start := time.Now()
    resp, err := httpClient.Get(url)
    if err != nil {
        return 0, false, err.Error()
    }
    defer resp.Body.Close()

    // 读取响应体（丢弃）
    io.Copy(io.Discard, resp.Body)

    delay = float32(time.Since(start).Milliseconds())

    // 检查状态码
    if resp.StatusCode < 200 || resp.StatusCode > 399 {
        return delay, false, fmt.Sprintf("HTTP %d", resp.StatusCode)
    }

    // 提取 TLS 证书信息
    if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
        cert := resp.TLS.PeerCertificates[0]
        data = cert.Issuer.CommonName + "|" + cert.NotAfter.String()
    }

    return delay, true, data
}
```

#### 任务清单
- [ ] 在 proto 中添加 ServiceMonitor 任务类型
- [ ] Agent 接收任务并路由到对应处理函数
- [ ] 实现 ICMP Ping（依赖 `github.com/prometheus-community/pro-bing`）
- [ ] 实现 TCP Ping
- [ ] 实现 HTTP GET + TLS 证书提取
- [ ] Agent 上报结果到 Server（包含自己的 HostID）
- [ ] 添加探测超时和重试机制
- [ ] 实现探测结果批量上报（减少网络开销）

---

### Phase 4: 告警系统集成 (Week 4-5)

#### 状态码定义

```go
const (
    StatusNoData         = 1
    StatusGood           = 2  // >95%
    StatusLowAvailability = 3  // 80-95%
    StatusDown           = 4  // <80%
)

func GetStatusCode(upPercent float64) uint8 {
    if upPercent == 0 {
        return StatusNoData
    }
    if upPercent > 95 {
        return StatusGood
    }
    if upPercent > 80 {
        return StatusLowAvailability
    }
    return StatusDown
}

func StatusCodeToString(code uint8) string {
    switch code {
    case StatusGood:
        return "Good"
    case StatusLowAvailability:
        return "Low Availability"
    case StatusDown:
        return "Down"
    default:
        return "No Data"
    }
}
```

#### 告警检查逻辑

```go
// 在 worker 中调用
func (ss *ServiceSentinel) checkAlerts(report *ReportData, service *ServiceMonitor, newStatus uint8) {
    lastStatus := ss.currentStatus[service.ID].lastStatus

    // 1. 状态变更告警
    if newStatus == StatusDown || newStatus != lastStatus {
        if service.Notify && (lastStatus != 0 || newStatus == StatusDown) {
            msg := fmt.Sprintf("[%s] %s Reporter: %s, Error: %s",
                StatusCodeToString(newStatus),
                service.Name,
                hostName,
                report.Data)

            muteLabel := fmt.Sprintf("service_%d_status", service.ID)

            // 状态变更时清除静音
            if newStatus != lastStatus {
                notificationService.UnMute(service.NotificationGroupID, muteLabel)
            }

            go notificationService.Send(service.NotificationGroupID, msg, muteLabel)
        }
    }

    // 2. 延迟告警
    if service.LatencyNotify && report.Delay > 0 {
        minLabel := fmt.Sprintf("service_%d_latency_min", service.ID)
        maxLabel := fmt.Sprintf("service_%d_latency_max", service.ID)

        if report.Delay > service.MaxLatency {
            msg := fmt.Sprintf("[Latency] %s %.2fms > %.2fms, Reporter: %s",
                service.Name, report.Delay, service.MaxLatency, hostName)
            go notificationService.Send(service.NotificationGroupID, msg, maxLabel)
        } else if report.Delay < service.MinLatency {
            msg := fmt.Sprintf("[Latency] %s %.2fms < %.2fms, Reporter: %s",
                service.Name, report.Delay, service.MinLatency, hostName)
            go notificationService.Send(service.NotificationGroupID, msg, minLabel)
        } else {
            // 延迟正常，清除静音
            notificationService.UnMute(service.NotificationGroupID, minLabel)
            notificationService.UnMute(service.NotificationGroupID, maxLabel)
        }
    }

    // 3. TLS 证书告警
    if strings.HasPrefix(report.Data, "SSL证书错误：") {
        if !strings.HasSuffix(report.Data, "timeout") &&
           !strings.HasSuffix(report.Data, "EOF") {
            if service.Notify {
                msg := fmt.Sprintf("[TLS] Fetch cert info failed, Reporter: %s, Error: %s",
                    service.Name, report.Data)
                muteLabel := fmt.Sprintf("service_%d_tls_network", service.ID)
                go notificationService.Send(service.NotificationGroupID, msg, muteLabel)
            }
        }
    } else {
        // 解析证书信息
        certParts := strings.Split(report.Data, "|")
        if len(certParts) > 1 {
            expiresAt, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", certParts[1])

            // 7天内过期告警
            if expiresAt.Before(time.Now().AddDate(0, 0, 7)) {
                msg := fmt.Sprintf("[TLS] %s certificate expires at %s",
                    service.Name, expiresAt.Format("2006-01-02 15:04:05"))
                muteLabel := fmt.Sprintf("service_%d_tls_expire_%s", service.ID,
                    expiresAt.Format("20060102"))
                go notificationService.Send(service.NotificationGroupID, msg, muteLabel)
            }
        }
    }

    // 4. 触发任务
    if service.EnableTriggerTask && lastStatus != 0 {
        if newStatus == StatusGood && lastStatus != StatusGood {
            // 恢复任务
            go cronService.SendTriggerTasks(service.RecoverTriggerTasks, report.Reporter)
        } else if lastStatus == StatusGood && newStatus != StatusGood {
            // 失败任务
            go cronService.SendTriggerTasks(service.FailTriggerTasks, report.Reporter)
        }
    }
}
```

#### 任务清单
- [ ] 实现状态码计算和字符串转换
- [ ] 在 worker 中集成告警检查
- [ ] 实现静音标签机制（避免重复告警）
- [ ] 集成现有通知系统（`internal/services/notification`）
- [ ] 实现触发任务功能（失败/恢复时执行指定任务）
- [ ] 添加告警频率限制（避免告警风暴）

---

### Phase 5: API 接口层 (Week 5-6)

#### REST API 设计

```
# 服务监控任务管理
POST   /api/v1/service-monitors          创建监控任务
GET    /api/v1/service-monitors          列出所有任务
GET    /api/v1/service-monitors/:id      获取任务详情
PATCH  /api/v1/service-monitors/:id      更新任务
POST   /api/v1/batch-delete/service-monitors  批量删除

# 统计数据
GET    /api/v1/service-monitors/overview              获取所有任务概览（30天数据）
GET    /api/v1/service-monitors/:id/history           获取历史数据
GET    /api/v1/hosts/:id/service-history              获取节点的探测数据
GET    /api/v1/service-monitors/hosts-with-service    获取有监控数据的节点列表
```

#### Handler 实现示例

```go
// internal/api/handlers/service_monitor_handler.go

// @Summary List all service monitors
// @Description Get list of all service monitors with 30-day statistics
// @Tags service-monitor
// @Produce json
// @Success 200 {object} ServiceMonitorResponse
// @Router /api/v1/service-monitors/overview [get]
func (h *ServiceMonitorHandler) GetOverview(c *gin.Context) {
    stats := h.sentinelService.CopyStats()

    c.JSON(http.StatusOK, gin.H{
        "services": stats,
    })
}

// @Summary Create service monitor
// @Description Create a new service monitor task
// @Tags service-monitor
// @Accept json
// @Produce json
// @Param request body ServiceMonitorForm true "Monitor configuration"
// @Success 200 {object} IDResponse
// @Router /api/v1/service-monitors [post]
func (h *ServiceMonitorHandler) Create(c *gin.Context) {
    var form ServiceMonitorForm
    if err := c.ShouldBindJSON(&form); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 权限检查
    if !h.checkHostPermission(c, form.SkipHosts) {
        c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
        return
    }

    monitor := &models.ServiceMonitor{
        Name:                form.Name,
        Type:                form.Type,
        Target:              strings.TrimSpace(form.Target),
        Duration:            form.Duration,
        SkipHosts:           form.SkipHosts,
        Cover:               form.Cover,
        Notify:              form.Notify,
        NotificationGroupID: form.NotificationGroupID,
        MinLatency:          form.MinLatency,
        MaxLatency:          form.MaxLatency,
        LatencyNotify:       form.LatencyNotify,
        EnableTriggerTask:   form.EnableTriggerTask,
        EnableShowInService: form.EnableShowInService,
        FailTriggerTasks:    form.FailTriggerTasks,
        RecoverTriggerTasks: form.RecoverTriggerTasks,
    }

    if err := h.repo.Create(monitor); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 注册到 Sentinel
    if err := h.sentinelService.Update(monitor); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"id": monitor.ID})
}

// @Summary Get service history by host
// @Description Get 24-hour service probe history for a specific host
// @Tags service-monitor
// @Param id path uint64 true "Host ID"
// @Produce json
// @Success 200 {object} []ServiceHistoryInfo
// @Router /api/v1/hosts/{id}/service-history [get]
func (h *ServiceMonitorHandler) GetHostHistory(c *gin.Context) {
    hostID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

    histories, err := h.repo.GetHostHistory(hostID, 24*time.Hour)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, histories)
}
```

#### 任务清单
- [ ] 创建 ServiceMonitorHandler
- [ ] 实现 CRUD 接口
- [ ] 实现统计数据查询接口
- [ ] 添加路由注册
- [ ] 添加权限控制（RBAC）
- [ ] 添加 Swagger 注解
- [ ] 实现批量操作接口

---

### Phase 6: 前端展示层 (Week 6-8)

#### 页面结构

```
ui/src/pages/hosts/
├── service-monitor-list-page.tsx       # 监控任务列表
├── service-monitor-form-page.tsx       # 创建/编辑表单
├── service-monitor-detail-page.tsx     # 任务详情页
└── network-topology-page.tsx           # 节点间网络拓扑（可选）

ui/src/components/service-monitor/
├── monitor-type-selector.tsx           # 探测类型选择
├── target-input.tsx                    # 目标配置输入
├── host-selector.tsx                   # 探测节点选择器
├── availability-heatmap.tsx            # 30天可用性热力图 ⭐
├── latency-trend-chart.tsx             # 延迟趋势图
├── status-indicator.tsx                # 状态指示器
└── network-matrix.tsx                  # 节点间延迟矩阵
```

#### 30 天可用性热力图实现

```typescript
// ui/src/components/service-monitor/availability-heatmap.tsx

interface HeatmapData {
  date: string;
  dayIndex: number;
  status: 'good' | 'low' | 'down' | 'nodata';
  uptime: number;       // 百分比 (0-100)
  avgDelay: number;     // 毫秒
  up: number;           // 成功次数
  down: number;         // 失败次数
}

interface AvailabilityHeatmapProps {
  serviceId: string;
  data: {
    delay: number[];    // 30 个元素
    up: number[];       // 30 个元素
    down: number[];     // 30 个元素
  };
}

export function AvailabilityHeatmap({ serviceId, data }: AvailabilityHeatmapProps) {
  const heatmapData: HeatmapData[] = data.up.map((up, index) => {
    const down = data.down[index];
    const total = up + down;
    const uptime = total > 0 ? (up / total) * 100 : 0;

    const status = getStatus(uptime);
    const date = new Date();
    date.setDate(date.getDate() - (29 - index));

    return {
      date: date.toISOString().split('T')[0],
      dayIndex: index,
      status,
      uptime,
      avgDelay: data.delay[index],
      up,
      down,
    };
  });

  function getStatus(uptime: number): HeatmapData['status'] {
    if (uptime === 0) return 'nodata';
    if (uptime > 95) return 'good';
    if (uptime > 80) return 'low';
    return 'down';
  }

  function getColorByStatus(status: HeatmapData['status']) {
    switch (status) {
      case 'good': return '#10b981';      // 绿色
      case 'low': return '#f59e0b';       // 橙色
      case 'down': return '#ef4444';      // 红色
      default: return '#6b7280';          // 灰色
    }
  }

  return (
    <div className="grid grid-cols-30 gap-1">
      {heatmapData.map((item) => (
        <Tooltip key={item.dayIndex}>
          <TooltipTrigger>
            <div
              className="w-8 h-8 rounded cursor-pointer transition-all hover:scale-110"
              style={{ backgroundColor: getColorByStatus(item.status) }}
            />
          </TooltipTrigger>
          <TooltipContent>
            <div className="text-sm">
              <p className="font-medium">{item.date}</p>
              <p>可用率: {item.uptime.toFixed(2)}%</p>
              <p>平均延迟: {item.avgDelay.toFixed(2)}ms</p>
              <p>成功: {item.up} | 失败: {item.down}</p>
            </div>
          </TooltipContent>
        </Tooltip>
      ))}
    </div>
  );
}
```

#### 节点间网络拓扑

```typescript
// ui/src/components/service-monitor/network-matrix.tsx

interface NetworkMatrixProps {
  hosts: Host[];
  latencyData: {
    [sourceId: string]: {
      [targetId: string]: number; // 延迟（毫秒）
    };
  };
}

export function NetworkMatrix({ hosts, latencyData }: NetworkMatrixProps) {
  function getLatencyColor(latency: number | undefined) {
    if (!latency) return '#e5e7eb';  // 灰色（无数据）
    if (latency < 50) return '#10b981';   // 绿色（优秀）
    if (latency < 100) return '#84cc16';  // 黄绿（良好）
    if (latency < 200) return '#f59e0b';  // 橙色（一般）
    return '#ef4444';                     // 红色（较差）
  }

  return (
    <div className="overflow-auto">
      <table className="min-w-full">
        <thead>
          <tr>
            <th className="sticky left-0 bg-white">源节点 \ 目标节点</th>
            {hosts.map(host => (
              <th key={host.id} className="px-4 py-2">{host.name}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {hosts.map(sourceHost => (
            <tr key={sourceHost.id}>
              <th className="sticky left-0 bg-white px-4 py-2">{sourceHost.name}</th>
              {hosts.map(targetHost => {
                const latency = latencyData[sourceHost.id]?.[targetHost.id];
                const isSelf = sourceHost.id === targetHost.id;

                return (
                  <td
                    key={targetHost.id}
                    className="px-4 py-2 text-center"
                    style={{
                      backgroundColor: isSelf ? '#f3f4f6' : getLatencyColor(latency),
                    }}
                  >
                    {isSelf ? '-' : latency ? `${latency.toFixed(0)}ms` : 'N/A'}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

#### 任务清单
- [ ] 实现监控任务列表页
  - [ ] 表格展示（支持排序、筛选）
  - [ ] 状态指示器
  - [ ] 快速操作菜单
- [ ] 实现创建/编辑表单
  - [ ] 探测类型选择器
  - [ ] 目标配置输入（根据类型动态变化）
  - [ ] 节点选择器（支持"全选"/"排除"模式）
  - [ ] 告警配置面板
- [ ] 实现详情页
  - [ ] 30 天可用性热力图 ⭐
  - [ ] 延迟趋势折线图
  - [ ] 最近探测结果表格
  - [ ] 告警历史记录
- [ ] 实现节点间网络拓扑页
  - [ ] 延迟矩阵表格
  - [ ] 节点关系图（可选，使用 react-flow）
- [ ] 实现节点探测历史多线图 ⭐（参考 Nezha）
  - [ ] 后端 API：查询节点执行的所有探测任务历史
  - [ ] 前端组件：多条折线展示该节点到各目标的延迟趋势
  - [ ] 图例显示：探测任务名称 + 丢包率
  - [ ] 交互功能：悬停显示详细数据点
- [ ] 集成到主机管理子系统导航

---

## 📈 节点探测历史多线图（基于 Nezha 实现）

> 参考：nezha/cmd/dashboard/controller/service.go:listServiceHistory

### 功能说明

展示**单个节点执行的所有探测任务**的 24 小时延迟趋势，用于：
- 对比该节点到不同目标的网络质量
- 发现延迟波动和网络抖动
- 识别网络问题的时间模式

### API 实现

#### 后端接口

```go
// GET /api/v1/hosts/:id/probe-history
// 查询节点在过去24小时执行的所有探测任务历史

// internal/api/handlers/service_monitor_handler.go
func (h *ServiceMonitorHandler) GetHostProbeHistory(c *gin.Context) {
    hostID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

    // 权限检查
    host, err := h.hostRepo.GetByID(hostID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
        return
    }

    // 查询该节点执行的所有探测历史（过去24小时）
    var histories []*models.ServiceHistory
    err = h.db.Model(&models.ServiceHistory{}).
        Select("service_id, created_at, host_id, avg_delay").
        Where("host_id = ?", hostID).
        Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
        Order("service_id, created_at").
        Find(&histories).Error

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 按 ServiceID 分组聚合
    var sortedServiceIDs []uint64
    resultMap := make(map[uint64]*ServiceProbeHistory)

    for _, history := range histories {
        infos, ok := resultMap[history.ServiceID]

        if !ok {
            // 获取监控任务详情
            monitor, _ := h.sentinelService.Get(history.ServiceID)

            infos = &ServiceProbeHistory{
                MonitorID:   history.ServiceID,
                MonitorName: monitor.Name,        // 探测目标名称
                HostID:      history.HostID,
                HostName:    host.Name,           // 执行节点名称
                Timestamps:  []int64{},
                Delays:      []float32{},
            }
            resultMap[history.ServiceID] = infos
            sortedServiceIDs = append(sortedServiceIDs, history.ServiceID)
        }

        // 添加数据点
        infos.Timestamps = append(infos.Timestamps,
            history.CreatedAt.Truncate(time.Minute).Unix()*1000) // 毫秒时间戳
        infos.Delays = append(infos.Delays, history.AvgDelay)
    }

    // 计算丢包率
    for _, serviceID := range sortedServiceIDs {
        infos := resultMap[serviceID]

        // 查询该监控任务的 up/down 统计
        var stats struct {
            TotalUp   uint64
            TotalDown uint64
        }
        h.db.Model(&models.ServiceHistory{}).
            Select("SUM(up) as total_up, SUM(down) as total_down").
            Where("service_id = ? AND host_id = ?", serviceID, hostID).
            Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
            Scan(&stats)

        total := stats.TotalUp + stats.TotalDown
        if total > 0 {
            infos.LossRate = float32(stats.TotalDown) / float32(total)
        }
    }

    // 按顺序返回
    ret := make([]*ServiceProbeHistory, 0, len(sortedServiceIDs))
    for _, id := range sortedServiceIDs {
        ret = append(ret, resultMap[id])
    }

    c.JSON(http.StatusOK, gin.H{
        "host_id":   hostID,
        "host_name": host.Name,
        "time_range": gin.H{
            "from": time.Now().Add(-24 * time.Hour).Unix() * 1000,
            "to":   time.Now().Unix() * 1000,
        },
        "probe_histories": ret,
    })
}
```

#### 数据模型

```go
// internal/models/service_probe_history.go
type ServiceProbeHistory struct {
    MonitorID   uint64    `json:"monitor_id"`   // 监控任务ID
    MonitorName string    `json:"monitor_name"` // 探测目标名称（如"家里云"）
    HostID      uint64    `json:"host_id"`      // 执行节点ID
    HostName    string    `json:"host_name"`    // 执行节点名称
    LossRate    float32   `json:"loss_rate"`    // 丢包率 (0.0-1.0)
    Timestamps  []int64   `json:"timestamps"`   // 时间戳数组（毫秒）
    Delays      []float32 `json:"delays"`       // 延迟数组（毫秒）
}
```

### 前端实现

#### 组件代码

```typescript
// ui/src/components/service-monitor/host-probe-chart.tsx

import { useQuery } from '@tanstack/react-query';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { serviceMonitorApi } from '@/services/service-monitor';

interface ServiceProbeHistory {
  monitor_id: number;
  monitor_name: string;
  host_id: number;
  host_name: string;
  loss_rate: number;        // 0.0-1.0
  timestamps: number[];     // 毫秒时间戳
  delays: number[];         // 毫秒
}

interface HostProbeChartProps {
  hostId: string;
  height?: number;
}

const COLORS = [
  '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6',
  '#ec4899', '#14b8a6', '#f97316', '#06b6d4', '#84cc16',
];

export function HostProbeChart({ hostId, height = 400 }: HostProbeChartProps) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['host-probe-history', hostId],
    queryFn: () => serviceMonitorApi.getHostProbeHistory(hostId),
    refetchInterval: 60000, // 每分钟刷新
  });

  if (isLoading) {
    return <div className="flex items-center justify-center h-96">加载中...</div>;
  }

  if (error) {
    return <div className="text-red-500">加载失败: {error.message}</div>;
  }

  if (!data || data.probe_histories.length === 0) {
    return <div className="text-gray-500 text-center py-8">暂无探测数据</div>;
  }

  // 转换数据格式为 recharts 需要的格式
  // 构建时间戳到数据点的映射
  const allTimestamps = new Set<number>();
  data.probe_histories.forEach((history: ServiceProbeHistory) => {
    history.timestamps.forEach(ts => allTimestamps.add(ts));
  });

  const sortedTimestamps = Array.from(allTimestamps).sort((a, b) => a - b);

  // 构建数据点数组
  const chartData = sortedTimestamps.map(timestamp => {
    const point: any = { timestamp };

    data.probe_histories.forEach((history: ServiceProbeHistory) => {
      const index = history.timestamps.indexOf(timestamp);
      point[`monitor_${history.monitor_id}`] = index >= 0 ? history.delays[index] : null;
    });

    return point;
  });

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="mb-4">
        <h3 className="text-lg font-semibold">
          {data.host_name} - 探测延迟趋势
        </h3>
        <p className="text-sm text-gray-500">
          过去 24 小时的探测数据
        </p>
      </div>

      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" />

          <XAxis
            dataKey="timestamp"
            type="number"
            domain={['dataMin', 'dataMax']}
            tickFormatter={(ts) => {
              const date = new Date(ts);
              return `${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
            }}
            label={{ value: '时间', position: 'insideBottom', offset: -5 }}
          />

          <YAxis
            label={{ value: '延迟 (ms)', angle: -90, position: 'insideLeft' }}
          />

          <Tooltip
            labelFormatter={(ts) => {
              const date = new Date(ts as number);
              return date.toLocaleString('zh-CN');
            }}
            formatter={(value: any, name: string) => {
              if (value === null) return ['N/A', name];
              return [`${value.toFixed(2)}ms`, name];
            }}
          />

          <Legend
            formatter={(value, entry: any) => {
              const monitorId = parseInt(value.replace('monitor_', ''));
              const history = data.probe_histories.find(
                (h: ServiceProbeHistory) => h.monitor_id === monitorId
              );

              if (!history) return value;

              const lossRatePercent = (history.loss_rate * 100).toFixed(1);
              return `${history.monitor_name} ${lossRatePercent}%`;
            }}
          />

          {data.probe_histories.map((history: ServiceProbeHistory, index: number) => (
            <Line
              key={history.monitor_id}
              type="monotone"
              dataKey={`monitor_${history.monitor_id}`}
              name={`monitor_${history.monitor_id}`}
              stroke={COLORS[index % COLORS.length]}
              strokeWidth={2}
              dot={{ r: 3 }}
              activeDot={{ r: 5 }}
              connectNulls={false}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>

      {/* 图例详情 */}
      <div className="mt-4 grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
        {data.probe_histories.map((history: ServiceProbeHistory, index: number) => (
          <div
            key={history.monitor_id}
            className="flex items-center space-x-2 p-2 rounded border"
          >
            <div
              className="w-4 h-4 rounded"
              style={{ backgroundColor: COLORS[index % COLORS.length] }}
            />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">{history.monitor_name}</p>
              <p className="text-xs text-gray-500">
                丢包率: {(history.loss_rate * 100).toFixed(1)}%
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
```

#### API 服务

```typescript
// ui/src/services/service-monitor.ts

export const serviceMonitorApi = {
  // 获取节点探测历史（24小时）
  getHostProbeHistory: async (hostId: string) => {
    const response = await apiClient.get(`/api/v1/hosts/${hostId}/probe-history`);
    return response.data;
  },
};
```

#### 页面集成

```typescript
// ui/src/pages/hosts/host-detail-page.tsx

import { HostProbeChart } from '@/components/service-monitor/host-probe-chart';

export function HostDetailPage() {
  const { hostId } = useParams();

  return (
    <div className="space-y-6">
      {/* 其他内容 */}

      {/* 探测历史图表 */}
      <section>
        <HostProbeChart hostId={hostId} />
      </section>
    </div>
  );
}
```

### 数据流程说明

```
1. 用户访问节点详情页（如"腾讯云-上海"）
   ↓
2. 前端调用 GET /api/v1/hosts/{id}/probe-history
   ↓
3. 后端查询该节点执行的所有探测任务历史
   WHERE host_id = {id} AND created_at >= NOW() - 24 HOUR
   ↓
4. 按 ServiceID 分组聚合数据
   - 监控任务1（家里云）: [timestamps[], delays[]]
   - 监控任务2（香港KC）: [timestamps[], delays[]]
   - 监控任务3（成都电信）: [timestamps[], delays[]]
   ↓
5. 计算每个监控任务的丢包率
   loss_rate = total_down / (total_up + total_down)
   ↓
6. 返回 JSON 数据
   {
     "host_name": "腾讯云-上海",
     "probe_histories": [
       {
         "monitor_name": "家里云",
         "loss_rate": 0.0,
         "timestamps": [timestamp1, timestamp2, ...],
         "delays": [68, 70, 69, ...]
       },
       ...
     ]
   }
   ↓
7. 前端使用 recharts 渲染多线图
   - X轴: 时间（HH:mm）
   - Y轴: 延迟（ms）
   - 每条线: 一个探测目标
   - 图例: 目标名称 + 丢包率%
```

### 与 Nezha 的对应关系

| Nezha 术语 | Tiga 术语 | 说明 |
|-----------|----------|------|
| Server | Host | 执行探测的节点 |
| Service | ServiceMonitor | 监控任务（定义探测目标） |
| ServiceHistory | ServiceHistory | 探测历史记录 |
| `/service/:id` | `/hosts/:id/probe-history` | API 端点 |
| `server_id` | `host_id` | 执行节点ID |
| `service_id` | `monitor_id` | 监控任务ID |

### 示例场景

假设有以下监控配置：
- **节点A**（腾讯云-上海）
- **监控任务1**：探测"家里云"（ICMP），由节点A执行
- **监控任务2**：探测"香港KC"（ICMP），由节点A执行
- **监控任务3**：探测"成都电信"（ICMP），由节点A执行

访问"节点A"的详情页时，图表显示：
- **标题**："腾讯云-上海 - 探测延迟趋势"
- **3条折线**：
  - 蓝色：家里云 0.0%
  - 绿色：香港KC 0.3%
  - 橙色：成都电信 0.1%
- **X轴**：16:00 → 04:00 → 12:00
- **Y轴**：0-400ms

---

## 🎯 节点间网络监控实现方案

### 原理说明

通过为每对节点创建探测任务，构建完整的 N×(N-1) 延迟矩阵：

```
假设有节点 A(ID=1), B(ID=2), C(ID=3)

# 创建 6 个监控任务（双向探测）

任务1: Target=A的IP, Reporters=[B,C], Type=ICMP
  → ServiceHistory: (任务1, HostID=2, delay=10ms)  # B→A
  → ServiceHistory: (任务1, HostID=3, delay=20ms)  # C→A

任务2: Target=B的IP, Reporters=[A,C], Type=ICMP
  → ServiceHistory: (任务2, HostID=1, delay=12ms)  # A→B
  → ServiceHistory: (任务2, HostID=3, delay=18ms)  # C→B

任务3: Target=C的IP, Reporters=[A,B], Type=ICMP
  → ServiceHistory: (任务3, HostID=1, delay=22ms)  # A→C
  → ServiceHistory: (任务3, HostID=2, delay=16ms)  # B→C

结果矩阵:
     A    B    C
A    -   12ms 22ms
B   10ms  -   16ms
C   20ms 18ms  -
```

### 数据查询

```sql
-- 查询节点 A(ID=1) 到所有其他节点的延迟
SELECT
    sm.target,
    h.name as target_host,
    sh.avg_delay
FROM service_history sh
JOIN service_monitors sm ON sh.service_id = sm.id
JOIN host_nodes h ON sm.target = h.ip
WHERE sh.host_id = 1
  AND sh.created_at >= NOW() - INTERVAL 1 HOUR
ORDER BY sh.created_at DESC;

-- 查询所有节点间的平均延迟（最近1小时）
SELECT
    sh.host_id as source_id,
    h1.name as source_name,
    h2.id as target_id,
    h2.name as target_name,
    AVG(sh.avg_delay) as avg_latency,
    COUNT(*) as sample_count
FROM service_history sh
JOIN service_monitors sm ON sh.service_id = sm.id
JOIN host_nodes h1 ON sh.host_id = h1.id
JOIN host_nodes h2 ON sm.target = h2.ip
WHERE sh.created_at >= NOW() - INTERVAL 1 HOUR
  AND sh.host_id > 0
GROUP BY sh.host_id, h1.name, h2.id, h2.name
ORDER BY source_id, target_id;
```

### UI 展示形式

1. **表格矩阵** (已实现)
   - 行=源节点，列=目标节点
   - 单元格颜色表示延迟等级
   - 对角线显示 "-"（自己到自己）

2. **节点拓扑图** (可选)
   - 使用 `react-flow` 或 `d3.js`
   - 节点=主机
   - 边的粗细/颜色表示延迟
   - 支持拖拽和缩放

3. **延迟热力图** (可选)
   - 横轴=时间
   - 纵轴=节点对
   - 颜色=延迟值

---

## 🔧 技术栈与依赖

### 新增依赖

```bash
# Server 端
go get github.com/robfig/cron/v3              # 任务调度
go get github.com/prometheus-community/pro-bing  # ICMP Ping (Agent端)

# 前端
npm install recharts                           # 图表库（已有）
npm install react-flow-renderer                # 节点拓扑图（可选）
```

### 现有依赖（复用）
- gRPC 通信框架
- GORM 数据库
- 通知系统（`internal/services/notification`）
- 告警规则引擎（`antonmedv/expr`）

---

## 📊 数据库索引优化

```sql
-- 最重要的复合索引
CREATE INDEX idx_service_host_time
ON service_history(service_id, host_id, created_at DESC, avg_delay);

-- 覆盖查询索引
CREATE INDEX idx_created_service
ON service_history(created_at DESC, service_id)
WHERE host_id = 0;  -- 汇总数据查询

-- 节点查询索引
CREATE INDEX idx_host_created
ON service_history(host_id, created_at DESC)
WHERE host_id > 0;
```

---

## ⚙️ 配置项

```yaml
# config.yaml
service_monitor:
  enabled: true
  max_monitors: 200                 # 最大监控任务数
  avg_ping_count: 3                 # Ping 聚合次数（减少写入）
  history_retention_days: 90        # 历史数据保留天数
  worker_buffer_size: 200           # 上报通道缓冲大小
  current_status_window: 30         # 滑动窗口大小（样本数）
  daily_refresh_cron: "0 0 0 * * *" # 每日数据刷新时间
```

---

## 🧪 测试计划

### 单元测试
- [ ] ServiceMonitor 模型的 CRUD
- [ ] Ping 数据聚合逻辑
- [ ] 状态码计算（Good/Low/Down）
- [ ] 30 天数据左移逻辑
- [ ] TLS 证书解析和过期检测

### 集成测试
- [ ] Agent 执行探测并上报
- [ ] Server 接收并存储结果
- [ ] 定时任务调度准确性
- [ ] 告警触发逻辑
- [ ] 节点间延迟矩阵构建

### 性能测试
- [ ] 100 个监控任务的调度性能
- [ ] 1000 条/秒上报数据的处理能力
- [ ] 30 天历史数据查询性能（< 500ms）
- [ ] 并发探测执行压力测试

---

## 📅 里程碑与时间线

- **Week 2**: 完成数据模型和 ServiceSentinel 框架
- **Week 4**: 完成 Server 调度器和 Agent 探测执行器
- **Week 6**: 完成告警系统和 API 接口
- **Week 8**: 完成前端展示（含 30 天热力图）和测试

---

## 📚 参考资料

- **Nezha 源码分析**:
  - ServiceSentinel: `nezha/service/singleton/servicesentinel.go`
  - Agent 探测: `nezha-agent/cmd/agent/main.go:677-769`
  - 数据模型: `nezha/model/service.go`
- **现有规格文档**: `.claude/specs/002-nezha-webssh/`
- **探测库文档**:
  - [prometheus-community/pro-bing](https://github.com/prometheus-community/pro-bing)
  - [robfig/cron](https://github.com/robfig/cron)

---

*服务监控增强计划添加于 2025-10-08*
