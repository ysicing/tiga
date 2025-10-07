# 📂 Tiga DevOps Dashboard 功能-代码映射报告

## 🏗️ 项目概览

- **技术栈**: Go 1.24+ (后端) + React 19 + TypeScript (前端)
- **后端框架**: Gin + GORM + Kubernetes client-go
- **前端框架**: Vite + TailwindCSS + Radix UI + TanStack Query
- **架构模式**:
  - 后端：仓储模式 (Repository Pattern) + 管理器模式 (Manager Pattern)
  - 前端：组件化 + 多子系统布局 (Layout-based Architecture)
- **状态管理**: React Context + TanStack Query
- **样式方案**: TailwindCSS + Radix UI
- **构建工具**: Vite (前端) + Go build (后端)
- **包管理**: pnpm (前端) + go mod (后端)
- **数据库**: SQLite/PostgreSQL/MySQL

## 📊 功能模块统计

- **页面级组件**: 57 个 (含安装向导、登录、仪表板等)
- **可复用组件**: 130+ 个 (UI 组件、业务组件、图表等)
- **后端 API 处理器**: 19 个主要处理器 + 子模块
- **子系统数量**: 8 个 (DevOps、VMs、K8s、Middleware、MinIO、Docker、Storage、WebServer)
- **API 路由组**: 10+ 个功能组

## 🗂️ 目录结构概览

```
tiga/
├── cmd/tiga/              # 主应用入口
├── internal/              # 后端核心代码
│   ├── api/              # API 层
│   │   ├── handlers/    # 请求处理器
│   │   ├── middleware/  # 认证、RBAC、审计
│   │   └── routes.go    # 路由定义
│   ├── models/          # GORM 数据模型
│   ├── repository/      # 数据访问层
│   ├── services/        # 业务逻辑层
│   └── install/         # Web 安装向导
├── pkg/                  # 可复用包
│   ├── handlers/        # K8s 资源处理器
│   ├── kube/           # K8s 客户端工具
│   ├── auth/           # 认证工具
│   └── middleware/     # HTTP 中间件
├── ui/src/              # 前端源码
│   ├── pages/          # 页面组件
│   ├── components/     # UI 组件
│   ├── layouts/        # 布局包装器
│   ├── services/       # API 客户端
│   ├── contexts/       # React 上下文
│   └── hooks/          # 自定义 hooks
└── static/             # 前端构建产物
```

---

## 🎯 功能映射表

### 🔐 认证与登录 - 用户登录

**🔤 用户描述方式**:
- 主要: "登录", "登录页", "用户登录", "密码登录"
- 别名: "登录界面", "登录入口", "login", "sign in"

**📍 代码位置**:
- 前端页面: `ui/src/pages/login.tsx` - 登录表单和 OAuth 按钮
- 认证服务: `internal/services/auth/login_service.go` - 登录业务逻辑
- API 路由: `/api/auth/login/password` - 密码登录端点
- 处理器: `internal/api/handlers/auth_handler.go` - 认证处理器

**🎨 视觉标识**:
- 外观: 居中卡片式登录表单，带应用 logo
- 文本: "登录到 Tiga"、"用户名"、"密码"、"登录" 按钮
- OAuth: Google、GitHub 登录按钮（如已配置）

**⚡ 修改指引**:
- 修改登录表单样式: 编辑 `ui/src/pages/login.tsx`
- 修改登录逻辑: 编辑 `internal/services/auth/login_service.go`
- 添加 OAuth 提供商: 在后台系统设置中配置，或编辑 `pkg/auth/oauth_provider.go`

---

### 🏠 总览仪表板 - 主页大屏

**🔤 用户描述方式**:
- 主要: "首页", "主页", "总览", "仪表板", "大屏"
- 别名: "概览页", "dashboard", "overview", "主控制台"

**📍 代码位置**:
- 前端页面: `ui/src/pages/overview-dashboard-new.tsx` - 主仪表板
- 子系统卡片: `ui/src/components/subsystem-card.tsx` - 可点击的子系统入口卡片
- 路由: `/` - 根路径

**🎨 视觉标识**:
- 外观: 网格布局，8 个彩色子系统卡片（蓝、靛、绿、粉、紫、青、蓝绿、橙）
- 文本: "DevOps"、"主机管理"、"中间件"、"MinIO"、"Kubernetes"、"Docker"、"Web 服务器"、"监控告警"
- 图标: 每个子系统有对应图标（设置、服务器、数据库、存储桶、云、容器、WWW、铃铛）

**⚡ 修改指引**:
- 修改子系统布局: 编辑 `ui/src/pages/overview-dashboard-new.tsx` 第 23-88 行的 `SUBSYSTEMS` 数组
- 修改卡片样式: 编辑 `ui/src/components/subsystem-card.tsx`
- 添加新子系统: 在 `SUBSYSTEMS` 数组添加新项，并在 `ui/src/routes.tsx` 添加对应路由

---

### 🖥️ 主机管理 - 虚拟机列表

**🔤 用户描述方式**:
- 主要: "主机列表", "虚拟机管理", "服务器列表", "主机管理"
- 别名: "VMs", "hosts", "物理机", "云主机"

**📍 代码位置**:
- 前端页面: `ui/src/pages/hosts.tsx` - 主机列表页面
- 布局: `ui/src/layouts/vms-layout.tsx` - VMs 子系统布局
- API 端点: `/api/v1/instances?type=vm` - 获取主机实例列表
- 后端处理器: `internal/api/handlers/instance_handler.go` - 实例管理

**🎨 视觉标识**:
- 外观: 带侧边栏的表格视图，显示主机名、IP、状态等
- 文本: "主机管理"、"添加主机"、"状态：运行中/已停止"
- 按钮: "新建主机" (右上角蓝色按钮)

**⚡ 修改指引**:
- 修改表格列: 编辑 `ui/src/pages/hosts.tsx` 中的表格定义
- 修改侧边栏菜单: 编辑 `ui/src/components/vms-sidebar.tsx`
- 修改后端逻辑: 编辑 `internal/services/instance_service.go`

---

### ➕ 新建主机 - 主机创建表单

**🔤 用户描述方式**:
- 主要: "新建主机", "添加服务器", "创建虚拟机", "添加主机"
- 别名: "新增主机", "主机表单", "create host", "add server"

**📍 代码位置**:
- 前端页面: `ui/src/pages/host-form.tsx` - 主机创建/编辑表单
- 路由: `/vms/new` (新建) 或 `/vms/:id/edit` (编辑)
- API 端点: `POST /api/v1/instances` - 创建实例
- 后端处理器: `internal/api/handlers/instance_handler.go:CreateInstance`

**🎨 视觉标识**:
- 外观: 表单页面，包含主机名、类型、连接信息等输入字段
- 文本: "新建主机"、"保存"、"取消"
- 字段: 主机名、实例类型、连接地址、端口、认证信息

**⚡ 修改指引**:
- 修改表单字段: 编辑 `ui/src/pages/host-form.tsx`
- 修改验证规则: 编辑表单组件中的 validation schema
- 修改后端创建逻辑: 编辑 `internal/api/handlers/instance_handler.go`

---

### 📊 主机详情 - 主机监控与管理

**🔤 用户描述方式**:
- 主要: "主机详情", "服务器详情", "查看主机", "主机信息"
- 别名: "主机监控", "host details", "server info"

**📍 代码位置**:
- 前端页面: `ui/src/pages/host-detail.tsx` - 主机详情页
- 指标页面: `ui/src/pages/host-metrics.tsx` - 主机监控图表
- API 端点: `GET /api/v1/instances/:id` - 获取实例详情
- 健康检查: `GET /api/v1/instances/:id/health` - 实例健康状态

