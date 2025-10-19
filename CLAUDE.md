# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

Tiga 是一个现代化的 DevOps Dashboard 平台，提供全面的 Kubernetes 集群管理、Docker实例、中间件等子系统运维能力。这是一个全栈应用，后端使用 Go，前端使用 React TypeScript。

**核心技术栈：**
- 后端：Go 1.24+、Gin 框架、GORM、Kubernetes client-go
- 前端：React 19、TypeScript、Vite、TailwindCSS、Radix UI
- 数据库：SQLite（默认）、PostgreSQL、MySQL
- 认证：JWT + OAuth 支持（Google、GitHub）

## 常用命令

### 开发

```bash
# 后端开发（会先构建前端）
task dev

# 仅后端开发（不重新构建前端）
task dev:backend

# 前端开发（Vite 热重载）
task dev:frontend

# 构建完整项目（前端 + 后端）
task backend
```

### 测试

```bash
# 仅单元测试（跳过集成测试）
task test

# 所有测试包括集成测试（需要 Docker）
task test-integration

# 生成 HTML 覆盖率报告
task test-report
```

### 代码质量

```bash
# 格式化和 lint
task lint          # 完整 linting（gofmt、goimports、gci）
task gofmt         # 仅格式化 Go 代码
task golint        # 运行 golangci-lint

# 前端 linting
cd ui && pnpm lint
cd ui && pnpm format
```

### 构建和部署

```bash
# 生产构建
task backend       # 构建到 bin/tiga

# 跨平台构建
task cross-compile # 构建 Linux amd64/arm64 二进制

# Docker
task docker

# 清理构建产物
task clean
```

### API 文档

```bash
# 生成 Swagger 文档
./scripts/generate-swagger.sh

# 启动服务后在 http://localhost:12306/swagger/index.html 查看文档
```

## 架构设计

### 后端结构

```
internal/
├── api/                    # HTTP API 层
│   ├── handlers/          # 请求处理器（按领域划分）
│   ├── middleware/        # 认证、RBAC、CORS、限流、审计日志
│   └── routes.go          # 路由定义
├── app/                   # 应用启动和生命周期管理
├── config/                # 配置管理（YAML + 环境变量）
├── db/                    # 数据库初始化和迁移
├── install/               # Web 安装向导
├── models/                # GORM 模型和数据库 schema
├── repository/            # 数据访问层（仓储模式）
└── services/              # 业务逻辑层
    ├── auth/             # JWT、RBAC、OAuth
    ├── managers/         # 实例管理器（MinIO、MySQL、Redis、Docker 等）
    ├── alert/            # 告警处理和通知
    ├── scheduler/        # 后台任务调度
    └── notification/     # 多渠道通知

pkg/                       # 可复用包
├── handlers/             # Kubernetes 专用处理器
│   └── resources/       # K8s 资源 CRUD 操作
├── kube/                # Kubernetes 客户端工具
├── auth/                # 认证工具
├── cluster/             # 多集群管理
├── crypto/              # 加密/解密
├── middleware/          # HTTP 中间件
├── prometheus/          # 指标收集
├── rbac/                # K8s RBAC 集成
└── utils/               # 共享工具
```

### 前端结构

```
ui/src/
├── components/           # 可复用 UI 组件
├── pages/               # 路由页面组件
├── layouts/             # 布局包装器（DevOps、K8s、MinIO 等）
├── services/            # API 客户端服务
├── contexts/            # React 上下文（auth、cluster 等）
├── hooks/               # 自定义 React hooks
├── lib/                 # 工具和辅助函数
└── types/               # TypeScript 类型定义
```

### 关键架构模式

**仓储模式（Repository Pattern）**：数据访问通过 `internal/repository/` 中的仓储抽象，服务层使用仓储处理业务逻辑。

**管理器模式（Manager Pattern）**：不同服务类型（MinIO、MySQL、Docker 等）实现统一的 `ServiceManager` 接口（位于 `internal/services/managers/`），由 `ManagerCoordinator` 协调管理。

**多子系统设计**：UI 组织为多个子系统（VMs、K8s、MinIO、Middleware、Docker、Storage、WebServer），每个子系统有独立的布局和页面，从统一的概览仪表板访问。

