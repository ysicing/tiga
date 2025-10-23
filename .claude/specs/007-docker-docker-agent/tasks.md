# 任务清单：Docker实例远程管理

**功能分支**：`007-docker-docker-agent`
**创建日期**：2025-10-22
**输入**：来自 `plan.md`、`data-model.md`、`contracts/`、`quickstart.md`
**预计任务数**：60个任务，20个可并行

---

## 执行说明

**TDD原则**：
- ✅ 契约测试必须在实现之前编写并失败
- ✅ 集成测试在核心实现后运行
- ✅ 每个任务完成后提交代码

**并行执行**：
- `[P]` 标记表示可以与其他 `[P]` 任务并行执行
- 不同文件、无依赖关系的任务可并行
- 同一文件的修改必须顺序执行

**任务编号**：T001 ~ T060

---

## Phase A: 数据层（8个任务，4个并行）

### 基础设施

- [X] **T001** 创建项目目录结构
  - 创建 `internal/models/docker_*.go`
  - 创建 `internal/repository/docker/`
  - 创建 `pkg/grpc/proto/docker/`
  - 创建 `internal/api/handlers/docker/`
  - 创建 `ui/src/pages/docker/`
  - **文件**：项目根目录

- [X] **T002** 安装Go依赖
  - Docker SDK for Go v27.x
  - gRPC和protobuf代码生成工具
  - testcontainers-go（用于集成测试）
  - **文件**：`go.mod`

- [X] **T003** [P] 安装前端依赖
  - xterm.js v5.3+（终端UI）
  - xterm-addon-fit（自适应大小）
  - xterm-addon-attach（WebSocket附加）
  - **文件**：`ui/package.json`

### 数据模型

- [X] **T004** [P] 创建DockerInstance模型
  - 定义GORM模型结构
  - 添加状态常量（online/offline/archived）
  - 实现辅助方法（IsOnline、CanOperate、MarkOffline等）
  - **文件**：`internal/models/docker_instance.go`

- [X] **T005** [P] 扩展AuditLog模型
  - 添加Docker操作类型常量（container_start、container_stop等）
  - 添加资源类型常量（docker_container、docker_image、docker_instance）
  - 实现DockerOperationDetails结构体
  - 添加辅助函数（NewDockerAuditLog、ParseDockerDetails）
  - **文件**：`internal/models/audit_log.go`

- [X] **T006** [P] 创建Container和Image模型
  - 定义Container结构体（非持久化）
  - 定义ContainerPort、ContainerMount、ContainerNetwork子结构
  - 定义ContainerStats结构体
  - 定义Image结构体
  - 定义ImageConfig、ImageHistory子结构
  - 添加辅助方法（IsRunning、CanStart、GetMainTag等）
  - **文件**：`internal/models/docker_container.go`、`internal/models/docker_image.go`

### 仓储层

- [X] **T007** [P] 定义DockerInstanceRepository接口
  - 定义接口方法（Create、Update、Delete、GetByID等）
  - 定义查询过滤器结构（DockerInstanceFilters）
  - 添加批量操作方法（UpdateHealthStatus、MarkAllInstancesOfflineByAgentID）
  - **文件**：`internal/repository/interfaces.go`

- [X] **T008** 实现DockerInstanceRepository
  - 实现所有接口方法
  - 使用GORM查询构建器
  - 添加分页和排序支持
  - **文件**：`internal/repository/docker/instance_repository.go`
  - **依赖**：T004、T007

---

## Phase B: Agent层（15个任务，3个并行）

### gRPC协议定义

- [X] **T009** [P] 编写docker.proto
  - 定义DockerService服务（15个RPC方法）
  - 定义消息类型（GetDockerInfoRequest、ListContainersRequest等）
  - 定义流式消息（ExecRequest、ExecResponse、LogEntry等）
  - **文件**：`pkg/grpc/proto/docker/docker.proto`

- [X] **T010** 生成gRPC代码
  - 运行protoc命令生成Go代码
  - 验证生成的docker.pb.go和docker_grpc.pb.go
  - **文件**：`pkg/grpc/proto/docker/docker.pb.go`、`pkg/grpc/proto/docker/docker_grpc.pb.go`
  - **依赖**：T009

### Agent端实现

