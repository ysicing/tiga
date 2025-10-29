# 实施计划：K8s 终端录制与审计增强

**分支**：`010-k8s-pod-009` | **日期**：2025-10-27 | **规格**：[spec.md](./spec.md)
**输入**：来自 `.claude/specs/010-k8s-pod-009/spec.md` 的功能规格

## 执行流程（/spec-kit:plan 命令范围）
```
1. 从输入路径加载功能规格 ✅
   → 规格文件已加载，所有 5 个澄清项已解决
2. 填充技术上下文（扫描需要澄清的内容）✅
   → 项目类型检测：Web 应用（后端 Go + 前端 React TypeScript）
   → 结构决策：使用现有 backend/ + ui/ 双层结构
3. 根据章程文档内容填充章程检查部分
   → 章程文件不存在，跳过章程检查（使用 SOLID 原则作为指导）
4. 评估章程检查部分
   → 无违规：设计符合现有架构模式（Repository + Service + Handler）
   → 更新进度跟踪：初始章程检查通过
5. 执行阶段 0 → research.md
6. 执行阶段 1 → contracts、data-model.md、quickstart.md
7. 重新评估章程检查
8. 规划阶段 2 → 描述任务生成方法
9. 停止 - 准备执行 /spec-kit:tasks 命令
```

**重要**：/spec-kit:plan 命令在步骤 9 停止。阶段 2-4 由其他命令执行：
- 阶段 2：/spec-kit:tasks 命令创建 tasks.md
- 阶段 3-4：实施执行（手动或通过工具）

## 摘要

K8s 终端录制与审计增强功能，支持节点终端和容器终端的自动录制，以及 K8s 资源操作的全面审计（包括只读操作）。

**核心功能**：
1. **终端录制**：自动录制所有 K8s 节点终端和 Pod 容器终端会话（Asciinema v2 格式，2 小时时长限制）
2. **操作审计**：记录所有 K8s 资源操作（创建、更新、删除、查看）到统一 AuditEvent 系统
3. **终端访问审计**：记录所有终端访问操作并关联到录制记录
4. **审计日志查询**：提供多维度筛选和搜索功能（操作者、操作类型、资源类型、时间范围）
5. **自动清理**：录制文件和审计日志 90 天自动清理

**技术方法**：
- 复用 009-3 统一终端录制系统（TerminalRecording 模型、StorageService、CleanupService）
- 复用统一 AuditEvent 系统（添加 K8s 子系统支持）
- 扩展现有 K8s 终端功能（pkg/kube/terminal.go）集成录制能力
- 为 K8s 资源操作 API 添加审计中间件/拦截器

## 技术上下文

**语言/版本**：Go 1.24+、TypeScript 5+
**主要依赖**：
- 后端：Gin (HTTP 框架)、GORM (ORM)、kubernetes/client-go、WebSocket
- 前端：React 19、TailwindCSS、Radix UI、TanStack Query
- 录制：Asciinema v2 格式
- 存储：本地文件系统 或 MinIO

**存储**：
- 数据库：PostgreSQL / MySQL / SQLite（GORM 自动迁移）
- 录制文件：本地存储（/var/lib/tiga/recordings/） 或 MinIO
- 审计日志：数据库（audit_events 表）

**测试**：
- 后端：Go testing（单元测试）、testcontainers-go（集成测试）
- 前端：Vitest、React Testing Library
- 契约测试：基于 OpenAPI 规范

**目标平台**：Linux 服务器（支持 Docker、K8s）

**项目类型**：web（前端 + 后端分离）

**性能目标**：
- 终端连接延迟增加 < 100ms（添加录制后）
- 资源操作延迟增加 < 50ms（添加审计后）
- 审计日志查询：< 500ms（分页 50 条，索引优化）
- 录制文件上传：< 200ms（本地）/ < 1s（MinIO）

**约束**：
- 录制时长限制：2 小时（7200 秒）
- 录制文件保留：90 天
- 审计日志保留：90 天
- 只读审计启用：审计日志量增加 10-50 倍
- 性能影响：最小化（异步审计日志写入）
- 存储管理：自动清理过期数据
- 兼容性：必须与现有 009-3 和 AuditEvent 系统集成

