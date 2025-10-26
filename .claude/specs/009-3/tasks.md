# 任务：统一终端录制系统

**输入**：来自 `.claude/specs/009-3/` 的设计文档
**前提条件**：plan.md、research.md、data-model.md、contracts/recording-api.yaml、quickstart.md

## 执行流程（main）
```
1. 从功能目录加载 plan.md
   ✓ 提取：Go 1.24.3, Gin, GORM, MinIO, robfig/cron, testify
2. 加载设计文档：
   ✓ data-model.md：3 个实体（TerminalRecording, RecordingStorageConfig, RecordingStatistics）
   ✓ contracts/：9 个 API 端点
   ✓ research.md：8 个技术决策
   ✓ quickstart.md：10 个测试场景
3. 生成任务：
   ✓ 设置：项目配置、依赖、迁移脚本
   ✓ 测试：9 个契约测试、10 个集成测试
   ✓ 核心：模型扩展、Repository、服务层、API 处理器、前端
   ✓ 集成：终端处理器集成、Cron 任务、路由
   ✓ 优化：单元测试、性能测试、文档
4. 应用任务规则：
   ✓ 不同文件 = [P] 可并行
   ✓ 同一文件 = 顺序执行
   ✓ TDD：测试在实现之前
5. 任务编号：T001-T066
6. 生成依赖关系图
7. 创建并行执行示例
8. 验证完整性：所有契约有测试，所有实体有模型
```

## 格式：`[编号] [P?] 描述`
- **[P]**：可以并行运行（不同文件，无依赖关系）
- 所有路径相对于仓库根目录

## 路径约定
- **Go 后端**：`internal/` 架构（models, repository, services, api/handlers）
- **前端**：`ui/src/` React TypeScript
- **测试**：`tests/` 三层（contract, integration, unit）
- **配置**：`internal/config/config.go`（扩展 RecordingConfig）

---

## 阶段 3.1：设置（T001-T004）

- [x] **T001** [P] 创建录制服务目录结构
  - 路径：`internal/services/recording/` 目录
  - 创建空目录：`storage_service.go`、`cleanup_service.go`、`manager_service.go` 占位符
  - 前置条件：无

- [x] **T002** [P] 扩展配置结构添加 RecordingConfig
  - 路径：`internal/config/config.go`
  - 添加 `RecordingConfig` 结构体（storage_type, base_path, retention_days, cleanup_schedule, MinIOConfig）
  - 参考：research.md "存储路径配置统一"章节
  - 前置条件：无

- [x] **T003** [P] 创建数据库迁移脚本
  - 路径：`internal/db/migrations.go`（新增 `MigrateTerminalRecordings` 函数）
  - SQL：添加 `recording_type`, `type_metadata`, `storage_type`, `tags` 字段
  - 创建索引：`idx_terminal_recordings_type`, `idx_terminal_recordings_cleanup`
  - 参考：data-model.md "数据迁移脚本"章节
  - 前置条件：无

- [x] **T004** [P] 配置 Go lint 工具
  - 路径：`.golangci.yml`（如不存在则创建）
  - 确保启用 gofmt, goimports, govet, staticcheck
  - 前置条件：无

---

## 阶段 3.2：测试优先（TDD）⚠️ 必须在 3.3 之前完成
**关键：这些测试必须编写并且必须在任何实现之前失败**

### 契约测试（T005-T013）[P] 可并行

- [x] **T005** [P] 契约测试：GET /recordings（录制列表）
  - 路径：`tests/contract/recording_list_test.go`
  - 测试分页、过滤（recording_type, user_id, time range）、排序
  - 断言：HTTP 200、JSON schema、pagination 结构
  - 参考：contracts/recording-api.yaml `listRecordings` 定义
  - 前置条件：无

- [x] **T006** [P] 契约测试：GET /recordings/:id（录制详情）
  - 路径：`tests/contract/recording_detail_test.go`
  - 测试有效 ID、无效 ID、权限控制
  - 断言：HTTP 200/404、RecordingDetail schema、type_metadata JSONB
  - 参考：contracts/recording-api.yaml `getRecording` 定义
  - 前置条件：无

- [x] **T007** [P] 契约测试：DELETE /recordings/:id（删除录制）
  - 路径：`tests/contract/recording_delete_test.go`
  - 测试删除成功、权限拒绝、不存在资源
  - 断言：HTTP 200/403/404、数据库记录删除、文件删除
  - 参考：contracts/recording-api.yaml `deleteRecording` 定义
  - 前置条件：无

