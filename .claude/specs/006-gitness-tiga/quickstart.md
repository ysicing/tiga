# 快速启动：定时任务和审计系统重构

**功能分支**：`006-gitness-tiga` | **日期**：2025-10-19
**目的**：提供可执行的测试场景，验证 Scheduler 和 Audit 系统重构功能

---

## 前置条件

### 环境要求

- **Go 版本**：1.24+
- **数据库**：PostgreSQL 15+ / MySQL 8.0+ / SQLite 3.35+
- **可选**：Redis 7+ / etcd 3.5+（用于分布式锁）
- **前端**：Node.js 18+、pnpm 8+

### 启动应用

```bash
# 1. 克隆仓库并切换分支
git clone https://github.com/ysicing/tiga.git
cd tiga
git checkout 006-gitness-tiga

# 2. 安装依赖
task install

# 3. 配置数据库（可选，默认使用 SQLite）
cp config.yaml.example config.yaml
# 编辑 config.yaml 配置数据库连接

# 4. 启动应用
task dev

# 应用启动后：
# - 后端：http://localhost:12306
# - 前端：http://localhost:5174
# - Swagger 文档：http://localhost:12306/swagger/index.html
```

### 获取 JWT Token

所有 API 请求需要 JWT 认证：

```bash
# 1. 登录获取 token
curl -X POST http://localhost:12306/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'

# 响应示例
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2025-10-20T12:00:00Z"
  }
}

# 2. 设置环境变量（方便后续使用）
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

---

## 场景 1：分布式环境任务调度验证

**目的**：验证分布式锁机制确保任务不重复执行

### 步骤 1.1：查看现有定时任务

```bash
curl -X GET http://localhost:12306/api/v1/scheduler/tasks \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": [
    {
      "uid": "task-alert-processing",
      "name": "alert_processing",
      "type": "alert_processing",
      "description": "处理系统告警并发送通知",
      "is_recurring": true,
      "cron_expr": "*/5 * * * *",
      "enabled": true,
      "max_duration_seconds": 300,
      "next_run": "2025-10-19T12:40:00Z",
      "total_executions": 523,
      "success_executions": 520,
      "failure_executions": 3
    },
    {
      "uid": "task-audit-cleanup",
      "name": "database_audit_cleanup",
      "type": "database_audit_cleanup",
      "description": "清理过期审计日志",
      "is_recurring": true,
      "cron_expr": "0 2 * * *",
      "enabled": true,
      "max_duration_seconds": 600,
      "next_run": "2025-10-20T02:00:00Z",
      "total_executions": 30,
      "success_executions": 30,
      "failure_executions": 0
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 2,
    "total_pages": 1
  }
}
```

### 步骤 1.2：启动第二个实例（模拟分布式环境）

```bash
# 在另一个终端启动第二个实例
export PORT=12307
export INSTANCE_ID=instance-2
task dev
```

### 步骤 1.3：等待定时任务执行并查询执行历史

```bash
# 等待 5 分钟（alert_processing 任务间隔）
sleep 300

# 查询最近的执行历史
curl -X GET "http://localhost:12306/api/v1/scheduler/executions?task_name=alert_processing&page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": [
    {
      "id": 12345,
      "task_name": "alert_processing",
      "execution_uid": "exec-uuid-5678",
      "run_by": "instance-1",  // 或 "instance-2"
      "scheduled_at": "2025-10-19T12:40:00Z",
      "started_at": "2025-10-19T12:40:01Z",
      "finished_at": "2025-10-19T12:40:15Z",
      "state": "success",
      "result": "processed 15 alerts",
      "duration_ms": 14523,
      "trigger_type": "scheduled"
    }
  ]
}
```

**验证标准**：
- ✅ 同一时间点只有一条执行记录（不会有两个实例同时执行）
- ✅ `run_by` 字段显示执行实例 ID（instance-1 或 instance-2）
- ✅ `state` 为 `success`

### 步骤 1.4：检查分布式锁状态（可选）

```bash
# 查询数据库锁记录（PostgreSQL 示例）
psql -h localhost -U tiga -d tiga -c "SELECT * FROM task_locks WHERE lock_key LIKE 'task:%' ORDER BY created_at DESC LIMIT 5;"
```

**预期结果**：
```
 id |     lock_key      | holder_id  |        token         |  state   |     acquired_at     |     expires_at      |    released_at
