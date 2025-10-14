# 实施计划：数据库管理系统

**分支**：`003-nosql-sql` | **日期**：2025-10-10 | **规格**：[spec.md](./spec.md)
**输入**：来自 `.claude/specs/003-nosql-sql/spec.md` 的功能规格

## 执行流程（/spec-kit:plan 命令范围）
```
1. 从输入路径加载功能规格 ✅
   → 已加载规格,包含43个功能需求
2. 填充技术上下文（扫描需要澄清的内容） ✅
   → 项目类型:Web应用(前端React+后端Go)
   → 已检测到现有技术栈
3. 根据章程文档内容填充章程检查部分 ✅
   → 5条章程原则全部检查完成
4. 评估下面的章程检查部分 ✅
   → 初始章程检查：全部通过，无违规项
   → 更新进度跟踪：初始章程检查通过
5. 执行阶段 0 → research.md ✅
   → 生成research.md,包含5个研究任务的技术决策
   → 所有技术未知项已解决
6. 执行阶段 1 → contracts、data-model.md、quickstart.md ✅
   → 生成data-model.md（6个实体完整定义）
   → 生成contracts/database-api.yaml（OpenAPI 3.0规范）
   → 生成quickstart.md（Docker测试环境和API示例）
7. 重新评估章程检查部分 ✅
   → 设计后章程检查：无新增违规
   → 更新进度跟踪：设计后章程检查通过
8. 规划阶段 2 → 描述任务生成方法（不要创建 tasks.md） ✅
   → 任务生成策略已定义（47个预计任务）
   → 排序策略和并行标记已规划
9. 停止 - 准备执行 /spec-kit:tasks 命令 ✅
```

**重要**：/spec-kit:plan 命令在步骤 8 停止。阶段 2-4 由其他命令执行：
- 阶段 2：/spec-kit:tasks 命令创建 tasks.md
- 阶段 3-4：实施执行（手动或通过工具）

## 摘要

本功能实现了统一的数据库管理系统,支持MySQL、PostgreSQL和Redis三种数据库类型。核心能力包括:
- **实例管理**: 连接多个数据库实例,监控连接状态和元信息
- **数据库CRUD**: 列出、创建、删除数据库(MySQL/PostgreSQL),浏览Redis DB
- **用户与权限**: 创建数据库用户,授予数据库级权限(只读/管理两种角色)
- **查询执行**: 提供SQL控制台(MySQL/PostgreSQL)和Redis命令行,支持结果导出
- **安全审计**: 完全禁止DDL操作,拦截危险DML,全量审计日志保留90天
- **响应限制**: 单次查询响应不超过10MB,超时30秒自动中断

技术方法基于现有Tiga平台架构,复用认证(JWT+RBAC)和审计系统,新增数据库驱动集成层。

## 技术上下文

**语言/版本**:
- 后端: Go 1.24+
- 前端: TypeScript 5.x + React 19

**主要依赖**:
- 后端: Gin框架, GORM, database/sql驱动(github.com/go-sql-driver/mysql, github.com/lib/pq, github.com/redis/go-redis/v9)
- 前端: Vite, TailwindCSS, Radix UI, TanStack Query

**存储**:
- 应用数据库: SQLite/PostgreSQL/MySQL(已有)
- 管理目标: MySQL 5.7+/8.0+, PostgreSQL 12+, Redis 6.0+(ACL支持)

**测试**:
- 后端: Go testing, testcontainers-go(集成测试)
- 前端: Vitest, React Testing Library

**目标平台**:
- 后端: Linux/macOS服务器
- 前端: 现代浏览器(Chrome 90+, Firefox 88+, Safari 14+)

**项目类型**: Web应用(已确定)

**性能目标**:
- 查询响应: <10MB数据传输,30秒超时
- 数据库连接: 最多支持50个并发实例连接
- 审计日志查询: <2秒响应时间(90天数据)

**约束**:
- 安全: 完全禁止DDL(DROP/TRUNCATE/ALTER),拦截无WHERE的UPDATE/DELETE
- 权限: 数据库级粒度,仅只读/管理两种角色
- 审计: 保留90天,超期自动清理
- 凭据: AES-256加密存储,使用现有crypto包

**规模/范围**:
- 数据库实例: 最多100个
- 用户数: 最多1000个数据库用户(跨所有实例)
- 审计日志: 预计每天1000条记录,90天约9万条

