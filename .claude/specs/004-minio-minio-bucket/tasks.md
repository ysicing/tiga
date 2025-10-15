# 任务：MinIO 对象存储管理系统

**功能分支**：`004-minio-minio-bucket` | **日期**：2025-10-14
**输入**：来自 `.claude/specs/004-minio-minio-bucket/` 的设计文档
**前提条件**：plan.md ✅、research.md ✅、data-model.md ✅、contracts/ ✅

## 执行流程（已完成）
```
1. 从功能目录加载 plan.md ✅
   → 技术栈：Go 1.24.3、Gin、GORM、MinIO SDK v7、React 19、TypeScript 5.x
   → 项目结构：Web 应用（后端 internal/ + 前端 ui/src/）
2. 加载设计文档 ✅
   → data-model.md：5个实体（MinIOInstance、MinIOUser、BucketPermission、ShareLink、AuditLog）
   → contracts/：1个契约文件（minio-instance.yaml）+ 需生成5个契约
   → research.md：技术选型完成（MinIO SDK、tus-js-client、Monaco Editor等）
3. 生成任务类别 ✅
   → 设置：4个任务
   → 测试优先：15个任务（6个契约测试 + 9个集成测试）
   → 核心实现：31个任务（模型、Repository、Service、Handler、工具）
   → 前端实现：14个任务（API客户端、页面、组件）
   → 优化：6个任务
4. 应用任务规则 ✅
   → 不同文件 = [P] 并行标记
   → 同一文件 = 顺序执行
   → 测试在实现之前（TDD）
5. 任务总数：70个
6. 依赖关系已定义 ✅
7. 并行执行示例已生成 ✅
8. 验证完整性 ✅
```

**任务状态统计**：
- 总任务数：70
- 可并行任务：28个（标记 [P]）
- 顺序任务：42个

---

## 阶段 3.1：设置（4个任务）

- [ ] **T001** 创建 MinIO 功能目录结构
  - **路径**：`internal/models/`, `internal/repository/minio/`, `internal/services/minio/`, `internal/api/handlers/minio/`, `pkg/minio/`
  - **说明**：创建后端目录结构，复用现有架构模式
  - **依赖**：无

- [ ] **T002** 安装 MinIO Go SDK 和相关依赖
  - **命令**：`go get github.com/minio/minio-go/v7`
  - **说明**：添加 MinIO SDK、testcontainers-go（minio模块）
  - **依赖**：无

- [ ] **T003** [P] 创建前端 MinIO 功能目录结构
  - **路径**：`ui/src/pages/minio/`, `ui/src/components/minio/`, `ui/src/services/minio-api.ts`, `ui/src/hooks/`
  - **说明**：创建前端目录结构
  - **依赖**：无

- [ ] **T004** [P] 安装前端预览和上传依赖
  - **依赖**：tus-js-client, react-photo-view, @monaco-editor/react, react-markdown, remark-gfm, react-window
  - **命令**：`cd ui && pnpm add tus-js-client react-photo-view @monaco-editor/react react-markdown remark-gfm react-window`
  - **依赖**：无

---

## 阶段 3.2：测试优先（TDD）⚠️ 必须在 3.3 之前完成

### 契约测试（6个任务，全部可并行）
**关键：这些测试必须编写并且必须在任何实现之前失败**

- [ ] **T005** [P] 实例管理契约测试
  - **文件**：`tests/contract/minio/instance_contract_test.go`
  - **测试端点**：
    * POST /api/v1/minio/instances（创建实例）
    * GET /api/v1/minio/instances（列出实例）
    * GET /api/v1/minio/instances/{id}（获取详情）
    * PUT /api/v1/minio/instances/{id}（更新实例）
    * DELETE /api/v1/minio/instances/{id}（删除实例）
    * POST /api/v1/minio/instances/{id}/test（测试连接）
  - **契约**：`contracts/minio-instance.yaml`
  - **预期**：测试失败（无实现）
  - **依赖**：T002

- [ ] **T006** [P] Bucket 管理契约测试
  - **文件**：`tests/contract/minio/bucket_contract_test.go`
  - **测试端点**：
    * GET /api/v1/minio/instances/{id}/buckets
    * POST /api/v1/minio/instances/{id}/buckets
    * GET /api/v1/minio/instances/{id}/buckets/{name}
    * DELETE /api/v1/minio/instances/{id}/buckets/{name}
    * PUT /api/v1/minio/instances/{id}/buckets/{name}/policy
  - **说明**：先生成契约文件 contracts/bucket.yaml
  - **预期**：测试失败
  - **依赖**：T002

