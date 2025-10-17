# 快速启动：K8s子系统

**功能**：005-k8s-kite-k8s | **日期**：2025-10-17

## 概述

本文档提供 K8s 子系统功能的快速验证指南，包括环境准备、功能测试和验收场景执行。

---

## 前置条件

### 环境要求

- **Go**：1.24+
- **Node.js**：18+
- **Docker**：用于运行测试 K8s 集群（可选）
- **kubectl**：用于手动验证
- **K8s 集群**：至少一个可用的 K8s 集群（1.24+）

### 可选依赖

- **OpenKruise**：安装 OpenKruise Operator（测试 CloneSet 功能）
- **Traefik**：安装 Traefik Ingress Controller（测试 IngressRoute 功能）
- **Prometheus**：安装 Prometheus Operator（测试监控功能）

### 准备测试集群

**选项 1：使用 Kind（本地测试）**
```bash
# 创建测试集群
kind create cluster --name tiga-test

# 安装 OpenKruise
kubectl apply -f https://raw.githubusercontent.com/openkruise/kruise/master/config/crd/bases/apps.kruise.io_clonesets.yaml
```

**选项 2：使用现有集群**
```bash
# 确保 kubeconfig 配置正确
kubectl cluster-info

# 验证集群版本
kubectl version --short
```

---

## 启动应用

### 1. 启动后端

```bash
# 进入项目根目录
cd /Users/ysicing/go/src/github.com/ysicing/tiga

# 启动开发模式（会先构建前端）
task dev

# 或仅启动后端
task dev:backend
```

**验证后端启动**：
```bash
# 检查健康端点
curl http://localhost:12306/api/v1/health

# 预期输出
{"code":200,"message":"success","data":{"status":"ok"}}
```

### 2. 启动前端（可选，用于 UI 测试）

```bash
# 另开终端，启动前端开发服务器
cd ui
pnpm dev
```

**访问前端**：打开浏览器访问 `http://localhost:5173`

---

## 功能验证

### V1: 集群管理

#### 1.1 导入集群

**自动导入**：
- 应用启动时自动从 `~/.kube/config` 导入集群
- 检查日志输出：`Imported X clusters from kubeconfig`

**手动验证**：
```bash
# 获取集群列表
curl -X GET http://localhost:12306/api/v1/k8s/clusters \
  -H "Authorization: Bearer <token>"

# 预期：返回至少 1 个集群
```

#### 1.2 集群健康检查

**等待 60 秒**（首次健康检查执行）

**验证**：
```bash
# 再次获取集群列表
curl -X GET http://localhost:12306/api/v1/k8s/clusters \
  -H "Authorization: Bearer <token>"

# 预期：health_status 从 "unknown" 变为 "healthy"
#       node_count 和 pod_count 有数值
```

#### 1.3 测试集群连接

```bash
curl -X POST http://localhost:12306/api/v1/k8s/clusters/1/test-connection \
  -H "Authorization: Bearer <token>"

# 预期：{"code":200,"message":"集群连接成功",...}
```

---

### V2: Prometheus 自动发现

#### 2.1 前提条件

**安装 Prometheus**（如果集群中没有）：
```bash
# 使用 kube-prometheus-stack
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace
```

#### 2.2 添加新集群触发自动发现

```bash
# 添加新集群（使用不同的 kubeconfig）
curl -X POST http://localhost:12306/api/v1/k8s/clusters \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-cluster",
    "kubeconfig": "YXBpVmVyc2lvbjog..."
  }'

# 预期：返回 201，message 包含 "Prometheus 自动发现任务已启动"
```

#### 2.3 验证自动发现结果

**等待 10-30 秒**

```bash
# 获取集群详情
curl -X GET http://localhost:12306/api/v1/k8s/clusters/2 \
  -H "Authorization: Bearer <token>"

# 预期：prometheus_url 字段有值
#       如："http://prometheus-server.monitoring.svc.cluster.local:9090"
```

#### 2.4 手动重新检测

```bash
curl -X POST http://localhost:12306/api/v1/k8s/clusters/2/prometheus/rediscover \
  -H "Authorization: Bearer <token>"

# 预期：返回 202，message 包含 "任务已启动"
```

---

### V3: OpenKruise CRD 支持

#### 3.1 前提条件

**安装 OpenKruise**：
```bash
kubectl apply -f https://raw.githubusercontent.com/openkruise/kruise/master/config/crd/bases/apps.kruise.io_clonesets.yaml
```

#### 3.2 检测 CRD

```bash
curl -X GET http://localhost:12306/api/v1/k8s/clusters/1/crds \
  -H "Authorization: Bearer <token>"

# 预期：kruise.installed = true
```

#### 3.3 创建 CloneSet

```bash
# 创建测试 CloneSet
kubectl apply -f - <<EOF
apiVersion: apps.kruise.io/v1alpha1
kind: CloneSet
metadata:
  name: nginx-cloneset
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.21
EOF
```