**规模/范围**：
- 支持 10+ K8s 集群同时管理
- 100+ 并发终端会话
- 1000+ 日均审计日志记录（启用只读审计后）
- 10GB+ 日均录制文件（按 100 并发、平均 10 分钟、10MB/小时计算）

## 章程检查
*门禁：必须在阶段 0 研究之前通过。在阶段 1 设计后重新检查。*

**架构原则**（基于 SOLID 和项目现有模式）：

1. **Repository Pattern（仓储模式）** ✅
   - 使用 `internal/repository/` 抽象数据访问
   - 已存在：`TerminalRecordingRepository`、`AuditEventRepository`
   - 需要扩展：添加 K8s 审计查询方法

2. **Service Layer（服务层）** ✅
   - 使用 `internal/services/` 封装业务逻辑
   - 已存在：`recording.ManagerService`、`recording.StorageService`、`recording.CleanupService`
   - 需要新增：K8s 终端录制服务、K8s 审计服务

3. **Handler Layer（处理器层）** ✅
   - 使用 `internal/api/handlers/` 处理 HTTP 请求
   - 使用 `pkg/handlers/resources/` 处理 K8s 资源操作
   - 需要扩展：为 K8s 资源处理器添加审计拦截器

4. **Middleware Pattern（中间件模式）** ✅
   - 使用 `internal/api/middleware/` 处理横切关注点
   - 已存在：audit.go（通用审计中间件）
   - 需要扩展：为 K8s 子系统定制审计逻辑

5. **Dependency Injection（依赖注入）** ✅
   - 通过构造函��注入依赖
   - 避免全局状态（遵循 Phase 4 改进）

6. **Interface Abstraction（接口抽象）** ✅
   - Repository 接口化（Phase 4 完成）
   - 便于测试和 mock

**门禁检查**：
- ✅ 无需引入新的架构模式（复用现有）
- ✅ 符合项目现有结构（internal/、pkg/）
- ✅ 使用现有基础设施（TerminalRecording、AuditEvent）
- ✅ 性能约束可实现（异步写入、索引优化）
- ✅ 测试策略明确（契约测试 + 集成测试）

## 项目结构

### 文档（此功能）
```
.claude/specs/010-k8s-pod-009/
├── spec.md               # 功能规格（已完成，5 个澄清项已解决）
├── plan.md               # 此文件（/spec-kit:plan 命令输出）
├── research.md           # 阶段 0 输出（/spec-kit:plan 命令）
├── data-model.md         # 阶段 1 输出（/spec-kit:plan 命令）
├── quickstart.md         # 阶段 1 输出（/spec-kit:plan 命令）
├── contracts/            # 阶段 1 输出（/spec-kit:plan 命令）
│   ├── k8s-terminal-recording-api.yaml  # 终端录制 API 契约
│   └── k8s-audit-api.yaml               # 审计日志 API 契约
└── tasks.md              # 阶段 2 输出（/spec-kit:tasks 命令 - 不由 /spec-kit:plan 创建）
```

### 源代码（仓库根目录）

**Web 应用程序结构（现有）**：
```
backend/internal/
├── models/
│   ├── terminal_recording.go      # 已存在（009-3）- 需扩展 K8s 类型
│   └── audit_event.go              # 已存在 - 需添加 K8s 子系统常量
├── repository/
│   ├── terminal_recording_repo.go  # 已存在 - 可能需扩展查询方法
│   └── audit_event_repo.go         # 已存在 - 需添加 K8s 筛选方法
├── services/
│   ├── recording/                  # 已存在（009-3）
│   │   ├── storage_service.go      # 复用
│   │   ├── cleanup_service.go      # 复用
│   │   └── manager_service.go      # 复用
│   └── k8s/                        # 需新增
│       ├── terminal_recording_service.go  # K8s 终端录制服务
│       └── audit_service.go               # K8s 审计服务
├── api/
│   ├── handlers/
│   │   ├── recording/              # 已存在（009-3）- 可能需扩展 K8s 筛选
│   │   └── audit_handler.go        # 已存在 - 需添加 K8s 子系统查询
│   └── middleware/
│       └── audit.go                # 已存在 - 可能需扩展 K8s 逻辑

backend/pkg/
├── kube/
│   ├── terminal.go                 # 已存在 - 需集成录制功能
│   └── terminal_recorder.go        # 需新增 - 录制适配器
└── handlers/
    └── resources/
        └── *.go                    # K8s 资源处理器 - 需添加审计拦截器

frontend/ui/src/
├── pages/
│   ├── k8s/
│   │   └── audit/                  # 需新增 - K8s 审计日志页面
│   └── system/
│       └── recordings/             # 可能需扩展 - 支持 K8s 类型筛选
└── services/
    └── api/
        ├── k8s-audit-api.ts        # 需新增 - K8s 审计 API 客户端
        └── recording-api.ts        # 可能需扩展 - 添加 K8s 类型

tests/
├── contract/
│   └── k8s/
│       ├── terminal_recording_test.go  # 需新增 - 终端录制契约测试
│       └── audit_test.go               # 需新增 - 审计日志契约测试
└── integration/
    └── k8s/
        ├── terminal_recording_integration_test.go
        └── audit_integration_test.go
```