**🎨 视觉标识**:
- 外观: Tab 切换视图，显示基本信息、监控图表、操作日志等
- 文本: "基本信息"、"监控"、"操作"、CPU、内存、磁盘、网络
- 图表: 实时资源使用率折线图/面积图

**⚡ 修改指引**:
- 修改详情展示: 编辑 `ui/src/pages/host-detail.tsx`
- 修改监控图表: 编辑 `ui/src/pages/host-metrics.tsx` 和 `ui/src/components/chart/` 目录下的图表组件
- 修改后端健康检查: 编辑 `internal/api/handlers/instances/health.go`

---

### ☸️ Kubernetes - 集群选择器

**🔤 用户描述方式**:
- 主要: "集群选择", "切换集群", "选择 K8s 集群", "集群切换器"
- 别名: "cluster selector", "集群下拉框", "选择环境"

**📍 代码位置**:
- 前端组件: `ui/src/components/cluster-selector.tsx` - 集群选择下拉组件
- 上下文: `ui/src/contexts/cluster-context.tsx` - 集群状态管理
- API 端点: `GET /api/v1/clusters` - 获取集群列表
- 后端管理器: `pkg/cluster/cluster_manager.go` - 集群管理逻辑

**🎨 视觉标识**:
- 外观: K8s 布局右上角的下拉选择框，显示当前集群名称
- 文本: 集群名称列表（如 "docker-desktop"、"production"）
- 图标: 云图标 + 下拉箭头

**⚡ 修改指引**:
- 修改选择器样式: 编辑 `ui/src/components/cluster-selector.tsx`
- 修改集群管理: 编辑 `pkg/cluster/cluster_manager.go`
- 添加集群导入: 在系统设置 → 集群管理中操作

---

### 📦 Kubernetes - Pods 列表

**🔤 用户描述方式**:
- 主要: "Pod 列表", "容器组", "查看 Pods", "Pod 管理"
- 别名: "pods", "容器列表", "工作负载"

**📍 代码位置**:
- 前端页面: `ui/src/pages/resource-list.tsx` - 通用资源列表（含 Pods）
- Pod 表格: `ui/src/components/pod-table.tsx` - Pod 专用表格组件
- API 端点: `GET /api/v1/cluster/:clusterid/pods` - 获取 Pod 列表
- 后端处理器: `pkg/handlers/resources/pod_handler.go` - Pod CRUD 操作

**🎨 视觉标识**:
- 外观: 表格视图，显示 Pod 名称、命名空间、状态、重启次数、IP 等
- 文本: "Pods"、"Running"、"Pending"、"Failed"
- 状态图标: 绿色对勾(Running)、黄色圆圈(Pending)、红色叉(Failed)

**⚡ 修改指引**:
- 修改 Pod 表格列: 编辑 `ui/src/components/pod-table.tsx`
- 修改后端 Pod 查询: 编辑 `pkg/handlers/resources/pod_handler.go`
- 修改命名空间过滤: 使用页面上的命名空间选择器组件

---

### 🔍 Kubernetes - Pod 详情

**🔤 用户描述方式**:
- 主要: "Pod 详情", "查看 Pod", "容器详情", "Pod 信息"
- 别名: "pod detail", "容器组详情", "工作负载详情"

**📍 代码位置**:
- 前端页面: `ui/src/pages/pod-detail.tsx` 或 `ui/src/pages/resource-detail.tsx` - Pod 详情页
- 容器表格: `ui/src/components/container-table.tsx` - 容器信息展示
- 事件表格: `ui/src/components/event-table.tsx` - Pod 事件列表
- API 端点: `GET /api/v1/cluster/:clusterid/pods/:namespace/:name` - 获取 Pod 详情

**🎨 视觉标识**:
- 外观: Tab 视图，包含概览、YAML、事件、日志、终端等 Tab
- 文本: "概览"、"YAML"、"日志"、"终端"、"容器"
- 操作按钮: "查看日志"、"进入终端"、"编辑"、"删除"

**⚡ 修改指引**:
- 修改 Pod 详情页面: 编辑 `ui/src/pages/pod-detail.tsx`
- 修改容器信息展示: 编辑 `ui/src/components/container-table.tsx`
- 修改 YAML 查看器: 编辑 `ui/src/components/yaml-editor.tsx`

---

### 📝 Kubernetes - Pod 日志查看

**🔤 用户描述方式**:
- 主要: "Pod 日志", "容器日志", "查看日志", "日志查看器"
- 别名: "logs", "日志流", "实时日志", "容器输出"

**📍 代码位置**:
- 前端组件: `ui/src/components/log-viewer.tsx` - 日志查看器组件
- WebSocket 处理: `pkg/handlers/logs_handler.go` - 日志流 WebSocket
- API 端点: `GET /api/v1/cluster/:clusterid/logs/:namespace/:podName/ws` - WebSocket 日志流
- K8s 日志工具: `pkg/kube/log.go` - Kubernetes 日志流封装

**🎨 视觉标识**:
- 外观: 黑色终端风格的日志输出框，实时滚动
- 文本: 日志内容、时间戳、日志级别
- 控制: 容器选择下拉框、"跟随日志"开关、"清空"按钮

**⚡ 修改指引**:
- 修改日志查看器 UI: 编辑 `ui/src/components/log-viewer.tsx`
- 修改日志流逻辑: 编辑 `pkg/handlers/logs_handler.go`
- 修改日志过滤: 添加前端过滤逻辑或后端查询参数

---

### 💻 Kubernetes - Web 终端

**🔤 用户描述方式**:
- 主要: "Web 终端", "Pod 终端", "容器终端", "在线终端"
- 别名: "shell", "exec", "web shell", "命令行"

**📍 代码位置**:
- 前端组件: `ui/src/components/terminal.tsx` - xterm.js 终端组件
- WebSocket 处理: `pkg/handlers/terminal_handler.go` - Pod 终端 WebSocket
- 节点终端: `pkg/handlers/node_terminal_handler.go` - 节点终端 WebSocket
- API 端点: `GET /api/v1/cluster/:clusterid/terminal/:namespace/:podName/ws` - Pod 终端
- K8s 终端工具: `pkg/kube/terminal.go` - Kubernetes exec 封装

**🎨 视觉标识**:
- 外观: 黑色终端窗口，支持输入和输出，类似 SSH 客户端
- 文本: Shell 提示符（如 `root@pod-name:/#`）
- 交互: 支持键盘输入、Tab 补全、Ctrl+C 等操作

**⚡ 修改指引**:
- 修改终端 UI: 编辑 `ui/src/components/terminal.tsx`
- 修改终端连接逻辑: 编辑 `pkg/handlers/terminal_handler.go`
- 修改 xterm 配置: 在 `terminal.tsx` 中调整 xterm 选项

---

### 📄 Kubernetes - YAML 编辑器

**🔤 用户描述方式**:
- 主要: "YAML 编辑", "编辑配置", "修改 YAML", "配置编辑器"
- 别名: "yaml editor", "资源编辑", "在线编辑", "代码编辑器"

**📍 代码位置**:
- 前端组件: `ui/src/components/yaml-editor.tsx` - Monaco Editor YAML 编辑器
- 简化编辑器: `ui/src/components/simple-yaml-editor.tsx` - 简单 YAML 编辑器
- 差异查看器: `ui/src/components/yaml-diff-viewer.tsx` - YAML 差异对比
- API 端点: `POST /api/v1/cluster/:clusterid/resources/apply` - 应用 YAML 更改
- 后端处理器: `pkg/handlers/resource_apply_handler.go` - 资源应用处理

**🎨 视觉标识**:
- 外观: Monaco Editor（VS Code 编辑器内核），语法高亮、自动补全
- 文本: YAML 格式的 Kubernetes 资源定义
- 按钮: "保存"、"取消"、"格式化"

