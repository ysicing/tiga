# 数据模型：定时任务和审计系统重构

**功能分支**：`006-gitness-tiga` | **日期**：2025-10-19
**输入**：来自 `spec.md` 的功能需求和 `research.md` 的技术研究

---

## 概述

本文档定义 Scheduler 和 Audit 系统重构的完整数据模型。模型设计基于 Gitness 的最佳实践，采用强类型系统、验证机制和状态转换控制。

**设计原则**：
- 强类型约束（枚举、验证方法）
- 不可变审计日志（只能创建和查询）
- 支持分布式环境（分布式锁）
- 完整的执行历史追踪
- 智能数据截断（64KB 限制）

---

## Scheduler 数据模型

### 实体 1: ScheduledTask（定时任务配置）

**用途**：存储定时任务的配置信息，由 Scheduler 加载并调度执行。

**字段定义**：

```go
type ScheduledTask struct {
    // 基础字段
    UID         string    `gorm:"type:varchar(255);primaryKey" json:"uid"`
    Name        string    `gorm:"type:varchar(255);not null;unique" json:"name"`
    Type        string    `gorm:"type:varchar(255);not null" json:"type"`  // 任务类型（对应 Handler）
    Description string    `gorm:"type:text" json:"description"`

    // 调度配置
    IsRecurring bool      `gorm:"not null;default:false" json:"is_recurring"`
    CronExpr    string    `gorm:"type:varchar(255)" json:"cron_expr"`  // 如 "21 */4 * * *"
    Interval    int64     `gorm:"" json:"interval"`  // 间隔时间（秒），非 cron 时使用
    NextRun     time.Time `gorm:"index" json:"next_run"`  // 下次执行时间

    // 执行控制
    Enabled             bool   `gorm:"not null;default:true" json:"enabled"`
    MaxDurationSeconds  int    `gorm:"not null;default:3600" json:"max_duration_seconds"`  // 最大执行时间
    MaxRetries          int    `gorm:"not null;default:3" json:"max_retries"`
    TimeoutGracePeriod  int    `gorm:"not null;default:30" json:"timeout_grace_period"`  // 超时宽限期（秒）

    // 并发控制
    MaxConcurrent int    `gorm:"not null;default:1" json:"max_concurrent"`  // 最大并发执行数
    Priority      int    `gorm:"not null;default:0;index" json:"priority"`   // 优先级（越大越高）

    // 资源标签（用于 Filter 匹配）
    Labels JSONB `gorm:"type:text" json:"labels,omitempty"`  // map[string]string

    // 输入数据
    Data string `gorm:"type:text" json:"data"`  // 任务输入数据（JSON 字符串）

    // 统计信息
    TotalExecutions     int       `gorm:"not null;default:0" json:"total_executions"`
    SuccessExecutions   int       `gorm:"not null;default:0" json:"success_executions"`
    FailureExecutions   int       `gorm:"not null;default:0" json:"failure_executions"`
    ConsecutiveFailures int       `gorm:"not null;default:0" json:"consecutive_failures"`
    LastExecutedAt      time.Time `gorm:"" json:"last_executed_at,omitempty"`
    LastFailureError    string    `gorm:"type:text" json:"last_failure_error,omitempty"`

    // 时间戳
    CreatedAt time.Time `gorm:"index" json:"created_at"`
    UpdatedAt time.Time `gorm:"index" json:"updated_at"`
}
```

**索引设计**：
```sql
CREATE INDEX idx_scheduled_tasks_enabled ON scheduled_tasks(enabled);
CREATE INDEX idx_scheduled_tasks_next_run ON scheduled_tasks(next_run);
CREATE INDEX idx_scheduled_tasks_priority ON scheduled_tasks(priority DESC);
CREATE INDEX idx_scheduled_tasks_type ON scheduled_tasks(type);
```

**验证规则**：
- `name` 必须唯一且非空
- `type` 必须对应已注册的 Handler
- `cron_expr` 格式必须有效（当 `is_recurring=true` 时）
- `max_duration_seconds` 必须 > 0
- `timeout_grace_period` 必须 >= 0 且 < `max_duration_seconds`

**状态转换**：无（配置对象，无状态机）

---

### 实体 2: TaskExecution（任务执行历史）

**用途**：记录每次任务执行的详细信息，用于历史查询、统计分析和问题排查。