**中间件栈**：请求处理流程：CORS → Logger → Auth（JWT）→ RBAC → Rate Limit → Audit → Handler。

**双配置系统**：同时支持 YAML 配置文件（`config.yaml`）和环境变量，YAML 优先级更高。

## 数据库

**迁移机制**：启动时通过 `internal/app/app.go` 中的 `db.AutoMigrate()` 自动迁移。

**支持的数据库**：
- SQLite（默认，文件数据库）
- PostgreSQL（生产环境推荐）
- MySQL

**连接配置**：通过 `config.yaml` 或环境变量（`DB_TYPE`、`DB_HOST` 等）设置。

**模型位置**：`internal/models/` - 所有 GORM 模型，包含适当的关联和索引。

## 配置系统

**主配置系统**（推荐）：
- 配置文件：`config.yaml`（位于项目根目录）
- 配置包：`internal/config/`
- 支持 YAML 配置文件（优先）+ 环境变量（回退）
- 由主应用 `cmd/tiga/main.go` 使用

**配置结构** (Phase 4 改进):
```go
type Config struct {
    Server             ServerConfig             // HTTP/gRPC 服务器配置
    Database           DatabaseConfig           // 数据库连接配置
    Redis              RedisConfig              // Redis 配置
    JWT                JWTConfig                // JWT 认证配置
    OAuth              OAuthConfig              // OAuth 提供商配置
    Security           SecurityConfig           // 加密密钥、bcrypt 成本
    DatabaseManagement DatabaseManagementConfig // 数据库管理子系统配置
    Kubernetes         KubernetesConfig         // K8s 相关配置（Phase 4 新增）
    Webhook            WebhookConfig            // Webhook 配置（Phase 4 新增）
    Features           FeaturesConfig           // 功能特性开关（Phase 4 新增）
    Log                LogConfig                // 日志配置
}
```

**遗留配置**（pkg/common - 已废弃）：
- ⚠️ **DEPRECATED**: `pkg/common` 中的全局变量已在 Phase 4 中标记为废弃
- 迁移路径：
  - `common.NodeTerminalImage` → `config.Kubernetes.NodeTerminalImage`
  - `common.WebhookUsername/Password/Enabled` → `config.Webhook.*`
  - `common.GetEncryptKey()` → `config.Security.EncryptionKey`
  - `common.AnonymousUserEnabled` → `config.Features.AnonymousUserEnabled`
  - `common.DisableGZIP` → `config.Features.DisableGZIP`
  - `common.DisableVersionCheck` → `config.Features.DisableVersionCheck`
- 旧代码通过废弃函数保持兼容，计划在后续版本移除

**配置优先级**：环境变量 > YAML 配置 > 默认值

## 认证与授权

**JWT 流程**：
1. 通过 `/api/v1/auth/login` 登录，返回 access + refresh tokens
2. 所有受保护路由需要 `Authorization: Bearer <token>` header
3. `internal/api/middleware/auth.go` 中的中间件验证 JWT

**RBAC**：
- 基于角色的访问控制由中间件强制执行（`internal/api/middleware/rbac.go`）
- K8s 专用角色定义在 `internal/models/k8s_role.go`
- 全局 RBAC 系统在 `pkg/rbac/`

**OAuth**：支持 Google 和 GitHub OAuth 提供商（通过 `models.OAuthProvider` 配置）。

## Kubernetes 集成

**多集群支持**：
- 集群存储在 `models.Cluster`
- 启动时自动从 `~/.kube/config` 导入
- 通过前端集群选择器切换集群

**资源管理**：
- CRUD 操作在 `pkg/handlers/resources/`
- 支持内置资源（Pods、Deployments、Services 等）和 CRDs
- 资源历史记录在 `models.ResourceHistory`

**客户端管理**：
- K8s 客户端创建/缓存在 `pkg/kube/client.go`
- Terminal/exec 支持在 `pkg/kube/terminal.go`
- 日志流在 `pkg/kube/log.go`

## 安装向导