**⚡ 修改指引**:
- 修改编辑器配置: 编辑 `ui/src/components/yaml-editor.tsx` 中的 Monaco 配置
- 修改保存逻辑: 编辑 `pkg/handlers/resource_apply_handler.go`
- 添加验证: 在编辑器组件中添加 YAML 验证逻辑

---

### 🔎 Kubernetes - 全局搜索

**🔤 用户描述方式**:
- 主要: "全局搜索", "资源搜索", "搜索框", "查找资源"
- 别名: "search", "快速查找", "资源查询", "Kubernetes 搜索"

**📍 代码位置**:
- 前端组件: `ui/src/components/global-search.tsx` - 全局搜索弹窗
- 搜索提供者: `ui/src/components/global-search-provider.tsx` - 搜索状态管理
- API 端点: `GET /api/v1/cluster/:clusterid/search?q=xxx` - 全局搜索
- 后端处理器: `pkg/handlers/search_handler.go` - 搜索逻辑
- 工具函数: `pkg/utils/search.go` - 搜索算法

**🎨 视觉标识**:
- 外观: Cmd+K 快捷键触发的命令面板样式搜索框
- 文本: "搜索资源..."、搜索结果列表（资源类型、名称、命名空间）
- 快捷键: 显示 "⌘K" 或 "Ctrl+K" 提示

**⚡ 修改指引**:
- 修改搜索 UI: 编辑 `ui/src/components/global-search.tsx`
- 修改搜索算法: 编辑 `pkg/utils/search.go` 中的模糊匹配逻辑
- 修改搜索范围: 编辑 `pkg/handlers/search_handler.go` 中的资源类型列表

---

### 🚀 Kubernetes - Deployments 列表

**🔤 用户描述方式**:
- 主要: "Deployment 列表", "部署列表", "应用部署", "工作负载"
- 别名: "deployments", "应用管理", "服务部署"

**📍 代码位置**:
- 前端页面: `ui/src/pages/deployment-list-page.tsx` - Deployment 列表页
- 资源表格: `ui/src/components/resource-table.tsx` - 通用资源表格
- API 端点: `GET /api/v1/cluster/:clusterid/deployments` - 获取 Deployment 列表
- 后端处理器: `pkg/handlers/resources/deployment_handler.go` - Deployment 操作

**🎨 视觉标识**:
- 外观: 表格视图，显示 Deployment 名称、副本数、镜像、状态等
- 文本: "Deployments"、"Ready: 3/3"、"Up-to-date"、"Available"
- 状态: 绿色对勾（健康）、黄色感叹号（部分就绪）

**⚡ 修改指引**:
- 修改列表页面: 编辑 `ui/src/pages/deployment-list-page.tsx`
- 修改表格显示: 编辑 `ui/src/components/resource-table.tsx`
- 修改后端查询: 编辑 `pkg/handlers/resources/deployment_handler.go`

---

### 🛠️ Kubernetes - Deployment 详情

**🔤 用户描述方式**:
- 主要: "Deployment 详情", "部署详情", "查看部署", "应用详情"
- 别名: "deployment detail", "服务详情", "工作负载详情"

**📍 代码位置**:
- 前端页面: `ui/src/pages/deployment-detail.tsx` - Deployment 详情页
- Pod 监控: `ui/src/components/pod-monitoring.tsx` - Pod 实时监控组件
- API 端点: `GET /api/v1/cluster/:clusterid/deployments/:namespace/:name` - 获取详情

**🎨 视觉标识**:
- 外观: 多 Tab 视图，包含概览、Pods、副本集、事件、YAML
- 文本: "副本数"、"镜像"、"选择器"、"Pod 状态"
- 操作: "扩缩容"、"重启"、"编辑"、"删除"

**⚡ 修改指引**:
- 修改详情页面: 编辑 `ui/src/pages/deployment-detail.tsx`
- 修改 Pod 监控: 编辑 `ui/src/components/pod-monitoring.tsx`
- 添加自定义操作: 在详情页面添加操作按钮和对应 API 调用

---

### 🔄 Kubernetes - 扩缩容操作

**🔤 用户描述方式**:
- 主要: "扩缩容", "调整副本数", "Scale", "副本数调整"
- 别名: "横向扩展", "增减副本", "scale deployment"

**📍 代码位置**:
- 前端组件: 在 Deployment 详情页的操作按钮中
- API 端点: `PATCH /api/v1/cluster/:clusterid/deployments/:namespace/:name` - 更新副本数
- 后端处理器: `pkg/handlers/resources/deployment_handler.go:PatchResource`

**🎨 视觉标识**:
- 外观: 对话框或输入框，显示当前副本数，可输入新值
- 文本: "当前副本数: 3"、"目标副本数"、"确认"、"取消"

**⚡ 修改指引**:
- 修改扩缩容 UI: 在 Deployment 详情页添加对话框组件
- 修改 API 调用: 编辑 `ui/src/services/kubernetes.ts` 中的 patch 方法
- 修改后端逻辑: 编辑 `pkg/handlers/resources/deployment_handler.go`

---

### 🌐 Kubernetes - Services 列表

**🔤 用户描述方式**:
- 主要: "Service 列表", "服务列表", "网络服务", "服务管理"
- 别名: "services", "svc", "服务发现", "负载均衡"

**📍 代码位置**:
- 前端页面: `ui/src/pages/service-list-page.tsx` - Service 列表页
- 服务表格: `ui/src/components/service-table.tsx` - Service 专用表格
- API 端点: `GET /api/v1/cluster/:clusterid/services` - 获取 Service 列表
- 后端处理器: `pkg/handlers/resources/service_handler.go` - Service 操作

**🎨 视觉标识**:
- 外观: 表格视图，显示 Service 名称、类型、Cluster IP、端口、选择器
- 文本: "Services"、"ClusterIP"、"NodePort"、"LoadBalancer"
- 类型标签: 不同服务类型用不同颜色标签区分

**⚡ 修改指引**:
- 修改列表页面: 编辑 `ui/src/pages/service-list-page.tsx`
- 修改服务表格: 编辑 `ui/src/components/service-table.tsx`
- 修改后端查询: 编辑 `pkg/handlers/resources/service_handler.go`

---

### 📊 Kubernetes - 资源监控图表

**🔤 用户描述方式**:
- 主要: "资源监控", "监控图表", "CPU/内存图表", "性能监控"
- 别名: "monitoring", "metrics", "资源使用率", "性能指标"

**📍 代码位置**:
- 前端组件: `ui/src/components/chart/` 目录下的各种图表组件
  - `cpu-usage-chart.tsx` - CPU 使用率图表
  - `memory-usage-chart.tsx` - 内存使用率图表
  - `disk-io-usage-chart.tsx` - 磁盘 I/O 图表
  - `network-usage-chart.tsx` - 网络流量图表
- 监控组件: `ui/src/components/node-monitoring.tsx` 和 `pod-monitoring.tsx`
- API 端点: `GET /api/v1/cluster/:clusterid/prometheus/resource-usage-history` - Prometheus 指标
- 后端处理器: `pkg/handlers/prom_handler.go` - Prometheus 集成

**🎨 视觉标识**:
- 外观: Recharts 折线图/面积图，实时更新，带时间轴
- 文本: "CPU 使用率"、"内存使用率"、"网络流量"
- 颜色: 蓝色（CPU）、绿色（内存）、紫色（网络）

**⚡ 修改指引**:
- 修改图表组件: 编辑 `ui/src/components/chart/` 目录下的对应文件
- 修改数据源: 编辑 `pkg/handlers/prom_handler.go` 中的 Prometheus 查询
- 添加新图表: 创建新图表组件并接入监控数据流

