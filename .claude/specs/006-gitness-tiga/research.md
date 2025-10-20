# 研究文档：定时任务和审计系统重构

**分支**：`006-gitness-tiga` | **日期**：2025-10-19
**目的**：技术研究和设计决策，支持 Scheduler 和 Audit 系统重构

---

## 1. Gitness Job Scheduler 设计研究

### 核心架构

**参考代码**：
- `job/scheduler.go`: Scheduler 核心逻辑，负责调度任务
- `job/executor.go`: Executor 执行器，负责执行任务
- `job/types.go`: Job 数据模型定义
- `app/services/cleanup/service.go`: Cleanup 服务示例，展示如何使用 Scheduler + Executor

**关键设计模式**：

#### 1.1 Scheduler + Executor 分离架构

Gitness 采用经典的调度器-执行器分离模式：

**Scheduler 职责**：
- 从数据库加载待执行任务
- 根据 cron 表达式计算下次执行时间
- 使用分布式锁确保任务不重复执行
- 管理任务状态转换（Scheduled → Running → Finished/Failed）
- 处理任务取消请求（通过 PubSub）

**Executor 职责**：
- 注册任务处理器（Handler）
- 执行具体任务逻辑
- 处理 panic 恢复
- 报告任务进度
- 记录任务结果

```go
// Scheduler 接口（简化）
type Scheduler struct {
    store         Store              // 任务存储
    executor      *Executor          // 执行器
    mxManager     lock.MutexManager  // 分布式锁管理器
    pubsubService pubsub.PubSub      // 消息发布订阅

    instanceID    string             // 实例 ID
    maxRunning    int                // 最大并发任务数
    retentionTime time.Duration      // 任务历史保留时间

    cancelJobMap  map[string]context.CancelFunc  // 任务取消 map
}

// Executor 接口（简化）
type Executor struct {
    handlerMap map[string]Handler  // 任务类型 -> 处理器
    store      Store
    publisher  pubsub.Publisher
}

// Handler 接口
type Handler interface {
    Handle(ctx context.Context, input string, fn ProgressReporter) (result string, err error)
}
```

**决策**：Tiga 完全采用此架构，提供清晰的职责分离和可测试性。

#### 1.2 Job 数据模型

Gitness 的 Job 模型非常完善：

```go
type Job struct {
    UID                 string    // 唯一标识
    Type                string    // 任务类型
    Priority            Priority  // 优先级
    Data                string    // 任务输入数据
    Result              string    // 任务执行结果
    State               State     // 状态：Scheduled/Running/Finished/Failed/Cancelled

    // 调度相关
    Scheduled           int64     // 下次执行时间（Unix 毫秒）
    IsRecurring         bool      // 是否循环任务
    RecurringCron       string    // cron 表达式

    // 执行相关
    MaxDurationSeconds  int       // 最大执行时间
    MaxRetries          int       // 最大重试次数
    TotalExecutions     int       // 总执行次数
    RunBy               string    // 执行实例 ID
    RunDeadline         int64     // 执行截止时间
    RunProgress         int       // 执行进度（0-100）
    LastExecuted        int64     // 最后执行时间

    // 故障处理
    ConsecutiveFailures int       // 连续失败次数
    LastFailureError    string    // 最后失败错误

    // 其他
    GroupID             string    // 任务分组 ID
    Created             int64     // 创建时间
    Updated             int64     // 更新时间
}
```

**决策**：Tiga 参考此模型，简化为适合定时任务的字段。

#### 1.3 任务注册和调度模式

**Cleanup Service 示例**：