- [X] **T011** [P] 初始化Docker客户端
  - 实现NewDockerClient函数
  - 添加API版本检查（最低v1.41）
  - 添加错误处理和日志记录
  - **文件**：`agent/internal/docker/client.go`

- [X] **T012** 实现GetDockerInfo RPC
  - 调用docker.ServerVersion()
  - 构造DockerInfo响应
  - **文件**：`agent/internal/docker/info_service.go`
  - **依赖**：T010、T011

- [X] **T013** 实现ListContainers RPC
  - 调用docker.ContainerList()
  - 实现分页逻辑（切片）
  - 实现filter过滤（Docker原生filter）
  - **文件**：`agent/internal/docker/container_service.go`
  - **依赖**：T010、T011

- [X] **T014** 实现容器生命周期RPC
  - 实现StartContainer、StopContainer、RestartContainer
  - 实现PauseContainer、UnpauseContainer
  - 实现DeleteContainer（支持force和remove_volumes）
  - **文件**：`agent/internal/docker/container_service.go`
  - **依赖**：T010、T011

- [X] **T015** 实现GetContainerStats流式RPC
  - 调用docker.ContainerStats()
  - 解析Stats JSON并转换为protobuf
  - 实现流式推送（每秒一次）
  - **文件**：`agent/internal/docker/container_service.go`
  - **依赖**：T010、T011

- [X] **T016** 实现GetContainerLogs流式RPC
  - 调用docker.ContainerLogs()
  - 支持历史日志（tail参数）和实时跟踪（follow参数）
  - 实现流式推送
  - **文件**：`agent/internal/docker/container_service.go`
  - **依赖**：T010、T011

- [X] **T017** 实现ExecContainer双向流RPC
  - 处理ExecStart消息（创建docker exec会话）
  - 处理ExecInput消息（转发到stdin）
  - 处理ExecResize消息（调整TTY大小）
  - 读取stdout/stderr并发送ExecOutput消息
  - 处理进程退出并发送ExecExit消息
  - **文件**：`agent/internal/docker/exec_service.go`
  - **依赖**：T010、T011

- [X] **T018** 实现ListImages RPC
  - 调用docker.ImageList()
  - 支持filter过滤（dangling、reference等）
  - **文件**：`agent/internal/docker/image_service.go`
  - **依赖**：T010、T011

- [X] **T019** 实现GetImage RPC
  - 调用docker.ImageInspect()
  - 包含完整元数据（config、history等）
  - **文件**：`agent/internal/docker/image_service.go`
  - **依赖**：T010、T011

- [X] **T020** 实现DeleteImage RPC
  - 调用docker.ImageRemove()
  - 支持force和no_prune参数
  - 返回deleted和untagged列表
  - **文件**：`agent/internal/docker/image_service.go`
  - **依赖**：T010、T011

- [X] **T021** 实现PullImage流式RPC
  - 调用docker.ImagePull()
  - 解析拉取进度JSON
  - 实现流式进度推送
  - **文件**：`agent/internal/docker/image_service.go`
  - **依赖**：T010、T011

- [X] **T022** 实现TagImage RPC
  - 调用docker.ImageTag()
  - **文件**：`agent/internal/docker/image_service.go`
  - **依赖**：T010、T011

- [X] **T023** 注册DockerService到gRPC Server
  - 在Agent启动时注册服务
  - 添加连接日志
  - **文件**：`agent/cmd/agent/main.go`
  - **依赖**：T012-T022

---

## Phase C: 服务层（10个任务）

### 核心服务

- [X] **T024** 实现AgentForwarder服务
  - 实现连接池管理（复用Agent gRPC连接）
  - 实现所有转发方法（ListContainers、StartContainer等）
  - 添加超时控制（30秒）
  - 添加错误转换（gRPC状态码 → HTTP状态码）
  - **文件**：`internal/services/docker/agent_forwarder.go`
  - **依赖**：T010

- [X] **T025** 实现DockerInstanceService
  - 实现CRUD操作（Create、Update、Delete、GetByID等）
  - 实现自动发现逻辑（Agent上报Docker信息时创建实例）
  - 实现归档逻辑（Agent删除时标记archived）
  - **文件**：`internal/services/docker/instance_service.go`
  - **依赖**：T008、T024