应用包含基于 Web 的安装向导（`internal/install/`），处理初始设置：
- 数据库配置
- 管理员用户创建
- JWT 密钥生成
- 保存配置到 `config.yaml` 并设置 `install_lock: true`

**安装守卫**：前端路由守卫在安装完成前阻止访问。

## API 约定

**响应格式**：处理器使用 `internal/api/handlers/response.go` 中的 `SendSuccess()` 和 `SendError()` 保证 JSON 响应一致性。

**Swagger 注解**：通过处理器函数中的 swag 注释生成 API 文档（全局注解见 `cmd/tiga/main.go`）。

**版本控制**：所有 API 路由以 `/api/v1` 为前缀。

## 监控与告警

**指标收集**：
- Prometheus 集成在 `pkg/prometheus/`
- 实例指标存储在 `models.Metric`
- 通过 `ManagerCoordinator` 后台收集

**告警系统**：
- 告警规则在 `models.Alert`
- 告警事件在 `models.AlertEvent`
- 通过 `services.alert.AlertProcessor` 处理（每 30 秒运行一次）

**调度器**：后台任务由 `services.scheduler.Scheduler` 管理。

## 测试

**测试组织**：
- 单元测试与源文件并列（`*_test.go`）
- 集成测试在 `tests/` 目录
- 使用 `-short` 标志跳过集成测试

**测试容器**：集成测试使用 testcontainers-go 启动 PostgreSQL。

**覆盖率**：运行 `task test-report` 生成 HTML 覆盖率报告。

## 开发注意事项

**前端构建**：前端（`ui/`）构建到 `static/` 目录，嵌入到 Go 二进制文件中。后端从根路径提供静态文件，从 `/api/*` 提供 API。

**热重载**：前端开发使用 `task dev:frontend`，运行 Vite 开发服务器并代理到后端。

**数据库迁移**：GORM 自动迁移在启动时创建/更新表。当前不使用手动迁移。

**日志记录**：使用 logrus 进行结构化日志。通过 `LOG_LEVEL` 和 `LOG_FORMAT` 环境变量配置。

## 常见模式

**错误处理**：服务层返回错误，处理器通过 `SendError()` 转换为 HTTP 响应。

**上下文传播**：通过服务层传递 `context.Context` 以支持取消和超时控制。

**仓储查询**：在仓储中使用 GORM 查询构建器。尽量减少原始 SQL。

**前端数据获取**：使用 TanStack Query（`@tanstack/react-query`）管理服务端状态。

**类型安全**：后端使用强类型。前端启用 TypeScript 严格模式。

## 文件命名约定

- Go：`snake_case.go`（如 `user_handler.go`）
- React：`kebab-case.tsx`（如 `user-form-page.tsx`）
- 测试：`*_test.go`
- 配置：`config.yaml`、`.env`

## 新增功能：主机管理子系统（开发中）

**功能分支**：`002-nezha-webssh`
**参考规格**：`.claude/specs/002-nezha-webssh/`

### 核心特性
- **Agent-Server架构**：轻量级Agent部署在被监控主机，通过gRPC持久连接上报数据
- **实时监控**：CPU、内存、磁盘、网络等指标实时采集和展示
- **服务探测**：支持HTTP/TCP/ICMP服务健康检查，Agent端分布式执行
- **WebSSH终端**：浏览器内SSH终端，通过Agent代理实现安全访问
- **告警系统**：表达式引擎驱动的灵活告警规则，支持多通知渠道

### 技术栈
- **通信**：gRPC双向流（Agent-Server）、WebSocket（浏览器实时数据）
- **Agent**：Go独立二进制、gopsutil系统监控、systemd服务管理
- **调度**：robfig/cron服务探测定时任务
- **告警**：antonmedv/expr表达式引擎
- **前端**：Zustand实时状态管理、Recharts监控图表、xterm.js终端UI

### 数据模型
- `HostNode`：主机节点元数据
- `HostInfo`：主机硬件和系统信息
- `HostState`：实时监控指标（时序数据）
- `ServiceMonitor`：服务探测规则
- `MonitorAlertRule`：告警规则
- `AlertEvent`：告警事件
- `WebSSHSession`：SSH会话管理

