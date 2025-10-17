# 任务：K8s子系统完整实现（从Kite迁移）

**输入**：来自 `.claude/specs/005-k8s-kite-k8s/` 的设计文档
**前提条件**：✅ plan.md、✅ research.md、✅ data-model.md、✅ contracts/、✅ quickstart.md

**技术栈**：Go 1.24+、Gin、GORM、client-go v0.31.4、React 19、TypeScript
**项目类型**：Web 应用（后端 Go + 前端 React）
**总工作量**：25 个工作日（5 个阶段）

---

## 格式说明

- **[P]**：可以并行运行（不同文件，无依赖关系）
- **文件路径**：所有路径相对于仓库根目录 `/Users/ysicing/go/src/github.com/ysicing/tiga`

---

## 阶段 3.1：设置（Phase 0 基础）

### 依赖安装和配置

- [ ] **T001** [P] 添加 OpenKruise SDK 依赖到 `go.mod`
  ```bash
  go get github.com/openkruise/kruise-api@v1.8.0
  ```

- [ ] **T002** [P] 配置 Wire 依赖注入（如需要添加新服务）
  - 文件：`internal/app/wire.go`
  - 为 K8s 服务创建 Provider Set

- [ ] **T003** [P] 创建项目结构目录
  ```bash
  mkdir -p internal/services/k8s
  mkdir -p internal/services/prometheus
  mkdir -p internal/api/handlers/cluster
  mkdir -p pkg/handlers/resources/{kruise,tailscale,traefik,k3s}
  mkdir -p pkg/middleware
  mkdir -p tests/contract/k8s
  mkdir -p tests/integration/k8s
  mkdir -p tests/unit/k8s
  mkdir -p ui/src/pages/k8s/{clusters,resources,monitoring,search}
  mkdir -p ui/src/components/k8s
  mkdir -p ui/src/contexts
  ```

---

## 阶段 3.2：测试优先（TDD）⚠️ 必须在 3.3 之前完成

**关键：这些测试必须编写并且必须在任何实现之前失败**

### 契约测试（API 规范验证）

- [ ] **T004** [P] 契约测试：GET /api/v1/k8s/clusters（集群列表）
  - 文件：`tests/contract/k8s/cluster_list_test.go`
  - 验证响应格式、字段类型、状态码

- [ ] **T005** [P] 契约测试：GET /api/v1/k8s/clusters/:id（集群详情）
  - 文件：`tests/contract/k8s/cluster_get_test.go`

- [ ] **T006** [P] 契约测试：POST /api/v1/k8s/clusters（创建集群）
  - 文件：`tests/contract/k8s/cluster_create_test.go`

- [ ] **T007** [P] 契约测试：PUT /api/v1/k8s/clusters/:id（更新集群）
  - 文件：`tests/contract/k8s/cluster_update_test.go`

- [ ] **T008** [P] 契约测试：DELETE /api/v1/k8s/clusters/:id（删除集群）
  - 文件：`tests/contract/k8s/cluster_delete_test.go`

- [ ] **T009** [P] 契约测试：POST /api/v1/k8s/clusters/:id/test-connection（测试连接）
  - 文件：`tests/contract/k8s/cluster_test_connection_test.go`

- [ ] **T010** [P] 契约测试：POST /api/v1/k8s/clusters/:id/prometheus/rediscover（重新检测）
  - 文件：`tests/contract/k8s/prometheus_rediscover_test.go`

- [ ] **T011** [P] 契约测试：GET /api/v1/k8s/clusters/:cluster_id/clonesets（CloneSet 列表）
  - 文件：`tests/contract/k8s/cloneset_list_test.go`

- [ ] **T012** [P] 契约测试：PUT /api/v1/k8s/clusters/:cluster_id/clonesets/:name/scale（CloneSet 扩容）
  - 文件：`tests/contract/k8s/cloneset_scale_test.go`

- [ ] **T013** [P] 契约测试：POST /api/v1/k8s/clusters/:cluster_id/clonesets/:name/restart（CloneSet 重启）
  - 文件：`tests/contract/k8s/cloneset_restart_test.go`

