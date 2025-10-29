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
├── layouts/             # 布局包装器（System Management、VMs、K8s、MinIO、Databases、Docker、WebServer 等）
├── services/            # API 客户端服务
├── contexts/            # React 上下文（auth、cluster 等）
├── hooks/               # 自定义 React hooks
├── lib/                 # 工具和辅助函数
└── types/               # TypeScript 类型定义
```

### 关键架构模式

**仓储模式（Repository Pattern）**：数据访问通过 `internal/repository/` 中的仓储抽象，服务层使用仓储处理业务逻辑。

**管理器模式（Manager Pattern）**：不同服务类型（MinIO、MySQL、Docker 等）实现统一的 `ServiceManager` 接口（位于 `internal/services/managers/`），由 `ManagerCoordinator` 协调管理。

**多子系统设计**：UI 组织为多个子系统，每个子系统有独立的布局和页面，从统一的概览仪表板访问：
- **System Management**（系统管理）：定时任务、审计日志、用户管理、全局配置（集群、OAuth、RBAC）
- **VMs**（主机管理）：物理和虚拟服务器管理、SSH 访问、服务监控、告警规则
- **K8s**（Kubernetes）：集群管理、工作负载、CRD 支持（OpenKruise、Tailscale、Traefik、System Upgrade）
- **Databases**（数据库）：MySQL、PostgreSQL、Redis 实例管理、查询控制台、权限管理
- **MinIO**（对象存储）：MinIO 实例管理、Bucket 管理、用户和策略配置
- **Docker**（容器）：Docker 容器、镜像、网络管理
- **WebServer**（Web 服务器）：Caddy Web 服务器和反向代理管理

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

## 主机管理子系统（开发中）

**功能分支**：`002-nezha-webssh`
**参考规格**：`.claude/specs/002-nezha-webssh/`

Agent-Server 架构的主机监控系统，提供实时指标采集、服务探测、WebSSH 终端和告警功能。

**核心特性**：
- gRPC 双向流通信，轻量级 Agent 部署
- 实时监控（CPU、内存、磁盘、网络）
- 浏览器内 SSH 终端（通过 Agent 代理）
- 灵活告警规则（表达式引擎驱动）

详见规格文档以了解完整的数据模型、API 和实施阶段。

## 数据库管理子系统（已完成）

**功能分支**：`003-nosql-sql`
**参考规格**：`.claude/specs/003-nosql-sql/`
**完成度**：96%

统一管理 MySQL、PostgreSQL、Redis 数据库实例，提供查询控制台、权限管理和全操作审计。

**核心特性**：
- 多数据库统一管理（MySQL 8.0+、PostgreSQL 15+、Redis 7+）
- 安全优先（AES-256-GCM 加密、SQL/Redis 命令安全过滤、审计日志）
- 统一权限模型（readonly/readwrite 自动映射到不同数据库系统）
- 查询控制台（Monaco Editor、30秒超时、10MB 结果限制）

**关键文件**：
- 数据模型：`internal/models/db_*.go`
- 服务层：`internal/services/database/`（security_filter、manager、query_executor、audit_logger）
- 数据库驱动：`pkg/dbdriver/`

详见规格文档以了解完整的 API、安全特性、性能优化和测试覆盖。

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
**完成度**: 67%

多集群 Kubernetes 管理，支持 CRD、Prometheus 集成、全局搜索和节点终端。

**核心特性**：
- 多集群统一管理，实时健康状态监控
- 原生 CRD 支持（OpenKruise、Tailscale、Traefik、K3s System Upgrade）
- Prometheus 自动发现和监控数据查询
- 全局搜索（跨资源类型、智能评分排序）
- 资源关系图可视化（Deployment → ReplicaSet → Pod）
- 节点终端（WebSocket、特权 Pod 实现）
- 系统级只读模式

**扩展的 Cluster 模型字段**：`health_status`、`node_count`、`pod_count`、`prometheus_url`

**核心服务**（`internal/services/k8s/`）：
- `ClusterHealthService` - 60秒间隔后台健康检查
- `SearchService` - 并发全局搜索
- `RelationsService` - 资源关系递归查询
- `CacheService` - 5分钟 TTL 工作负载缓存
- `PrometheusAutoDiscoveryService` - 30秒超时自动发现

**关键实现**：
- Client 缓存（双检锁模式，线程安全）
- 只读模式中间件（阻止修改操作）
- 审计日志增强（记录集群操作和终端访问）

详见规格文档以了解完整的数据模型、API 端点、配置系统、测试覆盖和使用示例。

## Docker 管理子系统（架构优化）

**功能分支**: `007-docker-docker-agent`
**完成时间**: 2025-10-25

Agent-Server 架构的 Docker 远程管理，支持 NAT 穿透。

**核心设计**：Agent 位于 NAT/防火墙后，Server 无法主动连���。所有通信通过 Agent 主动建立的连接进行。

**双模式架构**：
1. **非流式操作**（任务队列）- 29个操作：容器/镜像/网络/卷管理、系统信息查询
2. **流式操作**（DockerStream）- 5个操作：容器终端、日志、统计、镜像拉取、事件流

**关键文件**：
- Agent: `cmd/tiga-agent/docker_handler.go`、`docker_stream_handler.go`
- Server: `internal/services/docker/agent_forwarder_v2.go`
- Proto: `proto/host_monitor.proto`

**架构优势**：
- NAT 友好（Agent 主动连接，无需公网 IP）
- 统一 Agent（不需要独立 Docker Agent 进程）
- 双模式互补（简单操作用任务队列，流式用专用连接）

详见规格文档以了解完整的通信流程、Proto 消息设计和使用示例。

## 统一终端录制系统（开发中）

**功能分支**：`009-3`
**参考规格**：`.claude/specs/009-3/`
**完成度**：20%

统一 Docker、WebSSH、K8s 三种终端的录制实现，提供统一的存储、清理和回放。

**核心目标**：
- 解决 3 个分散的终端录制实现（路径和清理策略不统一）
- 统一数据模型 + 统一存储服务（本地/MinIO）+ 统一清理任务
- 支持 Docker 容器、WebSSH 主机、K8s 节点、K8s Pod

**核心服务**（`internal/services/recording/`）：
- `StorageService` - 存储抽象（本地/MinIO）
- `CleanupService` - 定时清理（90天保留期）
- `ManagerService` - 录制管理（CRUD、搜索、统计）

**已完成工作**：
- ✅ 录制服务目录结构和配置扩展
- ✅ 数据库索引创建
- ✅ 9 个 API 端点契约测试（T005-T013）
- ⏸️ 10 个集成测试待完成

详见规格文档以了解完整的数据模型、API 端点、配置和参考文档。

## K8s 终端录制与审计增强（开发中）

**功能分支**: `010-k8s-pod-009`
**参考规格**: `.claude/specs/010-k8s-pod-009/`
**完成度**: 0%

K8s 终端录制与审计增强，支持节点终端和容器终端自动录制，以及 K8s 资源操作全面审计。

**核心特性**：
- 自动录制节点终端和容器终端会话（Asciinema v2 格式，2 小时限制）
- 记录所有 K8s 资源操作审计（创建、更新、删除、查看）
- 终端访问审计（关联到录制记录）
- 多维度审计日志查询（操作者、操作类型、资源类型、时间范围）
- 90 天自动清理（录制文件 + 审计日志）

**依赖系统**：
- 009-3 统一终端录制系统（TerminalRecording、StorageService、CleanupService）
- 统一 AuditEvent 系统

**扩展的数据模型**：
- **TerminalRecording**: 新增 `k8s_node` 和 `k8s_pod` 录制类型
- **AuditEvent**: 新增 K8s 子系统操作类型（CreateResource、UpdateResource、DeleteResource、ViewResource、NodeTerminalAccess、PodTerminalAccess）

**关键文件**：
- 模型：`internal/models/terminal_recording.go`（扩展 K8s 类型）、`internal/models/audit_event.go`（扩展 K8s 子系统）
- 服务：`internal/services/k8s/terminal_recording_service.go`、`internal/services/k8s/audit_service.go`
- 终端集成：`pkg/kube/terminal_recorder.go`
- 审计拦截：`pkg/handlers/resources/audit_interceptor.go`

**技术方法**：
- 装饰器模式：包装 TerminalSession 添加录制功能（非侵入式）
- 中间件模式：Gin 中间件拦截 K8s 资源操作添加审计
- 异步审计日志：Channel 缓冲（1000）+ 批量写入（100 条或 1 秒）
- 实时录制：Asciinema v2 格式，逐帧写入避免内存占用
- 2 小时限制：定时器触发停止录制，保持终端连接，发送 WebSocket 通知

**性能目标**：
- 终端连接延迟增加 < 100ms
- 资源操作延迟增加 < 50ms
- 审计日志查询 < 500ms（通过索引优化）

详见规格文档以了解完整的数据模型、API 契约、测试策略和实施计划。