---

### 🗂️ Kubernetes - ConfigMaps 管理

**🔤 用户描述方式**:
- 主要: "ConfigMap", "配置映射", "配置管理", "环境配置"
- 别名: "配置字典", "配置项", "config map"

**📍 代码位置**:
- 前端页面: `ui/src/pages/configmap-list-page.tsx` - ConfigMap 列表页
- 选择器: `ui/src/components/selector/configmap-selector.tsx` - ConfigMap 选择组件
- API 端点: `GET /api/v1/cluster/:clusterid/configmaps` - 获取 ConfigMap 列表
- 后端处理器: `pkg/handlers/resources/configmap_handler.go` - ConfigMap 操作

**🎨 视觉标识**:
- 外观: 表格视图，显示 ConfigMap 名称、命名空间、数据项数量、创建时间
- 文本: "ConfigMaps"、"键值对"、"数据"
- 操作: "查看"、"编辑"、"删除"

**⚡ 修改指引**:
- 修改列表页面: 编辑 `ui/src/pages/configmap-list-page.tsx`
- 修改数据展示: 在详情页中自定义 ConfigMap 数据展示格式
- 修改后端操作: 编辑 `pkg/handlers/resources/configmap_handler.go`

---

### 🔐 Kubernetes - Secrets 管理

**🔤 用户描述方式**:
- 主要: "Secret", "密钥管理", "凭证管理", "敏感信息"
- 别名: "secrets", "密码管理", "证书管理", "加密配置"

**📍 代码位置**:
- 前端页面: `ui/src/pages/secret-list-page.tsx` 和 `secret-detail.tsx` - Secret 管理页
- 选择器: `ui/src/components/selector/secret-selector.tsx` - Secret 选择组件
- API 端点: `GET /api/v1/cluster/:clusterid/secrets` - 获取 Secret 列表
- 后端处理器: `pkg/handlers/resources/secret_handler.go` - Secret 操作

**🎨 视觉标识**:
- 外观: 表格视图，显示 Secret 名称、类型、命名空间，数据脱敏显示
- 文本: "Secrets"、"Opaque"、"kubernetes.io/tls"、"已加密"
- 安全标识: 密码字段用 `****` 遮罩

**⚡ 修改指引**:
- 修改列表页面: 编辑 `ui/src/pages/secret-list-page.tsx`
- 修改详情展示: 编辑 `ui/src/pages/secret-detail.tsx` 中的脱敏逻辑
- 修改后端加密: 编辑 `pkg/crypto/encryption.go`

---

### 🌍 Kubernetes - Ingress 管理

**🔤 用户描述方式**:
- 主要: "Ingress", "入口管理", "路由规则", "域名配置"
- 别名: "ingress controller", "反向代理", "HTTP 路由"

**📍 代码位置**:
- 前端页面: `ui/src/pages/ingress-list-page.tsx` - Ingress 列表页
- API 端点: `GET /api/v1/cluster/:clusterid/ingresses` - 获取 Ingress 列表
- 后端处理器: `pkg/handlers/resources/ingress_handler.go` - Ingress 操作

**🎨 视觉标识**:
- 外观: 表格视图，显示 Ingress 名称、主机名、路径、后端服务
- 文本: "Ingresses"、"Host"、"Path"、"Service"、"Port"
- 规则: 显示路由规则列表（host → path → service）

**⚡ 修改指引**:
- 修改列表页面: 编辑 `ui/src/pages/ingress-list-page.tsx`
- 修改规则展示: 自定义 Ingress 规则的表格列和格式
- 修改后端查询: 编辑 `pkg/handlers/resources/ingress_handler.go`

---

### 📦 Kubernetes - 命名空间管理

**🔤 用户描述方式**:
- 主要: "命名空间", "Namespace", "环境隔离", "项目空间"
- 别名: "namespace", "ns", "名字空间"

**📍 代码位置**:
- 前端页面: `ui/src/pages/namespace-list-page.tsx` - Namespace 列表页
- 选择器: `ui/src/components/selector/namespace-selector.tsx` - Namespace 选择器
- API 端点: `GET /api/v1/cluster/:clusterid/namespaces` - 获取 Namespace 列表
- 后端处理器: `pkg/handlers/resources/namespace_handler.go` - Namespace 操作

**🎨 视觉标识**:
- 外观: 表格视图 + 页面顶部下拉选择器
- 文本: "Namespaces"、"default"、"kube-system"、"Active"
- 选择器: 显示当前命名空间，可切换到其他命名空间

**⚡ 修改指引**:
- 修改列表页面: 编辑 `ui/src/pages/namespace-list-page.tsx`
- 修改选择器: 编辑 `ui/src/components/selector/namespace-selector.tsx`
- 修改后端逻辑: 编辑 `pkg/handlers/resources/namespace_handler.go`

---

### 🖥️ Kubernetes - Nodes 管理

**🔤 用户描述方式**:
- 主要: "Node", "节点", "工作节点", "集群节点"
- 别名: "nodes", "worker nodes", "master nodes", "服务器节点"

**📍 代码位置**:
- 前端页面: `ui/src/pages/node-list-page.tsx` 和 `node-detail.tsx` - Node 管理页
- 监控组件: `ui/src/components/node-monitoring.tsx` - Node 监控
- API 端点: `GET /api/v1/cluster/:clusterid/nodes` - 获取 Node 列表
- 后端处理器: `pkg/handlers/resources/node_handler.go` - Node 操作

**🎨 视觉标识**:
- 外观: 表格视图，显示节点名称、状态、角色、版本、IP、资源使用率
- 文本: "Nodes"、"Ready"、"NotReady"、"master"、"worker"
- 状态图标: 绿色对勾（Ready）、红色叉（NotReady）

**⚡ 修改指引**:
- 修改列表页面: 编辑 `ui/src/pages/node-list-page.tsx`
- 修改详情页面: 编辑 `ui/src/pages/node-detail.tsx`
- 修改节点监控: 编辑 `ui/src/components/node-monitoring.tsx`

---

### 🔗 Kubernetes - CRD 和自定义资源

**🔤 用户描述方式**:
- 主要: "CRD", "自定义资源", "扩展资源", "CustomResourceDefinition"
- 别名: "custom resources", "CR", "扩展对象"

**📍 代码位置**:
- 前端页面: `ui/src/pages/crd-list-page.tsx` 和 `cr-list-page.tsx` - CRD 管理页
- 选择器: `ui/src/components/selector/crd-selector.tsx` - CRD 选择器
- API 端点: `GET /api/v1/cluster/:clusterid/crds` - 获取 CRD 列表
- 后端处理器: `pkg/handlers/resources/crd_handler.go` - CRD 操作

**🎨 视觉标识**:
- 外观: 两级页面：CRD 列表 → 具体 CR 实例列表
- 文本: "CRDs"、"Group"、"Version"、"Kind"
- 导航: 点击 CRD 进入该类型的资源列表

**⚡ 修改指引**:
- 修改 CRD 列表: 编辑 `ui/src/pages/crd-list-page.tsx`
- 修改 CR 列表: 编辑 `ui/src/pages/cr-list-page.tsx`
- 修改后端查询: 编辑 `pkg/handlers/resources/crd_handler.go`

---

### 📚 Kubernetes - 资源历史记录

**🔤 用户描述方式**:
- 主要: "资源历史", "变更记录", "操作历史", "版本历史"
- 别名: "history", "audit trail", "资源版本", "修改记录"

**📍 代码位置**:
- 前端组件: `ui/src/components/resource-history-table.tsx` - 资源历史表格
- 数据模型: `internal/models/resource_history.go` - 历史记录模型
- 后端仓储: `internal/repository/k8s_repository.go` - 历史记录查询

