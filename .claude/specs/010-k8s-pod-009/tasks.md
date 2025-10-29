# 任务：K8s 终端录制与审计增强

**功能分支**: `010-k8s-pod-009`
**输入**: 来自 `.claude/specs/010-k8s-pod-009/` 的设计文档
**前提条件**: plan.md、research.md、data-model.md、contracts/、quickstart.md

## 执行流程摘要

```
1. ✅ 加载 plan.md → 技术栈：Go 1.24+、React 19、PostgreSQL/MySQL
2. ✅ 加载 data-model.md → 3 个实体（TerminalRecording 扩展、AuditEvent 扩展、K8sTerminalSession）
3. ✅ 加载 contracts/ → 2 个契约文件（10 个 API 端点）
4. ✅ 加载 research.md → 5 个技术决策
5. ✅ 加载 quickstart.md → 10 个验证场景
6. → 生成任务：设置(3) + 测试(12) + 核心(15) + 集成(5) + 优化(5) = 40 个任务
7. → 应用依赖排序：测试优先（TDD）
8. → 标记并行任务 [P]
```

---

## 格式说明

- **[P]** = 可并行执行（不同文件，无依赖）
- **文件路径** = 绝对路径或仓库根目录相对路径
- **依赖** = 阻塞关系（必须先完成）

---

## 阶段 3.1：设置与准备

### T001: 扩展 TerminalRecording 模型支持 K8s 类型
- **文件**: `internal/models/terminal_recording.go`
- **依赖**: 无
- **标记**: [P]
- **描述**:
  - 添加 K8s 录制类型常量：
    ```go
    const (
        RecordingTypeK8sNode = "k8s_node"
        RecordingTypeK8sPod  = "k8s_pod"
    )
    ```
  - 添加 2 小时时长验证规则（7200 秒）
  - 添加类型元数据验证方法 `ValidateTypeMetadata()`
  - 添加过期检查方法 `IsExpired()` (90 天)
- **验收**:
  - `go test ./internal/models -run TestTerminalRecording`
  - 验证 K8s 类型常量定义
  - 验证 2 小时限制检查

### T002: 扩展 AuditEvent 模型支持 K8s 子系统
- **文件**: `internal/models/audit_event.go`
- **依赖**: 无
- **标记**: [P]
- **描述**:
  - 添加 K8s 操作类型常量：
    ```go
    const (
        ActionCreateResource    Action = "CreateResource"
        ActionUpdateResource    Action = "UpdateResource"
        ActionDeleteResource    Action = "DeleteResource"
        ActionViewResource      Action = "ViewResource"
        ActionNodeTerminalAccess Action = "NodeTerminalAccess"
        ActionPodTerminalAccess  Action = "PodTerminalAccess"
    )
    ```
  - 添加 K8s 资源类型常量：
    ```go
    const (
        ResourceTypeK8sNode ResourceType = "k8s_node"
        ResourceTypeK8sPod  ResourceType = "k8s_pod"
    )
    ```
  - 添加过期检查方法 `IsExpired()` (90 天)
  - 添加只读操作判断方法 `IsReadOnlyOperation()` 和 `IsModifyOperation()`
- **验收**:
  - `go test ./internal/models -run TestAuditEvent`
  - 验证所有 K8s 常量定义

### T003: 创建数据库索引（K8s 优化）
- **文件**: 数据库迁移或 GORM hooks
- **依赖**: T001, T002
- **标记**: 无
- **描述**:
  - 为 TerminalRecording 创建 K8s 专用索引：
    ```sql
    CREATE INDEX idx_terminal_recordings_k8s_cluster
    ON terminal_recordings((type_metadata->>'cluster_id'))
    WHERE recording_type IN ('k8s_node', 'k8s_pod');

    CREATE INDEX idx_terminal_recordings_k8s_node
    ON terminal_recordings((type_metadata->>'node_name'))
    WHERE recording_type = 'k8s_node';

    CREATE INDEX idx_terminal_recordings_k8s_pod
    ON terminal_recordings((type_metadata->>'pod_name'))
    WHERE recording_type = 'k8s_pod';
    ```
  - 为 AuditEvent 创建 K8s 查询优化索引（PostgreSQL）：
    ```sql
    CREATE INDEX idx_audit_events_k8s_query
    ON audit_events(resource_type, action, timestamp DESC)
    WHERE subsystem = 'kubernetes';
    ```
- **验收**:
  - 启动应用，验证索引自动创建
  - 运行 EXPLAIN 查询验证索引使用

---

## 阶段 3.2：测试优先（TDD）⚠️ 必须在 3.3 之前完成

**关键：这些测试必须编写并且必须在任何实现之前失败**

### 录制 API 契约测试

