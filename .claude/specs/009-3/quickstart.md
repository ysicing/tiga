# 快速启动指南：统一终端录制系统

**功能**：009-3 统一终端录制系统
**日期**：2025-10-26
**目的**：功能验证和集成测试场景

## 前置条件

### 环境要求
- Go 1.24+
- PostgreSQL/MySQL/SQLite（任选一个）
- Docker（用于集成测试）
- Node.js 18+（前端测试）

### 配置文件

创建测试配置文件 `config.test.yaml`：

```yaml
server:
  port: 12306
  host: 0.0.0.0

database:
  type: sqlite
  path: ./test.db

recording:
  storage_type: local
  base_path: ./test-data/recordings
  retention_days: 90
  cleanup_schedule: "0 4 * * *"
  cleanup_batch_size: 1000
  max_recording_size: 524288000  # 500MB
```

### 数据准备

```bash
# 1. 克隆仓库并切换到功能分支
git checkout 009-3

# 2. 安装依赖
go mod download
cd ui && pnpm install && cd ..

# 3. 运行数据库迁移
go run cmd/tiga/main.go --config config.test.yaml migrate

# 4. 创建测试用户
go run cmd/tiga/main.go --config config.test.yaml create-user \
  --username testuser \
  --password testpass123 \
  --role admin
```

---

## 测试场景

### 场景 1：Docker 容器终端录制（向后兼容验证）

**目的**：验证现有 Docker 录制功能仍然工作，数据迁移成功

**步骤**：

1. **启动应用**：
```bash
go run cmd/tiga/main.go --config config.test.yaml
```

2. **创建 Docker 终端会话**（模拟现有 API）：
```bash
curl -X POST http://localhost:12306/api/v1/docker/instances/{instance_id}/containers/{container_id}/terminal \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "shell": "/bin/sh",
    "rows": 30,
    "cols": 120
  }'
```

预期响应：
```json
{
  "success": true,
  "data": {
    "session_id": "uuid",
    "ws_url": "ws://localhost:12306/api/v1/docker/terminal/{session_id}",
    "expires_at": "2025-10-26T15:00:00Z"
  }
}
```

3. **连接 WebSocket 终端**（使用 wscat）：
```bash
wscat -c "ws://localhost:12306/api/v1/docker/terminal/{session_id}" \
  -H "Authorization: Bearer {token}"

# 发送命令
> {"type": "input", "data": "ls -la\n"}

# 接收输出
< {"type": "output", "data": "total 12\ndrwxr-xr-x ..."}
```

4. **结束会话并验证录制**：
```bash
# 关闭 WebSocket 连接
# 查询录制列表
curl http://localhost:12306/api/v1/recordings \
  -H "Authorization: Bearer {token}" \
  | jq '.data.recordings[] | select(.recording_type == "docker")'
```

**验收标准**：
- ✅ Docker 终端会话创建成功
- ✅ WebSocket 连接正常
- ✅ 录制文件保存在 `./test-data/recordings/YYYY-MM-DD/{recording_id}.cast`
- ✅ 数据库 `terminal_recordings` 表包含记录，`recording_type = 'docker'`
- ✅ `type_metadata` 字段包含 `instance_id` 和 `container_id`

---

### 场景 2：WebSSH 终端录制（新功能）

**目的**：验证 WebSSH 录制集成到统一系统

**步骤**：

1. **创建 WebSSH 会话**（通过现有 WebSSH 处理器）：
```bash
curl -X POST http://localhost:12306/api/v1/webssh/sessions \
  -H "Authorization: Bearer {token}" \
  -d '{
    "host_id": "uuid",
    "username": "root",
    "password": "encrypted-password"
  }'
```

2. **连接 WebSocket**：
```bash
wscat -c "ws://localhost:12306/api/v1/webssh/{session_id}"
```

3. **验证录制创建**：
```bash
curl http://localhost:12306/api/v1/recordings?recording_type=webssh \
  -H "Authorization: Bearer {token}"
```

**验收标准**：
- ✅ WebSSH 会话录制自动创建
- ✅ `recording_type = 'webssh'`
- ✅ `type_metadata` 包含 `host_id`, `host_name`, `ssh_port`
- ✅ 录制文件存储在统一路径

---

### 场景 3：K8s 节点终端录制（新功能）

**目的**：验证 K8s 节点终端录制集成

**步骤**：

1. **创建 K8s 节点终端会话**：
```bash
curl -X POST http://localhost:12306/api/v1/k8s/clusters/{cluster_id}/nodes/{node_name}/terminal \
  -H "Authorization: Bearer {token}" \
  -d '{
    "rows": 30,
    "cols": 120
  }'
```

