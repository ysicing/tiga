# tiga - 统一的 DevOps 运维平台

<div align="center">

<img src="./docs/assets/logo.svg" alt="tiga Logo" width="128" height="128">

_一个现代化、全栈、统一的 DevOps 运维管理平台_

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![React](https://img.shields.io/badge/React-19+-61DAFB?style=flat&logo=react)](https://reactjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5+-3178C6?style=flat&logo=typescript)](https://www.typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache-green.svg)](LICENSE)

[English](./README_EN.md) | **中文**

</div>

**tiga** 是一个企业级的统一 DevOps 运维平台，将 Kubernetes 集群管理、主机监控、数据库管理、容器编排、对象存储等多个运维子系统整合到一个优美的 Web 界面中。告别多个工具切换的烦恼，用一个平台管理你的整个基础设施。

> [!WARNING]
> 本项目正在快速迭代开发中，使用方式和 API 都有可能变化。

---

## 🌟 为什么选择 tiga？

### 🎯 **统一的运维体验**
不再需要在 kubectl、数据库客户端、SSH 终端、Docker CLI 之间来回切换。tiga 将所有运维工具整合到一个现代化的 Web 界面中，提供一致的操作体验。

### 🚀 **开箱即用，快速部署**
- **5分钟部署**：Helm 一键安装，或单个 Docker 容器运行
- **零配置集群发现**：自动从 `~/.kube/config` 导入 Kubernetes 集群
- **自动化服务发现**：自动检测集群内的 Prometheus、MinIO 等服务

### 🔐 **企业级安全**
- **多租户权限控制**：基于 RBAC 的细粒度权限管理
- **审计日志**：记录所有操作，满足合规要求
- **加密存储**：AES-256-GCM 加密敏感数据（密码、Token）
- **OAuth 集成**：支持 Google、GitHub 第三方登录

### 💡 **智能与自动化**
- **Prometheus 自动发现**：自动检测集群内的监控服务
- **智能告警**：基于表达式引擎的灵活告警规则
- **自动清理**：定时清理过期日志、录制文件
- **健康检查**：实时监控集群、主机、数据库健康状态

---

## 🏗️ 核心子系统

### ☸️ **Kubernetes 集群管理**

统一管理多个 Kubernetes 集群，提供比 kubectl 更友好的操作界面。

**核心功能**：
- 🌐 **多集群统一管理** - 在多个集群之间无缝切换，统一视图管理
- 🔍 **全局搜索** - 跨集群、跨资源类型的智能搜索（Pod、Deployment、Service）
- 📊 **资源关系图** - 可视化资源依赖关系（Deployment → ReplicaSet → Pod）
- 🏷️ **CRD 原生支持** - 完整支持 OpenKruise、Tailscale、Traefik、K3s Upgrade Controller
- 💻 **节点终端** - 浏览器内 SSH 进入 K8s 节点（通过特权 Pod）
- 📈 **Prometheus 集成** - 自动发现并查询集群监控数据
- 🛡️ **只读模式** - 系统级只读开关，保护生产环境

**特色亮点**：
- 📝 **Monaco 编辑器** - 实时编辑 YAML，语法高亮和校验
- 🎯 **智能评分搜索** - 精确匹配 100 分、名称包含 80 分、标签匹配 60 分
- 🔄 **5 分钟缓存** - ResourceVersion 感知的工作负载缓存
- 🔐 **细粒度 RBAC** - 集群级、命名空间级、资源级权限控制

---

### 🖥️ **主机管理（VMs）**

基于 Agent-Server 架构的轻量级主机监控系统，无需安装重量级监控套件。

**核心功能**：
- 📊 **实时监控** - CPU、内存、磁盘、网络等指标实时采集和展示
- 🔍 **服务探测** - HTTP/TCP/ICMP 健康检查，Agent 端分布式执行
- 💻 **WebSSH 终端** - 浏览器内 SSH 终端，通过 Agent 代理安全访问
- 🔔 **告警系统** - 表达式引擎驱动的灵活告警规则
- 👥 **主机分组** - 按环境、机房、业务线分组管理
- 📈 **历史数据** - 时序数据存储，支持趋势分析

**特色亮点**：
- ⚡ **轻量级 Agent** - Go 独立二进制，占用资源 < 20MB 内存
- 🔐 **gRPC 双向流** - Agent 主动连接，穿透 NAT/防火墙
- 🎨 **Recharts 图表** - 交互式监控图表，支持缩放和时间范围选择
- 🔄 **systemd 管理** - Agent 自动注册为系统服务

---

### 🗄️ **数据库管理**

统一管理 MySQL、PostgreSQL、Redis 数据库实例，提供 Web 查询控制台和权限管理。

**核心功能**：
- 🎯 **多数据库支持** - MySQL 8.0+、PostgreSQL 15+、Redis 7+ 统一管理
- 💻 **查询控制台** - Monaco Editor 编辑器，支持 SQL/Redis 命令执行
- 🔐 **权限管理** - 统一权限模型（readonly/readwrite）自动映射到不同数据库
- 👥 **用户管理** - 创建用户、修改密码、授权数据库
- 📋 **审计日志** - 记录所有查询和操作，支持分页和筛选
- 🔌 **连接池** - 自动管理数据库连接（50 open / 10 idle）

**安全特性**：
- 🛡️ **SQL 安全过滤** - 阻止 DDL 操作、无 WHERE 的 UPDATE/DELETE、危险函数
- 🚫 **Redis 命令黑名单** - 禁止 FLUSHDB、FLUSHALL、SHUTDOWN、CONFIG 等危险命令
- 🔒 **AES-256-GCM 加密** - 密码、Token 加密存储
- ⏱️ **查询限制** - 30 秒超时、10MB 结果大小限制

**特色亮点**：
- 🎨 **语法高亮** - 支持 SQL、Redis 命令语法高亮和自动补全
- 📊 **虚拟滚动** - react-window 实现大数据集流畅展示
- 🧹 **自动清理** - 定时删除 90 天前的审计日志
- ⚡ **性能优化** - SQL 安全过滤 <2ms、Redis 过滤 <100μs

---

### 🐳 **Docker 管理**

通过 Agent 远程管理 Docker 实例，支持 NAT 穿透，无需暴露 Docker API。

**核心功能**：
- 📦 **容器管理** - 启动、停止、重启、暂停、删除容器
- 🖼️ **镜像管理** - 拉取、删除、标签镜像，支持拉取进度流
- 🌐 **网络管理** - 创建、删除、连接、断开 Docker 网络
- 💾 **卷管理** - 创建、删除、清理未使用的卷
- 💻 **容器终端** - 浏览器内 Exec 进入容器（双向流）
- 📊 **容器统计** - 实时 CPU、内存、网络、I/O 统计（JSON 流）
- 📝 **容器日志** - Server-Sent Events (SSE) 实时流式日志

**架构亮点**：
- 🔄 **双模式架构** - 非流式操作用任务队列、流式操作用 DockerStream
- 🔐 **NAT 友好** - Agent 主动连接 Server，无需公网 IP
- 🎯 **统一 Agent** - 与主机监控共用同一个 Agent 进程
- 📡 **gRPC 流** - 高效的二进制流传输

---

### 📦 **MinIO 对象存储管理**

管理 MinIO 实例，提供 Bucket、用户、策略配置的 Web 界面。

**核心功能**：
- 🪣 **Bucket 管理** - 创建、删除、配置 Bucket
- 👥 **用户管理** - 创建用户、生成访问密钥
- 🔐 **策略管理** - 配置访问策略、权限控制
- 📊 **实例监控** - 存储容量、对象数量、请求统计

---

### 🌐 **Web 服务器管理（Caddy）**

管理 Caddy Web 服务器和反向代理配置。

**核心功能**：
- 🔄 **反向代理** - 配置反向代理规则
- 🔒 **自动 HTTPS** - Caddy 自动申请和续期 SSL 证书
- 📝 **配置管理** - 可视化编辑 Caddyfile

---

### ⚙️ **系统管理**

统一的系统配置和管理中心。

**核心功能**：
- 👥 **用户管理** - 用户创建、角色分配、权限管理
- 🔐 **OAuth 配置** - Google、GitHub 第三方登录配置
- 📋 **审计日志** - 系统级操作审计，支持高级筛选
- ⏰ **定时任务** - Cron 任务调度和管理
- 🌍 **全局配置** - 集群配置、功能开关、系统设置

---

## 🎨 现代化的用户体验

### 🌓 **多主题支持**
- 暗色主题、亮色主题、彩色主题
- 自动适应系统主题偏好
- 无缝切换，保存用户偏好

### 🌐 **国际化支持**
- 支持中文、英文双语切换
- 自动检测浏览器语言
- 覆盖所有 UI 文本和提示

### 📱 **响应式设计**
- 桌面端优化布局
- 平板适配
- 移动端友好（部分功能）

### 🎯 **交互优化**
- 快捷键支持（搜索、导航）
- 智能表单验证
- 实时反馈和提示
- Loading 状态优化

---

## 🏗️ 技术架构

### 后端技术栈
- **语言**：Go 1.24+
- **框架**：Gin (HTTP)、gRPC (Agent 通信)
- **数据库**：SQLite / PostgreSQL / MySQL（GORM ORM）
- **认证**：JWT + OAuth 2.0（Google、GitHub）
- **监控**：Prometheus 集成
- **客户端**：Kubernetes client-go、Docker SDK、各数据库官方驱动

### 前端技术栈
- **框架**：React 19 + TypeScript 5
- **构建工具**：Vite
- **UI 组件**：Radix UI + TailwindCSS
- **状态管理**：TanStack Query（服务端状态）、React Context（全局状态）
- **编辑器**：Monaco Editor（YAML、SQL）
- **图表**：Recharts（监控图表）
- **终端**：xterm.js（浏览器终端）

### 架构特点
- ✅ **Repository 模式** - 数据访问层抽象，易于测试和切换存储
- ✅ **Manager 模式** - 统一的服务管理器接口（ServiceManager）
- ✅ **中间件栈** - CORS → Logger → Auth → RBAC → Rate Limit → Audit
- ✅ **双配置系统** - YAML 配置文件 + 环境变量（YAML 优先）
- ✅ **Agent-Server 架构** - 主机监控和 Docker 管理采用 Agent 模式
- ✅ **审计日志** - 统一的 AuditEvent 系统，记录所有操作

---

## 🚀 快速开始

### 方式一：Docker 运行（推荐体验）

最简单的方式，适合快速体验：

```bash
docker run --rm -p 12306:12306 ghcr.io/ysicing/tiga:latest
```

访问 http://localhost:12306 开始使用。

### 方式二：Kubernetes 部署（推荐生产）

#### 使用 Helm

```bash
# 添加 Helm 仓库
helm repo add tiga https://ysicing.github.io/tiga
helm repo update

# 安装到 kube-system 命名空间
helm install tiga tiga/tiga -n kube-system

# 通过端口转发访问
kubectl port-forward -n kube-system svc/tiga 12306:12306
```

#### 使用 kubectl

```bash
# 在线安装
kubectl apply -f https://raw.githubusercontent.com/ysicing/tiga/main/deploy/install.yaml

# 或本地安装
kubectl apply -f deploy/install.yaml

# 通过端口转发访问
kubectl port-forward -n kube-system svc/tiga 12306:12306
```

### 方式三：从源码构建

适合开发者和需要定制的场景：

```bash
# 克隆仓库
git clone https://github.com/ysicing/tiga.git
cd tiga

# 安装依赖（需要 Go 1.24+、Node.js 18+）
task deps

# 构建前端 + 后端
task build

# 运行
./bin/tiga

# 或开发模式（热重载）
task dev              # 完整开发（前端 + 后端）
task dev:backend      # 仅后端
task dev:frontend     # 仅前端（Vite 热重载）
```

### 初始化设置

首次访问会进入安装向导，引导你完成：

1. **数据库配置** - 选择 SQLite（默认）、PostgreSQL 或 MySQL
2. **管理员账户** - 创建第一个管理员用户
3. **JWT 密钥** - 自动生成或手动指定
4. **集群导入** - 自动从 `~/.kube/config` 导入集群（可选）

完成后，配置保存到 `config.yaml`，并设置 `install_lock: true` 防止重复初始化。

---

## 📊 功能演示

### Kubernetes 集群管理
![K8s Cluster List](docs/screenshots/k8s-clusters.png)
_多集群统一管理，实时健康状态_

![K8s Global Search](docs/screenshots/k8s-search.png)
_全局搜索，智能评分排序_

### 主机监控
![Host Monitor](docs/screenshots/host-monitor.png)
_实时监控主机性能指标_

![WebSSH Terminal](docs/screenshots/webssh.png)
_浏览器内 SSH 终端_

### 数据库管理
![Database Query Console](docs/screenshots/db-query.png)
_Monaco 编辑器，SQL 语法高亮_

![Database Permissions](docs/screenshots/db-permissions.png)
_统一的权限管理_

### Docker 管理
![Docker Containers](docs/screenshots/docker-containers.png)
_容器列表和实时统计_

---

## 🗺️ 发展路线

### ✅ 已完成
- [x] Kubernetes 多集群管理（CRD 支持、全局搜索、节点终端）
- [x] 数据库管理（MySQL、PostgreSQL、Redis 查询控制台）
- [x] Docker Agent 架构优化（双模式：任务队列 + 流式操作）
- [x] 统一终端录制系统（Asciinema 格式，统一存储和清理）
- [x] 审计日志系统（统一 AuditEvent，记录所有操作）

### 🚧 进行中
- [ ] 主机管理完整实现（Agent 连接、服务探测、WebSSH）
- [ ] MinIO 对象存储管理
- [ ] Caddy Web 服务器管理
- [ ] 告警系统增强（多通知渠道：邮件、Webhook、企业微信）

### 📋 计划中
- [ ] 多租户支持（租户隔离、配额管理）
- [ ] 成本分析（云资源成本统计和优化建议）
- [ ] CI/CD 集成（GitOps、Pipeline 管理）
- [ ] 备份管理（数据库备份、K8s 资源备份）
- [ ] 日志聚合（集中式日志收集和检索）

---

## 🔧 配置说明

### 核心配置文件

tiga 使用 `config.yaml` 作为主配置文件，支持环境变量覆盖。

```yaml
# 服务器配置
server:
  port: 12306
  mode: release  # debug | release

# 数据库配置
database:
  type: sqlite          # sqlite | postgres | mysql
  host: localhost
  port: 5432
  database: tiga
  username: tiga
  password: ""

# JWT 认证
jwt:
  secret: "your-secret-key"
  expire_hours: 24
  refresh_expire_hours: 168

# Kubernetes 配置
kubernetes:
  node_terminal_image: "alpine:latest"
  enable_kruise: true
  enable_tailscale: true
  enable_traefik: true

# 功能特性开关
features:
  readonly_mode: false
  anonymous_user_enabled: false
  disable_version_check: false

# 录制配置
recording:
  storage_type: local
  base_path: ./data/recordings
  retention_days: 90
  cleanup_schedule: "0 4 * * *"
```

### 环境变量

所有配置项都可以通过环境变量覆盖，格式为 `TIGA_<SECTION>_<KEY>`：

```bash
export TIGA_SERVER_PORT=8080
export TIGA_DATABASE_TYPE=postgres
export TIGA_DATABASE_HOST=db.example.com
export TIGA_JWT_SECRET=my-secret-key
```

---

## 📖 文档

- 📘 [完整文档](https://tiga.ysicing.com)（建设中）
- 🏗️ [架构设计](./docs/architecture/)
- 🔧 [开发指南](./CLAUDE.md)
- 📝 [API 文档](http://localhost:12306/swagger/index.html)（启动后访问）
- 🐛 [问题追踪](https://github.com/ysicing/tiga/issues)

---

## 🤝 贡献

我们欢迎所有形式的贡献！

### 贡献方式
- 🐛 **报告 Bug** - 提交 Issue 描述问题
- 💡 **功能建议** - 提交 Issue 说明需求
- 📝 **改进文档** - 修正错误、补充说明
- 🔨 **提交代码** - Fork 项目，提交 Pull Request

### 开发流程
```bash
# Fork 项目并克隆
git clone https://github.com/YOUR_USERNAME/tiga.git
cd tiga

# 创建功能分支
git checkout -b feature/my-feature

# 开发和测试
task dev              # 启动开发服务器
task test             # 运行单元测试
task lint             # 代码检查

# 提交代码
git commit -m "feat: add my feature"
git push origin feature/my-feature

# 创建 Pull Request
```

### 代码规范
- Go 代码遵循 `gofmt` 和 `golangci-lint` 规范
- TypeScript 代码遵循 `eslint` 和 `prettier` 规范
- Commit 消息遵循 [Conventional Commits](https://www.conventionalcommits.org/)

---

## 📄 许可证

本项目采用 **Apache License 2.0** 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

## 🌟 Star 历史

[![Star History Chart](https://api.star-history.com/svg?repos=ysicing/tiga&type=Date)](https://star-history.com/#ysicing/tiga&Date)

---

## 📮 联系方式

- **作者**：ysicing
- **项目主页**：https://github.com/ysicing/tiga
- **问题反馈**：https://github.com/ysicing/tiga/issues

---

<div align="center">

**如果觉得有帮助，请给个 ⭐ Star 支持一下！**

Made with ❤️ by [ysicing](https://github.com/ysicing)

</div>
