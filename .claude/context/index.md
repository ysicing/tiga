# 📂 Tiga DevOps Dashboard 功能-代码映射报告

## 🏗️ 项目概览
- **技术栈**: Go 1.24+ (Backend) + React 19 (Frontend) + TypeScript
- **后端框架**: Gin + GORM + Kubernetes client-go
- **前端框架**: Vite + TailwindCSS + Radix UI
- **状态管理**: React Query (TanStack Query) + Zustand
- **样式方案**: TailwindCSS 4.1 + Radix UI + CVA (Class Variance Authority)
- **构建工具**: Vite 7 (Frontend) + Go build (Backend)
- **包管理**: pnpm (Frontend) + go mod (Backend)
- **数据库**: SQLite (默认) / PostgreSQL / MySQL
- **认证**: JWT + OAuth (Google, GitHub)

## 📊 功能模块统计
- **页面级组件**: 67 个 [主要页面/路由]
- **可复用组件**: 129+ 个 [通用UI组件]
- **业务逻辑模块**: 20+ 个 [Services/Repositories]
- **子系统**: 8 个 [VMs, K8s, MinIO, Middleware, Docker, Storage, WebServer, DevOps]
- **后端处理器**: 20+ 个 [API Handlers]
- **中间件**: 6+ 个 [Auth, RBAC, CORS, Rate Limit, Audit]

## 🗂️ 目录结构概览
```
tiga/
├── cmd/tiga/              # 主程序入口
├── internal/              # 后端核心代码
│   ├── api/              # API 层
│   │   ├── handlers/     # 请求处理器
│   │   └── middleware/   # 中间件
│   ├── models/           # 数据模型
│   ├── repository/       # 数据访问层
│   └── services/         # 业务逻辑层
├── pkg/                  # 可复用包
│   ├── handlers/         # K8s 专用处理器
│   └── kube/            # K8s 客户端工具
├── ui/                   # 前端代码
│   └── src/
│       ├── components/   # UI 组件
│       ├── pages/       # 页面组件
│       ├── layouts/     # 布局组件
│       ├── contexts/    # React 上下文
│       └── services/    # API 客户端
└── config.yaml          # 配置文件
```

---

## 🎯 功能映射表

### 系统级功能

#### 登录页面

**🔤 用户描述方式**:
- 主要: "登录页", "登录界面", "登录表单", "用户登录"
- 别名: "login page", "登录页面", "认证页面", "登录入口"

**📍 代码位置**:
- 主文件: `ui/src/pages/login.tsx` - 登录页面组件
- 认证上下文: `ui/src/contexts/auth-context.tsx` - 认证状态管理
- 后端处理: `internal/api/handlers/auth_handler.go` - 登录API处理
- 服务层: `internal/services/auth/login_service.go` - 登录业务逻辑
- API路由: `internal/api/routes.go:116-124` - 登录路由定义

**🎨 视觉标识**:
- 外观: 居中卡片式布局,顶部显示Logo和应用名称
- 文本: "登录"、"用户名"、"密码"、"登录方式"
- 功能: 支持密码登录和OAuth登录(Google、GitHub)

**⚡ 修改指引**:
- 修改登录UI: 编辑 `ui/src/pages/login.tsx`
- 修改登录逻辑: 编辑 `internal/services/auth/login_service.go`
- 修改认证方式: 编辑 `internal/api/handlers/auth_handler.go`
- 添加OAuth提供商: 修改数据库 `models.OAuthProvider` 配置

---

#### 总览仪表板

**🔤 用户描述方式**:
- 主要: "总览", "首页", "仪表板", "主页", "概览页"
- 别名: "dashboard", "overview", "主控制台", "统览"

**📍 代码位置**:
- 主文件: `ui/src/pages/overview-dashboard.tsx` - 总览仪表板
- 卡片组件: `ui/src/components/subsystem-card.tsx` - 子系统卡片
- 路由: `ui/src/App.tsx` - 路由到 `/`

**🎨 视觉标识**:
- 外观: 卡片网格布局,展示各子系统(MySQL、PostgreSQL、Redis、MinIO、Docker、K8s、Caddy)
- 文本: "系统概览"、各子系统名称、实例数量统计
- 交互: 点击卡片跳转到对应子系统