2. **连接并执行命令**：
```bash
wscat -c "ws://localhost:12306/api/v1/k8s/terminal/{session_id}"
```

3. **验证录制元数据**：
```bash
curl http://localhost:12306/api/v1/recordings/{recording_id} \
  -H "Authorization: Bearer {token}" \
  | jq '.data.type_metadata'
```

预期 `type_metadata`：
```json
{
  "type": "k8s_node",
  "cluster_id": "uuid",
  "cluster_name": "prod-k8s-cluster",
  "node_name": "worker-node-01"
}
```

**验收标准**：
- ✅ K8s 节点终端录制创建成功
- ✅ `recording_type = 'k8s_node'`
- ✅ TypeMetadata 包含 cluster_id, node_name

---

### 场景 4：统一录制管理界面

**目的**：验证前端统一录制列表和过滤功能

**步骤**：

1. **访问录制列表页**：
```
http://localhost:5174/recordings
```

2. **按类型过滤**：
   - 选择"Docker 容器"过滤器
   - 验证只显示 `recording_type = docker` 的录制

3. **按用户过滤**：
   - 选择当前用户
   - 验证只显示当前用户的录制

4. **搜索录制**：
   - 输入搜索关键词（如容器名）
   - 验证搜索结果准确

5. **查看录制详情**：
   - 点击任意录制
   - 验证详情页显示完整信息（TypeMetadata、存储路径、文件大小等）

**验收标准**：
- ✅ 录制列表显示所有类型（Docker/WebSSH/K8s）
- ✅ 类型过滤器正常工作
- ✅ 用户过滤器正常工作
- ✅ 搜索功能返回正确结果
- ✅ 详情页显示完整元数据

---

### 场景 5：录制回放

**目的**：验证 Asciinema 录制回放功能

**步骤**：

1. **获取录制回放内容**：
```bash
curl http://localhost:12306/api/v1/recordings/{recording_id}/playback \
  -H "Authorization: Bearer {token}" \
  > recording.json
```

2. **验证 Asciinema v2 格式**：
```bash
# 检查第一行是否为 JSON header
head -n 1 recording.json | jq '.'

# 检查剩余行是否为帧数组
tail -n +2 recording.json | head -n 5
```

预期格式：
```
{"version":2,"width":120,"height":30,"timestamp":1698765432}
[0.123456, "o", "$ ls\r\n"]
[1.234567, "i", "ls\r\n"]
```

3. **前端播放验证**：
   - 访问 `http://localhost:5174/recordings/{id}/player`
   - 验证 Asciinema 播放器加载
   - 验证播放/暂停/快进功能

**验收标准**：
- ✅ API 返回正确的 Asciinema v2 格式
- ✅ 前端播放器正常渲染
- ✅ 播放控制（播放/暂停/速度调节）正常

---

### 场景 6：自动清理任务

**目的**：验证清理任务自动删除过期和无效录制

**准备数据**：

```sql
-- 插入过期录制（91 天前）
INSERT INTO terminal_recordings (
  id, session_id, user_id, username, recording_type,
  started_at, ended_at, duration, storage_path, file_size
) VALUES (
  gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 'testuser', 'docker',
  NOW() - INTERVAL '91 days', NOW() - INTERVAL '91 days', 3600, '/path/to/old.cast', 12345
);

-- 插入无效录制（零大小）
INSERT INTO terminal_recordings (
  id, session_id, user_id, username, recording_type,
  started_at, ended_at, duration, storage_path, file_size
) VALUES (
  gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 'testuser', 'webssh',
  NOW(), NOW(), 0, '/path/to/invalid.cast', 0
);
```

**测试步骤**：

1. **手动触发清理**：
```bash
curl -X POST http://localhost:12306/api/v1/recordings/cleanup/trigger \
  -H "Authorization: Bearer {admin_token}"
```

预期响应：
```json
{
  "success": true,
  "message": "清理任务已启动",
  "task_id": "uuid"
}
```

2. **等待 5 秒后查询状态**：
```bash
curl http://localhost:12306/api/v1/recordings/cleanup/status \
  -H "Authorization: Bearer {admin_token}"
```

预期响应：
```json
{
  "success": true,
  "data": {
    "last_run_at": "2025-10-26T14:00:00Z",
    "status": "completed",
    "invalid_cleaned": 1,
    "expired_cleaned": 1,
    "orphan_cleaned": 0,
    "total_space_freed": 12345,
    "error_message": null
  }
}
```

3. **验证录制已删除**：
```bash
# 查询数据库，确认过期和无效录制已删除
curl http://localhost:12306/api/v1/recordings \
  -H "Authorization: Bearer {token}" \
  | jq '.data.recordings | length'
# 预期：减少 2 条
```

