# 研究文档：统一终端录制系统

**功能**：009-3 统一终端录制系统
**日期**：2025-10-26
**状态**：已完成

## 研究概述

本文档记录了统一终端录制系统实施前的技术研究，解决设计阶段的未知项和技术选型问题。

## 1. MinIO 存储抽象必要性评估

### 决策：保留 MinIO 可选支持，但不强制抽象

**研究发现**：
- 用户在澄清阶段明确选择 MinIO 为"可选功能"（非强制）
- 现有代码库已集成 MinIO SDK v7.0.95，用于数据库管理子系统
- 录制文件特点：顺序写入、一次写入多次读取、不需要事务性

**实施策略**：
- **Phase 1**：仅支持本地文件系统存储（覆盖 80% 用例）
- **Phase 2**（可选）：通过配置开关添加 MinIO 支持
- **接口设计**：使用简单的 `io.Writer` 接口而非重量级存储抽象

**理由**：
- 遵循 YAGNI 原则（You Aren't Gonna Need It）
- 本地文件系统满足大多数部署场景
- MinIO 支持作为配置选项，不增加核心复杂度

**考虑的替代方案**：
- ❌ 完整存储抽象层（StorageBackend 接口）- 过度设计
- ❌ 仅支持 MinIO - 增加部署复杂度
- ✅ 本地优先 + 可选 MinIO - 平衡灵活性和简洁性

---

## 2. 数据迁移策略

### 决策：无损迁移 + 双写适配

**研究发现**：
- 现有 `models.TerminalRecording` 模型 Docker 专用字段：
  - `InstanceID uuid.UUID` - Docker 实例 ID
  - `ContainerID string` - Docker 容器 ID
- 现有数据库表：`terminal_recordings`（已有约 PostgreSQL 约束）
- GORM AutoMigrate 支持添加列，但不支持重命名或删除

**迁移方案**：
```sql
-- Step 1: 添加新字段（GORM AutoMigrate 自动执行）
ALTER TABLE terminal_recordings
  ADD COLUMN recording_type VARCHAR(50) DEFAULT 'docker',
  ADD COLUMN type_metadata JSONB;

-- Step 2: 数据迁移（应用启动时执行）
UPDATE terminal_recordings
SET recording_type = 'docker',
    type_metadata = json_build_object(
      'instance_id', instance_id::text,
      'container_id', container_id
    )
WHERE recording_type IS NULL;

-- Step 3: 保留旧字段（向后兼容）
-- InstanceID 和 ContainerID 字段标记为 Deprecated，但不删除
```

**Go 模型扩展**：
```go
type TerminalRecording struct {
    BaseModel
    SessionID   uuid.UUID

    // 统一字段
    RecordingType string         `gorm:"type:varchar(50);not null;default:'docker';index" json:"recording_type"`
    TypeMetadata  datatypes.JSON `gorm:"type:jsonb" json:"type_metadata"`

    // Docker 专用字段（Deprecated，向后兼容）
    InstanceID  uuid.UUID `gorm:"type:uuid;index" json:"instance_id,omitempty"`  // @Deprecated: 使用 TypeMetadata
    ContainerID string    `gorm:"type:varchar(255)" json:"container_id,omitempty"` // @Deprecated

    // 通用字段
    UserID      uuid.UUID
    Username    string
    StartedAt   time.Time
    EndedAt     *time.Time
    // ... 其余字段保持不变
}
```

**TypeMetadata 结构示例**：
```json
// Docker 类型
{
  "instance_id": "uuid-string",
  "container_id": "container-sha256"
}

// WebSSH 类型
{
  "host_id": "uuid-string",
  "ssh_session_id": "session-uuid"
}

// K8s 类型
{
  "cluster_id": "uuid-string",
  "node_name": "worker-node-01",
  "namespace": "default",
  "pod_name": "optional-pod-name"
}
```

**理由**：
- 零停机迁移：新旧字段共存
- 向后兼容：现有 Docker API 继续工作
- 扩展性：TypeMetadata JSONB 支持任意终端类型

**考虑的替代方案**：
- ❌ 删除旧字段，强制迁移 - 破坏现有 API
- ❌ 多态表（docker_recordings, webssh_recordings）- 查询复杂
- ✅ 单表 + JSONB - PostgreSQL 原生支持，性能优秀

