# 审计子系统使用指南

本文档说明 Tiga 各个子系统在记录审计日志时应该使用的 `SubsystemType`。

## 概述

Tiga 使用统一的 `AuditEvent` 模型记录所有子系统的审计日志。每个子系统需要在创建审计事件时设置正确的 `Subsystem` 字段。

## 子系统类型（SubsystemType）

### 1. HTTP API 审计 (`http`)

**适用场景**：通用的 HTTP API 操作审计（未归属到特定子系统的 API 调用）

**使用位置**：
- `internal/api/middleware/audit.go` - HTTP 中间件自动记录

**示例操作**：
- 用户登录/登出
- 通用资源 CRUD 操作
- 配置修改

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemHTTP,
    Action: models.ActionCreated,
    ResourceType: models.ResourceTypeUser,
    // ...
}
```

---

### 2. MinIO 对象存储审计 (`minio`)

**适用场景**：MinIO 对象存储的所有操作

**使用位置**：
- `internal/services/minio/` - MinIO 服务层

**示例操作**：
- 上传/下载文件
- 创建/删除 Bucket
- 设置 Bucket 策略
- 授予/撤销权限
- 创建分享链接

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemMinIO,
    Action: models.ActionCreated,
    ResourceType: models.ResourceTypeMinIO,
    Resource: models.Resource{
        Identifier: "bucket-name/object-key",
        Data: map[string]string{
            "bucket": "my-bucket",
            "object_key": "files/document.pdf",
            "file_size": "1048576",
        },
    },
    // ...
}
```

---

### 3. 数据库管理审计 (`database`)

**适用场景**：数据库管理子系统的所有操作

**使用位置**：
- `internal/services/database/` - 数据库服务层
- `internal/api/handlers/database/` - 数据库 API 处理器

**示例操作**：
- 创建/删除数据库实例
- 创建/删除数据库
- 创建/删除用户
- 授予/撤销权限
- 执行 SQL 查询

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemDatabase,
    Action: models.ActionCreated,
    ResourceType: models.ResourceTypeDatabaseUser,
    Resource: models.Resource{
        Identifier: "user-uuid",
        Data: map[string]string{
            "username": "dbuser",
            "database": "production_db",
        },
    },
    // ...
}
```

---

### 4. 中间件管理审计 (`middleware`)

**适用场景**：Redis、MySQL、PostgreSQL 等中间件的管理操作

**使用位置**：
- `internal/services/managers/` - 中间件管理器

**示例操作**：
- 创建/删除 Redis 实例
- 修改中间件配置
- 健康检查
- 指标采集

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemMiddleware,
    Action: models.ActionUpdated,
    ResourceType: models.ResourceTypeRedis,
    // ...
}
```

---

### 5. Kubernetes 集群管理审计 (`kubernetes`)

**适用场景**：K8s 集群管理和资源操作

**使用位置**：
- `internal/api/handlers/cluster/` - K8s 集群处理器
- `pkg/handlers/` - K8s 资源处理器

**示例操作**：
- 添加/删除集群
- 部署/删除应用
- 修改 Deployment/Service
- 创建/删除 ConfigMap/Secret
- CRD 操作（OpenKruise、Traefik、Tailscale）

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemKubernetes,
    Action: models.ActionDeleted,
    ResourceType: models.ResourceTypePod,
    Resource: models.Resource{
        Identifier: "nginx-deployment-abc123",
        Data: map[string]string{
            "cluster_id": "cluster-uuid",
            "namespace": "default",
        },
    },
    // ...
}
```

---

### 6. Docker 实例管理审计 (`docker`)

**适用场景**：Docker 实例的管理操作

**使用位置**：
- `internal/services/managers/docker/` - Docker 管理器

**示例操作**：
- 创建/删除容器
- 启动/停止容器
- 修改容器配置
- 拉取/删除镜像

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemDocker,
    Action: models.ActionCreated,
    ResourceType: models.ResourceTypeInstance,
    Resource: models.Resource{
        Identifier: "container-id",
        Data: map[string]string{
            "image": "nginx:latest",
            "name": "web-server",
        },
    },
    // ...
}
```

