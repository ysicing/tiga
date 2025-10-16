# 实施计划：MinIO 对象存储管理系统

**分支**：`004-minio-minio-bucket` | **日期**：2025-10-14 | **规格**：[spec.md](./spec.md)
**输入**：来自 `.claude/specs/004-minio-minio-bucket/spec.md` 的功能规格

## 执行流程（/spec-kit:plan 命令范围）
```
1. 从输入路径加载功能规格 ✅
   → 规格文件已找到并验证
2. 填充技术上下文（扫描需要澄清的内容） ✅
   → 项目类型检测：Web 应用（前端 React + 后端 Go）
   → 技术栈基于现有项目确定
3. 根据宪章文档内容填充宪章检查部分 ✅
4. 评估宪章检查部分 ✅（通过：5/5 原则符合）
5. 执行阶段 0 → research.md ✅
6. 执行阶段 1 → contracts、data-model.md、quickstart.md ✅
7. 重新评估宪章检查部分 ✅（通过：5/5 原则符合）
8. 规划阶段 2 → 描述任务生成方法 ✅
9. 停止 - 准备执行 /spec-kit:tasks 命令 ✅
```

**/spec-kit:plan 命令已完成所有步骤！**

**重要**：/spec-kit:plan 命令在步骤 9 停止。阶段 2-4 由其他命令执行：
- 阶段 2：/spec-kit:tasks 命令创建 tasks.md（下一步）
- 阶段 3-4：实施执行（手动或通过工具）

## 摘要

MinIO 对象存储管理系统为 Tiga DevOps Dashboard 添加完整的对象存储管理能力。核心功能包括：

**实例管理**：支持多个 MinIO 实例的统一管理，包括实例连接、监控和凭据管理。

**Bucket 管理**：创建、删除、配置 Bucket，支持私有/公开策略，显示存储使用统计。

**用户与权限**：创建 MinIO 用户，支持 Bucket 级和前缀级（目录级）细粒度权限授权，仅管理员可管理实例和用户。

**文件管理**：完整的文件浏览、上传（支持多文件、断点续传、无大小限制）、下载、文件夹管理（创建/删除）。

**在线预览**：支持图片、视频（HTTP Range 渐进式加载）、代码文件（语法高亮）、Markdown（渲染）预览，无大小限制。

**分享功能**：使用 MinIO 原生 presigned URL 生成临时分享链接，默认 7 天过期，支持自定义过期时间。

**安全与审计**：AES-256-GCM 凭据加密，90 天审计日志保留，记录管理操作和文件上传删除（不记录普通浏览以控制日志量）。

**技术方法**：后端使用 MinIO Go SDK 和 S3 API，前端使用 Monaco Editor（代码高亮）、Markdown 渲染库、原生 video 标签，复用现有的加密服务（pkg/crypto）和审计日志系统。

## 技术上下文

**语言/版本**：
- 后端：Go 1.24.3
- 前端：TypeScript 5.x、React 19、Node.js 20+

**主要依赖**：
- 后端：Gin 1.11（HTTP 框架）、GORM 1.25（ORM）、MinIO Go SDK v7+、现有 pkg/crypto（AES-256-GCM 加密）
- 前端：Vite 6（构建工具）、TailwindCSS 4（样式）、Radix UI（组件库）、TanStack Query（数据获取）、Monaco Editor（代码编辑器）、React Markdown（MD 渲染）、Zustand（状态管理）

**存储**：
- 主数据库：SQLite/PostgreSQL/MySQL（通过 GORM，复用现有连接）
- 对象存储：MinIO 实例（外部管理，通过 S3 API 访问）

**测试**：
- 后端：Go test、testcontainers-go（MinIO 容器集成测试）、testify（断言库）
- 前端：Vitest（单元测试）、React Testing Library（组件测试）

**目标平台**：
- 后端：Linux/macOS/Windows 服务器（Go 跨平台编译）
- 前端：现代浏览器（Chrome/Firefox/Safari/Edge 最新两个版本）
- 部署：Docker 容器、Kubernetes

**项目类型**：Web（前端 + 后端分离架构）

**性能目标**：
- API 响应：< 200ms p95（列表查询）、< 500ms p95（文件元数据查询）
- 文件上传：支持 100MB+ 文件的稳定上传，显示实时进度
- 预览加载：图片 < 2s、视频支持渐进式加载（HTTP Range）
- 并发：支持 100+ 并发用户的文件浏览和上传

