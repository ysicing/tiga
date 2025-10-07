---
created: 2025-10-06T23:34:01+0800
last_updated: 2025-10-06T23:34:01+0800
version: 1.0
author: Claude Code PM System
---

# 项目结构与文件组织

## 📁 顶层目录结构

```
tiga/
├── cmd/                    # 应用入口点
│   └── tiga/              # 主应用可执行文件
├── internal/              # 私有应用代码（不可被其他项目导入）
│   ├── api/              # HTTP API 层
│   ├── app/              # 应用初始化和生命周期
│   ├── config/           # 配置管理
│   ├── db/               # 数据库初始化和工具
│   ├── install/          # 安装向导逻辑
│   ├── models/           # 数据库模型（GORM）
│   ├── repository/       # 数据访问层
│   └── services/         # 业务逻辑层
├── pkg/                   # 可复用的公共包
│   ├── auth/             # 认证工具
│   ├── cluster/          # Kubernetes 集群管理
│   ├── crypto/           # 加密/解密工具
│   ├── handlers/         # Kubernetes 专用处理器
│   ├── kube/             # Kubernetes 客户端工具
│   ├── middleware/       # HTTP 中间件
│   ├── prometheus/       # Prometheus 集成
│   ├── rbac/             # RBAC 权限管理
│   ├── utils/            # 通用工具函数
│   └── version/          # 版本管理
├── ui/                    # 前端 React 应用
│   ├── src/              # 源代码
│   └── public/           # 静态资源
├── static/                # 编译后的前端静态文件（嵌入到 Go 二进制）
├── tests/                 # 测试文件
│   ├── backend/          # 后端测试
│   └── e2e/              # 端到端测试
├── docs/                  # 项目文档
├── scripts/               # 构建和部署脚本
├── .claude/               # Claude Code 上下文和配置
├── config.yaml            # 主配置文件
├── Dockerfile             # Docker 镜像构建
├── Taskfile.yml           # Task 构建任务定义
├── go.mod                 # Go 模块依赖
└── README.md              # 项目说明
```

## 🏗️ 后端架构详解

### 1. cmd/ - 应用入口

```
cmd/tiga/
└── main.go                # 应用主入口
    ├── 初始化配置
    ├── 设置日志
    ├── 初始化数据库
    ├── 启动 HTTP 服务器
    └── 注册路由和中间件
```

**关键职责**:
- 应用启动和优雅关闭
- 全局配置加载
- Swagger 文档配置
- 服务器启动

### 2. internal/api/ - API 层

```
internal/api/
├── routes.go              # 所有 API 路由定义（核心路由文件）
├── handlers/              # 请求处理器
│   ├── alert_handler.go   # 告警管理
│   ├── audit_handler.go   # 审计日志
│   ├── auth_handler.go    # 认证处理
│   ├── instance_handler.go # 实例管理
│   ├── metrics_handler.go # 指标处理
│   ├── system_handler.go  # 系统配置
│   ├── user_handler.go    # 用户管理
│   ├── database/          # 数据库实例处理器
│   │   ├── databases.go
│   │   ├── query.go
│   │   └── users.go
│   ├── docker/            # Docker 处理器
│   │   ├── containers.go
│   │   ├── images.go
│   │   └── logs.go
│   ├── instances/         # 实例专用处理器
│   │   ├── health.go
│   │   └── metrics.go
│   ├── minio/             # MinIO 处理器
│   │   ├── buckets.go
│   │   └── objects.go
│   ├── response.go        # 统一响应格式
│   └── utils.go           # 处理器工具函数
└── middleware/            # API 中间件
    ├── auth.go            # JWT 认证中间件
    ├── rbac.go            # RBAC 权限检查
    ├── audit.go           # 审计日志记录
    └── rate_limit.go      # 请求限流
```