- [ ] **T007** [P] 用户管理契约测试
  - **文件**：`tests/contract/minio/user_contract_test.go`
  - **测试端点**：
    * GET /api/v1/minio/instances/{id}/users
    * POST /api/v1/minio/instances/{id}/users（验证SecretKey仅返回一次）
    * DELETE /api/v1/minio/instances/{id}/users/{username}
  - **说明**：先生成契约文件 contracts/minio-user.yaml
  - **预期**：测试失败
  - **依赖**：T002

- [ ] **T008** [P] 权限管理契约测试
  - **文件**：`tests/contract/minio/permission_contract_test.go`
  - **测试端点**：
    * POST /api/v1/minio/permissions（支持Bucket级和前缀级）
    * GET /api/v1/minio/permissions（按用户或Bucket筛选）
    * DELETE /api/v1/minio/permissions/{id}
  - **说明**：先生成契约文件 contracts/permission.yaml
  - **预期**：测试失败
  - **依赖**：T002

- [ ] **T009** [P] 文件操作契约测试
  - **文件**：`tests/contract/minio/file_contract_test.go`
  - **测试端点**：
    * GET /api/v1/minio/instances/{id}/files（列表、分页）
    * POST /api/v1/minio/instances/{id}/files（上传）
    * GET /api/v1/minio/instances/{id}/files/download（presigned URL）
    * GET /api/v1/minio/instances/{id}/files/preview（presigned URL）
    * DELETE /api/v1/minio/instances/{id}/files
  - **说明**：先生成契约文件 contracts/file-operations.yaml
  - **预期**：测试失败
  - **依赖**：T002

- [ ] **T010** [P] 分享链接契约测试
  - **文件**：`tests/contract/minio/share_contract_test.go`
  - **测试端点**：
    * POST /api/v1/minio/shares（生成presigned URL）
    * GET /api/v1/minio/shares（列出我的分享）
    * DELETE /api/v1/minio/shares/{id}（撤销分享）
  - **说明**：先生成契约文件 contracts/share-link.yaml
  - **预期**：测试失败
  - **依赖**：T002

### 集成测试环境（2个任务）

- [ ] **T011** 配置 testcontainers MinIO 环境
  - **文件**：`tests/testcontainers/minio_container.go`
  - **功能**：
    * 启动 MinIO 容器（minio/minio:latest）
    * 配置根用户（minioadmin/minioadmin）
    * 健康检查（/minio/health/live）
    * 提供客户端工厂方法
  - **参考**：research.md 第6节
  - **依赖**：T002

- [ ] **T012** 创建集成测试辅助函数
  - **文件**：`tests/integration/minio/helpers.go`
  - **功能**：
    * SetupTestData（创建测试Bucket、用户、文件）
    * CleanupTestData（清理测试数据）
    * AssertBucketExists、AssertUserExists等断言
  - **依赖**：T011

### 集成测试（7个任务，部分可并行）

- [ ] **T013** [P] 实例管理集成测试
  - **文件**：`tests/integration/minio/instance_test.go`
  - **场景**：
    * 创建实例并测试连接
    * 健康检查更新状态（online/offline/error）
    * 查询实例存储统计（TotalSize、UsedSize）
  - **依赖**：T011, T012

- [ ] **T014** [P] Bucket 管理集成测试
  - **文件**：`tests/integration/minio/bucket_test.go`
  - **场景**：
    * 创建私有Bucket
    * 修改Bucket策略（私有→公开读）
    * 删除非空Bucket（验证警告）
  - **依赖**：T011, T012

- [ ] **T015** [P] 用户和权限集成测试
  - **文件**：`tests/integration/minio/permission_test.go`
  - **场景**：
    * 创建MinIO用户（验证SecretKey仅返回一次）
    * 授予Bucket级只读权限
    * 授予前缀级读写权限（验证目录级隔离）
    * 撤销权限
  - **依赖**：T011, T012