- [x] **T008** [P] 契约测试：GET /recordings/search（搜索录制）
  - 路径：`tests/contract/recording_search_test.go`
  - 测试关键词搜索（用户名、描述、标签）
  - 断言：HTTP 200、搜索结果准确性、分页
  - 参考：contracts/recording-api.yaml `searchRecordings` 定义
  - 前置条件：无

- [x] **T009** [P] 契约测试：GET /recordings/statistics（统计信息）
  - 路径：`tests/contract/recording_statistics_test.go`
  - 测试总数、按类型分组、Top 用户
  - 断言：HTTP 200、RecordingStatistics schema、聚合准确性
  - 参考：contracts/recording-api.yaml `getStatistics` 定义
  - 前置条件：无

- [x] **T010** [P] 契约测试：GET /recordings/:id/playback（回放内容）
  - 路径：`tests/contract/recording_playback_test.go`
  - 测试 Asciinema v2 格式输出
  - 断言：HTTP 200、第一行 JSON header、剩余行为帧数组
  - 参考：contracts/recording-api.yaml `getPlaybackContent` 定义
  - 前置条件：无

- [x] **T011** [P] 契约测试：GET /recordings/:id/download（下载录制）
  - 路径：`tests/contract/recording_download_test.go`
  - 测试文件下载、Content-Disposition header
  - 断言：HTTP 200、application/octet-stream、文件完整性
  - 参考：contracts/recording-api.yaml `downloadRecording` 定义
  - 前置条件：无

- [x] **T012** [P] 契约测试：POST /recordings/cleanup/trigger（触发清理）
  - 路径：`tests/contract/recording_cleanup_trigger_test.go`
  - 测试管理员权限、异步任务启动
  - 断言：HTTP 202、task_id 返回、非管理员 403
  - 参考：contracts/recording-api.yaml `triggerCleanup` 定义
  - 前置条件：无

- [x] **T013** [P] 契约测试：GET /recordings/cleanup/status（清理状态）
  - 路径：`tests/contract/recording_cleanup_status_test.go`
  - 测试状态查询、统计信息
  - 断言：HTTP 200、CleanupStatus schema、last_run_at 时间戳
  - 参考：contracts/recording-api.yaml `getCleanupStatus` 定义
  - 前置条件：无

### 集成测试（T014-T023）[P] 可并行

- [ ] **T014** [P] 集成测试：场景 1 - Docker 容器终端录制（向后兼容）
  - 路径：`tests/integration/docker_recording_test.go`
  - 测试：创建 Docker 终端会话 → WebSocket 连接 → 录制文件生成
  - 验证：recording_type='docker'、type_metadata 包含 instance_id/container_id
  - 参考：quickstart.md 场景 1
  - 前置条件：无（使用 testcontainers-go）

- [ ] **T015** [P] 集成测试：场景 2 - WebSSH 终端录制
  - 路径：`tests/integration/webssh_recording_test.go`
  - 测试：创建 WebSSH 会话 → 录制自动创建 → 验证 type_metadata
  - 验证：recording_type='webssh'、host_id/ssh_port 字段
  - 参考：quickstart.md 场景 2
  - 前置条件：无

- [ ] **T016** [P] 集成测试：场景 3 - K8s 节点终端录制
  - 路径：`tests/integration/k8s_recording_test.go`
  - 测试：创建 K8s 节点终端 → 录制创建 → 验证 cluster_id/node_name
  - 验证：recording_type='k8s_node'、type_metadata JSONB 结构
  - 参考：quickstart.md 场景 3
  - 前置条件：无

- [ ] **T017** [P] 集成测试：场景 4 - 统一录制管理界面
  - 路径：`tests/integration/recording_ui_test.go`
  - 测试：列表 API → 类型过滤 → 用户过滤 → 搜索功能
  - 验证：所有类型显示、过滤准确、搜索结果正确
  - 参考：quickstart.md 场景 4
  - 前置条件：T005, T008（契约测试通过）

- [ ] **T018** [P] 集成测试：场景 5 - 录制回放
  - 路径：`tests/integration/recording_playback_test.go`
  - 测试：获取回放内容 → 验证 Asciinema v2 格式 → 前端播放器加载
  - 验证：header JSON、帧格式、播放控制
  - 参考：quickstart.md 场景 5
  - 前置条件：T010（回放契约测试通过）

