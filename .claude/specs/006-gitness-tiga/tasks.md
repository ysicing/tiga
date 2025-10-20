# 任务：定时任务和审计系统重构（简化版）

**输入**：来自 `.claude/specs/006-gitness-tiga/` 的设计文档 + 审计报告 + 审计统一方案
**前提条件**：plan.md、research.md、data-model.md、contracts/、quickstart.md、audit-report.md、audit-unification.md
**功能分支**：`006-gitness-tiga`
**版本**：v3.0（审计统一版）

---

## ⚠️ 重要：审计结果与统一方案

本任务列表基于两次审计优化：

**审计 1 - 简化**（`audit-report.md`）：
- **删除**：所有分布式锁相关任务（Tiga 是单实例应用）
- **简化**：基于现有代码增强，而非完全重写
- **保留**：核心价值功能（执行历史、统计、对象截断）
- **结果**：从 48 任务减少到 35 任务（减少 27%）

**审计 2 - 统一**（`audit-unification.md`）：
- **问题**：发现 3 套独立审计实现（HTTP、MinIO、Database）
- **方案**：采用方案 A（单表统一 `audit_events`）
- **变更**：
  - T012：改为创建统一 `models.AuditEvent` 模型
  - T020：改为创建统一 `AuditEventRepository`
  - T017：使用统一 `audit.AsyncLogger[*AuditEvent]`
  - 新增 T036-T037：MinIO 和 Database 迁移任务（可选）
- **结果**：从 35 任务增加到 **37 任务**

---

## 执行流程

基于以下设计文档生成任务：
- ✅ plan.md：技术栈（Go 1.24+、React 19）、项目结构
- ✅ research.md：Gitness 架构研究（参考，非照搬）
- ✅ data-model.md：简化后的数据模型（2 个新表 + 1 个统一表）
- ✅ contracts/：2 个 API 契约（scheduler_api.yaml、audit_api.yaml）
- ✅ quickstart.md：测试场景（移除分布式锁场景）
- ✅ audit-report.md：审计结果和改进策略
- ✅ audit-unification.md：审计系统统一方案（方案 A）

---

## 格式说明

- **[P]**：可以并行运行（不同文件，无依赖关系）
- **⚠️ 简化**：基于现有代码增强
- **路径约定**：
  - 后端：`internal/models/`、`internal/services/`、`internal/repository/`、`internal/api/handlers/`
  - 前端：`ui/src/pages/`、`ui/src/components/`、`ui/src/services/`
  - 测试：`tests/contract/`、`tests/integration/`、`tests/unit/`

---

## 阶段 3.1：设置

- [X] **T001** 创建项目结构和分支
  - 创建功能分支 `006-gitness-tiga`
  - 创建目录：`internal/services/scheduler/`（已存在，确认）、`tests/contract/`、`tests/integration/scheduler/`、`tests/integration/audit/`
  - **文件**：目录结构
  - **验证**：所有目录存在

- [X] **T002** 安装依赖和工具
  - Go 依赖：`github.com/robfig/cron/v3`（cron 表达式）、`github.com/google/uuid`（UUID 生成）
  - 测试工具：`github.com/stretchr/testify`、`github.com/testcontainers/testcontainers-go`
  - **文件**：`go.mod`
  - **验证**：`go mod tidy` 成功

- [X] **T003** [P] 配置代码检查工具
  - 确认 golangci-lint 配置（`.golangci.yml`）
  - 确认前端 ESLint 配置（`ui/.eslintrc.js`）
  - **文件**：`.golangci.yml`、`ui/eslint.config.js`
  - **验证**：`task gofmt` 通过

---

## 阶段 3.2：测试优先（TDD）⚠️ 必须在 3.3 之前完成

**关键：这些测试必须编写并且必须在任何实现之前失败**

### 契约测试 [P]