### API端点
- `/api/v1/hosts`：主机管理CRUD
- `/api/v1/hosts/{id}/state`：监控数据查询
- `/api/v1/host-groups`：主机分组
- `/api/v1/service-monitors`：服务探测规则
- `/api/v1/alert-rules`：告警规则
- `/api/v1/webssh/{session_id}`：WebSSH终端（WebSocket）
- `/api/v1/ws/host-monitor`：实时监控订阅（WebSocket）

### 实施阶段
- Phase 1：核心监控（Agent连接、数据采集、实时展示）
- Phase 2：服务探测（探测规则、任务调度、结果聚合）
- Phase 3：WebSSH（SSH代理、终端UI、会话管理）
- Phase 4：高级功能（分组管理、自定义告警、Prometheus集成）

## 新增功能：数据库管理子系统（已完成）

**功能分支**：`003-nosql-sql`
**参考规格**：`.claude/specs/003-nosql-sql/`
**完成度**：96% (54/56 任务)

### 核心特性
- **多数据库支持**：MySQL 8.0+、PostgreSQL 15+、Redis 7+ 统一管理
- **安全优先**：AES-256-GCM 密码加密、SQL 安全过滤、命令黑名单、全操作审计
- **权限管理**：统一权限模型（readonly/readwrite）自动映射到不同数据库系统
- **查询控制台**：支持 SQL/Redis 命令执行，30秒超时，10MB结果限制
- **连接管理**：连接池（50 open/10 idle）、连接缓存、健康检查

### 技术栈
- **数据库驱动**：go-sql-driver/mysql v1.8.1、lib/pq v1.10.9、go-redis/v9 v9.5.1
- **安全**：pkg/crypto（AES-256-GCM）、xwb1989/sqlparser（SQL 解析）
- **测试**：testcontainers-go（集成测试）、testify（断言库）
- **前端**：Monaco Editor（SQL 编辑器）、react-window（虚拟滚动）

### 数据模型
- `DatabaseInstance`：数据库实例（加密密码存储）
- `Database`：数据库元数据（支持 MySQL/PostgreSQL/Redis）
- `DatabaseUser`：数据库用户（加密密码）
- `PermissionPolicy`：权限策略（软删除支持）
- `QuerySession`：查询会话记录（性能指标）
- `DatabaseAuditLog`：审计日志（operator、action、details、client_ip）

### API端点
- `/api/v1/database/instances`：实例管理CRUD、连接测试
- `/api/v1/database/instances/{id}/databases`：数据库操作
- `/api/v1/database/instances/{id}/users`：用户管理
- `/api/v1/database/permissions`：权限授予和撤销
- `/api/v1/database/instances/{id}/query`：查询执行（安全过滤）
- `/api/v1/database/audit-logs`：审计日志查询（分页、过滤）

### 安全特性
**SQL 安全过滤**：
- 阻止 DDL 操作（DROP、TRUNCATE、ALTER、CREATE INDEX 等）
- 阻止无 WHERE 的 UPDATE/DELETE
- 阻止危险函数（LOAD_FILE、INTO OUTFILE、DUMPFILE）
- 性能目标：<2ms 验证时间

**Redis 命令过滤**：
- 黑名单：FLUSHDB、FLUSHALL、SHUTDOWN、CONFIG、SAVE、BGSAVE
- 大小写不敏感匹配
- 性能目标：<100μs 验证时间

**权限映射**：
- **MySQL/PostgreSQL**：
  - readonly → SELECT 权限
  - readwrite → SELECT, INSERT, UPDATE, DELETE 权限
- **Redis ACL**：
  - readonly → +@read -@write -@dangerous
  - readwrite → +@read +@write -@dangerous

### 性能优化
- **连接池**：MaxOpenConns=50、MaxIdleConns=10
- **连接缓存**：实例级缓存避免重复建连
- **查询限制**：
  - 超时：30秒（可配置）
  - 结果大小：10MB（超出自动截断并提示）
- **审计清理**：定时任务每天 2AM 删除 90 天前日志

