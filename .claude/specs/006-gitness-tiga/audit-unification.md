# 审计系统统一方案

**日期**：2025-10-20
**问题**：Tiga 存在多个审计实现，需要统一
**目标**：一个统一的审计系统，服务所有子系统

---

## 问题分析

### 当前状态

Tiga 目前有 **3 套审计实现**：

#### 1. 全局 HTTP 审计（中间件）
- **表**：`audit_logs`
- **模型**：`models.AuditLog`
- **实现**：`internal/api/middleware/audit.go`（简单 Goroutine）
- **用途**：记录所有 HTTP API 请求
- **字段**：
  ```go
  type AuditLog struct {
      UserID       uuid.UUID
      Username     string
      Action       string        // "read", "create", "update", "delete"
      ResourceType string
      ResourceID   string
      Changes      JSONMap       // JSONB
      ClientIP     string
      UserAgent    string
      Timestamp    time.Time
  }
  ```

#### 2. MinIO 子系统审计
- **表**：`minio_audit_logs`
- **模型**：`models.MinIOAuditLog`
- **实现**：`internal/services/minio/async_audit_logger.go`（使用通用框架）
- **用途**：记录 MinIO 文件操作（上传、下载、删除等）
- **字段**：
  ```go
  type MinIOAuditLog struct {
      InstanceID    uuid.UUID
      OperationType string        // "upload", "download", "delete"
      ResourceType  string        // "file", "bucket", "permission"
      ResourceName  string
      Action        string
      OperatorID    *uuid.UUID
      OperatorName  string
      ClientIP      string
      Status        string        // "success", "failed"
      ErrorMessage  string
      Details       JSONB
  }
  ```

#### 3. Database 子系统审计
- **表**：`db_audit_logs`
- **模型**：`models.DatabaseAuditLog`
- **实现**：`internal/services/database/async_audit_logger.go`（使用通用框架）
- **用途**：记录数据库管理操作（创建数据库、执行 SQL、权限变更等）
- **字段**：
  ```go
  type DatabaseAuditLog struct {
      InstanceID   *uuid.UUID
      Operator     string
      Action       string        // "create_db", "execute_query", "grant_permission"
      TargetType   string        // "database", "user", "permission"
      TargetName   string
      Details      string        // SQL 语句或操作详情
      Success      bool
      ErrorMessage string
      ClientIP     string
  }
  ```

### 已有的统一框架

**好消息**：Tiga 已经有一个通用的异步审计框架！

**位置**：`internal/services/audit/`
- `interface.go` - 定义 `AuditLog` 接口和 `AuditRepository[T]` 接口
- `async_logger.go` - 通用泛型实现 `AsyncLogger[T AuditLog]`

**特性**：
- ✅ Go 泛型支持多种审计日志类型
- ✅ 异步批量写入（默认批大小 50，5 秒刷新）
- ✅ 多 Worker 并发（默认 2 个）
- ✅ 优雅关闭
- ✅ Channel 满时丢弃并告警

**当前使用情况**：
- MinIO 和 Database 子系统**已经在使用**这个框架
- 全局 HTTP 中间件**未使用**（仍使用简单 Goroutine）

### 问题总结

| 问题 | 影响 | 优先级 |
|------|------|--------|
| **3 张独立的审计表** | 查询分散、数据孤岛、无法关联分析 | 🔴 高 |
| **字段不统一** | 无法用统一接口查询 | 🔴 高 |
| **中间件未使用通用框架** | 性能不一致、代码重复 | 🟡 中 |
| **缺少对象差异追踪** | 无法追踪变更内容 | 🔴 高（本次重构目标） |
| **无法跨子系统关联** | 例如：MinIO 文件删除 → 审计日志无法关联到 HTTP 请求 | 🟢 低 |

---

## 统一方案

### 方案 A：单表统一（推荐）⭐

**核心思想**：所有审计日志写入同一张表 `audit_events`，使用灵活的 JSON 字段存储子系统特定数据。

#### 统一数据模型