- [ ] **T014** [P] 契约测试：GET /api/v1/k8s/clusters/:cluster_id/tailscale/connectors（Tailscale Connector 列表）
  - 文件：`tests/contract/k8s/tailscale_connector_test.go`

- [ ] **T015** [P] 契约测试：GET /api/v1/k8s/clusters/:cluster_id/traefik/ingressroutes（Traefik IngressRoute 列表）
  - 文件：`tests/contract/k8s/traefik_ingressroute_test.go`

- [ ] **T016** [P] 契约测试：GET /api/v1/k8s/clusters/:cluster_id/search（全局搜索）
  - 文件：`tests/contract/k8s/search_test.go`

- [ ] **T017** [P] 契约测试：GET /api/v1/k8s/clusters/:cluster_id/crds（CRD 检测）
  - 文件：`tests/contract/k8s/crd_detection_test.go`

### 集成测试（业务场景验证）

- [ ] **T018** [P] 集成测试：集群健康检查（对应 quickstart.md V1.2）
  - 文件：`tests/integration/k8s/cluster_health_test.go`
  - 使用 testcontainers-go 启动 Kind 集群
  - 验证 health_status 从 "unknown" 变为 "healthy"
  - 验证 node_count 和 pod_count 统计

- [ ] **T019** [P] 集成测试：Prometheus 异步自动发现（对应验收场景 2）
  - 文件：`tests/integration/k8s/prometheus_discovery_test.go`
  - 部署 Prometheus Operator 到测试集群
  - 添加集群，等待 30 秒
  - 验证 prometheus_url 字段有值

- [ ] **T020** [P] 集成测试：CloneSet 扩缩容（对应验收场景 1）
  - 文件：`tests/integration/k8s/cloneset_scale_test.go`
  - 创建 3 副本的 CloneSet
  - 扩容到 5 副本
  - 验证 30 秒内显示 5 个运行中的 Pods

- [ ] **T021** [P] 集成测试：全局搜索性能（对应验收场景 5）
  - 文件：`tests/integration/k8s/search_performance_test.go`
  - 创建 50+ 命名空间，1000+ 资源
  - 搜索 "redis"
  - 验证响应时间 <1 秒

- [ ] **T022** [P] 集成测试：节点终端访问（对应验收场景 3）
  - 文件：`tests/integration/k8s/node_terminal_test.go`
  - 创建节点终端会话
  - 验证特权 Pod 创建
  - 验证命令执行（`ls /`）
  - 验证会话清理

---

## 阶段 3.3：核心实现 - Phase 0（仅在测试失败后）

### 数据模型扩展

- [ ] **T023** [P] 扩展 Cluster 模型
  - 文件：`internal/models/cluster.go`
  - 添加字段：`HealthStatus`、`LastConnectedAt`、`NodeCount`、`PodCount`、`PrometheusURL`
  - 添加索引：`health_status`
  - 添加验证规则（Prometheus URL 格式验证）

- [ ] **T024** [P] 扩展配置结构
  - 文件：`internal/config/config.go`
  - 添加 `KubernetesConfig`（NodeTerminalImage、EnableKruise、EnableTailscale、EnableTraefik、EnableK3sUpgrade）
  - 添加 `PrometheusConfig`（AutoDiscovery、DiscoveryTimeout、ClusterURLs）
  - 添加 `FeaturesConfig`（ReadonlyMode）

### 集群管理服务

- [ ] **T025** 实现 K8s Client 实例缓存
  - 文件：`pkg/kube/client.go`
  - 实现 `ClientCache` 结构体（map[uint]*K8sClient）
  - 实现 `GetOrCreate(cluster *models.Cluster)` 方法（双检锁模式）
  - 实现 `RemoveClient(clusterID uint)` 方法

