# 任务：构建版本信息注入与 Agent Docker 上报控制

**输入**：来自 `.claude/specs/008-commitid-commit-agent/` 的设计文档
**前提条件**：plan.md、research.md、data-model.md、contracts/version-api.md、quickstart.md

## 执行流程

本文档包含 33 个可执行任务，按依赖关系组织。标记 [P] 的任务可以并行执行。

### 项目结构

Web 应用（前端 + 后端）：
- 后端：`backend/`、`internal/`、`cmd/`、`pkg/`
- 前端：`ui/src/`
- 测试：`tests/contract/`、`tests/integration/`、`tests/unit/`
- 脚本：`scripts/`
- 构建：`Taskfile.yml`

## 阶段 3.1：构建基础设施

**优先级**：高（阻塞所有后续任务）

- [X] **T001** 创建 `scripts/version.sh` 版本提取脚本
  - **文件**：`scripts/version.sh`
  - **内容**：
    - 实现 git tag 提取逻辑（`git describe --tags --abbrev=0`）
    - 实现 commit 短 hash 提取（`git rev-parse --short=7 HEAD`）
    - 实现构建时间生成（`date -u +"%Y-%m-%dT%H:%M:%SZ"`）
    - 生成版本号格式：
      - 有 tag：`${TAG}-${COMMIT}`
      - 无 tag：`${DATE}-${COMMIT}`
      - 无 git：`dev` + `0000000`
    - 输出环境变量格式（`VERSION=...`、`BUILD_TIME=...`、`COMMIT_ID=...`）
    - 添加执行权限（`chmod +x`）
  - **参考**：research.md 第 2 节（版本提取逻辑）
  - **验收**：运行 `bash scripts/version.sh` 输出正确的环境变量格式

- [X] **T002** 修改 `Taskfile.yml` 集成版本脚本
  - **文件**：`Taskfile.yml`
  - **内容**：
    - 在 `build:server` 任务中：
      - 调用 `scripts/version.sh` 获取版本信息
      - 添加 `-ldflags` 参数注入 3 个变量
      - 示例：`-ldflags "-X github.com/ysicing/tiga/internal/version.Version=${VERSION} ..."`
    - 在 `build:agent` 任务中：
      - 同样集成版本注入逻辑
      - 确保 agent 二进制也包含版本信息
    - 保留现有构建配置（输出路径、编译参数等）
  - **参考**：research.md 第 2 节（Taskfile 集成）
  - **验收**：运行 `task backend` 无报错

- [ ] **T003** [P] 测试版本脚本（单元测试）
  - **文件**：`tests/unit/version_script_test.go`
  - **内容**：
    - 测试场景 1：有 git tag 场景（返回 `v1.2.3-a1b2c3d`）
    - 测试场景 2：无 git tag 场景（返回 `20251026-a1b2c3d`）
    - 测试场景 3：无 git 环境场景（返回 `dev` + `0000000`）
    - 验证环境变量输出格式
    - 验证脚本执行时间 <500ms
  - **参考**：quickstart.md 场景 1、边缘情况测试 1
  - **验收**：`go test -v ./tests/unit/version_script_test.go` 通过

## 阶段 3.2：测试优先（TDD）⚠️ 必须在 3.3 之前完成

**关键**：这些测试必须编写并且必须在任何实现之前失败

- [X] **T004** [P] 契约测试：版本 API 成功响应
  - **文件**：`tests/contract/version_api_test.go`
  - **内容**：
    - 测试 `GET /api/v1/version` 返回 200 OK
    - 验证响应头 `Content-Type: application/json`
    - 验证响应体包含 `version`、`build_time`、`commit_id` 字段
    - 验证字段非空
    - 使用 `httptest.NewRecorder()` 和 fake router
  - **参考**：contracts/version-api.md 测试用例 1
  - **验收**：测试编写完成后运行必须失败（因为端点未实现）