```go
// AuditEvent 是统一的审计事件模型
type AuditEvent struct {
    BaseModel

    // 核心字段（所有子系统通用）
    UserID       uuid.UUID  `gorm:"type:uuid;index" json:"user_id"`
    Username     string     `gorm:"size:255;index" json:"username"`
    Action       string     `gorm:"size:50;index" json:"action"`          // "created", "updated", "deleted", "read", "executed"
    ResourceType string     `gorm:"size:100;index" json:"resource_type"`  // "file", "database", "user", "pod", "cluster"
    ResourceID   string     `gorm:"size:255;index" json:"resource_id"`

    // 对象差异追踪（本次重构新增）
    OldObject            *string  `gorm:"type:text" json:"old_object,omitempty"`              // JSON 字符串，最大 64KB
    NewObject            *string  `gorm:"type:text" json:"new_object,omitempty"`              // JSON 字符串，最大 64KB
    OldObjectTruncated   bool     `gorm:"default:false" json:"old_object_truncated"`
    NewObjectTruncated   bool     `gorm:"default:false" json:"new_object_truncated"`
    TruncatedFields      JSONArray `gorm:"type:jsonb" json:"truncated_fields,omitempty"`

    // 子系统特定数据（JSONB 灵活存储）
    Subsystem    string    `gorm:"size:50;index" json:"subsystem"`        // "http", "minio", "database", "k8s", "scheduler"
    Metadata     JSONB     `gorm:"type:jsonb" json:"metadata"`            // 子系统特定字段

    // 执行结果
    Status       string    `gorm:"size:50;index" json:"status"`           // "success", "failed", "pending"
    ErrorMessage string    `gorm:"type:text" json:"error_message,omitempty"`

    // 请求上下文
    ClientIP     string    `gorm:"size:45;index" json:"client_ip"`
    UserAgent    string    `gorm:"size:512" json:"user_agent,omitempty"`
    RequestID    string    `gorm:"size:255;index" json:"request_id,omitempty"`  // 关联 HTTP 请求
    RequestMethod string   `gorm:"size:10" json:"request_method,omitempty"`     // "GET", "POST", "DELETE"

    // 时间戳
    Timestamp    time.Time `gorm:"index" json:"timestamp"`
}

func (AuditEvent) TableName() string { return "audit_events" }
```

#### Metadata 字段示例

**MinIO 子系统**：
```json
{
  "instance_id": "minio-uuid-1234",
  "operation_type": "upload",
  "bucket": "my-bucket",
  "object_key": "files/document.pdf",
  "file_size": 1048576,
  "content_type": "application/pdf"
}
```

**Database 子系统**：
```json
{
  "instance_id": "db-uuid-5678",
  "db_type": "postgresql",
  "target_database": "production_db",
  "query": "SELECT * FROM users WHERE id = $1",
  "execution_time_ms": 45,
  "rows_affected": 1
}
```

**Scheduler 子系统**：
```json
{
  "task_name": "alert_processing",
  "execution_uid": "exec-uuid-9999",
  "duration_ms": 12345,
  "retry_count": 0,
  "trigger_type": "scheduled"
}
```

#### 优点 ✅

1. **单一数据源**：所有审计数据在一张表，易于查询和分析
2. **统一查询接口**：一个 Repository，一套 API
3. **灵活扩展**：新增子系统无需新表，只需在 Metadata 中添加字段
4. **易于关联**：通过 `RequestID` 关联 HTTP 请求和子系统操作
5. **统一安全策略**：保留期、加密、访问控制统一管理
6. **简化维护**：一套索引、一套清理任务

#### 缺点 ⚠️

1. **表可能很大**：所有子系统的审计都在一张表（可通过分区表解决）
2. **JSONB 查询性能**：Metadata 字段查询需要 PostgreSQL JSONB 索引
3. **Schema 灵活性 vs 类型安全**：Metadata 是动态的，失去部分类型安全

#### 迁移策略

**阶段 1**：创建统一表（不影响现有功能）
```sql
CREATE TABLE audit_events (
    -- 基本字段
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    username VARCHAR(255),
    action VARCHAR(50),
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),

    -- 对象差异
    old_object TEXT,
    new_object TEXT,
    old_object_truncated BOOLEAN DEFAULT FALSE,
    new_object_truncated BOOLEAN DEFAULT FALSE,
    truncated_fields JSONB,

    -- 子系统和元数据
    subsystem VARCHAR(50),
    metadata JSONB,

    -- 执行结果
    status VARCHAR(50),
    error_message TEXT,

    -- 请求上下文
    client_ip VARCHAR(45),
    user_agent VARCHAR(512),
    request_id VARCHAR(255),
    request_method VARCHAR(10),

    -- 时间戳
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX idx_audit_events_action ON audit_events(action);
CREATE INDEX idx_audit_events_resource_type ON audit_events(resource_type);
CREATE INDEX idx_audit_events_subsystem ON audit_events(subsystem);
CREATE INDEX idx_audit_events_status ON audit_events(status);
CREATE INDEX idx_audit_events_timestamp ON audit_events(timestamp);
CREATE INDEX idx_audit_events_request_id ON audit_events(request_id);

-- JSONB 索引（PostgreSQL）
CREATE INDEX idx_audit_events_metadata ON audit_events USING GIN (metadata);
```