## 章程检查
*门禁：必须在阶段 0 研究之前通过。在阶段 1 设计后重新检查。*

### 原则1：安全优先设计 ✅
- [x] 所有API端点实现认证和授权(复用现有JWT+RBAC中间件)
- [x] 敏感数据(数据库凭据)使用AES-256加密存储,绝不记录或UI暴露明文
- [x] SQL注入防护:使用参数化查询,输入验证
- [x] 危险操作拦截:DDL完全禁止,DML语法检查

### 原则2：生产就绪性 ✅
- [x] 错误处理:数据库连接失败重试机制,优雅降级
- [x] 测试覆盖:后端≥70%(单元测试+集成测试),前端组件测试
- [x] 向后兼容:新增API不影响现有功能,使用独立路由前缀 `/api/v1/database/*`

### 原则3：卓越用户体验 ✅
- [x] 响应式UI:适配桌面/平板,SQL控制台支持语法高亮
- [x] 清晰反馈:查询执行进度显示,错误消息本地化(中英文)
- [x] 快速加载:查询结果分页,虚拟滚动大结果集

### 原则4：默认可观测性 ✅
- [x] 审计日志:所有数据库操作全量记录(操作者、时间、类型、结果)
- [x] 实例监控:连接状态实时监控,连接失败告警集成
- [x] 性能指标:查询执行时间跟踪,慢查询识别(>5秒)

### 原则5：开源承诺 ✅
- [x] API文档:Swagger注解完整,契约测试覆盖
- [x] 架构决策:研究文档记录数据库驱动选择和安全策略

**章程合规状态**: 全部通过,无违规项

## 项目结构

### 文档（此功能）
```
.claude/specs/003-nosql-sql/
├── spec.md              # 功能规格(已完成澄清)
├── plan.md              # 此文件（/spec-kit:plan 命令输出）
├── research.md          # 阶段 0 输出（待生成）
├── data-model.md        # 阶段 1 输出（待生成）
├── quickstart.md        # 阶段 1 输出（待生成）
├── contracts/           # 阶段 1 输出（待生成）
│   ├── instance-api.yaml
│   ├── database-api.yaml
│   ├── user-api.yaml
│   ├── query-api.yaml
│   └── audit-api.yaml
└── tasks.md             # 阶段 2 输出（/spec-kit:tasks 命令）
```

### 源代码（仓库根目录）
```
# Web应用结构(后端+前端)
internal/
├── api/
│   └── handlers/
│       └── database/           # 新增:数据库管理处理器
│           ├── instance.go     # 实例管理
│           ├── dbops.go        # 数据库操作
│           ├── user.go         # 用户管理
│           ├── permission.go   # 权限管理
│           ├── query.go        # 查询执行
│           └── audit.go        # 审计日志
├── models/
│   ├── db_instance.go          # 新增:数据库实例模型
│   ├── db_database.go          # 新增:数据库模型
│   ├── db_user.go              # 新增:数据库用户模型
│   ├── db_permission.go        # 新增:权限策略模型
│   ├── db_query_session.go     # 新增:查询会话模型
│   └── db_audit_log.go         # 新增:审计日志模型
├── repository/
│   └── database/               # 新增:数据库仓储层
│       ├── instance.go
│       ├── database.go
│       ├── user.go
│       ├── permission.go
│       └── audit.go
└── services/
    └── database/               # 新增:数据库业务逻辑
        ├── manager.go          # 数据库管理器接口
        ├── mysql_manager.go    # MySQL管理器实现
        ├── postgres_manager.go # PostgreSQL管理器实现
        ├── redis_manager.go    # Redis管理器实现
        ├── query_executor.go   # 查询执行引擎
        ├── security_filter.go  # SQL安全过滤器
        └── audit_logger.go     # 审计日志记录器

pkg/
└── dbdriver/                   # 新增:数据库驱动封装
    ├── driver.go               # 驱动接口
    ├── mysql.go
    ├── postgres.go
    └── redis.go

ui/src/
├── pages/
│   └── database/               # 新增:数据库管理页面
│       ├── InstanceList.tsx
│       ├── DatabaseList.tsx
│       ├── UserManagement.tsx
│       ├── PermissionManagement.tsx
│       ├── QueryConsole.tsx
│       └── AuditLog.tsx
├── components/
│   └── database/               # 新增:数据库相关组件
│       ├── InstanceCard.tsx
│       ├── DatabaseTable.tsx
│       ├── UserForm.tsx
│       ├── PermissionSelector.tsx
│       ├── SQLEditor.tsx       # Monaco Editor集成
│       └── QueryResultTable.tsx
├── services/
│   └── database.ts             # 新增:数据库API客户端
└── types/
    └── database.ts             # 新增:数据库类型定义

tests/
├── integration/
│   └── database/               # 新增:数据库集成测试
│       ├── instance_test.go
│       ├── query_test.go
│       └── security_test.go
└── contract/                   # 契约测试(阶段1生成)
```

