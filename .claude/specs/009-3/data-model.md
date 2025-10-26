# 数据模型：统一终端录制系统

**功能**：009-3 统一终端录制系统
**日期**：2025-10-26
**状态**：设计完成

## 模型概述

本文档定义统一终端录制系统的核心数据模型，基于研究阶段的技术决策（`research.md`）。

## 核心实体

### 1. TerminalRecording（终端录制）

**用途**：统一存储所有终端类型（Docker、WebSSH、K8s）的录制元数据

**Go 结构体**：
```go
// internal/models/terminal_recording.go
package models

import (
    "time"
    "github.com/google/uuid"
    "gorm.io/datatypes"
)

type TerminalRecording struct {
    BaseModel  // ID, CreatedAt, UpdatedAt

    // 会话信息
    SessionID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"session_id"`
    UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
    Username    string    `gorm:"type:varchar(255);not null" json:"username"`

    // 录制类型（支持多终端）
    RecordingType string         `gorm:"type:varchar(50);not null;default:'docker';index" json:"recording_type"` // docker, webssh, k8s
    TypeMetadata  datatypes.JSON `gorm:"type:jsonb" json:"type_metadata"` // 类型特定元数据

    // Docker 专用字段（向后兼容，标记为 Deprecated）
    InstanceID  uuid.UUID `gorm:"type:uuid;index" json:"instance_id,omitempty"`  // @Deprecated: 使用 TypeMetadata
    ContainerID string    `gorm:"type:varchar(255)" json:"container_id,omitempty"` // @Deprecated: 使用 TypeMetadata

    // 时间信息
    StartedAt time.Time  `gorm:"not null;index" json:"started_at"`
    EndedAt   *time.Time `gorm:"index" json:"ended_at,omitempty"`
    Duration  int        `gorm:"default:0" json:"duration"` // 持续时间（秒）

    // 存储信息
    StorageType string `gorm:"type:varchar(50);default:'local'" json:"storage_type"` // local, minio
    StoragePath string `gorm:"type:text;not null" json:"storage_path"`               // 文件路径或对象 key
    FileSize    int64  `gorm:"default:0;index" json:"file_size"`                     // 文件大小（字节）
    Format      string `gorm:"type:varchar(50);default:'asciinema'" json:"format"`   // 录制格式

    // 终端配置
    Rows  int    `gorm:"default:30" json:"rows"`
    Cols  int    `gorm:"default:120" json:"cols"`
    Shell string `gorm:"type:varchar(255)" json:"shell"`

    // 元数据
    ClientIP    string `gorm:"type:varchar(255)" json:"client_ip"`
    Description string `gorm:"type:text" json:"description,omitempty"`
    Tags        string `gorm:"type:text" json:"tags,omitempty"` // 逗号分隔的标签
}

// TableName 指定表名
func (TerminalRecording) TableName() string {
    return "terminal_recordings"
}

// IsCompleted 检查录制是否已完成
func (r *TerminalRecording) IsCompleted() bool {
    return r.EndedAt != nil
}

// IsExpired 检查录制是否已过期（根据保留期限）
func (r *TerminalRecording) IsExpired(retentionDays int) bool {
    if r.EndedAt == nil {
        return false
    }
    cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
    return r.EndedAt.Before(cutoffTime)
}

// IsInvalid 检查录制是否无效（零大小或零时长）
func (r *TerminalRecording) IsInvalid() bool {
    return r.FileSize == 0 || r.Duration == 0
}
```

**TypeMetadata 结构示例**：

```json
// Docker 容器终端
{
  "type": "docker",
  "instance_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "instance_name": "prod-docker-01",
  "container_id": "abc123def456",
  "container_name": "nginx-web",
  "image": "nginx:latest"
}

// WebSSH 主机终端
{
  "type": "webssh",
  "host_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "host_name": "prod-web-server-01",
  "host_ip": "192.168.1.100",
  "ssh_port": 22,
  "ssh_user": "admin"
}

// K8s 节点终端
{
  "type": "k8s_node",
  "cluster_id": "a3bb189e-8bf9-3888-9912-ace4e6543002",
  "cluster_name": "prod-k8s-cluster",
  "node_name": "worker-node-01",
  "namespace": null,
  "pod_name": null
}