**路由组织**:
- `/api/auth` - 认证相关（登录、OAuth）
- `/api/config` - 公共配置
- `/api/v1/admin` - 管理员功能
- `/api/v1/cluster/:clusterid` - Kubernetes 资源
- `/api/v1/instances` - 实例管理
- `/api/v1/minio` - MinIO 操作
- `/api/v1/alerts` - 告警管理
- `/api/v1/audit` - 审计日志

### 3. internal/models/ - 数据模型

```
internal/models/
├── base.go                # 基础模型（BaseModel）
├── user.go                # 用户模型
├── cluster.go             # K8s 集群模型
├── instance.go            # 实例模型
├── alert.go               # 告警规则模型
├── alert_event.go         # 告警事件模型
├── audit_log.go           # 审计日志模型
├── resource_history.go    # 资源历史模型
├── oauth_provider.go      # OAuth 提供商模型
├── role.go                # 角色模型
├── session.go             # 会话模型
├── metric.go              # 监控指标模型
├── backup.go              # 备份模型
├── event.go               # 事件模型
├── types.go               # 共享类型定义
└── compat.go              # 兼容性模型
```

**模型特点**:
- 所有模型继承 `BaseModel`（ID、创建时间、更新时间、删除时间）
- 使用 GORM 标签进行数据库映射
- 包含 JSON 序列化标签
- 支持软删除（Soft Delete）

### 4. internal/repository/ - 仓储层

```
internal/repository/
├── user_repo.go           # 用户数据访问
├── instance_repo.go       # 实例数据访问
├── instance_repo_optimized.go  # 优化版（带缓存）
├── alert_repo.go          # 告警数据访问
├── audit_repo.go          # 审计日志数据访问
├── audit_repo_optimized.go # 优化版（带缓存）
├── k8s_repository.go      # K8s 资源历史数据访问
├── metrics_repo.go        # 指标数据访问
└── oauth_provider_repo.go # OAuth 提供商数据访问
```

**仓储模式优势**:
- 抽象数据访问逻辑
- 便于单元测试（可 mock）
- 支持缓存层
- 统一错误处理

### 5. internal/services/ - 业务逻辑层

```
internal/services/
├── instance_service.go    # 实例管理服务
├── k8s_service.go         # Kubernetes 服务
├── auth/                  # 认证服务
│   ├── jwt_manager.go     # JWT 令牌管理
│   ├── login_service.go   # 登录服务
│   ├── session_service.go # 会话管理
│   └── oauth_service.go   # OAuth 服务
├── managers/              # 实例管理器（Manager Pattern）
│   ├── coordinator.go     # 管理器协调器
│   ├── service_manager.go # 管理器接口
│   ├── minio_manager.go   # MinIO 管理器
│   ├── mysql_manager.go   # MySQL 管理器
│   ├── postgresql_manager.go # PostgreSQL 管理器
│   ├── redis_manager.go   # Redis 管理器
│   ├── docker_manager.go  # Docker 管理器
│   └── base/              # 基础管理器
├── alert/                 # 告警服务
│   ├── processor.go       # 告警处理器
│   └── evaluator.go       # 告警规则评估
├── notification/          # 通知服务
│   ├── notifier.go        # 通知器接口
│   ├── email.go           # 邮件通知
│   ├── webhook.go         # Webhook 通知
│   ├── dingtalk.go        # 钉钉通知
│   └── slack.go           # Slack 通知
├── scheduler/             # 后台任务调度
│   └── scheduler.go       # 调度器
├── metrics/               # 指标服务
│   └── collector.go       # 指标收集器
└── performance/           # 性能监控
    └── monitor.go         # 性能监控器
```

**服务层职责**:
- 业务逻辑封装
- 事务管理
- 跨仓储操作协调
- 复杂业务规则实现

### 6. internal/install/ - 安装向导

```
internal/install/
├── handlers/              # 安装处理器
│   └── install_handler.go # 安装向导 API 处理
├── middleware/            # 安装中间件
│   └── install_middleware.go # 安装状态检查
├── models/                # 安装相关模型
│   └── install_config.go  # 安装配置
└── services/              # 安装服务
    └── install_service.go # 安装逻辑
```