#### 3.4 列出 CloneSet

```bash
curl -X GET http://localhost:12306/api/v1/k8s/clusters/1/clonesets?namespace=default \
  -H "Authorization: Bearer <token>"

# 预期：返回 nginx-cloneset
```

#### 3.5 扩容 CloneSet

```bash
curl -X PUT http://localhost:12306/api/v1/k8s/clusters/1/clonesets/nginx-cloneset/scale \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"replicas": 5}'

# 预期：返回 200
```

**验证**：
```bash
kubectl get cloneset nginx-cloneset -o jsonpath='{.spec.replicas}'
# 预期：5
```

---

### V4: 全局搜索

#### 4.1 搜索资源

```bash
curl -X GET "http://localhost:12306/api/v1/k8s/clusters/1/search?q=nginx" \
  -H "Authorization: Bearer <token>"

# 预期：返回包含 "nginx" 的 Pod、Deployment、Service 等资源
#       took_ms < 1000（1秒响应）
```

#### 4.2 过滤资源类型

```bash
curl -X GET "http://localhost:12306/api/v1/k8s/clusters/1/search?q=nginx&types=Pod,Deployment" \
  -H "Authorization: Bearer <token>"

# 预期：仅返回 Pod 和 Deployment 类型的结果
```

---

### V5: 节点终端（需要管理员权限）

#### 5.1 获取节点列表

```bash
kubectl get nodes
# 记录节点名称，如 "node-01"
```

#### 5.2 打开节点终端

**通过 WebSocket**（需要前端或 WebSocket 客户端）：
```
ws://localhost:12306/api/v1/k8s/clusters/1/nodes/node-01/terminal
```

**验证**：
- 能够执行 `ls /`、`ip addr` 等命令
- 30 分钟无活动后自动断开

---

## 验收场景执行

### 场景 1：CloneSet 扩缩容

**参考规格**：spec.md - 场景1

**步骤**：
1. ✅ 已在 V3.3 创建 3 副本的 CloneSet
2. ✅ 已在 V3.5 扩容到 5 副本
3. **验证**：30 秒内显示 5 个运行中的 Pods
   ```bash
   kubectl get pods -l app=nginx
   # 预期：5 个 Running 状态的 Pods
   ```

### 场景 2：Prometheus 异步自动发现

**参考规格**：spec.md - 场景2

**步骤**：
1. ✅ 已在 V2.2 添加新集群
2. ✅ 已在 V2.3 验证自动发现结果
3. **验证**：10 秒内检测到 Prometheus，保存 URL

### 场景 5：全局搜索

**参考规格**：spec.md - 场景5

**步骤**：
1. **假定**：集群中有 50+ 命名空间，1000+ 资源
2. **执行**：V4.1 搜索 "redis"
3. **验证**：1 秒内返回结果（检查 `took_ms` 字段）

---

## 清理

### 删除测试资源

```bash
# 删除 CloneSet
kubectl delete cloneset nginx-cloneset

# 删除 Prometheus（如果安装了）
helm uninstall prometheus -n monitoring

# 删除 Kind 集群（如果使用 Kind）
kind delete cluster --name tiga-test
```

### 删除测试集群

```bash
curl -X DELETE http://localhost:12306/api/v1/k8s/clusters/2 \
  -H "Authorization: Bearer <token>"
```

---

## 故障排查

### 问题 1：集群连接失败

**症状**：`health_status` 显示 "unavailable"

**排查**：
1. 检查 kubeconfig 是否有效：`kubectl cluster-info`
2. 检查网络连接：`curl -k <cluster-api-server>`
3. 检查日志：`task dev` 输出的错误信息

### 问题 2：Prometheus 自动发现失败

**症状**：30 秒后 `prometheus_url` 仍为空

**排查**：
1. 检查 Prometheus 是否安装：`kubectl get svc -n monitoring`
2. 检查服务端口：`kubectl get svc prometheus-server -n monitoring -o yaml`
3. 手动配置 Prometheus URL：
   ```bash
   curl -X PUT http://localhost:12306/api/v1/k8s/clusters/1 \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"prometheus_url": "http://prometheus-server.monitoring.svc.cluster.local:9090"}'
   ```

### 问题 3：CRD 不存在错误

**症状**：访问 CloneSet API 返回 404，message 包含 "CustomResourceDefinition not found"

**排查**：
1. 检查 OpenKruise 是否安装：`kubectl get crd clonesets.apps.kruise.io`
2. 安装 OpenKruise：参考 V3.1

---

## 成功标准

- ✅ 所有验证步骤通过
- ✅ 至少执行 3 个验收场景
- ✅ 无阻塞性错误
- ✅ 性能目标达成（搜索 <1s，API <500ms）

---

**文档版本**：v1.0
**最后更新**：2025-10-17
**作者**：Claude Code