### T004 [P]: 测试 GET /api/v1/recordings 契约（K8s 类型筛选）
- **文件**: `tests/contract/k8s/terminal_recording_list_test.go`
- **依赖**: T001
- **标记**: [P]
- **描述**:
  - 测试查询参数：`recording_type=k8s_node`, `recording_type=k8s_pod`
  - 测试 K8s 专用筛选：`cluster_id`, `node_name`, `namespace`, `pod_name`, `container_name`
  - 验证响应 JSON 模式包含 `type_metadata` 字段
  - 测试分页（page, page_size）
- **验收**: 测试运行并失败（录制 API 尚未扩展 K8s 类型筛选）

### T005 [P]: 测试 GET /api/v1/recordings/:id 契约（K8s 元数据）
- **文件**: `tests/contract/k8s/terminal_recording_detail_test.go`
- **依赖**: T001
- **标记**: [P]
- **描述**:
  - 测试 K8s 录制详情包含正确的 `type_metadata` 结构
  - 验证 k8s_node 元数据：`cluster_id`, `node_name`
  - 验证 k8s_pod 元数据：`cluster_id`, `namespace`, `pod_name`, `container_name`
- **验收**: 测试运行并失败

### T006 [P]: 测试 GET /api/v1/recordings/:id/play 契约（Asciinema v2）
- **文件**: `tests/contract/k8s/terminal_recording_play_test.go`
- **依赖**: T001
- **标记**: [P]
- **描述**:
  - 测试回放返回 Asciinema v2 格式（application/x-asciicast）
  - 验证 Header 行格式：`{"version": 2, "width": 120, ...}`
  - 验证 Frame 行格式：`[time, type, data]`
- **验收**: 测试运行并失败

### T007 [P]: 测试 GET /api/v1/recordings/stats 契约（K8s 统计）
- **文件**: `tests/contract/k8s/terminal_recording_stats_test.go`
- **依赖**: T001
- **标记**: [P]
- **描述**:
  - 测试统计 API 包含 `by_type.k8s_node` 和 `by_type.k8s_pod`
  - 验证响应包含 count, total_size, total_duration
- **验收**: 测试运行并失败

### 审计 API 契约测试

### T008 [P]: 测试 GET /api/v1/audit/events 契约（K8s 子系统筛选）
- **文件**: `tests/contract/k8s/audit_events_list_test.go`
- **依赖**: T002
- **标记**: [P]
- **描述**:
  - 测试 `subsystem=kubernetes` 筛选
  - 测试 K8s 操作类型筛选：`action=CreateResource`, `action=NodeTerminalAccess` 等
  - 测试 K8s 资源类型筛选：`resource_type=k8s_node`, `resource_type=k8s_pod`
  - 测试集群筛选：`cluster_id`
  - 验证响应包含 `resource.data` 结构（cluster_id, namespace, resource_name）
- **验收**: 测试运行并失败（审计 API 尚未支持 K8s 筛选）

### T009 [P]: 测试 GET /api/v1/audit/events/:id 契约（K8s 详情）
- **文件**: `tests/contract/k8s/audit_event_detail_test.go`
- **依赖**: T002
- **标记**: [P]
- **描述**:
  - 测试审计详情包含 `diff_object`（更新操作）
  - 验证 K8s 资源数据结构：`cluster_id`, `namespace`, `resource_name`, `change_summary`, `recording_id`
- **验收**: 测试运行并失败

### T010 [P]: 测试 GET /api/v1/audit/stats 契约（K8s 统计）
- **文件**: `tests/contract/k8s/audit_stats_test.go`
- **依赖**: T002
- **标记**: [P]
- **描述**:
  - 测试统计 API 包含 `by_action` 分组（CreateResource, UpdateResource 等）
  - 测试 `by_resource_type` 分组（包含 k8s_node, k8s_pod）
  - 测试 `success_rate` 计算
- **验收**: 测试运行并失败

### 集成测试（端到端场景）

### T011 [P]: 节点终端录制集成测试
- **文件**: `tests/integration/k8s/node_terminal_recording_test.go`
- **依赖**: T001
- **标记**: [P]
- **描述**:
  - 场景 1-4（quickstart.md）：连接节点终端 → 执行命令 → 断开 → 验证录制
  - 测试步骤：
    1. 模拟 WebSocket 连接到节点终端
    2. 发送测试命令（ls、ps）
    3. 断开连接
    4. 查询录制列表，验证 `recording_type=k8s_node`
    5. 验证 `type_metadata` 包含 `cluster_id` 和 `node_name`
    6. 验证录制文件存在于 `./recordings/k8s_node/{YYYY-MM-DD}/{id}.cast`
    7. 验证文件格式为 Asciinema v2
- **验收**: 测试运行并失败（节点终端录制尚未实现）