### 测试覆盖
- **契约测试**（6个文件）：API 规范验证
- **集成测试**（7个文件）：真实数据库环境测试（testcontainers）
- **单元测试**（3个文件）：核心逻辑验证（security_filter、credential、acl_mapping）
- **总计**：135+ 测试用例

### 关键文件
```
internal/
├── models/                # 数据模型（6个）
│   └── db_*.go
├── repository/database/   # 仓储层（5个）
├── services/database/     # 服务层（8个）
│   ├── security_filter.go     # SQL/Redis 安全过滤
│   ├── manager.go             # 实例管理器（连接缓存）
│   ├── query_executor.go      # 查询执行器（超时控制）
│   └── audit_logger.go        # 审计日志记录
└── api/handlers/database/ # API 处理器（6个）

pkg/
├── dbdriver/              # 数据库驱动（4个）
│   ├── driver.go          # 接口定义
│   ├── mysql.go, postgres.go, redis.go
└── crypto/                # 加密服务
    └── encryption.go      # AES-256-GCM 实现

tests/
├── contract/              # 契约测试（6个）
├── integration/database/  # 集成测试（7个）
└── unit/                  # 单元测试（3个）
```

### 使用示例
```bash
# 运行数据库管理相关测试
go test -v ./tests/contract/database_*.go
go test -v ./tests/integration/database/... -timeout 5m
go test -v ./tests/unit/...

# 启动应用（包含数据库管理功能）
task dev

# 访问 Swagger 文档
# http://localhost:12306/swagger/index.html
```

### 前端页面
- **实例列表**：显示所有数据库实例，支持创建、编辑、删除、连接测试
- **数据库列表**：查看和管理实例下的数据库
- **用户管理**：创建用户、修改密码、删除用户
- **权限管理**：授予和撤销数据库权限（readonly/readwrite）
- **查询控制台**：Monaco Editor 编辑器，执行 SQL/Redis 命令
- **审计日志**：查询和筛选所有数据库操作日志

### 待完成任务
- API 文档生成（运行 `./scripts/generate-swagger.sh`）
- 代码质量检查（运行 `task lint`）
- 手动验证（参考 `.claude/specs/003-nosql-sql/quickstart.md`）

## 架构改进：Phase 4 (2025-10-16)

**参考文档**: `docs/architecture/phase4-improvements.md`

### 概述

Phase 4 专注于提升代码架构质量，改进可测试性和可维护性。遵循 SOLID 原则和依赖注入最佳实践。

### 完成的改进

#### 1. Repository 接口抽象 ✅

**位置**: `internal/repository/interfaces.go`

定义了 8 个 Repository 接口，解耦数据访问层：
- `UserRepositoryInterface`
- `InstanceRepositoryInterface`
- `AlertRepositoryInterface`  
- `MetricsRepositoryInterface`
- `AuditLogRepositoryInterface`
- `ClusterRepositoryInterface`
- `ResourceHistoryRepositoryInterface`
- `OAuthProviderRepositoryInterface`

**收益**:
- 服务层可使用接口类型，便于单元测试 mock
- 符合依赖倒置原则 (DIP)
- 为未来切换存储后端奠定基础

**使用示例**:
```go
// 旧代码（紧耦合）
type UserService struct {
    repo *repository.UserRepository  // 具体实现
}

// 新代码（松耦合）
type UserService struct {
    repo repository.UserRepositoryInterface  // 接口
}

// 单元测试时可以 mock
type MockUserRepository struct{}
func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
    return &models.User{ID: id, Username: "test"}, nil
}
```

#### 2. 统一配置系统 ✅

**扩展**: `internal/config/config.go`

新增 3 个配置结构（详见"配置系统"章节）：
- `KubernetesConfig` - K8s 相关配置  
- `WebhookConfig` - Webhook 配置
- `FeaturesConfig` - 功能特性开关

**废弃**: `pkg/common/common.go` 中的全局变量

所有全局状态已迁移到配置结构，旧代码保持兼容（通过废弃函数）。

**加密工具改进** (`pkg/utils/secure.go`):
```go
// 新增：接受显式密钥参数
func EncryptStringWithKey(input, encryptionKey string) string
func DecryptStringWithKey(encrypted, encryptionKey string) (string, error)

// 旧函数保留（标记为 Deprecated）
func EncryptString(input string) string  // 使用 common.GetEncryptKey()
func DecryptString(encrypted string) (string, error)
```

