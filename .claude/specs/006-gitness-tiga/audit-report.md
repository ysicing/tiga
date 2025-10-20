# 任务计划审计报告

**日期**：2025-10-20
**审计对象**：`.claude/specs/006-gitness-tiga/tasks.md`（48 个任务）
**审计目的**：对比生成的计划与 Tiga 实际情况，识别过度工程化和不必要的任务

---

## 执行摘要

**关键发现**：
- ✅ **Tiga 是单实例应用**，无需分布式锁机制
- ✅ **Scheduler 已存在简单实现**，需要增强而非完全重写
- ✅ **Audit 已是异步的**，使用 Goroutine，无需复杂的缓冲通道
- ⚠️ **48 个任务中约 30% 是过度工程化的**（主要是分布式锁相关）
- ✅ **核心价值任务**：执行历史、任务统计、对象截断、强类型审计

**建议**：
1. **删除**：所有分布式锁相关任务（T017-T021，共 5 个任务）
2. **简化**：数据模型任务（移除 TaskLock 模型）
3. **保留**：执行历史、统计、审计改进、前端页面
4. **调整**：从 48 个任务减少到约 **35 个任务**

---

## 审计依据

### Tiga 当前架构分析

#### 1. 部署模型

**证据**：
- 没有 docker-compose 或 k8s deployment 配置
- `config.yaml` 中无 `instance_id` 或类似配置
- `Application` 结构体中无 `instanceID` 字段
- Scheduler 是单例：`scheduler.NewScheduler()`

**结论**：**Tiga 是单实例应用，不支持多实例分布式部署**

#### 2. 现有 Scheduler 实现

**文件**：`internal/services/scheduler/scheduler.go` (180 lines)

**现有功能**：
```go
type Scheduler struct {
    schedules map[string]*Schedule  // 任务注册表
    mu        sync.RWMutex          // 并发控制
    stopCh    chan struct{}         // 停止信号
    wg        sync.WaitGroup        // 优雅关闭
}

type Schedule struct {
    Task     Task
    Interval time.Duration
    ticker   *time.Ticker
    enabled  bool
    mu       sync.Mutex
}
```

**已有方法**：
- `Add(name string, task Task, interval time.Duration)`
- `Enable(name string)` / `Disable(name string)` ✅
- `Start()` / `Stop()` ✅
- `IsEnabled(name string)` ✅

**缺失功能**（需要添加）：
- ❌ **执行历史记录**（无持久化）
- ❌ **统计数据**（成功率、失败次数）
- ❌ **Cron 表达式支持**（目前只支持固定间隔）
- ❌ **手动触发**
- ❌ **超时控制**
- ❌ **Panic 恢复**（可能已有，需确认）

#### 3. 现有 Audit 实现

**文件**：`internal/api/middleware/audit.go` (239 lines)

**现有机制**：
```go
func (m *AuditMiddleware) AuditLog() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... 捕获请求 ...
        c.Next()

        go func() {  // 已经是异步的！
            auditLog := m.buildAuditLog(c, start, requestBody)
            if err := m.auditRepo.Create(c.Request.Context(), auditLog); err != nil {
                logrus.Errorf("Failed to create audit log: %v", err)
            }
        }()
    }
}
```

**已有功能**：
- ✅ 异步写入（Goroutine）
- ✅ 资源类型/ID 提取
- ✅ 客户端 IP 提取（RealIP）
- ✅ 跳过规则（`shouldSkipAudit()`）
- ✅ 请求/响应体捕获

**使用的 Action 类型**：
```go
func determineAction(method string) string {
    switch method {
    case http.MethodGet:    return "read"
    case http.MethodPost:   return "create"
    case http.MethodPut:    return "update"
    case http.MethodPatch:  return "update"
    case http.MethodDelete: return "delete"
    default:                return "unknown"
    }
}
```
- 简单字符串，不是枚举
- 已经工作良好，无需复杂化

**现有数据模型**：`models.AuditLog`
```go
type AuditLog struct {
    ID            uint      `gorm:"primarykey"`
    UserID        uuid.UUID `gorm:"type:uuid"`
    Username      string    `gorm:"size:255"`
    Action        string    `gorm:"size:50;index"`
    ResourceType  string    `gorm:"size:100;index"`
    ResourceID    string    `gorm:"size:255"`
    Changes       JSONMap   `gorm:"type:jsonb"`  // PostgreSQL JSONB 字段
    ClientIP      string    `gorm:"size:45"`
    UserAgent     string    `gorm:"size:512"`
    Timestamp     time.Time `gorm:"index"`
    CreatedAt     time.Time
}
```