- [ ] **T016** [P] 文件操作集成测试
  - **文件**：`tests/integration/minio/file_operations_test.go`
  - **场景**：
    * 上传文件到指定路径
    * 列出文件（验证分页）
    * 下载文件（验证presigned URL有效）
    * 删除文件
  - **依赖**：T011, T012

- [ ] **T017** [P] 分享链接集成测试
  - **文件**：`tests/integration/minio/share_test.go`
  - **场景**：
    * 生成7天有效分享链接
    * 验证未登录用户可访问
    * 验证过期链接拒绝访问
  - **依赖**：T011, T012

- [ ] **T018** [P] 审计日志集成测试
  - **文件**：`tests/integration/minio/audit_test.go`
  - **场景**：
    * 实例创建记录审计日志
    * Bucket操作记录审计日志
    * 文件上传记录审计日志
    * 审计日志查询和筛选（按时间、用户、操作类型）
  - **依赖**：T011, T012

- [ ] **T019** 安全和错误处理集成测试
  - **文件**：`tests/integration/minio/security_test.go`
  - **场景**：
    * 路径遍历攻击防护
    * 无权限用户访问拒绝
    * MinIO连接失败处理
    * 凭据加密验证
  - **依赖**：T011, T012

---

## 阶段 3.3：核心实现（仅在测试失败后）

### 数据模型（6个任务，全部可并行）

- [ ] **T020** [P] 创建 MinIOInstance 模型
  - **文件**：`internal/models/minio_instance.go`
  - **字段**：Name, Description, Endpoint, UseSSL, AccessKey, SecretKey（加密）, Status, Version, TotalSize, UsedSize, LastChecked
  - **关联**：Users, Permissions, ShareLinks, AuditLogs
  - **索引**：name, endpoint（unique）, status
  - **加密**：BeforeCreate/AfterFind hooks（复用pkg/crypto）
  - **依赖**：T002

- [ ] **T021** [P] 创建 MinIOUser 模型
  - **文件**：`internal/models/minio_user.go`
  - **字段**：InstanceID, Username, AccessKey, SecretKey（加密）, Status, Description
  - **关联**：Instance, Permissions, ShareLinks
  - **索引**：复合唯一索引（InstanceID + Username）
  - **依赖**：T002

- [ ] **T022** [P] 创建 BucketPermission 模型
  - **文件**：`internal/models/bucket_permission.go`
  - **字段**：InstanceID, UserID, BucketName, Prefix, Permission（readonly/writeonly/readwrite）, GrantedBy, Description
  - **关联**：Instance, User
  - **索引**：复合唯一索引（UserID + BucketName + Prefix）
  - **依赖**：T002

- [ ] **T023** [P] 创建 MinIOShareLink 模型
  - **文件**：`internal/models/minio_share_link.go`
  - **字段**：InstanceID, BucketName, ObjectKey, Token, ExpiresAt, Status, CreatedBy, AccessCount
  - **关联**：Instance, Creator（MinIOUser）
  - **索引**：token（unique）, expires_at, created_by
  - **依赖**：T002

- [ ] **T024** [P] 创建 MinIOAuditLog 模型
  - **文件**：`internal/models/minio_audit_log.go`
  - **字段**：InstanceID, OperationType, ResourceType, ResourceName, Action, OperatorID, OperatorName, ClientIP, Status, ErrorMessage, Details
  - **索引**：created_at, instance_id, operation_type, operator_id, resource_name
  - **清理策略**：90天（定时任务）
  - **依赖**：T002

- [ ] **T025** [P] 数据库迁移配置
  - **文件**：`internal/db/migrations.go`（更新）
  - **说明**：添加 MinIO 相关表到 AutoMigrate
  - **依赖**：T020-T024

### Repository 层（4个任务，部分可并行）

- [ ] **T026** [P] 实现 InstanceRepository
  - **文件**：`internal/repository/minio/instance_repository.go`
  - **方法**：Create, GetByID, List, Update, Delete, GetByEndpoint, UpdateStatus
  - **依赖**：T020

- [ ] **T027** [P] 实现 UserRepository
  - **文件**：`internal/repository/minio/user_repository.go`
  - **方法**：Create, GetByID, List（按实例）, Delete, GetByUsername, ListWithPermissions
  - **依赖**：T021

- [ ] **T028** [P] 实现 PermissionRepository
  - **文件**：`internal/repository/minio/permission_repository.go`
  - **方法**：Create, GetByID, List（按用户或Bucket）, Delete, GetByUserAndBucket
  - **依赖**：T022