### T012 [P]: 容器终端录制集成测试
- **文件**: `tests/integration/k8s/pod_terminal_recording_test.go`
- **依赖**: T001
- **标记**: [P]
- **描述**:
  - 场景 5-7（quickstart.md）：Exec 进入容器 → 执行命令 → 断开 → 验证录制
  - 测试步骤：
    1. 创建测试 Pod（nginx）
    2. 模拟 WebSocket Exec 连接
    3. 发送测试命令
    4. 断开连接
    5. 查询录制列表，验证 `recording_type=k8s_pod`
    6. 验证 `type_metadata` 包含 `cluster_id`, `namespace`, `pod_name`, `container_name`
    7. 验证录制文件存在于 `./recordings/k8s_pod/{YYYY-MM-DD}/{id}.cast`
- **验收**: 测试运行并失败

### T013 [P]: K8s 资源操作审计集成测试
- **文件**: `tests/integration/k8s/resource_audit_test.go`
- **依赖**: T002
- **标记**: [P]
- **描述**:
  - 场景 8-10（quickstart.md）：创建/更新/删除 Deployment → 验证审计日志
  - 测试步骤：
    1. 创建 Deployment（通过 API）
    2. 等待 2 秒（异步审计日志写入）
    3. 查询审计日志：`subsystem=kubernetes&action=CreateResource`
    4. 验证审计日志包含：cluster_id, namespace, resource_name, success=true
    5. 更新 Deployment（修改 replicas）
    6. 验证更新审计包含 `change_summary`
    7. 删除 Deployment
    8. 验证删除审计记录
- **验收**: 测试运行并失败

### T014 [P]: 终端访问审计集成测试
- **文件**: `tests/integration/k8s/terminal_access_audit_test.go`
- **依赖**: T002
- **标记**: [P]
- **描述**:
  - 场景 11-12（quickstart.md）：连接终端 → 验证审计日志（包含 recording_id）
  - 测试步骤：
    1. 连接节点终端
    2. 查询审计日志：`action=NodeTerminalAccess`
    3. 验证审计日志包含 `recording_id`
    4. 连接容器终端
    5. 查询审计日志：`action=PodTerminalAccess`
    6. 验证审计日志包含 `recording_id` 和正确的 Pod 信息
- **验收**: 测试运行并失败

### T015 [P]: 审计日志查询集成测试
- **文件**: `tests/integration/k8s/audit_query_test.go`
- **依赖**: T002
- **标记**: [P]
- **描述**:
  - 场景 13-16（quickstart.md）：测试多维度筛选和分页
  - 测试步骤：
    1. 按操作者筛选（user_id）
    2. 按操作类型筛选（action）
    3. 按时间范围筛选（start_time, end_time）
    4. 测试分页（page=1, page_size=50）
    5. 测试排序（order=desc）
    6. 验证查询性能 < 500ms
- **验收**: 测试运行并失败

---

## 阶段 3.3：核心实现（仅在测试失败后）

### 录制系统核心

### T016 [P]: 实现 AsciinemaRecorder（实时写入）
- **文件**: `internal/services/recording/asciinema_recorder.go`
- **依赖**: T001
- **标记**: [P]
- **描述**:
  - 实现 Asciinema v2 格式录制器
  - 方法：
    ```go
    type AsciinemaRecorder struct {
        file       *os.File
        startTime  time.Time
        recording  bool
        mutex      sync.Mutex
    }

    func NewAsciinemaRecorder(filePath string, width, height int, title string) (*AsciinemaRecorder, error)
    func (r *AsciinemaRecorder) WriteFrame(frameType string, data []byte) error
    func (r *AsciinemaRecorder) Stop() error
    ```
  - 写入 Header（第一行）：`{"version": 2, "width": 120, ...}`
  - 写入 Frame（后续行）：`[elapsed, type, data]`
  - 线程安全（使用 mutex）
- **验收**:
  - 单元测试：`go test ./internal/services/recording -run TestAsciinemaRecorder`
  - 验证生成的 .cast 文件格式正确

### T017 [P]: 实现 K8sTerminalSession（内存会话管理）
- **文件**: `pkg/kube/terminal_session.go`
- **依赖**: T001, T016
- **标记**: [P]
- **描述**:
  - 实现 K8s 终端会话结构（data-model.md 实体 3）
  - 字段：SessionID, Type, ClusterID, NodeName, Namespace, PodName, ContainerName, RecordingID, RecordingState, Recorder, StartedAt, Timer, Conn, Mutex
  - 方法：
    ```go
    func NewK8sTerminalSession(...) *K8sTerminalSession
    func (s *K8sTerminalSession) StartRecording(recorder *AsciinemaRecorder)
    func (s *K8sTerminalSession) StopRecording() error
    func (s *K8sTerminalSession) handleRecordingTimeout()
    func (s *K8sTerminalSession) Close() error
    ```
  - 启动 2 小时定时器（`time.AfterFunc`）
  - 定时器触发时：停止录制 + 发送 WebSocket 通知 + 保持连接
