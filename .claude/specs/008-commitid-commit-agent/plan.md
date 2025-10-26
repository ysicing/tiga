# 实施计划：构建版本信息注入与 Agent Docker 上报控制

**分支**：`008-commitid-commit-agent` | **日期**：2025-10-26 | **规格**：[spec.md](./spec.md)
**输入**：来自 `.claude/specs/008-commitid-commit-agent/spec.md` 的功能规格

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

本功能为 Tiga 平台添加构建时版本信息注入和 Agent Docker 上报控制功能：

**核心需求**：
1. **版本信息注入**：在构建时将版本号（git tag/日期 + commit 短 hash）、构建时间和 commit ID 注入到 server 和 agent 二进制文件中
2. **启动显示版本**：应用启动时在日志中显示版本信息，支持 `--version` 命令行选项
3. **服务端 API**：提供 HTTP API 接口暴露服务端版本信息供前端页面显示
4. **Agent 版本上报**：Agent 启动或定期上报时将自身版本信息上报到服务端
5. **Docker 上报控制**：Agent 支持通过配置禁用 Docker 实例上报（默认启用）

**技术方法**：
- 使用 Go 编译时 `-ldflags` 注入版本变量
- Taskfile 构建脚本集成 git 命令获取版本信息
- Proto 扩展支持 agent 版本字段
- 配置系统添加 docker 上报控制开关

## 技术上下文
**语言/版本**：Go 1.24+
**主要依赖**：
- 后端：Gin (HTTP 框架)、GORM (ORM)、gRPC (Agent 通信)、logrus (日志)
- 前端：React 19、TypeScript、TanStack Query
**存储**：不适用（版本信息存储在二进制文件和内存中）
**测试**：Go 测试框架（单元测试 + 集成测试）
**目标平台**：Linux amd64/arm64（服务端和 Agent）
**项目类型**：web（前端 + 后端）
**性能目标**：版本查询 API <10ms，构建脚本 <5秒额外耗时
**约束**：
- 必须兼容现有 Taskfile 构建流程
- 版本信息不能增加二进制文件显著体积（<1KB）
- 支持无 git 环境构建（使用默认值）
**规模/范围**：
- 2 个二进制文件（tiga server、tiga-agent）
- 1 个 HTTP API 端点
- 1 个 Proto 消息扩展
- 1 个配置选项

## 章程检查
*门禁：必须在阶段 0 研究之前通过。在阶段 1 设计后重新检查。*

**注意**：项目未配置宪章文件（`.claude/memory/constitution.md`），跳过章程检查。

遵循以下基本原则：
- ✅ KISS 原则：使用 Go 标准 `-ldflags` 机制，无需额外复杂工具
- ✅ 最小改动：仅修改 Taskfile、添加版本包、扩展 Proto
- ✅ 向后兼容：配置项缺失时使用安全默认值
- ✅ 可测试性：版本信息可通过单元测试验证

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
# Web 应用程序结构（前端 + 后端）

backend/
├── cmd/
│   ├── tiga/                    # 服务端主程序
│   │   └── main.go              # [修改] 添加版本显示
│   └── tiga-agent/              # Agent 主程序
│       └── main.go              # [修改] 添加版本显示
├── internal/
│   ├── api/handlers/
│   │   └── version/             # [新增] 版本 API 处理器
│   ├── config/
│   │   └── config.go            # [修改] 添加 agent.disable_docker_report 配置
│   └── version/                 # [新增] 版本信息包
│       ├── version.go           # 版本变量定义
│       └── version_test.go      # 版本测试
├── proto/
│   └── host_monitor.proto       # [修改] 添加 agent 版本字段
└── pkg/
    └── version/                 # [新增] 可复用版本工具（如需要）

frontend/
└── ui/src/
    ├── pages/
    │   └── settings/            # [修改] 添加版本信息显示
    └── services/
        └── version.ts           # [新增] 版本 API 客户端

Taskfile.yml                     # [修改] 添加版本注入逻辑
scripts/
└── version.sh                   # [新增] 版本信息提取脚本

tests/
├── contract/
│   └── version_api_test.go      # [新增] 版本 API 契约测试
├── integration/
│   └── version_test.go          # [新增] 版本集成测试
└── unit/
    └── version/
        └── version_test.go      # [新增] 版本单元测试
