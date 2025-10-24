# 统一终端录制方案

## 背景

当前系统中有多个子系统涉及终端功能，存在录制实现重复的问题：

1. **Docker 容器终端** - 已实现录制（`models.TerminalRecording`）
2. **主机管理 WebSSH** - 已实现录制器（`services/webssh/recorder.go`）
3. **K8s 节点终端** - 待实现录制
4. **K8s Pod 终端** - 待实现录制

**问题**：
- 代码重复，违反 DRY 原则
- 数据模型不统一（Docker 专用字段如 `ContainerID`）
- 录制器实现分散
- 无法统一管理和查询所有终端录制

## 设计目标

1. **通用性**：支持所有终端类型（Docker、WebSSH、K8s Node、K8s Pod）
2. **可复用**：核心录制逻辑只实现一次
3. **可扩展**：易于添加新的终端类型
4. **统一管理**：统一的 API 和存储
5. **向后兼容**：不破坏现有功能

## 架构设计

### 1. 统一数据模型

#### TerminalRecording (重构)

```go
package models

type TerminalType string

const (
    TerminalTypeDockerContainer TerminalType = "docker_container"
    TerminalTypeWebSSH          TerminalType = "webssh"
    TerminalTypeK8sNode         TerminalType = "k8s_node"
    TerminalTypeK8sPod          TerminalType = "k8s_pod"
)

// TerminalRecording 统一的终端录制模型
type TerminalRecording struct {
    BaseModel

    // Session 信息
    SessionID   uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex" json:"session_id"`
    Type        TerminalType `gorm:"type:varchar(50);not null;index" json:"type"`

    // 用户信息
    UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
    Username    string    `gorm:"type:varchar(255);not null" json:"username"`

    // 时间信息
    StartedAt   time.Time  `gorm:"not null;index" json:"started_at"`
    EndedAt     *time.Time `json:"ended_at,omitempty"`
    Duration    int        `gorm:"default:0" json:"duration"` // 秒

    // 存储信息
    StoragePath string `gorm:"type:text;not null" json:"storage_path"`
    FileSize    int64  `gorm:"default:0" json:"file_size"`
    Format      string `gorm:"type:varchar(50);default:'asciinema'" json:"format"`

    // 终端配置
    Rows        int    `gorm:"default:30" json:"rows"`
    Cols        int    `gorm:"default:120" json:"cols"`
    Shell       string `gorm:"type:varchar(255)" json:"shell,omitempty"`

    // 客户端信息
    ClientIP    string `gorm:"type:varchar(255)" json:"client_ip"`
    UserAgent   string `gorm:"type:text" json:"user_agent,omitempty"`

    // 元数据 (JSON 字段，存储特定类型的信息)
    Metadata    JSONB  `gorm:"type:jsonb" json:"metadata"`

    // 描述
    Description string `gorm:"type:text" json:"description,omitempty"`
}

// Metadata 结构示例：
//
// Docker Container:
// {
//   "instance_id": "uuid",
//   "container_id": "sha256:...",
//   "container_name": "my-container",
//   "image": "nginx:latest"
// }
//
// WebSSH:
// {
//   "host_id": "uuid",
//   "hostname": "server-01",
//   "host_ip": "192.168.1.100",
//   "ssh_user": "root"
// }
//
// K8s Node:
// {
//   "cluster_id": "uuid",
//   "cluster_name": "prod-cluster",
//   "node_name": "node-01"
// }
//
// K8s Pod:
// {
//   "cluster_id": "uuid",
//   "cluster_name": "prod-cluster",
//   "namespace": "default",
//   "pod_name": "nginx-pod",
//   "container_name": "nginx"
// }
```

### 2. 通用录制器接口

```go
package terminal

// Recorder 终端录制器接口
type Recorder interface {
    // RecordOutput 记录终端输出
    RecordOutput(data []byte) error

    // RecordInput 记录终端输入
    RecordInput(data []byte) error

    // Resize 更新终端尺寸
    Resize(cols, rows int)

    // Close 关闭录制器
    Close() error

    // GetMetadata 获取录制元数据
    GetMetadata() RecordingMetadata
}

type RecordingMetadata struct {
    FilePath     string
    BytesWritten int64
    Duration     time.Duration
}

// AsciinemaRecorder Asciinema v2 格式录制器实现
type AsciinemaRecorder struct {
    sessionID    string
    file         *os.File
    writer       *bufio.Writer
    startTime    time.Time
    cols         int
    rows         int
    mu           sync.Mutex
    closed       bool
    bytesWritten int64
}

// 实现 Recorder 接口
func (r *AsciinemaRecorder) RecordOutput(data []byte) error { ... }
func (r *AsciinemaRecorder) RecordInput(data []byte) error { ... }
func (r *AsciinemaRecorder) Resize(cols, rows int) { ... }
func (r *AsciinemaRecorder) Close() error { ... }
func (r *AsciinemaRecorder) GetMetadata() RecordingMetadata { ... }
```

### 3. 统一录制服务

```go
package terminal