- [ ] **T026** [P] 实现集群健康检查服务
  - 文件：`internal/services/k8s/cluster_health.go`
  - 后台 Goroutine，60 秒间隔
  - 调用 `GET /api/v1/nodes` 获取节点列表
  - 更新 `health_status`、`last_connected_at`、`node_count`、`pod_count`
  - 状态转换逻辑（unknown → healthy → warning → error → unavailable）

### API Handler 实现

- [ ] **T027** 实现集群列表 API Handler
  - 文件：`internal/api/handlers/cluster/list.go`
  - 路由：`GET /api/v1/k8s/clusters`
  - 返回所有集群（包含健康状态和统计信息）
  - 使用 ClusterRepository 查询数据库

- [ ] **T028** 实现集群详情 API Handler
  - 文件：`internal/api/handlers/cluster/get.go`
  - 路由：`GET /api/v1/k8s/clusters/:id`

- [ ] **T029** 实现创建集群 API Handler
  - 文件：`internal/api/handlers/cluster/create.go`
  - 路由：`POST /api/v1/k8s/clusters`
  - 验证 Kubeconfig 格式（解析 YAML，检查必需字段）
  - 测试集群连接
  - 保存到数据库
  - 触发 Prometheus 异步发现任务

- [ ] **T030** 实现更新集群 API Handler
  - 文件：`internal/api/handlers/cluster/update.go`
  - 路由：`PUT /api/v1/k8s/clusters/:id`
  - 清除 Client 缓存（如果更新 Kubeconfig）
  - 停止自动发现任务（如果手动设置 prometheus_url）

- [ ] **T031** 实现删除集群 API Handler
  - 文件：`internal/api/handlers/cluster/delete.go`
  - 路由：`DELETE /api/v1/k8s/clusters/:id`
  - 软删除（设置 deleted_at）
  - 清除 Client 缓存
  - 停止自动发现任务

- [ ] **T032** 实现测试集群连接 API Handler
  - 文件：`internal/api/handlers/cluster/test_connection.go`
  - 路由：`POST /api/v1/k8s/clusters/:id/test-connection`
  - 调用 `GET /api/v1/namespaces` 测试连接
  - 返回 Kubernetes 版本、节点数

### 中间件

- [ ] **T033** [P] 实现集群上下文中间件
  - 文件：`pkg/middleware/cluster_context.go`
  - 从 HTTP Header `X-Cluster-ID` 或查询参数 `cluster` 读取集群 ID
  - 验证集群存在性
  - 验证用户对集群的访问权限
  - 在 Context 中设置 cluster_id

---

## 阶段 3.3：核心实现 - Phase 1（Prometheus 增强）

### Prometheus 自动发现服务

- [ ] **T034** 实现 Prometheus 发现服务
  - 文件：`internal/services/prometheus/discovery.go`
  - 实现 `StartDiscoveryTask(ctx context.Context, cluster *models.Cluster)` 方法
  - 异步 Goroutine，30 秒超时
  - 搜索 Service（monitoring、prometheus 等命名空间）
  - 测试连通性（`GET /api/v1/status/config`，2 秒超时）
  - 选择最佳端点（LoadBalancer > Ingress > NodePort > ClusterIP）
  - 保存 Prometheus URL 到数据库

- [ ] **T035** 实现 Prometheus 发现任务管理器
  - 文件：`internal/services/prometheus/task_manager.go`
  - 跟踪正在运行的发现任务（cluster_id → task_context）
  - 实现 `Start(clusterID uint)` 方法
  - 实现 `Stop(clusterID uint)` 方法
  - 避免重复启动同一集群的任务

- [ ] **T036** 实现 Prometheus 重新检测 API Handler
  - 文件：`internal/api/handlers/cluster/prometheus_rediscover.go`
  - 路由：`POST /api/v1/k8s/clusters/:id/prometheus/rediscover`
  - 检查是否有正在运行的任务（返回 409 Conflict）
  - 调用任务管理器启动新任务
  - 返回 202 Accepted