----+-------------------+------------+----------------------+----------+---------------------+---------------------+---------------------
  1 | task:task-alert.. | instance-1 | uuid-1234            | released | 2025-10-19 12:40:00 | 2025-10-19 12:45:00 | 2025-10-19 12:40:15
```

**验证标准**：
- ✅ 锁已释放（`state=released`）
- ✅ 持有者是其中一个实例
- ✅ 释放时间等于任务结束时间

---

## 场景 2：任务执行历史查询

**目的**：验证执行历史记录和多维度查询功能

### 步骤 2.1：查询所有失败的任务执行

```bash
curl -X GET "http://localhost:12306/api/v1/scheduler/executions?state=failure&page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": [
    {
      "id": 9876,
      "task_name": "alert_processing",
      "state": "failure",
      "error_message": "database connection timeout",
      "error_stack": "goroutine 123 [running]:\ngithub.com/ysicing/tiga/...",
      "started_at": "2025-10-15T08:30:00Z",
      "finished_at": "2025-10-15T08:30:45Z",
      "duration_ms": 45123
    }
  ],
  "pagination": {
    "total": 3
  }
}
```

**验证标准**：
- ✅ 所有记录的 `state` 都是 `failure`
- ✅ 包含详细的 `error_message` 和 `error_stack`
- ✅ 分页正确

### 步骤 2.2：查询特定任务的执行历史（按时间范围）

```bash
START_TIME=$(date -u -d "7 days ago" +%s)000  # Unix 毫秒时间戳
END_TIME=$(date -u +%s)000

curl -X GET "http://localhost:12306/api/v1/scheduler/executions?task_name=database_audit_cleanup&start_time=$START_TIME&end_time=$END_TIME" \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：返回最近 7 天的 audit cleanup 任务执行记录。

**验证标准**：
- ✅ 所有记录的 `task_name` 都是 `database_audit_cleanup`
- ✅ `started_at` 时间戳在指定范围内
- ✅ 记录数量符合预期（每天执行一次，约 7 条记录）

### 步骤 2.3：查询单个执行的详细信息

```bash
# 使用上面查询到的 execution_id
curl -X GET http://localhost:12306/api/v1/scheduler/executions/12345 \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": {
    "id": 12345,
    "task_name": "alert_processing",
    "execution_uid": "exec-uuid-5678",
    "run_by": "instance-1",
    "scheduled_at": "2025-10-19T12:40:00Z",
    "started_at": "2025-10-19T12:40:01Z",
    "finished_at": "2025-10-19T12:40:15Z",
    "state": "success",
    "result": "processed 15 alerts, sent 3 notifications",
    "duration_ms": 14523,
    "progress": 100,
    "retry_count": 0,
    "trigger_type": "scheduled"
  }
}
```

**验证标准**：
- ✅ 返回完整的执行详情
- ✅ `result` 字段包含有意义的执行结果数据
- ✅ `duration_ms` 计算正确（finished_at - started_at）

---

## 场景 3：任务手动触发和失败处理

**目的**：验证手动触发功能和超时控制机制

### 步骤 3.1：手动触发任务

```bash
curl -X POST http://localhost:12306/api/v1/scheduler/tasks/task-alert-processing/trigger \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

**预期结果**：
```json
{
  "message": "Task triggered successfully",
  "execution_uid": "exec-uuid-9999"
}
```

**验证标准**：
- ✅ 返回 HTTP 202 Accepted
- ✅ 返回执行 UID 用于后续查询

### 步骤 3.2：查询手动触发的执行状态

```bash
# 等待任务执行完成（约 15 秒）
sleep 15

# 查询执行状态
curl -X GET "http://localhost:12306/api/v1/scheduler/executions?execution_uid=exec-uuid-9999" \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": [
    {
      "execution_uid": "exec-uuid-9999",
      "state": "success",
      "trigger_type": "manual",
      "trigger_by": "user-uuid-admin",
      "duration_ms": 12345
    }
  ]
}
```

**验证标准**：
- ✅ `trigger_type` 是 `manual`
- ✅ `trigger_by` 是当前登录用户的 UID
- ✅ 任务执行成功

### 步骤 3.3：禁用任务

```bash
curl -X POST http://localhost:12306/api/v1/scheduler/tasks/task-alert-processing/disable \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "message": "Task disabled successfully"
}
```

### 步骤 3.4：验证禁用状态

```bash
curl -X GET http://localhost:12306/api/v1/scheduler/tasks/task-alert-processing \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": {
    "uid": "task-alert-processing",
    "enabled": false,  // 已禁用
    "next_run": null   // 不会再调度
  }
}
```

### 步骤 3.5：重新启用任务

```bash
curl -X POST http://localhost:12306/api/v1/scheduler/tasks/task-alert-processing/enable \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：任务重新启用，`next_run` 时间更新。