**结构决策**:
- 采用Web应用结构(后端Go + 前端React)
- 后端新增 `internal/api/handlers/database`、`internal/services/database`、`pkg/dbdriver` 模块
- 前端新增 `ui/src/pages/database`、`ui/src/components/database` 子系统
- 复用现有认证(JWT)、RBAC、审计框架,避免重复造轮子
- 数据库驱动层 `pkg/dbdriver` 提供统一接口,支持多数据库类型扩展

## 阶段 0：概述与研究

**待研究项**（从技术上下文提取的未知或需深入了解的项）:
1. 数据库驱动选择和最佳实践
2. SQL安全过滤器实现方案
3. Redis ACL权限映射策略
4. 查询超时和大结果集处理
5. 审计日志自动清理机制

### 研究任务

#### Task 0.1: 数据库驱动选择
**研究内容**: Go语言的MySQL/PostgreSQL/Redis官方驱动及连接池最佳实践
**重点问题**:
- `database/sql` vs ORM(GORM) 在数据库管理场景的适用性
- 连接池配置(MaxOpenConns, MaxIdleConns, ConnMaxLifetime)
- 驱动特性对比(参数化查询、超时控制、错误处理)

#### Task 0.2: SQL安全过滤器
**研究内容**: SQL语法解析和DDL/DML识别方案
**重点问题**:
- 使用SQL parser库(如 vitess/sqlparser) vs 正则匹配
- 如何识别无WHERE条件的UPDATE/DELETE
- 如何处理批量语句和存储过程调用
- 性能影响评估(<10ms额外延迟)

#### Task 0.3: Redis ACL映射
**研究内容**: Redis 6.0+ ACL系统与Tiga权限模型映射
**重点问题**:
- 只读角色对应的Redis命令集(GET、KEYS、SCAN等)
- 管理角色对应的命令集(排除FLUSHDB、FLUSHALL、SHUTDOWN等)
- ACL规则生成和应用策略
- 用户创建后的ACL同步机制

#### Task 0.4: 查询超时和流式响应
**研究内容**: Go context超时控制和大结果集分块传输
**重点问题**:
- 使用 `context.WithTimeout` 实现30秒超时
- 10MB响应大小限制实现(计数器累加或预估行数)
- 流式JSON响应 vs 分页API
- 前端虚拟滚动库选择(react-window vs react-virtualized)

#### Task 0.5: 审计日志清理
**研究内容**: 定时任务调度和数据归档策略
**重点问题**:
- 使用现有 `services.scheduler.Scheduler` vs 独立cron
- 清理策略(软删除标记 vs 物理删除)
- 大批量删除的性能优化(批次删除,避免锁表)
- 可选的日志导出到外部系统(如Elasticsearch)

**输出**: `research.md`（包含所有研究发现和技术决策）

## 阶段 1：设计与契约
*前提条件：research.md 完成*

### 数据模型设计
基于功能规格中的关键实体生成 `data-model.md`:

**实体列表**:
1. DatabaseInstance (数据库实例)
2. Database (数据库)
3. DatabaseUser (数据库用户)
4. PermissionPolicy (权限策略)
5. QuerySession (查询会话)
6. DatabaseAuditLog (数据库审计日志)

### API契约设计
基于功能需求生成OpenAPI规范到 `/contracts/`:

**契约文件**:
- `instance-api.yaml`: 实例管理API (FR-001至FR-004)
- `database-api.yaml`: 数据库CRUD API (FR-005至FR-012)
- `user-api.yaml`: 用户管理API (FR-013至FR-018)
- `permission-api.yaml`: 权限管理API (FR-019至FR-023)
- `query-api.yaml`: 查询执行API (FR-025至FR-033)
- `audit-api.yaml`: 审计日志API (FR-036至FR-038)