- [ ] **T019** [P] 集成测试：场景 6 - 自动清理任务
  - 路径：`tests/integration/recording_cleanup_test.go`
  - 测试：插入过期录制 → 触发清理 → 验证删除
  - 验证：过期录制删除、无效录制删除、清理统计正确
  - 参考：quickstart.md 场景 6
  - 前置条件：T012, T013（清理契约测试通过）

- [ ] **T020** [P] 集成测试：场景 7 - 定时清理任务（Cron）
  - 路径：`tests/integration/recording_cron_test.go`
  - 测试：配置 Cron → 等待执行 → 验证日志和指标
  - 验证：Cron 按计划执行、Prometheus 指标更新
  - 参考：quickstart.md 场景 7
  - 前置条件：无

- [ ] **T021** [P] 集成测试：场景 8 - MinIO 对象存储（可选）
  - 路径：`tests/integration/recording_minio_test.go`
  - 测试：配置 MinIO → 创建录制 → 验证对象上传
  - 验证：文件上传成功、下载正确、清理删除对象
  - 参考：quickstart.md 场景 8
  - 前置条件：无（使用 testcontainers MinIO）
  - **标记**：可选功能（Phase 2）

- [ ] **T022** [P] 集成测试：场景 9 - 并发录制性能测试
  - 路径：`tests/integration/recording_performance_test.go`
  - 测试：100 并发终端会话 → 验证性能指标
  - 验证：连接成功率 >99%、P99 延迟 <100ms、无内存泄漏
  - 参考：quickstart.md 场景 9
  - 前置条件：无（性能测试）

- [ ] **T023** [P] 集成测试：场景 10 - 数据迁移验证
  - 路径：`tests/integration/recording_migration_test.go`
  - 测试：创建旧格式数据 → 启动应用 → 验证迁移结果
  - 验证：recording_type='docker'、type_metadata 填充、旧字段保留
  - 参考：quickstart.md 场景 10
  - 前置条件：T003（迁移脚本完成）

---

## 阶段 3.3：核心实现（仅在测试失败后）

### 数据模型和迁移（T024-T026）[P] 可并行

- [x] **T024** [P] 扩展 TerminalRecording 模型
  - 路径：`internal/models/terminal_recording.go`
  - 添加字段：RecordingType, TypeMetadata (JSONB), StorageType, Tags
  - 添加方法：IsExpired(), IsInvalid(), 验证规则
  - 标记旧字段：InstanceID, ContainerID 添加 `@Deprecated` 注释
  - 参考：data-model.md "TerminalRecording" 结构体定义
  - 前置条件：无

- [x] **T025** [P] 实现数据迁移逻辑
  - 路径：`internal/db/migrations.go`（实现 T003 的 `MigrateTerminalRecordings` 函数）
  - 逻辑：检测未迁移记录 → 批量更新 recording_type + type_metadata
  - 添加日志：记录迁移数量和错误
  - 参考：data-model.md "数据迁移脚本" Go 代码示例
  - 前置条件：T003（迁移脚本结构创建）
  - **注**: 已简化为仅创建索引，无需复杂数据迁移（项目未发布）
  - **简化**: 已移除独立的迁移文件，复合索引通过模型 AfterMigrate 钩子创建

- [x] **T026** [P] 在应用启动时调用迁移
  - 路径：`internal/db/database.go`（在 `AutoMigrate` 方法中）
  - 添加：`MigrateTerminalRecordings(d.DB)` 调用
  - 错误处理：迁移失败时记录错误但不阻止启动
  - 前置条件：T025（迁移逻辑实现）

### Repository 层（T027-T029）[P] 可并行

- [x] **T027** [P] 创建 RecordingRepository 接口
  - 路径：`internal/repository/terminal_recording_repo.go`
  - 定义接口方法：Create, GetByID, GetBySessionID, List, ListByType, FindExpired, FindInvalid, Search, Delete, BulkDelete, GetStatistics
  - 参考：data-model.md "数据访问模式" 章节
  - 前置条件：T024（模型扩展完成）

- [x] **T028** [P] 实现 RecordingRepository CRUD 方法
  - 路径：`internal/repository/terminal_recording_repo.go`（同文件，接口实现）
  - 实现：Create, GetByID, GetBySessionID, Delete, BulkDelete
  - 使用 GORM：处理 JSONB 查询、错误处理
  - 前置条件：T027（接口定义）