- [X] **T004** [P] Scheduler API 契约测试 ⚠️ 简化
  - 测试 8 个端点：
    - `GET /api/v1/scheduler/tasks`
    - `GET /api/v1/scheduler/tasks/:id`
    - `POST /api/v1/scheduler/tasks/:id/enable`
    - `POST /api/v1/scheduler/tasks/:id/disable`
    - `POST /api/v1/scheduler/tasks/:id/trigger`
    - `GET /api/v1/scheduler/executions`
    - `GET /api/v1/scheduler/executions/:id`
    - `GET /api/v1/scheduler/stats`
  - **简化**：无需验证分布式锁行为
  - **文件**：`tests/contract/scheduler_contract_test.go` ✅
  - **验证**：测试编译并全部失败（404）✅
  - **参考**：`contracts/scheduler_api.yaml`

- [X] **T005** [P] Audit API 契约测试
  - 测试 4 个端点：
    - `GET /api/v1/audit/events`
    - `GET /api/v1/audit/events/:id`
    - `GET /api/v1/audit/config`
    - `PUT /api/v1/audit/config`
  - **文件**：`tests/contract/audit_contract_test.go` ✅
  - **验证**：测试编译成功 ✅
  - **参考**：`contracts/audit_api.yaml`

### 集成测试 [P]

- [X] **T006** [P] 任务执行集成测试框架
  - 测试场景：
    - 任务调度和执行
    - 任务失败重试
    - 任务超时控制
    - 手动触发任务
  - **简化**：单实例测试，无需多实例竞争
  - **文件**：`tests/integration/scheduler/task_execution_test.go` ✅
  - **验证**：测试编译成功 ✅
  - **参考**：`quickstart.md` 场景 3

- [X] **T007** [P] 并发调度集成测试 ⚠️ 简化
  - 测试场景：
    - 多个任务并发执行（单实例内）
    - 最大并发控制
    - 优先级调度（可选）
  - **简化**：无需测试多实例锁竞争
  - **文件**：`tests/integration/scheduler/concurrent_test.go` ✅
  - **验证**：测试编译成功 ✅
  - **参考**：`quickstart.md` 场景 1

- [X] **T008** [P] 审计日志创建集成测试
  - 测试场景：
    - 审计事件记录（创建、更新、删除操作）
    - 客户端 IP 提取（X-Forwarded-For、X-Real-IP）
    - 对象差异追踪（OldObject vs NewObject）
    - 异步写入不阻塞业务操作（验证现有 Goroutine）
  - **文件**：`tests/integration/audit/event_creation_test.go` ✅
  - **验证**：测试编译成功 ✅
  - **参考**：`quickstart.md` 场景 4

- [X] **T009** [P] 审计日志查询性能测试
  - 测试场景：
    - 10000 条记录查询 <2 秒
    - 多维度过滤（资源类型、操作、时间范围）
    - 分页正确性
    - 索引使用验证
  - **文件**：`tests/integration/audit/query_performance_test.go` ✅
  - **验证**：测试编译成功 ✅
  - **参考**：`plan.md` 性能目标

- [X] **T010** [P] 对象截断集成测试
  - 测试场景：
    - 对象 ≤64KB 不截断
    - 对象 >64KB 智能截断字段值
    - 截断后 JSON 结构完整
    - 截断标识正确记录
  - **文件**：`tests/integration/audit/truncation_test.go`
  - **验证**：测试编译但全部失败（截断逻辑未实现）
  - **参考**：`quickstart.md` 场景 5

---

## 阶段 3.3：核心实现（仅在测试失败后）

### 数据模型 [P]

- [ ] **T011** [P] TaskExecution 模型创建
  - 创建 GORM 模型，包含所有字段（参考 data-model.md）
  - 实现 `ExecutionState` 枚举和 `Validate()` 方法
  - 添加索引（task_name、state、started_at）
  - **文件**：`internal/models/task_execution.go`
  - **验证**：模型可以编译，枚举验证正常
  - **参考**：`data-model.md` 实体 2