**字段定义**：

```go
type TaskExecution struct {
    // 基础字段
    ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
    TaskUID         string    `gorm:"type:varchar(255);not null;index" json:"task_uid"`
    TaskName        string    `gorm:"type:varchar(255);not null;index" json:"task_name"`
    TaskType        string    `gorm:"type:varchar(255);not null" json:"task_type"`

    // 执行上下文
    ExecutionUID    string    `gorm:"type:varchar(255);not null;unique" json:"execution_uid"`
    RunBy           string    `gorm:"type:varchar(255);not null" json:"run_by"`  // 实例 ID
    ScheduledAt     time.Time `gorm:"not null" json:"scheduled_at"`  // 计划执行时间
    StartedAt       time.Time `gorm:"not null;index" json:"started_at"`  // 实际开始时间
    FinishedAt      time.Time `gorm:"" json:"finished_at,omitempty"`  // 实际结束时间

    // 执行结果
    State           ExecutionState `gorm:"type:varchar(32);not null;index" json:"state"`
    Result          string         `gorm:"type:text" json:"result,omitempty"`  // 执行结果数据
    ErrorMessage    string         `gorm:"type:text" json:"error_message,omitempty"`
    ErrorStack      string         `gorm:"type:text" json:"error_stack,omitempty"`

    // 执行指标
    DurationMs      int64  `gorm:"not null;default:0" json:"duration_ms"`  // 执行时长（毫秒）
    Progress        int    `gorm:"not null;default:0" json:"progress"`  // 进度（0-100）
    RetryCount      int    `gorm:"not null;default:0" json:"retry_count"`

    // 触发方式
    TriggerType string `gorm:"type:varchar(32);not null" json:"trigger_type"`  // scheduled, manual
    TriggerBy   string `gorm:"type:varchar(255)" json:"trigger_by,omitempty"`  // 手动触发的用户 ID

    // 时间戳
    CreatedAt time.Time `gorm:"index" json:"created_at"`
    UpdatedAt time.Time `gorm:"index" json:"updated_at"`
}

// ExecutionState 任务执行状态枚举
type ExecutionState string

const (
    ExecutionStatePending   ExecutionState = "pending"    // 等待执行
    ExecutionStateRunning   ExecutionState = "running"    // 执行中
    ExecutionStateSuccess   ExecutionState = "success"    // 执行成功
    ExecutionStateFailure   ExecutionState = "failure"    // 执行失败
    ExecutionStateTimeout   ExecutionState = "timeout"    // 超时失败
    ExecutionStateCancelled ExecutionState = "cancelled"  // 已取消
    ExecutionStateInterrupted ExecutionState = "interrupted" // 系统中断（如重启）
)

// Validate 验证状态值有效性
func (s ExecutionState) Validate() error {
    switch s {
    case ExecutionStatePending, ExecutionStateRunning, ExecutionStateSuccess,
         ExecutionStateFailure, ExecutionStateTimeout, ExecutionStateCancelled,
         ExecutionStateInterrupted:
        return nil
    default:
        return fmt.Errorf("invalid execution state: %s", s)
    }
}
```

**索引设计**：
```sql
CREATE INDEX idx_task_executions_task_uid ON task_executions(task_uid);
CREATE INDEX idx_task_executions_task_name ON task_executions(task_name);
CREATE INDEX idx_task_executions_state ON task_executions(state);
CREATE INDEX idx_task_executions_started_at ON task_executions(started_at DESC);
CREATE INDEX idx_task_executions_composite ON task_executions(task_name, state, started_at DESC);
```

**验证规则**：
- `task_uid` 必须对应存在的 ScheduledTask
- `execution_uid` 必须唯一（UUID）
- `state` 必须是有效的 ExecutionState 枚举值
- `finished_at` >= `started_at`（当状态为终态时）
- `progress` 必须在 0-100 范围内

**状态转换**：
```
pending → running → (success | failure | timeout | cancelled)
running → interrupted（系统重启时）
```

---

### 实体 3: TaskLock（分布式锁记录）

**用途**：记录分布式锁的持有状态，用于数据库锁实现和锁状态监控。

**字段定义**：