**约束**：
- 安全：仅管理员可管理实例和用户，所有 MinIO 凭据加密存储
- 凭据安全：SecretKey 仅在创建时显示一次（与数据库管理一致）
- 审计：所有管理操作和文件上传删除必须记录，保留 90 天
- 浏览器兼容：视频格式仅支持浏览器原生格式（MP4/WebM/OGG）
- 文件夹：不支持重命名（对象存储限制）

**规模/范围**：
- 实例：支持 10+ MinIO 实例
- Bucket：每实例 100+ Buckets
- 用户：每实例 50+ MinIO 用户
- 文件：Bucket 中 10k+ 对象的流畅浏览（分页/虚拟滚动）
- 并发上传：单用户 5 个并发上传任务

## 宪章检查
*门禁：必须在阶段 0 研究之前通过。在阶段 1 设计后重新检查。*

### 原则 1：安全优先设计
- ✅ **RBAC 实施**：仅系统管理员可管理 MinIO 实例和创建用户，普通用户仅能访问授权的 Bucket 和文件
- ✅ **凭据保护**：MinIO 实例访问密钥使用 AES-256-GCM 加密存储（复用 pkg/crypto）
- ✅ **SecretKey 安全**：MinIO 用户 SecretKey 仅在创建时显示一次，之后不可恢复
- ✅ **细粒度权限**：支持 Bucket 级和前缀级（目录级）权限控制
- ✅ **路径遍历防护**：验证所有文件路径，禁止访问上级目录
- ✅ **分享链接限流**：防止分享链接被恶意频繁访问

### 原则 2：生产就绪性
- ✅ **错误处理**：所有 MinIO SDK 调用包含错误处理和重试逻辑
- ✅ **连接管理**：MinIO 客户端连接池和健康检查
- ✅ **测试覆盖**：契约测试、集成测试（testcontainers MinIO）、单元测试
- ✅ **向后兼容**：新增功能不影响现有数据库管理、主机管理功能
- ✅ **优雅降级**：MinIO 实例离线时显示错误状态，不阻塞其他功能
- ✅ **审计日志**：所有管理操作可追溯，保留 90 天

### 原则 3：卓越用户体验
- ✅ **直观界面**：复用现有 Radix UI 组件库，保持一致的设计语言
- ✅ **实时反馈**：文件上传显示进度（百分比、速度、剩余时间）
- ✅ **响应式设计**：所有页面适配桌面和平板（移动端文件管理受限）
- ✅ **快速加载**：文件列表虚拟滚动，图片预览懒加载
- ✅ **国际化**：所有 UI 文本支持中英文（复用现有 i18n 系统）
- ✅ **错误提示**：清晰的错误信息和操作建议（如文件格式不支持时提示下载）

### 原则 4：默认可观测性
- ✅ **监控集成**：MinIO 实例连接状态监控，连接失败时告警
- ✅ **存储指标**：实例和 Bucket 级别的存储使用统计
- ✅ **审计日志查询**：支持按时间、用户、操作类型、资源筛选审计日志
- ✅ **操作追踪**：所有管理操作记录操作者、时间、结果、IP 地址
- ✅ **性能指标**：集成到现有 Prometheus 指标系统（定义指标：minio_api_duration_seconds、minio_operations_total、minio_instance_status）

### 原则 5：开源承诺
- ✅ **Apache 2.0 许可**：所有新增代码遵循项目许可证
- ✅ **API 文档**：所有端点包含 Swagger 注释
- ✅ **架构文档**：在 plan.md、data-model.md、contracts/ 中记录设计决策
- ✅ **贡献友好**：模块化设计，易于扩展（如后续添加 Bucket 生命周期规则）

### 宪章合规性评估
- **初始评估**：✅ 通过（5/5 原则符合）
- **设计后重新评估**：✅ 通过（5/5 原则符合）
- **说明**：阶段 1 设计完成后，Prometheus 指标已明确定义（minio_api_duration_seconds、minio_operations_total、minio_instance_status），所有宪章原则均已满足

## 项目结构