- [ ] **T012** [P] 创建统一 AuditEvent 模型 🆕 统一审计
  - 创建 GORM 模型 `models.AuditEvent`（替换分散的审计表）
  - 实现 `audit.AuditLog` 接口（`GetID()`、`SetCreatedAt()`）
  - 包含统一字段：
    - **核心字段**：UserID, Username, Action, ResourceType, ResourceID
    - **对象差异**：OldObject, NewObject, OldObjectTruncated, NewObjectTruncated, TruncatedFields (TEXT, 最大 64KB)
    - **子系统特定**：Subsystem ("http", "minio", "database", "scheduler"), Metadata (JSONB)
    - **执行结果**：Status, ErrorMessage
    - **请求上下文**：ClientIP, UserAgent, RequestID, RequestMethod
    - **时间戳**：Timestamp
  - 添加索引（user_id、action、resource_type、subsystem、status、timestamp、request_id）
  - 添加 JSONB GIN 索引（PostgreSQL）
  - **文件**：`internal/models/audit_event.go`（新文件）
  - **验证**：模型编译，迁移创建 `audit_events` 表
  - **参考**：`audit-unification.md` 方案 A

### Scheduler 核心服务

- [ ] **T013** Scheduler 功能增强 ⚠️ 简化
  - **基于现有 `internal/services/scheduler/scheduler.go` 增强**：
    - 添加 `execRepo ExecutionRepository` 字段
    - 添加 `AddCron(name, cronExpr string, task Task)` 方法
    - 添加 `Trigger(name string)` 方法（手动触发）
    - 实现执行历史记录（调用 `execRepo.Create()`）
    - 实现 Panic 恢复机制
  - **无需实现**：分布式锁、Job 队列持久化
  - **文件**：`internal/services/scheduler/scheduler.go`（修改现有文件）
  - **验证**：可以添加 Cron 任务，T006 部分测试通过
  - **依赖**：T011
  - **参考**：`audit-report.md` 第 3.1 节

- [ ] **T014** 任务执行历史记录
  - 在 Scheduler 执行逻辑中添加历史记录写入
  - 实现状态转换跟踪（pending → running → success/failure/timeout）
  - 实现错误堆栈记录
  - 实现执行时长计算
  - **文件**：`internal/services/scheduler/history.go`（新文件）
  - **验证**：执行历史正确记录，T006 测试通过
  - **依赖**：T013

- [ ] **T015** 超时控制机制（Context + 宽限期）
  - 实现 Context 超时控制
  - 实现 30 秒宽限期机制
  - **文件**：`internal/services/scheduler/timeout.go`（新文件）
  - **验证**：超时任务正确终止，T006 测试通过
  - **依赖**：T013
  - **参考**：`research.md` 第 5 节

- [ ] **T016** 任务统计数据计算
  - 实现统计数据聚合（成功率、平均执行时间）
  - 实现单任务统计和全局统计
  - **文件**：`internal/services/scheduler/stats.go`（新文件）
  - **验证**：统计数据准确
  - **依赖**：T014
  - **参考**：`contracts/scheduler_api.yaml` /stats 端点

### Audit 核心服务

- [ ] **T017** Audit 中间件增强 🆕 使用统一审计
  - **基于现有 `internal/api/middleware/audit.go` 增强**：
    - 修改为使用统一 `audit.AsyncLogger[*models.AuditEvent]`（替换简单 Goroutine）
    - 在 `buildAuditLog()` 中构建 `AuditEvent` 对象
    - 从请求 Body 和响应 Body 提取 OldObject/NewObject
    - 调用 `TruncateObject()` 处理超大对象
    - 设置 `Subsystem = "http"`
  - **文件**：`internal/api/middleware/audit.go`（修改现有文件）
  - **验证**：所有请求自动记录到 `audit_events` 表，包含对象差异
  - **依赖**：T012（AuditEvent 模型）、T018（TruncateObject）
  - **参考**：`audit-unification.md` 第二阶段

- [ ] **T018** 对象截断策略实现（64KB 限制）
  - 实现 `TruncateObject()` 方法（智能截断算法）
  - 保留 JSON 结构，截断字段值
  - 记录截断字段列表
  - **文件**：`internal/services/audit/truncation.go`（新文件）
  - **验证**：T010 测试通过，JSON 可解析
  - **参考**：`research.md` 第 4 节

### Repository 层 [P]

- [ ] **T019** [P] Scheduler 仓储实现 ⚠️ 简化
  - 实现 `TaskExecutionRepository`（CRUD、历史查询、统计）
  - **无需实现**：TaskLockRepository（无分布式锁）
  - **文件**：`internal/repository/scheduler/execution.go`（新文件）
  - **验证**：仓储方法正常工作
  - **依赖**：T011