---

### 7. 主机管理审计 (`host`)

**适用场景**：主机监控和管理操作（VMs 子系统）

**使用位置**：
- `internal/api/handlers/host_*.go` - 主机处理器
- `internal/services/host/` - 主机服务层

**示例操作**：
- 添加/删除主机节点
- 更新主机信息
- 重新生成密钥
- 创建/删除服务探测规则
- 创建/删除告警规则

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemHost,
    Action: models.ActionCreated,
    ResourceType: models.ResourceTypeInstance,
    Resource: models.Resource{
        Identifier: "host-uuid",
        Data: map[string]string{
            "hostname": "server-001",
            "ip_address": "192.168.1.100",
        },
    },
    // ...
}
```

---

### 8. WebSSH 终端审计 (`webssh`)

**适用场景**：SSH 终端会话的所有操作

**使用位置**：
- `internal/services/webssh/` - WebSSH 服务层
- `internal/api/handlers/webssh_handler.go` - WebSSH 处理器

**示例操作**：
- 创建 SSH 会话
- 关闭会话
- 访问会话录像
- 下载录像文件

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemWebSSH,
    Action: models.ActionCreated,
    ResourceType: models.ResourceTypeInstance,
    Resource: models.Resource{
        Identifier: "session-uuid",
        Data: map[string]string{
            "host_id": "host-uuid",
            "username": "root",
            "duration_seconds": "1800",
        },
    },
    // ...
}
```

---

### 9. 调度器审计 (`scheduler`)

**适用场景**：定时任务调度操作

**使用位置**：
- `internal/services/scheduler/` - 调度器服务层
- `internal/api/handlers/scheduler/` - 调度器 API 处理器

**示例操作**：
- 启用/禁用任务
- 手动触发任务
- 修改任务配置
- 查看执行历史

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemScheduler,
    Action: models.ActionEnabled,
    ResourceType: models.ResourceTypeScheduledTask,
    Resource: models.Resource{
        Identifier: "task-uuid",
        Data: map[string]string{
            "task_name": "database_audit_cleanup",
            "cron_expr": "0 2 * * *",
        },
    },
    // ...
}
```

---

### 10. 告警系统审计 (`alert`)

**适用场景**：告警规则和事件管理

**使用位置**：
- `internal/services/alert/` - 告警服务层
- `internal/api/handlers/alert_handler.go` - 告警处理器

**示例操作**：
- 创建/删除告警规则
- 启用/禁用规则
- 确认告警事件
- 解决告警事件

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemAlert,
    Action: models.ActionCreated,
    ResourceType: models.ResourceTypeInstance,
    Resource: models.Resource{
        Identifier: "rule-uuid",
        Data: map[string]string{
            "rule_name": "High CPU Usage",
            "threshold": "80",
        },
    },
    // ...
}
```

---

### 11. 认证和授权审计 (`auth`)

**适用场景**：用户认证和授权操作

**使用位置**：
- `internal/services/auth/` - 认证服务层
- `internal/api/handlers/auth_handler.go` - 认证处理器
- `pkg/auth/` - 认证工具

**示例操作**：
- 用户登录/登出
- Token 刷新
- 密码修改
- OAuth 授权
- RBAC 权限检查失败

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemAuth,
    Action: models.ActionLogin,
    ResourceType: models.ResourceTypeUser,
    Resource: models.Resource{
        Identifier: "user-uuid",
        Data: map[string]string{
            "login_method": "password",
            "success": "true",
        },
    },
    // ...
}
```

---

### 12. 存储管理审计 (`storage`)

**适用场景**：存储资源的管理操作

**使用位置**：
- 存储相关的服务和处理器

**示例操作**：
- 创建/删除存储卷
- 挂载/卸载存储
- 修改存储配置

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemStorage,
    Action: models.ActionCreated,
    ResourceType: models.ResourceTypeInstance,
    // ...
}
```