**⚡ 修改指引**:
- 修改卡片布局: 编辑 `ui/src/pages/overview-dashboard.tsx:66-100`
- 修改卡片样式: 编辑 `ui/src/components/subsystem-card.tsx`
- 添加新子系统: 在 `overview-dashboard.tsx:39-47` 的 `ALL_SUBSYSTEM_TYPES` 中添加

---

#### 顶部导航栏

**🔤 用户描述方式**:
- 主要: "顶部导航", "顶栏", "导航栏", "header"
- 别名: "上方菜单", "页头", "头部栏"

**📍 代码位置**:
- 主文件: `ui/src/components/site-header.tsx` - 顶部导航栏组件
- 用户菜单: `ui/src/components/user-nav.tsx` - 用户下拉菜单
- 集群选择器: `ui/src/components/cluster-selector.tsx` - 集群切换器

**🎨 视觉标识**:
- 外观: 固定在页面顶部,包含搜索框、集群选择器、通知、设置、用户头像
- 文本: 用户名、"设置"、"退出登录"
- 交互: 点击用户头像展开下拉菜单

**⚡ 修改指引**:
- 修改导航栏: 编辑 `ui/src/components/site-header.tsx`
- 修改用户菜单: 编辑 `ui/src/components/user-nav.tsx`
- 添加新按钮: 在 `site-header.tsx` 中添加新的组件

---

#### 侧边栏导航

**🔤 用户描述方式**:
- 主要: "侧边栏", "左侧菜单", "导航菜单", "sidebar"
- 别名: "侧栏", "主菜单", "导航面板"

**📍 代码位置**:
- K8s侧边栏: `ui/src/components/app-sidebar.tsx` - Kubernetes子系统侧边栏
- VMs侧边栏: `ui/src/components/vms-sidebar.tsx` - 主机管理侧边栏
- 侧边栏配置: `ui/src/contexts/sidebar-config-context.tsx` - 侧边栏配置管理

**🎨 视觉标识**:
- 外观: 固定在页面左侧,可折叠,显示Logo和菜单项
- 文本: "返回总览"、各功能模块名称、图标
- 交互: 点击菜单项跳转页面,支持分组和固定项

**⚡ 修改指引**:
- 修改K8s侧边栏: 编辑 `ui/src/components/app-sidebar.tsx`
- 修改VMs侧边栏: 编辑 `ui/src/components/vms-sidebar.tsx`
- 添加菜单项: 在对应的 sidebar 组件中的 `menuItems` 数组添加
- 修改侧边栏配置: 编辑 `ui/src/contexts/sidebar-config-context.tsx`

---

### 主机管理子系统 (VMs)

#### 主机列表

**🔤 用户描述方式**:
- 主要: "主机列表", "服务器列表", "主机管理", "VMs列表"
- 别名: "host list", "主机页面", "服务器管理", "机器列表", "节点列表"

**📍 代码位置**:
- 主文件: `ui/src/pages/hosts/host-list-page.tsx` - 主机列表页面
- 主机卡片: `ui/src/components/hosts/host-card.tsx` - 主机卡片组件
- 状态管理: `ui/src/stores/host-store.ts` - 主机状态存储(Zustand)
- 后端处理: `internal/api/handlers/host_handler.go` - 主机API处理器
- 服务层: `internal/services/host/host_service.go` - 主机业务逻辑
- API路由: `internal/api/routes.go:335-346` - 主机路由

**🎨 视觉标识**:
- 外观: 卡片视图或表格视图,显示主机名称、状态、监控指标
- 文本: "主机节点"、"添加主机"、"刷新"、"搜索"
- 功能: 支持卡片/表格视图切换、搜索过滤、实时监控数据更新

**⚡ 修改指引**:
- 修改列表布局: 编辑 `ui/src/pages/hosts/host-list-page.tsx:200-300`
- 修改卡片样式: 编辑 `ui/src/components/hosts/host-card.tsx`
- 修改后端逻辑: 编辑 `internal/api/handlers/host_handler.go:ListHosts`
- 添加新字段: 修改 `internal/models/host_node.go` 模型