- [ ] **T029** [P] 实现 AuditRepository
  - **文件**：`internal/repository/minio/audit_repository.go`
  - **方法**：Create, List（分页、筛选）, DeleteBefore（清理90天前）
  - **依赖**：T024

### pkg/minio 工具包（3个任务，全部可并行）

- [ ] **T030** [P] 实现 MinIO 客户端管理器
  - **文件**：`pkg/minio/client.go`
  - **功能**：
    * ClientManager（客户端缓存，sync.Map）
    * GetClient（从缓存获取或创建）
    * HealthCheck（使用ListBuckets验证连接）
    * GetServerInfo（获取版本、存储统计）
  - **参考**：research.md 第1节
  - **依赖**：T002

- [ ] **T031** [P] 实现策略生成工具
  - **文件**：`pkg/minio/policy.go`
  - **功能**：
    * GenerateBucketPolicy（Bucket级策略）
    * GeneratePrefixPolicy（前缀级策略）
    * getActionsForPermission（readonly/writeonly/readwrite → S3 Actions）
    * ValidatePolicy（语法验证）
  - **参考**：research.md 第4节
  - **依赖**：T002

- [ ] **T032** [P] 实现 Presigned URL 生成工具
  - **文件**：`pkg/minio/presigned.go`
  - **功能**：
    * GenerateDownloadURL（下载链接，7天过期）
    * GenerateUploadURL（上传链接，1小时过期）
    * GeneratePreviewURL（预览链接，15分钟过期）
    * PresignedCache（TTL缓存，避免频繁签名）
  - **参考**：research.md 第1节
  - **依赖**：T002

### Service 层（7个任务，按依赖顺序）

- [ ] **T033** 实现 InstanceManager
  - **文件**：`internal/services/minio/instance_manager.go`
  - **功能**：
    * CreateInstance（加密SecretKey）
    * GetInstance（解密SecretKey）
    * UpdateInstance
    * DeleteInstance（检查关联数据）
    * TestConnection（调用pkg/minio/client.HealthCheck）
    * RefreshStatus（后台任务，更新Status和存储统计）
  - **依赖**：T026, T030

- [ ] **T034** 实现 BucketService
  - **文件**：`internal/services/minio/bucket_service.go`
  - **功能**：
    * ListBuckets（调用MinIO SDK）
    * CreateBucket（验证名称合法性）
    * DeleteBucket（检查是否为空，警告）
    * UpdateBucketPolicy（调用pkg/minio/policy）
    * GetBucketInfo（大小、对象数量）
  - **依赖**：T030, T031

- [ ] **T035** 实现 UserService
  - **文件**：`internal/services/minio/user_service.go`
  - **功能**：
    * CreateUser（生成AccessKey/SecretKey，加密存储，MinIO Admin API）
    * ListUsers（调用MinIO Admin API）
    * DeleteUser（清理权限，MinIO Admin API）
    * GetUserPermissions（列出用户的所有权限）
  - **依赖**：T027, T030

- [ ] **T036** 实现 PermissionService
  - **文件**：`internal/services/minio/permission_service.go`
  - **功能**：
    * GrantPermission（Bucket级或前缀级，生成策略JSON，调用MinIO SetBucketPolicy）
    * RevokePermission（更新策略JSON）
    * ListPermissions（按用户或Bucket）
    * ValidatePermission（检查冲突）
  - **依赖**：T028, T031

- [ ] **T037** 实现 FileService
  - **文件**：`internal/services/minio/file_service.go`
  - **功能**：
    * ListObjects（支持前缀筛选、分页）
    * UploadFile（调用MinIO PutObject）
    * DeleteFile（调用MinIO RemoveObject）
    * GetDownloadURL（调用pkg/minio/presigned）
    * GetPreviewURL（调用pkg/minio/presigned）
    * CreateFolder（上传空对象，key以"/"结尾）
    * DeleteFolder（递归删除）
  - **依赖**：T030, T032

- [ ] **T038** 实现 ShareService
  - **文件**：`internal/services/minio/share_service.go`
  - **功能**：
    * CreateShareLink（生成presigned URL，记录到数据库）
    * ListMyShares（按创建者查询）
    * RevokeShare（更新状态为revoked）
    * CleanupExpiredShares（定时任务，删除过期记录）
  - **依赖**：T030, T032

