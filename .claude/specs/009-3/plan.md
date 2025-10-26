# 实施计划：统一终端录制系统

**分支**：`009-3` | **日期**：2025-10-26 | **规格**：[.claude/specs/009-3/spec.md](./spec.md)
**输入**：来自 `.claude/specs/009-3/spec.md` 的功能规格

## 执行流程（/spec-kit:plan 命令范围）
```
1. 从输入路径加载功能规格
   → 如果未找到：错误 "路径 {path} 下没有功能规格"
2. 填充技术上下文（扫描需要澄清的内容）
   → 从文件系统结构或上下文检测项目类型（web=前端+后端，mobile=应用+API）
   → 根据项目类型设置结构决策
3. 根据章程文档内容填充章程检查部分
4. 评估下面的章程检查部分
   → 如果存在违规：在复杂性跟踪中记录
   → 如果无法提供理由：错误 "请先简化方法"
   → 更新进度跟踪：初始章程检查
5. 执行阶段 0 → research.md
   → 如果仍有需要澄清的内容：错误 "解决未知项"
6. 执行阶段 1 → contracts、data-model.md、quickstart.md、代理特定模板文件（例如，Claude Code 的 `CLAUDE.md`、GitHub Copilot 的 `.github/copilot-instructions.md`、Gemini CLI 的 `GEMINI.md`、Qwen Code 的 `QWEN.md` 或 opencode 的 `AGENTS.md`）
7. 重新评估章程检查部分
   → 如果有新违规：重构设计，返回阶段 1
   → 更新进度跟踪：设计后章程检查
8. 规划阶段 2 → 描述任务生成方法（不要创建 tasks.md）
9. 停止 - 准备执行 /spec-kit:tasks 命令
```

**重要**：/spec-kit:plan 命令在步骤 7 停止。阶段 2-4 由其他命令执行：
- 阶段 2：/spec-kit:tasks 命令创建 tasks.md
- 阶段 3-4：实施执行（手动或通过工具）

## 摘要

**核心需求**：统一三个分散的终端录制实现（Docker 容器、WebSSH、K8s 节点），提供统一的存储路径、清理策略和管理界面。

**主要目标**：
- 整合 3 个独立的录制实现到单一的 TerminalRecording 数据模型
- 统一存储路径配置（支持本地文件系统和可选的 MinIO 对象存储）
- 实现统一的自动清理机制（默认 90 天保留期，凌晨 4:00 执行）
- 提供统一的录制查看和管理界面
- 支持 Asciinema v2 格式的终端回放

**技术方法**：
- 扩展现有 `models.TerminalRecording` 模型以支持多种终端类型（通过 RecordingType 字段区分）
- 创建统一的 RecordingStorageService 抽象存储后端（本地/MinIO）
- 实现 RecordingCleanupService 统一清理逻辑（替换现有的 Docker 专用服务）
- 复用现有的 Asciinema 录制格式（已在 Docker 终端中使用）
- 通过 robfig/cron 定时任务调度清理

## 技术上下文
**语言/版本**：Go 1.24.3
**主要依赖**：Gin v1.11.0 (web framework), GORM v1.31.0 (ORM), gRPC v1.76.0 (Agent通信), MinIO SDK v7.0.95 (对象存储), go-redis/v9 (缓存), robfig/cron v3.0.1 (任务调度), gorilla/websocket v1.5.4 (终端流), testify (测试), testcontainers-go (集成测试)
**存储**：PostgreSQL/MySQL/SQLite (多数据库支持) + MinIO (可选对象存储)
**测试**：testify (断言), testcontainers-go (集成测试环境)
**目标平台**：Linux 服务器
**项目类型**：web (Go backend API + React TypeScript frontend)
**性能目标**：录制写入 <10ms/frame、清理任务 <5min (10k 录制)、录制回放加载 <1s (需验证)
**约束**：并发终端会话 <1000、单个录制文件 <500MB、存储 I/O 性能依赖底层文件系统/MinIO
**规模/范围**：预计同时活跃终端 100-500、历史录制 10k-100k 条、总存储 100GB-1TB

## 章程检查
*门禁：必须在阶段 0 研究之前通过。在阶段 1 设计后重新检查。*

### 安全性原则
- ✅ **RBAC 访问控制**：录制查看和删除需要权限检查（复用现有 RBAC 系统）
- ✅ **审计日志**：所有录制访问和删除操作记录审计日志（扩展现有 AuditLog）
- ✅ **敏感数据保护**：录制文件包含终端操作历史，仅授权用户可访问