- [X] **T005** [P] 契约测试：版本 API Schema 验证
  - **文件**：`tests/contract/version_api_schema_test.go`
  - **内容**：
    - 验证 `version` 格式：正则 `^(v?\d+\.\d+\.\d+|\d{8}|dev|snapshot)(-[0-9a-f]{7})?$`
    - 验证 `build_time` 格式：RFC3339 或 "unknown"
    - 验证 `commit_id` 格式：7 位十六进制或 "0000000"
    - 使用 `regexp.MatchString()` 和 `time.Parse(time.RFC3339, ...)`
  - **参考**：contracts/version-api.md 测试用例 2、data-model.md 验证规则
  - **验收**：测试编写完成后运行必须失败

- [X] **T006** [P] 契约测试：版本 API 性能
  - **文件**：`tests/contract/version_api_performance_test.go`
  - **内容**：
    - 循环调用 API 100 次
    - 计算平均延迟
    - 断言平均延迟 <10ms
    - 断言所有请求返回 200 OK
  - **参考**：contracts/version-api.md 测试用例 3、quickstart.md 性能验证
  - **验收**：测试编写完成后运行必须失败

- [X] **T007** [P] 契约测试：版本 API 响应大小
  - **文件**：`tests/contract/version_api_size_test.go`
  - **内容**：
    - 调用 API 并获取响应体
    - 验证响应体大小 <500 bytes
    - 验证 JSON 格式正确
  - **参考**：contracts/version-api.md 测试用例 4
  - **验收**：测试编写完成后运行必须失败

- [ ] **T008** [P] 集成测试：完整构建流程
  - **文件**：`tests/integration/version_build_test.go`
  - **内容**：
    - 调用 `task clean && task backend` 构建项目
    - 验证 `bin/tiga` 和 `bin/tiga-agent` 存在
    - 运行 `./bin/tiga --version` 并解析输出
    - 验证版本信息格式正确
    - 清理构建产物
  - **参考**：quickstart.md 场景 1、集成测试
  - **验收**：测试编写完成后运行必须失败

- [ ] **T009** [P] 集成测试：启动日志版本显示
  - **文件**：`tests/integration/version_startup_test.go`
  - **内容**：
    - 启动 `./bin/tiga` 并捕获前 10 行日志
    - 验证日志包含 `version=`、`build_time=`、`commit_id=` 字段
    - 验证版本信息与构建时注入的值一致
    - 优雅停止进程
  - **参考**：quickstart.md 场景 2
  - **验收**：测试编写完成后运行必须失败

- [ ] **T010** [P] 集成测试：Agent 版本上报
  - **文件**：`tests/integration/agent_version_report_test.go`
  - **内容**：
    - 启动测试 gRPC 服务端（监听 VersionInfo 消息）
    - 启动 `./bin/tiga-agent` 并连接测试服务端
    - 验证服务端接收到 `VersionInfo` Proto 消息
    - 验证 `version`、`build_time`、`commit_id` 字段值
    - 清理进程和服务端
  - **参考**：quickstart.md 场景 7、data-model.md Proto 定义
  - **验收**：测试编写完成后运行必须失败

## 阶段 3.3：后端核心实现（仅在测试失败后）

**依赖**：T001-T010 测试已编写并失败

- [X] **T011** [P] 创建版本包
  - **文件**：`internal/version/version.go`
  - **内容**：
    - 定义包级变量：`Version`、`BuildTime`、`CommitID`（默认值 "dev"、"unknown"、"0000000"）
    - 定义 `Info` 结构体（包含 3 个字段，JSON tag）
    - 实现 `GetInfo() Info` 函数返回版本信息
    - 添加包文档注释说明编译时注入机制
  - **参考**：research.md 第 1 节、data-model.md VersionInfo 定义
  - **验收**：代码编译无错误

- [ ] **T012** [P] 版本包单元测试
  - **文件**：`internal/version/version_test.go`
  - **内容**：
    - 测试 `GetInfo()` 返回正确的结构体
    - 测试默认值处理（Version="dev" 等）
    - 测试 JSON 序列化（`json.Marshal(GetInfo())`）
    - 验证字段类型和非空性
  - **参考**：data-model.md 验证规则
  - **验收**：`go test -v ./internal/version/` 通过