```go
type Service struct {
    scheduler *job.Scheduler
    executor  *job.Executor
    // ... 其他依赖
}

// 初始化时注册任务
func (s *Service) Register(ctx context.Context) error {
    // 1. 注册任务处理器
    if err := s.registerJobHandlers(); err != nil {
        return err
    }

    // 2. 调度循环任务
    if err := s.scheduleRecurringCleanupJobs(ctx); err != nil {
        return err
    }

    return nil
}

// 注册处理器
func (s *Service) registerJobHandlers() error {
    // 注册 webhook 清理任务
    err := s.executor.Register(
        "gitness:cleanup:webhook-executions",
        newWebhookExecutionsCleanupJob(...),
    )
    if err != nil {
        return err
    }

    // 注册更多任务...
    return nil
}

// 调度循环任务
func (s *Service) scheduleRecurringCleanupJobs(ctx context.Context) error {
    // 添加循环任务
    err := s.scheduler.AddRecurring(
        ctx,
        "gitness:cleanup:webhook-executions",  // 任务 ID
        "gitness:cleanup:webhook-executions",  // 任务类型
        "21 */4 * * *",                        // cron 表达式
        1 * time.Minute,                       // 最大执行时间
    )
    return err
}
```

**任务实现示例**：

```go
type webhookExecutionsCleanupJob struct {
    retentionTime         time.Duration
    webhookExecutionStore store.WebhookExecutionStore
}

func (j *webhookExecutionsCleanupJob) Handle(
    ctx context.Context,
    _ string,  // 输入数据（本例未使用）
    progress job.ProgressReporter,
) (string, error) {
    // 计算截止时间
    olderThan := time.Now().Add(-j.retentionTime)

    // 执行清理
    n, err := j.webhookExecutionStore.DeleteOld(ctx, olderThan)
    if err != nil {
        return "", fmt.Errorf("failed to delete: %w", err)
    }

    // 返回结果
    result := fmt.Sprintf("deleted %d records", n)
    return result, nil
}
```

**决策**：Tiga 采用相同的注册和调度模式。

#### 1.4 分布式锁集成

Gitness Scheduler 使用分布式锁确保任务不重复执行：

```go
func (s *Scheduler) processReadyJobs(ctx context.Context, now time.Time) {
    // 获取全局锁
    mx, err := s.mxManager.NewMutex("scheduler_global")
    if err != nil {
        return
    }

    err = mx.Lock(ctx)
    if err != nil {
        return
    }
    defer mx.Unlock(ctx)

    // 在锁保护下查询和分配任务
    jobs, err := s.store.FindExecutable(ctx, now, s.maxRunning)
    // 分配任务给当前实例...
}
```

**决策**：Tiga 复用分布式锁机制。

#### 1.5 任务状态转换

```
创建: Scheduled (初始状态)
  ↓
调度: Scheduled → Running (设置 RunBy、RunDeadline)
  ↓
执行: Running (可报告进度 0-100)
  ↓
完成: Running → Finished (成功)
  或: Running → Failed (失败)
  或: Running → Cancelled (取消)
  ↓
循环: Finished/Failed → Scheduled (计算下次执行时间)
```

**决策**：Tiga 采用相同的状态机。

### Tiga 适配方案

**保留**：
- Scheduler + Executor 架构
- Job 数据模型（简化）
- 任务注册模式
- cron 表达式调度
- 分布式锁保护
- 任务状态转换

**简化**：
- 移除 PubSub 取消机制（可选功能）
- 移除任务优先级（定时任务无需）
- 移除任务分组（当前未使用）

**增强**：
- 添加任务启用/禁用 API
- 添加手动触发 API
- 添加任务执行统计查询
- 添加任务执行历史分页查询

---

## 2. Gitness Audit 设计研究

### 强类型系统

**文件分析**：
- `audit/audit.go`: 核心类型定义和验证
- `audit/interface.go`: 服务接口
- `audit/middleware.go`: HTTP 中间件和 IP 提取

#### 2.1 Action 和 ResourceType 枚举

```go
type Action string
const (
    ActionCreated   Action = "created"
    ActionUpdated   Action = "updated"
    ActionDeleted   Action = "deleted"
    ActionBypassed  Action = "bypassed"
    ActionForcePush Action = "forcePush"
)

func (a Action) Validate() error {
    switch a {
    case ActionCreated, ActionUpdated, ActionDeleted, ActionBypassed, ActionForcePush:
        return nil
    default:
        return ErrActionUndefined
    }
}
```