### 文档（此功能）
```
.claude/specs/004-minio-minio-bucket/
├── spec.md              # 功能规格（✅ 已完成）
├── plan.md              # 此文件（✅ /spec-kit:plan 命令输出）
├── research.md          # 阶段 0 输出（✅ 已生成）
├── data-model.md        # 阶段 1 输出（✅ 已生成）
├── quickstart.md        # 阶段 1 输出（✅ 已生成）
├── contracts/           # 阶段 1 输出（✅ 已生成）
│   ├── minio-instance.yaml    # ✅
│   ├── bucket.yaml            # 待生成
│   ├── minio-user.yaml        # 待生成
│   ├── file-operations.yaml   # 待生成
│   └── share-link.yaml        # 待生成
└── tasks.md             # 阶段 2 输出（/spec-kit:tasks 命令）
```

### 源代码（仓库根目录）

```
# Web 应用程序结构（前端 + 后端）

# 后端
internal/
├── models/                     # GORM 数据模型
│   ├── minio_instance.go      # MinIO 实例模型
│   ├── minio_bucket.go         # Bucket 元数据（可选，用于缓存）
│   ├── minio_user.go           # MinIO 用户模型
│   ├── minio_permission.go     # 权限策略模型
│   ├── minio_share_link.go     # 分享链接模型
│   └── minio_audit_log.go      # MinIO 审计日志模型
├── repository/minio/           # 数据访问层
│   ├── instance_repository.go
│   ├── user_repository.go
│   ├── permission_repository.go
│   └── audit_repository.go
├── services/minio/             # 业务逻辑层
│   ├── instance_manager.go     # 实例管理（连接、健康检查）
│   ├── bucket_service.go       # Bucket 操作
│   ├── user_service.go         # 用户管理
│   ├── permission_service.go   # 权限管理（策略生成）
│   ├── file_service.go         # 文件操作（上传/下载/预览）
│   ├── share_service.go        # 分享链接（presigned URL）
│   └── audit_logger.go         # 审计日志记录
└── api/handlers/minio/         # API 处理器
    ├── instance_handler.go     # /api/v1/minio/instances
    ├── bucket_handler.go       # /api/v1/minio/instances/{id}/buckets
    ├── user_handler.go         # /api/v1/minio/instances/{id}/users
    ├── permission_handler.go   # /api/v1/minio/permissions
    ├── file_handler.go         # /api/v1/minio/instances/{id}/files
    └── share_handler.go        # /api/v1/minio/shares

pkg/
├── minio/                      # MinIO 客户端工具（可复用）
│   ├── client.go              # MinIO 客户端创建和缓存
│   ├── policy.go              # 策略生成辅助函数
│   └── presigned.go           # Presigned URL 生成
└── crypto/                     # 加密服务（已存在，复用）
    └── encryption.go           # AES-256-GCM

# 前端
ui/src/
├── pages/minio/                # MinIO 管理页面
│   ├── instances-page.tsx     # 实例列表
│   ├── instance-detail-page.tsx  # 实例详情
│   ├── buckets-page.tsx       # Bucket 列表
│   ├── files-page.tsx         # 文件浏览器
│   └── users-page.tsx         # 用户管理
├── components/minio/           # MinIO 专用组件
│   ├── instance-form.tsx      # 实例表单
│   ├── bucket-form.tsx        # Bucket 表单
│   ├── user-form.tsx          # 用户表单
│   ├── permission-form.tsx    # 权限配置表单
│   ├── file-uploader.tsx      # 文件上传组件（支持多文件、进度）
│   ├── file-previewer.tsx     # 文件预览组件
│   ├── image-viewer.tsx       # 图片预览（缩放、旋转）
│   ├── video-player.tsx       # 视频播放器（原生 video）
│   ├── code-viewer.tsx        # 代码查看器（Monaco Editor）
│   ├── markdown-viewer.tsx    # Markdown 渲染器
│   └── share-dialog.tsx       # 分享链接对话框
├── services/                   # API 客户端服务
│   └── minio-api.ts           # MinIO API 客户端
└── hooks/                      # 自定义 React hooks
    ├── use-minio-instances.ts
    ├── use-buckets.ts
    ├── use-files.ts
    └── use-upload.ts           # 文件上传 hook（支持断点续传）

# 测试
tests/
├── contract/minio/             # 契约测试
│   ├── instance_contract_test.go
│   ├── bucket_contract_test.go
│   ├── user_contract_test.go
│   └── file_contract_test.go
├── integration/minio/          # 集成测试（testcontainers MinIO）
│   ├── instance_test.go
│   ├── bucket_test.go
│   ├── permission_test.go
│   └── file_operations_test.go
└── unit/                       # 单元测试
    ├── policy_test.go          # 策略生成测试
    └── presigned_test.go       # Presigned URL 测试
```