- [ ] **T039** 实现 AuditLogger
  - **文件**：`internal/services/minio/audit_logger.go`
  - **功能**：
    * LogOperation（记录操作到数据库）
    * LogInstanceOperation（实例创建/更新/删除/测试）
    * LogBucketOperation（Bucket创建/删除/策略更新）
    * LogUserOperation（用户创建/删除）
    * LogPermissionOperation（权限授予/撤销）
    * LogFileOperation（文件上传/删除）
    * LogShareOperation（分享创建/撤销）
  - **依赖**：T029

### Handler 层（6个任务，按依赖顺序）

- [ ] **T040** 实现 InstanceHandler
  - **文件**：`internal/api/handlers/minio/instance_handler.go`
  - **端点**：
    * POST /api/v1/minio/instances
    * GET /api/v1/minio/instances
    * GET /api/v1/minio/instances/{id}
    * PUT /api/v1/minio/instances/{id}
    * DELETE /api/v1/minio/instances/{id}
    * POST /api/v1/minio/instances/{id}/test
  - **中间件**：Auth（JWT）, RBAC（仅管理员）
  - **Swagger注释**：完整API文档
  - **依赖**：T033, T039

- [ ] **T041** 实现 BucketHandler
  - **文件**：`internal/api/handlers/minio/bucket_handler.go`
  - **端点**：
    * GET /api/v1/minio/instances/{id}/buckets
    * POST /api/v1/minio/instances/{id}/buckets
    * GET /api/v1/minio/instances/{id}/buckets/{name}
    * DELETE /api/v1/minio/instances/{id}/buckets/{name}
    * PUT /api/v1/minio/instances/{id}/buckets/{name}/policy
  - **中间件**：Auth, RBAC（仅管理员）
  - **依赖**：T034, T039

- [ ] **T042** 实现 UserHandler
  - **文件**：`internal/api/handlers/minio/user_handler.go`
  - **端点**：
    * GET /api/v1/minio/instances/{id}/users
    * POST /api/v1/minio/instances/{id}/users（仅返回一次SecretKey）
    * DELETE /api/v1/minio/instances/{id}/users/{username}
  - **中间件**：Auth, RBAC（仅管理员）
  - **依赖**：T035, T039

- [ ] **T043** 实现 PermissionHandler
  - **文件**：`internal/api/handlers/minio/permission_handler.go`
  - **端点**：
    * POST /api/v1/minio/permissions
    * GET /api/v1/minio/permissions
    * DELETE /api/v1/minio/permissions/{id}
  - **中间件**：Auth, RBAC（仅管理员）
  - **依赖**：T036, T039

- [ ] **T044** 实现 FileHandler
  - **文件**：`internal/api/handlers/minio/file_handler.go`
  - **端点**：
    * GET /api/v1/minio/instances/{id}/files
    * POST /api/v1/minio/instances/{id}/files
    * GET /api/v1/minio/instances/{id}/files/download
    * GET /api/v1/minio/instances/{id}/files/preview
    * DELETE /api/v1/minio/instances/{id}/files
  - **中间件**：Auth, RBAC（权限检查）
  - **依赖**：T037, T039

- [ ] **T045** 实现 ShareHandler
  - **文件**：`internal/api/handlers/minio/share_handler.go`
  - **端点**：
    * POST /api/v1/minio/shares
    * GET /api/v1/minio/shares
    * DELETE /api/v1/minio/shares/{id}
  - **中间件**：Auth
  - **依赖**：T038, T039

### 路由注册（1个任务）

- [ ] **T046** 注册 MinIO API 路由
  - **文件**：`internal/api/routes.go`（更新）
  - **说明**：添加 MinIO 路由组，注册所有 Handler
  - **依赖**：T040-T045

---

## 阶段 3.4：前端实现（14个任务）

### API 客户端（1个任务）

- [ ] **T047** 实现 MinIO API 客户端
  - **文件**：`ui/src/services/minio-api.ts`
  - **方法**：
    * Instances: create, list, get, update, delete, test
    * Buckets: list, create, delete, updatePolicy
    * Users: list, create, delete
    * Permissions: grant, revoke, list
    * Files: list, upload, download, preview, delete
    * Shares: create, list, revoke
  - **配置**：使用 TanStack Query（react-query）
  - **依赖**：T004