### 生产就绪性
- ✅ **测试覆盖率**：>70% 目标 - 包括契约测试、单元测试、集成测试
- ✅ **错误处理**：存储失败、清理失败、并发冲突的完整错误处理
- ✅ **性能验证**：清理任务性能测试（10k 录制 <5min）

### 用户体验
- ✅ **统一界面**：单一录制管理页面，替代 3 个分散入口
- ✅ **搜索和过滤**：按用户、时间范围、终端类型快速查找
- ✅ **回放体验**：Asciinema 格式支持暂停、快进、速度调节

### 可观测性
- ✅ **Prometheus 指标**：录制总数、存储占用、清理统计、失败率
- ✅ **日志记录**：清理任务执行日志、存储操作日志、错误详情

### 简洁性原则
- ✅ **代码统一**：消除 3 个重复实现，单一 RecordingService
- ✅ **最小化依赖**：复用现有 MinIO SDK、GORM、cron 调度器
- ⚠️ **存储抽象**：需要评估是否过度设计（本地 + MinIO 双存储支持）

### 向后兼容性
- ⚠️ **数据迁移**：现有 Docker 录制数据需要迁移到统一模型（InstanceID → RecordingType + TypeMetadata）
- ✅ **API 兼容**：现有 Docker 录制 API 保持不变，通过适配层调用统一服务

## 项目结构

### 文档（此功能）
```
specs/[###-feature]/
├── plan.md              # 此文件（/spec-kit:plan 命令输出）
├── research.md          # 阶段 0 输出（/spec-kit:plan 命令）
├── data-model.md        # 阶段 1 输出（/spec-kit:plan 命令）
├── quickstart.md        # 阶段 1 输出（/spec-kit:plan 命令）
├── contracts/           # 阶段 1 输出（/spec-kit:plan 命令）
└── tasks.md             # 阶段 2 输出（/spec-kit:tasks 命令 - 不由 /spec-kit:plan 创建）
```

### 源代码（仓库根目录）
```
# Go 后端（internal 架构）
internal/
├── models/
│   └── terminal_recording.go        # [修改] 扩展为多终端类型支持
├── repository/
│   └── terminal_recording_repo.go   # [新建] Repository 接口和实现
├── services/
│   ├── recording/
│   │   ├── storage_service.go       # [新建] 存储抽象（本地/MinIO）
│   │   ├── cleanup_service.go       # [新建] 统一清理服务
│   │   └── manager_service.go       # [新建] 录制管理服务
│   └── docker/
│       └── recording_cleanup_service.go  # [废弃] 迁移逻辑到统一服务
└── api/handlers/
    └── recording/
        ├── recording_handler.go     # [新建] 统一录制 API 处理器
        └── playback_handler.go      # [新建] 录制回放处理器

# 前端 UI（React TypeScript）
ui/src/
├── pages/
│   └── recordings/
│       ├── recording-list-page.tsx  # [新建] 录制列表页
│       └── recording-player-page.tsx # [新建] 录制播放页
├── components/
│   └── recording/
│       └── asciinema-player.tsx     # [新建] Asciinema 播放器组件
└── services/
    └── recording-service.ts         # [新建] 录制 API 客户端

# 测试
tests/
├── contract/
│   └── recording_api_test.go        # [新建] API 契约测试
├── integration/
│   └── recording_cleanup_test.go    # [新建] 清理服务集成测试
└── unit/
    └── storage_service_test.go      # [新建] 存储服务单元测试

# 数据库迁移（如需要）
migrations/
└── 009_unify_terminal_recordings.sql  # [可选] 数据迁移脚本
```

**结构决策**：
- **后端架构**：遵循现有 internal/ 分层架构（models → repository → services → api/handlers）
- **服务层拆分**：storage_service（存储抽象）、cleanup_service（清理逻辑）、manager_service（业务编排）
- **废弃代码处理**：Docker 专用 recording_cleanup_service.go 逻辑迁移后标记 @Deprecated
- **前端集成**：新增 recordings 子系统页面，复用现有布局和路由系统
- **测试组织**：遵循现有 contract/integration/unit 三层测试结构

## 阶段 0：概述与研究
1. **从上面的技术上下文中提取未知项**：
   - 对于每个需要澄清的内容 → 研究任务
   - 对于每个依赖 → 最佳实践任务
   - 对于每个集成 → 模式任务