---

## 3. Asciinema v2 格式最佳实践

### 决策：完全遵循 Asciinema v2 规范

**研究发现**：
- 现有 Docker 终端已实现 Asciinema v2 格式（`terminal_handler.go:248-261`）
- 格式规范：https://github.com/asciinema/asciinema/blob/develop/doc/asciicast-v2.md
- 文件扩展名：`.cast`

**格式结构**：
```
第 1 行：JSON 头部（Header）
{
  "version": 2,
  "width": 120,
  "height": 30,
  "timestamp": 1698765432,
  "env": {"SHELL": "/bin/bash", "TERM": "xterm-256color"}
}

第 2+ 行：事件帧（每行一个 JSON 数组）
[0.123456, "o", "$ ls\r\n"]        # 输出事件
[1.234567, "i", "ls\r\n"]          # 输入事件
[2.345678, "o", "file1.txt\r\n"]  # 输出事件
```

**实施要点**：
1. **时间戳精度**：float64 秒（6 位小数）
2. **数据转义**：JSON 字符串转义（`\r\n`, `\t`, `"`）
3. **缓冲写入**：使用 `bufio.Writer` 批量写入，减少系统调用
4. **并发安全**：`sync.Mutex` 保护录制缓冲区

**性能优化**：
```go
// 现有实现（terminal_handler.go:640-666）
func (h *TerminalHandler) recordFrame(session *TerminalSession, frameType string, data []byte) {
    session.RecordingMutex.Lock()
    defer session.RecordingMutex.Unlock()

    timestamp := time.Since(session.StartTime).Seconds()
    frame := [3]interface{}{timestamp, frameType, string(data)}
    frameJSON, _ := json.Marshal(frame)

    session.RecordingBuffer.Write(frameJSON)
    session.RecordingBuffer.WriteString("\n")
}
```

**改进建议**：
- 添加错误处理（当前忽略 `json.Marshal` 错误）
- 定期 Flush 缓冲区（避免内存占用过大）
- 添加文件大小限制（500MB）

**理由**：
- 标准格式，兼容 asciinema CLI 和 Web Player
- 现有实现经过验证，无需重新设计
- 社区生态支持（播放器库、转换工具）

---

## 4. 清理任务性能优化

### 决策：批处理 + 索引优化 + 并行删除

**性能目标**：10,000 条录制 <5 分钟清理

**研究发现**：
- 现有清理服务批量大小：1000 条（`recording_cleanup_service.go:35`）
- 清理条件：`FileSize == 0 OR Duration == 0`
- 当前问题：逐条删除，无并行处理

**优化方案**：

1. **数据库索引优化**：
```sql
-- 复合索引加速过期录制查询
CREATE INDEX idx_recordings_cleanup
ON terminal_recordings(ended_at, file_size, duration)
WHERE ended_at IS NOT NULL;

-- 部分索引（仅索引完成的录制）
CREATE INDEX idx_recordings_retention
ON terminal_recordings(ended_at)
WHERE ended_at IS NOT NULL AND ended_at < NOW() - INTERVAL '90 days';
```

2. **批量删除**：
```go
// 替代逐条删除
// 旧代码：
for _, id := range invalidIDs {
    repo.Delete(ctx, id)  // N 次数据库调用
}

// 新代码：
repo.BulkDelete(ctx, invalidIDs)  // 1 次数据库调用
// DELETE FROM terminal_recordings WHERE id = ANY($1)
```

3. **并行文件删除**：
```go
// 使用 worker pool 并行删除文件
const workers = 10
sem := make(chan struct{}, workers)
var wg sync.WaitGroup

for _, recording := range toDelete {
    wg.Add(1)
    sem <- struct{}{}  // 限流

    go func(r *TerminalRecording) {
        defer wg.Done()
        os.Remove(r.StoragePath)
        <-sem
    }(recording)
}
wg.Wait()
```

4. **增量清理**：
```go
// 避免一次性加载全部过期录制
// 每次清理最多处理 5000 条
const maxCleanupPerRun = 5000

// 分批执行（每天凌晨 4:00 运行）
for cleaned := 0; cleaned < maxCleanupPerRun; {
    batch := repo.FindExpiredRecordings(ctx, limit=1000)
    if len(batch) == 0 {
        break
    }
    cleaned += deleteBatch(batch)
}
```