#### 3. 架构问题识别 ✅

**问题**: `internal/app/app.go` 中的 God Object

Application 结构体包含 16 个字段，195 行 Initialize 方法，违反：
- 单一职责原则 (SRP)
- 开闭原则 (OCP)

**决策**: 暂缓重构（按 Option C），优先完成接口抽象和配置统一。

### 遵循的架构原则

- ✅ **SOLID 原则**: 所有 5 项原则
- ✅ **Repository Pattern**: 数据访问抽象
- ✅ **Dependency Injection**: 配置和依赖注入
- ✅ **避免全局状态**: 消除可变全局变量
- ✅ **向后兼容**: 零破坏性改进

### 测试验证

**单元测试**:
```bash
go test ./tests/unit/n1_query_test.go  # 3/3 通过
go test ./tests/unit/async_audit_logger_test.go  # 9/9 通过
```

**编译验证**:
```bash
go build ./internal/repository/...  # ✅
go build ./internal/config/...      # ✅  
go build ./internal/app             # ✅
```

### 后续改进计划

**短期** (1-2 周):
1. 迁移 `pkg/common` 使用者到新配置系统
2. 服务层接口化

**中期** (1-2 月):
1. App 结构重构（分阶段）
2. 完全移除 `pkg/common`

**长期** (3-6 月):
1. 存储抽象化（利用 Repository 接口）
2. 微服务化准备

详见完整文档：`docs/architecture/phase4-improvements.md`


## Kubernetes 集群管理子系统（已完成）

**功能分支**: `005-k8s-kite-k8s`
**参考规格**: `.claude/specs/005-k8s-kite-k8s/`
**完成度**: 67% (55/82 任务)

### 核心特性

- **多集群管理**: 支持多个 Kubernetes 集群统一管理，集群健康状态实时监控
- **CRD 支持**: 原生支持 OpenKruise、Tailscale、Traefik、K3s System Upgrade Controller
- **Prometheus 集成**: 自动发现集群内 Prometheus 实例，支持监控数据查询
- **全局搜索**: 跨资源类型全局搜索（Pod、Deployment、Service、ConfigMap），智能评分排序
- **资源关系图**: 可视化资源依赖关系（Deployment → ReplicaSet → Pod）
- **节点终端**: 基于 WebSocket 的节点 SSH 终端，通过特权 Pod 实现
- **只读模式**: 系统级只读模式，阻止所有修改操作
- **审计日志**: 记录所有集群操作和终端访问

### 技术栈

- **后端**: client-go v0.31.4、dynamic client、OpenKruise SDK v1.8.0
- **前端**: React 19、ClusterContext、xterm.js（终端UI）
- **缓存**: 5 分钟 TTL 内存缓存（ResourceVersion 感知）
- **测试**: testcontainers-go + Kind（集成测试）、fake client（单元测试）

### 数据模型

**核心模型**（位于 `internal/models/`）:
- `Cluster`: 集群元数据（名称、Kubeconfig、健康状态、Prometheus URL）
- `ResourceHistory`: 资源变更历史记录
- `K8sRole`: Kubernetes RBAC 角色定义

**扩展字段**（Cluster 模型）:
```go
type Cluster struct {
    BaseModel
    Name             string    `json:"name"`
    Config           string    `json:"-"`                // Kubeconfig（加密存储）
    InCluster        bool      `json:"in_cluster"`       // 是否为 In-Cluster 配置
    Enable           bool      `json:"enable"`
    Description      string    `json:"description"`
    
    // Phase 0 扩展字段
    HealthStatus     string    `json:"health_status"`    // unknown, healthy, warning, error, unavailable
    LastConnectedAt  time.Time `json:"last_connected_at"`
    NodeCount        int       `json:"node_count"`
    PodCount         int       `json:"pod_count"`
    PrometheusURL    string    `json:"prometheus_url"`   // 自动发现或手动配置
}
```

### 服务架构

**核心服务**（位于 `internal/services/k8s/`）:

