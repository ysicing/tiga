# 实施计划：定时任务和审计系统重构

**分支**：`006-gitness-tiga` | **日期**：2025-10-19 | **规格**：[spec.md](./spec.md)
**输入**：来自 `.claude/specs/006-gitness-tiga/spec.md` 的功能规格

## 执行流程（/spec-kit:plan 命令范围）
```
1. 从输入路径加载功能规格 ✅
   → 规格已加载，包含 5 个澄清决策
2. 填充技术上下文（扫描需要澄清的内容） ✅
   → Web 应用类型：Go 后端 + React 前端
   → 所有技术栈明确，无需澄清
3. 根据章程文档内容填充章程检查部分
   → 进行中
4. 评估下面的章程检查部分
   → 待执行
5. 执行阶段 0 → research.md
   → 待执行
6. 执行阶段 1 → contracts、data-model.md、quickstart.md
   → 待执行
7. 重新评估章程检查部分
   → 待执行
8. 规划阶段 2 → 描述任务生成方法
   → 待执行
9. 停止 - 准备执行 /spec-kit:tasks 命令
```

## 摘要

基于 Gitness 的设计模式，对 Tiga 的定时任务调度系统（Scheduler）和审计系统（Audit）进行完全重构，无需考虑向后兼容。

**Scheduler 重构目标**：
- 分布式锁机制（支持数据库/Redis/etcd，默认数据库）
- 任务执行历史和状态追踪
- 动态启用/禁用、手动触发、优先级调度
- 超时控制（Context 取消 + 宽限期强制终止）
- 并发控制和可观测性

**Audit 重构目标**：
- 强类型系统（Action、ResourceType 枚举）
- 差异对象追踪（OldObject vs NewObject，64KB 限制）
- Functional Options 模式配置
- 智能客户端 IP 提取（支持代理 header）
- 异步写入和批量优化

**技术方法**：参考 Gitness 的 `job` 包（Scheduler + Executor）和 `audit` 包设计，采用接口抽象、强类型验证、上下文传播等最佳实践。

## 技术上下文

**语言/版本**：Go 1.24+（后端）、TypeScript/React 19（前端）
**主要依赖**：
- 后端：Gin 框架、GORM、job.Scheduler + job.Executor（定时任务）、分布式锁抽象层（Database/Redis/etcd）
- 前端：TailwindCSS、Radix UI、React Query
- 参考：Gitness `job` 包、`app/services/cleanup`、`audit` 包

**存储**：
- 主存储：SQLite（默认）/ PostgreSQL / MySQL（可配置）
- 分布式锁：数据库行锁（默认）/ Redis / etcd（可配置切换）
- 审计日志：与主存储相同，90天保留期（可配置）

**测试**：
- 后端：Go testing、testify、testcontainers-go（集成测试）
- 前端：Vitest、React Testing Library
- 契约测试：验证 API 规范一致性

**目标平台**：Linux/macOS/Windows 服务器（Go 交叉编译）

**性能目标**：
- 分布式锁操作延迟：<100ms（99th percentile）
- 审计日志异步写入延迟：<1 秒（平均）
- 任务历史查询：<500ms（1000 条记录）
- 审计日志查询：<2 秒（10000 条记录，索引支持）
- 并发审计日志写入：≥1000 req/s
- 并发任务调度：≥100 任务

**约束**：
- 审计日志对象数据：≤64KB（智能截断超长字段值）
- 任务超时宽限期：30秒（默认，可配置）
- 日志保留期：90天（默认，可后台配置）
- 审计日志写入失败：丢弃并告警（不使用备用存储）

**规模/范围**：
- 支持 100+ 定时任务
- 支持 1000+ 并发审计日志写入
- 支持多实例分布式部署（通过分布式锁协调）

## 章程检查
*门禁：必须在阶段 0 研究之前通过。在阶段 1 设计后重新检查。*

### 原则 1：安全优先设计
✅ **符合**：
- 审计系统记录所有操作的完整上下文（用户、IP、时间戳、资源）
- 审计日志只能创建和查询，不支持修改和删除（防篡改）
- 使用强类型系统防止注入和数据不一致
- 正确提取客户端真实 IP（支持代理 header）

### 原则 2：生产就绪性
✅ **符合**：
- 分布式锁机制确保多实例环境任务不重复执行
- 任务执行历史和错误追踪支持问题排查
- 超时控制机制（Context 取消 + 宽限期）防止任务挂起
- 全面的边缘情况处理（panic 恢复、连接断开、系统重启）
- 契约测试和集成测试覆盖