### Hooks（5个任务，全部可并行）

- [ ] **T048** [P] 实现 useMinIOInstances hook
  - **文件**：`ui/src/hooks/use-minio-instances.ts`
  - **功能**：useQuery（list）, useMutation（create/update/delete/test）
  - **依赖**：T047

- [ ] **T049** [P] 实现 useBuckets hook
  - **文件**：`ui/src/hooks/use-buckets.ts`
  - **功能**：useQuery（list）, useMutation（create/delete/updatePolicy）
  - **依赖**：T047

- [ ] **T050** [P] 实现 useFiles hook
  - **文件**：`ui/src/hooks/use-files.ts`
  - **功能**：useQuery（list），无限滚动分页
  - **依赖**：T047

- [ ] **T051** [P] 实现 useUpload hook
  - **文件**：`ui/src/hooks/use-upload.ts`
  - **功能**：
    * 多文件并发上传（最多5个并发）
    * 使用 tus-js-client 实现断点续传
    * 实时进度、速度、剩余时间计算
    * 任务队列管理
  - **参考**：research.md 第2节
  - **依赖**：T004, T047

- [ ] **T052** [P] 实现 useShares hook
  - **文件**：`ui/src/hooks/use-shares.ts`
  - **功能**：useQuery（list）, useMutation（create/revoke）
  - **依赖**：T047

### 组件（5个任务，部分可并行）

- [ ] **T053** [P] 实现文件上传组件
  - **文件**：`ui/src/components/minio/file-uploader.tsx`
  - **功能**：
    * 拖拽上传、点击选择
    * 显示上传队列（进度、速度、剩余时间）
    * 暂停/恢复/取消按钮
    * 错误处理和重试
  - **依赖**：T051

- [ ] **T054** [P] 实现文件预览组件
  - **文件**：`ui/src/components/minio/file-previewer.tsx`
  - **子组件**：
    * ImageViewer（react-photo-view，缩放、旋转）
    * VideoPlayer（原生video标签，HTTP Range支持）
    * CodeViewer（Monaco Editor，语法高亮）
    * MarkdownViewer（react-markdown + remark-gfm）
  - **参考**：research.md 第3节
  - **依赖**：T004

- [ ] **T055** [P] 实现实例表单组件
  - **文件**：`ui/src/components/minio/instance-form.tsx`
  - **字段**：Name, Description, Endpoint, UseSSL, AccessKey, SecretKey
  - **验证**：实时验证（endpoint格式、必填项）
  - **依赖**：T004

- [ ] **T056** [P] 实现权限配置表单
  - **文件**：`ui/src/components/minio/permission-form.tsx`
  - **字段**：User（下拉）, Bucket（下拉）, Prefix（可选）, Permission（只读/只写/读写）
  - **说明**：支持前缀级权限配置
  - **依赖**：T004

- [ ] **T057** 实现分享对话框
  - **文件**：`ui/src/components/minio/share-dialog.tsx`
  - **功能**：
    * 选择过期时间（1小时、1天、7天、30天）
    * 生成链接
    * 一键复制到剪贴板
  - **依赖**：T004

### 页面（3个任务，顺序执行）

- [ ] **T058** 实现实例管理页面
  - **文件**：`ui/src/pages/minio/instances-page.tsx`
  - **功能**：
    * 实例列表（状态、版本、存储统计）
    * 创建/编辑/删除实例
    * 测试连接按钮
    * 点击进入实例详情
  - **依赖**：T048, T055

- [ ] **T059** 实现文件浏览器页面
  - **文件**：`ui/src/pages/minio/files-page.tsx`
  - **功能**：
    * Bucket选择器
    * 文件列表（虚拟滚动，react-window）
    * 面包屑导航
    * 文件搜索和排序
    * 上传/下载/预览/分享按钮
    * 文件夹创建/删除
  - **参考**：research.md 第5节（虚拟滚动）
  - **依赖**：T050, T051, T053, T054, T057

- [ ] **T060** 实现用户和权限管理页面
  - **文件**：`ui/src/pages/minio/users-page.tsx`
  - **功能**：
    * 用户列表
    * 创建用户（显示SecretKey警告：仅显示一次）
    * 权限配置（授予/撤销）
    * 权限列表（按用户查看）
  - **依赖**：T048, T049, T056