// RecordingService 统一的录制服务
type RecordingService struct {
    repo        repository.TerminalRecordingRepository
    storage     storage.RecordingStorage
    config      *config.TerminalRecordingConfig
}

// StartRecording 开始录制会话
func (s *RecordingService) StartRecording(ctx context.Context, req *StartRecordingRequest) (*Recorder, error) {
    // 1. 创建数据库记录
    recording := &models.TerminalRecording{
        SessionID:   req.SessionID,
        Type:        req.Type,
        UserID:      req.UserID,
        Username:    req.Username,
        StartedAt:   time.Now(),
        Rows:        req.Rows,
        Cols:        req.Cols,
        ClientIP:    req.ClientIP,
        Metadata:    req.Metadata,
    }

    if err := s.repo.Create(ctx, recording); err != nil {
        return nil, err
    }

    // 2. 创建录制器
    recorder, err := NewAsciinemaRecorder(req.SessionID.String(), req.Cols, req.Rows, s.config.StorageDir)
    if err != nil {
        return nil, err
    }

    return recorder, nil
}

// FinishRecording 完成录制
func (s *RecordingService) FinishRecording(ctx context.Context, sessionID uuid.UUID, recorder Recorder) error {
    metadata := recorder.GetMetadata()

    now := time.Now()
    updates := map[string]interface{}{
        "ended_at":   &now,
        "duration":   int(metadata.Duration.Seconds()),
        "file_size":  metadata.BytesWritten,
        "storage_path": metadata.FilePath,
    }

    return s.repo.UpdateBySessionID(ctx, sessionID, updates)
}

// GetRecording 获取录制记录
func (s *RecordingService) GetRecording(ctx context.Context, id uuid.UUID) (*models.TerminalRecording, error) {
    return s.repo.GetByID(ctx, id)
}

// ListRecordings 列出录制记录
func (s *RecordingService) ListRecordings(ctx context.Context, filter *RecordingFilter) ([]*models.TerminalRecording, int64, error) {
    return s.repo.List(ctx, filter)
}

// DeleteRecording 删除录制（软删除 + 文件清理）
func (s *RecordingService) DeleteRecording(ctx context.Context, id uuid.UUID) error {
    recording, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }

    // 删除文件
    if err := s.storage.Delete(recording.StoragePath); err != nil {
        logrus.Warnf("Failed to delete recording file: %v", err)
    }

    // 软删除数据库记录
    return s.repo.Delete(ctx, id)
}

// GetRecordingContent 获取录制内容（用于播放）
func (s *RecordingService) GetRecordingContent(ctx context.Context, id uuid.UUID) ([]byte, error) {
    recording, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    return s.storage.Read(recording.StoragePath)
}
```

### 4. 存储抽象

```go
package storage

// RecordingStorage 录制文件存储接口
type RecordingStorage interface {
    // Write 写入录制文件
    Write(path string, data []byte) error

    // Read 读取录制文件
    Read(path string) ([]byte, error)

    // Delete 删除录制文件
    Delete(path string) error

    // Exists 检查文件是否存在
    Exists(path string) bool
}

// LocalStorage 本地文件系统存储
type LocalStorage struct {
    baseDir string
}

// MinIOStorage MinIO 对象存储
type MinIOStorage struct {
    client *minio.Client
    bucket string
}
```

### 5. Repository 接口

```go
package repository

type TerminalRecordingRepository interface {
    Create(ctx context.Context, recording *models.TerminalRecording) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.TerminalRecording, error)
    GetBySessionID(ctx context.Context, sessionID uuid.UUID) (*models.TerminalRecording, error)
    UpdateBySessionID(ctx context.Context, sessionID uuid.UUID, updates map[string]interface{}) error
    List(ctx context.Context, filter *RecordingFilter) ([]*models.TerminalRecording, int64, error)
    Delete(ctx context.Context, id uuid.UUID) error
}