---

#### 主机详情页

**🔤 用户描述方式**:
- 主要: "主机详情", "主机监控", "服务器详情", "主机信息"
- 别名: "host detail", "主机详情页", "监控详情", "机器信息"

**📍 代码位置**:
- 主文件: `ui/src/pages/hosts/host-detail-page.tsx` - 主机详情页面
- 监控图表: `ui/src/components/hosts/monitor-chart.tsx` - 单指标图表
- 多线图表: `ui/src/components/hosts/multi-line-chart.tsx` - 多指标图表
- 后端处理: `internal/api/handlers/host_handler.go:GetCurrentState` - 获取当前状态
- API路由: `internal/api/routes.go:344-345` - 状态查询路由

**🎨 视觉标识**:
- 外观: Tab分页布局(概览、CPU、内存、磁盘、网络)、实时监控图表
- 文本: "返回"、"SSH终端"、"编辑"、时间范围选择(1小时、6小时、24小时、7天)
- 功能: 实时监控数据展示、历史数据查询、快捷操作按钮

**⚡ 修改指引**:
- 修改详情页布局: 编辑 `ui/src/pages/hosts/host-detail-page.tsx:103-300`
- 修改图表样式: 编辑 `ui/src/components/hosts/monitor-chart.tsx`
- 修改时间范围: 编辑 `host-detail-page.tsx:18-23` 的 `TIME_RANGES` 配置
- 添加新Tab: 在 `host-detail-page.tsx` 的 `<Tabs>` 中添加新的 `<TabsTrigger>` 和 `<TabsContent>`

---

#### WebSSH终端

**🔤 用户描述方式**:
- 主要: "SSH终端", "网页终端", "远程终端", "WebSSH"
- 别名: "web terminal", "浏览器终端", "在线SSH", "命令行终端"

**📍 代码位置**:
- 主文件: `ui/src/pages/hosts/host-ssh-page.tsx` - SSH终端页面
- 终端组件: `ui/src/components/hosts/webssh-terminal.tsx` - xterm.js终端组件
- 后端处理: `internal/api/handlers/webssh_handler.go` - WebSSH WebSocket处理
- 服务层: `internal/services/webssh/session_manager.go` - SSH会话管理
- 终端管理: `internal/services/host/terminal_manager.go` - 终端实例管理
- API路由: `internal/api/routes.go:366-372` - WebSSH路由

**🎨 视觉标识**:
- 外观: 全屏黑色终端界面,显示命令行提示符
- 文本: "返回"、连接状态提示
- 功能: 实时SSH交互、命令执行、输出显示

**⚡ 修改指引**:
- 修改终端UI: 编辑 `ui/src/pages/hosts/host-ssh-page.tsx:80-120`
- 修改终端样式: 编辑 `ui/src/components/hosts/webssh-terminal.tsx`
- 修改WebSocket处理: 编辑 `internal/api/handlers/webssh_handler.go:HandleWebSocket`
- 修改终端配置: 在 `host-ssh-page.tsx` 中修改 xterm.js 配置

---

### Kubernetes 子系统 (K8s)

#### 集群概览

**🔤 用户描述方式**:
- 主要: "K8s概览", "集群概览", "Kubernetes首页", "K8s总览"
- 别名: "k8s overview", "集群主页", "k8s仪表板", "集群信息"

**📍 代码位置**:
- 主文件: `ui/src/pages/overview.tsx` - K8s概览页面
- 后端处理: `pkg/handlers/overview_handler.go` - 概览API处理
- API路由: `internal/api/routes.go:221` - K8s概览路由

**🎨 视觉标识**:
- 外观: 卡片布局,显示节点数、Pod数、Deployment数等集群统计信息
- 文本: "集群概览"、"节点"、"Pods"、"Deployments"、"Services"
- 功能: 快速了解集群整体状态

**⚡ 修改指引**:
- 修改概览页: 编辑 `ui/src/pages/overview.tsx`
- 修改后端数据: 编辑 `pkg/handlers/overview_handler.go:GetOverview`
- 添加新统计项: 在 `overview.tsx` 中添加新的卡片组件