- [X] **T013** 修改服务端主程序添加版本显示
  - **文件**：`cmd/tiga/main.go`
  - **内容**：
    - 导入 `internal/version` 包
    - 在 `main()` 函数开头处理 `--version` 参数：
      - 检查 `os.Args[1] == "--version"` 或 `"version"`
      - 格式化输出版本信息（4 行：标题、Version、BuildTime、CommitID）
      - 调用 `os.Exit(0)`
    - 在启动日志中添加版本信息：
      - 使用 `log.WithFields()` 输出 version、build_time、commit_id
      - 位置：在 "Starting Tiga Server" 日志中
  - **参考**：research.md 第 6 节、quickstart.md 场景 2 和 3
  - **验收**：运行 `./bin/tiga --version` 显示版本信息并退出

- [X] **T014** 修改 Agent 主程序添加版本显示
  - **文件**：`cmd/tiga-agent/main.go`
  - **内容**：
    - 同 T013，但针对 Agent：
      - 处理 `--version` 参数
      - 输出 "Tiga Agent" 标题
      - 启动日志显示版本信息
  - **参考**：research.md 第 6 节、quickstart.md 场景 2 和 3
  - **验收**：运行 `./bin/tiga-agent --version` 显示版本信息并退出

- [X] **T015** [P] 创建版本 API 处理器
  - **文件**：`internal/api/handlers/version/handler.go`
  - **内容**：
    - 导入 `internal/version` 包和 `gin` 框架
    - 实现 `GetVersion(c *gin.Context)` 处理器：
      - 调用 `version.GetInfo()`
      - 返回 `c.JSON(200, info)`
    - 添加 Swagger 注解：
      - `@Summary 获取服务端版本信息`
      - `@Tags system`
      - `@Produce json`
      - `@Success 200 {object} version.Info`
      - `@Router /api/v1/version [get]`
  - **参考**：research.md 第 5 节、contracts/version-api.md
  - **验收**：代码编译无错误

- [X] **T016** 在路由中注册版本 API
  - **文件**：`internal/api/routes.go`
  - **内容**：
    - 导入 `internal/api/handlers/version` 包
    - 在 API v1 路由组中添加：
      - `v1.GET("/version", version.GetVersion)`
    - 位置：无需认证路由组（公开访问）
    - 确保不应用认证中间件
  - **参考**：research.md 第 5 节、contracts/version-api.md（无需认证）
  - **验收**：启动服务后 `curl http://localhost:12306/api/v1/version` 返回 JSON

- [X] **T017** 验证契约测试通过
  - **文件**：无（验证步骤）
  - **内容**：
    - 运行 `go test -v ./tests/contract/version_*.go`
    - 验证所有 4 个契约测试通过（T004-T007）
    - 如失败，修复实现代码
  - **参考**：contracts/version-api.md 所有测试用例
  - **验收**：所有契约测试通过

## 阶段 3.4：Proto 和 gRPC 实现

**依赖**：T011-T014（版本包已实现）

- [X] **T018** 修改 Proto 定义添加 VersionInfo
  - **文件**：`proto/host_monitor.proto`
  - **内容**：
    - 添加 `VersionInfo` 消息定义（3 个字符串字段）
    - 在 `HostState` 消息中添加：
      - `optional VersionInfo version_info = 20;`
    - 使用较大的字段号（20）避免与现有字段冲突
    - 添加注释说明字段用途
  - **参考**：research.md 第 3 节、data-model.md Proto 定义
  - **验收**：运行 `task proto` 或 `make proto` 重新生成 Go 代码无错误

- [X] **T019** Agent 上报版本信息
  - **文件**：`cmd/tiga-agent/main.go`（或 gRPC 上报逻辑文件）
  - **内容**：
    - 导入 `internal/version` 包
    - 在 `ReportState()` 或类似函数中：
      - 创建 `VersionInfo` Proto 消息
      - 填充 `Version`、`BuildTime`、`CommitId` 字段（从 `version` 包获取）
      - 添加到 `HostState` 消息的 `version_info` 字段
    - 测试序列化（`proto.Marshal()`）
  - **参考**：research.md 第 3 节、data-model.md Proto 使用示例
  - **验收**：运行 Agent 并查看 gRPC 消息包含 version_info