**验收标准**：
- ✅ 清理任务成功启动（HTTP 202）
- ✅ 过期录制（>90 天）被删除
- ✅ 无效录制（零大小/零时长）被删除
- ✅ 清理状态 API 返回正确统计
- ✅ 文件系统中的 .cast 文件也被删除

---

### 场景 7：定时清理任务（Cron）

**目的**：验证 Cron 定时清理按计划执行

**配置**（测试环境，每分钟执行）：
```yaml
recording:
  cleanup_schedule: "* * * * *"  # 每分钟（仅测试）
```

**验证步骤**：

1. **启动应用并等待 2 分钟**

2. **查看日志**：
```bash
tail -f logs/tiga.log | grep RecordingCleanup
```

预期日志：
```
[RecordingCleanup] Starting scheduled cleanup
[RecordingCleanup] Completed: invalid=0, expired=0, orphan=0
[RecordingCleanup] Starting scheduled cleanup  # 第二次执行
```

3. **验证 Prometheus 指标**：
```bash
curl http://localhost:12306/metrics | grep recording_cleanup
```

预期指标：
```
recording_cleanup_runs_total 2
recording_cleanup_last_run_timestamp 1698765492
recording_cleanup_invalid_cleaned_total 0
recording_cleanup_expired_cleaned_total 0
```

**验收标准**：
- ✅ Cron 任务按 cleanup_schedule 执行
- ✅ 每次执行记录日志
- ✅ Prometheus 指标正确更新

---

### 场景 8：MinIO 对象存储（可选功能）

**目的**：验证 MinIO 存储后端集成

**前置条件**：
```bash
# 启动 MinIO（Docker）
docker run -d \
  -p 9000:9000 \
  -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# 创建 bucket
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/terminal-recordings
```

**配置**：
```yaml
recording:
  storage_type: minio
  base_path: terminal-recordings  # bucket 名称
  minio:
    endpoint: localhost:9000
    bucket: terminal-recordings
    access_key: minioadmin
    secret_key: minioadmin
    use_ssl: false
```

**测试步骤**：

1. **创建录制**（任意终端类型）

2. **验证文件上传到 MinIO**：
```bash
mc ls local/terminal-recordings/2025-10-26/
```

预期输出：
```
[2025-10-26 14:30:00 CST]  12KiB {recording-id}.cast
```

3. **下载录制验证**：
```bash
curl http://localhost:12306/api/v1/recordings/{id}/download \
  -H "Authorization: Bearer {token}" \
  -o downloaded.cast

# 验证文件完整性
diff downloaded.cast <(mc cat local/terminal-recordings/2025-10-26/{id}.cast)
```

**验收标准**：
- ✅ 录制文件成功上传到 MinIO
- ✅ 文件按日期目录组织（`YYYY-MM-DD/`）
- ✅ 下载 API 从 MinIO 正确读取
- ✅ 清理任务删除 MinIO 对象

---

### 场景 9：并发录制性能测试

**目的**：验证系统支持 100+ 并发终端会话

**工具**：使用 `k6` 负载测试工具

**测试脚本**（k6-load-test.js）：
```javascript
import ws from 'k6/ws';
import { check } from 'k6';

export let options = {
  vus: 100,  // 100 虚拟用户
  duration: '5m',
};

export default function () {
  const url = 'ws://localhost:12306/api/v1/docker/terminal/{session_id}';
  const params = { headers: { 'Authorization': 'Bearer {token}' } };

  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function open() {
      // 发送终端输入
      socket.send(JSON.stringify({ type: 'input', data: 'echo test\n' }));
    });

    socket.on('message', function (message) {
      check(message, {
        'received output': (msg) => msg.includes('output'),
      });
    });

    socket.setTimeout(function () {
      socket.close();
    }, 60000); // 1 分钟会话
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
```

**执行测试**：
```bash
k6 run k6-load-test.js
```

**验收标准**：
- ✅ 100 并发 WebSocket 连接成功率 >99%
- ✅ P99 消息延迟 <100ms
- ✅ 无内存泄漏（稳定在 2GB 以内）
- ✅ 所有录制文件完整保存

---

### 场景 10：数据迁移验证

**目的**：验证现有 Docker 录制数据迁移到统一模型

**前置条件**：
```sql
-- 创建旧格式数据（仅有 instance_id, container_id）
INSERT INTO terminal_recordings (
  id, session_id, instance_id, container_id, user_id, username,
  started_at, ended_at, duration, storage_path, file_size, format
) VALUES (
  gen_random_uuid(), gen_random_uuid(),
  'old-instance-id', 'old-container-id',
  gen_random_uuid(), 'testuser',
  NOW() - INTERVAL '1 hour', NOW(), 3600,
  '/old/path/recording.cast', 12345, 'asciinema'
);
```