### 7. pkg/ - 公共包

```
pkg/
├── handlers/              # Kubernetes 专用处理器
│   ├── resources/         # K8s 资源 CRUD
│   │   ├── routes.go      # 资源路由注册
│   │   ├── pod_handler.go
│   │   ├── deployment_handler.go
│   │   ├── service_handler.go
│   │   ├── configmap_handler.go
│   │   ├── secret_handler.go
│   │   ├── node_handler.go
│   │   ├── namespace_handler.go
│   │   ├── ingress_handler.go
│   │   ├── crd_handler.go
│   │   └── ... (更多资源)
│   ├── logs_handler.go    # 日志 WebSocket
│   ├── terminal_handler.go # Pod 终端 WebSocket
│   ├── node_terminal_handler.go # Node 终端 WebSocket
│   ├── search_handler.go  # 全局搜索
│   ├── overview_handler.go # 集群概览
│   ├── prom_handler.go    # Prometheus 指标
│   ├── resource_apply_handler.go # 资源应用
│   ├── user_handler.go    # 用户管理
│   └── webhook_handler.go # Webhook
├── kube/                  # Kubernetes 客户端工具
│   ├── client.go          # 客户端创建和缓存
│   ├── log.go             # 日志流处理
│   └── terminal.go        # 终端 exec 处理
├── cluster/               # 集群管理
│   ├── cluster_manager.go # 集群管理器
│   └── cluster_handler.go # 集群处理器
├── auth/                  # 认证工具
│   ├── handler.go         # 认证处理
│   ├── oauth_manager.go   # OAuth 管理
│   └── oauth_provider.go  # OAuth 提供商
├── middleware/            # HTTP 中间件
│   ├── cluster.go         # 集群中间件
│   ├── cors.go            # CORS 处理
│   ├── logger.go          # 日志记录
│   ├── metrics.go         # 指标收集
│   └── rbac.go            # RBAC 权限检查
├── prometheus/            # Prometheus 集成
│   └── client.go          # Prometheus 客户端
├── rbac/                  # RBAC 权限管理
│   ├── manager.go         # RBAC 管理器
│   └── rbac.go            # 权限检查逻辑
├── crypto/                # 加密工具
│   └── encryption.go      # 加密/解密函数
├── utils/                 # 工具函数
│   ├── pods.go            # Pod 工具
│   ├── search.go          # 搜索工具
│   ├── secure.go          # 安全工具
│   └── utils.go           # 通用工具
└── version/               # 版本管理
    ├── version.go         # 版本信息
    └── update_checker.go  # 更新检查
```

## 🎨 前端架构详解

### 1. ui/src/ - 前端源码结构