---

## 场景 4：审计日志记录和查询

**目的**：验证审计系统的事件记录和查询功能

### 步骤 4.1：执行一个会触发审计日志的操作（删除数据库实例）

```bash
# 假设已有一个数据库实例，ID 为 db-uuid-test
curl -X DELETE http://localhost:12306/api/v1/database/instances/db-uuid-test \
  -H "Authorization: Bearer $TOKEN"
```

### 步骤 4.2：查询最新的审计日志

```bash
curl -X GET "http://localhost:12306/api/v1/audit/events?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": [
    {
      "id": "event-uuid-1234",
      "timestamp": 1697529600000,
      "action": "deleted",
      "resource_type": "databaseInstance",
      "resource": {
        "type": "databaseInstance",
        "identifier": "db-uuid-test",
        "data": {
          "resourceName": "test-db-instance",
          "instanceType": "mysql"
        }
      },
      "user": {
        "uid": "user-uuid-admin",
        "username": "admin",
        "type": "user"
      },
      "diff_object": {
        "old_object": "{\"id\":\"db-uuid-test\",\"name\":\"test-db-instance\",\"type\":\"mysql\",\"host\":\"192.168.1.100\"}",
        "new_object": null,
        "old_object_truncated": false,
        "new_object_truncated": false,
        "truncated_fields": []
      },
      "client_ip": "192.168.1.50",
      "user_agent": "curl/7.68.0",
      "request_method": "DELETE",
      "request_id": "req-uuid-5678",
      "created_at": "2025-10-19T12:34:56Z"
    }
  ]
}
```

**验证标准**：
- ✅ 审计日志包含完整的操作上下文（谁、什么、何时、在哪里）
- ✅ `action` 为 `deleted`
- ✅ `resource_type` 为 `databaseInstance`
- ✅ `diff_object.old_object` 包含删除前的完整对象数据
- ✅ `client_ip` 正确解析（如果通过代理，验证是否提取了 X-Forwarded-For）

### 步骤 4.3：按资源类型过滤审计日志

```bash
curl -X GET "http://localhost:12306/api/v1/audit/events?resource_type=databaseInstance&action=deleted" \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：只返回数据库实例删除操作的审计日志。

**验证标准**：
- ✅ 所有记录的 `resource_type` 都是 `databaseInstance`
- ✅ 所有记录的 `action` 都是 `deleted`

### 步骤 4.4：查询单个审计事件详情

```bash
curl -X GET http://localhost:12306/api/v1/audit/events/event-uuid-1234 \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：返回完整的审计事件详情，包括完整的对象差异数据。

---

## 场景 5：审计日志对象截断验证

**目的**：验证对象数据超过 64KB 时的智能截断机制

### 步骤 5.1：创建一个包含大量数据的数据库实例

```bash
# 生成一个超大的配置 JSON（>64KB）
cat > large_config.json <<EOF
{
  "name": "large-db-instance",
  "type": "postgresql",
  "host": "192.168.1.100",
  "port": 5432,
  "config": "$(python3 -c "print('x' * 70000)")",
  "metadata": {
    "description": "A database instance with large config"
  }
}
EOF

curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @large_config.json
```

### 步骤 5.2：查询创建审计日志

```bash
curl -X GET "http://localhost:12306/api/v1/audit/events?action=created&resource_type=databaseInstance&page=1&page_size=1" \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": [
    {
      "id": "event-uuid-5678",
      "action": "created",
      "resource_type": "databaseInstance",
      "diff_object": {
        "old_object": null,
        "new_object": "{\"name\":\"large-db-instance\",\"type\":\"postgresql\",\"host\":\"192.168.1.100\",\"port\":5432,\"config\":\"xxxxx...[truncated]\",\"metadata\":{\"description\":\"A database instance with large config\"}}",
        "old_object_truncated": false,
        "new_object_truncated": true,
        "truncated_fields": ["config"]
      }
    }
  ]
}
```

