# 技术研究：K8s子系统实施

**日期**：2025-10-17 | **功能**：005-k8s-kite-k8s

## 研究摘要

本文档记录 K8s 子系统实施前的技术研究成果，包括技术选型依据、最佳实践、替代方案评估和潜在风险。所有技术上下文中的"需要澄清"项已通过研究解决。

---

## R1：多集群管理架构

### 决策：集群级别 Client 实例缓存 + 数据库表自增 ID 标识

**选择理由**：
- **性能**：为每个集群维护独立的 client-go 实例，避免重复创建（创建 ClientSet 需要解析 kubeconfig + 建立连接，约 100-500ms）
- **唯一标识**：使用数据库表自增 ID 作为主键，简单可靠，避免集群名称冲突（不同 context 可能有相同名称）
- **上下文隔离**：通过 HTTP Header `X-Cluster-Name` 或查询参数 `?cluster=xxx` 传递集群上下文，中间件根据 ID 检索 Client 实例

**考虑的替代方案**：
- ❌ **每次请求创建 Client**：性能差（每个请求 100-500ms 开销）
- ❌ **使用集群名称作为主键**：kubeconfig 中的 context 名称可能重复或变更
- ❌ **使用 Kubeconfig Hash**：不可读，调试困难

**参考实现**：
- Kite 项目：`pkg/kube/client.go` - 使用全局 Client 实例（单集群）
- Tiga 现有：`pkg/kube/client.go` - 已有 Client 创建逻辑，需扩展为缓存 map

**实施建议**：
```go
// pkg/kube/client.go
type ClientCache struct {
    sync.RWMutex
    clients map[uint]*K8sClient  // key: cluster.ID
}

func (c *ClientCache) GetOrCreate(cluster *models.Cluster) (*K8sClient, error) {
    c.RLock()
    if client, ok := c.clients[cluster.ID]; ok {
        c.RUnlock()
        return client, nil
    }
    c.RUnlock()

    c.Lock()
    defer c.Unlock()
    // Double-check after acquiring write lock
    if client, ok := c.clients[cluster.ID]; ok {
        return client, nil
    }

    client, err := NewK8sClientFromKubeconfig(cluster.Kubeconfig)
    if err != nil {
        return nil, err
    }
    c.clients[cluster.ID] = client
    return client, nil
}
```

**风险与缓解**：
- **风险**：内存泄漏（删除集群后 Client 未清理）
- **缓解**：实现 `RemoveClient(clusterID uint)` 方法，在删除集群时调用

---

## R2：OpenKruise CRD 处理

### 决策：通用 CRD 处理器模式 + unstructured.Unstructured

**选择理由**：
- **灵活性**：OpenKruise 有 10+ CRD，通用模式避免为每个 CRD 写重复代码
- **CRD 检测**：在操作前检查 CRD 是否存在（`GET /apis/apiextensions.k8s.io/v1/customresourcedefinitions/{name}`），避免 404 错误
- **动态处理**：使用 `unstructured.Unstructured` 处理不同版本的 CRD（避免强类型依赖）

**考虑的替代方案**：
- ❌ **为每个 CRD 创建强类型结构体**：维护成本高，版本兼容性差
- ❌ **直接使用 dynamic client**：没有 CRD 检测，用户体验差（返回晦涩的 404 错误）

**参考实现**：
- Kite 项目：`pkg/handlers/resources/tailscale_handler.go`（集群级别 CRD）
- Kite 项目：`pkg/handlers/resources/traefik_handler.go`（命名空间级别 CRD）