- [x] **T029** 实现 RecordingRepository 查询和统计方法
  - 路径：`internal/repository/terminal_recording_repo.go`（续 T028）
  - 实现：List（分页、过滤、排序）、ListByType、FindExpired、FindInvalid、Search、GetStatistics
  - 优化：使用索引、JSONB 查询优化、聚合查询
  - 参考：data-model.md "性能考虑" 章节
  - 前置条件：T028（CRUD 方法完成）

### 服务层（T030-T035）

- [ ] **T030** [P] 实现 StorageService（本地文件系统）
  - 路径：`internal/services/recording/storage_service.go`
  - 接口：WriteRecording(recordingID, data), ReadRecording(path), DeleteRecording(path)
  - 实现：日期分区目录（YYYY-MM-DD）、文件写入、删除
  - 参考：research.md "录制文件组织结构" 章节
  - 前置条件：T002（RecordingConfig 配置）

- [ ] **T031** [P] 实现 CleanupService 核心逻辑
  - 路径：`internal/services/recording/cleanup_service.go`
  - 方法：CleanupInvalidRecordings(), CleanupExpiredRecordings(), CleanupOrphanFiles()
  - 优化：批量删除（1000 条/批）、并行文件删除（10 workers）
  - 参考：research.md "清理任务性能优化" 章节
  - 前置条件：T027（Repository 接口），T030（StorageService）

- [ ] **T032** 实现 CleanupService Run 方法
  - 路径：`internal/services/recording/cleanup_service.go`（续 T031）
  - 方法：Run(ctx) 统一清理入口
  - 逻辑：清理无效 → 清理过期 → 清理孤儿 → 记录指标
  - 日志：记录清理统计和错误
  - 前置条件：T031（核心逻辑完成）

- [ ] **T033** [P] 实现 ManagerService CRUD
  - 路径：`internal/services/recording/manager_service.go`
  - 方法：ListRecordings(分页、过滤), GetRecording(ID), DeleteRecording(ID), SearchRecordings(query)
  - 业务逻辑：权限检查、RBAC 集成、审计日志
  - 前置条件：T027（Repository 接口）

- [ ] **T034** 实现 ManagerService 统计和回放
  - 路径：`internal/services/recording/manager_service.go`（续 T033）
  - 方法：GetStatistics(), GetPlaybackContent(ID), DownloadRecording(ID)
  - 回放：读取 .cast 文件、解析 Asciinema 格式、文件流
  - 前置条件：T033（CRUD 完成），T030（StorageService）

- [ ] **T035** 实现 MinIO StorageService（可选 - Phase 2）
  - 路径：`internal/services/recording/minio_storage_service.go`
  - 接口：实现与 StorageService 相同的接口
  - MinIO：上传对象、下载对象、删除对象、bucket 管理
  - 参考：research.md "MinIO 存储抽象必要性评估" 章节
  - 前置条件：T030（本地存储完成）
  - **标记**：可选功能

### API 处理器（T036-T041）

- [ ] **T036** [P] 实现 RecordingHandler 基础结构
  - 路径：`internal/api/handlers/recording/recording_handler.go`
  - 结构体：RecordingHandler（依赖 ManagerService, JWTManager）
  - 构造函数：NewRecordingHandler
  - 前置条件：T033（ManagerService CRUD）

- [ ] **T037** 实现 RecordingHandler 列表和详情
  - 路径：`internal/api/handlers/recording/recording_handler.go`（续 T036）
  - 方法：ListRecordings (GET /recordings), GetRecording (GET /recordings/:id)
  - 响应格式：使用 SendSuccess/SendError 统一格式
  - Swagger 注释：添加完整的 @Summary, @Tags, @Param, @Success, @Failure
  - 前置条件：T036（基础结构）

- [ ] **T038** 实现 RecordingHandler 删除、搜索、统计
  - 路径：`internal/api/handlers/recording/recording_handler.go`（续 T037）
  - 方法：DeleteRecording (DELETE /recordings/:id), SearchRecordings (GET /recordings/search), GetStatistics (GET /recordings/statistics)
  - RBAC：删除需要权限检查
  - 审计日志：记录删除操作
  - 前置条件：T037（列表和详情完成）