- [X] **T020** 服务端接收并记录 Agent 版本
  - **文件**：`internal/services/host/` 或 gRPC 处理器文件
  - **内容**：
    - 在 `HostState` 处理函数中：
      - 检查 `state.VersionInfo` 是否存在（向后兼容）
      - 提取 `version`、`build_time`、`commit_id`
      - 记录 INFO 级别日志：
        - `log.WithFields(log.Fields{"agent_version": ...}).Info("Agent connected")`
    - 可选：存储到数据库（如 `HostNode` 表添加版本字段）
  - **参考**：quickstart.md 场景 7、data-model.md 数据流
  - **验收**：启动 Agent 后服务端日志显示 agent_version 信息

## 阶段 3.5：配置系统扩展

**依赖**：无（独立任务）

- [X] **T021** [P] 扩展 AgentConfig 添加 DisableDockerReport
  - **文件**：`internal/config/config.go`
  - **内容**：
    - 在 `AgentConfig` 结构体中添加字段：
      - `DisableDockerReport bool`
      - YAML tag: `yaml:"disable_docker_report"`
      - Env tag: `env:"AGENT_DISABLE_DOCKER_REPORT"`
    - 添加字段注释说明默认值和用途
    - 确保配置加载函数支持环境变量覆盖
  - **参考**：research.md 第 4 节、data-model.md AgentConfig 定义
  - **验收**：代码编译无错误

- [X] **T022** Agent 实现 Docker 上报控制逻辑
  - **文件**：`cmd/tiga-agent/docker_handler.go`（或类似文件）
  - **内容**：
    - 实现 `shouldReportDocker()` 函数：
      - 返回 `!cfg.Agent.DisableDockerReport`
    - 在 Docker 实例上报逻辑前添加检查：
      - `if !shouldReportDocker() { log.Info("Docker instance reporting disabled"); return }`
    - 确保其他上报逻辑（主机状态等）不受影响
  - **参考**：research.md 第 4 节、quickstart.md 场景 4
  - **验收**：配置禁用后 Agent 日志显示 "Docker instance reporting disabled"

- [ ] **T023** [P] 配置加载测试
  - **文件**：`tests/unit/config_test.go`
  - **内容**：
    - 测试场景 1：YAML 配置加载（`disable_docker_report: true`）
    - 测试场景 2：环境变量覆盖（`AGENT_DISABLE_DOCKER_REPORT=true`）
    - 测试场景 3：默认值（字段缺失时 `DisableDockerReport = false`）
    - 验证配置优先级：环境变量 > YAML > 默认值
  - **参考**：data-model.md 配置来源优先级、quickstart.md 边缘情况测试 2
  - **验收**：`go test -v ./tests/unit/config_test.go` 通过

## 阶段 3.6：前端实现

**依赖**：T015-T016（版本 API 已实现）

- [X] **T024** [P] 创建 TypeScript 类型定义
  - **文件**：`ui/src/types/version.ts`
  - **内容**：
    - 定义 `VersionInfo` 接口（3 个字符串字段）
    - 添加 JSDoc 注释说明字段格式
    - 导出接口
  - **参考**：contracts/version-api.md TypeScript 定义
  - **验收**：TypeScript 编译无错误

- [X] **T025** [P] 创建版本 API 客户端
  - **文件**：`ui/src/services/version.ts`
  - **内容**：
    - 导入 `axios` 和 `VersionInfo` 类型
    - 实现 `versionAPI` 对象：
      - `async getVersion(): Promise<VersionInfo>` 方法
      - 调用 `axios.get<VersionInfo>('/api/v1/version')`
      - 返回 `response.data`
    - 添加错误处理
  - **参考**：contracts/version-api.md API 客户端示例
  - **验收**：TypeScript 编译无错误

- [X] **T026** 在设置页面添加版本信息显示
  - **文件**：`ui/src/pages/settings/` 或类似页面文件
  - **内容**：
    - 导入 `useQuery` 和 `versionAPI`
    - 使用 TanStack Query 获取版本：
      - `const { data: version, isLoading } = useQuery({ queryKey: ['version'], queryFn: versionAPI.getVersion, staleTime: 5*60*1000 })`
    - 渲染 UI 组件显示：
      - 版本号（`version.version`）
      - 构建时间（格式化为本地时间）
      - Commit ID（`version.commit_id`）
    - 添加加载状态处理
  - **参考**：contracts/version-api.md React 组件示例、quickstart.md 场景 6
  - **验收**：访问设置页面显示版本信息