**🎨 视觉标识**:
- 外观: 表格视图，显示操作时间、操作人、操作类型、YAML 差异
- 文本: "历史记录"、"创建"、"更新"、"删除"、"查看差异"
- 差异: YAML 差异对比视图（红色删除、绿色新增）

**⚡ 修改指引**:
- 修改历史表格: 编辑 `ui/src/components/resource-history-table.tsx`
- 修改差异查看器: 编辑 `ui/src/components/yaml-diff-viewer.tsx`
- 修改后端存储: 编辑 `internal/repository/k8s_repository.go`

---

### 🔗 Kubernetes - 关联资源

**🔤 用户描述方式**:
- 主要: "关联资源", "相关资源", "依赖关系", "资源关联"
- 别名: "related resources", "关系图", "资源依赖"

**📍 代码位置**:
- 前端组件: `ui/src/components/related-resource-table.tsx` - 关联资源表格
- 后端工具: `pkg/utils/` 目录下的资源关系分析工具

**🎨 视觉标识**:
- 外观: 资源详情页中的"关联资源" Tab，显示关联资源列表
- 文本: "关联资源"、"Service → Pods"、"Deployment → ReplicaSet → Pods"
- 关系: 显示资源之间的依赖和引用关系

**⚡ 修改指引**:
- 修改关联表格: 编辑 `ui/src/components/related-resource-table.tsx`
- 添加新关联类型: 在后端工具中添加资源关系分析逻辑
- 修改展示格式: 自定义关联资源的展示方式

---

### 🗄️ 中间件管理 - MySQL/PostgreSQL/Redis

**🔤 用户描述方式**:
- 主要: "中间件管理", "数据库管理", "MySQL", "Redis", "PostgreSQL"
- 别名: "middleware", "数据库实例", "缓存管理"

**📍 代码位置**:
- 前端页面: `ui/src/pages/middleware-overview.tsx` - 中间件概览页
- 布局: `ui/src/layouts/middleware-layout.tsx` - 中间件子系统布局
- 数据库管理: `ui/src/pages/database-management.tsx` - 数据库详细管理
- API 端点: `/api/v1/instances?type=mysql|postgresql|redis` - 获取实例列表
- 后端管理器: `internal/services/managers/` - MySQL、PostgreSQL、Redis 管理器

**🎨 视觉标识**:
- 外观: 带侧边栏的管理界面，左侧显示 MySQL、PostgreSQL、Redis 菜单
- 文本: "MySQL"、"PostgreSQL"、"Redis"、实例列表、连接信息
- 状态: 运行中、已停止、错误等状态标签

**⚡ 修改指引**:
- 修改概览页面: 编辑 `ui/src/pages/middleware-overview.tsx`
- 修改侧边栏: 编辑 `ui/src/components/middleware-sidebar.tsx`
- 修改后端管理器: 编辑 `internal/services/managers/` 目录下的对应管理器

---

### 🗃️ MinIO - 对象存储管理

**🔤 用户描述方式**:
- 主要: "MinIO", "对象存储", "S3 存储", "文件存储"
- 别名: "minio bucket", "对象存储桶", "云存储"

**📍 代码位置**:
- 前端页面: `ui/src/pages/minio-management.tsx` - MinIO 管理页
- 布局: `ui/src/layouts/minio-layout.tsx` - MinIO 子系统布局
- API 端点: `/api/v1/minio/instances/:id/buckets` - Bucket 管理
- 后端处理器: `internal/api/handlers/minio/buckets.go` 和 `objects.go` - MinIO 操作

**🎨 视觉标识**:
- 外观: Bucket 列表 + 对象浏览器（类似文件管理器）
- 文本: "Buckets"、"Objects"、"上传"、"下载"、"删除"
- 操作: 创建 Bucket、上传文件、浏览对象

**⚡ 修改指引**:
- 修改管理页面: 编辑 `ui/src/pages/minio-management.tsx`
- 修改 Bucket 操作: 编辑 `internal/api/handlers/minio/buckets.go`
- 修改对象操作: 编辑 `internal/api/handlers/minio/objects.go`

---

### 🐳 Docker - 容器管理

**🔤 用户描述方式**:
- 主要: "Docker", "容器管理", "Docker 容器", "容器列表"
- 别名: "docker containers", "container management", "容器化应用"

**📍 代码位置**:
- 前端页面: `ui/src/pages/docker-overview.tsx` - Docker 概览页
- 布局: `ui/src/layouts/docker-layout.tsx` - Docker 子系统布局
- API 端点: `/api/v1/instances?type=docker` - Docker 实例列表
- 后端处理器: `internal/api/handlers/docker/` - 容器、镜像、日志处理

**🎨 视觉标识**:
- 外观: 带侧边栏的 Docker 管理界面，显示容器、镜像、网络等
- 文本: "Containers"、"Images"、"Networks"、"Running"、"Stopped"
- 操作: 启动、停止、重启、删除容器

**⚡ 修改指引**:
- 修改概览页面: 编辑 `ui/src/pages/docker-overview.tsx`
- 修改侧边栏: 编辑 `ui/src/components/docker-sidebar.tsx`
- 修改后端处理: 编辑 `internal/api/handlers/docker/` 目录下的处理器

---

### 🔔 告警管理 - 告警规则

**🔤 用户描述方式**:
- 主要: "告警", "告警规则", "监控告警", "报警管理"
- 别名: "alerts", "alerting", "通知规则", "报警配置"

**📍 代码位置**:
- 前端页面: `ui/src/pages/alerts.tsx` - 告警管理页
- 数据模型: `internal/models/alert.go` 和 `alert_event.go` - 告警模型
- API 端点: `/api/v1/alerts/rules` - 告警规则管理
- 后端处理器: `internal/api/handlers/alert_handler.go` - 告警处理器
- 告警服务: `internal/services/alert/` - 告警处理和通知

**🎨 视觉标识**:
- 外观: 告警规则列表 + 告警事件列表，两个 Tab 切换
- 文本: "告警规则"、"告警事件"、"严重"、"警告"、"信息"
- 状态: 激活、已解决、已确认等状态标签

**⚡ 修改指引**:
- 修改告警页面: 编辑 `ui/src/pages/alerts.tsx`
- 修改告警逻辑: 编辑 `internal/services/alert/` 目录下的服务
- 修改通知渠道: 编辑 `internal/services/notification/` 目录下的通知服务

---

### 👥 用户管理 - 用户列表

**🔤 用户描述方式**:
- 主要: "用户管理", "用户列表", "账号管理", "用户账户"
- 别名: "users", "user management", "账号列表"

**📍 代码位置**:
- 前端页面: `ui/src/pages/users.tsx` - 用户列表页
- 用户表单: `ui/src/pages/user-form.tsx` - 用户创建/编辑表单
- API 端点: `/api/v1/admin/users` - 用户管理（需管理员权限）
- 后端处理器: `pkg/handlers/user_handler.go` - 用户 CRUD 操作
- 数据模型: `internal/models/user.go` - 用户模型

**🎨 视觉标识**:
- 外观: 表格视图，显示用户名、邮箱、角色、状态、创建时间
- 文本: "用户管理"、"新建用户"、"编辑"、"删除"、"重置密码"
- 角色: "管理员"、"普通用户"标签

**⚡ 修改指引**:
- 修改用户列表: 编辑 `ui/src/pages/users.tsx`
- 修改用户表单: 编辑 `ui/src/pages/user-form.tsx`
- 修改后端逻辑: 编辑 `pkg/handlers/user_handler.go`

---

### 🔑 角色权限 - RBAC 管理

**🔤 用户描述方式**:
- 主要: "角色管理", "权限管理", "RBAC", "访问控制"
- 别名: "roles", "permissions", "角色权限", "权限配置"