- [ ] **T020** [P] 创建统一 AuditEventRepository 🆕 统一审计
  - 创建 `repository.AuditEventRepository`（新仓储）
  - 实现 `audit.AuditRepository[*models.AuditEvent]` 接口
  - 实现方法：
    - `Create(ctx, event)` - 创建审计事件
    - `BatchCreate(ctx, events)` - 批量创建（AsyncLogger 使用）
    - `GetByID(ctx, id)` - 查询单个事件
    - `List(ctx, filters)` - 多维度过滤查询（支持 Subsystem、Action、ResourceType、时间范围）
    - `Count(ctx, filters)` - 计数查询
    - `DeleteOlderThan(ctx, retentionDays)` - 清理过期日志
  - 优化分页查询性能
  - **文件**：`internal/repository/audit_event_repo.go`（新文件）
  - **验证**：仓储方法正常工作，查询性能符合目标（<2 秒）
  - **依赖**：T012（AuditEvent 模型）
  - **参考**：`audit-unification.md` 第一阶段

- [ ] **T021** [P] 查询索引设计和优化
  - 创建复合索引（参考 data-model.md 简化版）
  - 验证索引使用（EXPLAIN 查询计划）
  - 优化慢查询
  - **文件**：数据库迁移脚本
  - **验证**：T009 性能测试通过
  - **依赖**：T019、T020
  - **参考**：`research.md` 第 6.2 节

### API 处理器 [P]

- [ ] **T022** [P] Scheduler API 处理器实现
  - 实现 8 个端点处理器：
    - `ListTasks`、`GetTask`、`EnableTask`、`DisableTask`
    - `TriggerTask`（手动触发）、`ListExecutions`、`GetExecution`、`GetStats`
  - 实现请求验证和错误处理
  - **文件**：`internal/api/handlers/scheduler/tasks.go`、`executions.go`、`stats.go`（新文件）
  - **验证**：T004 契约测试通过
  - **依赖**：T013、T019
  - **参考**：`contracts/scheduler_api.yaml`

- [ ] **T023** [P] Audit API 处理器实现
  - 实现 4 个端点处理器：
    - `ListEvents`、`GetEvent`、`GetConfig`、`UpdateConfig`
  - 实现分页和过滤参数解析
  - **文件**：`internal/api/handlers/audit/events.go`、`config.go`（新文件）
  - **验证**：T005 契约测试通过
  - **依赖**：T017、T020
  - **参考**：`contracts/audit_api.yaml`

---

## 阶段 3.4：前端实现 [P]

- [ ] **T024** [P] Scheduler 管理页面
  - 创建任务列表页（显示所有任务、启用/禁用状态）
  - 实现任务启用/禁用操作
  - 实现任务手动触发
  - **文件**：`ui/src/pages/scheduler/task-list-page.tsx`、`ui/src/services/scheduler-service.ts`（新文件）
  - **验证**：页面可以显示和操作任务
  - **依赖**：T022

- [ ] **T025** [P] 任务执行历史页面
  - 创建执行历史列表页（分页、过滤）
  - 创建执行详情页（显示错误堆栈、执行时长）
  - 实现统计图表（成功率、执行时间趋势）
  - **文件**：`ui/src/pages/scheduler/execution-history-page.tsx`、`execution-detail-page.tsx`（新文件）
  - **验证**：页面可以查询和展示执行历史
  - **依赖**：T022

- [ ] **T026** [P] 审计日志页面
  - 创建审计日志列表页（分页、多维度过滤）
  - 创建审计事件详情页
  - 实现差异对比组件（OldObject vs NewObject）
  - **文件**：`ui/src/pages/audit/event-list-page.tsx`、`event-detail-page.tsx`、`ui/src/components/audit/diff-viewer.tsx`（新文件）
  - **验证**：页面可以查询和展示审计日志
  - **依赖**：T023

---

## 阶段 3.5：数据迁移和配置