- **验收**:
  - 单元测试：`go test ./pkg/kube -run TestK8sTerminalSession`
  - 验证 2 小时定时器触发

### T018: 实现 SessionManager（全局会话管理）
- **文件**: `pkg/kube/session_manager.go`
- **依赖**: T017
- **标记**: 无
- **描述**:
  - 实现全局会话管理器（使用 sync.Map）
  - 方法：
    ```go
    type SessionManager struct {
        sessions sync.Map // map[uuid.UUID]*K8sTerminalSession
    }

    func (m *SessionManager) AddSession(session *K8sTerminalSession)
    func (m *SessionManager) GetSession(sessionID uuid.UUID) (*K8sTerminalSession, bool)
    func (m *SessionManager) RemoveSession(sessionID uuid.UUID)
    ```
- **验收**:
  - 单元测试：`go test ./pkg/kube -run TestSessionManager`
  - 验证并发安全

### T019: 集成录制到节点终端（pkg/kube/terminal.go）
- **文件**: `pkg/kube/terminal.go`
- **依赖**: T016, T017, T018
- **标记**: 无
- **描述**:
  - 修改 `HandleNodeTerminal` 函数：
    1. 创建 K8sTerminalSession
    2. 创建 AsciinemaRecorder（路径：`./recordings/k8s_node/{YYYY-MM-DD}/{id}.cast`）
    3. 创建 TerminalRecording 数据库记录（recording_type=k8s_node）
    4. 启动录制（startRecording）
    5. 包装 WebSocket 读写，拦截数据流调用 `recorder.WriteFrame()`
    6. 连接断开或 2 小时超时：停止录制，上传文件，更新数据库记录
  - 使用装饰器模式（research.md 任务 1 决策）
- **验收**:
  - 运行 T011 集成测试，验证通过
  - 手动测试：连接节点终端 → 执行命令 → 验证录制文件

### T020: 集成录制到容器终端（pkg/kube/terminal.go）
- **文件**: `pkg/kube/terminal.go`
- **依赖**: T019
- **标记**: 无
- **描述**:
  - 修改 `HandlePodExec` 函数（类似 T019）
  - 录制类型：`recording_type=k8s_pod`
  - 路径：`./recordings/k8s_pod/{YYYY-MM-DD}/{id}.cast`
  - type_metadata 包含：cluster_id, namespace, pod_name, container_name
- **验收**:
  - 运行 T012 集成测试，验证通过
  - 手动测试：Exec 进入容器 → 执行命令 → 验证录制文件

### 审计系统核心

### T021 [P]: 实现 K8sAuditService（审计日志记录）
- **文件**: `internal/services/k8s/audit_service.go`
- **依赖**: T002
- **标记**: [P]
- **描述**:
  - 实现 K8s 审计服务
  - 方法：
    ```go
    type AuditService struct {
        asyncLogger *audit.AsyncLogger[*models.AuditEvent]
        clusterRepo repository.ClusterRepositoryInterface
    }

    func NewAuditService(asyncLogger *audit.AsyncLogger[*models.AuditEvent], clusterRepo repository.ClusterRepositoryInterface) *AuditService
    func (s *AuditService) LogResourceOperation(ctx context.Context, log *ResourceOperationLog)
    func (s *AuditService) LogTerminalAccess(ctx context.Context, log *TerminalAccessLog)
    func (s *AuditService) LogReadOperation(ctx context.Context, log *ReadOperationLog)
    ```
  - 使用现有 `async_audit_logger.go`（异步写入，批量 100 条或 1 秒）
  - subsystem 设置为 `SubsystemKubernetes`
- **验收**:
  - 单元测试：`go test ./internal/services/k8s -run TestAuditService`
  - 验证异步写入正常

### T022: 实现 K8sAuditMiddleware（Gin 中间件）
- **文件**: `internal/api/middleware/k8s_audit.go`
- **依赖**: T021
- **标记**: 无
- **描述**:
  - 实现 Gin 中间件拦截 `/api/v1/k8s/` 路径
  - 提取通用信息：user, client_ip, start_time
  - 注入审计上下文到 `context.Context`
  - 执行请求后记录审计日志（仅修改操作：POST/PUT/PATCH/DELETE）
  - 只读操作（GET）由处理器内部记录
  - 映射 HTTP 方法到 Action：
    ```go
    POST -> ActionCreateResource
    PUT/PATCH -> ActionUpdateResource
    DELETE -> ActionDeleteResource
    ```
- **验收**:
  - 单元测试：`go test ./internal/api/middleware -run TestK8sAuditMiddleware`