### 原则 3：卓越用户体验
✅ **符合**：
- 任务执行统计数据（成功率、平均执行时间、失败次数）
- 审计日志高效查询（分页、过滤、按多维度检索）
- 动态启用/禁用任务，无需重启服务
- 手动触发任务支持调试和紧急情况
- 后台配置界面调整保留期，立即生效

### 原则 4：默认可观测性
✅ **符合**：
- 任务执行历史完整记录（开始/结束时间、状态、错误堆栈、执行实例）
- 审计日志异步写入失败时触发告警
- 任务执行统计数据支持监控和分析
- 详细的错误堆栈和上下文信息记录
- 支持查询任务执行历史（分页、过滤）

### 原则 5：开源承诺
✅ **符合**：
- 参考 Gitness（Apache 2.0）的设计模式
- 保持 Tiga 项目的 Apache 2.0 许可证
- 清晰的接口抽象和文档，便于社区贡献
- 不引入专有依赖

### 质量标准检查
✅ **符合**：
- 后端代码覆盖率目标：80%（单元测试 + 集成测试）
- 性能目标明确且可衡量
- 完整的契约测试验证 API 规范

## 项目结构

### 文档（此功能）
```
.claude/specs/006-gitness-tiga/
├── spec.md              # 功能规格（已完成，含澄清）
├── plan.md              # 此文件（/spec-kit:plan 命令输出）
├── research.md          # 阶段 0 输出（待生成）
├── data-model.md        # 阶段 1 输出（待生成）
├── quickstart.md        # 阶段 1 输出（待生成）
├── contracts/           # 阶段 1 输出（待生成）
│   ├── scheduler_api.yaml    # Scheduler API 契约
│   └── audit_api.yaml        # Audit API 契约
└── tasks.md             # 阶段 2 输出（/spec-kit:tasks 命令）
```

### 源代码（仓库根目录）
```
# Web 应用结构（前端 + 后端）

# 后端 Go 代码
internal/
├── services/
│   ├── scheduler/           # 新重构的 Scheduler
│   │   ├── scheduler.go     # 接口定义和主调度器
│   │   ├── queue.go         # 任务队列（参考 Gitness）
│   │   ├── worker.go        # Worker 和 Filter 机制
│   │   ├── lock/            # 分布式锁抽象
│   │   │   ├── interface.go # 锁接口
│   │   │   ├── db.go        # 数据库锁实现
│   │   │   ├── redis.go     # Redis 锁实现
│   │   │   └── etcd.go      # etcd 锁实现
│   │   └── tasks/           # 任务实现
│   │       ├── alert.go
│   │       └── cleanup.go
│   └── audit/               # 新重构的 Audit
│       ├── audit.go         # 核心审计服务
│       ├── interface.go     # 服务接口
│       ├── types.go         # 强类型（Action、ResourceType）
│       ├── event.go         # 事件结构和验证
│       ├── options.go       # Functional Options
│       ├── middleware.go    # HTTP 中间件
│       └── context.go       # 上下文管理
├── models/
│   ├── scheduled_task.go    # 定时任务配置
│   ├── task_execution.go    # 任务执行历史
│   ├── task_lock.go         # 分布式锁记录
│   └── audit_event.go       # 审计事件（替换 audit_log.go）
├── repository/
│   ├── scheduler/           # Scheduler 仓储
│   │   ├── task.go
│   │   ├── execution.go
│   │   └── lock.go
│   └── audit/               # Audit 仓储
│       └── event.go
└── api/
    ├── handlers/
    │   ├── scheduler/       # Scheduler API 处理器
    │   │   ├── tasks.go
    │   │   ├── executions.go
    │   │   └── manual_trigger.go
    │   └── audit/           # Audit API 处理器
    │       └── events.go
    └── middleware/
        └── audit.go         # 审计中间件（重构）

# 前端 React 代码
ui/src/
├── pages/
│   ├── scheduler/           # Scheduler 管理页面
│   │   ├── task-list-page.tsx
│   │   ├── execution-history-page.tsx
│   │   └── task-detail-page.tsx
│   └── audit/               # Audit 日志页面
│       ├── event-list-page.tsx
│       └── event-detail-page.tsx
├── components/
│   ├── scheduler/
│   │   ├── task-card.tsx
│   │   ├── execution-timeline.tsx
│   │   └── task-stats.tsx
│   └── audit/
│       ├── event-table.tsx
│       ├── diff-viewer.tsx  # OldObject vs NewObject 对比
│       └── event-filter.tsx
└── services/
    ├── scheduler-service.ts
    └── audit-service.ts

# 测试代码
tests/
├── contract/                # 契约测试
│   ├── scheduler_contract_test.go
│   └── audit_contract_test.go
├── integration/             # 集成测试
│   ├── scheduler/
│   │   ├── distributed_lock_test.go
│   │   ├── task_execution_test.go
│   │   └── concurrent_test.go
│   └── audit/
│       ├── event_creation_test.go
│       ├── query_performance_test.go
│       └── truncation_test.go
└── unit/                    # 单元测试
    ├── scheduler/
    │   ├── queue_test.go
    │   ├── worker_test.go
    │   └── lock/
    │       ├── db_test.go
    │       ├── redis_test.go
    │       └── etcd_test.go
    └── audit/
        ├── types_test.go
        ├── validation_test.go
        ├── options_test.go
        └── truncation_test.go
```