```go
type TaskLock struct {
    // 基础字段
    ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
    LockKey    string    `gorm:"type:varchar(255);not null;unique" json:"lock_key"`  // 锁键（如 "scheduler_global"、"task:task-uid"）

    // 持有者信息
    HolderID   string    `gorm:"type:varchar(255);not null" json:"holder_id"`  // 持有者实例 ID
    Token      string    `gorm:"type:varchar(255);not null" json:"token"`  // 锁令牌（UUID）

    // 锁状态
    State      LockState `gorm:"type:varchar(32);not null;index" json:"state"`
    AcquiredAt time.Time `gorm:"not null;index" json:"acquired_at"`  // 获取时间
    ExpiresAt  time.Time `gorm:"not null;index" json:"expires_at"`   // 过期时间
    ReleasedAt time.Time `gorm:"" json:"released_at,omitempty"`      // 释放时间

    // 元数据
    Purpose    string `gorm:"type:varchar(255)" json:"purpose,omitempty"`  // 锁用途描述

    // 时间戳
    CreatedAt time.Time `gorm:"index" json:"created_at"`
    UpdatedAt time.Time `gorm:"index" json:"updated_at"`
}

// LockState 锁状态枚举
type LockState string

const (
    LockStateActive   LockState = "active"    // 活跃状态
    LockStateExpired  LockState = "expired"   // 已过期（但未释放）
    LockStateReleased LockState = "released"  // 已释放
)

// Validate 验证状态值有效性
func (s LockState) Validate() error {
    switch s {
    case LockStateActive, LockStateExpired, LockStateReleased:
        return nil
    default:
        return fmt.Errorf("invalid lock state: %s", s)
    }
}
```

**索引设计**：
```sql
CREATE UNIQUE INDEX idx_task_locks_key ON task_locks(lock_key) WHERE state = 'active';
CREATE INDEX idx_task_locks_state ON task_locks(state);
CREATE INDEX idx_task_locks_expires_at ON task_locks(expires_at);
CREATE INDEX idx_task_locks_holder ON task_locks(holder_id);
```

**验证规则**：
- `lock_key` 必须唯一（对于 `state=active` 的记录）
- `expires_at` > `acquired_at`
- `released_at` >= `acquired_at`（当 `state=released` 时）

**状态转换**：
```
active → released（主动释放）
active → expired（超时自动过期）
```

---

## Audit 数据模型

### 实体 4: AuditEvent（审计事件）

**用途**：记录所有关键操作的审计日志，支持追溯和合规审查。不可修改和删除。

**字段定义**：

```go
type AuditEvent struct {
    // 基础字段
    ID        string    `gorm:"type:varchar(255);primaryKey" json:"id"`  // UUID
    Timestamp int64     `gorm:"not null;index" json:"timestamp"`  // Unix 毫秒时间戳

    // 操作信息
    Action       Action       `gorm:"type:varchar(64);not null;index" json:"action"`
    ResourceType ResourceType `gorm:"type:varchar(64);not null;index" json:"resource_type"`
    Resource     Resource     `gorm:"type:text;serializer:json" json:"resource"`  // JSON 序列化

    // 操作主体
    User      Principal `gorm:"type:text;serializer:json" json:"user"`
    SpacePath string    `gorm:"type:varchar(512)" json:"space_path,omitempty"`  // 空间路径（可选）

    // 差异对象（变更前后）
    DiffObject DiffObject `gorm:"type:text;serializer:json" json:"diff_object,omitempty"`

    // 客户端信息
    ClientIP      string `gorm:"type:varchar(45);index" json:"client_ip"`
    UserAgent     string `gorm:"type:text" json:"user_agent,omitempty"`
    RequestMethod string `gorm:"type:varchar(16)" json:"request_method,omitempty"`  // GET, POST, etc.
    RequestID     string `gorm:"type:varchar(128);index" json:"request_id,omitempty"`

    // 自定义数据
    Data map[string]string `gorm:"type:text;serializer:json" json:"data,omitempty"`

    // 时间戳（仅创建）
    CreatedAt time.Time `gorm:"index" json:"created_at"`
}
```

**复合类型定义**：