### T023: 为 K8s 资源处理器添加审计拦截（创建/更新/删除）
- **文件**: `pkg/handlers/resources/*.go`（deployment_handler.go, service_handler.go 等）
- **依赖**: T022
- **标记**: 无
- **描述**:
  - 修改 GenericResourceHandler 的 Create/Update/Delete 方法
  - 在操作完成后调用 `auditService.LogResourceOperation()`
  - 记录详情：
    - Create: resource_name, namespace, cluster_id, success
    - Update: resource_name, change_summary（生成 YAML diff）, old_object, new_object
    - Delete: resource_name, namespace, cluster_id, success
  - 使用 `generateChangeSummary()` 函数生成变更摘要（research.md 任务 3）
- **验收**:
  - 运行 T013 集成测试，验证通过
  - 手动测试：创建/更新/删除 Deployment → 查询审计日志

### T024 [P]: 为只读操作添加审计（查看详情、查看日志）
- **文件**: `pkg/handlers/resources/pod_handler.go`（GetPodDetails, GetPodLogs）
- **依赖**: T021
- **标记**: [P]
- **描述**:
  - 在 GetPodDetails、GetPodLogs、GetPodYAML、GetPodEvents 等方法中添加审计
  - 使用 `go` 异步记录（不阻塞响应）：
    ```go
    go auditService.LogReadOperation(context.Background(), &ReadOperationLog{
        Action:       models.ActionViewResource,
        ResourceType: models.ResourceTypePod,
        ResourceName: name,
        Namespace:    namespace,
        ClusterID:    clusterID,
        User:         user,
        ClientIP:     clientIP,
    })
    ```
  - Action 类型：`ActionViewResource`
- **验收**:
  - 手动测试：查看 Pod 详情 → 查询审计日志（action=ViewResource）

### T025: 为终端访问添加审计（关联 recording_id）
- **文件**: `pkg/kube/terminal.go`
- **依赖**: T019, T020, T021
- **标记**: 无
- **描述**:
  - 在 HandleNodeTerminal 和 HandlePodExec 中添加审计
  - 终端连接成功后调用 `auditService.LogTerminalAccess()`
  - 记录：
    - Action: `ActionNodeTerminalAccess` 或 `ActionPodTerminalAccess`
    - ResourceType: `ResourceTypeK8sNode` 或 `ResourceTypeK8sPod`
    - recording_id: 关联的 TerminalRecording ID
    - cluster_id, node_name/pod_name, success
- **验收**:
  - 运行 T014 集成测试，验证通过
  - 手动测试：连接终端 → 查询审计日志（验证 recording_id 存在）

### 查询与统计

### T026: 扩展 TerminalRecordingRepository 查询方法
- **文件**: `internal/repository/terminal_recording_repo.go`
- **依赖**: T001
- **标记**: 无
- **描述**:
  - 添加 K8s 类型筛选方法：
    ```go
    func (r *TerminalRecordingRepository) FindByCluster(ctx context.Context, clusterID uuid.UUID, page, pageSize int) ([]*models.TerminalRecording, int64, error)
    func (r *TerminalRecordingRepository) FindByNode(ctx context.Context, clusterID uuid.UUID, nodeName string, page, pageSize int) ([]*models.TerminalRecording, int64, error)
    func (r *TerminalRecordingRepository) FindByPod(ctx context.Context, clusterID uuid.UUID, namespace, podName string, page, pageSize int) ([]*models.TerminalRecording, int64, error)
    ```
  - 使用 JSONB 查询：`type_metadata->>'cluster_id' = ?`
  - 利用 T003 创建的索引
- **验收**:
  - 单元测试：`go test ./internal/repository -run TestTerminalRecordingRepository`
  - 验证查询性能（使用 EXPLAIN）

### T027: 扩展 AuditEventRepository 查询方法
- **文件**: `internal/repository/audit_event_repo.go`
- **依赖**: T002
- **标记**: 无
- **描述**:
  - 添加 K8s 子系统筛选方法：
    ```go
    func (r *AuditEventRepository) FindBySubsystem(ctx context.Context, subsystem models.SubsystemType, filters *AuditFilters, page, pageSize int) ([]*models.AuditEvent, int64, error)
    func (r *AuditEventRepository) GetStatsBySubsystem(ctx context.Context, subsystem models.SubsystemType, startTime, endTime time.Time) (*AuditStats, error)
    ```
  - AuditFilters 包含：action, resource_type, cluster_id, user_id, start_time, end_time
  - 使用 T003 创建的索引（idx_audit_events_k8s_query）
- **验收**:
  - 单元测试：`go test ./internal/repository -run TestAuditEventRepository`
  - 运行 T015 集成测试，验证查询性能

### T028: 实现录制 API Handler 扩展（K8s 类型）
- **文件**: `internal/api/handlers/recording/handler.go`
- **依赖**: T026
- **标记**: 无
- **描述**:
  - 扩展 `ListRecordings` 方法支持 K8s 筛选参数：
    - `recording_type=k8s_node|k8s_pod`
    - `cluster_id`, `node_name`, `namespace`, `pod_name`, `container_name`
  - 扩展 `GetRecordingStats` 方法包含 K8s 类型统计
  - 复用现有 GetRecording、PlayRecording、DeleteRecording 方法（无需修改）
