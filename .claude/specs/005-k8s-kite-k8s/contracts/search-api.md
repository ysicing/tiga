# API 契约：全局搜索

**功能**：005-k8s-kite-k8s | **日期**：2025-10-17

## S1: 全局搜索

**端点**：`GET /api/v1/k8s/clusters/:cluster_id/search`

**描述**：在集群中搜索资源（跨命名空间、多资源类型）

**权限**：需要身份验证（JWT）

**路径参数**：
- `cluster_id`：集群 ID（uint）

**查询参数**：
- `q`：搜索关键词（必填，最小 1 字符）
- `types`：资源类型过滤（可选，逗号分隔，如 `Pod,Deployment,Service`）
- `namespace`：命名空间过滤（可选，默认搜索所有命名空间）
- `limit`：结果数量限制（可选，默认 50，最大 100）

**请求示例**：
```bash
curl -X GET "http://localhost:12306/api/v1/k8s/clusters/1/search?q=nginx&types=Pod,Deployment&limit=20" \
  -H "Authorization: Bearer <token>"
```

**响应**（200 OK）：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "results": [
      {
        "type": "Pod",
        "name": "nginx-deployment-7d64c5-abcde",
        "namespace": "default",
        "score": 100,
        "matched_fields": ["name"],
        "resource": {
          "apiVersion": "v1",
          "kind": "Pod",
          "metadata": {
            "name": "nginx-deployment-7d64c5-abcde",
            "namespace": "default",
            "labels": {
              "app": "nginx"
            }
          },
          "status": {
            "phase": "Running"
          }
        }
      },
      {
        "type": "Deployment",
        "name": "nginx-deployment",
        "namespace": "default",
        "score": 95,
        "matched_fields": ["name", "labels.app"],
        "resource": {
          "apiVersion": "apps/v1",
          "kind": "Deployment",
          "metadata": {
            "name": "nginx-deployment",
            "namespace": "default",
            "labels": {
              "app": "nginx"
            }
          },
          "spec": {
            "replicas": 3
          }
        }
      }
    ],
    "total": 2,
    "query": "nginx",
    "took_ms": 150
  }
}
```

**响应字段说明**：
- `results`：搜索结果数组
  - `type`：资源类型
  - `name`：资源名称
  - `namespace`：命名空间（集群级别资源为空）
  - `score`：相关性评分（0-100）
  - `matched_fields`：匹配的字段列表
  - `resource`：完整的资源对象
- `total`：结果总数
- `query`：搜索关键词
- `took_ms`：查询耗时（毫秒）

**搜索逻辑**：
1. 并发查询多个资源类型（Pod、Deployment、Service、ConfigMap、Secret）
2. 匹配字段：
   - 资源名称（精确匹配优先，模糊匹配次之）
   - 标签（`labels.*`）
   - 注解（`annotations.*`）
3. 评分算法：
   - 精确匹配名称：100 分
   - 名称包含关键词：80 分
   - 标签匹配：60 分
   - 注解匹配：40 分
4. 结果按评分降序排列
5. 限制返回前 N 条结果

**错误响应**：
- `400 Bad Request`：缺少 `q` 参数或参数格式错误
- `401 Unauthorized`：未提供有效的 JWT token
- `404 Not Found`：集群不存在
- `408 Request Timeout`：搜索超时（>10秒）
- `500 Internal Server Error`：搜索失败

---

**文档版本**：v1.0
**最后更新**：2025-10-17
**作者**：Claude Code