**结构决策**：
- ✅ 使用现有 Web 应用结构（backend/ + ui/）
- ✅ 遵循项目现有模式（models → repository → services → handlers）
- ✅ 复用现有基础设施（009-3 录制系统、统一 AuditEvent）
- ✅ 最小化新增代码（扩展而非重写）
- ✅ 前端页面集成到现有 K8s 和 System Management 子系统

## 阶段 0：概述与研究

**目标**：研究技术选型、集成方案和最佳实践

### 研究任务列表

1. **K8s 终端录制集成** - 研究如何在现有 WebSocket 终端中集成 Asciinema 录制
   - 调研 pkg/kube/terminal.go 的 WebSocket 实现
   - 研究 Asciinema v2 格式实时写入方案
   - 研究 2 小时时长限制实现（定时器 + 优雅停止）

2. **录制文件存储最佳实践** - 研究大量录制文件的存储和管理
   - 调研 009-3 StorageService 的本地/MinIO 存储实现
   - 研究录制文件路径规范（/recordings/k8s_node/ 和 /recordings/k8s_pod/）
   - 研究 90 天自动清理机制（CleanupService 集成）

3. **K8s 审计拦截器模式** - 研究如何为 K8s 资源操作添加审计
   - 调研 pkg/handlers/resources/ 的资源处理器实现
   - 研究审计拦截器模式（装饰器模式 vs 中间件模式）
   - 研究只读操作审计（查看 Pod 详情、查看日志）的实现

4. **审计日志性能优化** - 研究审计日志大量写入的性能优化
   - 调研现有异步审计日志实现（async_audit_logger.go）
   - 研究批量写入和缓冲策略
   - 研究索引优化（按操作者、操作类型、时间范围查询）

5. **前端审计日志 UI 设计** - 研究审计日志查询界面的最佳实践
   - 调研现有审计日志页面（ui/src/pages/system/audit/）
   - 研究多维度筛选器设计（操作者、操作类型、资源类型、时间范围）
   - 研究分页和无限滚动实现

### 研究输出要求

对于每个研究任务，输出到 `research.md`：
- **决策**：选择了什么技术/方案
- **理由**：为什么选择（性能、兼容性、可维护性）
- **考虑的替代方案**：还评估了什么，为何未选择
- **实施要点**：关键技术细节和注意事项

**输出**：research.md，所有技术决策已明确

## 阶段 1：设计与契约
*前提条件：research.md 完成*

### 1. 数据模型设计 → `data-model.md`

基于功能规格的关键实体：

#### TerminalRecording（扩展 009-3 模型）

**已有字段**（无需修改）：
- `id`, `session_id`, `user_id`, `username`
- `recording_type`, `type_metadata` (JSONB)
- `started_at`, `ended_at`, `duration`
- `storage_type`, `storage_path`, `file_size`, `format`
- `rows`, `cols`, `shell`, `client_ip`

**扩展说明**：
- `recording_type` 新增值：`"k8s_node"`, `"k8s_pod"`
- `type_metadata` 结构（K8s 节点终端）：
  ```json
  {
    "cluster_id": "uuid",
    "node_name": "string"
  }
  ```
- `type_metadata` 结构（K8s 容器终端）：
  ```json
  {
    "cluster_id": "uuid",
    "namespace": "string",
    "pod_name": "string",
    "container_name": "string"
  }
  ```

**验证规则**（需新增）：
- FR-304：`duration` ≤ 7200 秒（2 小时限制）
- FR-307：`ended_at` + 90 天 < 当前时间 → 可清理