- [ ] **T037** [P] 增强 Prometheus 客户端
  - 文件：`pkg/prometheus/client.go`
  - 支持集群级别配置（从 `config.Prometheus.ClusterURLs` 读取）
  - 手动配置优先级高于自动发现结果
  - 实现连接池和重试机制

---

## 阶段 3.3：核心实现 - Phase 2（高级资源和 CRD 支持）

### 通用 CRD 处理器

- [ ] **T038** 实现通用 CRD 处理器框架
  - 文件：`pkg/kube/crd.go`
  - 实现 `CRDHandler` 结构体（ResourceName、CRDName、Kind、Group、Version、Namespaced）
  - 实现 `CheckCRDExists(ctx, client)` 方法
  - 实现 `List(ctx, client, namespace)` 方法（使用 unstructured.Unstructured）
  - 实现 `Get(ctx, client, namespace, name)` 方法
  - 实现 `Create/Update/Delete` 方法

- [ ] **T039** 实现 CRD 检测 API
  - 文件：`internal/api/handlers/k8s/crd_detection.go`
  - 路由：`GET /api/v1/k8s/clusters/:cluster_id/crds`
  - 检测 OpenKruise、Tailscale、Traefik、K3s Upgrade Controller CRD
  - 返回已安装的 CRD 列表

### OpenKruise CRD Handler

- [ ] **T040** [P] 实现 CloneSet Handler
  - 文件：`pkg/handlers/resources/kruise/cloneset.go`
  - 复用通用 CRD 处理器
  - 实现 Scale 操作（`PUT /scale`）
  - 实现 Restart 操作（`POST /restart`）
  - 注册路由到 Gin

- [ ] **T041** [P] 实现 Advanced DaemonSet Handler
  - 文件：`pkg/handlers/resources/kruise/advanced_daemonset.go`
  - 复用通用 CRD 处理器
  - 实现基本 CRUD 操作

- [ ] **T042** [P] 实现 Advanced StatefulSet Handler
  - 文件：`pkg/handlers/resources/kruise/advanced_statefulset.go`
  - 复用通用 CRD 处理器
  - 实现基本 CRUD 操作

### Tailscale CRD Handler（集群级别）

- [ ] **T043** [P] 实现 Tailscale Connector Handler
  - 文件：`pkg/handlers/resources/tailscale/connector.go`
  - 设置 `Namespaced=false`（集群级别资源）
  - 实现基本 CRUD 操作

- [ ] **T044** [P] 实现 Tailscale ProxyClass Handler
  - 文件：`pkg/handlers/resources/tailscale/proxyclass.go`

- [ ] **T045** [P] 实现 Tailscale ProxyGroup Handler
  - 文件：`pkg/handlers/resources/tailscale/proxygroup.go`

### Traefik CRD Handler（命名空间级别）

- [ ] **T046** [P] 实现 Traefik IngressRoute Handler
  - 文件：`pkg/handlers/resources/traefik/ingressroute.go`
  - 设置 `Namespaced=true`（命名空间级别资源）
  - 支持 `namespace=_all` 跨命名空间查询

- [ ] **T047** [P] 实现 Traefik Middleware Handler
  - 文件：`pkg/handlers/resources/traefik/middleware.go`

- [ ] **T048** [P] 实现 Traefik TLSOption Handler
  - 文件：`pkg/handlers/resources/traefik/tlsoption.go`

- [ ] **T049** [P] 实现其他 6 个 Traefik CRD Handler（批量实现）
  - 文件：`pkg/handlers/resources/traefik/{ingressroutetcp,ingressrouteudp,middlewaretcp,tlsstore,traefikservice,serverstransport}.go`
  - 复用通用 CRD 处理器
  - 实现基本 CRUD 操作

### K3s System Upgrade Controller Handler

- [ ] **T050** [P] 实现 K3s Plan Handler
  - 文件：`pkg/handlers/resources/k3s/plan.go`
  - 设置 `Namespaced=true`（命名空间级别资源）
  - 实现基本 CRUD 操作

### 菜单动态显示逻辑