- [ ] **T027** 现有任务迁移到增强 Scheduler ⚠️ 简化
  - 迁移 `alert_processing` 任务（添加执行历史记录）
  - 迁移 `database_audit_cleanup` 任务
  - 更新任务注册逻辑（使用 `AddCron()` 方法）
  - **文件**：`internal/services/scheduler/tasks/alert.go`、`cleanup.go`（修改现有文件）
  - **验证**：现有任务正常调度，执行历史正确记录
  - **依赖**：T013、T014

- [ ] **T028** [P] 配置文件更新 ⚠️ 简化
  - 添加 Scheduler 配置（超时时间、最大重试次数）
  - 添加 Audit 配置（保留期、对象大小限制）
  - **无需添加**：分布式锁类型配置
  - **文件**：`config.yaml`
  - **验证**：配置可以加载和生效
  - **依赖**：T013、T017

---

## 阶段 3.6：优化和文档

- [ ] **T029** [P] Swagger 文档生成
  - 为所有新 API 端点添加 Swagger 注释
  - 运行 `./scripts/generate-swagger.sh`
  - **文件**：处理器文件中的注释
  - **验证**：Swagger 文档正确显示新端点
  - **依赖**：T022、T023

- [ ] **T030** [P] 部署文档更新 ⚠️ 简化
  - 更新部署指南（超时配置、保留期配置）
  - 添加性能调优建议
  - 添加故障排查指南
  - **无需添加**：分布式锁配置、多实例部署
  - **文件**：`docs/deployment.md`
  - **验证**：文档清晰完整
  - **依赖**：T028

- [ ] **T031** 代码质量检查
  - 运行 `task lint` 并修复所有警告
  - 运行 `task gofmt` 格式化代码
  - 检查代码覆盖率（目标 80%）
  - **文件**：所有源文件
  - **验证**：`task lint` 通过，覆盖率达标
  - **依赖**：所有实现任务

- [ ] **T032** 手动验证（quickstart.md 场景）⚠️ 简化
  - 执行场景 1：单实例任务调度验证
  - 执行场景 2：任务执行历史查询
  - 执行场景 3：任务手动触发和失败处理
  - 执行场景 4：审计日志记录和查询
  - 执行场景 5：审计日志对象截断验证
  - 执行场景 6：审计配置管理
  - 执行场景 7：任务统计数据查询
  - **跳过场景**：分布式环境任务调度验证
  - **文件**：无（手动测试）
  - **验证**：所有场景通过
  - **依赖**：所有实现任务
  - **参考**：`quickstart.md`（跳过场景 1 的多实例部分）

---

## 阶段 3.7：可选增强任务（根据时间和需求决定）

- [ ] **T033** [可选] 强类型 Action/ResourceType 枚举
  - 创建 `Action` 枚举（created、updated、deleted 等）
  - 创建 `ResourceType` 枚举（cluster、database、user 等）
  - 实现 `Validate()` 方法
  - **文件**：`internal/models/audit_types.go`（新文件）
  - **理由**：现有字符串已工作良好，强类型可以提高类型安全但非必需
  - **参考**：`data-model.md` 实体 5 和 6

- [ ] **T034** [可选] Scheduler 优先级队列
  - 实现优先级调度（高优先级任务优先执行）
  - **文件**：`internal/services/scheduler/priority.go`（新文件）
  - **理由**：当前任务量不大，简单 FIFO 足够

- [ ] **T035** [可选] Scheduler Filter 机制
  - 实现 `Filter` 接口（资源标签匹配）
  - **文件**：`internal/services/scheduler/filter.go`（新文件）
  - **理由**：Gitness 特性，Tiga 可能不需要

---

## 阶段 3.8：审计系统统一（可选）🆕

**前提条件**：T012（AuditEvent 模型）、T020（AuditEventRepository）已完成

- [x] **T036** [可选] MinIO 改用统一审计
  - 修改 MinIO 服务使用 `models.AuditEvent` 替换 `models.MinIOAuditLog`
  - 将 MinIO 特定字段序列化到 `Data` map 字段：
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
  - 设置 `Subsystem = "minio"`
  - 使用统一 `audit.AsyncLogger[*models.AuditEvent]`
  - **文件**：`internal/services/minio/`（多个文件）
  - **验证**：MinIO 操作正确记录到 `audit_events` 表
  - **依赖**：T012、T020
  - **参考**：`audit-unification.md` 第三阶段