type RecordingFilter struct {
    UserID     *uuid.UUID
    Type       *models.TerminalType
    StartDate  *time.Time
    EndDate    *time.Time
    Page       int
    PageSize   int
}
```

## 实施计划

### Phase 1: 核心基础设施 (Week 1)

**任务**：
- [ ] T001: 创建统一数据模型 `models.TerminalRecording`
- [ ] T002: 实现通用录制器接口 `pkg/terminal/recorder.go`
- [ ] T003: 迁移 WebSSH 录制器到通用实现
- [ ] T004: 实现存储接口（Local + MinIO）
- [ ] T005: 实现 Repository 层
- [ ] T006: 实现统一录制服务
- [ ] T007: 数据库迁移（添加新表，迁移 Docker 录制数据）

**交付物**：
- 通用的终端录制框架
- 完整的单元测试

### Phase 2: Docker 集成 (Week 1)

**任务**：
- [ ] T008: 重构 Docker 终端 handler 使用统一服务
- [ ] T009: 更新 Docker 终端录制 API
- [ ] T010: 数据迁移脚本（从旧模型到新模型）
- [ ] T011: 集成测试

**交付物**：
- Docker 终端录制使用统一框架
- 向后兼容

### Phase 3: WebSSH 集成 (Week 1)

**任务**：
- [ ] T012: 重构 WebSSH handler 使用统一服务
- [ ] T013: 更新 WebSSH 录制 API
- [ ] T014: 添加 WebSSH 录制元数据
- [ ] T015: 集成测试

**交付物**：
- WebSSH 终端录制使用统一框架

### Phase 4: K8s 集成 (Week 1)

**任务**：
- [ ] T016: 实现 K8s 节点终端录制
- [ ] T017: 实现 K8s Pod 终端录制
- [ ] T018: 添加 K8s 录制元数据
- [ ] T019: 集成测试

**交付物**：
- K8s 终端录制功能

### Phase 5: 统一 API 和清理 (Week 1)

**任务**：
- [ ] T020: 实现统一的录制查询 API
- [ ] T021: 实现统一的录制播放 API
- [ ] T022: 实现统一的录制删除 API
- [ ] T023: 前端适配（统一录制列表页面）
- [ ] T024: 清理旧代码
- [ ] T025: 文档更新

**交付物**：
- 统一的录制管理界面
- 完整的 API 文档

## 配置

```yaml
# config.yaml
terminal_recording:
  enabled: true
  storage:
    type: local  # local 或 minio
    local:
      base_dir: /var/tiga/recordings
    minio:
      endpoint: minio:9000
      bucket: terminal-recordings
      access_key: ${MINIO_ACCESS_KEY}
      secret_key: ${MINIO_SECRET_KEY}
  retention:
    days: 90  # 保留天数
    cleanup_cron: "0 2 * * *"  # 每天 2AM 清理
  format: asciinema  # 录制格式
  max_size_mb: 500  # 单个录制最大大小
```

## API 设计

### 统一的录制 API

```
GET    /api/v1/terminal-recordings           - 列出所有录制
GET    /api/v1/terminal-recordings/:id       - 获取录制详情
GET    /api/v1/terminal-recordings/:id/playback - 获取录制内容（播放）
DELETE /api/v1/terminal-recordings/:id       - 删除录制

# 按类型过滤
GET    /api/v1/terminal-recordings?type=docker_container
GET    /api/v1/terminal-recordings?type=webssh
GET    /api/v1/terminal-recordings?type=k8s_pod

# 按用户过滤
GET    /api/v1/terminal-recordings?user_id=xxx

# 按时间范围过滤
GET    /api/v1/terminal-recordings?start_date=2025-01-01&end_date=2025-01-31
```

### 向后兼容的 API

```
# Docker（保留，内部调用统一 API）
GET    /api/v1/docker/recordings
GET    /api/v1/docker/recordings/:id
DELETE /api/v1/docker/recordings/:id
```

## 数据迁移

### 迁移步骤

1. **创建新表** `terminal_recordings`
2. **迁移 Docker 录制数据**：
   ```sql
   INSERT INTO terminal_recordings (
       id, session_id, type, user_id, username,
       started_at, ended_at, duration,
       storage_path, file_size, format,
       rows, cols, shell,
       client_ip, metadata,
       created_at, updated_at
   )
   SELECT
       id, session_id, 'docker_container', user_id, username,
       started_at, ended_at, duration,
       storage_path, file_size, format,
       rows, cols, shell,
       client_ip,
       JSON_BUILD_OBJECT(
           'instance_id', instance_id,
           'container_id', container_id
       ),
       created_at, updated_at
   FROM terminal_recordings_old;
   ```
3. **验证数据一致性**
4. **删除旧表**（在所有系统迁移完成后）

## 测试策略

### 单元测试
- 录制器接口测试
- 存储接口测试
- Repository 测试
- Service 测试

### 集成测试
- Docker 终端录制端到端测试
- WebSSH 终端录制端到端测试
- K8s 终端录制端到端测试

### 性能测试
- 大文件录制（500MB）
- 并发录制（100 个会话）
- 查询性能（10000+ 录制）

## 风险与缓解

### 风险 1: 数据迁移失败
**缓解**：
- 先在测试环境验证
- 提供回滚脚本
- 迁移前备份数据

### 风险 2: 存储空间不足
**缓解**：
- 实现自动清理机制
- 配置最大文件大小
- 监控存储使用率

### 风险 3: 向后兼容性问题
**缓解**：
- 保留旧 API 端点
- 逐步迁移
- 充分测试

## 优势

1. **代码复用**：核心录制逻辑只实现一次
2. **统一管理**：所有终端录制在一个地方查询和管理
3. **易于扩展**：添加新的终端类型只需添加 Metadata
4. **统一体验**：前端只需一个录制列表页面
5. **存储灵活**：支持本地和 MinIO 存储
6. **审计友好**：统一的录制格式和查询接口

## 参考资料

- [Asciinema v2 Format](https://github.com/asciinema/asciinema/blob/develop/doc/asciicast-v2.md)
- 现有实现：
  - `internal/models/terminal_recording.go` (Docker)
  - `internal/services/webssh/recorder.go` (WebSSH)