- [ ] **T051** [P] 实现前端菜单动态显示逻辑
  - 文件：`ui/src/components/k8s/DynamicMenu.tsx`
  - 调用 CRD 检测 API
  - 根据返回结果显示/隐藏菜单项（OpenKruise、Tailscale、Traefik）

---

## 阶段 3.3：核心实现 - Phase 3（资源增强和搜索）

### 资源关系服务

- [ ] **T052** [P] 实现资源关系服务
  - 文件：`internal/services/k8s/relations.go`
  - 定义静态关系映射（Deployment → ReplicaSet → Pod）
  - 实现递归查询（限制最大深度 3）
  - 使用 `ownerReferences` 字段追踪父子关系
  - 检测循环引用（记录已访问的资源 UID）

### 缓存服务

- [ ] **T053** [P] 实现工作负载缓存服务
  - 文件：`internal/services/k8s/cache.go`
  - 缓存键：`cluster_id:resource_type`
  - 缓存值：资源列表（JSON）
  - 过期时间：5 分钟
  - 实现手动刷新接口
  - 实现 ResourceVersion 检测（缓存失效）

### 全局搜索服务

- [ ] **T054** 实现全局搜索服务
  - 文件：`internal/services/k8s/search.go`
  - 并发查询 4 个资源类型（Pod、Deployment、Service、ConfigMap）
  - 评分算法：精确匹配 100 分、名称包含 80 分、标签匹配 60 分、注解匹配 40 分
  - 结果按评分降序排列
  - 限制返回前 50 条结果
  - 缓存搜索结果（5 分钟有效期）

- [ ] **T055** 实现搜索 API Handler
  - 文件：`internal/api/handlers/k8s/search.go`
  - 路由：`GET /api/v1/k8s/clusters/:cluster_id/search`
  - 支持查询参数：`q`、`types`、`namespace`、`limit`
  - 10 秒超时控制

---

## 阶段 3.3：核心实现 - Phase 4（终端和只读模式）

### 节点终端

- [ ] **T056** 实现节点终端 Handler
  - 文件：`pkg/kube/node_terminal.go`
  - 创建特权 Pod（hostNetwork、hostPID、privileged）
  - 建立 WebSocket 连接
  - 支持完整的终端交互（Ctrl+C、Tab 补全）
  - 实现 30 分钟超时清理（自动断开、清理 Pod）

### 只读模式中间件

- [ ] **T057** [P] 实现只读模式中间件
  - 文件：`pkg/middleware/readonly.go`
  - 阻止 POST、PUT、PATCH、DELETE 请求
  - 返回清晰的错误信息（"只读模式已启用"）
  - 从配置读取 `features.readonly_mode`

### 审计日志增强

- [ ] **T058** 增强审计日志
  - 文件：`internal/models/audit_log.go`（扩展现有）
  - 添加集群名称字段
  - 记录所有资源修改操作
  - 记录所有节点终端访问

---

## 阶段 3.4：集成

### Wire 依赖注入集成

- [ ] **T059** 集成 Wire 依赖注入
  - 文件：`internal/app/wire.go`
  - 添加 K8s 服务到 ServiceSet
  - 添加 Prometheus 发现服务到 ServiceSet
  - 运行 `task wire` 重新生成 `wire_gen.go`

### 路由注册

- [ ] **T060** 注册集群管理路由
  - 文件：`internal/api/routes.go`
  - 注册集群 CRUD 路由
  - 应用集群上下文中间件
  - 应用只读模式中间件

- [ ] **T061** 注册 CRD 资源管理路由
  - 文件：`internal/api/routes.go`
  - 注册 OpenKruise、Tailscale、Traefik、K3s 路由
  - 应用集群上下文中间件

### 启动时初始化

- [ ] **T062** 启动时自动导入集群
  - 文件：`internal/app/app.go`（扩展 Initialize 方法）
  - 从 `~/.kube/config` 自动导入集群
  - 启动集群健康检查服务