- [x] **T037** [可选] Database 改用统一审计
  - 修改 Database 服务使用 `models.AuditEvent` 替换 `models.DatabaseAuditLog`
  - 将 Database 特定字段序列化到 `Data` map 字段：
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
  - 设置 `Subsystem = "database"`
  - 使用统一 `audit.AsyncLogger[*models.AuditEvent]`
  - **文件**：`internal/services/database/`（多个文件）
  - **验证**：Database 操作正确记录到 `audit_events` 表
  - **依赖**：T012、T020
  - **参考**：`audit-unification.md` 第三阶段

---

## 依赖关系图

```
设置 (T001-T003)
  ↓
契约测试 [P] (T004-T005)
  ↓
集成测试框架 [P] (T006-T010)
  ↓
数据模型 [P] (T011-T012) 🆕 T012 改为统一 AuditEvent
  ↓
Scheduler 核心 (T013 → T014 → T015-T016)
  ↓
Audit 核心 (T018 → T017) 🆕 T017 依赖 T012
  ↓
Repository [P] (T019-T021) 🆕 T020 改为统一 AuditEventRepository
  ↓
API 处理器 [P] (T022-T023)
  ↓
前端 [P] (T024-T026)
  ↓
数据迁移和配置 (T027-T028)
  ↓
优化和文档 [P] (T029-T032)
  ↓
可选增强 [P] (T033-T035)
  ↓
审计系统统一（可选）[P] (T036-T037) 🆕 依赖 T012+T020
```

---

## 并行执行示例

### 示例 1：契约测试（阶段 3.2 开始）

```bash
# 在单个消息中启动两个契约测试任务
Task: "实现 Scheduler API 契约测试（tests/contract/scheduler_contract_test.go），测试 8 个端点，参考 contracts/scheduler_api.yaml"
Task: "实现 Audit API 契约测试（tests/contract/audit_contract_test.go），测试 4 个端点，参考 contracts/audit_api.yaml"
```

### 示例 2：集成测试框架（T006-T010）

```bash
# 在单个消息中启动 5 个集成测试框架任务
Task: "实现任务执行集成测试框架（tests/integration/scheduler/task_execution_test.go），测试调度、重试、超时控制"
Task: "实现并发调度集成测试（tests/integration/scheduler/concurrent_test.go），测试单实例内并发执行"
Task: "实现审计日志创建集成测试（tests/integration/audit/event_creation_test.go），测试事件记录和 IP 提取"
Task: "实现审计日志查询性能测试（tests/integration/audit/query_performance_test.go），验证 10000 条记录 <2 秒"
Task: "实现对象截断集成测试（tests/integration/audit/truncation_test.go），验证 64KB 限制和结构完整性"
```

### 示例 3：数据模型（T011-T012）

```bash
# 在单个消息中启动 2 个数据模型任务
Task: "创建 TaskExecution 模型（internal/models/task_execution.go），实现 ExecutionState 枚举和 Validate() 方法"
Task: "创建统一 AuditEvent 模型（internal/models/audit_event.go），实现 audit.AuditLog 接口，包含 Subsystem 和 Metadata 字段"
```

### 示例 4：Repository 层（T019-T021）

```bash
# 在单个消息中启动 3 个 Repository 任务
Task: "实现 TaskExecutionRepository（internal/repository/scheduler/execution.go），CRUD 和统计查询"
Task: "创建统一 AuditEventRepository（internal/repository/audit_event_repo.go），实现 audit.AuditRepository[*AuditEvent] 接口"
Task: "设计和优化查询索引（数据库迁移脚本），创建复合索引并验证性能"
```

### 示例 5：API 处理器（T022-T023）

```bash
# 在单个消息中启动 2 个 API 处理器任务
Task: "实现 Scheduler API 处理器（internal/api/handlers/scheduler/），包括 tasks.go、executions.go、stats.go，共 8 个端点"
Task: "实现 Audit API 处理器（internal/api/handlers/audit/），包括 events.go、config.go，共 4 个端点"
```

---

## 验证清单

**门禁：在标记项目完成之前检查**