```
ui/src/
├── main.tsx               # 应用入口
├── App.tsx                # 根组件
├── routes.tsx             # 路由配置（所有页面路由定义）
├── vite-env.d.ts          # Vite 类型定义
├── index.css              # 全局样式
├── App.css                # 应用样式
├── pages/                 # 页面组件（57+ 个）
│   ├── login.tsx          # 登录页
│   ├── overview-dashboard-new.tsx # 主仪表板
│   ├── hosts.tsx          # 主机列表
│   ├── host-form.tsx      # 主机表单
│   ├── host-detail.tsx    # 主机详情
│   ├── alerts.tsx         # 告警管理
│   ├── users.tsx          # 用户管理
│   ├── roles.tsx          # 角色管理
│   ├── settings.tsx       # 系统设置
│   ├── pod-list-page.tsx  # Pod 列表
│   ├── pod-detail.tsx     # Pod 详情
│   ├── deployment-list-page.tsx # Deployment 列表
│   ├── deployment-detail.tsx # Deployment 详情
│   ├── resource-list.tsx  # 通用资源列表
│   ├── resource-detail.tsx # 通用资源详情
│   ├── minio-management.tsx # MinIO 管理
│   ├── database-management.tsx # 数据库管理
│   ├── docker-overview.tsx # Docker 概览
│   ├── middleware-overview.tsx # 中间件概览
│   ├── storage-overview.tsx # 存储概览
│   ├── webserver-overview.tsx # Web 服务器概览
│   ├── install/           # 安装向导页面
│   │   ├── index.tsx      # 安装主页
│   │   ├── components/    # 安装组件
│   │   └── steps/         # 安装步骤
│   │       ├── database-step.tsx
│   │       ├── admin-step.tsx
│   │       ├── settings-step.tsx
│   │       └── confirm-step.tsx
│   └── __tests__/         # 页面测试
├── components/            # UI 组件（130+ 个）
│   ├── ui/                # 基础 UI 组件（Radix UI + shadcn/ui）
│   │   ├── button.tsx
│   │   ├── card.tsx
│   │   ├── dialog.tsx
│   │   ├── input.tsx
│   │   ├── select.tsx
│   │   ├── table.tsx
│   │   ├── tabs.tsx
│   │   ├── sidebar.tsx
│   │   └── ... (30+ 个基础组件)
│   ├── app-sidebar.tsx    # 主应用侧边栏
│   ├── site-header.tsx    # 页面头部
│   ├── footer.tsx         # 页面底部
│   ├── user-menu.tsx      # 用户菜单
│   ├── mode-toggle.tsx    # 主题切换
│   ├── language-toggle.tsx # 语言切换
│   ├── cluster-selector.tsx # 集群选择器
│   ├── global-search.tsx  # 全局搜索
│   ├── yaml-editor.tsx    # YAML 编辑器
│   ├── terminal.tsx       # Web 终端
│   ├── log-viewer.tsx     # 日志查看器
│   ├── pod-table.tsx      # Pod 表格
│   ├── service-table.tsx  # Service 表格
│   ├── resource-table.tsx # 通用资源表格
│   ├── chart/             # 图表组件
│   │   ├── cpu-usage-chart.tsx
│   │   ├── memory-usage-chart.tsx
│   │   ├── disk-io-usage-chart.tsx
│   │   └── network-usage-chart.tsx
│   ├── editors/           # 编辑器组件
│   │   ├── deployment-create-dialog.tsx
│   │   ├── environment-editor.tsx
│   │   ├── image-editor.tsx
│   │   └── resource-editor.tsx
│   ├── settings/          # 设置组件
│   │   ├── cluster-management.tsx
│   │   ├── oauth-provider-management.tsx
│   │   ├── rbac-management.tsx
│   │   └── user-management.tsx
│   ├── selector/          # 选择器组件
│   │   ├── namespace-selector.tsx
│   │   ├── pod-selector.tsx
│   │   ├── configmap-selector.tsx
│   │   ├── secret-selector.tsx
│   │   └── crd-selector.tsx
│   ├── guards/            # 路由守卫
│   │   └── install-guard.tsx
│   └── __tests__/         # 组件测试
├── layouts/               # 布局组件（8 个子系统）
│   ├── devops-layout.tsx  # DevOps 子系统布局
│   ├── vms-layout.tsx     # VMs 子系统布局
│   ├── k8s-layout.tsx     # Kubernetes 子系统布局
│   ├── middleware-layout.tsx # 中间件子系统布局
│   ├── minio-layout.tsx   # MinIO 子系统布局
│   ├── docker-layout.tsx  # Docker 子系统布局
│   ├── storage-layout.tsx # 存储子系统布局
│   └── webserver-layout.tsx # Web 服务器子系统布局
├── services/              # API 客户端服务
│   ├── api.ts             # Axios 基础配置
│   ├── auth.ts            # 认证 API
│   ├── kubernetes.ts      # Kubernetes API
│   ├── clusters.ts        # 集群 API
│   ├── instances.ts       # 实例 API
│   ├── alerts.ts          # 告警 API
│   ├── audit.ts           # 审计 API
│   └── users.ts           # 用户 API
├── contexts/              # React 上下文
│   ├── auth-context.tsx   # 认证状态
│   ├── cluster-context.tsx # 集群状态
│   └── theme-context.tsx  # 主题状态
├── hooks/                 # 自定义 Hooks
│   ├── use-mobile.ts      # 移动端检测
│   ├── use-toast.ts       # Toast 通知
│   └── use-debounce.ts    # 防抖
├── lib/                   # 工具库
│   └── utils.ts           # 工具函数（cn 等）
├── types/                 # TypeScript 类型定义
│   ├── kubernetes.ts      # K8s 类型
│   ├── instance.ts        # 实例类型
│   ├── user.ts            # 用户类型
│   └── api.ts             # API 响应类型
├── i18n/                  # 国际化
│   ├── config.ts          # i18n 配置
│   ├── en.json            # 英文翻译
│   └── zh.json            # 中文翻译
├── styles/                # 样式文件
│   └── globals.css        # 全局样式
└── assets/                # 静态资源
    └── logo.png           # Logo 图片
```