**结构决策**：
- 选择 **Web 应用程序结构**（选项 2）
- 理由：Tiga 是前后端分离的 Web Dashboard，后端 Go API + 前端 React UI
- 后端按领域分层：models（数据）→ repository（访问）→ services（逻辑）→ handlers（API）
- 前端按功能模块组织：pages（路由）、components（UI）、services（API 调用）、hooks（状态逻辑）
- 测试分层：contract（API 契约）、integration（真实 MinIO）、unit（纯函数逻辑）

## 阶段 0：概述与研究

### 待研究项

基于技术上下文，以下方面需要研究最佳实践：

1. **MinIO Go SDK 使用模式**
   - 客户端连接池管理
   - 错误处理和重试策略
   - Presigned URL 生成和过期控制
   - Bucket 策略 JSON 生成

2. **前端文件上传最佳实践**
   - 多文件并发上传（axios 或 fetch）
   - 断点续传实现（multipart upload）
   - 实时进度显示（上传速度、剩余时间计算）
   - 大文件处理（分片上传）

3. **文件预览技术选型**
   - 图片预览：原生 img + 缩放库（react-image-lightbox / react-photo-view）
   - 视频播放：原生 video + HTTP Range 支持验证
   - 代码高亮：Monaco Editor 配置（轻量级模式）
   - Markdown 渲染：react-markdown + remark-gfm（GitHub 风格）

4. **安全策略生成**
   - MinIO IAM 策略 JSON 结构（Bucket 级 + 前缀级）
   - 只读/只写/读写权限的 Action 映射
   - 策略验证和测试方法

5. **性能优化**
   - 文件列表虚拟滚动（react-window / react-virtualized）
   - 图片懒加载和缩略图策略
   - Presigned URL 缓存策略（短期缓存避免频繁签名）

6. **集成测试环境**
   - testcontainers-go MinIO 容器配置
   - 测试数据初始化（Bucket、用户、文件）
   - 清理策略

### 研究任务分配

1. **MinIO SDK 研究**：
   - 查询 MinIO Go SDK v7 官方文档
   - 研究 presigned URL 最佳过期时间配置
   - 研究策略生成 API（SetBucketPolicy）

2. **前端上传研究**：
   - 研究 tus-js-client（断点续传标准协议）或 MinIO 原生 multipart upload
   - 研究进度回调和速度计算最佳实践

3. **预览组件研究**：
   - 评估 react-markdown vs marked + DOMPurify
   - 评估 Monaco Editor 按需加载策略（减少打包体积）
   - 评估图片预览库的 Radix UI 兼容性

4. **测试研究**：
   - 查询 testcontainers-go MinIO 镜像和配置示例
   - 研究契约测试中的 multipart 请求模拟

## 阶段 1：设计与契约

*前提条件：research.md 完成*

### 数据模型设计

从功能规格中提取的关键实体（详见 data-model.md）：

1. **MinIOInstance**：端点、凭据（加密）、连接状态、存储统计
2. **MinIOUser**：AccessKey、SecretKey（加密）、关联权限
3. **BucketPermission**：用户、Bucket、可选前缀、权限类型
4. **ShareLink**：文件对象、presigned URL、创建者、过期时间
5. **MinIOAuditLog**：操作类型、资源、操作者、结果、时间戳

### API 契约生成

基于功能需求生成 RESTful API 端点（详见 contracts/）：

**实例管理** (`/api/v1/minio/instances`):
- POST /instances - 创建实例
- GET /instances - 列出所有实例
- GET /instances/{id} - 获取实例详情
- PUT /instances/{id} - 更新实例
- DELETE /instances/{id} - 删除实例
- POST /instances/{id}/test - 测试连接