// K8s Pod 终端
{
  "type": "k8s_pod",
  "cluster_id": "a3bb189e-8bf9-3888-9912-ace4e6543002",
  "cluster_name": "prod-k8s-cluster",
  "node_name": "worker-node-02",
  "namespace": "default",
  "pod_name": "nginx-deployment-abc123",
  "container_name": "nginx"
}
```

**索引策略**：
```sql
-- 主键
CREATE UNIQUE INDEX idx_terminal_recordings_pkey ON terminal_recordings(id);

-- 会话查询
CREATE UNIQUE INDEX idx_terminal_recordings_session ON terminal_recordings(session_id);

-- 用户查询
CREATE INDEX idx_terminal_recordings_user ON terminal_recordings(user_id);

-- 类型查询
CREATE INDEX idx_terminal_recordings_type ON terminal_recordings(recording_type);

-- 时间查询
CREATE INDEX idx_terminal_recordings_started ON terminal_recordings(started_at);
CREATE INDEX idx_terminal_recordings_ended ON terminal_recordings(ended_at) WHERE ended_at IS NOT NULL;

-- 清理查询优化（复合索引）
CREATE INDEX idx_terminal_recordings_cleanup ON terminal_recordings(ended_at, file_size, duration) WHERE ended_at IS NOT NULL;

-- Docker 向后兼容
CREATE INDEX idx_terminal_recordings_instance ON terminal_recordings(instance_id) WHERE instance_id IS NOT NULL;
```

**字段验证规则**：
- `recording_type` ∈ {"docker", "webssh", "k8s_node", "k8s_pod"}
- `storage_type` ∈ {"local", "minio"}
- `format` ∈ {"asciinema"}（未来可扩展）
- `session_id` 必须唯一
- `started_at` 不能晚于 `ended_at`
- `duration` 必须 ≥ 0
- `file_size` 必须 ≥ 0
- `rows` ∈ [10, 200]
- `cols` ∈ [40, 300]

---

### 2. RecordingStorageConfig（存储配置）

**用途**：录制存储配置（通常存储在 `config.yaml`，不需要数据库表）

**Go 结构体**：
```go
// internal/config/config.go
package config

type RecordingConfig struct {
    // 基础存储配置
    StorageType string `yaml:"storage_type" env:"RECORDING_STORAGE_TYPE" default:"local"`
    BasePath    string `yaml:"base_path" env:"RECORDING_BASE_PATH" default:"./data/recordings"`

    // 清理策略
    RetentionDays    int    `yaml:"retention_days" env:"RECORDING_RETENTION_DAYS" default:"90"`
    CleanupSchedule  string `yaml:"cleanup_schedule" env:"RECORDING_CLEANUP_SCHEDULE" default:"0 4 * * *"`
    CleanupBatchSize int    `yaml:"cleanup_batch_size" env:"RECORDING_CLEANUP_BATCH_SIZE" default:"1000"`

    // 性能配置
    MaxRecordingSize int64 `yaml:"max_recording_size" env:"RECORDING_MAX_SIZE" default:"524288000"` // 500MB

    // MinIO 配置（可选）
    MinIO MinIOConfig `yaml:"minio"`
}

type MinIOConfig struct {
    Endpoint  string `yaml:"endpoint" env:"MINIO_ENDPOINT"`
    Bucket    string `yaml:"bucket" env:"MINIO_BUCKET" default:"terminal-recordings"`
    AccessKey string `yaml:"access_key" env:"MINIO_ACCESS_KEY"`
    SecretKey string `yaml:"secret_key" env:"MINIO_SECRET_KEY"`
    UseSSL    bool   `yaml:"use_ssl" env:"MINIO_USE_SSL" default:"true"`
}
```

**配置示例**（config.yaml）：
```yaml
recording:
  storage_type: local
  base_path: ./data/recordings
  retention_days: 90
  cleanup_schedule: "0 4 * * *"
  cleanup_batch_size: 1000
  max_recording_size: 524288000  # 500MB

  # MinIO 可选配置
  # minio:
  #   endpoint: minio.example.com:9000
  #   bucket: terminal-recordings
  #   access_key: ${MINIO_ACCESS_KEY}
  #   secret_key: ${MINIO_SECRET_KEY}
  #   use_ssl: true
```

---

### 3. RecordingStatistics（录制统计）

**用途**：录制统计信息（通过聚合查询计算，无需单独表）

**Go 结构体**：
```go
// internal/services/recording/types.go
package recording