```go
// Resource 资源定义
type Resource struct {
    Type       ResourceType      `json:"type"`
    Identifier string            `json:"identifier"`  // 资源 ID
    Data       map[string]string `json:"data,omitempty"`  // 资源元数据（如 resourceName、clusterName）
}

// Principal 操作主体
type Principal struct {
    UID      string        `json:"uid"`
    Username string        `json:"username"`
    Type     PrincipalType `json:"type"`
}

// PrincipalType 主体类型枚举
type PrincipalType string

const (
    PrincipalTypeUser      PrincipalType = "user"
    PrincipalTypeService   PrincipalType = "service"
    PrincipalTypeAnonymous PrincipalType = "anonymous"
)

// DiffObject 差异对象
type DiffObject struct {
    OldObject          string   `json:"old_object,omitempty"`  // JSON 字符串，最大 64KB
    NewObject          string   `json:"new_object,omitempty"`  // JSON 字符串，最大 64KB
    OldObjectTruncated bool     `json:"old_object_truncated"`
    NewObjectTruncated bool     `json:"new_object_truncated"`
    TruncatedFields    []string `json:"truncated_fields,omitempty"`  // 被截断的字段列表
}
```

**索引设计**：
```sql
CREATE INDEX idx_audit_events_timestamp ON audit_events(timestamp DESC);
CREATE INDEX idx_audit_events_action ON audit_events(action);
CREATE INDEX idx_audit_events_resource_type ON audit_events(resource_type);
CREATE INDEX idx_audit_events_client_ip ON audit_events(client_ip);
CREATE INDEX idx_audit_events_request_id ON audit_events(request_id);
CREATE INDEX idx_audit_events_composite ON audit_events(resource_type, action, timestamp DESC);
```

**验证规则**：
- `id` 必须是有效的 UUID
- `action` 必须是有效的 Action 枚举值
- `resource_type` 必须是有效的 ResourceType 枚举值
- `user.uid` 必须非空
- `old_object` 和 `new_object` 长度必须 ≤ 64KB
- `timestamp` 必须 > 0

**不可变性**：
- 不支持 UPDATE 操作
- 不支持 DELETE 操作（仅支持过期清理）
- 只能通过 INSERT 创建审计日志

---

### 实体 5: Action（操作类型枚举）

**用途**：定义所有支持的操作类型，确保操作名称一致性。

**枚举值定义**：

```go
type Action string

const (
    // 基础 CRUD 操作
    ActionCreated Action = "created"
    ActionUpdated Action = "updated"
    ActionDeleted Action = "deleted"
    ActionRead    Action = "read"  // 敏感资源读取操作

    // 状态变更操作
    ActionEnabled  Action = "enabled"
    ActionDisabled Action = "disabled"

    // 特殊操作（参考 Gitness）
    ActionBypassed  Action = "bypassed"   // 绕过检查
    ActionForcePush Action = "forcePush"  // 强制推送

    // 认证操作
    ActionLogin  Action = "login"
    ActionLogout Action = "logout"

    // 权限操作
    ActionGranted Action = "granted"
    ActionRevoked Action = "revoked"
)

// Validate 验证操作类型有效性
func (a Action) Validate() error {
    switch a {
    case ActionCreated, ActionUpdated, ActionDeleted, ActionRead,
         ActionEnabled, ActionDisabled, ActionBypassed, ActionForcePush,
         ActionLogin, ActionLogout, ActionGranted, ActionRevoked:
        return nil
    default:
        return fmt.Errorf("invalid action: %s", a)
    }
}

// String 返回字符串表示
func (a Action) String() string {
    return string(a)
}
```

**扩展指南**：
- 添加新操作时必须更新 `Validate()` 方法
- 操作名称使用 camelCase（遵循 Gitness 约定）
- 避免使用过于宽泛的操作名称（如 "modified"）

---

### 实体 6: ResourceType（资源类型枚举）

**用途**：定义所有被审计的资源类型，确保资源类型名称一致性。

**枚举值定义**：