#### AuditEvent（扩展统一审计模型）

**已有字段**（无需修改）：
- `id`, `timestamp`, `action`, `resource_type`, `resource`
- `subsystem`, `user`, `client_ip`, `data`

**扩展说明**：
- `subsystem` 已有值：`SubsystemKubernetes = "kubernetes"` ✅
- `action` 新增值（需添加常量）：
  - `ActionCreateResource = "CreateResource"`
  - `ActionUpdateResource = "UpdateResource"`
  - `ActionDeleteResource = "DeleteResource"`
  - `ActionViewResource = "ViewResource"` （只读操作）
  - `ActionNodeTerminalAccess = "NodeTerminalAccess"`
  - `ActionPodTerminalAccess = "PodTerminalAccess"`
- `resource_type` 新增值（需添加常量）：
  - `ResourceTypeK8sNode = "k8s_node"`
  - `ResourceTypeK8sPod = "k8s_pod"`
  - （其他 K8s 资源类型已存在：`ResourceTypeDeployment`, `ResourceTypeService` 等）
- `resource.data` 结构（K8s 审计）：
  ```json
  {
    "cluster_id": "uuid",
    "namespace": "string",      // 可选
    "resource_name": "string",
    "change_summary": "string",  // 仅更新操作
    "recording_id": "uuid"       // 仅终端访问
  }
  ```

**索引优化**（已存在，需验证）：
- `idx_audit_events_composite` (resource_type, action, timestamp)
- `idx_audit_events_subsystem` (subsystem)
- `idx_audit_events_client_ip` (client_ip)

**验证规则**：
- FR-801：`created_at` + 90 天 < 当前时间 → 可清理
- FR-704：支持按 `action` 筛选只读操作（`ActionViewResource`）

#### K8sTerminalSession（内存结构，非数据库实体）

**字段**：
```go
type K8sTerminalSession struct {
    SessionID    uuid.UUID  // 会话 ID
    Type         string     // "k8s_node" 或 "k8s_pod"
    ClusterID    uuid.UUID  // 集群 ID
    NodeName     string     // 节点名称（仅节点终端）
    Namespace    string     // 命名空间（仅容器终端）
    PodName      string     // Pod 名称（仅容器终端）
    ContainerName string    // 容器名称（仅容器终端）
    RecordingID  uuid.UUID  // 关联的 TerminalRecording ID
    RecordingState string   // "recording", "stopped", "time_limit_reached"
    StartedAt    time.Time  // 开始时间
    Recorder     *AsciinemaRecorder  // 录制器实例
}
```

**生命周期**：
- 终端连接时创建
- 终端断开或达到 2 小时限制时销毁
- 用于协调录制和审计

### 2. API 契约设计 → `/contracts/`

#### 契约 1：K8s 终端录制 API (`k8s-terminal-recording-api.yaml`)

**端点**（复用现有，需验证支持 K8s 类型）：
1. `GET /api/v1/recordings` - 查询录制列表（支持按 `recording_type=k8s_node` 筛选）
2. `GET /api/v1/recordings/:id` - 获取录制详情
3. `GET /api/v1/recordings/:id/play` - 回放录制（返回 Asciinema v2 格式）
4. `DELETE /api/v1/recordings/:id` - 删除录制
5. `POST /api/v1/recordings/cleanup` - 手动触发清理

**查询参数扩展**：
- `recording_type`: `docker`, `webssh`, `k8s_node`, `k8s_pod`
- `cluster_id`: UUID（K8s 类型专用）
- `node_name`: string（K8s 节点终端专用）
- `namespace`: string（K8s 容器终端专用）
- `pod_name`: string（K8s 容器终端专用）

#### 契约 2：K8s 审计日志 API (`k8s-audit-api.yaml`)

**端点**（复用现有，需验证支持 K8s 子系统）：
1. `GET /api/v1/audit/events` - 查询审计日志
2. `GET /api/v1/audit/events/:id` - 获取审计详情
3. `GET /api/v1/audit/stats` - 审计统计（按操作类型、用户分组）
4. `POST /api/v1/audit/cleanup` - 手动触发清理