- [ ] **T039** [P] 实现 PlaybackHandler
  - 路径：`internal/api/handlers/recording/playback_handler.go`
  - 方法：GetPlaybackContent (GET /recordings/:id/playback), DownloadRecording (GET /recordings/:id/download)
  - 回放：返回 Asciinema v2 JSON 格式
  - 下载：设置 Content-Disposition header、文件流
  - 前置条件：T034（ManagerService 回放完成）

- [ ] **T040** [P] 实现 CleanupHandler
  - 路径：`internal/api/handlers/recording/cleanup_handler.go`
  - 方法：TriggerCleanup (POST /recordings/cleanup/trigger), GetCleanupStatus (GET /recordings/cleanup/status)
  - 异步触发：使用 goroutine 启动清理任务
  - 权限：仅管理员可触发
  - 前置条件：T032（CleanupService 完成）

- [ ] **T041** 注册录制 API 路由
  - 路径：`internal/api/routes.go`
  - 添加路由组：`/api/v1/recordings`
  - 注册处理器：ListRecordings, GetRecording, DeleteRecording, SearchRecordings, GetStatistics, GetPlaybackContent, DownloadRecording, TriggerCleanup, GetCleanupStatus
  - 中间件：Auth, RBAC（删除和清理需要权限）
  - 前置条件：T038, T039, T040（所有处理器完成）

### 前端实现（T042-T047）[P] 可并行

- [ ] **T042** [P] 创建录制 API 客户端
  - 路径：`ui/src/services/recording-service.ts`
  - 方法：listRecordings, getRecording, deleteRecording, searchRecordings, getStatistics, getPlaybackContent, downloadRecording, triggerCleanup, getCleanupStatus
  - 使用 Axios：错误处理、token 认证、类型定义
  - 前置条件：无

- [ ] **T043** [P] 创建录制列表页
  - 路径：`ui/src/pages/recordings/recording-list-page.tsx`
  - 功能：表格显示、分页、类型过滤、用户过滤、搜索、删除按钮
  - 状态管理：TanStack Query（`useRecordings` hook）
  - UI 组件：复用现有 Table, Pagination, Filter 组件
  - 前置条件：T042（API 客户端）

- [ ] **T044** [P] 创建录制详情页
  - 路径：`ui/src/pages/recordings/recording-detail-page.tsx`
  - 功能：显示完整元数据、TypeMetadata JSON 格式化、操作按钮（播放、下载、删除）
  - 路由：`/recordings/:id`
  - 前置条件：T042（API 客户端）

- [ ] **T045** [P] 创建 Asciinema 播放器组件
  - 路径：`ui/src/components/recording/asciinema-player.tsx`
  - 依赖：asciinema-player npm 包
  - 功能：加载 .cast 文件、播放控制、速度调节
  - Props：recordingId（通过 API 获取回放内容）
  - 前置条件：无（独立组件）

- [ ] **T046** [P] 创建录制播放页
  - 路径：`ui/src/pages/recordings/recording-player-page.tsx`
  - 功能：嵌入 AsciinemaPlayer 组件、元数据侧边栏
  - 路由：`/recordings/:id/player`
  - 前置条件：T045（播放器组件）

- [ ] **T047** 添加录制路由和导航
  - 路径：`ui/src/App.tsx`（路由定义）、`ui/src/layouts/main-layout.tsx`（导航菜单）
  - 路由：`/recordings`, `/recordings/:id`, `/recordings/:id/player`
  - 导航：添加"终端录制"菜单项到主导航
  - 前置条件：T043, T044, T046（所有页面完成）

---

## 阶段 3.4：集成

### 终端处理器集成（T048-T050）

- [ ] **T048** 集成 Docker 终端录制到统一系统
  - 路径：`internal/api/handlers/docker/terminal_handler.go`
  - 修改：finalizeRecording 方法调用统一 RecordingRepository
  - 设置：recording_type='docker', type_metadata JSONB（instance_id, container_id）
  - 向后兼容：保留旧字段 InstanceID, ContainerID
  - 参考：research.md "数据迁移策略" 章节
  - 前置条件：T024（模型扩展），T027（Repository 接口）

- [ ] **T049** 集成 WebSSH 终端录制到统一系统
  - 路径：`internal/services/webssh/recorder.go`
  - 修改：SessionRecorder 使用统一 TerminalRecording 模型
  - 设置：recording_type='webssh', type_metadata（host_id, ssh_port）
  - 统一路径：使用 RecordingConfig.BasePath
  - 前置条件：T024（模型扩展），T027（Repository 接口）