**缺失功能**（需要添加）：
- ❌ **对象差异追踪**（OldObject vs NewObject）
- ❌ **对象截断机制**（64KB 限制）
- ❌ **强类型 Action/ResourceType**（可选，不强求）
- ✅ **异步写入已有**（无需复杂的缓冲通道）

---

## 任务分类与评估

### 阶段 3.1：设置（T001-T003）

| 任务 | 状态 | 评估 |
|------|------|------|
| T001 创建项目结构和分支 | ✅ 保留 | 必要 |
| T002 安装依赖和工具 | ✅ 保留 | 需要 `cron/v3`、`uuid` |
| T003 配置代码检查工具 | ✅ 保留 | 必要 |

**保留 3/3 任务**

---

### 阶段 3.2：测试优先（T004-T011）

#### 契约测试 [P]

| 任务 | 状态 | 评估 |
|------|------|------|
| T004 Scheduler API 契约测试 | ⚠️ **简化** | 8 个端点，移除分布式锁相关验证 |
| T005 Audit API 契约测试 | ✅ 保留 | 4 个端点，验证 API 规范 |

**保留 2/2 任务（1 个需简化）**

#### 集成测试 [P]

| 任务 | 状态 | 评估 |
|------|------|------|
| T006 分布式锁集成测试框架 | ❌ **删除** | Tiga 是单实例，无需分布式锁 |
| T007 任务执行集成测试框架 | ✅ 保留 | 验证任务调度、重试、超时 |
| T008 并发调度集成测试 | ⚠️ **简化** | 验证并发执行，但无需多实例竞争测试 |
| T009 审计日志创建集成测试 | ✅ 保留 | 验证事件记录、IP 提取、异步写入 |
| T010 审计日志查询性能测试 | ✅ 保留 | 验证 10000 条 <2 秒性能目标 |
| T011 对象截断集成测试 | ✅ 保留 | 验证 64KB 限制、智能截断、结构完整性 |

**删除 1 个，简化 1 个，保留 4/6 任务**

---

### 阶段 3.3：核心实现

#### 数据模型 [P] (T012-T016)

| 任务 | 状态 | 评估 |
|------|------|------|
| T012 ScheduledTask 模型创建 | ⚠️ **简化** | 基于现有 `Schedule` 结构增强，无需完全重写 |
| T013 TaskExecution 模型创建 | ✅ 保留 | **新增功能**：执行历史持久化 |
| T014 TaskLock 模型创建 | ❌ **删除** | 单实例无需分布式锁模型 |
| T015 AuditEvent 模型创建 | ⚠️ **简化** | 基于现有 `AuditLog` 扩展，添加 OldObject/NewObject 字段 |
| T016 Action 和 ResourceType 枚举 | ⚠️ **可选** | 现有字符串已工作良好，强类型可选 |

**删除 1 个，简化 3 个，完全保留 1/5 任务**

#### 分布式锁实现 (T017-T021)

| 任务 | 状态 | 评估 |
|------|------|------|
| T017 锁接口抽象设计 | ❌ **删除** | 单实例无需分布式锁 |
| T018 数据库锁实现（默认） | ❌ **删除** | 同上 |
| T019 Redis 锁实现 | ❌ **删除** | 同上 |
| T020 etcd 锁实现 | ❌ **删除** | 同上 |
| T021 锁实现单元测试 | ❌ **删除** | 同上 |

**删除 5/5 任务**（约占总任务的 10%）

#### Scheduler 核心服务 (T022-T027)

| 任务 | 状态 | 评估 |
|------|------|------|
| T022 Scheduler 接口和主调度器实现 | ⚠️ **调整** | **增强现有 Scheduler**，添加 Cron 支持、手动触发，<br>无需"集成分布式锁" |
| T023 任务队列实现 | ⚠️ **简化** | 增强现有 `schedules map`，添加优先级队列（可选） |
| T024 Worker 和 Filter 机制实现 | ⚠️ **简化** | 添加 Panic 恢复、进度报告，Filter 可选 |
| T025 任务执行历史记录 | ✅ 保留 | **核心价值**：持久化执行记录 |
| T026 超时控制机制（Context + 宽限期） | ✅ 保留 | **核心价值**：防止任务挂起 |
| T027 任务统计数据计算 | ✅ 保留 | **核心价值**：成功率、平均时长 |