**阶段 2**：新审计写入统一表
- 中间件：写入 `audit_events`（`subsystem="http"`）
- MinIO 服务：写入 `audit_events`（`subsystem="minio"`）
- Database 服务：写入 `audit_events`（`subsystem="database"`）
- Scheduler 服务：写入 `audit_events`（`subsystem="scheduler"`）

**阶段 3**：历史数据迁移（可选）
- 从 `audit_logs`、`minio_audit_logs`、`db_audit_logs` 迁移到 `audit_events`
- 保留原表 90 天，逐步清理

**阶段 4**：删除旧表和旧代码
- 删除 `models.MinIOAuditLog`、`models.DatabaseAuditLog`
- 删除 `repository/minio/audit_repository.go`、`repository/database/audit.go`
- 统一使用 `repository.AuditEventRepository`

---

### 方案 B：多表 + 视图统一（保守）

**核心思想**：保留现有 3 张表，创建统一视图用于查询。

#### 统一视图

```sql
CREATE VIEW unified_audit_view AS
SELECT
    id, user_id, username, 'http' AS subsystem, action, resource_type, resource_id,
    client_ip, user_agent, NULL AS instance_id, NULL AS operation_type,
    timestamp, created_at
FROM audit_logs

UNION ALL

SELECT
    id, operator_id AS user_id, operator_name AS username, 'minio' AS subsystem,
    action, resource_type, resource_name AS resource_id,
    client_ip, NULL AS user_agent, instance_id, operation_type,
    created_at AS timestamp, created_at
FROM minio_audit_logs

UNION ALL

SELECT
    id, NULL AS user_id, operator AS username, 'database' AS subsystem,
    action, target_type AS resource_type, target_name AS resource_id,
    client_ip, NULL AS user_agent, instance_id, NULL AS operation_type,
    created_at AS timestamp, created_at
FROM db_audit_logs;
```

#### 优点 ✅

1. **零破坏性**：不影响现有代码和数据
2. **渐进迁移**：可以逐步迁移到统一模型
3. **性能隔离**：每个子系统独立表，互不影响

#### 缺点 ⚠️

1. **查询复杂**：UNION ALL 性能较差
2. **字段不统一**：视图需要大量 NULL 和类型转换
3. **仍有重复代码**：3 个 Repository、3 个 AsyncLogger 包装
4. **难以添加统一字段**：例如 OldObject/NewObject 需要在 3 张表都添加

---

## 推荐方案：方案 A（单表统一）

### 原因

1. **符合本次重构目标**：需要添加 OldObject/NewObject 字段，方案 A 只需在一张表添加
2. **简化维护**：一套代码、一套测试、一套 API
3. **易于扩展**：未来新增子系统（如 K8s、Host 监控）无需新表
4. **统一查询**：前端只需调用一个 API 即可查询所有审计日志
5. **已有框架支持**：`audit.AsyncLogger[T]` 框架已经支持泛型，改造成本低

### 实施步骤

#### 第一阶段：数据模型和 Repository（优先级 1）

**任务**：
1. 创建 `models.AuditEvent`（统一模型）
2. 创建 `repository.AuditEventRepository`
3. 实现 `AuditEvent` 实现 `audit.AuditLog` 接口

**文件**：
- `internal/models/audit_event.go`（新文件）
- `internal/repository/audit_event_repo.go`（新文件）