**性能预估**：
- 数据库批量删除：10k 条 ~5 秒
- 文件并行删除（10 workers）：10k 文件 ~2 分钟（假设平均 10MB/文件）
- 总计：<3 分钟（满足 <5 分钟目标）

**理由**：
- 批量操作减少数据库往返次数
- 并行删除利用 I/O 并发能力
- 增量清理避免长时间锁表

**考虑的替代方案**：
- ❌ 单线程删除 - 慢（预估 >10 分钟）
- ❌ 软删除（标记删除）- 不释放存储空间
- ✅ 批量 + 并行 - 性能最优

---

## 5. 并发录制写入性能

### 决策：内存缓冲 + 异步落盘

**性能目标**：<10ms/frame 写入延迟

**研究发现**：
- 现有实现：同步写入 `bytes.Buffer`（内存缓冲）
- 落盘时机：会话结束时（`finalizeRecording`）
- 单帧大小：平均 50-200 字节（JSON 编码后）

**性能分析**：
```go
// 当前实现（内存缓冲，无 I/O）
func recordFrame(session *TerminalSession, frameType string, data []byte) {
    session.RecordingMutex.Lock()  // ~0.1μs
    defer session.RecordingMutex.Unlock()

    timestamp := time.Since(session.StartTime).Seconds()  // ~0.5μs
    frame := [3]interface{}{timestamp, frameType, string(data)}
    frameJSON, _ := json.Marshal(frame)  // ~5-10μs (小对象)

    session.RecordingBuffer.Write(frameJSON)  // ~1μs (内存写入)
    session.RecordingBuffer.WriteString("\n")  // ~0.5μs
}
// 总计：~7-12μs，满足 <10ms 目标
```

**优化建议**（已足够快，仅作预防）：
1. **预分配缓冲区**：
```go
// 避免动态扩容
session.RecordingBuffer = bytes.NewBuffer(make([]byte, 0, 1*1024*1024))  // 1MB 初始容量
```

2. **定期 Flush**（避免内存溢出）：
```go
// 每 100MB 或 10 分钟 flush 到磁盘
if session.RecordingBuffer.Len() > 100*1024*1024 || time.Since(session.LastFlush) > 10*time.Minute {
    appendToFile(session.RecordingPath, session.RecordingBuffer.Bytes())
    session.RecordingBuffer.Reset()
}
```

3. **无锁优化**（如果需要极致性能）：
```go
// 使用 channel 替代 Mutex
type RecordingChannel chan RecordingFrame

go func() {
    for frame := range session.RecordingChan {
        writeFrameToBuffer(frame)
    }
}()
```

**压力测试计划**：
- 场景：100 并发终端会话，每秒 10 帧
- 目标：P99 延迟 <10ms
- 工具：Go benchmark + pprof

**理由**：
- 内存缓冲避免频繁磁盘 I/O
- 会话结束时一次性落盘，减少系统调用
- 当前实现已满足性能目标

---

## 6. 录制文件组织结构

### 决策：日期分区 + UUID 文件名

**研究发现**：
- 现有 Docker 实现：`{recordingDir}/{YYYY-MM-DD}/{recordingID}.cast`
- WebSSH 实现：无日期分区，平铺在 `./data/recordings/`

**统一结构**：
```
{BASE_RECORDING_DIR}/
├── 2025-10-26/
│   ├── 3fa85f64-5717-4562-b3fc-2c963f66afa6.cast  # Docker 容器录制
│   ├── 7c9e6679-7425-40de-944b-e07fc1f90ae7.cast  # WebSSH 录制
│   └── a3bb189e-8bf9-3888-9912-ace4e6543002.cast  # K8s 节点录制
├── 2025-10-27/
│   └── ...
└── 2025-10-28/
    └── ...
```

**目录组织规则**：
1. **第一层**：日期（`YYYY-MM-DD`）- 便于按日期清理和归档
2. **第二层**：录制文件（`{recordingID}.cast`）- UUID 避免冲突