**查询参数扩展**：
- `subsystem`: `kubernetes`（筛选 K8s 子系统）
- `action`: `CreateResource`, `UpdateResource`, `DeleteResource`, `ViewResource`, `NodeTerminalAccess`, `PodTerminalAccess`
- `resource_type`: `deployment`, `service`, `pod`, `k8s_node`, `k8s_pod` 等
- `cluster_id`: UUID
- `user_id`: UUID
- `start_time`: ISO 8601
- `end_time`: ISO 8601
- `page`: int
- `page_size`: int (默认 50，最大 100)

### 3. 契约测试生成

对于每个 API 契约，生成契约测试到 `tests/contract/k8s/`：
- `terminal_recording_contract_test.go` - 验证录制 API 的请求/响应模式
- `audit_contract_test.go` - 验证审计 API 的请求/响应模式

测试策略：
- 断言 HTTP 状态码
- 断言响应 JSON 模式（使用 jsonschema）
- 断言必填字段存在
- 测试必须失败（尚无实现）

### 4. 集成测试场景 → 从用户故事提取

基于功能规格的验收场景，生成集成测试到 `tests/integration/k8s/`：

**测试文件 1**：`terminal_recording_integration_test.go`
- 场景 1-4：节点终端录制（连接、录制、断开、回放）
- 场景 5-7：容器终端录制（Exec、录制、断开）

**测试文件 2**：`audit_integration_test.go`
- 场景 8-10：K8s 资源操作审计（创建、删除、更新）
- 场景 11-12：终端访问审计（节点终端、容器终端）
- 场景 13-16：审计日志查询（筛选、分页、全文搜索）

### 5. 快速启动测试 → `quickstart.md`

快速启动测试 = 用户故事验证步骤

**步骤 1**：启动测试环境
```bash
# 启动 K8s 集群（kind 或 minikube）
kind create cluster --name tiga-test

# 启动 tiga 服务（录制功能已启用）
task dev:backend

# 初始化测试数据（创建测试用户、导入集群）
go run scripts/setup-test-data.go
```

**步骤 2**：验证节点终端录制
```bash
# 1. 通过 Web 界面连接节点终端（手动或自动化）
# 2. 执行一些命令（ls、ps、uptime）
# 3. 断开连接
# 4. 验证录制文件已创建（/var/lib/tiga/recordings/k8s_node/{session_id}.cast）
curl http://localhost:12306/api/v1/recordings?recording_type=k8s_node

# 5. 验证可以回放
curl http://localhost:12306/api/v1/recordings/{id}/play
```

**步骤 3**：验证资源操作审计
```bash
# 1. 通过 API 创建 Deployment
curl -X POST http://localhost:12306/api/v1/k8s/clusters/{id}/deployments

# 2. 验证审计日志已记录
curl "http://localhost:12306/api/v1/audit/events?subsystem=kubernetes&action=CreateResource"

# 3. 验证审计日志包含必需字段
jq '.data[0].resource.data.cluster_id' response.json
```

**步骤 4**：验证审计日志查询
```bash
# 1. 按操作者筛选
curl "http://localhost:12306/api/v1/audit/events?user_id={uuid}"

# 2. 按操作类型筛选
curl "http://localhost:12306/api/v1/audit/events?action=DeleteResource"

# 3. 按时间范围筛选
curl "http://localhost:12306/api/v1/audit/events?start_time=2025-10-27T00:00:00Z&end_time=2025-10-27T23:59:59Z"
```

**验收标准**：
- 所有 API 调用返回 200 OK
- 录制文件存在且格式正确（Asciinema v2）
- 审计日志包含所有必需字段
- 筛选和分页功能正常

### 6. 增量更新 CLAUDE.md（O(1) 操作）

**更新位置**：`/root/go/src/github.com/ysicing/tiga/CLAUDE.md`

**更新内容**（在 "Kubernetes 集群管理子系统" 章节下）：