- [x] 所有契约都有对应的测试（T004、T005）
- [x] 所有实体都有模型任务（T011、T012 统一模型）
- [x] 所有测试都在实现之前（阶段 3.2 在 3.3 之前）
- [x] 并行任务真正独立（标记 [P] 的任务操作不同文件）
- [x] 每个任务指定确切的文件路径
- [x] 没有任务修改与另一个 [P] 任务相同的文件
- [x] 所有集成测试场景都已覆盖（quickstart.md 简化版场景）
- [x] 性能目标明确且可测试（<2s 审计查询）
- [x] 基于现有代码增强，而非完全重写
- [x] 删除了所有分布式锁相关任务
- [x] 审计系统已统一（方案 A：单表 audit_events）

---

## 任务总数：37 个（审计统一版）

**对比原计划**（48 任务）：
- **删除**：10 个任务（分布式锁 5 个 + 其他 5 个）
- **简化**：18 个任务（标记 ⚠️ 简化）
- **统一**：3 个任务改为统一审计（T012、T017、T020，标记 🆕）
- **新增**：2 个可选任务（T036-T037：子系统迁移）
- **保留**：20 个任务（完全保留）
- **新增可选**：3 个任务（T033-T035：其他增强）

**任务分布**：
- **设置**：3 个（T001-T003）
- **测试优先**：7 个（T004-T010）
- **核心实现**：19 个（T011-T029）
  - 数据模型：2 个 [P]（T011 Scheduler，T012 统一 Audit 🆕）
  - Scheduler 核心：4 个（T013-T016）
  - Audit 核心：2 个（T017 🆕、T018）
  - Repository：3 个 [P]（T019 Scheduler，T020 统一 Audit 🆕，T021 索引）
  - API 处理器：2 个 [P]（T022-T023）
  - 前端：3 个 [P]（T024-T026）
  - 数据迁移和配置：2 个（T027-T028）
- **优化和文档**：4 个（T029-T032）
- **可选增强**：3 个（T033-T035）
- **审计统一（可选）**：2 个（T036-T037：MinIO 和 Database 迁移）🆕

**预计并行执行次数**：9 轮（标记 [P] 的任务组）
**预计顺序执行次数**：~12 轮（无 [P] 标记的任务组）
**预计总执行时间**：减少约 35%（相比原计划 48 任务，统一审计略增加 2 个可选任务）

---

## 注意事项

### TDD 强制要求
- **T004-T010 必须在 T011 之前完成**
- 验证所有测试失败后再开始实现
- 每个阶段完成后运行相应的测试验证进度

### 并行执行规则
- **[P] 标记**：只有操作不同文件且无依赖关系的任务才能并行
- **验证方法**：检查文件路径是否重叠

### 提交策略
- 每个任务完成后单独提交
- 提交信息格式：`[T001] 创建项目结构和分支`
- 契约测试失败时也要提交（TDD 要求）

### 简化原则
- **基于现有代码增强**，不完全重写
- **保留 Tiga 简洁性**，避免 Gitness 复杂性
- **单实例优先**，为多实例预留接口（可选）

### 故障恢复
- 如果某个任务失败，不要跳过，修复后继续
- 集成测试失败时，检查依赖任务是否真正完成
- 性能测试不达标时，运行性能分析工具（pprof）

---

**任务状态**：✅ 已生成（37 个任务，审计统一版）
**准备度**：可立即开始执行（从 T001 开始）
**下一步**：运行 `git checkout -b 006-gitness-tiga` 并开始 T001

---

**变更日志**：
- **v3.0**（2025-10-20）：审计系统统一，采用方案 A（单表 `audit_events`），从 35 任务增加到 37 任务
  - T012：改为创建统一 `models.AuditEvent` 模型
  - T020：改为创建统一 `AuditEventRepository`
  - T017：使用统一 `audit.AsyncLogger[*AuditEvent]`
  - 新增 T036-T037：MinIO 和 Database 迁移任务（可选）
- **v2.0**（2025-10-20）：基于 `audit-report.md` 简化，从 48 任务减少到 35 任务
- **v1.0**（2025-10-19）：初始版本，48 任务（已废弃）