### 契约测试生成
每个契约生成对应的契约测试文件:
- `tests/contract/instance_contract_test.go`
- `tests/contract/database_contract_test.go`
- `tests/contract/user_contract_test.go`
- `tests/contract/permission_contract_test.go`
- `tests/contract/query_contract_test.go`
- `tests/contract/audit_contract_test.go`

**测试策略**: 使用HTTP请求断言模式匹配,初始状态为FAIL(无实现)

### 集成测试场景提取
从用户故事生成集成测试场景:
- 场景1: MySQL实例连接和数据库列表查询 (验收场景1)
- 场景2: PostgreSQL用户创建和只读权限授予 (验收场景2)
- 场景3: 危险SQL拦截测试 (验收场景3)
- 场景4: Redis用户和ACL权限映射 (验收场景6)
- 场景5: 超大结果集截断测试 (验收场景9)

### 快速启动文档
生成 `quickstart.md`,包含:
- 环境准备(测试数据库实例启动,使用Docker Compose)
- 首次配置(添加测试实例,创建测试用户)
- 核心流程演示(执行查询,查看审计日志)
- 验证步骤(契约测试通过,集成测试通过)

### CLAUDE.md增量更新
运行 `.claude/scripts/specify/bash/update-agent-context.sh` 更新项目上下文:
- 新增技术决策记录(数据库驱动选择、安全过滤器方案)
- 更新最近变更(数据库管理子系统)
- 添加数据库管理相关的开发指南

**输出**: `data-model.md`、`/contracts/*.yaml`、失败的契约测试、`quickstart.md`、更新的`CLAUDE.md`

## 阶段 2：任务规划方法
*本节描述 /spec-kit:tasks 命令将执行的操作 - 不要在 /spec-kit:plan 期间执行*

### 任务生成策略
- **基础**: 加载 `~/.claude/templates/specify/tasks-template.md`
- **数据模型任务**: 每个实体生成模型创建任务 (6个任务) [P]
- **契约任务**: 每个契约生成实现任务和测试任务 (12个任务)
- **服务层任务**: 数据库管理器、查询执行器、安全过滤器 (8个任务)
- **API层任务**: 处理器实现、路由注册 (6个任务)
- **前端任务**: 页面组件、API集成 (10个任务)
- **集成测试任务**: 5个验收场景测试 (5个任务)

### 排序策略
1. **TDD顺序**: 契约测试 → 模型 → 仓储 → 服务 → 处理器
2. **依赖顺序**:
   - 安全过滤器优先(其他服务依赖)
   - 数据库管理器(服务层核心)
   - 查询执行器(依赖管理器和过滤器)
3. **并行标记 [P]**:
   - 独立模型文件可并行创建
   - 不同数据库类型的管理器可并行实现
   - 前端组件可并行开发

### 预计任务数
- 模型层: 6个任务
- 仓储层: 5个任务
- 服务层: 8个任务
- API层: 6个任务
- 前端: 10个任务
- 测试: 12个任务(契约6+集成6)
- **总计**: 约47个任务

**重要**: 此阶段由 `/spec-kit:tasks` 命令执行,而不是由 `/spec-kit:plan` 执行

## 阶段 3+：未来实施
*这些阶段超出了 /spec-kit:plan 命令的范围*

**阶段 3**：任务执行（/spec-kit:tasks 命令创建 tasks.md）
**阶段 4**：实施（按照章程原则执行 tasks.md）
**阶段 5**：验证（运行测试、执行 quickstart.md、性能验证）

## 复杂性跟踪
*仅在章程检查有必须证明合理的违规时填写*

无违规项,无需记录复杂性偏差。

## 进度跟踪
*此清单在执行流程期间更新*

**阶段状态**：
- [x] 阶段 0：研究完成（/spec-kit:plan 命令） ✅
- [x] 阶段 1：设计完成（/spec-kit:plan 命令） ✅
- [x] 阶段 2：任务规划完成（/spec-kit:plan 命令 - 仅描述方法） ✅
- [ ] 阶段 3：任务已生成（/spec-kit:tasks 命令）
- [ ] 阶段 4：实施完成
- [ ] 阶段 5：验证通过