**实施模式**：
```go
// pkg/kube/crd.go
type CRDHandler struct {
    ResourceName string
    CRDName      string
    Kind         string
    Group        string
    Version      string
    Namespaced   bool
}

func (h *CRDHandler) CheckCRDExists(ctx context.Context, client *K8sClient) error {
    crd := &apiextensionsv1.CustomResourceDefinition{}
    err := client.Get(ctx, types.NamespacedName{Name: h.CRDName}, crd)
    if err != nil {
        if errors.IsNotFound(err) {
            return fmt.Errorf("CustomResourceDefinition %s not found, please install %s", h.CRDName, h.Kind)
        }
        return err
    }
    return nil
}

func (h *CRDHandler) List(ctx context.Context, client *K8sClient, namespace string) (*unstructured.UnstructuredList, error) {
    if err := h.CheckCRDExists(ctx, client); err != nil {
        return nil, err
    }

    gvr := schema.GroupVersionResource{
        Group:    h.Group,
        Version:  h.Version,
        Resource: h.ResourceName,
    }

    opts := []client.ListOption{}
    if h.Namespaced && namespace != "" && namespace != "_all" {
        opts = append(opts, client.InNamespace(namespace))
    }

    list := &unstructured.UnstructuredList{}
    list.SetGroupVersionKind(schema.GroupVersionKind{
        Group:   h.Group,
        Version: h.Version,
        Kind:    h.Kind + "List",
    })

    err := client.List(ctx, list, opts...)
    return list, err
}
```

**OpenKruise 核心 CRD**（需实现）：
1. **CloneSet**：`apps.kruise.io/v1alpha1` - 增强版 Deployment（原地升级、灰度发布）
2. **Advanced DaemonSet**：`apps.kruise.io/v1alpha1` - 增强版 DaemonSet（分批升级）
3. **Advanced StatefulSet**：`apps.kruise.io/v1beta1` - 增强版 StatefulSet（原地升级）

**风险与缓解**：
- **风险**：CRD 版本变更导致不兼容
- **缓解**：在 CheckCRDExists 时读取 CRD 的 `spec.versions` 字段，匹配支持的版本

---

## R3：Traefik CRD 处理

### 决策：命名空间级别 CRD 处理器 + 跨命名空间查询支持

**选择理由**：
- **命名空间隔离**：Traefik CRD 是命名空间级别资源，不同命名空间的 IngressRoute 相互独立
- **跨命名空间查询**：支持 `namespace=_all` 参数，一次性查询所有命名空间的 Traefik 资源
- **菜单动态显示**：检测到 Traefik CRD 时自动显示菜单项，未安装时隐藏

**Traefik 核心 CRD**（9 个）：
1. **IngressRoute**：HTTP 路由配置
2. **IngressRouteTCP**：TCP 路由配置
3. **IngressRouteUDP**：UDP 路由配置
4. **Middleware**：HTTP 中间件（限流、认证、重试等）
5. **MiddlewareTCP**：TCP 中间件
6. **TLSOption**：TLS 配置选项
7. **TLSStore**：TLS 证书存储
8. **TraefikService**：负载均衡策略
9. **ServersTransport**：后端服务器传输配置

**参考实现**：
- Kite 项目：`pkg/handlers/resources/traefik_handler.go`

**实施建议**：
- 复用 R2 中的通用 CRD 处理器模式
- 为每个 Traefik CRD 创建一个 Handler 实例（设置 `Namespaced=true`）

---

## R4：Tailscale CRD 处理

### 决策：集群级别 CRD 处理器（无命名空间过滤）

**选择理由**：
- **集群级别资源**：Tailscale CRD（Connector、ProxyClass、ProxyGroup）是全局资源，不属于任何命名空间
- **无命名空间参数**：List 操作忽略 `namespace` 参数，查询所有实例

**Tailscale 核心 CRD**（3 个）：
1. **Connector**：连接 K8s 集群到 Tailscale 网络
2. **ProxyClass**：代理配置类（定义代理行为）
3. **ProxyGroup**：代理组（管理多个代理实例）

**参考实现**：
- Kite 项目：`pkg/handlers/resources/tailscale_handler.go`

**实施建议**：
- 复用 R2 中的通用 CRD 处理器模式
- 设置 `Namespaced=false`，在 List 时不添加 `client.InNamespace()` 选项

---

## R5：K3s System Upgrade Controller 集成

### 决策：支持 Plan CRD，提供集群升级编排能力

**选择理由**：
- **轻量级升级管理**：K3s System Upgrade Controller 使用 Plan CRD 定义升级策略（升级版本、并发数、节点选择器）
- **适用场景**：K3s 集群（边缘计算、IoT）的零停机升级