- [ ] **T050** 集成 K8s 终端录制到统一系统
  - 路径：`pkg/kube/terminal.go`（或相关 K8s 终端处理文件）
  - 修改：创建录制时使用统一 TerminalRecording 模型
  - 设置：recording_type='k8s_node' 或 'k8s_pod', type_metadata（cluster_id, node_name）
  - 前置条件：T024（模型扩展），T027（Repository 接口）

### Cron 任务集成（T051-T052）

- [ ] **T051** 创建 RecordingCleanupTask 调度器任务
  - 路径：`internal/services/recording/cleanup_task.go`
  - 实现：scheduler.Task 接口（Run, Name 方法）
  - 调用：CleanupService.Run(ctx)
  - 日志：记录执行结果
  - 前置条件：T032（CleanupService Run 方法）

- [ ] **T052** 注册 RecordingCleanupTask 到调度器
  - 路径：`internal/app/app.go`（在 `Initialize` 方法中）
  - 添加：`scheduler.AddTask("0 4 * * *", NewRecordingCleanupTask(cleanupService))`
  - 读取配置：使用 RecordingConfig.CleanupSchedule（默认 "0 4 * * *"）
  - 前置条件：T051（调度器任务创建），T002（RecordingConfig）

### Prometheus 指标（T053）

- [ ] **T053** [P] 添加录制系统 Prometheus 指标
  - 路径：`pkg/prometheus/metrics.go`
  - 指标：
    - `recording_total_count`（按 recording_type 分组）
    - `recording_total_size_bytes`
    - `recording_cleanup_runs_total`
    - `recording_cleanup_deleted_total`（按原因分组：expired, invalid, orphan）
    - `recording_cleanup_last_run_timestamp`
  - 注册：在 CleanupService 中调用 Prometheus 更新
  - 前置条件：无（独立指标定义）

---

## 阶段 3.5：优化

### 单元测试（T054-T058）[P] 可并行

- [ ] **T054** [P] RecordingRepository 单元测试
  - 路径：`tests/unit/recording_repository_test.go`
  - 测试：CRUD 方法、查询方法、统计方法
  - Mock：使用 sqlite in-memory 数据库
  - 覆盖率：>80%
  - 前置条件：T027-T029（Repository 实现）

- [ ] **T055** [P] StorageService 单元测试
  - 路径：`tests/unit/storage_service_test.go`
  - 测试：WriteRecording, ReadRecording, DeleteRecording
  - Mock：使用临时目录
  - 验证：日期分区、文件完整性
  - 前置条件：T030（StorageService 实现）

- [ ] **T056** [P] CleanupService 单元测试
  - 路径：`tests/unit/cleanup_service_test.go`
  - 测试：CleanupInvalidRecordings, CleanupExpiredRecordings, CleanupOrphanFiles
  - Mock：Repository 接口 + 临时文件
  - 验证：批量删除、并行文件操作
  - 前置条件：T031-T032（CleanupService 实现）

- [ ] **T057** [P] ManagerService 单元测试
  - 路径：`tests/unit/manager_service_test.go`
  - 测试：CRUD 方法、搜索、统计、回放
  - Mock：Repository 接口
  - 验证：业务逻辑、权限检查
  - 前置条件：T033-T034（ManagerService 实现）

- [ ] **T058** [P] 数据模型验证单元测试
  - 路径：`tests/unit/terminal_recording_model_test.go`
  - 测试：IsExpired, IsInvalid, 验证规则
  - 验证：TypeMetadata JSONB 解析、时间计算
  - 前置条件：T024（模型扩展）

### 性能测试（T059-T061）[P] 可并行

- [ ] **T059** [P] 录制写入性能基准测试
  - 路径：`tests/benchmark/recording_write_bench_test.go`
  - 基准测试：recordFrame 函数性能
  - 目标：<10ms/frame（实际预期 <1ms）
  - 工具：Go benchmark + pprof
  - 参考：research.md "并发录制写入性能" 章节
  - 前置条件：T048（Docker 录制集成）

- [ ] **T060** [P] 清理任务性能测试
  - 路径：`tests/benchmark/cleanup_performance_test.go`
  - 测试：10k 录制清理时间
  - 目标：<5 分钟（预期 <3 分钟）
  - 验证：批量删除、并行文件操作性能
  - 参考：research.md "清理任务性能优化" 章节
  - 前置条件：T031-T032（CleanupService 实现）

