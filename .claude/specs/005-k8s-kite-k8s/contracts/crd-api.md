# API 契约：CRD 资源管理

**功能**：005-k8s-kite-k8s | **日期**：2025-10-17

## 概述

CRD 资源管理 API 提供 OpenKruise、Tailscale、Traefik、K3s Upgrade Controller 等 CRD 资源的 CRUD 操作。

**通用模式**：所有 CRD 资源 API 遵循相同的 RESTful 模式

---

## 通用端点模式

### 列表（List）

**端点**：`GET /api/v1/k8s/clusters/:cluster_id/:crd_type`

**请求参数**：
- `namespace`：命名空间（可选，仅用于命名空间级别 CRD）
  - 省略或 `_all`：查询所有命名空间
  - 特定命名空间：仅查询该命名空间

**响应示例**（以 CloneSet 为例）：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "apiVersion": "apps.kruise.io/v1alpha1",
        "kind": "CloneSet",
        "metadata": {
          "name": "nginx-cloneset",
          "namespace": "default",
          "uid": "abc-123-def",
          "resourceVersion": "1000",
          "creationTimestamp": "2025-10-17T08:00:00Z"
        },
        "spec": {
          "replicas": 3,
          "selector": {
            "matchLabels": {
              "app": "nginx"
            }
          },
          "template": {
            "metadata": {
              "labels": {
                "app": "nginx"
              }
            },
            "spec": {
              "containers": [
                {
                  "name": "nginx",
                  "image": "nginx:1.21"
                }
              ]
            }
          }
        },
        "status": {
          "replicas": 3,
          "readyReplicas": 3,
          "updatedReplicas": 3
        }
      }
    ],
    "total": 1
  }
}
```

---

### 详情（Get）

**端点**：`GET /api/v1/k8s/clusters/:cluster_id/:crd_type/:name`

**请求参数**：
- `namespace`：命名空间（可选，仅用于命名空间级别 CRD，默认 `default`）

---

### 创建（Create）

**端点**：`POST /api/v1/k8s/clusters/:cluster_id/:crd_type`

**请求体**：完整的 CRD YAML/JSON（unstructured）

---

### 更新（Update）

**端点**：`PUT /api/v1/k8s/clusters/:cluster_id/:crd_type/:name`

**请求参数**：
- `namespace`：命名空间（可选，仅用于命名空间级别 CRD）

**请求体**：完整的 CRD YAML/JSON（包含 `resourceVersion`）

---

### 删除（Delete）

**端点**：`DELETE /api/v1/k8s/clusters/:cluster_id/:crd_type/:name`

**请求参数**：
- `namespace`：命名空间（可选，仅用于命名空间级别 CRD）

---

## 支持的 CRD 类型

### OpenKruise

| CRD 类型 | 端点路径 | 级别 | 特殊操作 |
|---------|---------|------|---------|
| CloneSet | `/clonesets` | 命名空间 | Scale, Restart |
| Advanced DaemonSet | `/advanced-daemonsets` | 命名空间 | - |
| Advanced StatefulSet | `/advanced-statefulsets` | 命名空间 | - |

**特殊操作示例**：

#### Scale CloneSet

**端点**：`PUT /api/v1/k8s/clusters/:cluster_id/clonesets/:name/scale`

**请求体**：
```json
{
  "replicas": 5
}
```

#### Restart CloneSet

**端点**：`POST /api/v1/k8s/clusters/:cluster_id/clonesets/:name/restart`

**描述**：触发 CloneSet 滚动重启（原地升级）

---

### Tailscale（集群级别）

| CRD 类型 | 端点路径 | 级别 |
|---------|---------|------|
| Connector | `/tailscale/connectors` | 集群 |
| ProxyClass | `/tailscale/proxyclasses` | 集群 |
| ProxyGroup | `/tailscale/proxygroups` | 集群 |

**注意**：Tailscale 资源是集群级别，不支持 `namespace` 参数

---

### Traefik（命名空间级别）

| CRD 类型 | 端点路径 | 级别 |
|---------|---------|------|
| IngressRoute | `/traefik/ingressroutes` | 命名空间 |
| IngressRouteTCP | `/traefik/ingressroutetcps` | 命名空间 |
| IngressRouteUDP | `/traefik/ingressrouteudps` | 命名空间 |
| Middleware | `/traefik/middlewares` | 命名空间 |
| MiddlewareTCP | `/traefik/middlewaretcps` | 命名空间 |
| TLSOption | `/traefik/tlsoptions` | 命名空间 |
| TLSStore | `/traefik/tlsstores` | 命名空间 |
| TraefikService | `/traefik/traefikservices` | 命名空间 |
| ServersTransport | `/traefik/serverstransports` | 命名空间 |

---

### K3s System Upgrade Controller（命名空间级别）

| CRD 类型 | 端点路径 | 级别 |
|---------|---------|------|
| Plan | `/k3s/plans` | 命名空间 |

---

## CRD 检测 API

**端点**：`GET /api/v1/k8s/clusters/:cluster_id/crds`

**描述**：检测集群中已安装的 CRD

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "kruise": {
      "installed": true,
      "crds": ["CloneSet", "Advanced DaemonSet", "Advanced StatefulSet"]
    },
    "tailscale": {
      "installed": false,
      "crds": []
    },
    "traefik": {
      "installed": true,
      "crds": ["IngressRoute", "Middleware", "TLSOption"]
    },
    "k3s_upgrade": {
      "installed": false,
      "crds": []
    }
  }
}
```

---

## 错误响应

### CRD 不存在

**状态码**：`404 Not Found`

**响应体**：
```json
{
  "code": 404,
  "message": "CustomResourceDefinition clonesets.apps.kruise.io not found, please install OpenKruise",
  "data": null
}
```

### 集群上下文错误

**状态码**：`400 Bad Request`

**响应体**：
```json
{
  "code": 400,
  "message": "集群上下文缺失，请提供 X-Cluster-ID header 或 cluster 查询参数",
  "data": null
}
```

### ResourceVersion 冲突

**状态码**：`409 Conflict`

**响应体**：
```json
{
  "code": 409,
  "message": "资源已被修改，请刷新后重试（ResourceVersion mismatch）",
  "data": {
    "current_version": "1001",
    "provided_version": "1000"
  }
}
```

---

**文档版本**：v1.0
**最后更新**：2025-10-17
**作者**：Claude Code
