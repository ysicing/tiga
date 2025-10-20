# Tiga Dashboard 部署指南

本文档提供 Tiga Dashboard 的部署配置、性能调优和故障排查指南。

## 目录

- [基础配置](#基础配置)
- [Scheduler 配置](#scheduler-配置)
- [Audit 审计配置](#audit-审计配置)
- [性能调优](#性能调优)
- [故障排查](#故障排查)

---

## 基础配置

Tiga 支持通过 `config.yaml` 文件或环境变量进行配置。配置文件优先级高于环境变量。

### 最小配置示例

```yaml
server:
    install_lock: true
    app_name: Tiga Dashboard
    http_port: 12306
    grpc_port: 12307
database:
    type: sqlite
    database: /var/lib/tiga/data/tiga.db
security:
    jwt_secret: <your-jwt-secret>
    encryption_key: <your-encryption-key>
```

---

## Scheduler 配置

Scheduler 子系统管理定时任务调度和执行历史记录。

### 配置参数

| 参数                        | 环境变量                          | 默认值 | 说明                          |
| --------------------------- | --------------------------------- | ------ | ----------------------------- |
| `default_timeout_seconds`   | `SCHEDULER_DEFAULT_TIMEOUT`       | 3600   | 任务默认超时时间（秒）        |
| `default_max_retries`       | `SCHEDULER_DEFAULT_MAX_RETRIES`   | 3      | 任务失败后的最大重试次数      |
| `default_grace_period`      | `SCHEDULER_DEFAULT_GRACE_PERIOD`  | 30     | 任务超时后的宽限期（秒）      |

### 配置示例

```yaml
# config.yaml
scheduler:
    default_timeout_seconds: 3600  # 1 小时超时
    default_max_retries: 3          # 最多重试 3 次
    default_grace_period: 30        # 30 秒宽限期
```

### 配置说明

- **超时时间（Timeout）**：任务执行的最大时长。超时后，Scheduler 会尝试优雅地终止任务。
- **宽限期（Grace Period）**：任务超时后，给予任务清理资源的额外时间。宽限期结束后强制终止任务。
- **最大重试次数**：任务失败后自动重试的最大次数。每次重试之间有指数退避延迟。

### 性能调优建议

1. **调整超时时间**：
   - 短期任务（<5 分钟）：`default_timeout_seconds: 300`
   - 中期任务（5-30 分钟）：`default_timeout_seconds: 1800`
   - 长期任务（>30 分钟）：`default_timeout_seconds: 7200`

2. **重试策略**：
   - 高可靠性任务：`default_max_retries: 5`
   - 普通任务：`default_max_retries: 3`
   - 幂等性差的任务：`default_max_retries: 0`

3. **宽限期设置**：
   - 需要清理资源的任务：`default_grace_period: 60`
   - 快速终止任务：`default_grace_period: 10`

---

## Audit 审计配置

Audit 子系统记录所有关键操作的审计日志，支持追溯和合规审查。

### 配置参数

| 参数                | 环境变量                   | 默认值  | 说明                           |
| ------------------- | -------------------------- | ------- | ------------------------------ |
| `retention_days`    | `AUDIT_RETENTION_DAYS`     | 90      | 审计日志保留天数               |
| `max_object_bytes`  | `AUDIT_MAX_OBJECT_BYTES`   | 65536   | 对象差异字段最大字节数（64KB） |

### 配置示例

```yaml
# config.yaml
audit:
    retention_days: 90              # 保留 90 天审计日志
    max_object_bytes: 65536         # 对象最大 64KB
```

### 配置说明

- **保留期（Retention Days）**：审计日志的保留天数。超过保留期的日志会被自动清理（每天 2:00 AM 执行清理任务）。
- **对象大小限制（Max Object Bytes）**：审计日志中 `OldObject` 和 `NewObject` 字段的最大字节数。超过限制的对象会被智能截断（保留 JSON 结构，截断字段值）。

### 支持的子系统

Tiga 审计系统统一记录所有子系统的操作日志，通过 `subsystem` 字段区分不同来源：

| 子系统 | 标识符 | 说明 |
|--------|--------|------|
| HTTP API | `http` | 通用 HTTP API 操作 |
| MinIO | `minio` | MinIO 对象存储操作 |
| Database | `database` | 数据库管理操作 |
| Middleware | `middleware` | Redis、MySQL、PostgreSQL 等中间件管理 |
| Kubernetes | `kubernetes` | K8s 集群管理和资源操作 |
| Docker | `docker` | Docker 实例管理 |
| Host | `host` | 主机监控和管理（VMs 子系统） |
| WebSSH | `webssh` | SSH 终端会话 |
| Scheduler | `scheduler` | 定时任务调度 |
| Alert | `alert` | 告警规则和事件 |
| Auth | `auth` | 认证和授权 |
| Storage | `storage` | 存储管理 |
| WebServer | `webserver` | Web 服务器管理 |

**查询示例**：

```sql
-- 查询 Kubernetes 子系统的所有操作
SELECT * FROM audit_events WHERE subsystem = 'kubernetes' ORDER BY timestamp DESC LIMIT 100;

-- 查询主机管理的删除操作
SELECT * FROM audit_events WHERE subsystem = 'host' AND action = 'deleted';

-- 统计各子系统的操作数量
SELECT subsystem, COUNT(*) as count
FROM audit_events
WHERE timestamp > EXTRACT(EPOCH FROM NOW() - INTERVAL '7 days') * 1000
GROUP BY subsystem
ORDER BY count DESC;
```

**详细使用指南**：参见 `docs/audit-subsystems.md`

### 性能调优建议

1. **保留期设置**：
   - 合规要求严格：`retention_days: 180`（6 个月）
   - 一般环境：`retention_days: 90`（3 个月）
   - 开发测试环境：`retention_days: 30`（1 个月）

2. **对象大小限制**：
   - 大型对象场景：`max_object_bytes: 131072`（128KB）
   - 一般场景：`max_object_bytes: 65536`（64KB）
   - 低存储环境：`max_object_bytes: 32768`（32KB）

3. **数据库优化**：
   - 审计日志表建议使用 PostgreSQL（支持 JSONB 和 GIN 索引）
   - 定期运行 VACUUM 操作清理已删除记录
   - 考虑使用表分区（按月分区）提升查询性能

---

## 性能调优

### 数据库性能

#### PostgreSQL 推荐配置

```yaml
database:
    type: postgres
    host: localhost
    port: 5432
    name: tiga
    max_open_conns: 100     # 最大连接数
    max_idle_conns: 10      # 最大空闲连接数
```

**调优建议**：
- 生产环境：`max_open_conns: 200`, `max_idle_conns: 20`
- 高并发环境：`max_open_conns: 500`, `max_idle_conns: 50`
- 低资源环境：`max_open_conns: 50`, `max_idle_conns: 5`

#### 审计日志查询优化

审计日志表已包含以下索引：
- `idx_audit_events_timestamp`（时间戳索引）
- `idx_audit_events_action`（操作类型索引）
- `idx_audit_events_resource_type`（资源类型索引）
- `idx_audit_events_subsystem`（子系统索引）
- `idx_audit_events_composite`（复合索引：资源类型、操作、时间戳）

**性能目标**：10,000 条记录查询 < 2 秒

### Scheduler 性能

#### 任务并发控制

每个任务的 `max_concurrent` 字段控制并发执行数：
- 单实例应用：`max_concurrent: 1`（默认）
- 资源密集型任务：`max_concurrent: 1`
- 轻量级任务：`max_concurrent: 5`

#### 执行历史清理

定期清理旧的执行历史记录以保持数据库性能：

```sql
-- 删除 30 天前的执行历史
DELETE FROM task_executions WHERE started_at < NOW() - INTERVAL '30 days';
```

建议通过 Scheduled Task 自动化执行此清理操作。

### 内存优化

#### Go 应用内存限制

通过环境变量设置 Go 应用的内存限制：

```bash
GOMEMLIMIT=2GiB  # 限制 Go 堆内存为 2GB
```

**推荐配置**：
- 小型部署（<100 用户）：`GOMEMLIMIT=1GiB`
- 中型部署（100-1000 用户）：`GOMEMLIMIT=4GiB`
- 大型部署（>1000 用户）：`GOMEMLIMIT=8GiB`

---

## 故障排查

### Scheduler 问题

#### 问题 1：任务超时频繁发生

**症状**：任务执行历史显示大量 `timeout` 状态。

**排查步骤**：
1. 检查任务配置的 `max_duration_seconds` 是否过短
2. 查看任务执行日志（`task_executions.result` 字段）
3. 检查系统资源（CPU、内存）是否不足

**解决方案**：
```yaml
# 增加超时时间
scheduler:
    default_timeout_seconds: 7200  # 增加到 2 小时
```

#### 问题 2：任务重试过多导致资源浪费

**症状**：同一任务多次重试失败，消耗大量系统资源。

**排查步骤**：
1. 查看任务执行历史的 `retry_count` 字段
2. 检查任务失败的原因（`result` 和 `error` 字段）
3. 确认任务是否适合重试（是否有副作用）

**解决方案**：
```yaml
# 减少重试次数
scheduler:
    default_max_retries: 1  # 只重试 1 次
```

#### 问题 3：任务卡在 `running` 状态

**症状**：任务显示 `running` 状态，但长时间未完成。

**排查步骤**：
1. 检查应用日志是否有 Panic 或崩溃
2. 查看任务的 `started_at` 时间戳
3. 检查是否超过 `max_duration_seconds + timeout_grace_period`

**解决方案**：
- 重启应用程序（Scheduler 会自动将 `running` 状态的过期任务标记为 `interrupted`）
- 增加宽限期以避免强制终止：
  ```yaml
  scheduler:
      default_grace_period: 60  # 增加到 60 秒
  ```

### Audit 审计日志问题

#### 问题 1：审计日志查询缓慢

**症状**：审计日志列表页加载时间超过 5 秒。

**排查步骤**：
1. 检查审计日志表的记录数：
   ```sql
   SELECT COUNT(*) FROM audit_events;
   ```
2. 检查是否缺少索引：
   ```sql
   \d audit_events  -- PostgreSQL
   ```
3. 运行 EXPLAIN 分析查询计划：
   ```sql
   EXPLAIN ANALYZE SELECT * FROM audit_events WHERE timestamp > ...;
   ```

**解决方案**：
- 减少保留期：
  ```yaml
  audit:
      retention_days: 30  # 减少到 30 天
  ```
- 手动创建缺失的索引：
  ```sql
  CREATE INDEX idx_audit_events_user_uid ON audit_events(user_uid);
  ```
- 使用表分区（PostgreSQL 11+）：
  ```sql
  -- 按月分区
  CREATE TABLE audit_events_2025_10 PARTITION OF audit_events
      FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
  ```

#### 问题 2：对象截断过多

**症状**：审计日志中的 `old_object_truncated` 或 `new_object_truncated` 字段频繁为 `true`。

**排查步骤**：
1. 检查对象大小分布：
   ```sql
   SELECT AVG(LENGTH(diff_object::text)) AS avg_size,
          MAX(LENGTH(diff_object::text)) AS max_size
   FROM audit_events
   WHERE diff_object IS NOT NULL;
   ```
2. 查看被截断的字段列表：
   ```sql
   SELECT truncated_fields FROM audit_events
   WHERE old_object_truncated = true OR new_object_truncated = true
   LIMIT 10;
   ```

**解决方案**：
- 增加对象大小限制：
  ```yaml
  audit:
      max_object_bytes: 131072  # 增加到 128KB
  ```
- 优化应用逻辑，减少不必要的大对象字段

#### 问题 3：审计日志未记录某些操作

**症状**：某些 API 操作没有生成审计日志。

**排查步骤**：
1. 检查 `internal/api/middleware/audit.go` 中的 `shouldSkipAudit()` 函数
2. 确认该 API 路由是否添加了审计中间件
3. 检查应用日志是否有 "Invalid audit event" 错误

**解决方案**：
- 确认路由注册时包含了 `middleware.AuditLog()` 中间件：
  ```go
  protected := v1.Group("")
  protected.Use(middleware.AuthRequired(), middleware.AuditLog())
  ```
- 检查 AsyncLogger 是否启动：
  ```go
  asyncLogger.Start()  // 确保调用了 Start()
  ```

### 数据库连接问题

#### 问题 1：连接池耗尽

**症状**：应用日志显示 "too many connections" 或 "connection pool timeout"。

**排查步骤**：
1. 检查当前连接数：
   ```sql
   SELECT COUNT(*) FROM pg_stat_activity;  -- PostgreSQL
   ```
2. 查看连接池配置：
   ```yaml
   database:
       max_open_conns: 100
       max_idle_conns: 10
   ```

**解决方案**：
- 增加连接池大小：
  ```yaml
  database:
      max_open_conns: 200
      max_idle_conns: 20
  ```
- 检查是否有连接泄漏（未关闭的查询）

#### 问题 2：SQLite 锁定错误

**症状**：应用日志显示 "database is locked"。

**排查步骤**：
1. 检查是否有多个进程同时访问 SQLite 数据库
2. 确认 SQLite 数据库文件权限正确

**解决方案**：
- 切换到 PostgreSQL（生产环境推荐）：
  ```yaml
  database:
      type: postgres
      host: localhost
      port: 5432
      name: tiga
  ```
- 减少并发写入操作
- 启用 WAL 模式：
  ```sql
  PRAGMA journal_mode=WAL;
  ```

---

## 健康检查

### 应用健康检查端点

- **健康检查**：`GET /health`
- **就绪检查**：`GET /ready`
- **Swagger 文档**：`http://localhost:12306/swagger/index.html`

### 监控指标

推荐使用 Prometheus 监控以下指标：
- Scheduler 任务执行成功率
- Scheduler 任务平均执行时间
- 审计日志写入速率
- 数据库连接池使用率
- API 响应时间（P50、P95、P99）

---

## 日志配置

### 日志级别

通过环境变量设置日志级别：

```bash
LOG_LEVEL=debug  # debug, info, warn, error
LOG_FORMAT=json  # json, text
```

**推荐配置**：
- 开发环境：`LOG_LEVEL=debug`, `LOG_FORMAT=text`
- 生产环境：`LOG_LEVEL=info`, `LOG_FORMAT=json`

### 日志输出

应用默认将日志输出到标准输出（stdout）。可以通过 Docker 或 systemd 重定向到文件：

```bash
# Docker
docker run -v /var/log/tiga:/logs tiga:latest 2>&1 | tee /logs/tiga.log

# systemd
StandardOutput=append:/var/log/tiga/tiga.log
StandardError=append:/var/log/tiga/tiga-error.log
```

---

## 备份与恢复

### 数据库备份

#### PostgreSQL

```bash
# 备份数据库
pg_dump -U tiga -d tiga -F c -f tiga_backup_$(date +%Y%m%d).dump

# 恢复数据库
pg_restore -U tiga -d tiga -c tiga_backup_20251020.dump
```

#### SQLite

```bash
# 备份数据库
cp /var/lib/tiga/data/tiga.db /backup/tiga_backup_$(date +%Y%m%d).db

# 恢复数据库
cp /backup/tiga_backup_20251020.db /var/lib/tiga/data/tiga.db
```

### 配置文件备份

定期备份 `config.yaml` 文件：

```bash
cp config.yaml /backup/config_$(date +%Y%m%d).yaml
```

---

## 安全建议

1. **JWT Secret**：使用强随机字符串（至少 32 字符）
2. **加密密钥**：使用 Base64 编码的 32 字节随机密钥
3. **数据库密码**：避免在 `config.yaml` 中明文存储，使用环境变量：
   ```bash
   export DB_PASSWORD="your-secure-password"
   ```
4. **HTTPS**：生产环境必须启用 TLS/SSL
5. **审计日志**：定期审查审计日志，检测异常操作

---

## 升级指南

### 版本升级步骤

1. **备份数据库和配置文件**
2. **停止应用**：
   ```bash
   systemctl stop tiga
   ```
3. **替换二进制文件**：
   ```bash
   cp tiga /usr/local/bin/tiga
   chmod +x /usr/local/bin/tiga
   ```
4. **运行数据库迁移**（自动执行，启动时）
5. **启动应用**：
   ```bash
   systemctl start tiga
   ```
6. **验证健康状态**：
   ```bash
   curl http://localhost:12306/health
   ```

### 数据库迁移

Tiga 使用 GORM AutoMigrate 自动处理数据库迁移。启动时会自动创建或更新表结构。

---

## 支持与反馈

- **文档**：[https://github.com/ysicing/tiga](https://github.com/ysicing/tiga)
- **问题反馈**：提交 GitHub Issue
- **Swagger API 文档**：`http://localhost:12306/swagger/index.html`

---

**最后更新**：2025-10-20
**适用版本**：Tiga v1.0+（Phase 006-gitness-tiga）