## 阶段 3.7：集成测试

**依赖**：所有实现任务完成（T011-T026）

- [ ] **T027** [P] 端到端版本 API 测试
  - **文件**：`tests/integration/version_e2e_test.go`
  - **内容**：
    - 启动完整的服务端（包括数据库、路由等）
    - 调用真实的 `GET /api/v1/version` API
    - 验证响应状态码、头部、JSON 格式
    - 验证版本信息与环境变量或构建参数一致
    - 清理测试环境
  - **参考**：quickstart.md 场景 6、集成测试流程
  - **验收**：`go test -v ./tests/integration/version_e2e_test.go` 通过

- [ ] **T028** [P] 配置文件集成测试
  - **文件**：`tests/integration/config_docker_report_test.go`
  - **内容**：
    - 创建测试配置文件（`disable_docker_report: true`）
    - 启动 Agent 并传入配置文件路径
    - 捕获 Agent 日志
    - 验证日志包含 "Docker instance reporting disabled"
    - 清理配置文件和进程
  - **参考**：quickstart.md 场景 4
  - **验收**：测试通过

- [ ] **T029** [P] 无 git 环境构建测试
  - **文件**：`tests/integration/version_no_git_test.go`
  - **内容**：
    - 创建临时目录并复制代码（不包括 `.git`）
    - 在临时目录执行 `task backend`
    - 验证构建成功
    - 验证 `./bin/tiga --version` 输出 "dev" 和 "0000000"
    - 清理临时目录
  - **参考**：quickstart.md 边缘情况测试 1
  - **验收**：测试通过

- [ ] **T030** [P] 性能基准测试
  - **文件**：`tests/integration/version_benchmark_test.go`
  - **内容**：
    - 使用 ApacheBench 或 Go benchmark 测试版本 API
    - 并发 10 连接，总计 1000 请求
    - 验证平均延迟 <10ms
    - 验证吞吐量 >1000 req/s
    - 记录性能指标
  - **参考**：quickstart.md 性能验证测试 1
  - **验收**：性能指标达标

## 阶段 3.8：优化和文档

**依赖**：所有测试通过（T027-T030）

- [X] **T031** 生成 Swagger 文档
  - **文件**：无（执行脚本）
  - **内容**：
    - 运行 `./scripts/generate-swagger.sh`
    - 验证生成的 `docs/swagger.json` 或 `docs/swagger.yaml` 包含 `/api/v1/version` 端点
    - 启动服务并访问 `http://localhost:12306/swagger/index.html`
    - 测试 Swagger UI 中的版本 API
  - **参考**：quickstart.md 回归测试清单
  - **验收**：Swagger UI 正确显示版本 API 文档

- [X] **T032** 更新项目 README
  - **文件**：`README.md`
  - **内容**：
    - 添加 "版本查询" 章节：
      - 说明 `--version` 命令行参数用法
      - 说明版本 API 端点（`GET /api/v1/version`）
      - 示例输出
    - 更新构建说明：
      - 说明版本信息自动注入机制
      - 说明无 git 环境的默认值
  - **参考**：quickstart.md 场景 3 和 6
  - **验收**：README 更新完成

- [X] **T033** 执行完整验收测试
  - **文件**：无（手动测试）
  - **内容**：
    - 按照 `quickstart.md` 执行所有 7 个验收场景
    - 执行 2 个边缘情况测试
    - 执行 2 个性能验证
    - 记录测试结果（通过/失败、实际性能指标）
    - 修复任何失败的场景
  - **参考**：quickstart.md 完整文档
  - **验收**：所有验收场景通过

## 依赖关系图

```
构建基础设施（T001-T003）
    ↓
测试优先/TDD（T004-T010）[P]
    ↓
    ├─→ 后端核心（T011-T017）
    │       ↓
    │   Proto/gRPC（T018-T020）
    │       ↓
    │   前端实现（T024-T026）[P]
    │
    └─→ 配置系统（T021-T023）[P]
            ↓
    集成测试（T027-T030）[P]
            ↓
    优化/文档（T031-T033）
```