```

**结构决策**：
- 使用 `internal/version/` 包存储版本变量，避免循环依赖
- 版本 API 处理器放在 `internal/api/handlers/version/`
- 构建脚本逻辑放在 `scripts/version.sh`，由 Taskfile 调用
- Proto 文件直接在现有 `proto/host_monitor.proto` 中扩展

## 阶段 0：概述与研究

**状态**：✅ 完成

**研究任务已完成**：
1. Go 编译时版本注入机制（-ldflags -X）
2. Taskfile 版本信息提取脚本设计
3. gRPC Proto 消息扩展模式
4. Agent 配置系统扩展方案
5. 服务端版本 API 设计
6. 命令行版本显示实现

**关键决策**：
- 使用 Go `-ldflags -X` 注入版本变量（Version、BuildTime、CommitID）
- 创建 `scripts/version.sh` 脚本提取 git 信息，由 Taskfile 调用
- 在 `HostState` Proto 消息中添加 `optional VersionInfo` 字段
- 配置新增 `agent.disable_docker_report` 字段（默认 false）
- 新增 `GET /api/v1/version` API 端点
- 支持 `--version` 和 `version` 子命令

**输出**：`research.md`（已创建）

## 阶段 1：设计与契约
*前提条件：research.md 完成*

**状态**：✅ 完成

**已完成的设计文档**：

1. **data-model.md**（已创建）：
   - VersionInfo 内存结构定义
   - VersionInfo Proto 消息定义
   - AgentConfig 配置结构扩展
   - API 响应数据结构
   - 数据流和关系图

2. **contracts/version-api.md**（已创建）：
   - GET /api/v1/version 端点契约
   - HTTP 请求/响应规范
   - OpenAPI 3.0 规范
   - JSON Schema 验证
   - 契约测试用例（4个场景）
   - 前端集成示例（TypeScript）

3. **quickstart.md**（已创建）：
   - 7 个验收场景详细步骤
   - 边缘情况测试（2个）
   - 性能验证（2个）
   - 集成测试流程
   - 故障排查指南
   - 回归测试清单

**设计亮点**：
- 零数据库依赖，版本信息存储在二进制文件
- 向后兼容的 Proto 扩展（optional 字段）
- 无需认证的版本 API（便于监控）
- 双向配置（YAML + 环境变量）
- 完整的测试覆盖（契约、性能、边缘情况）

**输出**：data-model.md、contracts/version-api.md、quickstart.md（已创建）

**注意**：本功能不需要代理文件更新，因为：
- 版本信息注入是构建时特性，不涉及运行时代码复杂度
- 配置扩展仅增加一个简单布尔字段
- 无需专门的上下文指导

## 阶段 2：任务规划方法
*本节描述 /spec-kit:tasks 命令将执行的操作 - 不要在 /spec-kit:plan 期间执行*

**状态**：✅ 方法已规划（tasks.md 由 /spec-kit:tasks 命令生成）

**任务生成策略**：

本功能将生成以下类别的任务：

### 1. 构建基础设施任务（优先级：高）

**T001-T003：版本脚本和 Taskfile 集成** [P]
- T001：创建 `scripts/version.sh` 脚本
  - 实现 git tag、commit、date 提取逻辑
  - 处理无 git 环境的默认值
  - 输出环境变量格式
- T002：修改 `Taskfile.yml` 集成版本脚本
  - 在 `build:server` 任务中添加 -ldflags
  - 在 `build:agent` 任务中添加 -ldflags
  - 测试构建流程
- T003：测试版本脚本（单元测试）
  - 测试有 tag 场景
  - 测试无 tag 场景
  - 测试无 git 场景

### 2. 后端核心实现任务（优先级：高）

**T004-T007：版本包实现** [P]
- T004：创建 `internal/version/version.go`
  - 定义 Version、BuildTime、CommitID 变量
  - 实现 GetInfo() 函数
  - 添加包文档注释
- T005：创建 `internal/version/version_test.go`
  - 测试 GetInfo() 返回正确结构
  - 测试默认值处理
- T006：修改 `cmd/tiga/main.go` 添加版本显示
  - 处理 --version 参数
  - 启动日志打印版本
  - 测试命令行参数
- T007：修改 `cmd/tiga-agent/main.go` 添加版本显示
  - 同 T006，但针对 Agent

**T008-T010：版本 API 实现**
- T008：创建 `internal/api/handlers/version/handler.go`
  - 实现 GetVersion() 处理器
  - 添加 Swagger 注解
  - 返回 JSON 响应
- T009：在 `internal/api/routes.go` 注册路由
  - 添加 GET /api/v1/version 路由
  - 无需认证中间件
- T010：创建契约测试 `tests/contract/version_api_test.go`
  - 测试 HTTP 200 响应
  - 测试 JSON Schema 验证
  - 测试性能（<10ms）
  - 测试响应体大小（<500 bytes）

### 3. Proto 和 gRPC 实现任务（优先级：高）

**T011-T013：Proto 扩展**
- T011：修改 `proto/host_monitor.proto`
  - 添加 VersionInfo 消息定义
  - 在 HostState 中添加 optional version_info 字段
  - 重新生成 Go 代码（`make proto` 或 `task proto`）
- T012：修改 `cmd/tiga-agent/main.go` 上报版本
  - 在 ReportState() 中填充 version_info
  - 测试 gRPC 消息序列化
- T013：修改服务端接收版本信息
  - 在 HostState 处理器中记录 agent 版本
  - 添加日志输出（INFO 级别）

### 4. 配置系统扩展任务（优先级：中）

**T014-T016：Agent 配置扩展**
- T014：修改 `internal/config/config.go`
  - 在 AgentConfig 中添加 DisableDockerReport 字段
  - 添加 YAML tag 和 env tag
  - 添加配置注释
- T015：修改 Agent Docker 上报逻辑
  - 在 `cmd/tiga-agent/docker_handler.go` 中检查配置
  - 实现 shouldReportDocker() 函数
  - 添加日志："Docker instance reporting disabled"
- T016：测试配置加载
  - 测试 YAML 配置
  - 测试环境变量覆盖
  - 测试默认值（false）

### 5. 前端实现任务（优先级：中）

**T017-T019：前端版本显示** [P]
- T017：创建 `ui/src/types/version.ts`
  - 定义 VersionInfo TypeScript 类型
- T018：创建 `ui/src/services/version.ts`
  - 实现 getVersion() API 客户端
  - 使用 axios
- T019：添加版本显示到设置页面
  - 在 `ui/src/pages/settings/` 中添加版本信息组件
  - 使用 TanStack Query 获取版本
  - 格式化显示（版本、构建时间、Commit）

### 6. 测试任务（优先级：中）

**T020-T023：集成测试** [P]
- T020：创建 `tests/integration/version_test.go`
  - 测试完整构建流程
  - 测试版本注入
  - 测试 API 端到端
- T021：创建 `tests/integration/config_test.go`
  - 测试配置加载
  - 测试环境变量优先级
- T022：边缘情况测试
  - 测试无 git 环境构建
  - 测试构建脚本在不同 shell 下执行
- T023：性能基准测试
  - 版本 API 压力测试（ApacheBench）
  - 构建脚本性能测试

### 7. 文档和清理任务（优先级：低）

**T024-T026：文档和验收**
- T024：更新项目 README
  - 添加版本查询命令说明
  - 更新构建说明
- T025：生成 Swagger 文档
  - 运行 `./scripts/generate-swagger.sh`
  - 验证 /api/v1/version 端点文档
- T026：执行 quickstart.md 验收测试
  - 完成所有 7 个验收场景
  - 记录测试结果

**排序策略**：

1. **构建基础设施优先**（T001-T003）：
   - 没有版本脚本和 Taskfile 集成，后续所有任务无法测试

2. **后端核心并行**（T004-T010）：
   - 版本包（T004-T007）和版本 API（T008-T010）可并行开发
   - 标记 [P] 表示可并行执行

3. **Proto 和配置依赖后端**（T011-T016）：
   - 需要版本包完成后才能引用

4. **前端依赖后端 API**（T017-T019）：
   - 需要版本 API 实现完成
   - 可与 Proto 任务并行

5. **测试最后**（T020-T023）：
   - 需要所有实现完成

6. **文档和验收最后**（T024-T026）：
   - 需要所有功能和测试完成

**依赖关系图**：

```
T001-T003 (构建脚本)
    ↓