1. **ClusterHealthService**: 集群健康检查（60秒间隔后台任务）
   - 调用 `/api/v1/nodes` 获取节点列表
   - 更新 `health_status`、`node_count`、`pod_count`
   - 状态转换：unknown → healthy → warning → error → unavailable

2. **RelationsService**: 资源关系服务
   - 静态关系映射：Deployment → ReplicaSet → Pod（8 种关系）
   - 递归查询（最大深度 3）
   - 循环引用检测（visited map）

3. **CacheService**: 工作负载缓存
   - 缓存键：`clusterID:resourceType:namespace`
   - TTL：5 分钟
   - ResourceVersion 检测自动失效
   - 线程安全（RWMutex）

4. **SearchService**: 全局搜索
   - 并发查询 4 个资源类型
   - 评分算法：
     - 精确匹配：100 分
     - 名称包含：80 分
     - 标签匹配：60 分
     - 注解匹配：40 分
   - 结果限制：50 条

5. **PrometheusAutoDiscoveryService**: Prometheus 自动发现
   - 异步 Goroutine，30 秒超时
   - 搜索命名空间：monitoring、prometheus、kube-system
   - 端点优先级：LoadBalancer > NodePort > ClusterIP
   - 连通性测试：`GET /api/v1/status/config`（2 秒超时）

### API 端点

**集群管理**:
- `GET /api/v1/k8s/clusters` - 集群列表
- `GET /api/v1/k8s/clusters/:id` - 集群详情
- `POST /api/v1/k8s/clusters` - 创建集群
- `PUT /api/v1/k8s/clusters/:id` - 更新集群
- `DELETE /api/v1/k8s/clusters/:id` - 删除集群
- `POST /api/v1/k8s/clusters/:id/test-connection` - 测试连接
- `POST /api/v1/k8s/clusters/:id/prometheus/rediscover` - 重新检测 Prometheus

**资源管理**（CRD 支持）:
- OpenKruise: `/api/v1/k8s/clusters/:cluster_id/clonesets`（扩容、重启）
- Tailscale: `/api/v1/k8s/clusters/:cluster_id/tailscale/connectors`
- Traefik: `/api/v1/k8s/clusters/:cluster_id/traefik/ingressroutes`
- K3s: `/api/v1/k8s/clusters/:cluster_id/k3s/plans`

**增强功能**:
- `GET /api/v1/k8s/clusters/:cluster_id/search?q=<query>` - 全局搜索
- `GET /api/v1/k8s/clusters/:cluster_id/resources/:kind/:name/relations` - 资源关系
- `GET /api/v1/k8s/clusters/:cluster_id/crds` - CRD 检测

**节点终端**:
- `POST /api/v1/k8s/clusters/:cluster_id/nodes/:name/terminal` - 创建终端会话
- `WS /api/v1/k8s/terminal/:session_id` - WebSocket 终端连接

### 配置系统

**Kubernetes 配置**（config.yaml）:
```yaml
kubernetes:
  node_terminal_image: "alpine:latest"  # 终端特权 Pod 镜像
  enable_kruise: true                   # 启用 OpenKruise
  enable_tailscale: true                # 启用 Tailscale
  enable_traefik: true                  # 启用 Traefik
  enable_k3s_upgrade: true              # 启用 K3s Upgrade Controller
```

**Prometheus 配置**:
```yaml
prometheus:
  auto_discovery: true         # 启用自动发现
  discovery_timeout: 30        # 发现超时（秒）
  cluster_urls:                # 手动配置（优先级高于自动发现）
    cluster-1: "http://prometheus.monitoring:9090"
```

**功能特性**:
```yaml
features:
  readonly_mode: false         # 只读模式开关
```

### 前端页面

**位置**: `ui/src/pages/k8s/`

**核心页面**:
- `cluster-list-page.tsx`: 集群列表（健康状态、节点数、Pod 数）
- `cluster-detail-page.tsx`: 集群详情（概览、配置、Prometheus 三个 Tab）
- `cluster-form-page.tsx`: 集群创建/编辑表单
- `search-page.tsx`: 全局搜索页（搜索、过滤、结果分组）
- `monitoring-page.tsx`: Prometheus 监控配置

