# 功能规格：K8s子系统完整实现（从Kite迁移）

**功能分支**：`005-k8s-kite-k8s`
**创建日期**：2025-10-17
**状态**：草稿
**输入**：用户描述："完成k8s子系统开发，将之前kite关于k8s的功能移植到k8s"

## 执行流程（main）
```
1. 从输入解析用户描述
   → 已提供：迁移Kite项目的K8s功能到Tiga
2. 从描述中提取关键概念
   → 识别：Kite项目、K8s管理功能、功能迁移
3. 对于每个不清晰的方面：
   → 已通过代码分析确定具体功能范围
4. 填充用户场景与测试部分
   → 已定义7个核心用户故事（包含US-1.5高级CRD使用场景）
5. 生成功能需求
   → 已生成功能需求（6大类、60子需求：FR-0有10个、FR-1有14个、FR-2有9个、FR-3有6个、FR-4有7个、FR-5有7个、FR-6有7个）
6. 识别关键实体（如果涉及数据）
   → 配置扩展（多集群管理）
7. 运行审查清单
   → ✅ 无需要澄清项
   → ✅ 无实现细节（仅描述功能）
8. 返回：成功（规格已准备好进行规划）
```

---

## ⚡ 快速概述

**功能目标**：将Kite项目中经过验证的K8s管理功能完整迁移到Tiga，打造功能完善的K8s子系统

**价值主张**：
- 提供企业级K8s资源管理能力
- 支持OpenKruise高级工作负载（原地升级、增强版DaemonSet/StatefulSet）
- 支持Traefik高级路由管理（IngressRoute、Middleware、TLS配置等）
- 支持Tailscale零信任网络资源（Connector、ProxyClass、ProxyGroup）
- 支持K3s集群升级编排（System Upgrade Controller）
- 零配置Prometheus监控集成（异步自动发现）
- 全局搜索和资源关系可视化
- 节点级别调试能力
- **多集群统一管理**

**功能范围**：
- ✅ **多集群管理**（集群列表、切换、独立上下文）
- ✅ **高级CRD支持**（OpenKruise、Tailscale、Traefik、K3s Upgrade Controller）
- ✅ Prometheus智能自动发现（集群级别、异步后台检测）
- ✅ 资源关系可视化
- ✅ 全局资源搜索
- ✅ 节点终端
- ✅ 只读模式
- ❌ Webhook事件处理（本期不做）
- ❌ Admission Controller统一视图（个人用户场景不适用）
- ❌ OAuth认证（Tiga已有）
- ❌ 前端UI（保持Tiga现有）

---

## 用户场景与测试 *（必填）*

### 主要用户故事

**US-1: OpenKruise工作负载管理（DevOps工程师）**

张工程师想要利用OpenKruise的高级特性（如原地升级）来管理公司的微服务集群。他访问Tiga K8s子系统,发现界面上显示了CloneSet列表。他点击"扩容"按钮，将副本数从3调整到5，系统立即生效。他还可以通过"滚动重启"功能重启所有Pod而不影响服务可用性。

**US-1.5: Traefik路由管理（平台工程师）**