### 2. 前端路由组织

```
/ (根路径)
├── /login                 # 登录页
├── /install               # 安装向导
├── / (主页)               # 概览仪表板
├── /vms                   # VMs 子系统
│   ├── /vms               # 主机列表
│   ├── /vms/new           # 新建主机
│   ├── /vms/:id           # 主机详情
│   ├── /vms/:id/edit      # 编辑主机
│   └── /vms/:id/metrics   # 主机监控
├── /devops                # DevOps 子系统
│   ├── /devops/alerts     # 告警管理
│   ├── /devops/users      # 用户管理
│   ├── /devops/roles      # 角色管理
│   └── /devops/settings   # 系统设置
├── /k8s                   # Kubernetes 子系统
│   ├── /k8s/overview      # 集群概览
│   ├── /k8s/:resource     # 资源列表（pods、deployments 等）
│   ├── /k8s/:resource/:name # 资源详情
│   ├── /k8s/:resource/:namespace/:name # 带命名空间的资源
│   └── /k8s/crds/:crd     # CRD 资源
├── /middleware            # 中间件子系统
│   ├── /middleware/mysql
│   ├── /middleware/postgresql
│   └── /middleware/redis
├── /minio/:instanceId     # MinIO 子系统
├── /database/:instanceId  # 数据库子系统
├── /docker                # Docker 子系统
├── /storage               # 存储子系统
└── /webserver             # Web 服务器子系统
```

## 📝 文件命名约定

### 后端 (Go)
- **文件名**: `snake_case.go`
  - 示例: `user_handler.go`, `cluster_manager.go`
- **测试文件**: `*_test.go`
  - 示例: `user_handler_test.go`
- **接口**: 通常以 `I` 开头或使用 `er` 结尾
  - 示例: `ServiceManager`, `Notifier`
- **结构体**: `PascalCase`
  - 示例: `UserRepository`, `AlertService`

### 前端 (TypeScript/React)
- **组件文件**: `kebab-case.tsx`
  - 示例: `cluster-selector.tsx`, `pod-table.tsx`
- **页面文件**: `kebab-case.tsx` 或直接文件名
  - 示例: `hosts.tsx`, `pod-list-page.tsx`
- **类型文件**: `kebab-case.ts`
  - 示例: `kubernetes.ts`, `api-types.ts`
- **服务文件**: `kebab-case.ts`
  - 示例: `auth-service.ts`, `api-client.ts`
- **组件名**: `PascalCase`
  - 示例: `ClusterSelector`, `PodTable`

### 配置和文档
- **配置文件**: `lowercase` 或 `kebab-case`
  - 示例: `config.yaml`, `docker-compose.yml`
- **文档**: `UPPERCASE.md` 或 `kebab-case.md`
  - 示例: `README.md`, `project-structure.md`
- **脚本**: `kebab-case.sh`
  - 示例: `generate-swagger.sh`, `install-hooks.sh`

## 🗂️ 关键目录说明