- **验收**:
  - 运行 T004-T007 契约测试，验证通过
  - 手动测试：`curl "/api/v1/recordings?recording_type=k8s_node&cluster_id=xxx"`

### T029: 实现审计 API Handler 扩展（K8s 子系统）
- **文件**: `internal/api/handlers/audit_handler.go`
- **依赖**: T027
- **标记**: 无
- **描述**:
  - 扩展 `ListAuditEvents` 方法支持 K8s 筛选参数：
    - `subsystem=kubernetes`
    - `action`, `resource_type`, `cluster_id`, `user_id`, `start_time`, `end_time`
  - 扩展 `GetAuditStats` 方法包含 K8s 统计：
    - `by_action`: CreateResource, UpdateResource, DeleteResource, ViewResource, NodeTerminalAccess, PodTerminalAccess
    - `by_resource_type`: k8s_node, k8s_pod
  - 复用现有 GetAuditEvent 方法（无需修改）
- **验收**:
  - 运行 T008-T010 契约测试，验证通过
  - 手动测试：`curl "/api/v1/audit/events?subsystem=kubernetes&action=CreateResource"`

### T030: 注册 K8sAuditMiddleware 到路由
- **文件**: `internal/api/routes.go`
- **依赖**: T022
- **标记**: 无
- **描述**:
  - 为 `/api/v1/k8s/*` 路由组添加 K8sAuditMiddleware
  - 顺序：Auth → RBAC → K8sAudit → Handler
  - 示例：
    ```go
    k8sGroup := v1.Group("/k8s")
    k8sGroup.Use(middleware.AuthMiddleware())
    k8sGroup.Use(middleware.RBACMiddleware())
    k8sGroup.Use(middleware.K8sAuditMiddleware(auditService))
    ```
- **验收**:
  - 启动应用，验证中间件加载
  - 手动测试：创建资源 → 验证审计日志记录

---

## 阶段 3.4：前端集成

### T031 [P]: 扩展录制列表页面筛选器（K8s 类型）
- **文件**: `ui/src/pages/system/recordings/index.tsx`
- **依赖**: T028
- **标记**: [P]
- **描述**:
  - 添加 K8s 录制类型筛选：
    - 录制类型下拉菜单包含：Docker、WebSSH、K8s Node、K8s Pod
    - K8s 类型选中时显示额外筛选：集群、节点名称、命名空间、Pod 名称
  - 更新 API 调用：`/api/v1/recordings?recording_type=k8s_node&cluster_id=xxx`
  - 显示 K8s 元数据：在表格中显示集群、节点/Pod 信息
- **验收**:
  - 前端开发服务器：`cd ui && pnpm dev`
  - 手动测试：筛选 K8s 录制 → 验证显示正确

### T032 [P]: 实现 K8s 审计日志页面（多维度筛选）
- **文件**: `ui/src/pages/k8s/audit/index.tsx`
- **依赖**: T029
- **标记**: [P]
- **描述**:
  - 创建审计日志页面组件（参考 research.md 任务 5 设计）
  - 筛选器面板：
    - 子系统：Kubernetes（默认）
    - 操作类型：创建、更新、删除、查看、节点终端访问、容器终端访问
    - 资源类型：Deployment、Service、Pod、k8s_node、k8s_pod
    - 集群选择下拉
    - 时间范围选择器（预设：最近 1 小时、24 小时、7 天、30 天）
    - 操作者筛选
  - 数据表格：时间、操作者、操作类型、资源类型、资源名称、集群、结果、操作（详情按钮）
  - 使用 TanStack Query 数据获取
- **验收**:
  - 手动测试：筛选 K8s 审计日志 → 验证分页和排序

### T033: 实现审计日志详情抽屉（变更对比）
- **文件**: `ui/src/pages/k8s/audit/components/AuditLogDetails.tsx`
- **依赖**: T032
- **标记**: 无
- **描述**:
  - 创建详情抽屉组件（Radix UI Sheet）
  - 显示基础信息：时间、操作者、客户端 IP、操作类型
  - 显示资源信息：资源类型、资源名称、命名空间、集群
  - 显示变更对比（仅更新操作）：
    - 使用 YAML diff 高亮组件
    - 显示 change_summary
    - 显示 old_object 和 new_object（YAML 格式）
  - 显示录制链接（仅终端访问）：点击 recording_id 跳转到录制回放页面
- **验收**:
  - 手动测试：点击审计日志详情 → 验证变更对比显示正确

### T034 [P]: 实现审计统计图表
- **文件**: `ui/src/pages/k8s/audit/components/AuditStatsChart.tsx`
- **依赖**: T032
- **标记**: [P]
- **描述**:
  - 创建统计图表组件（使用 recharts 或类似库）
  - 柱状图：按操作类型分组（创建、更新、删除、查看、终端访问）
  - 饼图：成功率（successful vs failed）
  - 表格：按用户分组统计
