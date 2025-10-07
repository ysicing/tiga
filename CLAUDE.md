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

**遗留配置**（pkg/common）：
- 一些旧代码仍使用 `pkg/common` 中的硬编码默认值
- 这些值会逐步迁移到主配置系统
- 主要包括：JWT 密钥、节点终端镜像、加密密钥等

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