---

#### Pod 列表

**🔤 用户描述方式**:
- 主要: "Pod列表", "容器组", "Pods", "Pod管理"
- 别名: "pod list", "容器列表", "pod页面", "应用实例"

**📍 代码位置**:
- 主文件: `ui/src/pages/pod-list-page.tsx` - Pod列表页面
- 后端处理: `pkg/handlers/resources/pods.go` - Pod资源处理
- API路由: `internal/api/routes.go:253` 通过 `resources.RegisterRoutes` 注册

**🎨 视觉标识**:
- 外观: 表格显示,包含Pod名称、命名空间、状态、重启次数、年龄
- 文本: "Pods"、"创建Pod"、"删除"、"查看日志"、"进入终端"
- 功能: 列表展示、搜索过滤、创建/删除Pod、查看日志、进入容器

**⚡ 修改指引**:
- 修改列表页: 编辑 `ui/src/pages/pod-list-page.tsx`
- 修改后端逻辑: 编辑 `pkg/handlers/resources/pods.go`
- 添加新列: 在 `pod-list-page.tsx` 的表格定义中添加

---

#### 全局搜索

**🔤 用户描述方式**:
- 主要: "全局搜索", "K8s搜索", "资源搜索", "快速查找"
- 别名: "global search", "搜索功能", "查找资源", "快捷搜索"

**📍 代码位置**:
- 搜索组件: `ui/src/components/global-search.tsx` - 搜索对话框组件
- 搜索上下文: `ui/src/components/global-search-provider.tsx` - 搜索状态管理
- 后端处理: `pkg/handlers/search_handler.go` - 全局搜索API
- API路由: `internal/api/routes.go:240` - 搜索路由

**🎨 视觉标识**:
- 外观: 快捷键触发(Ctrl+K / Cmd+K)的命令面板样式对话框
- 文本: "搜索资源..."、资源类型、资源名称、命名空间
- 功能: 跨命名空间搜索K8s资源、快速导航

**⚡ 修改指引**:
- 修改搜索UI: 编辑 `ui/src/components/global-search.tsx`
- 修改搜索逻辑: 编辑 `pkg/handlers/search_handler.go:GlobalSearch`
- 修改快捷键: 在 `global-search.tsx` 中修改快捷键配置

---

### 通用 UI 组件

#### 主题切换

**🔤 用户描述方式**:
- 主要: "主题切换", "暗色模式", "亮色模式", "主题设置"
- 别名: "theme toggle", "dark mode", "light mode", "颜色主题"

**📍 代码位置**:
- 主题切换: `ui/src/components/theme-toggle.tsx` - 主题切换组件
- 主题提供者: 使用 `next-themes` 包提供主题管理

**🎨 视觉标识**:
- 外观: 月亮/太阳图标按钮,通常在顶部导航栏
- 文本: "暗色模式"、"亮色模式"、"系统主题"
- 功能: 切换亮色/暗色主题、跟随系统主题

**⚡ 修改指引**:
- 修改切换组件: 编辑 `ui/src/components/theme-toggle.tsx`
- 修改主题配置: 编辑 `ui/src/App.tsx` 中的 `ThemeProvider` 配置
- 添加自定义主题: 修改 `ui/src/styles/themes/` 中的主题文件

---

## 🔧 后端 API 路由总览

### 认证相关 (无需认证)
- `POST /api/auth/login/password` - 密码登录
- `POST /api/auth/refresh` - 刷新令牌
- `GET /api/auth/providers` - 获取OAuth提供商列表
- `GET /api/config` - 获取应用配置(登录页使用)

### 认证相关 (需要认证)
- `GET /api/auth/user` - 获取当前用户信息
- `POST /api/auth/logout` - 退出登录

### Kubernetes 子系统 (需要认证)
- `GET /api/v1/clusters` - 获取集群列表
- `GET /api/v1/cluster/:clusterid/overview` - 集群概览
- `GET /api/v1/cluster/:clusterid/pods` - Pod列表(及其他K8s资源)
- `GET /api/v1/cluster/:clusterid/search` - 全局搜索
- `GET /api/v1/cluster/:clusterid/logs/:namespace/:podName/ws` - Pod日志(WebSocket)
- `GET /api/v1/cluster/:clusterid/terminal/:namespace/:podName/ws` - Pod终端(WebSocket)