- [ ] **T063** 启动时启动 Prometheus 发现任务
  - 文件：`internal/app/app.go`
  - 为所有启用的集群启动异步发现任务
  - 仅当 `prometheus.auto_discovery=true` 且无手动配置时启动

---

## 阶段 3.5：前端实现

### 集群管理页面

- [ ] **T064** [P] 实现集群列表页面
  - 文件：`ui/src/pages/k8s/clusters/ClusterListPage.tsx`
  - 显示集群列表（名称、健康状态、节点数、Pod数）
  - 支持创建、编辑、删除集群
  - 支持测试连接、重新检测 Prometheus

- [ ] **T065** [P] 实现集群详情页面
  - 文件：`ui/src/pages/k8s/clusters/ClusterDetailPage.tsx`
  - 显示集群详细信息
  - 显示 Prometheus 发现状态
  - 支持手动配置 Prometheus URL

- [ ] **T066** [P] 实现集群切换器组件
  - 文件：`ui/src/components/k8s/ClusterSelector.tsx`
  - 下拉菜单显示所有集群
  - 切换集群时更新 Context
  - 界面顶部显示"当前集群"名称

- [ ] **T067** [P] 实现 ClusterContext
  - 文件：`ui/src/contexts/ClusterContext.tsx`
  - 维护当前选中的集群 ID
  - 提供 `setCluster(id)` 方法
  - 切换集群时清除缓存和临时状态

### CRD 资源页面

- [ ] **T068** [P] 实现 OpenKruise CloneSet 页面
  - 文件：`ui/src/pages/k8s/resources/kruise/CloneSetPage.tsx`
  - 列表、创建、编辑、删除、扩容、重启功能
  - 使用 TanStack Query 管理数据

- [ ] **T069** [P] 实现 Traefik IngressRoute 页面
  - 文件：`ui/src/pages/k8s/resources/traefik/IngressRoutePage.tsx`
  - 列表、创建、编辑、删除功能
  - 支持关联 Middleware 快速跳转

### 监控和搜索页面

- [ ] **T070** [P] 实现全局搜索页面
  - 文件：`ui/src/pages/k8s/search/SearchPage.tsx`
  - 搜索框输入关键词
  - 按资源类型分组显示结果
  - 点击资源跳转到详情页

- [ ] **T071** [P] 实现资源关系图组件
  - 文件：`ui/src/components/k8s/ResourceRelations.tsx`
  - 可视化资源依赖关系（Deployment → ReplicaSet → Pod）
  - 支持点击节点跳转

- [ ] **T072** [P] 扩展 Prometheus 监控页面
  - 文件：`ui/src/pages/k8s/monitoring/MonitoringPage.tsx`（扩展现有）
  - 显示 Prometheus 发现状态（检测中、已发现、未发现、手动配置）
  - 支持手动重新检测

- [ ] **T073** [P] 扩展节点终端面板
  - 文件：`ui/src/components/k8s/TerminalPanel.tsx`（扩展现有）
  - 节点终端支持（选择节点 → 打开终端）
  - 使用 xterm.js 渲染终端 UI

---

## 阶段 3.6：优化

### 单元测试

- [ ] **T074** [P] 单元测试：资源关系服务
  - 文件：`tests/unit/k8s/relations_test.go`
  - 测试静态关系映射
  - 测试递归查询逻辑
  - 测试循环引用检测

- [ ] **T075** [P] 单元测试：缓存服务
  - 文件：`tests/unit/k8s/cache_test.go`
  - 测试缓存 CRUD 操作
  - 测试过期时间
  - 测试 ResourceVersion 检测

- [ ] **T076** [P] 单元测试：全局搜索评分算法
  - 文件：`tests/unit/k8s/search_test.go`
  - 测试精确匹配、模糊匹配、标签匹配
  - 测试评分排序

- [ ] **T077** [P] 单元测试：Prometheus 发现逻辑
  - 文件：`tests/unit/k8s/prometheus_discovery_test.go`
  - 测试 Service 识别
  - 测试端点优先级选择
  - 测试连通性测试

### 性能测试