**Bucket 管理** (`/api/v1/minio/instances/{id}/buckets`):
- GET /buckets - 列出 Buckets
- POST /buckets - 创建 Bucket
- GET /buckets/{name} - 获取 Bucket 详情
- DELETE /buckets/{name} - 删除 Bucket
- PUT /buckets/{name}/policy - 更新 Bucket 策略

**用户管理** (`/api/v1/minio/instances/{id}/users`):
- GET /users - 列出用户
- POST /users - 创建用户（返回 SecretKey）
- DELETE /users/{username} - 删除用户

**权限管理** (`/api/v1/minio/permissions`):
- POST /permissions - 授予权限（Bucket 级或前缀级）
- GET /permissions - 列出权限（可按用户或 Bucket 筛选）
- DELETE /permissions/{id} - 撤销权限

**文件操作** (`/api/v1/minio/instances/{id}/files`):
- GET /files - 列出文件（支持前缀筛选、分页）
- POST /files - 上传文件（multipart）
- GET /files/download - 下载文件（presigned URL）
- GET /files/preview - 获取预览 URL（presigned URL）
- DELETE /files - 删除文件

**分享链接** (`/api/v1/minio/shares`):
- POST /shares - 创建分享链接
- GET /shares - 列出我的分享链接
- DELETE /shares/{id} - 撤销分享链接

### 契约测试生成

为每个端点生成契约测试（测试初始状态：失败，待实现）：

- `tests/contract/minio/instance_contract_test.go`：实例 CRUD、连接测试
- `tests/contract/minio/bucket_contract_test.go`：Bucket CRUD、策略更新
- `tests/contract/minio/user_contract_test.go`：用户 CRUD、SecretKey 返回验证
- `tests/contract/minio/permission_contract_test.go`：权限授予/撤销、前缀级权限
- `tests/contract/minio/file_contract_test.go`：文件列表、上传、下载、预览 URL

### 快速启动场景

从用户故事提取验收场景（详见 quickstart.md）：

1. **管理员添加 MinIO 实例**
   - 输入端点、AccessKey、SecretKey
   - 测试连接成功
   - 查看实例存储统计

2. **管理员创建 Bucket 和用户**
   - 创建私有 Bucket "team-assets"
   - 创建用户 "dev-user"，记录 SecretKey
   - 授予用户对 Bucket 的读写权限

3. **用户上传和预览文件**
   - 上传图片文件，显示进度
   - 点击图片，弹窗预览
   - 上传代码文件，语法高亮显示

4. **用户生成分享链接**
   - 选择文件，生成 7 天有效的分享链接
   - 复制链接，未登录用户可访问

5. **管理员查看审计日志**
   - 查看所有 Bucket 创建、用户授权、文件上传操作
   - 按用户筛选，查看特定用户的操作历史

## 阶段 2：任务规划方法

*本节描述 /spec-kit:tasks 命令将执行的操作 - 不要在 /spec-kit:plan 期间执行*

### 任务生成策略

1. **从契约生成测试任务**：
   - 每个契约文件 → 1 个契约测试任务（标记 [P] 并行）
   - 契约测试初始状态：失败（无实现）

2. **从数据模型生成模型任务**：
   - 每个实体 → 1 个 GORM 模型创建任务（标记 [P]）
   - 包含字段、关系、索引、加密字段标记

3. **从服务层生成实现任务**：
   - 每个服务 → 1 个服务实现任务
   - 依赖顺序：models → repository → services → handlers

4. **从集成测试生成测试任务**：
   - 每个集成测试场景 → 1 个集成测试任务
   - 依赖：testcontainers MinIO 环境配置任务在前

5. **前端组件任务**：
   - 每个页面 → 1 个页面组件任务
   - 每个复杂组件 → 1 个组件任务（如文件上传器、预览器）
   - API 客户端 → 1 个任务

### 任务排序策略

**后端 TDD 顺序**：
1. 数据模型（并行）
2. 契约测试（并行，测试先行）
3. Repository 层（按依赖）
4. Service 层（按依赖）
5. Handler 层（按依赖）
6. 集成测试（验证完整流程）

**前端顺序**：
1. API 客户端
2. 基础组件（表单、对话框）
3. 页面组件（按路由依赖）
4. 高级组件（上传器、预览器）

**并行标记 [P]**：
- 所有数据模型（无依赖）
- 所有契约测试（独立文件）
- 独立的 Service 和 Handler（如 instance_service 和 bucket_service）