type RecordingStatistics struct {
    // 总体统计
    TotalCount      int64  `json:"total_count"`
    TotalSize       int64  `json:"total_size"`        // 字节
    TotalSizeHuman  string `json:"total_size_human"`  // "120.5 GB"

    // 按类型分组
    ByType map[string]*TypeStatistics `json:"by_type"`

    // 按用户分组（Top 10）
    TopUsers []UserStatistics `json:"top_users"`

    // 时间范围
    OldestRecording *time.Time `json:"oldest_recording,omitempty"`
    NewestRecording *time.Time `json:"newest_recording,omitempty"`

    // 存储健康
    InvalidCount int64   `json:"invalid_count"` // 零大小或零时长
    OrphanCount  int64   `json:"orphan_count"`  // 文件存在但无数据库记录
    ErrorRate    float64 `json:"error_rate"`    // 无效/总数
}

type TypeStatistics struct {
    RecordingType string `json:"recording_type"`
    Count         int64  `json:"count"`
    TotalSize     int64  `json:"total_size"`
    AvgDuration   int    `json:"avg_duration"` // 秒
}

type UserStatistics struct {
    UserID   uuid.UUID `json:"user_id"`
    Username string    `json:"username"`
    Count    int64     `json:"count"`
}
```

**聚合查询示例**：
```sql
-- 总体统计
SELECT
    COUNT(*) as total_count,
    SUM(file_size) as total_size,
    MIN(started_at) as oldest_recording,
    MAX(started_at) as newest_recording
FROM terminal_recordings;

-- 按类型分组
SELECT
    recording_type,
    COUNT(*) as count,
    SUM(file_size) as total_size,
    AVG(duration) as avg_duration
FROM terminal_recordings
GROUP BY recording_type;

-- 按用户分组（Top 10）
SELECT
    user_id,
    username,
    COUNT(*) as count
FROM terminal_recordings
GROUP BY user_id, username
ORDER BY count DESC
LIMIT 10;

-- 无效录制统计
SELECT COUNT(*) as invalid_count
FROM terminal_recordings
WHERE file_size = 0 OR duration = 0;
```

---

## 数据迁移脚本

### 现有数据迁移（Docker → 统一模型）

**迁移 SQL**（在应用启动时自动执行）：
```sql
-- Step 1: 添加新字段（GORM AutoMigrate 自动执行）
ALTER TABLE terminal_recordings
  ADD COLUMN IF NOT EXISTS recording_type VARCHAR(50) DEFAULT 'docker',
  ADD COLUMN IF NOT EXISTS type_metadata JSONB,
  ADD COLUMN IF NOT EXISTS storage_type VARCHAR(50) DEFAULT 'local',
  ADD COLUMN IF NOT EXISTS tags TEXT;

-- Step 2: 创建索引
CREATE INDEX IF NOT EXISTS idx_terminal_recordings_type
ON terminal_recordings(recording_type);

CREATE INDEX IF NOT EXISTS idx_terminal_recordings_cleanup
ON terminal_recordings(ended_at, file_size, duration)
WHERE ended_at IS NOT NULL;

-- Step 3: 迁移现有数据（仅迁移未设置 recording_type 的记录）
UPDATE terminal_recordings
SET
  recording_type = 'docker',
  type_metadata = jsonb_build_object(
    'type', 'docker',
    'instance_id', instance_id::text,
    'container_id', container_id
  )
WHERE recording_type IS NULL OR recording_type = '';

-- Step 4: 验证迁移
SELECT
  recording_type,
  COUNT(*) as count,
  COUNT(type_metadata) as metadata_count
FROM terminal_recordings
GROUP BY recording_type;
```

**Go 迁移代码**（应用启动时执行）：
```go
// internal/db/migrations.go
package db