**代码示例**：
```go
// internal/models/audit_event.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type AuditEvent struct {
    BaseModel

    // 核心字段
    UserID       uuid.UUID  `gorm:"type:uuid;index" json:"user_id"`
    Username     string     `gorm:"size:255;index" json:"username"`
    Action       string     `gorm:"size:50;index" json:"action"`
    ResourceType string     `gorm:"size:100;index" json:"resource_type"`
    ResourceID   string     `gorm:"size:255;index" json:"resource_id"`

    // 对象差异（本次重构新增）
    OldObject            *string   `gorm:"type:text" json:"old_object,omitempty"`
    NewObject            *string   `gorm:"type:text" json:"new_object,omitempty"`
    OldObjectTruncated   bool      `gorm:"default:false" json:"old_object_truncated"`
    NewObjectTruncated   bool      `gorm:"default:false" json:"new_object_truncated"`
    TruncatedFields      JSONArray `gorm:"type:jsonb" json:"truncated_fields,omitempty"`

    // 子系统特定
    Subsystem    string    `gorm:"size:50;index" json:"subsystem"`
    Metadata     JSONB     `gorm:"type:jsonb" json:"metadata"`

    // 执行结果
    Status       string    `gorm:"size:50;index" json:"status"`
    ErrorMessage string    `gorm:"type:text" json:"error_message,omitempty"`

    // 请求上下文
    ClientIP     string    `gorm:"size:45;index" json:"client_ip"`
    UserAgent    string    `gorm:"size:512" json:"user_agent,omitempty"`
    RequestID    string    `gorm:"size:255;index" json:"request_id,omitempty"`
    RequestMethod string   `gorm:"size:10" json:"request_method,omitempty"`

    // 时间戳
    Timestamp    time.Time `gorm:"index" json:"timestamp"`
}

func (AuditEvent) TableName() string { return "audit_events" }

// 实现 audit.AuditLog 接口
func (a *AuditEvent) GetID() string {
    return a.ID.String()
}

func (a *AuditEvent) SetCreatedAt(t time.Time) {
    a.CreatedAt = t
}
```

#### 第二阶段：中间件改造（优先级 2）

**任务**：
1. 中间件使用 `audit.AsyncLogger[*models.AuditEvent]`
2. 添加对象差异捕获
3. 调用 `TruncateObject()` 处理超大对象

**文件**：
- `internal/api/middleware/audit.go`（修改现有文件）

**改动**：
```go
// 旧代码
type AuditMiddleware struct {
    auditRepo *repository.AuditLogRepository
}

func (m *AuditMiddleware) AuditLog() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ...
        go func() {  // 简单 Goroutine
            auditLog := m.buildAuditLog(c, start, requestBody)
            m.auditRepo.Create(c.Request.Context(), auditLog)
        }()
    }
}

// 新代码
type AuditMiddleware struct {
    asyncLogger *audit.AsyncLogger[*models.AuditEvent]
    truncator   *audit.ObjectTruncator
}

func (m *AuditMiddleware) AuditLog() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 捕获请求 Body
        oldObject := extractOldObject(c)
        c.Next()
        // 捕获响应 Body
        newObject := extractNewObject(c)

        // 截断超大对象
        oldObjStr, oldTruncated, oldFields := m.truncator.Truncate(oldObject)
        newObjStr, newTruncated, newFields := m.truncator.Truncate(newObject)

        // 构建审计事件
        event := &models.AuditEvent{
            UserID:       getCurrentUser(c).ID,
            Username:     getCurrentUser(c).Username,
            Action:       determineAction(c.Request.Method),
            ResourceType: extractResourceType(c),
            ResourceID:   extractResourceID(c),
            Subsystem:    "http",
            OldObject:    oldObjStr,
            NewObject:    newObjStr,
            OldObjectTruncated: oldTruncated,
            NewObjectTruncated: newTruncated,
            TruncatedFields:    append(oldFields, newFields...),
            ClientIP:     RealIP(c.Request),
            RequestID:    c.GetString("request_id"),
            Timestamp:    time.Now().UTC(),
        }

        // 异步写入（使用通用框架）
        m.asyncLogger.Enqueue(event)
    }
}
```

#### 第三阶段：MinIO 和 Database 改造（优先级 3）

**任务**：
1. MinIO 服务改用 `models.AuditEvent`
2. Database 服务改用 `models.AuditEvent`
3. 将特定字段序列化到 `Metadata` JSONB 字段

**示例**（MinIO）：
```go
// 旧代码
auditLog := &models.MinIOAuditLog{
    InstanceID:    instanceID,
    OperationType: "upload",
    ResourceType:  "file",
    ResourceName:  objectKey,
    Status:        "success",
}
asyncLogger.LogOperation(ctx, auditLog)

// 新代码
metadata := models.JSONB{
    "instance_id":    instanceID.String(),
    "operation_type": "upload",
    "bucket":         bucketName,
    "object_key":     objectKey,
    "file_size":      fileSize,
    "content_type":   contentType,
}

event := &models.AuditEvent{
    UserID:       userID,
    Username:     username,
    Action:       "created",
    ResourceType: "file",
    ResourceID:   objectKey,
    Subsystem:    "minio",
    Metadata:     metadata,
    Status:       "success",
    ClientIP:     clientIP,
    Timestamp:    time.Now().UTC(),
}

asyncLogger.Enqueue(event)
```

#### 第四阶段：数据迁移（可选）