T004-T007 (版本包) [P]  ←→  T008-T010 (版本API) [P]
    ↓                           ↓
T011-T013 (Proto)           T017-T019 (前端) [P]
    ↓                           ↓
T014-T016 (配置)             ───┘
    ↓
T020-T023 (测试) [P]
    ↓
T024-T026 (文档)
```

**预计任务数量**：26 个任务

**预计实施时间**：
- 构建基础（T001-T003）：2 小时
- 后端核心（T004-T010）：4 小时
- Proto 和配置（T011-T016）：3 小时
- 前端（T017-T019）：2 小时
- 测试（T020-T023）：3 小时
- 文档（T024-T026）：1 小时
- **总计**：15 小时（约 2 个工作日）

**并行化潜力**：
- 标记 [P] 的任务可并行执行
- 理论上可压缩到 1 个工作日（如果多人协作）

**重要**：此阶段由 `/spec-kit:tasks` 命令执行，将生成 `tasks.md` 文件包含详细的任务列表

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
- [x] 阶段 3：任务已生成（/spec-kit:tasks 命令）
- [ ] 阶段 4：实施完成
- [ ] 阶段 5：验证通过

**门禁状态**：
- [x] 初始章程检查：通过（无宪章文件，遵循基本原则）
- [x] 设计后章程检查：通过
- [x] 所有需要澄清的内容已解决
- [x] 复杂性偏差已记录（无偏差）

**输出文件清单**：
- [x] plan.md（本文件）
- [x] research.md
- [x] data-model.md
- [x] contracts/version-api.md
- [x] quickstart.md
- [x] tasks.md（由 /spec-kit:tasks 命令生成）

---
*基于章程 v2.1.1 - 参见 `.claude/memory/constitution.md`*
*注：项目未配置宪章文件，遵循 KISS 原则和基本最佳实践*