刘工程师负责公司K8s集群的流量入口管理，集群使用Traefik作为Ingress Controller。他需要为新部署的API服务配置路由规则。他在Tiga K8s子系统中打开"IngressRoute"页面，系统自动检测到集群已安装Traefik CRD，显示现有的路由列表。他点击"创建"，填写路由规则：
- 域名：api.example.com
- 路径：/v2/*
- 后端服务：api-service:8080
- 中间件：rate-limit-middleware（限流）

保存后，系统立即创建IngressRoute资源，流量开始路由到新服务。他在详情页中看到关联的Middleware配置，点击可以快速跳转编辑限流参数。

**US-2: 零配置监控（集群管理员）**

李管理员刚接手一个新集群，他不确定Prometheus是否已部署、部署在哪个命名空间、使用什么访问方式。他在Tiga中添加集群后，系统在后台异步启动Prometheus服务发现任务。几秒钟后，他打开K8s监控页面，系统显示"已自动发现Prometheus服务"，并成功展示集群的CPU、内存使用历史图表。整个过程无需任何手动配置。

如果系统未能自动发现Prometheus（例如非标准部署），监控页面会显示"未发现Prometheus服务，请手动配置"，并提供配置表单。他手动输入Prometheus地址后，立即可以查看监控数据。

**US-3: 节点级别调试（SRE）**

王工程师发现某个节点的网络异常，但通过Pod终端无法排查。他打开节点终端功能，直接进入节点的Shell环境，执行`ip route`、`netstat`等命令，快速定位到是iptables规则冲突导致的问题。排查完成后关闭终端，系统自动清理了临时Pod。

**US-4: 资源关系追踪（开发人员）**

陈开发发现某个Service返回502错误，他在Tiga中打开这个Service的详情页，点击"相关资源"标签，系统显示了：
- 该Service选择的3个Pods（其中1个处于CrashLoopBackOff状态）
- 这些Pods来自哪个Deployment
- 该Deployment是否有HPA控制

他点击问题Pod的名称，直接跳转到Pod详情页，查看日志发现是内存溢出。

**US-5: 全局搜索（DevOps工程师）**

赵工程师记得某个ConfigMap包含数据库连接信息，但忘记了具体名称和命名空间。他在搜索框输入"mysql"，1秒内返回结果：
- 5个包含"mysql"的ConfigMaps
- 3个名称包含"mysql"的Deployments
- 2个包含"mysql"标签的Services

他点击第一个ConfigMap，快速找到了需要的配置信息。

**US-6: 多集群管理（集群管理员）**

周管理员负责管理公司的3个K8s集群（生产、预发布、开发）。他打开Tiga K8s子系统，首先看到集群列表页面：
- **生产集群**（50个命名空间、500+ Pods）- 状态：健康
- **预发布集群**（20个命名空间、200+ Pods）- 状态：健康
- **开发集群**（10个命名空间、100+ Pods）- 状态：警告（1个节点异常）

他点击"生产集群"进入集群视图，界面顶部显示"当前集群：生产"。他查看Deployments列表、修改副本数、查看Pod日志，所有操作都在生产集群上下文中执行。然后他点击集群切换按钮，选择"开发集群"，界面立即切换到开发集群的资源视图，Prometheus监控自动切换到开发集群的Prometheus实例。

### 验收场景

#### 场景1：CloneSet扩缩容
1. **假定** 集群中存在一个3副本的CloneSet "app-v1"
2. **当** 用户在界面上选择该CloneSet并点击"扩容"，设置副本数为5
3. **那么** 系统调用Kruise API将副本数更新为5，并在30秒内显示5个运行中的Pods

#### 场景2：Prometheus异步自动发现
1. **假定** 用户刚添加一个新集群到Tiga，集群的`monitoring`命名空间部署了Prometheus Operator
2. **当** 系统在后台异步运行Prometheus服务发现任务（添加集群后触发）
3. **那么** 系统在10秒内检测到`prometheus-operated`服务，测试连通性成功，将Prometheus URL自动保存到集群配置。用户访问监控页面时，直接显示CPU/内存使用趋势图表，无需等待或手动配置

#### 场景3：节点终端访问
1. **假定** 用户具有管理员权限
2. **当** 用户点击节点列表中的"终端"按钮，选择节点"node-01"
3. **那么** 系统在该节点上创建特权Pod，建立WebSocket连接，浏览器中显示终端界面，用户可以执行`ls /`等命令

#### 场景4：资源关系查询
1. **假定** 存在Deployment "nginx" → ReplicaSet "nginx-7d64c5" → Pods 3个
2. **当** 用户查看Deployment "nginx"的详情页，点击"相关资源"标签
3. **那么** 系统显示1个ReplicaSet、3个Pods，以及可能存在的HPA和Services

#### 场景5：全局搜索
1. **假定** 集群中有50个命名空间，1000+资源
2. **当** 用户在搜索框输入"redis"
3. **那么** 系统在1秒内返回所有包含"redis"关键词的Pods、Deployments、Services、ConfigMaps，按资源类型分组显示

#### 场景6：多集群切换
1. **假定** 系统中配置了3个集群（production、staging、dev）
2. **当** 用户在集群列表页面点击"production"，然后点击集群切换按钮选择"staging"
3. **那么** 界面顶部显示"当前集群：staging"，资源列表、搜索、监控等所有功能都在staging集群上下文中执行

### 边缘情况

- **Prometheus异步发现中**：用户添加集群后立即访问监控页面，系统显示"正在检测Prometheus服务，请稍候..."，后台任务完成后自动刷新显示监控数据
- **Prometheus未发现**：异步任务在30秒内未检测到Prometheus服务，监控页面显示"未发现Prometheus服务，请手动配置"，提供配置表单和指南链接
- **Prometheus手动配置优先**：用户手动配置Prometheus URL后，系统停止该集群的自动发现任务，使用手动配置的地址
- **Prometheus连通性失败**：自动发现到的Prometheus端点连通性测试失败，系统继续尝试其他候选端点（NodePort、ClusterIP等），全部失败后提示手动配置
- **Kruise未安装**：CloneSet等菜单自动隐藏，不影响其他功能
- **Traefik未安装**：IngressRoute等Traefik资源菜单自动隐藏
- **Tailscale未安装**：Connector等Tailscale资源菜单自动隐藏
- **CRD不存在时的操作请求**：系统返回"CustomResourceDefinition not found"，引导用户检查集群CRD安装情况
- **CRD版本不兼容**：系统检测到CRD API版本与处理器不匹配，显示版本信息并建议升级
- **节点终端会话超时**：30分钟无操作后自动断开连接，清理临时Pod
- **搜索查询超时**：10秒内未返回结果，显示"搜索超时，请缩小搜索范围"
- **集群连接失败**：系统显示"无法连接到集群 XXX，请检查网络和凭证"，集群状态标记为"不可用"，Prometheus自动发现任务暂停
- **资源关系循环引用**：系统检测到循环依赖，限制最大查询深度为3层
- **缓存失效延迟**：资源被手动修改后，缓存可能有5秒延迟，用户可以手动刷新
- **切换集群时的上下文丢失**：用户切换集群时，搜索历史、过滤条件等临时状态会被清除
- **集群级别CRD的命名空间过滤**：用户尝试按命名空间过滤Tailscale Connector等集群级别资源，系统忽略命名空间参数并提示"此资源为集群级别"
- **并发编辑冲突**：用户A和用户B同时编辑同一资源（如Deployment副本数），用户A先保存成功。用户B保存时，Kubernetes API Server 返回 409 Conflict 错误（ResourceVersion 不匹配），系统提示"资源已被修改，请刷新后重试"
- **并发删除冲突**：用户A正在查看资源详情页，用户B删除了该资源。用户A尝试操作时，系统返回 404 错误并提示"资源已被删除"
- **Prometheus发现失败不重试**：系统在添加集群后尝试自动发现 Prometheus，30秒内失败后不再自动重试。用户可在集群详情页或监控页面点击"重新检测"按钮手动触发新的发现任务

---

## 澄清

### 会话 2025-10-17

- 问：集群在Tiga系统中如何唯一标识？ → 答：数据库表自增ID
- 问：多集群管理的可扩展性限制是什么？ → 答：无硬性限制（仅受数据库和服务器资源约束）
- 问：当多个用户同时操作同一集群的同一资源时，如何处理冲突？ → 答：依赖 Kubernetes API Server 的默认行为（ResourceVersion 验证）
- 问：Prometheus 自动发现任务失败后的重试策略是什么？ → 答：一次性尝试，失败后不重试（用户可手动触发重新检测）

---

## 需求 *（必填）*

### 功能需求

#### FR-0: 多集群管理

- **FR-0.1**：系统必须在首页显示集群列表（包含集群名称、节点数、Pod数、状态）
- **FR-0.2**：用户必须能够点击集群进入集群视图
- **FR-0.3**：系统必须在界面顶部显示当前操作的集群名称
- **FR-0.4**：用户必须能够通过集群切换器切换到其他集群
- **FR-0.5**：系统必须在切换集群时清除当前集群的缓存和临时状态
- **FR-0.6**：系统必须为每个集群维护独立的K8s Client实例
- **FR-0.7**：系统必须在集群列表中显示集群健康状态（健康/警告/错误/不可用）
- **FR-0.8**：系统必须支持从kubeconfig自动导入集群
- **FR-0.9**：用户必须能够添加、编辑、删除集群配置
- **FR-0.10**：系统必须在集群不可用时显示清晰的错误信息

#### FR-1: 高级资源和CRD管理

- **FR-1.1**：系统必须支持CloneSet资源的列表、创建、查看、编辑、删除操作
- **FR-1.2**：用户必须能够对CloneSet执行Scale操作（修改副本数）
- **FR-1.3**：用户必须能够对CloneSet执行Restart操作（滚动重启）
- **FR-1.4**：系统必须支持Advanced DaemonSet资源的完整CRUD操作
- **FR-1.5**：系统必须支持Advanced StatefulSet资源的完整CRUD操作
- **FR-1.6**：系统必须在Kruise未安装时自动隐藏Kruise相关菜单
- **FR-1.7**：系统必须为所有Kruise操作提供统一的成功/错误反馈
- **FR-1.8**：系统必须实现通用CRD处理器模式（在操作前检查CRD存在性，使用unstructured.Unstructured处理动态资源）
- **FR-1.9**：系统必须支持Tailscale资源管理（Connector、ProxyClass、ProxyGroup），所有资源为集群级别（cluster-scoped）
- **FR-1.10**：系统必须支持Traefik资源管理（IngressRoute、IngressRouteTCP、IngressRouteUDP、Middleware、MiddlewareTCP、TLSOption、TLSStore、TraefikService、ServersTransport），所有资源为命名空间级别（namespace-scoped）
- **FR-1.11**：系统必须支持K3s系统升级控制器资源（Plan CRD），用于集群升级编排
- **FR-1.12**：系统必须自动检测集群中安装的CRD，未安装时自动隐藏对应菜单项
- **FR-1.13**：系统必须正确处理集群级别CRD（无命名空间过滤）和命名空间级别CRD（支持命名空间过滤和跨命名空间查询）
- **FR-1.14**：系统必须在CRD不存在时返回清晰错误（"CustomResourceDefinition not found"），而非K8s API错误

#### FR-2: Prometheus智能监控

- **FR-2.1**：系统必须在添加集群后启动异步后台任务，自动发现集群中的Prometheus服务（在monitoring、prometheus等常见命名空间）
- **FR-2.2**：系统必须支持通过Service名称、标签、端口识别Prometheus服务
- **FR-2.3**：系统必须支持多种Prometheus访问方式（ClusterIP、NodePort、LoadBalancer、Ingress）
- **FR-2.4**：系统必须实现优先级算法，选择最佳Prometheus端点（LoadBalancer > Ingress > NodePort > ClusterIP）
- **FR-2.5**：系统必须测试Prometheus端点连通性，仅保存可用端点到集群配置
- **FR-2.6**：系统必须支持用户手动配置Prometheus URL，手动配置优先级高于自动发现结果
- **FR-2.7**：系统必须在异步发现失败时静默处理（不阻塞用户操作），在监控页面提示"正在检测Prometheus服务..."或"未发现Prometheus，请手动配置"
- **FR-2.8**：系统必须为每个集群独立运行Prometheus发现任务，支持集群特定配置覆盖全局配置
- **FR-2.9**：自动发现任务失败后不自动重试，用户可通过"重新检测"按钮手动触发新的发现任务

#### FR-3: 资源增强

- **FR-3.1**：系统必须显示资源间的依赖关系（如Deployment → ReplicaSets → Pods）
- **FR-3.2**：用户必须能够点击相关资源快速跳转到详情页
- **FR-3.3**：系统必须实现工作负载资源的缓存机制，提升查询性能（缓存有效期5分钟）
- **FR-3.4**：系统必须支持手动刷新缓存
- **FR-3.5**：系统必须为常见资源类型（Pods、Deployments、Services等）提供通用CRUD接口
- **FR-3.6**：系统必须支持特殊资源的专用操作（如Node的Drain、HPA的手动触发等）

#### FR-4: 终端增强

- **FR-4.1**：管理员用户必须能够打开K8s节点的终端（通过浏览器WebSocket）
- **FR-4.2**：系统必须在目标节点上创建特权Pod以建立终端连接
- **FR-4.3**：系统必须支持完整的终端交互（Ctrl+C、Tab补全、颜色输出）
- **FR-4.4**：系统必须在会话结束后自动清理临时Pod
- **FR-4.5**：系统必须限制节点终端访问权限为管理员角色
- **FR-4.6**：Pod终端必须支持多容器Pod的容器选择
- **FR-4.7**：系统必须在30分钟无活动后自动断开终端会话

#### FR-5: 全局搜索

- **FR-5.1**：用户必须能够在所有命名空间和资源类型中搜索
- **FR-5.2**：系统必须支持模糊匹配（资源名称、标签、注解）
- **FR-5.3**：系统必须在1秒内返回搜索结果（对于1000+资源的集群）
- **FR-5.4**：系统必须按资源类型分组显示搜索结果
- **FR-5.5**：系统必须限制搜索结果数量（默认50条）
- **FR-5.6**：系统必须实现搜索结果缓存（5分钟有效期）
- **FR-5.7**：系统必须支持搜索超时控制（10秒超时）

#### FR-6: 集群上下文和权限

- **FR-6.1**：系统必须支持只读模式，阻止所有POST、PUT、PATCH、DELETE请求
- **FR-6.2**：系统必须在只读模式下返回清晰的错误信息（"只读模式已启用"）
- **FR-6.3**：系统必须支持从HTTP Header或查询参数读取集群名称
- **FR-6.4**：系统必须为每个集群创建并缓存独立的K8s Client实例
- **FR-6.5**：系统必须记录所有资源修改操作到审计日志（包含集群名称）
- **FR-6.6**：系统必须记录所有节点终端访问到审计日志（包含集群名称）
- **FR-6.7**：系统必须在切换集群时验证用户对目标集群的访问权限

### 关键实体 *（如果功能涉及数据则包含）*

- **Cluster**：K8s集群配置（Tiga已有）
  - **唯一标识**：数据库表自增ID（主键）
  - 包含属性：集群名称、Kubeconfig内容、是否默认、启用状态
  - 扩展需求：添加健康状态字段、最后连接时间、节点数统计、Pod数统计

- **配置扩展**：现有Config结构的扩展
  - KubernetesConfig：节点终端镜像、Kruise/Tailscale/Traefik/K3s Upgrade启用标志
  - PrometheusConfig：默认URL、**集群特定URL映射（key: cluster name, value: Prometheus URL）**、自动发现开关、超时配置
  - FeaturesConfig：只读模式开关

---

## 非功能需求

### 性能需求

- 资源列表查询响应时间必须<500ms（缓存命中<100ms）
- 全局搜索响应时间必须<1s（对于1000+资源）
- Prometheus异步发现任务必须在30秒内完成（成功或失败）
- Prometheus查询响应时间必须<2s
- WebSocket终端延迟必须<100ms
- 缓存命中率必须>70%

### 可扩展性需求

- 系统支持的集群数量无硬性限制，仅受以下因素约束：
  - 数据库性能（集群元数据存储）
  - 服务器内存（K8s Client实例缓存、工作负载缓存）
  - 服务器CPU（并发API请求处理）
- 建议配置参考：
  - 小规模（<10集群）：2核4GB内存
  - 中等规模（10-50集群）：4核8GB内存
  - 大规模（50+集群）：8核16GB内存，考虑水平扩展

### 可靠性需求

- 系统可用性必须达到99.9%（排除K8s集群本身故障）
- API调用错误率必须<0.1%
- 资源变更后缓存失效时间必须<5秒

### 安全需求

- 节点终端访问必须限制为管理员权限
- 所有资源修改操作必须记录审计日志（包含集群名称、用户、时间戳）
- Secret数据默认必须隐藏（需要点击"显示"）
- 终端会话必须在30分钟无活动后自动断开
- 集群切换必须验证用户对目标集群的访问权限

### 兼容性需求

- 系统必须支持Kubernetes 1.24+
- 系统必须支持OpenKruise 1.0+（可选依赖）
- 系统必须支持Traefik 2.10+（可选依赖）
- 系统必须支持Tailscale Operator 1.52+（可选依赖）
- 系统必须支持K3s System Upgrade Controller v0.13+（可选依赖）
- 系统必须支持Chrome 90+、Firefox 88+、Safari 14+、Edge 90+
- 系统必须在SQLite、PostgreSQL、MySQL上正常运行

---

## 实施计划

### 阶段划分

**Phase 0: 多集群管理基础（3天）**
- 增强Cluster模型（健康状态、统计信息）
- 实现集群列表API和Handler
- 实现集群切换中间件
- 实现集群健康检查机制

**Phase 1: Prometheus增强（5天）**
- 实现Prometheus后台异步发现服务（集群添加时触发，一次性尝试，30秒超时）
- 实现Prometheus服务检测逻辑（Service识别、端点优先级选择、连通性测试）
- 增强Prometheus客户端支持**集群级别配置**（手动配置优先于自动发现）
- 添加Prometheus API Handler和监控页面状态提示（检测中、已发现、未发现、手动配置）
- 实现"重新检测"功能（用户手动触发新的发现任务）

**Phase 2: 高级资源和CRD支持（7天）**
- 实现通用CRD处理器框架（CRD存在性检查、unstructured.Unstructured封装）
- 添加OpenKruise依赖和Handler（CloneSet、Advanced DaemonSet、Advanced StatefulSet）
- 实现Tailscale资源Handler（Connector、ProxyClass、ProxyGroup - 集群级别）
- 实现Traefik资源Handler（IngressRoute、Middleware等9个CRD - 命名空间级别）
- 实现K3s System Upgrade Controller Handler（Plan CRD）
- 实现CRD自动检测和菜单动态显示逻辑

**Phase 3: 资源增强和搜索（6天）**
- 实现通用资源处理器
- 实现资源关系服务
- 实现工作负载缓存服务（集群级别缓存）
- 实现全局搜索服务

**Phase 4: 终端和只读模式（4天）**
- 实现节点终端Handler
- 实现只读模式中间件
- 完善审计日志（包含集群上下文）

### 总工作量

- 25个工作日（Phase 0: 3天 + Phase 1: 5天 + Phase 2: 7天 + Phase 3: 6天 + Phase 4: 4天）
- 5个可独立交付的阶段
- 预留8天缓冲时间

---

## 审查与验收清单

### 内容质量
- [x] 没有实现细节（语言、框架、API）
- [x] 专注于用户价值和业务需求
- [x] 为非技术利益相关者编写
- [x] 所有必填章节已完成

### 需求完整性
- [x] 没有剩余的 [需要澄清] 标记
- [x] 需求可测试且明确
- [x] 成功标准可衡量
- [x] 范围明确界定
- [x] 已识别依赖关系和假设

---

## 执行状态

- [x] 用户描述已解析
- [x] 关键概念已提取
- [x] 歧义已标记
- [x] 用户场景已定义
- [x] 需求已生成
- [x] 实体已识别
- [x] 审查清单已通过

---

## 附录

### 参考项目

- **Kite**: https://github.com/ysicing/kite
  - 功能来源项目
  - 已分析20个资源处理器：
    - 基础资源处理器（17个）
    - 高级CRD处理器：Tailscale（3个CRD）、Traefik（9个CRD）、K3s Upgrade Controller（1个CRD）
  - 核心功能：Prometheus自动发现、节点终端、资源关系可视化

- **Tiga现有K8s功能**:
  - 基础资源管理：`pkg/handlers/resources/`
  - K8s客户端：`pkg/kube/`
  - 集群管理：`internal/services/k8s_service.go`

### 配置示例

```yaml
kubernetes:
  node_terminal_image: gcr.io/google-containers/ubuntu:latest
  enable_kruise: true
  enable_tailscale: false
  enable_traefik: true
  enable_k3s_upgrade: false  # K3s系统升级控制器

prometheus:
  # 自动发现机制：添加集群后异步后台检测，30秒超时
  auto_discovery: true
  discovery_timeout: 30

  # 集群特定的Prometheus配置（优先级最高，会停止对应集群的自动发现）
  cluster_urls:
    production: https://prometheus.prod.example.com  # 手动配置
    staging: http://prometheus-server.monitoring.svc.cluster.local:9090  # 手动配置
    # dev集群未配置，使用自动发现

features:
  readonly_mode: false
```

### 多集群配置说明

系统通过以下方式管理多个K8s集群：

1. **集群存储**：使用Tiga现有的`clusters`表（`internal/models/cluster.go`）
2. **自动导入**：启动时从`~/.kube/config`自动导入所有context
3. **集群切换**：通过HTTP Header `X-Cluster-Name` 或查询参数 `?cluster=xxx` 指定当前操作的集群
4. **Client缓存**：每个集群的K8s Client实例缓存在内存中，避免重复创建
5. **Prometheus配置**：
   - 添加集群后自动触发异步发现任务（30秒超时）
   - 支持集群级别手动配置（`prometheus.cluster_urls`），手动配置优先级最高
   - 未手动配置的集群使用自动发现机制
   - 自动发现成功后，Prometheus URL保存到集群配置中

### 常见问题

**Q1: 如何管理多个K8s集群？**
A: Tiga支持多集群管理。首次启动时自动从kubeconfig导入所有集群，也可以在UI中手动添加。用户先在集群列表页面选择集群，进入后所有操作都在该集群上下文中执行。通过顶部的集群切换器可以快速切换到其他集群。

**Q2: OpenKruise未安装会怎样？**
A: 系统会自动检测Kruise CRD，未检测到时隐藏相关菜单。

**Q2.5: 如何启用Traefik或Tailscale资源支持？**
A: 系统通过CRD自动检测机制工作。如果集群中已安装Traefik或Tailscale Operator，系统会自动检测到对应的CRD并在菜单中显示相关资源类型。也可以在配置文件中设置`kubernetes.enable_traefik: true`或`kubernetes.enable_tailscale: true`来显式启用。未安装对应CRD时，系统会在用户尝试访问时返回友好的错误提示。

**Q2.6: 集群级别CRD和命名空间级别CRD有什么区别？**
A: **集群级别CRD**（如Tailscale Connector、ProxyClass）是全局资源，不属于任何命名空间，列表时显示所有实例。**命名空间级别CRD**（如Traefik IngressRoute、Middleware）属于特定命名空间，可以按命名空间过滤，也支持跨命名空间查询（使用`_all`命名空间参数）。系统会根据CRD定义自动判断资源级别。

**Q3: 每个集群的Prometheus配置不同怎么办？**
A: 系统支持两种方式配置Prometheus：
1. **自动发现**（默认）：添加集群后，系统在后台异步运行Prometheus服务发现任务，自动检测并保存可用的Prometheus URL到集群配置。用户无需任何操作。
2. **手动配置**：可以在`prometheus.cluster_urls`中为每个集群单独配置Prometheus URL，或在集群详情页面手动输入。手动配置优先级高于自动发现，配置后系统会停止该集群的自动发现任务。

未配置的集群使用自动发现机制。如果30秒内未发现Prometheus服务，监控页面会提示手动配置。

**Q4: 节点终端是否有安全风险？**
A: 节点终端需要管理员权限，所有访问记录审计日志（包含集群名称），会话30分钟后自动断开。

**Q5: 只读模式如何启用？**
A: 设置`features.readonly_mode: true`，或为用户分配只读角色。

**Q6: 切换集群时会丢失当前的搜索结果吗？**
A: 是的。切换集群时，搜索历史、过滤条件等临时状态会被清除，因为不同集群的资源是独立的。

**Q7: 系统最多支持管理多少个集群？**
A: 无硬性限制。系统支持的集群数量仅受服务器资源约束（数据库性能、内存、CPU）。建议配置：小规模（<10集群）2核4GB内存，中等规模（10-50集群）4核8GB内存，大规模（50+集群）8核16GB内存并考虑水平扩展。

**Q8: 多个用户同时修改同一资源会发生什么？**
A: 系统依赖 Kubernetes API Server 的默认行为。当用户B尝试保存时，如果用户A已先保存，API Server 会返回 409 Conflict 错误（ResourceVersion 不匹配），系统提示"资源已被修改，请刷新后重试"。用户需要重新加载资源后再修改。

**Q9: Prometheus 自动发现失败后会重试吗？**
A: 不会自动重试。系统在添加集群后尝试一次自动发现（30秒超时），失败后不再自动重试。用户可以在集群详情页或监控页面点击"重新检测"按钮手动触发新的发现任务。

---

**文档版本**: v1.0
**最后更新**: 2025-10-17
**作者**: Claude Code