**📍 代码位置**:
- 前端页面: `ui/src/pages/roles.tsx` - 角色列表页
- 角色表单: `ui/src/pages/role-form.tsx` - 角色创建/编辑表单
- 设置组件: `ui/src/components/settings/rbac-management.tsx` - RBAC 配置
- 数据模型: `internal/models/role.go` - 角色模型
- RBAC 中间件: `internal/api/middleware/rbac.go` - RBAC 权限检查

**🎨 视觉标识**:
- 外观: 角色列表 + 权限分配界面
- 文本: "角色管理"、"权限"、"资源访问"、"操作权限"
- 权限树: 树形结构显示资源和操作权限

**⚡ 修改指引**:
- 修改角色页面: 编辑 `ui/src/pages/roles.tsx`
- 修改权限配置: 编辑 `ui/src/components/settings/rbac-management.tsx`
- 修改 RBAC 逻辑: 编辑 `pkg/rbac/rbac.go` 和 `internal/api/middleware/rbac.go`

---

### ⚙️ 系统设置 - 配置管理

**🔤 用户描述方式**:
- 主要: "系统设置", "配置管理", "设置页面", "系统配置"
- 别名: "settings", "configuration", "全局配置", "参数设置"

**📍 代码位置**:
- 前端页面: `ui/src/pages/settings.tsx` - 系统设置页
- 集群管理: `ui/src/components/settings/cluster-management.tsx` - 集群配置
- OAuth 管理: `ui/src/components/settings/oauth-provider-management.tsx` - OAuth 配置
- 用户管理: `ui/src/components/settings/user-management.tsx` - 用户管理
- API 端点: `/api/v1/admin/system/config` - 系统配置
- 后端处理器: `internal/api/handlers/system_handler.go` - 系统配置处理

**🎨 视觉标识**:
- 外观: Tab 切换界面，包含集群、用户、OAuth、RBAC 等配置项
- 文本: "集群管理"、"用户管理"、"OAuth 配置"、"RBAC 配置"
- 操作: 各配置项的增删改查

**⚡ 修改指引**:
- 修改设置页面: 编辑 `ui/src/pages/settings.tsx`
- 修改具体配置: 编辑 `ui/src/components/settings/` 目录下的对应组件
- 修改后端配置: 编辑 `internal/api/handlers/system_handler.go`

---

### 🔧 安装向导 - 初始化配置

**🔤 用户描述方式**:
- 主要: "安装向导", "初始化", "首次配置", "系统安装"
- 别名: "installation wizard", "setup", "初始设置", "引导配置"

**📍 代码位置**:
- 前端页面: `ui/src/pages/install/index.tsx` - 安装向导主页
- 步骤组件: `ui/src/pages/install/steps/` - 各安装步骤
  - `database-step.tsx` - 数据库配置
  - `admin-step.tsx` - 管理员创建
  - `settings-step.tsx` - 系统设置
  - `confirm-step.tsx` - 确认配置
- 安装守卫: `ui/src/components/guards/install-guard.tsx` - 安装状态检查
- API 端点: `/api/install/*` - 安装 API
- 后端处理器: `internal/install/handlers/` - 安装处理器

**🎨 视觉标识**:
- 外观: 步骤式向导界面，顶部进度指示器
- 文本: "欢迎使用 Tiga"、"数据库配置"、"创建管理员"、"完成安装"
- 进度: 步骤 1/4 → 2/4 → 3/4 → 4/4

**⚡ 修改指引**:
- 修改安装步骤: 编辑 `ui/src/pages/install/steps/` 目录下的步骤组件
- 修改安装逻辑: 编辑 `internal/install/handlers/` 目录下的处理器
- 修改进度显示: 编辑 `ui/src/pages/install/components/progress-indicator.tsx`

---

### 📊 审计日志 - 操作记录

**🔤 用户描述方式**:
- 主要: "审计日志", "操作日志", "审计记录", "活动日志"
- 别名: "audit log", "operation history", "行为记录", "系统日志"

**📍 代码位置**:
- 数据模型: `internal/models/audit_log.go` - 审计日志模型
- API 端点: `/api/v1/audit` - 审计日志查询
- 后端处理器: `internal/api/handlers/audit_handler.go` - 审计日志处理器
- 仓储: `internal/repository/audit_repo.go` - 审计日志数据访问
- 中间件: `internal/api/middleware/audit.go` - 自动记录审计日志

**🎨 视觉标识**:
- 外观: 时间线视图或表格视图，显示操作时间、用户、操作类型、资源等
- 文本: "审计日志"、"创建"、"更新"、"删除"、"用户"、"资源"
- 过滤: 按用户、操作类型、时间范围过滤

**⚡ 修改指引**:
- 修改日志模型: 编辑 `internal/models/audit_log.go`
- 修改查询接口: 编辑 `internal/api/handlers/audit_handler.go`
- 修改记录逻辑: 编辑 `internal/api/middleware/audit.go`

---

### 🎨 主题切换 - 深色/浅色模式

**🔤 用户描述方式**:
- 主要: "主题切换", "深色模式", "浅色模式", "暗黑模式"
- 别名: "dark mode", "light mode", "theme toggle", "外观设置"

**📍 代码位置**:
- 前端组件: `ui/src/components/mode-toggle.tsx` - 主题切换按钮
- 主题提供者: `ui/src/components/theme-provider.tsx` - 主题状态管理
- 颜色主题: `ui/src/components/color-theme-provider.tsx` - 颜色主题

**🎨 视觉标识**:
- 外观: 右上角的太阳/月亮图标按钮
- 文本: 无文字，仅图标
- 切换: 点击切换深色/浅色模式，实时生效

**⚡ 修改指引**:
- 修改切换按钮: 编辑 `ui/src/components/mode-toggle.tsx`
- 修改主题配置: 编辑 `ui/src/components/theme-provider.tsx`
- 修改颜色变量: 编辑 `ui/src/index.css` 中的 CSS 变量

---

### 🌐 语言切换 - 中英文切换

**🔤 用户描述方式**:
- 主要: "语言切换", "中英文切换", "国际化", "多语言"
- 别名: "language toggle", "i18n", "切换语言", "翻译"

**📍 代码位置**:
- 前端组件: `ui/src/components/language-toggle.tsx` - 语言切换按钮
- 国际化配置: `ui/src/i18n/` - 翻译文件和配置

**🎨 视觉标识**:
- 外观: 右上角的语言图标按钮（地球图标或语言缩写）
- 文本: "中文"、"English" 或 "中" / "EN"
- 下拉菜单: 显示可用语言列表

**⚡ 修改指引**:
- 修改切换按钮: 编辑 `ui/src/components/language-toggle.tsx`
- 添加新语言: 在 `ui/src/i18n/` 目录添加新的翻译文件
- 修改翻译: 编辑 `ui/src/i18n/` 目录下的 JSON 翻译文件

---

### 👤 用户菜单 - 账户操作

**🔤 用户描述方式**:
- 主要: "用户菜单", "账户菜单", "个人中心", "退出登录"
- 别名: "user menu", "profile menu", "账号设置", "注销"

**📍 代码位置**:
- 前端组件: `ui/src/components/user-menu.tsx` - 用户菜单组件
- 认证上下文: `ui/src/contexts/auth-context.tsx` - 用户认证状态

**🎨 视觉标识**:
- 外观: 右上角的用户头像/用户名，点击展开下拉菜单
- 文本: 用户名、"个人设置"、"退出登录"
- 菜单: 下拉菜单包含账户相关操作

**⚡ 修改指引**:
- 修改菜单组件: 编辑 `ui/src/components/user-menu.tsx`
- 修改菜单项: 在组件中添加或删除菜单项
- 修改登出逻辑: 编辑认证上下文中的 logout 方法