2. **生成并派发研究代理**：
   ```
   对于技术上下文中的每个未知项：
     任务："研究 {未知项} 用于 {功能上下文}"
   对于每个技术选择：
     任务："查找 {领域} 中 {技术} 的最佳实践"
   ```

3. **在 `research.md` 中合并发现**，使用格式：
   - 决策：[选择了什么]
   - 理由：[为什么选择]
   - 考虑的替代方案：[还评估了什么]

**输出**：research.md，所有需要澄清的内容都已解决

## 阶段 1：设计与契约
*前提条件：research.md 完成*

1. **从功能规格中提取实体** → `data-model.md`：
   - 实体名称、字段、关系
   - 来自需求的验证规则
   - 如适用的状态转换

2. **从功能需求生成 API 契约**：
   - 对于每个用户操作 → 端点
   - 使用标准 REST/GraphQL 模式
   - 输出 OpenAPI/GraphQL 模式到 `/contracts/`

3. **从契约生成契约测试**：
   - 每个端点一个测试文件
   - 断言请求/响应模式
   - 测试必须失败（尚无实现）

4. **从用户故事中提取测试场景**：
   - 每个故事 → 集成测试场景
   - 快速启动测试 = 故事验证步骤

5. **增量更新代理文件**（O(1) 操作）：
   - 运行 `.specify/scripts/bash/update-agent-context.sh claude`
     **重要**：按上述指定执行。不要添加或删除任何参数。
   - 如果存在：仅从当前计划添加新技术
   - 在标记之间保留手动添加
   - 更新最近变更（保留最后 3 个）
   - 保持在 150 行以下以提高令牌效率
   - 输出到仓库根目录