**门禁状态**：
- [x] 初始章程检查：通过（无违规）
- [x] 设计后章程检查：通过（无新增违规）
- [x] 所有需要澄清的内容已解决（剩余2个低影响项推迟）
- [x] 复杂性偏差已记录（无偏差）

---
*基于章程 v1.0.0 - 参见 `.claude/memory/constitution.md`*

---

## 完成报告

**执行时间**: 2025-10-10
**命令**: `/spec-kit:plan`
**状态**: ✅ 成功完成

### 已交付成果

**阶段 0 - 研究与技术决策**:
- ✅ `research.md` - 5个研究任务,所有技术决策已完成
  - 数据库驱动选择: database/sql + 官方驱动
  - SQL安全过滤: xwb1989/sqlparser + 规则引擎
  - Redis ACL映射: @read/@write命令类别
  - 查询超时: context.WithTimeout(30s) + 10MB限制
  - 审计清理: Scheduler批次删除(90天保留)

**阶段 1 - 设计与契约**:
- ✅ `data-model.md` - 6个核心实体完整定义
  - DatabaseInstance, Database, DatabaseUser, PermissionPolicy, QuerySession, DatabaseAuditLog
  - 包含字段定义、验证规则、索引策略、关系图
- ✅ `contracts/database-api.yaml` - OpenAPI 3.0 完整规范
  - 6个API组: instances, databases, users, permissions, query, audit
  - 25个端点,包含请求/响应schema和安全定义
- ✅ `quickstart.md` - 快速启动和验证指南
  - Docker Compose测试环境(MySQL 8.0, PostgreSQL 15, Redis 7)
  - API使用示例(curl命令)
  - 4个核心场景演示
  - 验证步骤(契约测试、集成测试、前端检查)

**章程合规**:
- ✅ 初始检查: 5条原则全部通过,无违规
- ✅ 设计后检查: 无新增违规
- ✅ 安全优先: AES-256加密、DDL禁止、审计日志
- ✅ 生产就绪: 错误处理、测试覆盖≥70%、向后兼容
- ✅ 用户体验: 响应式UI、语法高亮、虚拟滚动
- ✅ 可观测性: 全量审计、实例监控、慢查询识别
- ✅ 开源承诺: Swagger文档、架构决策记录

### 关键技术决策

| 决策领域 | 选择方案 | 理由 |
|---------|---------|------|
| 数据库驱动 | database/sql + 官方驱动 | 标准接口、连接池、参数化查询防注入 |
| SQL安全 | xwb1989/sqlparser | AST解析准确识别DDL/DML,性能0.5-2ms |
| Redis权限 | ACL @read/@write类别 | 原生ACL功能,避免自定义过滤复杂度 |
| 查询限制 | 30s超时 + 10MB响应 | context超时控制,字节计数截断 |
| 前端虚拟滚动 | react-window | 轻量(18KB),适合大结果集展示 |
| 审计清理 | Scheduler批次删除 | 复用现有调度器,1000条/批避免锁表 |

### 下一步行动

**立即执行**:
```bash
/spec-kit:tasks  # 生成tasks.md,包含约47个实施任务
```

**任务生成后**:
1. 审查tasks.md中的任务排序和依赖关系
2. 确认并行任务标记[P]正确
3. 开始TDD实施流程(契约测试 → 模型 → 服务 → API)

**预计工作量**:
- 后端实施: 约25个任务(模型6 + 仓储5 + 服务8 + API6)
- 前端实施: 约10个任务(页面6 + 组件4)
- 测试实施: 约12个任务(契约6 + 集成6)
- **总计**: 47个任务

### 文档清单

所有文档位于 `.claude/specs/003-nosql-sql/`:
- [x] `spec.md` - 功能规格(43个需求,已澄清)
- [x] `plan.md` - 此实施计划
- [x] `research.md` - 技术研究(5个任务)
- [x] `data-model.md` - 数据模型(6个实体)
- [x] `contracts/database-api.yaml` - API契约(25个端点)
- [x] `quickstart.md` - 快速启动指南
- [ ] `tasks.md` - 实施任务(待/spec-kit:tasks命令生成)

---

**规划阶段完成标志**: 所有设计文档已生成,章程检查通过,准备进入任务生成阶段。