**Plan CRD**：
- **Group**：`upgrade.cattle.io`
- **Version**：`v1`
- **Kind**：`Plan`
- **Scope**：命名空间级别

**参考实现**：
- Kite 项目：`pkg/handlers/resources/system_upgrade_handler.go`

**实施建议**：
- 复用 R2 中的通用 CRD 处理器模式
- 提供基本的 CRUD 操作（不涉及升级逻辑，仅管理 Plan 资源）

---

## R6：Prometheus 异步自动发现

### 决策：后台 Goroutine + 一次性尝试 + 手动重新检测

**选择理由**：
- **异步非阻塞**：添加集群后立即返回，不阻塞用户操作
- **一次性尝试**：30 秒超时内尝试发现 Prometheus，失败后不自动重试（避免资源浪费）
- **手动触发**：用户可在集群详情页或监控页面点击"重新检测"按钮重新尝试

**发现策略**（优先级从高到低）：
1. **检查手动配置**：如果 `prometheus.cluster_urls[cluster.Name]` 存在，直接使用（跳过自动发现）
2. **搜索 Service**：在 `monitoring`、`prometheus`、`kube-system` 等命名空间搜索以下 Service：
   - 名称匹配：`prometheus-server`、`prometheus-operated`、`prometheus-k8s`
   - 标签匹配：`app=prometheus`、`operated-prometheus=true`
   - 端口匹配：9090
3. **测试连通性**：对每个候选端点执行 `GET /api/v1/status/config` 请求（2 秒超时）
4. **选择最佳端点**（优先级）：
   - LoadBalancer > Ingress > NodePort > ClusterIP
5. **保存到数据库**：将可用的 Prometheus URL 保存到 `cluster.PrometheusURL` 字段

**参考实现**：
- Kite 项目：`pkg/prometheus/discovery.go`（同步发现）
- 需修改为异步模式

**实施建议**：
```go
// internal/services/prometheus/discovery.go
type DiscoveryService struct {
    clusterRepo repository.ClusterRepositoryInterface
}

func (s *DiscoveryService) StartDiscoveryTask(ctx context.Context, cluster *models.Cluster) {
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        // Skip if manual config exists
        if cluster.PrometheusURL != "" {
            return
        }

        prometheusURL, err := s.discoverPrometheus(ctx, cluster)
        if err != nil {
            logrus.Warnf("Prometheus auto-discovery failed for cluster %s: %v", cluster.Name, err)
            return
        }

        // Save to database
        cluster.PrometheusURL = prometheusURL
        if err := s.clusterRepo.Update(ctx, cluster); err != nil {
            logrus.Errorf("Failed to save Prometheus URL for cluster %s: %v", cluster.Name, err)
        }
    }()
}
```

**风险与缓解**：
- **风险**：Goroutine 泄漏
- **缓解**：使用 `context.WithTimeout` 保证 30 秒后自动退出

---

## R7：资源关系可视化

### 决策：静态关系映射 + 递归查询（限制深度）

**选择理由**：
- **静态关系**：K8s 资源间的关系是确定的（如 Deployment → ReplicaSet → Pod）
- **递归查询**：从起点资源递归查询关联资源，限制最大深度为 3 层（避免循环引用）
- **缓存优化**：工作负载资源（Pod、ReplicaSet、Deployment）缓存 5 分钟，减少 API 调用

**关系映射示例**：
```go
// internal/services/k8s/relations.go
var resourceRelations = map[string][]string{
    "Deployment": {"ReplicaSet", "Pod", "HorizontalPodAutoscaler", "Service"},
    "ReplicaSet": {"Pod"},
    "Service":    {"Pod", "Endpoints"},
    "Pod":        {"PersistentVolumeClaim", "ConfigMap", "Secret"},
}
```

**实施建议**：
- 使用 client-go 的 `ownerReferences` 字段追踪父子关系
- 使用 label selector 查询关联资源（如 Service 选择 Pod）

**风险与缓解**：
- **风险**：循环引用导致无限递归
- **缓解**：记录已访问的资源 UID，检测到重复时停止查询

---

## R8：全局搜索性能优化

### 决策：并发查询 + 结果限制 + 缓存