**0 个删除，3 个简化，3 个完全保留**

#### Audit 核心服务 (T028-T033)

| 任务 | 状态 | 评估 |
|------|------|------|
| T028 Audit 服务接口实现 | ⚠️ **简化** | 基于现有 `AuditMiddleware` 扩展，<br>添加 `RecordEvent()` 方法 |
| T029 Event 验证机制 | ⚠️ **可选** | 现有实现已有基本验证，强验证可选 |
| T030 Functional Options 实现 | ⚠️ **可选** | 现有直接创建 struct 已工作良好，<br>Gitness 风格 Options 可选 |
| T031 对象截断策略实现（64KB 限制） | ✅ 保留 | **核心价值**：智能截断超大对象 |
| T032 异步写入和批量优化 | ❌ **删除** | **已有异步写入**（Goroutine），<br>缓冲通道过度工程化 |
| T033 审计中间件重构 | ⚠️ **简化** | 增强现有中间件，添加 OldObject/NewObject 捕获 |

**删除 1 个，简化 4 个，完全保留 1/6 任务**

#### Repository 层 [P] (T034-T036)

| 任务 | 状态 | 评估 |
|------|------|------|
| T034 Scheduler 仓储实现 | ⚠️ **简化** | 只需 TaskExecutionRepository，<br>无需 TaskLockRepository |
| T035 Audit 仓储实现 | ⚠️ **简化** | 基于现有 `AuditLogRepository` 扩展 |
| T036 查询索引设计和优化 | ✅ 保留 | 性能优化必要 |

**0 个删除，2 个简化，1 个完全保留**

#### API 处理器 [P] (T037-T038)

| 任务 | 状态 | 评估 |
|------|------|------|
| T037 Scheduler API 处理器实现 | ✅ 保留 | 8 个端点，符合契约 |
| T038 Audit API 处理器实现 | ✅ 保留 | 4 个端点，符合契约 |

**保留 2/2 任务**

---

### 阶段 3.4：前端实现 [P] (T039-T041)

| 任务 | 状态 | 评估 |
|------|------|------|
| T039 Scheduler 管理页面 | ✅ 保留 | 任务列表、启用/禁用、手动触发 |
| T040 任务执行历史页面 | ✅ 保留 | 执行记录、统计图表 |
| T041 审计日志页面 | ✅ 保留 | 事件列表、差异对比组件 |

**保留 3/3 任务**

---

### 阶段 3.5：数据迁移和配置 (T042-T044)

| 任务 | 状态 | 评估 |
|------|------|------|
| T042 历史审计日志数据迁移脚本 | ⚠️ **简化** | 从 `audit_logs` 扩展字段，<br>无需完全迁移到新表 |
| T043 现有任务迁移到新 Scheduler | ⚠️ **简化** | 增强现有任务注册，<br>添加执行历史记录 |
| T044 配置文件更新 | ⚠️ **简化** | 无需"锁类型"配置，<br>只需审计保留期、超时等 |

**0 个删除，3 个简化**

---

### 阶段 3.6：优化和文档 (T045-T048)

| 任务 | 状态 | 评估 |
|------|------|------|
| T045 Swagger 文档生成 | ✅ 保留 | 必要 |
| T046 部署文档更新 | ⚠️ **简化** | 无需"分布式锁配置"章节 |
| T047 代码质量检查 | ✅ 保留 | 必要 |
| T048 手动验证（quickstart.md 场景） | ⚠️ **简化** | 移除分布式锁验证场景 |

**0 个删除，2 个简化，2 个完全保留**

---

## 统计总结

| 类别 | 数量 | 占比 |
|------|------|------|
| **完全保留** | 20 | 42% |
| **需要简化** | 18 | 37% |
| **需要删除** | 10 | 21% |
| **总计** | 48 | 100% |

**删除的任务**（10 个）：
- T006: 分布式锁集成测试框架
- T014: TaskLock 模型创建
- T017-T021: 分布式锁实现（5 个任务）
- T032: 异步写入和批量优化（已有异步 Goroutine）

**简化后的任务总数**：约 **35-38 个任务**（取决于可选任务的取舍）

---

## 核心改进建议

### 1. Scheduler 改进（保留价值）