## 并行执行示例

### 第 1 组：测试优先（T004-T010）
所有契约测试和集成测试可以同时编写（不同文件）：
```bash
# 在 Claude Code 中同时启动 7 个 Task 代理：
Task: "在 tests/contract/version_api_test.go 中测试版本 API 成功响应"
Task: "在 tests/contract/version_api_schema_test.go 中测试版本 API Schema 验证"
Task: "在 tests/contract/version_api_performance_test.go 中测试版本 API 性能"
Task: "在 tests/contract/version_api_size_test.go 中测试版本 API 响应大小"
Task: "在 tests/integration/version_build_test.go 中测试完整构建流程"
Task: "在 tests/integration/version_startup_test.go 中测试启动日志版本显示"
Task: "在 tests/integration/agent_version_report_test.go 中测试 Agent 版本上报"
```

### 第 2 组：后端核心（T011-T012）
版本包和测试可以并行：
```bash
Task: "在 internal/version/version.go 中创建版本包"
Task: "在 internal/version/version_test.go 中编写版本包单元测试"
```

### 第 3 组：配置系统（T021, T023）
配置扩展和测试可以并行：
```bash
Task: "在 internal/config/config.go 中扩展 AgentConfig"
Task: "在 tests/unit/config_test.go 中编写配置加载测试"
```

### 第 4 组：前端实现（T024-T025）
TypeScript 类型和 API 客户端可以并行：
```bash
Task: "在 ui/src/types/version.ts 中创建 TypeScript 类型定义"
Task: "在 ui/src/services/version.ts 中创建版本 API 客户端"
```

### 第 5 组：集成测试（T027-T030）
所有集成测试可以同时运行（不同文件）：
```bash
Task: "在 tests/integration/version_e2e_test.go 中进行端到端测试"
Task: "在 tests/integration/config_docker_report_test.go 中测试配置文件"
Task: "在 tests/integration/version_no_git_test.go 中测试无 git 环境构建"
Task: "在 tests/integration/version_benchmark_test.go 中进行性能基准测试"
```

## 注意事项

**[P] 标记规则**：
- 不同文件 = 可并行 [P]
- 相同文件 = 顺序执行（无 [P]）
- 例如：T013 和 T014 修改不同文件，但依赖 T011 完成

**TDD 流程**：
- ⚠️ 必须先编写测试（T004-T010）
- 验证测试失败（红色）
- 然后实现功能（T011-T026）
- 最后验证测试通过（绿色）

**提交策略**：
- 每个任务完成后提交一次
- 提交消息格式：`[008-commitid-commit-agent] T001: 创建版本提取脚本`
- 关键里程碑（如所有测试通过）打 tag

**避免事项**：
- ❌ 不要在同一个 [P] 任务组中修改同一文件
- ❌ 不要跳过测试直接实现
- ❌ 不要创建模糊的任务（每个任务必须有明确的文件路径和验收标准）

## 验证清单

在开始执行前验证：
- [x] 所有契约都有对应的测试（T004-T007 → contracts/version-api.md）
- [x] 所有实体都有模型任务（VersionInfo → T011、AgentConfig → T021）
- [x] 所有测试都在实现之前（T004-T010 在 T011-T026 之前）
- [x] 并行任务真正独立（所有 [P] 任务修改不同文件）
- [x] 每个任务指定确切的文件路径
- [x] 没有任务修改与另一个 [P] 任务相同的文件

## 预计工作量

- **阶段 3.1**（构建基础）：2-3 小时
- **阶段 3.2**（测试优先）：3-4 小时
- **阶段 3.3**（后端核心）：2-3 小时
- **阶段 3.4**（Proto/gRPC）：1-2 小时
- **阶段 3.5**（配置系统）：1-2 小时
- **阶段 3.6**（前端实现）：2-3 小时
- **阶段 3.7**（集成测试）：2-3 小时
- **阶段 3.8**（优化/文档）：1-2 小时

**总计**：14-22 小时（约 2-3 个工作日）

**并行化收益**：如果多人协作，理论上可压缩至 1 个工作日（8 小时）

---

**下一步**：按顺序执行任务，或使用 Task 代理并行执行标记 [P] 的任务组。