```markdown
## Kubernetes 终端录制与审计（开发中）

**功能分支**: `010-k8s-pod-009`
**参考规格**: `.claude/specs/010-k8s-pod-009/`
**完成度**: 0%

K8s 终端录制与审计增强，支持节点终端和容器终端自动录制，以及 K8s 资源操作全面审计。

**核心特性**：
- 自动录制节点终端和容器终端会话（Asciinema v2 格式，2 小时限制）
- 记录所有 K8s 资源操作审计（创建、更新、删除、查看）
- 终端访问审计（关联到录制记录）
- 多维度审计日志查询（操作者、操作类型���资源类型、时间范围）
- 90 天自动清理（录制文件 + 审计日志）

**依赖系统**：
- 009-3 统一终端录制系统（TerminalRecording、StorageService、CleanupService）
- 统一 AuditEvent 系统

**关键文件**：
- 模型：`internal/models/terminal_recording.go`（扩展 K8s 类型）、`internal/models/audit_event.go`（扩展 K8s 子系统）
- 服务：`internal/services/k8s/terminal_recording_service.go`、`internal/services/k8s/audit_service.go`
- 终端集成：`pkg/kube/terminal_recorder.go`
- 审计拦截：`pkg/handlers/resources/audit_interceptor.go`

详见规格文档以了解完整的数据模型、API 契约、测试策略和实施计划。
```

**输出**：
- `data-model.md`：实体定义、字段、关系、验证规则
- `/contracts/k8s-terminal-recording-api.yaml`：OpenAPI 3.0 规范
- `/contracts/k8s-audit-api.yaml`：OpenAPI 3.0 规范
- `tests/contract/k8s/*_test.go`：失败的契约测试
- `tests/integration/k8s/*_test.go`：集成测试场景
- `quickstart.md`：快速启动验证步骤
- `CLAUDE.md`：增量更新（保持 < 150 行增量）

## 阶段 2：任务规划方法
*本节描述 /spec-kit:tasks 命令将执行的操作 - 不要在 /spec-kit:plan 期间执行*

**任务生成策略**：

1. **从阶段 1 设计文档生成任务**：
   - 加载 `~/.claude/templates/specify/tasks-template.md` 作为基础
   - 从 `data-model.md` 提取实体 → 模型扩展任务
   - 从 `/contracts/` 提取 API → 契约测试任务 + 实现任务
   - 从 `quickstart.md` 提取场景 → 集成测试任务

2. **任务分组**（按 TDD 顺序）：
   - **组 0：基础准备**（2-3 个任务）
     - 扩展 TerminalRecording 模型（添加 K8s 类型常量和验证）
     - 扩展 AuditEvent 模型（添加 K8s 子系统常量）
     - 运行数据库迁移验证

   - **组 1：终端录制**（6-8 个任务）
     - 实现 K8sTerminalRecorder（Asciinema 实时写入）
     - 集成录制到 pkg/kube/terminal.go（节点终端）
     - 集成录制到 pkg/kube/terminal.go（容器终端）
     - 实现 2 小时时长限制（定时器 + WebSocket 通知）
     - 实现录制停止逻辑（优雅关闭）
     - 契约测试：录��� API（K8s 类型筛选）
     - 集成测试：节点终端录制端到端
     - 集成测试：容器终端录制端到端

   - **组 2：操作审计**（5-7 个任务）
     - 实现 K8s 审计拦截器（装饰器模式）
     - 为资源处理器添加审计（创建���更新、删除）
     - 为资源处理器添加审计（只读操作：查看详情、查看日志）
     - 实现终端访问审计（关联 recording_id）
     - 契约测试：审计 API（K8s 子系统筛选）
     - 集成测试：资源操作审计
     - 集成测试：终端访问审计

   - **组 3：审计日志查询**（3-5 个任务）
     - 扩展 AuditEventRepository（添加 K8s 筛选方法）
     - 实现审计日志查询 Handler（多维度筛选）
     - 优化审计日志查询索引
     - 契约测试：审计查询 API
     - 集成测试：审计日志查询和筛选

   - **组 4：自动清理**（2-3 个任务）
     - 验证录制清理集成（CleanupService 已支持 K8s 类型）
     - 实现审计日志清理定时任务
     - 集成测试：自动清理验证

   - **组 5：前端集成**（4-6 个任务）
     - 实现 K8s 审计日志页面（多维度筛选器）
     - 实现录制列表筛选（支持 K8s 类型）
     - 实现审计日志详情查看
     - 实现审计统计图表（按操作类型、用户分组）
     - E2E 测试：审计日志查询流程
     - E2E 测试：录制回放流程

3. **任务排序策略**：
   - **TDD 顺序**：测试在实现之前（契约测试 → 实现 → 集成测试）
   - **依赖顺序**：模型 → Repository → Service → Handler → 前端
   - **并行标记 [P]**：独立文件可并行执行（如不同资源处理器的审计）