func MigrateTerminalRecordings(db *gorm.DB) error {
    // Step 1: AutoMigrate 添加新字段
    if err := db.AutoMigrate(&models.TerminalRecording{}); err != nil {
        return err
    }

    // Step 2: 迁移现有 Docker 录制数据
    var count int64
    db.Model(&models.TerminalRecording{}).
        Where("recording_type IS NULL OR recording_type = ''").
        Count(&count)

    if count > 0 {
        logrus.Infof("Migrating %d existing Docker recordings to unified model", count)

        // 批量迁移
        err := db.Exec(`
            UPDATE terminal_recordings
            SET
              recording_type = 'docker',
              type_metadata = jsonb_build_object(
                'type', 'docker',
                'instance_id', instance_id::text,
                'container_id', container_id
              )
            WHERE recording_type IS NULL OR recording_type = ''
        `).Error

        if err != nil {
            return fmt.Errorf("migration failed: %w", err)
        }

        logrus.Infof("Successfully migrated %d recordings", count)
    }

    return nil
}
```

---

## 实体关系图

```
┌────────────────────────────────┐
│   TerminalRecording            │
├────────────────────────────────┤
│ PK: id (uuid)                  │
│ UK: session_id (uuid)          │
│     user_id (uuid) ────┐       │
│     username           │       │
│                        │       │
│     recording_type     │       │
│     type_metadata      │       │
│                        │       │
│     started_at         │       │
│     ended_at           │       │
│     duration           │       │
│                        │       │
│     storage_type       │       │
│     storage_path       │       │
│     file_size          │       │
│     format             │       │
│                        │       │
│     rows, cols, shell  │       │
│     client_ip          │       │
│     description, tags  │       │
└────────────────────────────────┘
                         │
                         │ FK
                         ▼
                ┌────────────────┐
                │   User         │
                ├────────────────┤
                │ PK: id (uuid)  │
                │     username   │
                └────────────────┘

TypeMetadata (JSONB) 内容示例：
┌──────────────────────────────────────┐
│ Docker:                              │
│   instance_id, container_id,         │
│   container_name, image              │
├──────────────────────────────────────┤
│ WebSSH:                              │
│   host_id, host_name, host_ip,       │
│   ssh_port, ssh_user                 │
├──────────────────────────────────────┤
│ K8s:                                 │
│   cluster_id, node_name,             │
│   namespace, pod_name                │
└──────────────────────────────────────┘
```

---

## 状态转换图

```
┌─────────┐
│ 会话开始  │
└────┬────┘
     │ 创建 TerminalRecording
     │ (started_at = NOW, ended_at = NULL)
     ▼
┌──────────────────┐
│   录制中          │
│ ended_at = NULL  │
│ duration = 0     │
│ file_size = 0    │
└────┬─────────────┘
     │ 终端数据流
     │ recordFrame() → RecordingBuffer
     ▼
┌──────────────────┐     会话正常结束
│   缓冲写入        │ ──────────────────►  ┌──────────────────┐
│ buffer > 100MB?  │                      │   已完成          │
└────┬─────────────┘                      │ ended_at != NULL │
     │ 定期 Flush                         │ duration > 0     │
     │                                    │ file_size > 0    │
     ▼                                    └────┬─────────────┘
┌──────────────────┐                          │ 保留 90 天
│   持久化中        │                          ▼
│ 写入磁盘/MinIO    │                     ┌──────────────────┐
└──────────────────┘                     │   已过期          │
                                         │ ended_at < cutoff │
     会话异常中断                         └────┬─────────────┘
          │                                   │ 清理任务
          ▼                                   ▼
┌──────────────────┐                     ┌──────────────────┐
│   无效录制        │ ───────────────────►│   已删除          │
│ file_size = 0    │  清理任务            │ (数据库 + 文件)   │
│ OR duration = 0  │                     └──────────────────┘
└──────────────────┘
```

---

## 数据完整性约束

### 数据库约束

```sql
-- 主键约束
ALTER TABLE terminal_recordings
  ADD CONSTRAINT pk_terminal_recordings PRIMARY KEY (id);

-- 唯一约束
ALTER TABLE terminal_recordings
  ADD CONSTRAINT uk_terminal_recordings_session UNIQUE (session_id);

-- 非空约束
ALTER TABLE terminal_recordings
  ALTER COLUMN recording_type SET NOT NULL,
  ALTER COLUMN started_at SET NOT NULL,
  ALTER COLUMN storage_path SET NOT NULL;