**迁移步骤**：

1. **启动应用（自动迁移）**：
```bash
go run cmd/tiga/main.go --config config.test.yaml
```

预期日志：
```
Migrating 1 existing Docker recordings to unified model
Successfully migrated 1 recordings
```

2. **验证迁移结果**：
```sql
SELECT
  id,
  recording_type,
  type_metadata,
  instance_id,  -- 应仍然保留（向后兼容）
  container_id  -- 应仍然保留
FROM terminal_recordings
WHERE id = '{migrated_id}';
```

预期结果：
```
recording_type: 'docker'
type_metadata: {
  "type": "docker",
  "instance_id": "old-instance-id",
  "container_id": "old-container-id"
}
instance_id: 'old-instance-id'  -- 保留
container_id: 'old-container-id'  -- 保留
```

**验收标准**：
- ✅ 所有旧数据成功迁移
- ✅ `recording_type` 设置为 'docker'
- ✅ `type_metadata` 正确填充
- ✅ 旧字段 `instance_id`, `container_id` 保留（向后兼容）
- ✅ 现有 API 仍然正常工作

---

## 回归测试

### 确保现有功能不受影响

1. **Docker 终端 WebSocket 连接** - 与迁移前行为一致
2. **Docker 录制列表 API** - 返回相同结果（向后兼容）
3. **WebSSH 会话** - 录制功能正常集成
4. **K8s 节点终端** - 录制功能正常集成

---

## 性能基线

### 关键性能指标（KPI）

| 指标 | 目标 | 验证方法 |
|------|------|----------|
| 录制写入延迟 | <10ms/frame | Go benchmark |
| 清理任务执行时间（10k 录制） | <5 分钟 | 集成测试 |
| 录制回放加载时间 | <1 秒 | 前端性能测试 |
| 并发终端会话数 | >100 | k6 负载测试 |
| 数据库查询响应时间（P99） | <200ms | Prometheus 监控 |

### 性能测试命令

```bash
# 1. 录制写入性能（Go benchmark）
go test -bench=BenchmarkRecordFrame ./internal/services/recording/...

# 2. 清理任务性能
go test -run=TestCleanupPerformance ./tests/integration/recording_cleanup_test.go -timeout 10m

# 3. API 响应时间（ab 工具）
ab -n 1000 -c 10 -H "Authorization: Bearer {token}" \
   http://localhost:12306/api/v1/recordings
```

---

## 故障排查

### 常见问题

1. **录制文件未创建**
   - 检查 `recording.base_path` 目录权限
   - 查看日志：`grep "Failed to write recording" logs/tiga.log`

2. **清理任务未执行**
   - 验证 Cron 表达式：`recording.cleanup_schedule`
   - 查看调度器日志：`grep "RecordingCleanup" logs/tiga.log`

3. **MinIO 连接失败**
   - 验证 MinIO 服务状态：`mc admin info local`
   - 检查网络连接：`telnet localhost 9000`

4. **数据迁移失败**
   - 备份数据库：`pg_dump tiga > backup.sql`
   - 查看迁移错误：`grep "migration failed" logs/tiga.log`
   - 回滚迁移：`psql tiga < backup.sql`

---

## 测试清单

### 功能测试

- [ ] Docker 容器终端录制（向后兼容）
- [ ] WebSSH 主机终端录制（新功能）
- [ ] K8s 节点终端录制（新功能）
- [ ] K8s Pod 终端录制（新功能）
- [ ] 统一录制列表页（所有类型）
- [ ] 录制类型过滤
- [ ] 录制搜索
- [ ] 录制详情页
- [ ] Asciinema 回放
- [ ] 录制下载
- [ ] 录制删除
- [ ] 自动清理任务（过期录制）
- [ ] 自动清理任务（无效录制）
- [ ] 手动触发清理
- [ ] MinIO 存储（可选）

### 性能测试

- [ ] 录制写入延迟 <10ms
- [ ] 清理任务 10k 录制 <5 分钟
- [ ] 回放加载 <1 秒
- [ ] 并发会话 >100
- [ ] API 响应 P99 <200ms

### 安全测试

- [ ] 录制访问需要 RBAC 权限
- [ ] 录制删除需要 RBAC 权限
- [ ] 清理任务仅管理员可触发
- [ ] 审计日志记录所有操作

### 兼容性测试

- [ ] 数据迁移无损
- [ ] 现有 Docker API 向后兼容
- [ ] PostgreSQL 支持
- [ ] MySQL 支持
- [ ] SQLite 支持

---

**快速启动指南完成时间**：2025-10-26
**测试执行者**：待分配
**预计测试周期**：3-5 工作日