### 主机管理 (VMs) (需要认证)
- `POST /api/v1/vms/hosts` - 创建主机
- `GET /api/v1/vms/hosts` - 主机列表
- `GET /api/v1/vms/hosts/:id` - 主机详情
- `PUT /api/v1/vms/hosts/:id` - 更新主机
- `DELETE /api/v1/vms/hosts/:id` - 删除主机
- `GET /api/v1/vms/hosts/:id/state/current` - 当前状态
- `GET /api/v1/vms/hosts/:id/state/history` - 历史状态
- `GET /api/v1/vms/webssh/:session_id` - WebSSH终端(WebSocket)
- `GET /api/v1/vms/ws/hosts/monitor` - 实时监控(WebSocket)

---

## 📚 常用修改场景示例

### 场景 1: 修改登录页的品牌标识

**用户描述**: "我想修改登录页的Logo和应用名称"

**定位步骤**:
1. 在映射表中搜索"登录页"
2. 找到 `ui/src/pages/login.tsx`
3. 修改Logo: 替换 `ui/src/assets/logo.png`
4. 修改应用名称: 编辑 `config.yaml` 中的 `app_name` 和 `app_subtitle`

---

### 场景 2: 在主机列表添加新的筛选条件

**用户描述**: "主机列表需要按地域筛选"

**定位步骤**:
1. 在映射表中搜索"主机列表"
2. 找到 `ui/src/pages/hosts/host-list-page.tsx`
3. 在 `HostFormData` 类型中添加 `region` 字段
4. 在后端 `internal/models/host_node.go` 中添加 `Region` 字段
5. 在列表页添加地域筛选器组件

---

### 场景 3: 修改WebSSH终端的颜色主题

**用户描述**: "SSH终端太亮了,想改成绿色主题"

**定位步骤**:
1. 在映射表中搜索"WebSSH终端"
2. 找到 `ui/src/pages/hosts/host-ssh-page.tsx`
3. 在 xterm.js 初始化配置中修改 `theme` 选项

---

## 🎯 快速查找技巧

### 按功能类型查找
- **界面元素**: 搜索组件名称(如"卡片"、"表格"、"按钮")
- **数据展示**: 搜索"列表"、"详情"、"图表"
- **交互功能**: 搜索"添加"、"删除"、"编辑"、"搜索"
- **管理功能**: 搜索"管理"、"配置"、"设置"

### 按子系统查找
- **主机监控**: 搜索"主机"、"VMs"、"监控"、"SSH"
- **Kubernetes**: 搜索"K8s"、"Pod"、"集群"、"容器"
- **实例管理**: 搜索"实例"、"MinIO"、"数据库"、"Docker"
- **用户权限**: 搜索"用户"、"角色"、"权限"、"OAuth"

### 按技术组件查找
- **前端页面**: `ui/src/pages/` 目录
- **UI组件**: `ui/src/components/` 目录
- **后端API**: `internal/api/handlers/` 目录
- **业务逻辑**: `internal/services/` 目录
- **数据模型**: `internal/models/` 目录

---

## 📝 注意事项

1. **前后端分离**: 前端在 `ui/` 目录,后端在 `internal/` 和 `pkg/` 目录
2. **API版本**: 所有API以 `/api/v1` 为前缀
3. **认证**: 大部分API需要JWT认证,通过 `Authorization: Bearer <token>` header
4. **实时数据**: 监控数据、日志、终端等使用WebSocket实现实时通信
5. **状态管理**: 前端使用React Query(服务端状态) + Zustand(客户端状态)
6. **类型安全**: 前端TypeScript严格模式,后端Go强类型

---

*本文档由AI自动生成,用于帮助AI更好地理解和修改Tiga项目代码。*

**文档生成时间**: 2025-10-07
**项目版本**: Tiga DevOps Dashboard (主机管理子系统开发中)