**决策**：Tiga 定义类似的强类型枚举，防止拼写错误。额外的 Action：
- `ActionRead`（读取操作）
- `ActionLogin`/`ActionLogout`（认证操作）
- `ActionEnabled`/`ActionDisabled`（状态变更）

#### 2.2 Resource 结构

```go
type Resource struct {
    Type       ResourceType
    Identifier string
    Data       map[string]string
}

func NewResource(rtype ResourceType, identifier string, keyValues ...string) Resource {
    r := Resource{
        Type:       rtype,
        Identifier: identifier,
        Data:       make(map[string]string, len(keyValues)),
    }
    for i := 0; i < len(keyValues); i += 2 {
        k, v := keyValues[i], keyValues[i+1]
        r.Data[k] = v
    }
    return r
}
```

**决策**：采用相同的 `Resource` 设计，支持灵活的资源元数据。

#### 2.3 Event 结构和验证

```go
type Event struct {
    ID            string
    Timestamp     int64
    Action        Action
    User          types.Principal
    SpacePath     string
    Resource      Resource
    DiffObject    DiffObject
    ClientIP      string
    RequestMethod string
    Data          map[string]string
}

func (e *Event) Validate() error {
    if err := e.Action.Validate(); err != nil {
        return fmt.Errorf("invalid action: %w", err)
    }
    if e.User.UID == "" {
        return ErrUserIsRequired
    }
    // ...
    return nil
}
```

**决策**：完全采用 Event 结构和验证机制，确保审计日志完整性。

### Functional Options 模式

**Gitness 实现**：

```go
type FuncOption func(e *Event)

func (f FuncOption) Apply(event *Event) {
    f(event)
}

type Option interface {
    Apply(e *Event)
}

func WithID(value string) FuncOption {
    return func(e *Event) {
        e.ID = value
    }
}

func WithNewObject(value any) FuncOption {
    return func(e *Event) {
        e.DiffObject.NewObject = value
    }
}
```

**决策**：采用 Functional Options 模式，提高 API 灵活性和可扩展性。

### 客户端 IP 提取

**Gitness 实现**：

```go
func RealIP(r *http.Request) string {
    var ip string

    if tcip := r.Header.Get(trueClientIP); tcip != "" {
        ip = tcip
    } else if xrip := r.Header.Get(xRealIP); xrip != "" {
        ip = xrip
    } else if xff := r.Header.Get(xForwardedFor); xff != "" {
        i := strings.Index(xff, ",")
        if i == -1 {
            i = len(xff)
        }
        ip = xff[:i]
    } else {
        ip = strings.Split(r.RemoteAddr, ":")[0]
    }

    if ip == "" || net.ParseIP(ip) == nil {
        return ""
    }
    return ip
}
```

**决策**：完全复用此逻辑，支持所有常见代理 header。

---

## 3. 分布式锁实现研究

### 3.1 数据库行锁（默认实现）

#### PostgreSQL 实现

```sql
-- 获取锁
SELECT pg_advisory_lock(hash_function('lock_key'));

-- 释放锁
SELECT pg_advisory_unlock(hash_function('lock_key'));

-- 或使用事务级锁
BEGIN;
SELECT * FROM task_locks WHERE lock_key = 'key' FOR UPDATE;
-- 执行任务
COMMIT;
```

**优点**：
- 无额外依赖
- 事务级一致性
- 自动死锁检测

**缺点**：
- 性能较 Redis 低
- 需要保持数据库连接

**决策**：作为默认实现，适合小规模部署。

#### MySQL 实现

```sql
-- 获取锁（超时10秒）
SELECT GET_LOCK('lock_key', 10);

-- 释放锁
SELECT RELEASE_LOCK('lock_key');

-- 检查锁状态
SELECT IS_FREE_LOCK('lock_key');
```

**优点**：
- 简单易用
- 独立于事务