### /cmd/tiga/
- **用途**: 应用程序主入口
- **包含**: main.go（应用启动逻辑）
- **依赖**: internal 和 pkg 包

### /internal/
- **用途**: 私有应用代码，不应被外部项目导入
- **特点**: Go 编译器强制此规则
- **包含**: API、服务、模型、仓储等核心业务逻辑

### /pkg/
- **用途**: 可被外部项目导入的公共库
- **特点**: 可复用、通用的工具和功能
- **包含**: Kubernetes 工具、中间件、工具函数等

### /ui/
- **用途**: 前端 React 应用
- **构建**: Vite
- **输出**: 编译到 `/static` 目录

### /static/
- **用途**: 前端编译产物
- **嵌入**: 通过 `embed` 嵌入到 Go 二进制文件
- **提供**: 由 Gin 静态文件服务提供

### /tests/
- **用途**: 测试文件
- **包含**: 后端集成测试、E2E 测试
- **运行**: `task test` 或 `task test-integration`

### /docs/
- **用途**: 项目文档
- **包含**: 用户指南、配置说明、API 文档、截图等
- **格式**: Markdown

### /scripts/
- **用途**: 构建和部署脚本
- **包含**: Swagger 生成、版本管理、Chart 验证等
- **语言**: Bash shell 脚本

## 🔧 构建和配置文件

### 后端
- `go.mod` / `go.sum`: Go 模块依赖
- `Taskfile.yml`: Task 构建任务定义
- `Dockerfile`: Docker 镜像构建
- `config.yaml`: 主配置文件

### 前端
- `ui/package.json`: npm 依赖和脚本
- `ui/vite.config.ts`: Vite 构建配置
- `ui/tsconfig.json`: TypeScript 配置
- `ui/components.json`: shadcn/ui 组件配置
- `ui/eslint.config.js`: ESLint 配置
- `ui/prettier.config.cjs`: Prettier 配置

### 其他
- `.gitignore`: Git 忽略文件
- `LICENSE`: Apache 2.0 许可证
- `README.md`: 项目说明

## 📦 模块组织模式

### 1. 后端分层架构
```
HTTP 请求
    ↓
Middleware (认证、RBAC、审计)
    ↓
Handler (请求处理)
    ↓
Service (业务逻辑)
    ↓
Repository (数据访问)
    ↓
Database (数据存储)
```

### 2. 前端组件层次
```
App.tsx
    ↓
Routes (路由配置)
    ↓
Layouts (布局包装器)
    ↓
Pages (页面组件)
    ↓
Components (可复用组件)
    ↓
UI Components (基础 UI)
```

### 3. 仓储模式
- **仓储接口**: 定义数据访问方法
- **仓储实现**: 具体的数据库操作
- **服务层**: 调用仓储完成业务逻辑
- **优点**: 可测试、可替换、解耦

### 4. 管理器模式
- **管理器接口**: `ServiceManager`
- **具体管理器**: MinIO、MySQL、Redis 等
- **协调器**: `ManagerCoordinator` 统一管理所有实例
- **优点**: 统一接口、易于扩展、职责清晰

## 🎯 快速定位指南

### 找前端页面
- 查看 `ui/src/routes.tsx` 获取路由配置
- 页面文件在 `ui/src/pages/`

### 找 API 端点
- 查看 `internal/api/routes.go` 获取所有路由
- 处理器在 `internal/api/handlers/` 和 `pkg/handlers/`

### 找数据模型
- 查看 `internal/models/` 获取所有 GORM 模型

### 找业务逻辑
- 查看 `internal/services/` 获取业务服务

### 找 Kubernetes 操作
- 查看 `pkg/handlers/resources/` 获取资源 CRUD
- 查看 `pkg/kube/` 获取 K8s 客户端工具

### 找 UI 组件
- 基础组件: `ui/src/components/ui/`
- 业务组件: `ui/src/components/`
- 布局组件: `ui/src/layouts/`