- [ ] **T061** [P] API 响应时间性能测试
  - 路径：`tests/benchmark/api_performance_test.go`
  - 测试：GET /recordings P99 响应时间
  - 目标：<200ms
  - 工具：ab 或 k6 负载测试
  - 前置条件：T037-T038（RecordingHandler 实现）

### 文档和 Swagger（T062-T064）[P] 可并行

- [ ] **T062** [P] 生成 Swagger 文档
  - 路径：运行 `./scripts/generate-swagger.sh`
  - 验证：`/swagger/index.html` 包含所有录制 API 端点
  - 检查：所有处理器的 Swagger 注释完整
  - 前置条件：T037-T040（所有处理器实现）

- [ ] **T063** [P] 更新 CLAUDE.md 项目文档
  - 路径：`CLAUDE.md`（已在设计阶段更新）
  - 验证：统一终端录制系统章节信息准确
  - 添加：使用示例、故障排查指南
  - 前置条件：所有实现完成

- [ ] **T064** [P] 创建用户文档
  - 路径：`docs/terminal-recording.md`（新建）
  - 内容：功能介绍、配置指南、API 使用、故障排查
  - 参考：quickstart.md 场景
  - 前置条件：所有实现完成

### 代码质量和清理（T065-T066）

- [ ] **T065** 运行 lint 并修复问题
  - 命令：`task lint`
  - 修复：gofmt, goimports, govet, staticcheck 报告的问题
  - 验证：零警告和错误
  - 前置条件：所有实现完成

- [ ] **T066** 移除重复代码和废弃代码
  - 废弃：`internal/services/docker/recording_cleanup_service.go`（标记为 @Deprecated）
  - 迁移逻辑到统一 CleanupService 后可选删除
  - 清理：未使用的导入、注释掉的代码
  - 前置条件：T031-T032（统一清理服务完成）

---

## 依赖关系

### 关键路径（Critical Path）
```
设置（T001-T004）
  → 契约测试（T005-T013）[可并行]
  → 模型扩展（T024）
  → Repository（T027-T029）
  → 服务层（T030-T034）
  → API 处理器（T036-T041）
  → 前端（T042-T047）[可并行]
  → 集成（T048-T053）
  → 优化（T054-T066）[可并行]
```

### 详细依赖
- **T005-T013** 不依赖任何实现（TDD）
- **T014-T023** 依赖契约测试通过
- **T024** 阻塞 T027（Repository 需要模型定义）
- **T027** 阻塞 T028-T029, T030-T034（服务层需要 Repository）
- **T033-T034** 阻塞 T036-T041（处理器需要 ManagerService）
- **T036-T041** 阻塞 T042（前端需要 API）
- **T002** 阻塞 T030, T052（配置需要先定义）
- **T048-T050** 需要 T024, T027（模型和 Repository）
- **T051** 依赖 T032（CleanupService）
- **所有优化任务** 依赖对应实现完成

---

## 并行执行示例

### 第 1 批：设置和配置（T001-T004）[P]
```bash
# 4 个任务可并行
Task: "创建录制服务目录结构" --subagent_type=code
Task: "扩展配置结构添加 RecordingConfig" --subagent_type=code
Task: "创建数据库迁移脚本" --subagent_type=code
Task: "配置 Go lint 工具" --subagent_type=code
```

### 第 2 批：契约测试（T005-T013）[P] ⚠️ TDD
```bash
# 9 个契约测试可并行
Task: "契约测试：GET /recordings（录制列表）" --subagent_type=requirements-testing
Task: "契约测试：GET /recordings/:id（录制详情）" --subagent_type=requirements-testing
Task: "契约测试：DELETE /recordings/:id（删除录制）" --subagent_type=requirements-testing
Task: "契约测试：GET /recordings/search（搜索录制）" --subagent_type=requirements-testing
Task: "契约测试：GET /recordings/statistics（统计信息）" --subagent_type=requirements-testing
Task: "契约测试：GET /recordings/:id/playback（回放内容）" --subagent_type=requirements-testing
Task: "契约测试：GET /recordings/:id/download（下载录制）" --subagent_type=requirements-testing
Task: "契约测试：POST /recordings/cleanup/trigger（触发清理）" --subagent_type=requirements-testing
Task: "契约测试：GET /recordings/cleanup/status（清理状态）" --subagent_type=requirements-testing
```