**缺点**：
- 锁超时后不自动释放
- 连接断开时锁自动释放（可能导致双执行）

**决策**：支持 MySQL，但推荐 PostgreSQL。

### 3.2 Redis 锁实现

#### Redlock 算法

```go
// 伪代码
func Lock(key string, ttl time.Duration) (bool, error) {
    value := uuid.New()
    ok, err := redis.SetNX(key, value, ttl)
    if !ok {
        return false, nil
    }
    return true, nil
}

func Unlock(key string, value string) error {
    // Lua 脚本保证原子性
    script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `
    return redis.Eval(script, []string{key}, value)
}
```

**优点**：
- 高性能（<5ms）
- 自动过期
- 支持分布式部署

**缺点**：
- 需要 Redis 依赖
- 时钟漂移问题

**决策**：作为高性能选项，适合大规模部署。

### 3.3 etcd 锁实现

#### Lease 机制

```go
// 伪代码
func Lock(key string, ttl time.Duration) (leaseID, error) {
    lease, err := etcd.Grant(ttl)
    txn := etcd.Txn().
        If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
        Then(clientv3.OpPut(key, value, clientv3.WithLease(lease.ID))).
        Else()

    resp, err := txn.Commit()
    if !resp.Succeeded {
        return 0, ErrLockFailed
    }
    return lease.ID, nil
}
```

**优点**：
- 强一致性
- 自动续租
- 高可用

**缺点**：
- 部署复杂
- 延迟较高（10-50ms）

**决策**：作为可选实现，适合高可用需求。

### 锁接口设计

```go
package lock

type Mutex interface {
    Lock(ctx context.Context) error
    Unlock(ctx context.Context) error
}

type Manager interface {
    NewMutex(key string, opts ...Option) (Mutex, error)
}

type Option func(*Options)

type Options struct {
    TTL      time.Duration
    RetryMax int
    RetryInterval time.Duration
}

func WithTTL(ttl time.Duration) Option {
    return func(o *Options) {
        o.TTL = ttl
    }
}
```

**决策**：统一接口，运行时配置切换实现。

---

## 4. 审计日志对象截断策略研究

### 4.1 JSON 序列化大小估算

```go
// 示例对象
type DatabaseInstance struct {
    ID          string
    Name        string
    Description string  // 可能很长
    Config      string  // JSON 配置，可能超大
    Metadata    map[string]string
}