**结构决策**：
- Web 应用结构（选项 2）：Go 后端 + React 前端
- 后端遵循 Tiga 现有结构：`internal/services`、`internal/models`、`internal/repository`、`internal/api`
- 前端遵循 Tiga 现有结构：`ui/src/pages`、`ui/src/components`、`ui/src/services`
- 完全替换现有 `internal/services/scheduler` 和 `internal/api/middleware/audit.go`
- 新增分布式锁抽象层 `internal/services/scheduler/lock`
- 新增强类型审计系统 `internal/services/audit`

## 阶段 0：概述与研究
*状态：待执行*

### 研究任务清单

**1. Gitness Job Scheduler 设计研究**：
- 分析 Gitness `job/scheduler.go`、`job/executor.go`、`job/types.go`
- 理解 Scheduler + Executor 架构（注册、调度、执行分离）
- 研究 Job 状态管理（Scheduled → Running → Finished/Failed）
- 研究分布式锁接口 `lock.MutexManager`
- 提取 cron 定时任务调度模式

**2. Gitness Cleanup Service 研究**：
- 分析 `app/services/cleanup/service.go`
- 理解任务注册模式（Register + Handler）
- 研究 cron 表达式配置（如 "21 */4 * * *"）
- 研究任务执行历史和结果记录

**3. Gitness Audit 设计研究**：
- 分析 Gitness `audit/audit.go`、`interface.go`、`middleware.go`
- 理解强类型系统（Action、ResourceType、Resource）
- 研究 Functional Options 模式实现
- 研究 Event 验证机制和 DiffObject 设计

**3. 分布式锁实现研究**：
- 数据库行锁：PostgreSQL `SELECT ... FOR UPDATE`、MySQL `GET_LOCK()`
- Redis 锁：Redlock 算法、过期时间策略
- etcd 锁：Lease 机制、TTL 配置
- 锁接口抽象设计（参考 Gitness `lock.MutexManager`）

**4. 审计日志对象截断策略研究**：
- JSON 序列化大小估算
- 智能截断算法（保留结构、截断字段值）
- 截断标识存储方案
- 截断字段列表记录

**5. 任务执行超时机制研究**：
- Context 取消传播
- Goroutine 强制终止方法（panic recovery）
- 宽限期实现策略
- 资源清理最佳实践

**6. 性能优化研究**：
- 批量审计日志写入策略
- 查询索引设计（任务执行历史、审计日志）
- 连接池配置（数据库、Redis）
- 异步写入队列设计

**输出**：research.md，包含所有研究发现和技术决策

## 阶段 1：设计与契约
*前提条件：research.md 完成*
*状态：待执行*

### 1. 数据模型设计 → `data-model.md`

从功能规格提取的实体：

**Scheduler 实体**：
- ScheduledTask（定时任务配置）
- TaskExecution（任务执行历史）
- TaskLock（分布式锁记录）

**Audit 实体**：
- AuditEvent（审计事件）
- Action（操作类型枚举）
- ResourceType（资源类型枚举）
- Resource（资源定义）
- Principal（操作主体）

详细字段、关系、验证规则和状态转换将在 `data-model.md` 中定义。

### 2. API 契约生成 → `/contracts/`