---

### 🔍 搜索框 - 页面内搜索

**🔤 用户描述方式**:
- 主要: "搜索", "搜索框", "查找", "过滤"
- 别名: "search", "filter", "查询", "搜索功能"

**📍 代码位置**:
- 前端组件: `ui/src/components/search.tsx` - 页面搜索框
- 全局搜索: `ui/src/components/global-search.tsx` - 全局搜索（Cmd+K）

**🎨 视觉标识**:
- 外观: 页面顶部的搜索输入框，带放大镜图标
- 文本: "搜索..." 占位符
- 快捷键: 显示 "⌘K" 快捷键提示

**⚡ 修改指引**:
- 修改搜索框样式: 编辑 `ui/src/components/search.tsx`
- 修改搜索逻辑: 编辑 `ui/src/components/global-search.tsx`
- 修改快捷键: 在 `global-search-provider.tsx` 中修改快捷键绑定

---

### 📍 面包屑导航 - 路径导航

**🔤 用户描述方式**:
- 主要: "面包屑", "路径导航", "导航栏", "当前位置"
- 别名: "breadcrumb", "导航路径", "页面路径"

**📍 代码位置**:
- 前端组件: `ui/src/components/dynamic-breadcrumb.tsx` - 动态面包屑组件

**🎨 视觉标识**:
- 外观: 页面顶部的路径导航，"首页 / K8s / Pods / pod-name"
- 文本: 当前页面路径，用 "/" 分隔
- 交互: 可点击面包屑导航到上级页面

**⚡ 修改指引**:
- 修改面包屑样式: 编辑 `ui/src/components/dynamic-breadcrumb.tsx`
- 修改路径生成: 在组件中自定义路径生成逻辑
- 修改分隔符: 修改面包屑分隔符样式

---

### 📱 侧边栏 - 导航菜单

**🔤 用户描述方式**:
- 主要: "侧边栏", "导航菜单", "菜单栏", "左侧菜单"
- 别名: "sidebar", "navigation", "侧边导航", "主菜单"

**📍 代码位置**:
- 前端组件: 多个侧边栏组件
  - `ui/src/components/app-sidebar.tsx` - 主应用侧边栏
  - `ui/src/components/devops-sidebar.tsx` - DevOps 侧边栏
  - `ui/src/components/vms-sidebar.tsx` - VMs 侧边栏
  - `ui/src/components/middleware-sidebar.tsx` - Middleware 侧边栏
  - 等等

**🎨 视觉标识**:
- 外观: 左侧可折叠的导航菜单，显示菜单项和图标
- 文本: 根据子系统不同显示不同菜单项
- 折叠: 可折叠为图标模式

**⚡ 修改指引**:
- 修改侧边栏内容: 编辑对应子系统的 sidebar 组件
- 修改折叠行为: 编辑 `ui/src/components/ui/sidebar.tsx`
- 添加新菜单项: 在对应侧边栏组件中添加菜单项

---

### 📄 页脚 - 版本信息

**🔤 用户描述方式**:
- 主要: "页脚", "底部信息", "版本号", "版权信息"
- 别名: "footer", "底部栏", "版本信息"

**📍 代码位置**:
- 前端组件: `ui/src/components/footer.tsx` - 页脚组件
- 版本信息: `ui/src/components/version-info.tsx` - 版本显示

**🎨 视觉标识**:
- 外观: 页面底部的版权和版本信息栏
- 文本: "© 2025 Tiga"、"v1.0.0"、链接等

**⚡ 修改指引**:
- 修改页脚内容: 编辑 `ui/src/components/footer.tsx`
- 修改版本显示: 编辑 `ui/src/components/version-info.tsx`
- 修改版本来源: 编辑后端版本信息 API

---

### ➕ 创建资源 - 快捷创建对话框

**🔤 用户描述方式**:
- 主要: "创建资源", "新建资源", "创建对话框", "快捷创建"
- 别名: "create dialog", "新建", "添加资源"

**📍 代码位置**:
- 前端组件: `ui/src/components/create-resource-dialog.tsx` - 创建资源对话框

**🎨 视觉标识**:
- 外观: 右上角的 "+" 图标，点击弹出创建资源对话框
- 文本: "创建资源"、资源类型选择、YAML 输入
- 操作: 选择资源类型 → 输入 YAML → 创建

**⚡ 修改指引**:
- 修改对话框: 编辑 `ui/src/components/create-resource-dialog.tsx`
- 修改资源类型: 在对话框中添加或删除可创建的资源类型
- 修改创建逻辑: 编辑对话框中的 API 调用

---

### 🗑️ 删除确认 - 删除对话框

**🔤 用户描述方式**:
- 主要: "删除确认", "删除对话框", "确认删除", "删除提示"
- 别名: "delete confirmation", "删除弹窗", "删除警告"

**📍 代码位置**:
- 前端组件: `ui/src/components/delete-confirmation-dialog.tsx` - 删除确认对话框

**🎨 视觉标识**:
- 外观: 删除操作时弹出的确认对话框，通常为红色主题
- 文本: "确认删除?"、"此操作无法撤销"、"删除"、"取消"
- 警告: 显示删除警告信息和资源名称

**⚡ 修改指引**:
- 修改对话框样式: 编辑 `ui/src/components/delete-confirmation-dialog.tsx`
- 修改确认逻辑: 在组件中自定义删除确认流程
- 修改提示文案: 修改对话框中的警告文本

---

## 🔧 后端核心功能映射

### API 路由结构

**路由前缀说明**:
- `/api/auth` - 认证相关（登录、OAuth、当前用户）
- `/api/config` - 公共配置（应用配置、系统配置）
- `/api/v1/init_check` - 初始化检查
- `/api/v1/admin` - 管理员功能（用户、集群、OAuth、系统配置）
- `/api/v1/cluster/:clusterid` - Kubernetes 资源操作
- `/api/v1/instances` - 实例管理（VMs、数据库、MinIO 等）
- `/api/v1/minio` - MinIO 对象存储
- `/api/v1/alerts` - 告警管理
- `/api/v1/audit` - 审计日志

**代码位置**: `internal/api/routes.go:25-284`

---

### 中间件栈

**执行顺序**:
1. CORS - 跨域请求处理
2. Logger - 请求日志记录
3. AuthRequired - JWT 认证验证
4. RBAC - 基于角色的访问控制
5. RateLimit - 请求限流
6. Audit - 审计日志记录

**代码位置**:
- CORS: `pkg/middleware/cors.go`
- Logger: `pkg/middleware/logger.go`
- Auth: `internal/api/middleware/auth.go`
- RBAC: `internal/api/middleware/rbac.go` 和 `pkg/middleware/rbac.go`
- Audit: `internal/api/middleware/audit.go`

---

### 数据库模型

**核心模型位置**: `internal/models/`

- `user.go` - 用户模型（包含认证、角色）
- `cluster.go` - Kubernetes 集群模型
- `instance.go` - 实例模型（VMs、数据库、MinIO、Docker 等）
- `alert.go` / `alert_event.go` - 告警规则和事件
- `audit_log.go` - 审计日志
- `resource_history.go` - 资源历史记录
- `oauth_provider.go` - OAuth 提供商
- `role.go` - 角色和权限
- `metric.go` - 监控指标

---

### 仓储层（Repository Pattern）

**仓储位置**: `internal/repository/`

- `user_repo.go` - 用户数据访问
- `instance_repo.go` - 实例数据访问
- `alert_repo.go` - 告警数据访问
- `audit_repo.go` - 审计日志数据访问
- `k8s_repository.go` - Kubernetes 资源历史数据访问
- `metrics_repo.go` - 监控指标数据访问
- `oauth_provider_repo.go` - OAuth 提供商数据访问