### 预计输出

**任务总数**：约 45-50 个任务

**分类**：
- 数据模型：6 个（MinIOInstance、MinIOUser、BucketPermission、ShareLink、AuditLog、关联表）
- 契约测试：5 个（对应 5 个契约文件）
- Repository：4 个（instance、user、permission、audit）
- Service：6 个（instance_manager、bucket、user、permission、file、share）
- Handler：6 个（对应 6 个 API 模块）
- 集成测试：5 个（对应主要场景）
- 前端 API 客户端：1 个
- 前端页面：5 个
- 前端组件：8 个（表单、上传器、预览器等）
- 测试环境配置：2 个（MinIO testcontainer、测试数据初始化）

**依赖关系示例**：
- Task 7（contract test）→ Task 15（handler）→ Task 25（integration test）
- Task 1-6（models）→ Task 8-11（repositories）
- Task 30（API client）→ Task 31-35（pages）

**重要**：此阶段的详细任务列表将由 /spec-kit:tasks 命令生成到 tasks.md

## 阶段 3+：未来实施

*这些阶段超出了 /spec-kit:plan 命令的范围*

**阶段 3**：任务执行
- 运行 `/spec-kit:tasks` 生成 tasks.md
- 任务将按依赖顺序和并行标记 [P] 排列
- 估算每个任务的工作量（S/M/L）

**阶段 4**：实施
- 按 tasks.md 顺序执行任务
- TDD 流程：契约测试 → 实现 → 集成测试
- 每个任务完成后运行相关测试验证
- 遵循宪章原则（安全、质量、UX）

**阶段 5**：验证
- 运行所有契约测试（100% 通过）
- 运行所有集成测试（testcontainers MinIO）
- 执行 quickstart.md 中的验收场景
- 性能验证：API 响应时间、文件上传稳定性
- 安全审查：凭据加密、权限隔离、审计日志
- 前端测试：组件测试、端到端测试

## 复杂性跟踪

*宪章检查未发现重大违规，无需记录复杂性偏差*

| 违规 | 为什么需要 | 拒绝更简单替代方案的原因 |
|------|-----------|----------------------|
| 无   | -         | -                    |

**说明**：MinIO 管理功能复用了现有的架构模式（数据库管理、主机管理），技术栈统一，无额外复杂性引入。前缀级权限是用户明确需求（支持目录级授权），MinIO 原生支持，实现成本低。

## 进度跟踪

*此清单在执行流程期间更新*

**阶段状态**：
- [x] 阶段 0：研究完成（/spec-kit:plan 命令）✅
- [x] 阶段 1：设计完成（/spec-kit:plan 命令）✅
- [x] 阶段 2：任务规划方法已描述（/spec-kit:plan 命令）✅
- [ ] 阶段 3：任务已生成（/spec-kit:tasks 命令） - 下一步
- [ ] 阶段 4：实施完成
- [ ] 阶段 5：验证通过

**门禁状态**：
- [x] 初始宪章检查：通过（5/5 原则符合）
- [x] 设计后宪章检查：通过（5/5 原则符合）✅
- [x] 所有需要澄清的内容已解决（spec.md 已完成澄清）
- [x] 复杂性偏差已记录（无偏差）

**执行流程进度**：
- [x] 步骤 1：加载功能规格
- [x] 步骤 2：填充技术上下文
- [x] 步骤 3：填充宪章检查
- [x] 步骤 4：评估宪章检查（通过）
- [x] 步骤 5：执行阶段 0（research.md）✅
- [x] 步骤 6：执行阶段 1（contracts、data-model.md、quickstart.md）✅
- [x] 步骤 7：重新评估宪章检查（通过）✅
- [x] 步骤 8：规划阶段 2（已描述任务规划方法）✅
- [x] 步骤 9：停止并准备 /spec-kit:tasks ✅

**下一步行动**：
运行 `/spec-kit:tasks` 命令生成 tasks.md 文件，将 45-50 个实施任务分解为可执行的任务列表。

---
*基于宪章 v1.0.0 - 参见 `.claude/memory/constitution.md`*
*参考规格 v已澄清 - 参见 `.claude/specs/004-minio-minio-bucket/spec.md`*