**Scheduler API 契约**（`contracts/scheduler_api.yaml`）：
- `GET /api/v1/scheduler/tasks` - 任务列表
- `GET /api/v1/scheduler/tasks/{id}` - 任务详情
- `POST /api/v1/scheduler/tasks/{id}/enable` - 启用任务
- `POST /api/v1/scheduler/tasks/{id}/disable` - 禁用任务
- `POST /api/v1/scheduler/tasks/{id}/trigger` - 手动触发
- `GET /api/v1/scheduler/executions` - 执行历史（分页、过滤）
- `GET /api/v1/scheduler/executions/{id}` - 执行详情
- `GET /api/v1/scheduler/stats` - 统计数据

**Audit API 契约**（`contracts/audit_api.yaml`）：
- `GET /api/v1/audit/events` - 审计日志列表（分页、过滤）
- `GET /api/v1/audit/events/{id}` - 审计事件详情
- `GET /api/v1/audit/config` - 审计配置（保留期等）
- `PUT /api/v1/audit/config` - 更新审计配置

所有契约使用 OpenAPI 3.0 规范，包含完整的请求/响应模式、错误码、示例。

### 3. 契约测试生成

每个 API 端点一个测试文件：
- `tests/contract/scheduler_tasks_test.go`
- `tests/contract/scheduler_executions_test.go`
- `tests/contract/audit_events_test.go`

测试必须验证：
- 请求/响应 schema 匹配契约
- HTTP 状态码符合规范
- 错误响应格式一致

### 4. 快速启动场景 → `quickstart.md`

从用户故事提取测试场景：
- **场景 1**：分布式环境任务调度验证
- **场景 2**：任务执行历史查询
- **场景 3**：任务手动触发和失败处理
- **场景 4**：审计日志记录和查询
- **场景 5**：审计日志对象截断验证

每个场景包含：
- 前置条件
- 操作步骤（API 调用或 UI 操作）
- 预期结果
- 验证方法

### 5. 更新 CLAUDE.md

运行 `.specify/scripts/bash/update-agent-context.sh claude` 更新项目上下文（仅添加新技术，保留手动添加内容）。