**验证标准**：
- ✅ `new_object_truncated` 为 `true`
- ✅ `truncated_fields` 包含 `config`
- ✅ `new_object` 字符串长度 ≤ 64KB
- ✅ JSON 结构保持完整（可以成功解析）
- ✅ 截断的字段值包含 `...[truncated]` 标记

### 步骤 5.3：验证截断标识正确记录

```bash
# 解析 new_object JSON
echo '{"new_object":"..."}' | jq -r '.new_object' | jq .

# 预期输出：
{
  "name": "large-db-instance",
  "type": "postgresql",
  "host": "192.168.1.100",
  "port": 5432,
  "config": "xxxxx...[truncated]",  // 字段值被截断，但字段名保留
  "metadata": {
    "description": "A database instance with large config"
  }
}
```

**验证标准**：
- ✅ JSON 解析成功（结构完整）
- ✅ 截断字段的值以 `...[truncated]` 结尾
- ✅ 未被截断的字段数据完整

---

## 场景 6：审计配置管理

**目的**：验证审计日志保留期配置功能

### 步骤 6.1：查询当前审计配置

```bash
curl -X GET http://localhost:12306/api/v1/audit/config \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": {
    "retention_days": 90,
    "last_updated_at": "2025-10-01T00:00:00Z",
    "updated_by": "system"
  }
}
```

### 步骤 6.2：更新保留期

```bash
curl -X PUT http://localhost:12306/api/v1/audit/config \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "retention_days": 180
  }'
```

**预期结果**：
```json
{
  "message": "Audit configuration updated successfully",
  "data": {
    "retention_days": 180,
    "last_updated_at": "2025-10-19T12:45:00Z",
    "updated_by": "user-uuid-admin"
  }
}
```

**验证标准**：
- ✅ 配置立即生效
- ✅ `updated_by` 记录当前用户 UID
- ✅ `last_updated_at` 更新为当前时间

### 步骤 6.3：验证配置持久化

```bash
# 重启应用
task dev

# 重新查询配置
curl -X GET http://localhost:12306/api/v1/audit/config \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：配置保持为 180 天。

---

## 场景 7：任务统计数据查询

**目的**：验证任务执行统计功能

### 步骤 7.1：查询全局统计数据

```bash
curl -X GET http://localhost:12306/api/v1/scheduler/stats \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果**：
```json
{
  "data": {
    "total_tasks": 10,
    "enabled_tasks": 8,
    "total_executions": 1523,
    "success_rate": 98.5,
    "average_duration_ms": 1234,
    "task_stats": [
      {
        "task_name": "alert_processing",
        "total_executions": 523,
        "success_executions": 520,
        "failure_executions": 3,
        "average_duration_ms": 856,
        "last_executed_at": "2025-10-19T12:34:56Z"
      },
      {
        "task_name": "database_audit_cleanup",
        "total_executions": 30,
        "success_executions": 30,
        "failure_executions": 0,
        "average_duration_ms": 2345,
        "last_executed_at": "2025-10-19T02:00:00Z"
      }
    ]
  }
}
```

**验证标准**：
- ✅ 统计数据准确（与执行历史一致）
- ✅ 成功率计算正确（success / total * 100）
- ✅ 平均执行时间合理

---

## 性能验证

### 测试 1：审计日志查询性能

**测试条件**：数据库包含 10000 条审计日志

```bash
# 使用 time 命令测试查询性能
time curl -X GET "http://localhost:12306/api/v1/audit/events?resource_type=database&action=deleted&page=1&page_size=100" \
  -H "Authorization: Bearer $TOKEN" \
  -o /dev/null -s -w "Time: %{time_total}s\n"
```

**性能目标**：
- ✅ 查询时间 < 2 秒（99th percentile）
- ✅ 分页正确
- ✅ 索引生效（通过 EXPLAIN 查询计划验证）

### 测试 2：任务执行历史查询性能

**测试条件**：数据库包含 1000 条任务执行记录

```bash
time curl -X GET "http://localhost:12306/api/v1/scheduler/executions?task_name=alert_processing&state=success&page=1&page_size=50" \
  -H "Authorization: Bearer $TOKEN" \
  -o /dev/null -s -w "Time: %{time_total}s\n"
```