4. **任务格式**（示例）：
   ```markdown
   ### 任务 1：扩展 TerminalRecording 模型支持 K8s 类型
   - **文件**: `internal/models/terminal_recording.go`
   - **依赖**: 无
   - **标记**: [P]
   - **描述**: 添加 K8s 录制类型常量（`RecordingTypeK8sNode`, `RecordingTypeK8sPod`），添加 2 小时时长验证规则
   - **验收**: `go test ./internal/models -run TestTerminalRecording`
   ```

**预计输出**：25-30 个编号、有序的任务到 `tasks.md`

**重要**：此阶段由 `/spec-kit:tasks` 命令执行，而不是由 `/spec-kit:plan` 执行

## 阶段 3+：未来实施
*这些阶段超出了 /spec-kit:plan 命令的范围*

**阶段 3**：任务执行（/spec-kit:tasks 命令创建 tasks.md）
**阶段 4**：实施（按照 SOLID 原则执行 tasks.md）
**阶段 5**：验证（运行测试、执行 quickstart.md、性能验证）

## 复杂性跟踪
*仅在章程检查有必须证明合理的违规时填写*

| 违规 | 为什么需要 | 拒绝更简单替代方案的原因 |
|-----------|------------|-------------------------------------|
| 无违规 | - | - |

**说明**：
- ✅ 复用现有架构模式（Repository + Service + Handler）
- ✅ 复用现有基础设施（TerminalRecording、AuditEvent）
- ✅ 最小化新增代码（扩展而非重写）
- ✅ 符合 SOLID 原则（接口抽象、依赖注入）

## 进度跟踪
*此清单在执行流程期间更新*

**阶段状态**：
- [x] 阶段 0：研究准备（/spec-kit:plan 命令 - 任务列表已定义）
- [x] 阶段 0：研究完成（生成 research.md - 5 个研究任务完成）
- [x] 阶段 1：设计完成（生成 data-model.md、contracts/、quickstart.md、更新 CLAUDE.md）
- [x] 阶段 2：任务规划完成（/spec-kit:plan 命令 - 任务生成方法已描述）
- [x] 阶段 3：任务已生成（/spec-kit:tasks 命令 - 40 个任务已生成）
- [ ] 阶段 4：实施完成
- [ ] 阶段 5：验证通过

**生成的文档**：
- ✅ `plan.md` (643 行) - 实施计划和架构决策
- ✅ `research.md` (1160 行) - 5 个技术研究任务的详细决策
- ✅ `data-model.md` (823 行) - 3 个实体的完整数据模型
- ✅ `contracts/k8s-terminal-recording-api.yaml` (OpenAPI 3.0 规范)
- ✅ `contracts/k8s-audit-api.yaml` (OpenAPI 3.0 规范)
- ✅ `quickstart.md` (500+ 行) - 10 步手动验证指南
- ✅ `tasks.md` (850+ 行) - 40 个编号任务（TDD 顺序）
- ✅ `CLAUDE.md` - 新增功能文档章节（43 行）

**门禁状态**：
- [x] 初始章程检查：通过（无违规）
- [x] 设计后章程检查：通过（无新违规，设计符合现有架构）
- [x] 所有需要澄清的内容已解决（5/5 澄清项完成）
- [x] 复杂性偏差已记录（无偏差）

**下一步操作**：
开始实施阶段 4。按照 `tasks.md` 中定义的 40 个任务顺序执行：

1. **阶段 3.1（设置）**: T001-T003（模型扩展 + 数据库索引）
2. **阶段 3.2（测试优先）**: T004-T015（12 个契约测试和集成测试，必须失败）
3. **阶段 3.3（核心实现）**: T016-T030（录制系统 + 审计系统 + API）
4. **阶段 3.4（前端集成）**: T031-T035（录制页面 + 审计页面）
5. **阶段 3.5（优化验证）**: T036-T040（清理 + 性能 + 手动验证 + 文档）

**并行执行建议**: 参考 tasks.md 中的"并行执行示例"章节，可同时运行标记 [P] 的任务。

**预计完成时间**: 28-39 小时（单人）或 8-12 小时（3 人并行）。

---
*基于 spec-kit 工作流 - 参见 `.claude/specs/010-k8s-pod-009/spec.md`*