**CRD 页面** (列表 + 详情):
- OpenKruise: `cloneset-list-page.tsx`、`advanced-daemonset-list-page.tsx`
- Tailscale: `connector-list-page.tsx`、`proxyclass-list-page.tsx`
- Traefik: `ingressroute-list-page.tsx`、`middleware-list-page.tsx`
- K3s: `upgrade-plans-list-page.tsx`

**核心组件**:
- `cluster-selector.tsx`: 集群切换器（下拉菜单）
- `resource-relations.tsx`: 资源关系树形视图
- `terminal.tsx`: xterm.js 终端 UI

**状态管理**:
- `ClusterContext`（cluster-context.tsx）: 当前选中集群 ID
- 切换集群时清除缓存和临时状态

### 测试覆盖

**集成测试**（位于 `tests/integration/k8s/`）:
- `cluster_health_test.go`: 集群健康检查（239 行，使用 testcontainers + Kind）
- `prometheus_discovery_test.go`: Prometheus 自动发现框架
- `search_performance_test.go`: 全局搜索性能（<1 秒，1000+ 资源）

**单元测试**（位于 `tests/unit/k8s/`）:
- `relations_test.go`: 资源关系服务（270 行，5 个测试用例）
- `cache_test.go`: 缓存服务（266 行，7 个测试用例）
- `search_test.go`: 搜索服务（332 行，6 个测试用例）

**测试框架**:
- testcontainers-go: Docker 容器化 K8s 集群（Kind）
- fake.NewSimpleClientset(): Kubernetes fake client
- fake.NewSimpleDynamicClient(): Dynamic client fake

### 使用示例

**导入集群**:
```bash
# 启动应用，自动从 ~/.kube/config 导入集群
task dev

# 访问前端
# http://localhost:5174/k8s/clusters
```

**全局搜索**:
```bash
# API 调用
curl -H "Authorization: Bearer <token>" \
     "http://localhost:12306/api/v1/k8s/clusters/<cluster-id>/search?q=redis&limit=50"

# 前端页面
# http://localhost:5174/k8s/search
```

**节点终端**:
```bash
# 1. 创建终端会话
POST /api/v1/k8s/clusters/:cluster_id/nodes/:node_name/terminal

# 2. 连接 WebSocket
ws://localhost:12306/api/v1/k8s/terminal/:session_id

# 前端页面会自动处理 WebSocket 连接
```

### 关键实现细节

**Client 缓存**（pkg/kube/client.go）:
- 双检锁模式创建 K8s client
- 集群更新/删除时自动清除缓存
- 线程安全

**节点终端原理**:
1. 创建特权 Pod（hostNetwork、hostPID、privileged）
2. 通过 WebSocket 建立终端连接
3. 使用 xterm.js 渲染终端 UI
4. 30 分钟超时自动清理 Pod

**只读模式中间件**（internal/api/middleware/readonly.go）:
- 阻止 POST、PUT、PATCH、DELETE 请求
- 白名单：登录、健康检查、测试连接
- 返回 HTTP 403 Forbidden

**审计日志增强**:
- ClusterID 和 ClusterName 字段
- 记录所有资源修改操作
- 记录所有节点终端访问

### 待完成任务

**高优先级**:
- [ ] 契约测试（0/14）：API 规范验证
- [ ] 剩余集成测试（T020、T022）
- [ ] Prometheus 发现单元测试（T077）

**中优先级**:
- [ ] 通用 CRD 处理器框架（T038）
- [ ] CRD 检测 API（T039）
- [ ] 性能测试（T078）

**低优先级**:
- [ ] Swagger 注释（K8s API 端点）
- [ ] 手动验证（quickstart.md 验证场景）
- [ ] 代码质量检查（`task lint`）

### 参考文档

- 任务清单：`.claude/specs/005-k8s-kite-k8s/tasks.md`
- 快速开始：`.claude/specs/005-k8s-kite-k8s/quickstart.md`
- 数据模型：`.claude/specs/005-k8s-kite-k8s/data-model.md`
- API 契约：`.claude/specs/005-k8s-kite-k8s/contracts/`