- **验收**:
  - 手动测试：查看统计图表 → 验证数据正确

### T035: 添加 K8s 审计日志路由和导航
- **文件**: `ui/src/layouts/k8s-layout.tsx`、`ui/src/router.tsx`
- **依赖**: T032
- **标记**: 无
- **描述**:
  - 在 K8s 子系统导航菜单添加"审计日志"链接
  - 添加路由：`/k8s/audit`
  - 确保需要认证和 RBAC 权限
- **验收**:
  - 手动测试：从 K8s 菜单访问审计日志页面

---

## 阶段 3.5：优化与验证

### T036: 录制清理集成验证（复用 CleanupService）
- **文件**: 验证现有 `internal/services/recording/cleanup_service.go`
- **依赖**: T020
- **标记**: 无
- **描述**:
  - 验证 CleanupService 已支持 K8s 录制类型清理
  - 测试步骤：
    1. 创建测试录制（k8s_node 和 k8s_pod）
    2. 修改 ended_at 为 91 天前
    3. 手动触发清理：`POST /api/v1/recordings/cleanup`
    4. 验证录制记录被删除，文件被删除
  - 无需修改代码（CleanupService 通过 recording_type 自动识别）
- **验收**:
  - 手动测试：触发清理 → 验证过期 K8s 录制被删除

### T037 [P]: 审计日志清理实现（90 天保留期）
- **文件**: `internal/services/k8s/audit_cleanup_service.go`
- **依赖**: T021
- **标记**: [P]
- **描述**:
  - 实现 K8s 审计日志清理服务（复用 CleanupService 模式）
  - 方法：
    ```go
    func (s *AuditCleanupService) CleanExpiredEvents(ctx context.Context, retentionDays int) (int64, error)
    ```
  - 删除条件：`created_at < NOW() - INTERVAL '90 days'`
  - 支持 dry_run 模式
  - 集成到定时任务调度器（每日凌晨 2 点运行）
- **验收**:
  - 单元测试：`go test ./internal/services/k8s -run TestAuditCleanupService`
  - 手动测试：`POST /api/v1/audit/cleanup` → 验证过期审计被删除

### T038 [P]: 性能基准测试
- **文件**: `tests/benchmark/k8s_audit_benchmark_test.go`
- **依赖**: T023, T027
- **标记**: [P]
- **描述**:
  - 基准测试终端连接延迟（目标 < 100ms）
  - 基准测试资源操作延迟（目标 < 50ms）
  - 基准测试审计日志查询（目标 < 500ms）
  - 基准测试审计日志写入吞吐量（目标 > 1000 TPS）
  - 使用 `go test -bench` 运行
- **验收**:
  - 运行基准测试：`go test -bench=. ./tests/benchmark/`
  - 验证所有性能指标满足目标

### T039: 执行 quickstart.md 手动验证
- **文件**: `.claude/specs/010-k8s-pod-009/quickstart.md`
- **依赖**: T001-T035
- **标记**: 无
- **描述**:
  - 按照 quickstart.md 的 10 个步骤逐一验证：
    1. 环境设置（kind 集群、Tiga 服务、测试数据）
    2. 验证节点终端录制
    3. 验证容器终端录制
    4. 验证资源操作审计
    5. 验证终端访问审计
    6. 验证审计日志查询功能
    7. 验证录制限制和清理
    8. 前端验证（录制页面、审计页面）
    9. 性能验证
    10. 清理
  - 对照成功检查清单（quickstart.md 末尾）
- **验收**:
  - 所有 20+ 检查项通过 ✅

### T040: 更新文档和代码注释
- **文件**: 相关代码文件、README.md、CLAUDE.md
- **依赖**: T039
- **标记**: 无
- **描述**:
  - 为新增代码添加 GoDoc 注释
  - 更新 CLAUDE.md 的完成度：0% → 100%
  - 更新 README.md（如需要）
  - 为 Swagger 添加 K8s 审计 API 注释
  - 生成 Swagger 文档：`./scripts/generate-swagger.sh`
- **验收**:
  - `go doc ./internal/services/k8s`
  - 访问 `http://localhost:12306/swagger/index.html` 验证 API 文档

---

## 依赖关系图

```
设置阶段：
T001 [P] ────┬────> T003
T002 [P] ────┘

测试阶段（TDD）：
T001 ──> T004 [P], T005 [P], T006 [P], T007 [P], T011 [P]
T002 ──> T008 [P], T009 [P], T010 [P], T013 [P], T014 [P], T015 [P]

核心实现（录制）：
T001, T016 [P] ──> T017 [P] ──> T018 ──> T019 ──> T020

核心实现（审计）：
T002 ──> T021 [P] ──> T022 ──> T023 ──> T024 [P], T025

查询扩展：
T001 ──> T026 ──> T028
T002 ──> T027 ──> T029
T022 ──> T030

前端集成：
T028 ──> T031 [P]
T029 ──> T032 [P] ──> T033 ──> T035
T032 ──> T034 [P]

优化验证：
T020 ──> T036
T021 ──> T037 [P]
T023, T027 ──> T038 [P]
T001-T035 ──> T039 ──> T040
```