**优势**：
- 清理效率：按日期目录删除，避免扫描全部文件
- 归档友好：可按月打包 `tar -czf 2025-10.tar.gz 2025-10-*/`
- 可读性：目录名即时间范围

**代码实现**：
```go
func (s *RecordingStorageService) GetRecordingPath(recordingID uuid.UUID, startedAt time.Time) string {
    dateDir := startedAt.Format("2006-01-02")
    filename := fmt.Sprintf("%s.cast", recordingID.String())
    return filepath.Join(s.baseDir, dateDir, filename)
}
```

**理由**：
- 遵循现有 Docker 实现（已验证）
- 文件系统友好（避免单目录文件过多）
- 清理任务高效（删除整个日期目录）

---

## 7. 存储路径配置统一

### 决策：config.yaml + 环境变量回退

**研究发现**：
- 现有配置系统：`internal/config/config.go`（YAML 优先 + 环境变量回退）
- Docker 录制路径：`TERMINAL_RECORDING_DIR` 环境变量
- WebSSH 录制路径：构造函数参数（硬编码）

**统一配置结构**：
```yaml
# config.yaml
recording:
  # 基础存储配置
  storage_type: "local"  # local | minio
  base_path: "./data/recordings"  # 本地文件系统路径

  # 清理策略
  retention_days: 90          # 保留期限（天）
  cleanup_schedule: "0 4 * * *"  # Cron 表达式（每天凌晨 4:00）
  cleanup_batch_size: 1000    # 批处理大小

  # MinIO 配置（可选，当 storage_type=minio 时）
  minio:
    endpoint: "minio.example.com:9000"
    bucket: "terminal-recordings"
    access_key: "${MINIO_ACCESS_KEY}"  # 支持环境变量引用
    secret_key: "${MINIO_SECRET_KEY}"
    use_ssl: true
```

**Go 配置结构**：
```go
// internal/config/config.go
type Config struct {
    // ... 现有字段
    Recording RecordingConfig `yaml:"recording"`
}

type RecordingConfig struct {
    StorageType       string      `yaml:"storage_type" env:"RECORDING_STORAGE_TYPE" default:"local"`
    BasePath          string      `yaml:"base_path" env:"RECORDING_BASE_PATH" default:"./data/recordings"`
    RetentionDays     int         `yaml:"retention_days" env:"RECORDING_RETENTION_DAYS" default:"90"`
    CleanupSchedule   string      `yaml:"cleanup_schedule" env:"RECORDING_CLEANUP_SCHEDULE" default:"0 4 * * *"`
    CleanupBatchSize  int         `yaml:"cleanup_batch_size" env:"RECORDING_CLEANUP_BATCH_SIZE" default:"1000"`
    MinIO             MinIOConfig `yaml:"minio"`
}

type MinIOConfig struct {
    Endpoint  string `yaml:"endpoint" env:"MINIO_ENDPOINT"`
    Bucket    string `yaml:"bucket" env:"MINIO_BUCKET" default:"terminal-recordings"`
    AccessKey string `yaml:"access_key" env:"MINIO_ACCESS_KEY"`
    SecretKey string `yaml:"secret_key" env:"MINIO_SECRET_KEY"`
    UseSSL    bool   `yaml:"use_ssl" env:"MINIO_USE_SSL" default:"true"`
}
```

**环境变量优先级**（兼容现有系统）：
```bash
# 新配置系统
RECORDING_BASE_PATH=/data/recordings
RECORDING_RETENTION_DAYS=60

# 向后兼容（Deprecated）
TERMINAL_RECORDING_DIR=/data/recordings  # 映射到 RECORDING_BASE_PATH
```

**理由**：
- 统一配置入口，避免分散
- 遵循现有配置模式（Phase 4 架构改进）
- 支持环境变量覆盖（容器化部署友好）

---

## 8. 清理策略实施细节

### 决策：基于 EndedAt 的时间窗口清理

**清理触发条件**（优先级顺序）：
1. **过期录制**：`ended_at < NOW() - retention_days`
2. **无效录制**：`file_size = 0 OR duration = 0`（即使未过期也删除）
3. **孤儿录制**：文件存在但数据库记录不存在（反之亦然）