### 路由配置（1个任务）

- [ ] **T061** 配置前端路由
  - **文件**：`ui/src/App.tsx`（更新）
  - **路由**：
    * /minio/instances
    * /minio/instances/:id/files
    * /minio/instances/:id/users
  - **依赖**：T058-T060

---

## 阶段 3.5：优化（6个任务）

### 单元测试（3个任务，全部可并行）

- [ ] **T062** [P] 策略生成单元测试
  - **文件**：`pkg/minio/policy_test.go`
  - **测试**：
    * GenerateBucketPolicy（Bucket级）
    * GeneratePrefixPolicy（前缀级）
    * 只读/只写/读写权限映射
    * JSON格式验证
  - **依赖**：T031

- [ ] **T063** [P] Presigned URL 单元测试
  - **文件**：`pkg/minio/presigned_test.go`
  - **测试**：
    * 下载URL生成（7天过期）
    * 上传URL生成（1小时过期）
    * 预览URL生成（15分钟过期）
    * URL缓存机制
  - **依赖**：T032

- [ ] **T064** [P] 凭据加密单元测试
  - **文件**：`internal/models/minio_instance_test.go`
  - **测试**：
    * BeforeCreate加密hook
    * AfterFind解密hook
    * 加密后密文不可读
  - **依赖**：T020

### 性能和文档（3个任务）

- [ ] **T065** 性能测试和优化
  - **文件**：`tests/performance/minio_performance_test.go`
  - **目标**：
    * API响应时间 < 200ms p95（列表查询）
    * 文件元数据查询 < 500ms p95
    * 文件列表虚拟滚动（10k+对象流畅）
  - **依赖**：阶段 3.3、3.4 完成

- [ ] **T066** 生成 Swagger API 文档
  - **命令**：`./scripts/generate-swagger.sh`
  - **验证**：访问 http://localhost:12306/swagger/index.html
  - **说明**：所有 Handler 已包含 Swagger 注释
  - **依赖**：T040-T045

- [ ] **T067** 运行 quickstart.md 手动验证
  - **场景**：
    1. 管理员添加 MinIO 实例
    2. 管理员创建 Bucket 和用户
    3. 用户上传和预览文件
    4. 用户生成分享链接
    5. 管理员查看审计日志
  - **文档**：`.claude/specs/004-minio-minio-bucket/quickstart.md`
  - **依赖**：阶段 3.3、3.4 完成

### 代码质量（3个任务，全部可并行）

- [ ] **T068** [P] 运行代码检查
  - **命令**：`task lint`
  - **修复**：gofmt、goimports、golangci-lint 报告的问题
  - **依赖**：阶段 3.3 完成

- [ ] **T069** [P] 前端代码检查
  - **命令**：`cd ui && pnpm lint && pnpm format`
  - **修复**：ESLint、Prettier 报告的问题
  - **依赖**：阶段 3.4 完成

- [ ] **T070** [P] 删除重复代码和死代码
  - **说明**：使用代码审查工具检查重复代码，移除未使用的函数和变量
  - **依赖**：阶段 3.3、3.4 完成

---

## 依赖关系图

### 关键路径
```
T001-T002（设置）
  → T005-T010（契约测试）[P]
  → T011-T012（集成测试环境）
  → T020-T025（数据模型）[P]
  → T026-T029（Repository）[P]
  → T030-T032（pkg/minio工具）[P]
  → T033-T039（Service层）
  → T040-T045（Handler层）
  → T046（路由注册）
  → T047（API客户端）
  → T048-T052（Hooks）[P]
  → T053-T057（组件）[P]
  → T058-T060（页面）
  → T061（路由配置）
  → T065-T070（优化）
```

### 并行任务组
**第1组（设置）**：T001, T002, T003, T004（4个任务并行）

**第2组（契约测试）**：T005, T006, T007, T008, T009, T010（6个任务并行）

**第3组（集成测试）**：T013, T014, T015, T016, T017, T018（6个任务并行，需T011-T012完成）

**第4组（数据模型）**：T020, T021, T022, T023, T024（5个任务并行）

**第5组（Repository + 工具）**：T026, T027, T028, T029, T030, T031, T032（7个任务并行）

**第6组（Hooks）**：T048, T049, T050, T051, T052（5个任务并行）

**第7组（组件）**：T053, T054, T055, T056（4个任务并行）