- [ ] **T078** 性能测试：API 响应时间
  - 文件：`tests/performance/k8s/api_performance_test.go`
  - 验证资源列表查询 <500ms
  - 验证全局搜索 <1s
  - 验证 WebSocket 终端延迟 <100ms

### 文档生成

- [ ] **T079** [P] 生成 Swagger API 文档
  - 运行：`./scripts/generate-swagger.sh`
  - 验证所有 K8s API 端点已文档化
  - 访问 `http://localhost:12306/swagger/index.html` 验证

- [ ] **T080** [P] 更新 CLAUDE.md
  - 文件：`CLAUDE.md`
  - 添加 K8s 子系统功能说明
  - 更新 API 端点列表
  - 更新常用命令

---

## 阶段 3.7：手动验证

- [ ] **T081** 执行 quickstart.md 验证场景
  - 参考：`.claude/specs/005-k8s-kite-k8s/quickstart.md`
  - V1：集群管理（导入、健康检查、测试连接）
  - V2：Prometheus 自动发现
  - V3：OpenKruise CRD 支持（CloneSet 扩缩容）
  - V4：全局搜索
  - V5：节点终端（需要管理员权限）

- [ ] **T082** 代码质量检查
  - 运行：`task lint`
  - 修复所有 linting 错误
  - 运行：`task test`
  - 确保所有测试通过

---

## 依赖关系

### 关键依赖路径

1. **T023-T024** 阻塞所有后续任务（数据模型是基础）
2. **T025** 阻塞 T026-T032（Client 缓存是集群管理的前提）
3. **T038** 阻塞 T040-T050（通用 CRD 处理器是所有 CRD Handler 的基础）
4. **T052-T054** 可并行实施（资源关系、缓存、搜索是独立模块）
5. **T059-T063** 必须在所有服务实现后执行（集成阶段）
6. **T064-T073** 可与后端任务并行实施（前后端独立开发）
7. **T074-T077** 必须在对应服务实现后执行（单元测试依赖实现）
8. **T081-T082** 必须在所有功能实现后执行（验收阶段）

### 测试依赖

- **契约测试（T004-T017）** 必须在对应 API Handler（T027-T032、T040-T050、T055）之前编写并失败
- **集成测试（T018-T022）** 必须在对应功能实现后执行
- **单元测试（T074-T077）** 必须在对应服务实现后执行

---

## 并行执行示例

### 阶段 3.2：测试优先（所有测试可并行）

```bash
# 同时启动 T004-T017（契约测试）：
Task prompt="在 tests/contract/k8s/cluster_list_test.go 中测试 GET /api/v1/k8s/clusters 契约" subagent_type="general-purpose"
Task prompt="在 tests/contract/k8s/cluster_get_test.go 中测试 GET /api/v1/k8s/clusters/:id 契约" subagent_type="general-purpose"
Task prompt="在 tests/contract/k8s/cluster_create_test.go 中测试 POST /api/v1/k8s/clusters 契约" subagent_type="general-purpose"
# ... 其他 14 个契约测试

# 同时启动 T018-T022（集成测试）：
Task prompt="在 tests/integration/k8s/cluster_health_test.go 中实现集群健康检查集成测试" subagent_type="general-purpose"
Task prompt="在 tests/integration/k8s/prometheus_discovery_test.go 中实现 Prometheus 异步自动发现集成测试" subagent_type="general-purpose"
# ... 其他 3 个集成测试
```

### 阶段 3.3：Phase 0 基础（可并行任务）

```bash
# T023-T024：数据模型扩展（不同文件，可并行）
Task prompt="在 internal/models/cluster.go 中扩展 Cluster 模型，添加健康状态和统计信息字段" subagent_type="general-purpose"
Task prompt="在 internal/config/config.go 中扩展配置结构，添加 KubernetesConfig、PrometheusConfig、FeaturesConfig" subagent_type="general-purpose"

# T026、T033：服务和中间件（不同文件，可并行）
Task prompt="在 internal/services/k8s/cluster_health.go 中实现集群健康检查服务" subagent_type="general-purpose"
Task prompt="在 pkg/middleware/cluster_context.go 中实现集群上下文中间件" subagent_type="general-purpose"
```