**任务**：
1. 迁移 `audit_logs` → `audit_events`
2. 迁移 `minio_audit_logs` → `audit_events`
3. 迁移 `db_audit_logs` → `audit_events`

**SQL 脚本**：
```sql
-- 迁移全局审计日志
INSERT INTO audit_events (
    id, user_id, username, action, resource_type, resource_id,
    subsystem, metadata, status, client_ip, user_agent, timestamp, created_at
)
SELECT
    id, user_id, username, action, resource_type, resource_id,
    'http' AS subsystem,
    changes AS metadata,  -- Changes 字段映射到 Metadata
    'success' AS status,
    client_ip, user_agent, timestamp, created_at
FROM audit_logs
WHERE created_at > NOW() - INTERVAL '90 days';

-- 迁移 MinIO 审计日志
INSERT INTO audit_events (
    id, user_id, username, action, resource_type, resource_id,
    subsystem, metadata, status, error_message, client_ip, timestamp, created_at
)
SELECT
    id, operator_id, operator_name, action, resource_type, resource_name,
    'minio' AS subsystem,
    jsonb_build_object(
        'instance_id', instance_id,
        'operation_type', operation_type,
        'details', details
    ) AS metadata,
    status, error_message, client_ip, created_at, created_at
FROM minio_audit_logs
WHERE created_at > NOW() - INTERVAL '90 days';

-- 迁移 Database 审计日志（类似）
```

#### 第五阶段：清理旧代码（优先级 5）

**任务**：
1. 删除 `models.MinIOAuditLog`、`models.DatabaseAuditLog`
2. 删除 `repository/minio/audit_repository.go`
3. 删除 `repository/database/audit.go`
4. 删除 `services/minio/async_audit_logger.go`（包装类）
5. 删除 `services/database/async_audit_logger.go`（包装类）
6. 删除旧表（90 天后）

---

## 与本次重构的集成

### 任务列表调整

原 `tasks.md` 中的任务需要调整：

**新增任务**（统一审计）：
- **T036** [P] 创建统一 AuditEvent 模型
  - 创建 `models.AuditEvent`
  - 实现 `audit.AuditLog` 接口
  - **文件**：`internal/models/audit_event.go`
  - **验证**：模型编译，迁移创建表

- **T037** [P] 创建统一 AuditEventRepository
  - 实现 `AuditEventRepository`
  - 实现 `audit.AuditRepository[*models.AuditEvent]` 接口
  - **文件**：`internal/repository/audit_event_repo.go`
  - **依赖**：T036

- **T038** MinIO 改用统一审计（可选）
  - 修改 MinIO 服务使用 `models.AuditEvent`
  - 将特定字段序列化到 `Metadata`
  - **文件**：`internal/services/minio/`（多个文件）
  - **依赖**：T036、T037

- **T039** Database 改用统一审计（可选）
  - 修改 Database 服务使用 `models.AuditEvent`
  - **文件**：`internal/services/database/`（多个文件）
  - **依赖**：T036、T037

**调整现有任务**：
- **T012**（原 AuditLog 模型扩展）→ 改为创建统一 `models.AuditEvent` 模型
- **T017**（原 Audit 中间件增强）→ 改用 `audit.AsyncLogger[*models.AuditEvent]`
- **T020**（原 Audit 仓储扩展）→ 改为创建统一 `AuditEventRepository`

**任务总数调整**：
- 原 35 个任务
- 修改 3 个任务（T012、T017、T020）改为统一审计
- 新增 2 个任务（T036-T037：子系统迁移，可选）
- **新总数**：37 个任务

---

## 总结

### 短期目标（本次重构）

1. ✅ 创建统一 `models.AuditEvent` 模型
2. ✅ 中间件改用 `audit.AsyncLogger[*models.AuditEvent]`
3. ✅ 添加对象差异追踪（OldObject/NewObject）
4. ✅ 实现 64KB 截断机制
5. ⚠️ MinIO 和 Database 迁移（可选，根据时间决定）

### 中期目标（后续重构）

1. 迁移 MinIO 和 Database 子系统到统一审计
2. 历史数据迁移（90 天内）
3. 删除旧表和旧代码

### 长期目标

1. 新增子系统（K8s、Host 监控、Scheduler）统一使用 `models.AuditEvent`
2. 审计数据分析和可视化
3. 审计日志导出（Elasticsearch、Splunk）

---

**下一步**：✅ **已完成** - 用户已确认统一方案，`tasks.md` 已更新至 v3.0（37 任务）