**输出**：data-model.md、/contracts/*、失败的测试、quickstart.md、代理特定文件

## 阶段 2：任务规划方法
*本节描述 /spec-kit:tasks 命令将执行的操作 - 不要在 /spec-kit:plan 期间执行*

**任务生成策略**：
- 加载 `~/.claude/templates/specify/tasks-template.md` 作为基础
- 从阶段 1 设计文档（contracts、data-model.md、quickstart.md）生成任务
- 每个 API 契约端点 → 契约测试任务 [P]
- 每个数据模型实体字段 → 模型扩展和迁移任务 [P]
- 每个 quickstart 测试场景 → 集成测试任务
- 使测试通过的实现任务（服务层、处理器、前端组件）

**具体任务来源映射**：

1. **从 data-model.md 生成**：
   - 任务 1: 扩展 `TerminalRecording` 模型（添加 RecordingType、TypeMetadata、StorageType）[P]
   - 任务 2: 创建数据库迁移脚本（添加新字段、索引）[P]
   - 任务 3: 实现数据迁移逻辑（迁移现有 Docker 录制）[P]
   - 任务 4: 创建 `RecordingRepository` 接口和实现 [P]
   - 任务 5: 实现 Repository 查询方法（ListByType、FindExpired、Search 等）[P]

2. **从 contracts/recording-api.yaml 生成**：
   - 任务 6-14: 每个 API 端点对应一个契约测试 [P]
     - GET /recordings → 契约测试（分页、过滤）
     - GET /recordings/:id → 契约测试（详情）
     - DELETE /recordings/:id → 契约测试（删除）
     - GET /recordings/search → 契约测试（搜索）
     - GET /recordings/statistics → 契约测试（统计）
     - GET /recordings/:id/playback → 契约测试（回放）
     - GET /recordings/:id/download → 契约测试（下载）
     - POST /recordings/cleanup/trigger → 契约测试（清理触发）
     - GET /recordings/cleanup/status → 契约测试（清理状态）

3. **服务层实现任务**：
   - 任务 15: 实现 `StorageService`（本地文件系统）[P]
   - 任务 16: 实现 `CleanupService`（统一清理逻辑）[P]
   - 任务 17: 实现 `ManagerService`（录制 CRUD）[P]
   - 任务 18: 实现 `StatisticsService`（聚合查询）[P]
   - 任务 19: 集成清理任务到 Cron 调度器 [P]

4. **API 处理器实现任务**：
   - 任务 20: 实现 `RecordingHandler`（列表、详情、删除、搜索、统计）
   - 任务 21: 实现 `PlaybackHandler`（回放、下载）
   - 任务 22: 实现 `CleanupHandler`（触发、状态）
   - 任务 23: 注册录制 API 路由

5. **现有终端处理器集成任务**：
   - 任务 24: 集成 Docker 终端录制到统一系统
   - 任务 25: 集成 WebSSH 终端录制到统一系统
   - 任务 26: 集成 K8s 节点终端录制到统一系统

6. **前端实现任务**：
   - 任务 27: 创建录制列表页（`recording-list-page.tsx`）
   - 任务 28: 创建录制详情页（`recording-detail-page.tsx`）
   - 任务 29: 创建 Asciinema 播放器组件（`asciinema-player.tsx`）
   - 任务 30: 创建录制 API 客户端（`recording-service.ts`）
   - 任务 31: 添加录制路由和导航

7. **从 quickstart.md 生成集成测试任务**：
   - 任务 32: Docker 容器终端录制集成测试（场景 1）
   - 任务 33: WebSSH 终端录制集成测试（场景 2）
   - 任务 34: K8s 节点终端录制集成测试（场景 3）
   - 任务 35: 统一界面集成测试（场景 4）
   - 任务 36: 录制回放集成测试（场景 5）
   - 任务 37: 自动清理任务集成测试（场景 6-7）
   - 任务 38: 数据迁移验证测试（场景 10）

8. **可选任务（MinIO 支持 - Phase 2）**：
   - 任务 39: 实现 MinIO StorageService [可选]
   - 任务 40: MinIO 存储集成测试（场景 8）[可选]

9. **性能测试和优化任务**：
   - 任务 41: 录制写入性能基准测试
   - 任务 42: 清理任务性能测试（10k 录制）
   - 任务 43: 并发录制负载测试（场景 9）

10. **文档和 Swagger 任务**：
    - 任务 44: 添加 Swagger 注释到录制 API 处理器
    - 任务 45: 生成 Swagger 文档
    - 任务 46: 更新用户文档

**排序策略**：
- **TDD 顺序**：契约测试（任务 6-14）→ 实现任务（任务 15-26）→ 集成测试（任务 32-38）
- **依赖顺序**：
  1. 模型扩展和迁移（任务 1-3）→ Repository（任务 4-5）
  2. Repository → 服务层（任务 15-19）
  3. 服务层 → API 处理器（任务 20-23）
  4. API 处理器 → 前端（任务 27-31）
  5. 基础功能完成 → 集成测试（任务 32-38）
- **并行执行标记 [P]**：
  - 模型扩展、迁移脚本、Repository 接口可并行
  - 契约测试可并行
  - 服务层实现可并行（Storage、Cleanup、Manager 独立）
  - 前端组件可并行

**预计输出**：tasks.md 中的 45-50 个编号、有序的任务

**重要**：此阶段由 /spec-kit:tasks 命令执行，而不是由 /spec-kit:plan 执行

## 阶段 3+：未来实施
*这些阶段超出了 /spec-kit:plan 命令的范围*

**阶段 3**：任务执行（/spec-kit:tasks 命令创建 tasks.md）
**阶段 4**：实施（按照章程原则执行 tasks.md）
**阶段 5**：验证（运行测试、执行 quickstart.md、性能验证）

## 复杂性跟踪
*仅在章程检查有必须证明合理的违规时填写*

| 违规 | 为什么需要 | 拒绝更简单替代方案的原因 |
|-----------|------------|-------------------------------------|
| [例如，第 4 个项目] | [当前需求] | [为什么 3 个项目不够] |
| [例如，仓库模式] | [具体问题] | [为什么直接数据库访问不够] |


## 进度跟踪
*此清单在执行流程期间更新*

**阶段状态**：
- [x] 阶段 0：研究完成（/spec-kit:plan 命令）
- [x] 阶段 1：设计完成（/spec-kit:plan 命令）
- [x] 阶段 2：任务规划完成（/spec-kit:plan 命令 - 仅描述方法）
- [ ] 阶段 3：任务已生成（/spec-kit:tasks 命令）
- [ ] 阶段 4：实施完成
- [ ] 阶段 5：验证通过

**门禁状态**：
- [x] 初始章程检查：通过（2 个警告项待研究阶段解决）
- [x] 设计后章程检查：通过（警告项已在研究和设计阶段解决）
- [x] 所有需要澄清的内容已解决（spec.md 澄清会话完成）
- [x] 复杂性偏差已记录（无违规，警告项已通过简化设计解决）

---
*基于章程 v2.1.1 - 参见 `.claude/memory/constitution.md`*