```go
type ResourceType string

const (
    // Kubernetes 资源
    ResourceTypeCluster     ResourceType = "cluster"
    ResourceTypePod         ResourceType = "pod"
    ResourceTypeDeployment  ResourceType = "deployment"
    ResourceTypeService     ResourceType = "service"
    ResourceTypeConfigMap   ResourceType = "configMap"
    ResourceTypeSecret      ResourceType = "secret"

    // 数据库资源
    ResourceTypeDatabase         ResourceType = "database"
    ResourceTypeDatabaseInstance ResourceType = "databaseInstance"
    ResourceTypeDatabaseUser     ResourceType = "databaseUser"

    // 中间件资源
    ResourceTypeMinIO      ResourceType = "minio"
    ResourceTypeRedis      ResourceType = "redis"
    ResourceTypeMySQL      ResourceType = "mysql"
    ResourceTypePostgreSQL ResourceType = "postgresql"

    // 系统资源
    ResourceTypeUser     ResourceType = "user"
    ResourceTypeRole     ResourceType = "role"
    ResourceTypeInstance ResourceType = "instance"

    // 定时任务资源
    ResourceTypeScheduledTask ResourceType = "scheduledTask"
)

// Validate 验证资源类型有效性
func (r ResourceType) Validate() error {
    switch r {
    case ResourceTypeCluster, ResourceTypePod, ResourceTypeDeployment,
         ResourceTypeService, ResourceTypeConfigMap, ResourceTypeSecret,
         ResourceTypeDatabase, ResourceTypeDatabaseInstance, ResourceTypeDatabaseUser,
         ResourceTypeMinIO, ResourceTypeRedis, ResourceTypeMySQL, ResourceTypePostgreSQL,
         ResourceTypeUser, ResourceTypeRole, ResourceTypeInstance,
         ResourceTypeScheduledTask:
        return nil
    default:
        return fmt.Errorf("invalid resource type: %s", r)
    }
}

// String 返回字符串表示
func (r ResourceType) String() string {
    return string(r)
}
```

**扩展指南**：
- 添加新资源类型时必须更新 `Validate()` 方法
- 资源类型名称使用 camelCase（遵循 Gitness 约定）
- 根据资源层次结构命名（如 `databaseInstance` vs `database`）

---

## 关系图

```
ScheduledTask (1) ─────< (N) TaskExecution
    │                          │
    │                          │ run_by
    │                          │
    └─────── TaskLock (N) ─────┘

AuditEvent ──> Action (枚举)
    │
    └──────> ResourceType (枚举)
    │
    └──────> Principal
    │
    └──────> Resource
    │
    └──────> DiffObject
```

**关系说明**：
- 一个 ScheduledTask 可以有多个 TaskExecution（1:N）
- TaskLock 记录与任务和实例的关联关系（N:N）
- AuditEvent 包含枚举类型和嵌套结构（无外键关联）

---

## 数据约束

### 大小限制

| 字段 | 最大长度 | 处理策略 |
|------|----------|----------|
| `ScheduledTask.data` | 10KB | 超出拒绝创建 |
| `TaskExecution.result` | 64KB | 超出截断并标记 |
| `TaskExecution.error_stack` | 10KB | 超出截断保留栈顶 |
| `AuditEvent.diff_object.old_object` | 64KB | 智能截断字段值 |
| `AuditEvent.diff_object.new_object` | 64KB | 智能截断字段值 |

### 保留期策略

| 表 | 默认保留期 | 清理策略 |
|----|-----------|----------|
| `scheduled_tasks` | 永久 | 手动删除 |
| `task_executions` | 90 天 | 定时任务清理（每天 2AM） |
| `task_locks` | 7 天（已释放） | 定时任务清理（每天 3AM） |
| `audit_events` | 90 天 | 定时任务清理（每天 2AM） |

---

## 迁移策略

### 从现有模型迁移

**旧模型**：`internal/models/audit_log.go` (`AuditLog`)

**迁移脚本**：
```sql
-- 创建新表
CREATE TABLE audit_events (...);

-- 数据迁移（将旧 audit_logs 迁移到新 audit_events）
INSERT INTO audit_events (
    id, timestamp, action, resource_type, resource, user, client_ip,
    user_agent, request_method, request_id, created_at
)
SELECT
    id,
    EXTRACT(EPOCH FROM created_at) * 1000 AS timestamp,
    action,
    resource_type,
    JSON_BUILD_OBJECT(
        'type', resource_type,
        'identifier', COALESCE(resource_id::text, ''),
        'data', JSON_BUILD_OBJECT('resourceName', resource_name)
    ) AS resource,
    JSON_BUILD_OBJECT(
        'uid', COALESCE(user_id::text, ''),
        'username', username,
        'type', 'user'
    ) AS user,
    ip_address AS client_ip,
    user_agent,
    'POST' AS request_method,  -- 假设默认值
    request_id,
    created_at
FROM audit_logs
WHERE created_at > NOW() - INTERVAL '90 days';  -- 只迁移最近 90 天的数据

-- 验证迁移
SELECT COUNT(*) FROM audit_events;
SELECT COUNT(*) FROM audit_logs WHERE created_at > NOW() - INTERVAL '90 days';

-- 备份旧表
ALTER TABLE audit_logs RENAME TO audit_logs_backup;
```