**第8组（单元测试）**：T062, T063, T064（3个任务并行）

**第9组（代码质量）**：T068, T069, T070（3个任务并行）

---

## 并行执行示例

### 示例 1：同时启动契约测试（TDD）
```bash
# 使用 Task 代理工具并行执行
Task --agent requirements-code "在 tests/contract/minio/instance_contract_test.go 中测试实例管理契约" &
Task --agent requirements-code "在 tests/contract/minio/bucket_contract_test.go 中测试Bucket管理契约" &
Task --agent requirements-code "在 tests/contract/minio/user_contract_test.go 中测试用户管理契约" &
Task --agent requirements-code "在 tests/contract/minio/permission_contract_test.go 中测试权限管理契约" &
Task --agent requirements-code "在 tests/contract/minio/file_contract_test.go 中测试文件操作契约" &
Task --agent requirements-code "在 tests/contract/minio/share_contract_test.go 中测试分享链接契约"
```

### 示例 2：同时创建数据模型
```bash
Task --agent requirements-code "在 internal/models/minio_instance.go 中创建 MinIOInstance 模型" &
Task --agent requirements-code "在 internal/models/minio_user.go 中创建 MinIOUser 模型" &
Task --agent requirements-code "在 internal/models/bucket_permission.go 中创建 BucketPermission 模型" &
Task --agent requirements-code "在 internal/models/minio_share_link.go 中创建 MinIOShareLink 模型" &
Task --agent requirements-code "在 internal/models/minio_audit_log.go 中创建 MinIOAuditLog 模型"
```

### 示例 3：同时实现 Hooks
```bash
Task --agent requirements-code "在 ui/src/hooks/use-minio-instances.ts 中实现 useMinIOInstances hook" &
Task --agent requirements-code "在 ui/src/hooks/use-buckets.ts 中实现 useBuckets hook" &
Task --agent requirements-code "在 ui/src/hooks/use-files.ts 中实现 useFiles hook" &
Task --agent requirements-code "在 ui/src/hooks/use-upload.ts 中实现 useUpload hook" &
Task --agent requirements-code "在 ui/src/hooks/use-shares.ts 中实现 useShares hook"
```

---

## 验证清单

*门禁：在开始实施前检查*

- [x] 所有契约都有对应的测试（6个契约 → 6个契约测试 T005-T010）
- [x] 所有实体都有模型任务（5个实体 → 5个模型任务 T020-T024）
- [x] 所有测试都在实现之前（阶段 3.2 在 3.3 之前）
- [x] 并行任务真正独立（28个 [P] 任务验证无文件冲突）
- [x] 每个任务指定确切的文件路径（所有任务包含文件路径）
- [x] 没有任务修改与另一个 [P] 任务相同的文件（已验证）

---

## 执行建议

### 推荐实施顺序

**第1周（基础设施）**：
1. 设置任务（T001-T004）
2. 契约测试（T005-T010，验证失败）
3. 集成测试环境（T011-T012）
4. 数据模型（T020-T025）
5. Repository层（T026-T029）
6. pkg/minio工具（T030-T032）

**第2周（核心实现）**：
1. Service层（T033-T039）
2. Handler层（T040-T045）
3. 路由注册（T046）
4. 验证契约测试通过
5. 运行集成测试（T013-T019）

**第3周（前端实现）**：
1. API客户端（T047）
2. Hooks（T048-T052）
3. 组件（T053-T057）
4. 页面（T058-T060）
5. 路由配置（T061）

**第4周（优化和验收）**：
1. 单元测试（T062-T064）
2. 性能测试（T065）
3. 文档生成（T066）
4. 手动验证（T067）
5. 代码质量（T068-T070）

### 提交策略

- **原子提交**：每个任务完成后立即提交
- **提交信息格式**：`[004-minio] T###: 任务描述`
- **分支管理**：`004-minio-minio-bucket` 分支开发，完成后 PR 到 `main`

### 测试策略

- **TDD严格执行**：契约测试必须先失败再实现
- **集成测试频率**：每完成一个Service/Handler后运行相关集成测试
- **E2E验证**：阶段3.4完成后运行quickstart.md场景

---

**任务生成完成！共70个任务，预估4周完成。**

**下一步**：运行 `task lint` 检查代码质量，然后开始执行 T001。