**Cron 任务实现**：
```go
// internal/services/recording/cleanup_service.go
func (s *CleanupService) Run(ctx context.Context) error {
    logrus.Info("[RecordingCleanup] Starting scheduled cleanup")

    // Step 1: 清理无效录制（零大小/零时长）
    invalidCount := s.cleanupInvalidRecordings(ctx)

    // Step 2: 清理过期录制
    expiredCount := s.cleanupExpiredRecordings(ctx)

    // Step 3: 清理孤儿文件
    orphanCount := s.cleanupOrphanFiles(ctx)

    // Step 4: 记录指标
    s.recordMetrics(invalidCount, expiredCount, orphanCount)

    logrus.Infof("[RecordingCleanup] Completed: invalid=%d, expired=%d, orphan=%d",
        invalidCount, expiredCount, orphanCount)
    return nil
}
```

**清理查询优化**：
```go
// 使用索引优化的查询
func (r *RecordingRepository) FindExpiredRecordings(ctx context.Context, retentionDays int, limit int) ([]*models.TerminalRecording, error) {
    var recordings []*models.TerminalRecording

    cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

    err := r.db.WithContext(ctx).
        Where("ended_at IS NOT NULL AND ended_at < ?", cutoffTime).
        Order("ended_at ASC").  // 先删除最旧的
        Limit(limit).
        Find(&recordings).Error

    return recordings, err
}
```

**孤儿文件清理**（双向同步）：
```go
// 清理数据库有记录但文件不存在的情况
func (s *CleanupService) cleanupMissingFiles(ctx context.Context) int {
    recordings, _ := s.repo.FindAll(ctx)
    cleaned := 0

    for _, rec := range recordings {
        if _, err := os.Stat(rec.StoragePath); os.IsNotExist(err) {
            s.repo.Delete(ctx, rec.ID)
            cleaned++
        }
    }
    return cleaned
}

// 清理文件存在但数据库无记录的情况
func (s *CleanupService) cleanupOrphanFiles(ctx context.Context) int {
    // 遍历录制目录
    filepath.Walk(s.config.BasePath, func(path string, info os.FileInfo, err error) error {
        if !strings.HasSuffix(path, ".cast") {
            return nil
        }

        // 从文件名提取 UUID
        recordingID := extractUUIDFromPath(path)

        // 检查数据库是否存在
        if _, err := s.repo.GetByID(ctx, recordingID); err != nil {
            os.Remove(path)  // 删除孤儿文件
        }
        return nil
    })
}
```

---

## 研究结论

### 已解决的未知项
1. ✅ MinIO 存储抽象 - 本地优先，可选 MinIO
2. ✅ 数据迁移策略 - 无损迁移，向后兼容
3. ✅ Asciinema 格式 - 完全遵循 v2 规范
4. ✅ 清理任务性能 - 批量 + 并行，<3 分钟
5. ✅ 并发写入性能 - 内存缓冲，<10μs/frame
6. ✅ 文件组织结构 - 日期分区 + UUID
7. ✅ 配置统一 - config.yaml + 环境变量

### 关键技术选择
- **数据模型**：单表 + JSONB（`recording_type` + `type_metadata`）
- **存储策略**：本地文件系统（Phase 1），MinIO 可选（Phase 2）
- **清理机制**：Cron 任务（凌晨 4:00）+ 批量删除 + 并行文件操作
- **录制格式**：Asciinema v2（`.cast` 文件）
- **配置管理**：统一 `RecordingConfig` 结构

### 风险和缓解措施
| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 数据迁移失败 | 现有录制丢失 | 迁移前备份 + 回滚脚本 |
| 清理任务删除活跃录制 | 数据丢失 | 检查 `ended_at IS NOT NULL` |
| 并发写入冲突 | 录制损坏 | `sync.Mutex` + 事务性写入 |
| MinIO 连接失败 | 录制功能不可用 | 回退到本地存储 + 告警 |

### 下一步行动
阶段 1 设计阶段将基于这些研究结论生成：
- `data-model.md` - 统一 TerminalRecording 模型定义
- `contracts/` - API 契约（录制 CRUD、回放、清理）
- `quickstart.md` - 测试场景和验证步骤

---

**研究完成时间**：2025-10-26
**审核状态**：待设计阶段验证