---

### 13. Web 服务器管理审计 (`webserver`)

**适用场景**：Web 服务器（Nginx、Apache 等）的管理操作

**使用位置**：
- Web 服务器管理相关的服务和处理器

**示例操作**：
- 创建/删除 Web 服务器实例
- 修改配置文件
- 重启/停止服务

```go
auditEvent := &models.AuditEvent{
    Subsystem: models.SubsystemWebServer,
    Action: models.ActionUpdated,
    ResourceType: models.ResourceTypeInstance,
    // ...
}
```

---

## 使用建议

### 1. 选择正确的 Subsystem

- **原则**：选择最能代表操作性质的子系统类型
- **示例**：在 K8s 集群中创建 Pod → 使用 `SubsystemKubernetes`，而非 `SubsystemHTTP`

### 2. 一致性

- 同一子系统的所有操作应使用相同的 `Subsystem` 值
- 便于后续按子系统过滤和统计审计日志

### 3. Metadata 字段使用

- 子系统特定的详细信息应存储在 `Resource.Data` 字段中（map[string]string）
- 避免在 `Action` 或 `ResourceType` 中包含子系统特定信息

### 4. 查询示例

```sql
-- 查询 K8s 子系统的所有操作
SELECT * FROM audit_events WHERE subsystem = 'kubernetes' ORDER BY timestamp DESC;

-- 查询主机管理的删除操作
SELECT * FROM audit_events WHERE subsystem = 'host' AND action = 'deleted';

-- 统计各子系统的操作数量
SELECT subsystem, COUNT(*) FROM audit_events GROUP BY subsystem;
```

---

## 迁移计划

### 阶段 1：核心子系统（已完成）
- ✅ HTTP API
- ✅ Scheduler

### 阶段 2：存储和数据库（T036-T037）
- [ ] MinIO（可选）
- [ ] Database（可选）

### 阶段 3：容器和编排
- [ ] Kubernetes
- [ ] Docker

### 阶段 4：主机管理
- [ ] Host
- [ ] WebSSH

### 阶段 5：其他子系统
- [ ] Alert
- [ ] Auth
- [ ] Middleware
- [ ] Storage
- [ ] WebServer

---

## 实现清单

| 子系统 | SubsystemType | 实现位置 | 状态 |
|--------|---------------|----------|------|
| HTTP API | `http` | `internal/api/middleware/audit.go` | ✅ 已实现 |
| MinIO | `minio` | `internal/services/minio/` | 🟡 待迁移 |
| Database | `database` | `internal/services/database/` | 🟡 待迁移 |
| Middleware | `middleware` | `internal/services/managers/` | ⏳ 待实现 |
| Kubernetes | `kubernetes` | `internal/api/handlers/cluster/` | ⏳ 待实现 |
| Docker | `docker` | `internal/services/managers/docker/` | ⏳ 待实现 |
| Host | `host` | `internal/services/host/` | ⏳ 待实现 |
| WebSSH | `webssh` | `internal/services/webssh/` | ⏳ 待实现 |
| Scheduler | `scheduler` | `internal/services/scheduler/` | ⏳ 待实现 |
| Alert | `alert` | `internal/services/alert/` | ⏳ 待实现 |
| Auth | `auth` | `internal/services/auth/` | ⏳ 待实现 |
| Storage | `storage` | - | ⏳ 待实现 |
| WebServer | `webserver` | - | ⏳ 待实现 |

---

## 相关文档

- 审计系统统一方案：`.claude/specs/006-gitness-tiga/audit-unification.md`
- 数据模型定义：`.claude/specs/006-gitness-tiga/data-model.md`
- API 契约：`.claude/specs/006-gitness-tiga/contracts/audit_api.yaml`
- 部署配置：`docs/deployment.md`

---

**最后更新**：2025-10-20
**维护者**：Tiga 开发团队