**现有基础**（180 行，功能简单但可工作）：
```go
type Scheduler struct {
    schedules map[string]*Schedule
    mu        sync.RWMutex
    stopCh    chan struct{}
    wg        sync.WaitGroup
}
```

**建议增强**（增量改进，非重写）：
```go
type Scheduler struct {
    schedules map[string]*Schedule  // 保留
    mu        sync.RWMutex          // 保留
    stopCh    chan struct{}         // 保留
    wg        sync.WaitGroup        // 保留

    // 新增字段
    db         *gorm.DB              // 用于记录执行历史
    execRepo   ExecutionRepository   // 执行历史仓储
}

type Schedule struct {
    Task     Task
    Interval time.Duration
    CronExpr string              // 新增：支持 Cron 表达式
    ticker   *time.Ticker
    enabled  bool
    mu       sync.Mutex

    // 新增字段
    MaxDuration     time.Duration  // 超时时间
    MaxRetries      int            // 最大重试次数
    ConsecutiveFails int           // 连续失败次数
}
```

**新增功能**：
- Cron 表达式解析（使用 `robfig/cron/v3`）
- 执行历史持久化（写入 `task_executions` 表）
- 超时控制（Context + 30s 宽限期）
- Panic 恢复
- 手动触发 API
- 统计数据查询

**无需实现**：
- ❌ 分布式锁
- ❌ Job 队列持久化（内存 map 足够）
- ❌ Worker Pool（当前任务量不大）

### 2. Audit 改进（保留价值）

**现有基础**（239 行，已异步、已提取 IP）：
```go
func (m *AuditMiddleware) AuditLog() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 已有功能：
        // - 异步写入（Goroutine）✅
        // - 客户端 IP 提取（RealIP）✅
        // - 资源类型/ID 提取 ✅
        // - 跳过规则 ✅
        go func() {
            auditLog := m.buildAuditLog(c, start, requestBody)
            m.auditRepo.Create(c.Request.Context(), auditLog)
        }()
    }
}
```

**建议增强**（扩展现有模型）：
```go
type AuditLog struct {
    // 现有字段（保留）
    ID           uint
    UserID       uuid.UUID
    Username     string
    Action       string      // 保持字符串，无需枚举
    ResourceType string      // 保持字符串，无需枚举
    ResourceID   string
    Changes      JSONMap     // PostgreSQL JSONB
    ClientIP     string
    UserAgent    string
    Timestamp    time.Time
    CreatedAt    time.Time

    // 新增字段（对象差异追踪）
    OldObject            *string `gorm:"type:text"`         // JSON 字符串，最大 64KB
    NewObject            *string `gorm:"type:text"`         // JSON 字符串，最大 64KB
    OldObjectTruncated   bool    `gorm:"default:false"`     // 是否被截断
    NewObjectTruncated   bool    `gorm:"default:false"`
    TruncatedFields      JSONArray `gorm:"type:jsonb"`      // 被截断的字段列表
    RequestID            string  `gorm:"size:255;index"`    // 关联请求
}
```

**新增功能**：
- 对象差异捕获（中间件拦截 Body）
- 智能截断算法（64KB 限制）
  ```go
  func TruncateObject(obj interface{}) (string, bool, []string) {
      jsonBytes, _ := json.Marshal(obj)
      if len(jsonBytes) <= 64*1024 {
          return string(jsonBytes), false, nil
      }
      // 智能截断：保留结构，截断字段值
      return truncated, true, fields
  }
  ```
- 审计配置 API（保留期调整）

**无需实现**：
- ❌ 复杂的 buffered channel（现有 Goroutine 已足够）
- ❌ Functional Options 模式（直接创建 struct 更简单）
- ❌ 强类型 Action/ResourceType 枚举（字符串已工作良好）

### 3. 数据库 Schema 调整

**新表**（仅 1 个）：
```sql
CREATE TABLE task_executions (
    id BIGSERIAL PRIMARY KEY,
    task_name VARCHAR(255) NOT NULL,
    task_type VARCHAR(255) NOT NULL,
    execution_uid VARCHAR(255) UNIQUE NOT NULL,
    scheduled_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    state VARCHAR(50) NOT NULL, -- pending, running, success, failure, timeout
    result TEXT,
    error_message TEXT,
    error_stack TEXT,
    duration_ms BIGINT,
    retry_count INT DEFAULT 0,
    trigger_type VARCHAR(50) NOT NULL, -- scheduled, manual
    trigger_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_task_executions_task_name ON task_executions(task_name);
CREATE INDEX idx_task_executions_state ON task_executions(state);
CREATE INDEX idx_task_executions_started_at ON task_executions(started_at);
```