### 第 3 批：集成测试（T014-T023）[P]
```bash
# 10 个集成测试可并行
Task: "集成测试：场景 1 - Docker 容器终端录制（向后兼容）" --subagent_type=requirements-testing
Task: "集成测试：场景 2 - WebSSH 终端录制" --subagent_type=requirements-testing
Task: "集成测试：场景 3 - K8s 节点终端录制" --subagent_type=requirements-testing
# ... 其余 7 个场景
```

### 第 4 批：模型和迁移（T024-T026）[P]
```bash
# 3 个任务可并行（T026 稍后依赖 T025）
Task: "扩展 TerminalRecording 模型" --subagent_type=code
Task: "实现数据迁移逻辑" --subagent_type=code
# T026 需要等 T025 完成
```

### 第 5 批：Repository 和服务层（T027-T034）
```bash
# T027 单独执行（定义接口）
Task: "创建 RecordingRepository 接口" --subagent_type=code

# T028-T029 顺序（同文件）
Task: "实现 RecordingRepository CRUD 方法" --subagent_type=code
Task: "实现 RecordingRepository 查询和统计方法" --subagent_type=code

# T030-T032 可并行（不同文件）
Task: "实现 StorageService（本地文件系统）" --subagent_type=code
Task: "实现 CleanupService 核心逻辑" --subagent_type=code
Task: "实现 ManagerService CRUD" --subagent_type=code
```

### 第 6 批：前端（T042-T047）[P]
```bash
# 6 个前端任务可并行（不同文件）
Task: "创建录制 API 客户端" --subagent_type=frontend
Task: "创建录制列表页" --subagent_type=frontend
Task: "创建录制详情页" --subagent_type=frontend
Task: "创建 Asciinema 播放器组件" --subagent_type=frontend
Task: "创建录制播放页" --subagent_type=frontend
# T047 依赖上述完成
```

### 第 7 批：优化（T054-T064）[P]
```bash
# 单元测试、性能测试、文档可并行
Task: "RecordingRepository 单元测试" --subagent_type=requirements-testing
Task: "StorageService 单元测试" --subagent_type=requirements-testing
Task: "CleanupService 单元测试" --subagent_type=requirements-testing
Task: "录制写入性能基准测试" --subagent_type=performance
Task: "清理任务性能测试" --subagent_type=performance
Task: "生成 Swagger 文档" --subagent_type=code
Task: "更新 CLAUDE.md 项目文档" --subagent_type=code
Task: "创建用户文档" --subagent_type=code
```

---

## 验证清单

### 功能完整性
- [x] 所有契约都有对应测试（9/9）
- [x] 所有实体都有模型任务（TerminalRecording 扩展）
- [x] 所有测试都在实现之前（阶段 3.2 → 阶段 3.3）
- [x] 并行任务真正独立（不同文件或无依赖）
- [x] 每个任务指定确切文件路径
- [x] 没有任务修改与另一个 [P] 任务相同的文件

### 测试覆盖
- [x] 契约测试：9 个（所有 API 端点）
- [x] 集成测试：10 个（所有 quickstart 场景）
- [x] 单元测试：5 个（Repository, Services, Model）
- [x] 性能测试：3 个（写入、清理、API 响应）

### 质量保证
- [x] Lint 配置（T004）
- [x] 代码审查清单（T065-T066）
- [x] 文档完整性（T062-T064）
- [x] Swagger 自动生成（T062）

---

## 任务执行顺序摘要

1. **设置阶段**（1-4 天）：T001-T004
2. **TDD 测试编写**（3-5 天）：T005-T023（必须失败）
3. **核心实现**（10-15 天）：T024-T047
4. **集成**（3-5 天）：T048-T053
5. **优化和文档**（3-5 天）：T054-T066

**总预估**：20-34 工作日（4-7 周）

---

**任务生成完成时间**：2025-10-26
**总任务数**：66 个
**可并行任务数**：43 个（65%）
**关键路径长度**：23 个任务（串行）

**下一步**：选择任务执行方式：
1. 手动执行：按顺序勾选任务
2. 并行执行：使用上述并行示例启动多个 Task 代理
3. 自动执行：使用 `/spec-kit:implement` 命令（如果支持）