**优化版仓储**（带缓存和性能优化）:
- `instance_repo_optimized.go`
- `audit_repo_optimized.go`

---

### 业务服务层

**服务位置**: `internal/services/`

- `auth/` - 认证服务（登录、JWT、Session）
- `instance_service.go` - 实例管理服务
- `k8s_service.go` - Kubernetes 服务
- `managers/` - 实例管理器
  - `minio_manager.go` - MinIO 管理器
  - `mysql_manager.go` - MySQL 管理器
  - `postgresql_manager.go` - PostgreSQL 管理器
  - `redis_manager.go` - Redis 管理器
  - `docker_manager.go` - Docker 管理器
  - `coordinator.go` - 管理器协调器
- `alert/` - 告警处理服务
- `notification/` - 通知服务（邮件、Webhook、钉钉等）
- `scheduler/` - 后台任务调度器
- `metrics/` - 监控指标收集服务

---

## 🎯 常见开发场景

### 场景 1: 修改登录页 Logo

**用户描述**: "我想换掉登录页的 Logo 图片"

**操作步骤**:
1. 替换图片文件: `ui/src/assets/logo.png`
2. 如需调整大小: 编辑 `ui/src/pages/login.tsx`，修改图片的 className
3. 重新构建前端: `cd ui && pnpm run build`

---

### 场景 2: 添加新的 Kubernetes 资源类型

**用户描述**: "我想在 K8s 菜单中添加 StatefulSet 管理"

**操作步骤**:
1. 后端: 在 `pkg/handlers/resources/` 创建 `statefulset_handler.go`
2. 注册路由: 在 `pkg/handlers/resources/routes.go` 添加 StatefulSet 路由
3. 前端: 在 `ui/src/pages/` 创建 `statefulset-list-page.tsx` 和 `statefulset-detail.tsx`
4. 添加侧边栏菜单: 在 K8s 侧边栏组件中添加 StatefulSet 菜单项
5. 添加路由: 在 `ui/src/routes.tsx` 添加对应路由

---

### 场景 3: 修改主仪表板子系统卡片颜色

**用户描述**: "我想把 Kubernetes 卡片从紫色改成蓝色"

**操作步骤**:
1. 编辑 `ui/src/pages/overview-dashboard-new.tsx`
2. 找到第 57-63 行的 Kubernetes 配置
3. 修改 `color: 'bg-purple-500'` 为 `color: 'bg-blue-500'`
4. 保存后前端会自动热重载（开发模式）或需要重新构建（生产模式）

---

### 场景 4: 添加新的告警通知渠道

**用户描述**: "我想添加企业微信通知"

**操作步骤**:
1. 创建通知服务: 在 `internal/services/notification/` 创建 `wechat_notifier.go`
2. 实现 Notifier 接口: 实现 `Send()` 方法
3. 注册通知器: 在 `internal/services/alert/processor.go` 中注册新通知器
4. 前端配置: 在告警规则编辑页面添加企业微信配置选项

---

### 场景 5: 修改主题颜色

**用户描述**: "我想改变整个应用的主色调"

**操作步骤**:
1. 编辑 `ui/src/index.css`
2. 修改 CSS 变量（如 `--primary`、`--accent` 等）
3. 深色模式和浅色模式分别在 `:root` 和 `.dark` 选择器下配置
4. 保存后会自动应用新主题色

---

## 📝 开发指南速查

### 添加新页面

1. 创建页面组件: `ui/src/pages/your-page.tsx`
2. 添加路由: `ui/src/routes.tsx`
3. 添加侧边栏菜单项: 对应的 `*-sidebar.tsx` 组件
4. 添加 API 服务: `ui/src/services/`
5. 后端 API: `internal/api/handlers/` 和 `internal/api/routes.go`

### 添加新 API 端点

1. 创建处理器: `internal/api/handlers/your_handler.go`
2. 注册路由: `internal/api/routes.go`
3. 添加服务层: `internal/services/your_service.go`（如需要）
4. 添加仓储层: `internal/repository/your_repo.go`（如需要）
5. 添加数据模型: `internal/models/your_model.go`（如需要）

### 修改 UI 组件

1. 通用 UI 组件: `ui/src/components/ui/`
2. 业务组件: `ui/src/components/`
3. 页面组件: `ui/src/pages/`
4. 样式: TailwindCSS class 或 `ui/src/index.css`

### 数据库迁移

- 自动迁移: 修改 `internal/models/` 中的模型，启动时自动迁移
- 手动迁移: 当前不支持，依赖 GORM AutoMigrate

---

## 🔗 关键文件索引

### 前端核心文件

| 文件路径 | 功能描述 |
|---------|---------|
| `ui/src/routes.tsx` | 路由定义，所有页面路由配置 |
| `ui/src/App.tsx` | 应用主组件 |
| `ui/src/main.tsx` | 应用入口 |
| `ui/src/contexts/auth-context.tsx` | 认证状态管理 |
| `ui/src/contexts/cluster-context.tsx` | 集群状态管理 |
| `ui/src/services/api.ts` | API 客户端基础配置 |
| `ui/src/components/theme-provider.tsx` | 主题管理 |
| `ui/src/i18n/` | 国际化翻译文件 |

### 后端核心文件

| 文件路径 | 功能描述 |
|---------|---------|
| `cmd/tiga/main.go` | 应用入口 |
| `internal/api/routes.go` | 所有 API 路由定义 |
| `internal/app/app.go` | 应用初始化和生命周期 |
| `internal/config/config.go` | 配置管理 |
| `internal/db/database.go` | 数据库初始化 |
| `pkg/kube/client.go` | Kubernetes 客户端 |
| `pkg/cluster/cluster_manager.go` | 集群管理器 |

### 配置文件

| 文件路径 | 功能描述 |
|---------|---------|
| `config.yaml` | 主配置文件 |
| `Taskfile.yml` | 构建任务定义 |
| `ui/vite.config.ts` | Vite 构建配置 |
| `ui/package.json` | 前端依赖 |
| `go.mod` | Go 依赖 |

---

## 💡 提示

### 搜索技巧

- **按功能搜索**: 使用功能描述在本文档中搜索（如 "登录"、"Pod 列表"）
- **按文件名搜索**: 使用 IDE 的文件搜索功能查找对应文件
- **按组件名搜索**: 前端组件名通常为 kebab-case，如 `pod-table.tsx`
- **按 API 路径搜索**: 在 `internal/api/routes.go` 中搜索 API 路径

### 命名约定

- **前端页面**: `*-page.tsx` 或 `*.tsx`（如 `hosts.tsx`）
- **前端组件**: `kebab-case.tsx`（如 `cluster-selector.tsx`）
- **后端处理器**: `*_handler.go`（如 `user_handler.go`）
- **后端服务**: `*_service.go`（如 `instance_service.go`）
- **后端仓储**: `*_repo.go`（如 `user_repo.go`）
- **数据模型**: `*.go`（如 `user.go`、`cluster.go`）

### 快速定位

- **找前端页面**: 查看 `ui/src/routes.tsx` 的路由配置
- **找 API 端点**: 查看 `internal/api/routes.go` 的路由注册
- **找数据模型**: 查看 `internal/models/` 目录
- **找业务逻辑**: 查看 `internal/services/` 目录
- **找 Kubernetes 操作**: 查看 `pkg/handlers/resources/` 目录

---

## 📞 获取帮助

如需修改某个功能，请尝试：

1. 在本文档中搜索功能描述的关键词
2. 找到对应的"代码位置"和"修改指引"
3. 按照指引修改对应文件
4. 使用 `task dev` 运行开发环境测试修改

祝您开发愉快！🎉