// 序列化大小
data, _ := json.Marshal(obj)
size := len(data) // 字节数
```

**问题**：
- 复杂对象序列化后可能超过 64KB
- 简单截断会破坏 JSON 结构

### 4.2 智能截断算法

**策略**：
1. 序列化对象到 JSON
2. 如果大小 ≤64KB，直接返回
3. 如果大小 >64KB：
   - 解析 JSON 到 map
   - 按字段值大小排序
   - 从最大字段开始截断
   - 截断后的字段添加 `...[truncated]` 后缀
   - 重新序列化并检查大小
   - 重复直到 ≤64KB

**伪代码**：

```go
func TruncateObject(obj interface{}, maxSize int) ([]byte, []string, error) {
    data, err := json.Marshal(obj)
    if len(data) <= maxSize {
        return data, nil, nil
    }

    var m map[string]interface{}
    json.Unmarshal(data, &m)

    truncatedFields := []string{}

    for {
        // 找到最大字段
        largest := findLargestField(m)

        // 截断字段值
        m[largest] = truncateString(m[largest].(string), maxFieldSize)
        truncatedFields = append(truncatedFields, largest)

        // 重新序列化
        data, _ = json.Marshal(m)
        if len(data) <= maxSize {
            break
        }
    }

    return data, truncatedFields, nil
}
```

**决策**：实现智能截断算法，保证 JSON 结构完整性。

### 4.3 截断标识存储

**方案 1**：在 AuditEvent 添加字段

```go
type AuditEvent struct {
    // ...
    OldObjectTruncated bool     `json:"old_object_truncated"`
    NewObjectTruncated bool     `json:"new_object_truncated"`
    TruncatedFields    []string `json:"truncated_fields,omitempty"`
}
```

**方案 2**：在对象 JSON 中添加元数据

```json
{
    "id": "123",
    "name": "instance",
    "config": "...[truncated]",
    "_metadata": {
        "truncated": true,
        "truncated_fields": ["config"]
    }
}
```

**决策**：采用方案 1，避免污染对象数据。

---

## 5. 任务执行超时机制研究

### 5.1 Context 取消传播

```go
func executeTask(ctx context.Context, task Task) error {
    // 创建带超时的 context
    taskCtx, cancel := context.WithTimeout(ctx, task.Timeout)
    defer cancel()

    // 传递给任务
    return task.Run(taskCtx)
}
```

**优点**：
- 优雅终止
- 任务可以清理资源
- 标准 Go 模式

**缺点**：
- 依赖任务实现配合
- 无法强制终止

**决策**：作为第一阶段终止机制。

### 5.2 Goroutine 强制终止

```go
func executeTaskWithGracePeriod(ctx context.Context, task Task, gracePeriod time.Duration) error {
    // 创建带超时的 context
    taskCtx, cancel := context.WithTimeout(ctx, task.Timeout)
    defer cancel()

    // 在 goroutine 中执行
    done := make(chan error, 1)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                done <- fmt.Errorf("task panicked: %v", r)
            }
        }()
        done <- task.Run(taskCtx)
    }()

    // 等待完成或超时
    select {
    case err := <-done:
        return err
    case <-taskCtx.Done():
        // 超时，给宽限期
        select {
        case err := <-done:
            return err
        case <-time.After(gracePeriod):
            // 宽限期后仍未结束，标记为超时失败
            // 注意：goroutine 可能泄漏，但 Go 无法强制杀死
            return fmt.Errorf("task timeout after grace period")
        }
    }
}
```

**决策**：实现宽限期机制，记录未能优雅终止的任务。

### 5.3 资源清理最佳实践

**任务实现指南**：
1. 监听 `ctx.Done()` channel
2. 收到取消信号时立即清理资源
3. 使用 `defer` 确保清理代码执行
4. 避免长时间阻塞操作

```go
func (t *ExampleTask) Run(ctx context.Context) error {
    // 使用 defer 清理
    file, err := os.Open("data.txt")
    if err != nil {
        return err
    }
    defer file.Close()

    // 定期检查 ctx
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // 执行工作
            processChunk()
        }
    }
}
```

**决策**：提供任务实现最佳实践文档。

---

## 6. 性能优化研究

### 6.1 批量审计日志写入

**策略**：
- 使用 buffered channel 收集审计事件
- 定期刷新（每1秒或100条）
- GORM 批量插入

```go
type BatchAuditWriter struct {
    events chan *AuditEvent
    db     *gorm.DB
    ticker *time.Ticker
}

func (w *BatchAuditWriter) Start() {
    batch := make([]*AuditEvent, 0, 100)

    for {
        select {
        case event := <-w.events:
            batch = append(batch, event)
            if len(batch) >= 100 {
                w.flush(batch)
                batch = batch[:0]
            }
        case <-w.ticker.C:
            if len(batch) > 0 {
                w.flush(batch)
                batch = batch[:0]
            }
        }
    }
}