**输出**：data-model.md、contracts/*、契约测试、quickstart.md、CLAUDE.md

## 阶段 2：任务规划方法
*本节描述 /spec-kit:tasks 命令将执行的操作 - 不要在 /spec-kit:plan 期间执行*

### ⚠️ 审计简化结果

**审计文档**：`.claude/specs/006-gitness-tiga/audit-report.md`

**关键发现**：
- Tiga 是单实例应用，无需分布式锁
- 现有 Scheduler 和 Audit 已有良好基础
- 任务总数从 48 个减少到 **35 个**（减少 27%）

### 任务生成策略（简化版）

**基础**：加载 `~/.claude/templates/specify/tasks-template.md` + 审计报告

**从设计文档生成任务**：

1. **设置任务**：
   - T001：创建项目结构和分支
   - T002：安装依赖和工具
   - T003：配置代码检查工具

2. **测试优先（TDD）**[P]：
   - T004：Scheduler API 契约测试（8 个端点）⚠️ 简化
   - T005：Audit API 契约测试（4 个端点）
   - T006：任务执行集成测试框架
   - T007：并发调度集成测试 ⚠️ 简化
   - T008：审计日志创建集成测试
   - T009：审计日志查询性能测试
   - T010：对象截断集成测试

3. **数据模型**[P]：
   - T011：TaskExecution 模型创建
   - T012：AuditLog 模型扩展 ⚠️ 简化（扩展现有模型，非新建）
   - ❌ **已删除**：TaskLock 模型（无需分布式锁）

4. **❌ 分布式锁实现任务**（全部删除）：
   - 原 T017-T021（5 个任务）已删除
   - 理由：Tiga 是单实例应用

5. **Scheduler 核心任务**：
   - T013：Scheduler 功能增强 ⚠️ 简化（增强现有代码）
   - T014：任务执行历史记录
   - T015：超时控制机制（Context + 宽限期）
   - T016：任务统计数据计算

6. **Audit 核心任务**：
   - T017：Audit 中间件增强 ⚠️ 简化（增强现有中间件）
   - T018：对象截断策略实现（64KB 限制）
   - ❌ **已删除**：复杂异步队列（已有 Goroutine）

7. **Repository 任务**[P]：
   - T019：Scheduler 仓储实现 ⚠️ 简化（仅 TaskExecution）
   - T020：Audit 仓储扩展 ⚠️ 简化（扩展现有仓储）
   - T021：查询索引设计和优化

8. **API 处理器任务**[P]：
   - T022：Scheduler API 处理器实现（8 个端点）
   - T023：Audit API 处理器实现（4 个端点）

9. **前端任务**[P]：
   - T024：Scheduler 管理页面
   - T025：任务执行历史页面
   - T026：审计日志页面

10. **数据迁移和配置**：
    - T027：现有任务迁移到增强 Scheduler ⚠️ 简化
    - T028：配置文件更新 ⚠️ 简化（无需锁配置）

11. **优化和文档**[P]：
    - T029：Swagger 文档生成
    - T030：部署文档更新 ⚠️ 简化（无需多实例部署）
    - T031：代码质量检查
    - T032：手动验证（quickstart.md 场景）⚠️ 简化

12. **可选增强任务**（根据时间决定）：
    - T033：强类型 Action/ResourceType 枚举（可选）
    - T034：Scheduler 优先级队列（可选）
    - T035：Scheduler Filter 机制（可选）

### 排序策略

**TDD 顺序**：测试先于实现
- 契约测试 (T004-T005) → 先执行
- 集成测试框架 (T006-T010) → 并行执行
- 数据模型 (T011-T012) → 测试后
- 实现任务 (T013-T023) → 模型后

**依赖顺序**：
- 模型 (T011-T012) → 仓储 (T019-T021) → 服务 (T013-T018) → API (T022-T023)
- 后端完成 → 前端 (T024-T026)
- 实现完成 → 迁移和配置 (T027-T028)

**并行执行标记 [P]**：
- 契约测试、集成测试、数据模型、Repository、API 处理器、前端、文档任务可并行执行

**预计输出**：tasks.md 中的 **35 个编号、有序的任务**（简化版）

**重要**：此阶段由 /spec-kit:tasks 命令执行，而不是由 /spec-kit:plan 执行

**简化原则**：
- 基于现有代码增强，而非完全重写
- 保留 Tiga 简洁性，避免 Gitness 复杂性
- 单实例优先，为多实例预留接口（可选）

## 阶段 3+：未来实施
*这些阶段超出了 /spec-kit:plan 命令的范围*

**阶段 3**：任务执行（/spec-kit:tasks 命令创建 tasks.md）

**阶段 4**：实施（按照章程原则执行 tasks.md）
- 遵循 TDD 开发流程
- 代码覆盖率 ≥80%
- 所有契约测试通过
- 性能基准达标

**阶段 5**：验证（运行测试、执行 quickstart.md、性能验证）
- 单元测试全部通过
- 集成测试全部通过
- quickstart.md 场景验证通过
- 性能目标达成验证

## 复杂性跟踪
*仅在章程检查有必须证明合理的违规时填写*

| 违规 | 为什么需要 | 拒绝更简单替代方案的原因 |
|-----------|------------|-------------------------------------|
| 无 | - | - |

**说明**：本重构项目不引入违反章程原则的复杂性。所有设计决策符合安全优先、生产就绪、用户体验、可观测性和开源承诺原则。

## 进度跟踪
*此清单在执行流程期间更新*

**阶段状态**：
- [x] 阶段 0：研究完成（/spec-kit:plan 命令）
- [x] 阶段 1：设计完成（/spec-kit:plan 命令）
- [x] 阶段 2：任务规划完成（/spec-kit:plan 命令 - 仅描述方法）
- [x] 阶段 3：任务已生成（/spec-kit:tasks 命令）✅ **已简化**
- [x] 阶段 3.1：审计报告完成（audit-report.md）✅ **新增**
- [x] 阶段 3.2：任务列表简化完成（tasks.md v2.0）✅ **新增**
- [ ] 阶段 4：实施完成
- [ ] 阶段 5：验证通过

**门禁状态**：
- [x] 初始章程检查：通过（无违规）
- [x] 设计后章程检查：通过
- [x] 所有需要澄清的内容已解决（5 个澄清决策）
- [x] 复杂性偏差已记录（无偏差）
- [x] 审计报告完成：已识别过度工程化任务 ✅ **新增**
- [x] 任务简化完成：从 48 任务减少到 35 任务 ✅ **新增**

---
*基于章程 v1.0.0 - 参见 `.claude/memory/constitution.md`*