**扩展表**（1 个）：
```sql
ALTER TABLE audit_logs
ADD COLUMN old_object TEXT,
ADD COLUMN new_object TEXT,
ADD COLUMN old_object_truncated BOOLEAN DEFAULT FALSE,
ADD COLUMN new_object_truncated BOOLEAN DEFAULT FALSE,
ADD COLUMN truncated_fields JSONB,
ADD COLUMN request_id VARCHAR(255);

CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id);
```

**无需创建**：
- ❌ `task_locks` 表（无分布式锁需求）
- ❌ `scheduled_tasks` 表（内存 map 足够）

---

## 风险评估

### 删除分布式锁的风险

**问题**：如果未来 Tiga 需要多实例部署怎么办？

**缓解措施**：
1. **当前需求优先**：根据现有代码和配置，Tiga 不支持多实例
2. **预留接口**：可在 Scheduler 中预留 `LockProvider` 接口（空实现）
   ```go
   type LockProvider interface {
       TryLock(ctx context.Context, key string) (bool, error)
       Unlock(ctx context.Context, key string) error
   }

   // 当前实现（空操作）
   type NoopLockProvider struct{}
   func (n *NoopLockProvider) TryLock(ctx context.Context, key string) (bool, error) {
       return true, nil // 单实例总是成功
   }
   ```
3. **未来扩展**：如需多实例，只需实现 `DatabaseLockProvider` 或 `RedisLockProvider`
4. **文档说明**：在代码注释中标注"当前单实例，未来可扩展"

### 简化 Audit 的风险

**问题**：Gitness 的 Functional Options 模式更灵活？

**反驳**：
- Tiga 的中间件模式已工作良好
- 直接创建 struct 更简单、更易测试
- 可在未来需要时添加 Options，不影响现有代码

---

## 推荐执行计划

### 优先级 1：核心价值任务（立即执行）

**Scheduler**（6 个任务）：
- T013：TaskExecution 模型创建
- T025：任务执行历史记录
- T026：超时控制机制
- T027：任务统计数据计算
- T022（简化）：增强 Scheduler 支持 Cron
- T037：Scheduler API 处理器

**Audit**（4 个任务）：
- T031：对象截断策略实现（64KB 限制）
- T033（简化）：增强审计中间件捕获对象差异
- T015（简化）：扩展 AuditLog 模型
- T038：Audit API 处理器

**测试**（2 个任务）：
- T007：任务执行集成测试
- T009：审计日志创建集成测试

**总计**：12 个任务

### 优先级 2：必要支持任务（随后执行）

- T001-T003：项目设置
- T004-T005：契约测试
- T010-T011：性能和截断测试
- T034-T036：Repository 和索引
- T039-T041：前端页面
- T045、T047：文档和质量检查

**总计**：15 个任务

### 优先级 3：可选增强任务（根据时间决定）

- T016：强类型 Action/ResourceType 枚举（可选）
- T023：优先级队列（可选）
- T024：Filter 机制（可选）
- T029-T030：强验证和 Functional Options（可选）

**总计**：4 个任务

### 删除任务（不执行）

- T006、T014、T017-T021、T032

**总计**：8 个任务

---

## 总结

**最终任务数**：约 **31 个必要任务** + 4 个可选任务 = **35 个任务**（相比原计划减少 27%）

**核心原则**：
1. ✅ **基于现有代码增强**，而非完全重写
2. ✅ **保留 Tiga 的简洁性**，避免 Gitness 的复杂性
3. ✅ **优先核心价值功能**：执行历史、统计、对象截断
4. ✅ **删除不必要的复杂性**：分布式锁、复杂异步队列
5. ✅ **为未来扩展预留接口**，但不过度设计

**下一步行动**：
1. 用户确认此审计报告
2. 基于审计结果重新生成 `tasks.md`（35 任务版本）
3. 更新 `plan.md` 和 `quickstart.md` 移除分布式锁相关内容
4. 开始执行优先级 1 任务

---

**审计人**：Claude (AI Agent)
**审计日期**：2025-10-20
**批准状态**：待用户确认