func (w *BatchAuditWriter) flush(batch []*AuditEvent) {
    w.db.CreateInBatches(batch, 100)
}
```

**性能提升**：单条写入 100 req/s → 批量写入 1000+ req/s

**决策**：实现批量写入，提高吞吐量。

### 6.2 查询索引设计

**任务执行历史索引**：

```sql
CREATE INDEX idx_task_executions_task_name ON task_executions(task_name);
CREATE INDEX idx_task_executions_status ON task_executions(status);
CREATE INDEX idx_task_executions_created_at ON task_executions(created_at DESC);
CREATE INDEX idx_task_executions_composite ON task_executions(task_name, status, created_at DESC);
```

**审计日志索引**：

```sql
CREATE INDEX idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX idx_audit_events_resource_type ON audit_events(resource_type);
CREATE INDEX idx_audit_events_action ON audit_events(action);
CREATE INDEX idx_audit_events_created_at ON audit_events(created_at DESC);
CREATE INDEX idx_audit_events_composite ON audit_events(resource_type, action, created_at DESC);
```

**决策**：创建复合索引，优化常见查询场景。

### 6.3 连接池配置

**数据库连接池**：

```go
db.DB().SetMaxOpenConns(50)  // 最大打开连接数
db.DB().SetMaxIdleConns(10)  // 最大空闲连接数
db.DB().SetConnMaxLifetime(time.Hour) // 连接最大生命周期
```

**Redis 连接池**：

```go
redis.NewClient(&redis.Options{
    PoolSize:     50,
    MinIdleConns: 10,
    PoolTimeout:  4 * time.Second,
})
```

**决策**：根据负载动态调整连接池大小。

### 6.4 异步写入队列设计

**架构**：

```
审计中间件 → Buffered Channel (1000) → Batch Writer → Database
```

**优点**：
- 不阻塞业务请求
- 自动批量写入
- 队列满时丢弃（符合规格要求）

**决策**：实现异步队列，channel 容量 1000，满时丢弃并告警。

---

## 技术决策总结

| 领域 | 决策 | 理由 |
|-----|------|------|
| **Scheduler 架构** | 采用 Gitness 的组合模式（queue + canceler） | 清晰的职责分离，易于测试 |
| **分布式锁** | 支持 Database/Redis/etcd，默认 Database | 灵活性 + 零依赖默认 |
| **Audit 类型系统** | 强类型枚举 + Validate 方法 | 编译时检查，防止错误 |
| **Functional Options** | 完全采用 Gitness 模式 | API 灵活性和扩展性 |
| **对象截断** | 智能截断算法（保留结构） | 确保 JSON 可解析 |
| **任务超时** | Context 取消 + 30秒宽限期 | 优雅终止 + 防止任务挂起 |
| **批量写入** | 1秒或100条触发刷新 | 平衡延迟和吞吐量 |
| **查询索引** | 复合索引（type + action + time） | 优化常见查询场景 |
| **异步队列** | 1000容量 buffered channel | 高吞吐 + 满时丢弃策略 |

---

## 拒绝的替代方案

| 方案 | 为什么拒绝 |
|-----|----------|
| **使用 cron 表达式调度** | 当前任务均为固定间隔，不需要复杂的 cron 语法 |
| **任务结果缓存** | 任务执行频率低，缓存收益小 |
| **审计日志压缩存储** | 增加复杂度，64KB 限制已足够 |
| **实时审计日志流** | 当前需求只需查询，不需要实时推送 |
| **分布式任务队列（如 RabbitMQ）** | 过度工程化，当前场景不需要 |

---

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|-----|------|---------|
| **分布式锁死锁** | 任务无法执行 | TTL 自动过期 + 死锁检测日志 |
| **批量写入丢失** | 审计日志丢失 | 队列满时告警 + 监控队列深度 |
| **对象截断丢失关键信息** | 审计不完整 | 记录截断字段列表 + 管理员可查询原始对象 |
| **任务 goroutine 泄漏** | 内存泄漏 | 监控 goroutine 数量 + 定期重启 |
| **数据库锁性能瓶颈** | 高负载下延迟增加 | 提供 Redis 锁选项 + 性能测试 |

---

## 下一步行动

1. ✅ 研究完成
2. 进入阶段 1：设计与契约
   - 生成 data-model.md
   - 生成 API 契约（OpenAPI）
   - 生成契约测试
   - 生成 quickstart.md
3. 验证章程合规性
4. 进入阶段 2：任务规划

---

**研究状态**：✅ 完成
**审核者**：AI Agent
**批准日期**：2025-10-19