-- 检查约束
ALTER TABLE terminal_recordings
  ADD CONSTRAINT chk_recording_type CHECK (recording_type IN ('docker', 'webssh', 'k8s_node', 'k8s_pod')),
  ADD CONSTRAINT chk_storage_type CHECK (storage_type IN ('local', 'minio')),
  ADD CONSTRAINT chk_format CHECK (format IN ('asciinema')),
  ADD CONSTRAINT chk_duration CHECK (duration >= 0),
  ADD CONSTRAINT chk_file_size CHECK (file_size >= 0),
  ADD CONSTRAINT chk_terminal_size CHECK (rows BETWEEN 10 AND 200 AND cols BETWEEN 40 AND 300),
  ADD CONSTRAINT chk_time_order CHECK (ended_at IS NULL OR ended_at >= started_at);

-- 外键约束（如果 users 表存在）
ALTER TABLE terminal_recordings
  ADD CONSTRAINT fk_terminal_recordings_user
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
```

### 应用层验证

```go
// internal/services/recording/validator.go
package recording

func ValidateRecording(rec *models.TerminalRecording) error {
    // 录制类型验证
    validTypes := map[string]bool{
        "docker": true, "webssh": true,
        "k8s_node": true, "k8s_pod": true,
    }
    if !validTypes[rec.RecordingType] {
        return fmt.Errorf("invalid recording_type: %s", rec.RecordingType)
    }

    // 时间顺序验证
    if rec.EndedAt != nil && rec.EndedAt.Before(rec.StartedAt) {
        return errors.New("ended_at cannot be before started_at")
    }

    // 终端尺寸验证
    if rec.Rows < 10 || rec.Rows > 200 {
        return fmt.Errorf("rows must be between 10-200, got %d", rec.Rows)
    }
    if rec.Cols < 40 || rec.Cols > 300 {
        return fmt.Errorf("cols must be between 40-300, got %d", rec.Cols)
    }

    // TypeMetadata 验证
    if rec.RecordingType == "docker" {
        // 确保 TypeMetadata 包含 instance_id 和 container_id
        var metadata map[string]interface{}
        if err := json.Unmarshal(rec.TypeMetadata, &metadata); err != nil {
            return fmt.Errorf("invalid type_metadata JSON: %w", err)
        }
        if _, ok := metadata["instance_id"]; !ok {
            return errors.New("type_metadata missing instance_id for docker recording")
        }
    }

    return nil
}
```

---

## 数据访问模式

### 常见查询场景

```go
// 1. 按用户查询录制列表（分页）
func (r *RecordingRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*models.TerminalRecording, int64, error)

// 2. 按录制类型查询
func (r *RecordingRepository) ListByType(ctx context.Context, recordingType string, page, limit int) ([]*models.TerminalRecording, int64, error)

// 3. 查询过期录制
func (r *RecordingRepository) FindExpiredRecordings(ctx context.Context, retentionDays int, limit int) ([]*models.TerminalRecording, error)

// 4. 查询无效录制（零大小/零时长）
func (r *RecordingRepository) FindInvalidRecordings(ctx context.Context, limit int) ([]*models.TerminalRecording, error)

// 5. 按时间范围查询
func (r *RecordingRepository) ListByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*models.TerminalRecording, error)

// 6. 全文搜索（用户名、描述、标签）
func (r *RecordingRepository) Search(ctx context.Context, query string, page, limit int) ([]*models.TerminalRecording, int64, error)

// 7. 统计信息查询
func (r *RecordingRepository) GetStatistics(ctx context.Context) (*RecordingStatistics, error)
```

---

## 性能考虑

### 查询优化

1. **分页查询**：使用 LIMIT + OFFSET（或 cursor-based）
2. **索引覆盖**：复合索引覆盖常见查询
3. **JSONB 索引**：
   ```sql
   -- 为 TypeMetadata 创建 GIN 索引（支持 JSONB 查询）
   CREATE INDEX idx_terminal_recordings_metadata
   ON terminal_recordings USING GIN (type_metadata);

   -- 查询示例
   SELECT * FROM terminal_recordings
   WHERE type_metadata @> '{"container_id": "abc123"}';
   ```

### 写入优化

1. **批量插入**：使用 `db.CreateInBatches()`
2. **异步写入**：录制文件异步落盘（`finalizeRecording`）
3. **连接池**：GORM 默认连接池配置

### 清理优化

1. **批量删除**：每批 1000 条
2. **并行文件删除**：10 worker goroutines
3. **增量清理**：每次最多 5000 条，避免长时间锁表

---

**数据模型完成时间**：2025-10-26
**审核状态**：待契约设计和测试验证