### 阶段 3.3：Phase 2 CRD Handler（可并行任务）

```bash
# T040-T042：OpenKruise Handler（不同文件，可并行）
Task prompt="在 pkg/handlers/resources/kruise/cloneset.go 中实现 CloneSet Handler" subagent_type="general-purpose"
Task prompt="在 pkg/handlers/resources/kruise/advanced_daemonset.go 中实现 Advanced DaemonSet Handler" subagent_type="general-purpose"
Task prompt="在 pkg/handlers/resources/kruise/advanced_statefulset.go 中实现 Advanced StatefulSet Handler" subagent_type="general-purpose"

# T043-T045：Tailscale Handler（不同文件，可并行）
# T046-T049：Traefik Handler（不同文件，可并行）
# ... 共 11 个 CRD Handler 可并行实施
```

### 阶段 3.5：前端实现（所有前端任务可并行）

```bash
# T064-T073：所有前端任务（不同文件，可并行）
Task prompt="在 ui/src/pages/k8s/clusters/ClusterListPage.tsx 中实现集群列表页面" subagent_type="frontend"
Task prompt="在 ui/src/components/k8s/ClusterSelector.tsx 中实现集群切换器组件" subagent_type="frontend"
Task prompt="在 ui/src/pages/k8s/search/SearchPage.tsx 中实现全局搜索页面" subagent_type="frontend"
# ... 其他 7 个前端任务
```

---

## 注意事项

### TDD 原则

- ✅ **所有契约测试（T004-T017）** 必须在对应 API Handler 实现之前编写
- ✅ **所有集成测试（T018-T022）** 必须在功能实现后验证
- ✅ 验证测试失败后再开始实现

### 并行执行规则

- ✅ **[P] 任务**：不同文件，无依赖关系，可以并行执行
- ❌ **无 [P] 任务**：同一文件或有依赖关系，必须顺序执行

### 提交策略

- 每完成一个任务后提交一次 Git commit
- Commit message 格式：`[K8s][T0XX] <任务描述>`
- 示例：`[K8s][T023] 扩展 Cluster 模型添加健康状态字段`

### 避免的陷阱

- ❌ 不要在实现之前跳过测试编写
- ❌ 不要在同一文件上并行执行多个任务
- ❌ 不要在未验证测试失败的情况下开始实现
- ❌ 不要忘记清理资源（Client 缓存、Goroutine、WebSocket 连接）

---

## 验证清单

**门禁：在标记任务完成前检查**

- [ ] 所有契约测试（T004-T017）都已编写并通过
- [ ] 所有集成测试（T018-T022）都已编写并通过
- [ ] 所有实体（Cluster、ConfigExtension）都已扩展
- [ ] 所有 API 端点都已实现
- [ ] 所有 CRD Handler 都已实现
- [ ] 前端页面与后端 API 集成成功
- [ ] 所有 quickstart.md 验收场景通过
- [ ] 代码质量检查通过（`task lint`）
- [ ] 所有测试通过（`task test`）

---

## 预估完成时间

**总工作量**：25 个工作日
- 阶段 3.1（设置）：0.5 天
- 阶段 3.2（测试优先）：3 天
- 阶段 3.3（核心实现）：15 天
  - Phase 0：3 天
  - Phase 1：3 天
  - Phase 2：5 天
  - Phase 3：2 天
  - Phase 4：2 天
- 阶段 3.4（集成）：1 天
- 阶段 3.5（前端实现）：3 天（与后端并行）
- 阶段 3.6（优化）：1.5 天
- 阶段 3.7（手动验证）：1 天

**缓冲时间**：8 天（用于处理意外问题、代码审查、重构）

---

**文档版本**：v1.0
**最后更新**：2025-10-17
**作者**：Claude Code
**下一步**：开始执行 T001