- [X] **T026** 实现DockerHealthService
  - 实现CheckInstance方法（调用Agent GetDockerInfo）
  - 实现CheckAllInstances方法（10并发，研究任务7）
  - 更新健康状态和统计数据
  - **文件**：`internal/services/docker/health_service.go`
  - **依赖**：T024、T025

- [X] **T027** 实现ContainerService
  - 实现容器操作方法（Start、Stop、Restart、Delete）
  - 添加操作前检查（实例在线、容器存在）
  - 创建审计日志（操作前后快照）
  - **文件**：`internal/services/docker/container_service.go`
  - **依赖**：T024、T025

- [X] **T028** 实现ImageService
  - 实现镜像操作方法（Delete、Pull、Tag）
  - 添加Registry认证支持（Phase 1：读取config.json）
  - 创建审计日志
  - **文件**：`internal/services/docker/image_service.go`
  - **依赖**：T024、T025

- [X] **T029** 实现DockerCacheService
  - 实现缓存管理（5分钟TTL）
  - 缓存键格式：`instanceID:containers`、`instanceID:images`
  - 实现缓存失效逻辑（健康状态变化时清空）
  - **文件**：`internal/services/docker/cache_service.go`
  - **依赖**：T024

### 后台任务

- [X] **T030** 注册健康检查调度任务
  - 在Scheduler中注册60秒间隔任务
  - 调用DockerHealthService.CheckAllInstances
  - 添加错误日志
  - **文件**：`internal/services/scheduler/scheduler.go`
  - **依赖**：T026

- [X] **T031** 注册审计日志清理任务
  - 在Scheduler中注册每天2AM任务
  - 删除90天前的Docker操作日志
  - 添加清理统计日志
  - **文件**：`internal/services/scheduler/scheduler.go`
  - **依赖**：T005

### 集成

- [X] **T032** 集成DockerInstance到Agent模块
  - Agent连接时自动创建/更新Docker实例
  - Agent断开时标记实例离线
  - Agent删除时标记实例archived
  - **文件**：`internal/services/agent/agent_service.go`（现有文件）
  - **依赖**：T025

- [X] **T033** 数据库迁移
  - 添加DockerInstance表到AutoMigrate
  - 运行迁移并验证
  - **文件**：`internal/db/migrate.go`
  - **依赖**：T004

---

## Phase D: API层（12个任务，2个并行）

### REST API处理器

- [X] **T034** [P] 实现DockerInstanceHandler
  - 实现GetInstances（列表、分页、过滤）
  - 实现GetInstance（详情）
  - 实现CreateInstance（手动注册）
  - 实现UpdateInstance
  - 实现DeleteInstance
  - 实现TestConnection
  - **文件**：`internal/api/handlers/docker/instance_handler.go`
  - **依赖**：T025

- [X] **T035** [P] 实现ContainerHandler
  - 实现GetContainers（列表、分页、filter）
  - 实现GetContainer（详情）
  - 实现StartContainer
  - 实现StopContainer
  - 实现RestartContainer
  - 实现PauseContainer
  - 实现UnpauseContainer
  - 实现DeleteContainer
  - **文件**：`internal/api/handlers/docker/container_handler.go`
  - **依赖**：T027

- [X] **T036** 实现ContainerStatsHandler
  - 实现GetContainerStats（单次查询）
  - 实现GetContainerStats流式（SSE）
  - **文件**：`internal/api/handlers/docker/container_stats_handler.go`
  - **依赖**：T027

- [X] **T037** 实现ContainerLogsHandler
  - 实现GetContainerLogs（历史日志）
  - 实现GetContainerLogs流式（SSE）
  - **文件**：`internal/api/handlers/docker/container_logs_handler.go`
  - **依赖**：T027

- [X] **T038** 实现ImageHandler
  - 实现GetImages（列表、filter）
  - 实现GetImage（详情）
  - 实现DeleteImage
  - 实现PullImage（流式进度）
  - 实现TagImage
  - **文件**：`internal/api/handlers/docker/image_handler.go`
  - **依赖**：T028

- [X] **T039** 实现DockerAuditLogHandler
  - 实现GetDockerAuditLogs（查询、分页、过滤）
  - 支持按用户、操作类型、时间范围过滤
  - **文件**：`internal/api/handlers/docker/audit_handler.go`
  - **依赖**：T005