**注意事项**：
- 旧表的 `changes` 字段（JSONB）无法直接映射到 `DiffObject`，需要根据实际数据结构转换
- 迁移脚本需要在应用停止期间执行，避免数据不一致
- 迁移后保留旧表备份至少 30 天

---

## 验证与测试

### 数据完整性测试

**测试场景**：
1. 创建 ScheduledTask 并验证所有字段正确保存
2. 创建 TaskExecution 并验证状态转换符合规则
3. 创建 AuditEvent 并验证不可修改性
4. 验证枚举类型 Validate() 方法正确拒绝无效值

**测试代码示例**：
```go
func TestActionValidate(t *testing.T) {
    validActions := []Action{ActionCreated, ActionUpdated, ActionDeleted}
    for _, action := range validActions {
        assert.NoError(t, action.Validate())
    }

    invalidAction := Action("invalid")
    assert.Error(t, invalidAction.Validate())
}

func TestAuditEventImmutability(t *testing.T) {
    db := setupTestDB(t)

    // 创建审计事件
    event := &AuditEvent{
        ID:        uuid.New().String(),
        Timestamp: time.Now().UnixMilli(),
        Action:    ActionCreated,
        // ... 其他字段
    }
    db.Create(event)

    // 尝试修改（应该失败）
    result := db.Model(event).Update("action", ActionDeleted)
    assert.Error(t, result.Error)  // 期望错误
}
```

### 性能测试

**测试场景**：
1. 批量插入 1000 条 TaskExecution 记录（<2 秒）
2. 查询最近 90 天的审计日志（10000 条记录，<2 秒）
3. 按多条件过滤任务执行历史（<500ms）

---

## 附录：完整 SQL Schema

**PostgreSQL 版本**：

```sql
-- Scheduler 表
CREATE TABLE scheduled_tasks (
    uid VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(255) NOT NULL,
    description TEXT,
    is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
    cron_expr VARCHAR(255),
    interval BIGINT,
    next_run TIMESTAMP,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    max_duration_seconds INTEGER NOT NULL DEFAULT 3600,
    max_retries INTEGER NOT NULL DEFAULT 3,
    timeout_grace_period INTEGER NOT NULL DEFAULT 30,
    max_concurrent INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 0,
    labels TEXT,
    data TEXT,
    total_executions INTEGER NOT NULL DEFAULT 0,
    success_executions INTEGER NOT NULL DEFAULT 0,
    failure_executions INTEGER NOT NULL DEFAULT 0,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_executed_at TIMESTAMP,
    last_failure_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_executions (
    id BIGSERIAL PRIMARY KEY,
    task_uid VARCHAR(255) NOT NULL,
    task_name VARCHAR(255) NOT NULL,
    task_type VARCHAR(255) NOT NULL,
    execution_uid VARCHAR(255) NOT NULL UNIQUE,
    run_by VARCHAR(255) NOT NULL,
    scheduled_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP,
    state VARCHAR(32) NOT NULL,
    result TEXT,
    error_message TEXT,
    error_stack TEXT,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    progress INTEGER NOT NULL DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    trigger_type VARCHAR(32) NOT NULL,
    trigger_by VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_locks (
    id BIGSERIAL PRIMARY KEY,
    lock_key VARCHAR(255) NOT NULL UNIQUE,
    holder_id VARCHAR(255) NOT NULL,
    token VARCHAR(255) NOT NULL,
    state VARCHAR(32) NOT NULL,
    acquired_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    released_at TIMESTAMP,
    purpose VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Audit 表
CREATE TABLE audit_events (
    id VARCHAR(255) PRIMARY KEY,
    timestamp BIGINT NOT NULL,
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource TEXT NOT NULL,
    user TEXT NOT NULL,
    space_path VARCHAR(512),
    diff_object TEXT,
    client_ip VARCHAR(45),
    user_agent TEXT,
    request_method VARCHAR(16),
    request_id VARCHAR(128),
    data TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引（已在上文中定义）
-- ...
```

---

**文档状态**：✅ 完成
**审核者**：AI Agent
**批准日期**：2025-10-19