**选择理由**：
- **并发查询**：同时查询多个资源类型（Pods、Deployments、Services、ConfigMaps），使用 Goroutine + WaitGroup
- **结果限制**：默认返回前 50 条结果，避免大量数据传输
- **缓存**：搜索结果缓存 5 分钟（key: cluster_id + search_term）

**实施建议**：
```go
// internal/services/k8s/search.go
func (s *SearchService) Search(ctx context.Context, clusterID uint, term string) ([]SearchResult, error) {
    // Check cache
    cacheKey := fmt.Sprintf("search:%d:%s", clusterID, term)
    if cached, ok := s.cache.Get(cacheKey); ok {
        return cached.([]SearchResult), nil
    }

    var wg sync.WaitGroup
    resultsChan := make(chan []SearchResult, 4)

    resourceTypes := []string{"Pod", "Deployment", "Service", "ConfigMap"}
    for _, resType := range resourceTypes {
        wg.Add(1)
        go func(rt string) {
            defer wg.Done()
            results := s.searchResourceType(ctx, clusterID, rt, term)
            resultsChan <- results
        }(resType)
    }

    go func() {
        wg.Wait()
        close(resultsChan)
    }()

    var allResults []SearchResult
    for results := range resultsChan {
        allResults = append(allResults, results...)
        if len(allResults) >= 50 {
            break
        }
    }

    // Sort by relevance and limit
    sort.Slice(allResults, func(i, j int) bool {
        return allResults[i].Score > allResults[j].Score
    })
    if len(allResults) > 50 {
        allResults = allResults[:50]
    }

    // Cache results
    s.cache.Set(cacheKey, allResults, 5*time.Minute)

    return allResults, nil
}
```

**性能目标**：
- 并发查询 4 个资源类型：约 200-400ms
- 缓存命中：<100ms
- 符合规格要求的 1 秒响应

---

## R9：WebSocket 终端实现

### 决策：复用 Tiga 现有的终端实现 + 扩展节点终端

**选择理由**：
- **Pod 终端**：Tiga 已有实现（`pkg/kube/terminal.go`），复用即可
- **节点终端**：需创建特权 Pod 在目标节点上，建立 WebSocket 连接

**节点终端实现模式**：
1. 创建特权 Pod：
   - `hostNetwork: true`
   - `hostPID: true`
   - `privileged: true`
   - `nodeName: <target-node>`
2. 通过 Pod Exec 建立 Shell 连接
3. 30 分钟无活动后自动断开并清理 Pod

**参考实现**：
- Kite 项目：`pkg/kube/terminal.go`

---

## R10：依赖版本管理

### 决策：使用 Go Modules + 固定次版本号

**OpenKruise SDK**：
```go
// go.mod
require (
    github.com/openkruise/kruise-api v1.8.0  // 固定 1.8.x
)
```

**client-go 版本兼容性**：
- client-go v0.31.4（对应 K8s 1.31）
- 向下兼容 K8s 1.24+

**风险与缓解**：
- **风险**：OpenKruise API 版本不兼容
- **缓解**：在 CRD 检测时读取 `spec.versions` 字段，匹配支持的版本

---

## 研究结论

### 已解决的未知项
- ✅ 多集群管理架构（Client 缓存 + 数据库 ID）
- ✅ OpenKruise CRD 处理（通用模式 + unstructured）
- ✅ Traefik CRD 处理（命名空间级别）
- ✅ Tailscale CRD 处理（集群级别）
- ✅ K3s Upgrade Controller 集成（Plan CRD）
- ✅ Prometheus 异步发现（Goroutine + 一次性尝试）
- ✅ 资源关系可视化（静态映射 + 递归查询）
- ✅ 全局搜索优化（并发查询 + 缓存）
- ✅ WebSocket 终端（复用现有 + 扩展节点终端）
- ✅ 依赖版本管理（Go Modules + 固定次版本）

### 无需进一步澄清的项
所有技术上下文中的"需要澄清"项已通过研究解决，可直接进入阶段 1 设计。

---

**文档版本**：v1.0
**最后更新**：2025-10-17
**作者**：Claude Code