### WebSocket终端

- [X] **T040** 实现终端会话创建端点
  - POST /api/v1/docker/instances/:id/containers/:cid/terminal
  - 创建会话记录（session_id、TTL 30分钟）
  - 返回WebSocket URL
  - **文件**：`internal/api/handlers/docker/terminal_handler.go`

- [X] **T041** 实现WebSocket终端处理器
  - WS /api/v1/docker/terminal/:session_id
  - 验证JWT token和session_id
  - 通过Agent gRPC调用ExecContainer
  - 双向转发：WebSocket ↔ gRPC
  - 处理input、resize、ping消息
  - 发送output、error、exit、pong消息
  - **文件**：`internal/api/handlers/docker/terminal_handler.go`
  - **依赖**：T040、复用 `pkg/kube/terminal.go` 模式

### 路由注册

- [X] **T042** 注册Docker API路由
  - 注册所有REST端点（/api/v1/docker/*）
  - 注册WebSocket端点（/api/v1/docker/terminal/:session_id）
  - 添加RBAC中间件（Viewer/Operator/Admin权限）
  - 添加审计日志中间件
  - **文件**：`internal/api/routes.go`
  - **依赖**：T034-T041

### Swagger文档

- [X] **T043** 添加Swagger注解
  - 为所有Docker API端点添加swag注释
  - 运行 `./scripts/generate-swagger.sh`
  - 验证 `http://localhost:12306/swagger/index.html`
  - **文件**：`internal/api/handlers/docker/*.go`
  - **依赖**：T034-T041

- [X] **T044** 生成API文档
  - 更新README.md中的API文档链接
  - 添加使用示例
  - **文件**：`README.md`、`docs/api/docker.md`
  - **依赖**：T043

---

## Phase E: 前端层（12个任务，10个并行）

### 页面组件

- [X] **T045** [P] 创建Docker实例列表页
  - 显示实例列表（名称、健康状态、容器数、镜像数）
  - 支持搜索和过滤
  - 添加创建实例按钮
  - 健康状态指示器（绿色/红色/灰色）
  - **文件**：`ui/src/pages/docker/instance-list.tsx`

- [X] **T046** [P] 创建Docker实例详情页
  - 显示实例元数据
  - 五个Tab：概览、容器、镜像、网络、操作历史
  - 测试连接按钮
  - **文件**：`ui/src/pages/docker/instance-detail.tsx`

- [X] **T047** [P] 创建Docker实例表单页
  - 创建/编辑表单（名称、Agent ID、主机地址、端口、描述）
  - TLS配置支持（CA证书、客户端证书、客户端密钥）
  - 表单验证（Zod + React Hook Form）
  - **文件**：`ui/src/pages/docker/instance-form.tsx`

- [X] **T048** [P] 创建容器列表组件
  - 显示容器列表（ID、名称、镜像、状态、端口）
  - 分页组件（20条/页）
  - 搜索框（按名称过滤）
  - 状态过滤下拉框
  - 操作按钮（启动、停止、重启、暂停、删除、日志）
  - **文件**：`ui/src/components/docker/container-list.tsx`

- [X] **T049** [P] 创建容器日志查看器
  - SSE实时日志流
  - 历史日志查询（TanStack Query）
  - 控制选项（tail行数、时间戳、stdout/stderr过滤、自动滚动）
  - 自动滚动到底部
  - **文件**：`ui/src/components/docker/container-logs-viewer.tsx`

- [X] **T050** [P] 创建容器终端组件
  - xterm.js集成（终端UI）
  - WebSocket连接管理
  - 处理input、resize、ping消息
  - 处理output、error、exit、pong消息
  - 复用 `ui/src/components/terminal.tsx` 模式
  - **文件**：`ui/src/components/docker/container-terminal.tsx`

- [X] **T051** [P] 创建镜像列表组件
  - 显示镜像列表（ID、标签、大小、创建时间）
  - 搜索和过滤（按镜像名称）
  - 操作按钮（删除、标签、拉取）
  - 拉取镜像对话框集成
  - **文件**：`ui/src/components/docker/image-list.tsx`

- [X] **T052** [P] 创建镜像拉取对话框
  - 输入镜像名称和可选Registry认证
  - SSE流式拉取进度
  - 进度条动画和日志展示
  - 拉取状态管理（idle/pulling/success/error）
  - **文件**：`ui/src/components/docker/image-pull-dialog.tsx`

- [X] **T053** [P] 创建Docker审计日志页
  - 扩展审计服务（SUBSYSTEMS + ACTIONS）
  - 添加Docker资源类型（docker_container、docker_image、docker_instance）
  - 添加Docker操作类型（container_start、image_pull等）
  - 颜色编码增强
  - **文件**：`ui/src/services/audit-service.ts`、`ui/src/pages/audit-page.tsx`（扩展）

- [X] **T054** [P] 创建Docker API客户端
  - 40+ TypeScript接口定义
  - 30+ React Query hooks
  - 实例、容器、镜像、网络、审计日志API
  - 错误处理和缓存策略
  - **文件**：`ui/src/services/docker-api.ts`

### 路由和导航

- [X] **T055** 注册前端路由
  - 添加Docker子系统路由（/docker/*）
  - 注册所有页面组件（实例列表、详情、表单）
  - Docker Layout包装器
  - **文件**：`ui/src/routes.tsx`
  - **依赖**：T045-T053

- [X] **T056** 添加导航菜单
  - Docker Layout侧边栏菜单
  - 图标和子菜单（概览、实例、容器、镜像、网络）
  - **文件**：`ui/src/layouts/docker-layout.tsx`
  - **依赖**：T055

---

## Phase F: 测试层（3个任务，3个并行）

### 契约测试

- [X] **T057** [P] 编写gRPC契约测试
  - 测试agent_grpc.md中的所有RPC方法
  - 使用Mock Agent gRPC Server
  - 验证请求/响应格式
  - **文件**：`tests/contract/docker/agent_grpc_test.go`
  - **依赖**：T010
  - **必须在Agent实现前编写并失败**

- [X] **T058** [P] 编写REST API契约测试
  - 测试api_rest.md中的所有端点
  - 使用httptest
  - 验证请求/响应格式、错误码
  - **文件**：`tests/contract/docker/api_rest_test.go`
  - **依赖**：T042
  - **必须在API实现前编写并失败**

- [X] **T059** [P] 编写WebSocket契约测试
  - 测试websocket.md中的消息协议
  - 测试终端会话创建
  - 测试WebSocket消息（input、resize、ping、output、exit）
  - **文件**：`tests/contract/docker/websocket_test.go`
  - **依赖**：T040、T041
  - **必须在WebSocket实现前编写并失败**

### 集成测试

- [X] **T060** 编写Docker功能集成测试
  - 使用testcontainers启动真实Docker环境
  - 测试完整流程：
    - 创建实例 → 健康检查 → 列表容器 → 启动容器 → 查看日志 → 进入终端 → 拉取镜像 → 删除容器
  - 验证审计日志记录
  - **文件**：`tests/integration/docker/docker_integration_test.go`
  - **依赖**：T024-T041

---

## 依赖关系图

```
Phase A (数据层)
  T001-T008
    ↓
Phase B (Agent层)
  T009-T023
    ↓
Phase C (服务层)
  T024-T033
    ↓
Phase D (API层) + Phase E (前端层)
  T034-T056 (可部分并行)
    ↓
Phase F (测试层)
  T057-T060 (契约测试在实现前，集成测试在实现后)
```

**关键依赖**：
- T010（生成gRPC代码）阻塞所有Agent实现（T011-T023）
- T024（AgentForwarder）阻塞所有服务层（T025-T029）
- T042（路由注册）阻塞Swagger文档（T043-T044）
- T054（API客户端）阻塞所有前端页面（T045-T053）

---

## 并行执行示例

### 示例1：并行创建数据模型（Phase A）

```bash
# 同时启动4个任务
Task T004: "创建DockerInstance模型 in internal/models/docker_instance.go"
Task T005: "扩展AuditLog模型 in internal/models/audit_log.go"
Task T006: "创建Container和Image模型 in internal/models/docker_container.go"
Task T007: "定义DockerInstanceRepository接口 in internal/repository/interfaces.go"
```

### 示例2：并行创建前端页面（Phase E）

```bash
# 同时启动10个任务
Task T045: "创建Docker实例列表页 in ui/src/pages/docker/instance-list-page.tsx"
Task T046: "创建Docker实例详情页 in ui/src/pages/docker/instance-detail-page.tsx"
Task T047: "创建Docker实例表单页 in ui/src/pages/docker/instance-form-page.tsx"
Task T048: "创建容器列表组件 in ui/src/components/docker/container-list.tsx"
Task T049: "创建容器日志查看器 in ui/src/components/docker/container-logs-viewer.tsx"
Task T050: "创建容器终端组件 in ui/src/components/docker/container-terminal.tsx"
Task T051: "创建镜像列表组件 in ui/src/components/docker/image-list.tsx"
Task T052: "创建镜像拉取对话框 in ui/src/components/docker/image-pull-dialog.tsx"
Task T053: "创建Docker审计日志页 in ui/src/pages/system/audit-logs-page.tsx"
Task T054: "创建Docker API客户端 in ui/src/services/docker-api.ts"
```

### 示例3：并行编写契约测试（Phase F）

```bash
# 同时启动3个任务
Task T057: "编写gRPC契约测试 in tests/contract/docker/agent_grpc_test.go"
Task T058: "编写REST API契约测试 in tests/contract/docker/api_rest_test.go"
Task T059: "编写WebSocket契约测试 in tests/contract/docker/websocket_test.go"
```

---

## 验证清单

**执行前验证**：
- [x] 所有契约文件都有对应的契约测试任务（T057-T059）
- [x] 所有实体都有模型创建任务（T004-T006）
- [x] 所有端点都有实现任务（T034-T041）
- [x] 契约测试在实现之前（T057-T059 标记为"必须失败"）
- [x] 并行任务真正独立（不同文件、无依赖）
- [x] 每个任务指定确切的文件路径

**Phase A完成标准**：
- [X] 数据库迁移成功，docker_instances表创建
- [X] 所有GORM模型通过编译
- [X] Repository接口定义完整

**Phase B完成标准**：
- [X] gRPC代码生成成功
- [X] Agent可以连接Docker守护进程
- [X] 所有15个RPC方法实现并通过单元测试

**Phase C完成标准**：
- [X] AgentForwarder可以调用Agent gRPC
- [X] 健康检查任务正常运行（60秒间隔）
- [X] 审计日志正确记录所有操作

**Phase D完成标准**：
- [X] 所有REST端点返回正确的JSON格式
- [X] WebSocket终端可以连接并执行命令
- [X] Swagger文档生成成功

**Phase E完成标准**：
- [X] 前端可以显示Docker实例列表
- [X] 容器操作（启动、停止、删除）正常工作
- [X] 终端和日志查看器功能正常

**Phase F完成标准**：
- [X] 所有契约测试通过
- [X] 集成测试通过（使用真实Docker环境）
- [X] 测试覆盖率 >80%

---

## 注意事项

### TDD原则强制执行

⚠️ **重要**：契约测试（T057-T059）必须在相应实现之前编写并失败。

**正确流程**：
1. 编写契约测试 → 运行测试 → 确认失败（❌）
2. 实现功能代码 → 运行测试 → 确认通过（✅）
3. 提交代码

**错误流程**：
1. ❌ 同时编写测试和实现
2. ❌ 先实现后补测试
3. ❌ 测试通过时没有先验证失败

### 任务提交策略

- 每完成一个任务，提交一次代码
- 提交信息格式：`[T001] 创建项目目录结构`
- 关键里程碑（Phase完成）创建Git tag

### 文件路径约定

- 后端：`internal/`、`pkg/`、`agent/`
- 前端：`ui/src/`
- 测试：`tests/contract/`、`tests/integration/`
- 协议：`pkg/grpc/proto/docker/`

### 避免的错误

- ❌ 修改同一文件的两个 `[P]` 任务并行执行
- ❌ 模糊的任务描述（如"完成前端"）
- ❌ 跳过契约测试失败验证
- ❌ 任务之间有隐性依赖未标记

---

**任务清单生成时间**：2025-10-22
**预计执行时间**：2-3周（单人），1周（团队并行）
**下一步**：执行 Phase A 任务（T001-T008）

---

*基于 plan.md v1.0.0、data-model.md v1.0.0、contracts/ v1.0.0*
*任务清单版本：v1.0.0*