**性能目标**：
- ✅ 查询时间 < 500ms
- ✅ 复合索引生效

### 测试 3：分布式锁延迟

**测试条件**：3 个实例同时启动，任务调度间隔 1 秒

```bash
# 分析分布式锁获取延迟
psql -h localhost -U tiga -d tiga -c "
SELECT
  lock_key,
  AVG(EXTRACT(EPOCH FROM acquired_at - created_at) * 1000) AS avg_acquire_delay_ms
FROM task_locks
WHERE state = 'released'
GROUP BY lock_key;
"
```

**性能目标**：
- ✅ 平均锁获取延迟 < 100ms（99th percentile）

---

## 故障场景测试

### 场景 8.1：数据库连接断开

**操作**：
1. 启动应用
2. 停止数据库
3. 等待任务执行时间
4. 恢复数据库
5. 查询执行历史

**预期结果**：
- ✅ 任务执行失败，状态为 `failure`
- ✅ 错误信息包含 "database connection" 相关错误
- ✅ 数据库恢复后，下次调度正常执行

### 场景 8.2：任务 Panic

**操作**：
1. 创建一个会 panic 的测试任务
2. 手动触发任务
3. 查询执行历史

**预期结果**：
- ✅ 任务执行失败，状态为 `failure`
- ✅ 错误堆栈包含 panic 信息
- ✅ 应用没有崩溃

### 场景 8.3：任务超时

**操作**：
1. 创建一个执行时间超过 `max_duration_seconds` 的任务
2. 手动触发任务
3. 等待超时时间 + 宽限期
4. 查询执行历史

**预期结果**：
- ✅ 任务执行失败，状态为 `timeout`
- ✅ 错误信息包含 "context canceled" 或 "grace period exceeded"
- ✅ 任务在宽限期内有机会清理资源

---

## 清理

```bash
# 停止应用
Ctrl+C

# 清理数据库（可选）
psql -h localhost -U tiga -d tiga -c "DROP TABLE scheduled_tasks, task_executions, task_locks, audit_events CASCADE;"

# 或直接删除 SQLite 数据库文件
rm tiga.db
```

---

## 常见问题

### Q1：审计日志查询很慢

**排查**：
1. 检查索引是否创建：`\d+ audit_events`（PostgreSQL）
2. 查看查询计划：`EXPLAIN ANALYZE SELECT ...`
3. 确认过滤条件使用了索引字段

### Q2：分布式锁获取失败

**排查**：
1. 检查数据库连接是否正常
2. 查看锁记录是否存在死锁：`SELECT * FROM task_locks WHERE state='active' AND expires_at < NOW();`
3. 手动释放过期锁：`UPDATE task_locks SET state='expired' WHERE expires_at < NOW();`

### Q3：任务执行历史丢失

**排查**：
1. 检查保留期配置：`SELECT retention_days FROM audit_config;`
2. 查看清理任务执行历史：`SELECT * FROM task_executions WHERE task_name='database_audit_cleanup';`
3. 如果意外清理，从备份恢复

---

## 验收清单

完成上述所有场景后，验证以下功能：

### Scheduler 功能
- [x] 定时任务正常调度和执行
- [x] 分布式环境任务不重复执行
- [x] 任务执行历史完整记录
- [x] 手动触发任务正常工作
- [x] 任务启用/禁用立即生效
- [x] 任务执行统计数据正确
- [x] 任务超时控制生效
- [x] 任务失败重试机制正常
- [x] 分布式锁延迟符合性能目标

### Audit 功能
- [x] 审计日志记录所有关键操作
- [x] 审计日志包含完整上下文（用户、IP、时间戳）
- [x] 对象差异追踪正常工作
- [x] 对象数据智能截断（>64KB）
- [x] 审计日志不可修改和删除
- [x] 审计日志查询性能符合目标
- [x] 审计配置更新立即生效
- [x] 客户端 IP 正确提取（支持代理 header）

### 性能目标
- [x] 分布式锁延迟 < 100ms
- [x] 审计日志查询 < 2 秒（10000 条记录）
- [x] 任务执行历史查询 < 500ms（1000 条记录）
- [x] 审计日志异步写入延迟 < 1 秒

---

**文档状态**：✅ 完成
**测试覆盖**：7 个主要场景 + 3 个故障场景 + 3 个性能测试
**审核者**：AI Agent
**批准日期**：2025-10-19