---

## 并行执行示例

### 阶段 1：设置并行
```bash
# 同时执行 T001 和 T002（不同文件）
task agent -- "T001: 扩展 TerminalRecording 模型支持 K8s 类型。文件：internal/models/terminal_recording.go" &
task agent -- "T002: 扩展 AuditEvent 模型支持 K8s 子系统。文件：internal/models/audit_event.go" &
wait
```

### 阶段 2：测试优先并行（6 个契约测试）
```bash
# 录制 API 契约测试
task agent -- "T004: 测试 GET /api/v1/recordings 契约（K8s 类型筛选）。文件：tests/contract/k8s/terminal_recording_list_test.go" &
task agent -- "T005: 测试 GET /api/v1/recordings/:id 契约（K8s 元数据）。文件：tests/contract/k8s/terminal_recording_detail_test.go" &
task agent -- "T006: 测试 GET /api/v1/recordings/:id/play 契约（Asciinema v2）。文件：tests/contract/k8s/terminal_recording_play_test.go" &
task agent -- "T007: 测试 GET /api/v1/recordings/stats 契约（K8s 统计）。文件：tests/contract/k8s/terminal_recording_stats_test.go" &

# 审计 API 契约测试
task agent -- "T008: 测试 GET /api/v1/audit/events 契约（K8s 子系统筛选）。文件：tests/contract/k8s/audit_events_list_test.go" &
task agent -- "T009: 测试 GET /api/v1/audit/events/:id 契约（K8s 详情）。文件：tests/contract/k8s/audit_event_detail_test.go" &
wait

# 验证所有测试失败（TDD）
go test ./tests/contract/k8s/... -v
```

### 阶段 3：核心实现并行（独立文件）
```bash
# 录制核心组件
task agent -- "T016: 实现 AsciinemaRecorder（实时写入）。文件：internal/services/recording/asciinema_recorder.go" &
task agent -- "T017: 实现 K8sTerminalSession（内存会话管理）。文件：pkg/kube/terminal_session.go" &

# 审计核心组件
task agent -- "T021: 实现 K8sAuditService（审计日志记录）。文件：internal/services/k8s/audit_service.go" &
wait
```

### 阶段 4：前端并行
```bash
# 前端 UI 组件
task agent -- "T031: 扩展录制列表页面筛选器（K8s 类型）。文件：ui/src/pages/system/recordings/index.tsx" &
task agent -- "T032: 实现 K8s 审计日志页面（多维度筛选）。文件：ui/src/pages/k8s/audit/index.tsx" &
task agent -- "T034: 实现审计统计图表。文件：ui/src/pages/k8s/audit/components/AuditStatsChart.tsx" &
wait
```

---

## 预计时间估算

| 阶段 | 任务数 | 并行执行 | 估算时间 |
|------|--------|---------|---------|
| 3.1 设置 | 3 | 2 并行 | 2-3 小时 |
| 3.2 测试 | 12 | 6 并行 | 4-6 小时 |
| 3.3 核心实现 | 15 | 部分并行 | 12-16 小时 |
| 3.4 前端集成 | 5 | 3 并行 | 6-8 小时 |
| 3.5 优化验证 | 5 | 2 并行 | 4-6 小时 |
| **总计** | **40** | **高并行度** | **28-39 小时** |

**注意**：估算基于单人开发。使用并行执行和多人协作可显著缩短时间。

---

## 验证清单

*门禁：在标记功能完成前检查*

- [ ] 所有 40 个任务已完成
- [ ] 所有契约测试通过（T004-T010）
- [ ] 所有集成测试通过（T011-T015）
- [ ] 性能基准测试满足目标（T038）
- [ ] quickstart.md 手动验证通过（T039）
- [ ] 前端 UI 功能正常（T031-T035）
- [ ] 文档和注释完整（T040）
- [ ] 代码审查通过
- [ ] 无已知 Bug

---

## 下一步

1. **开始执行**: 从 T001 和 T002 开始（并行）
2. **TDD 严格遵循**: 测试必须在实现前失败
3. **提交频率**: 每个任务完成后提交一次
4. **进度跟踪**: 在 plan.md 更新进度
5. **问题反馈**: 遇到阻塞及时记录到 spec.md

---

**任务生成完成时间**: 2025-10-28
**基于规格**: `.claude/specs/010-k8s-pod-009/spec.md`（44 个功能需求）
**预计实施时间**: 28-39 小时（单人）或 8-12 小时（3 人并行）
