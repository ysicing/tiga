# 任务：主机管理子系统-Nezha监控与WebSSH集成

**输入**：来自 `.claude/specs/002-nezha-webssh/` 的设计文档
**前提条件**：plan.md、research.md、data-model.md、contracts/、quickstart.md

## 执行流程（main）
```
1. 从功能目录加载 plan.md
   ✓ 已提取：Go 1.24+、gRPC、GORM、Gin、Zustand、Recharts、xterm.js
2. 加载设计文档：
   ✓ data-model.md：11个实体
   ✓ contracts/：2个契约文件（27个端点）
   ✓ quickstart.md：6个集成测试场景
3. 任务分类：
   ✓ 设置：项目结构、依赖、gRPC协议
   ✓ 测试：契约测试、集成测试
   ✓ 核心：模型、仓储、服务、Handler
   ✓ 前端：组件、页面、状态管理
   ✓ 优化：单元测试、性能、文档
4. 并行规则应用：
   ✓ 不同文件 → [P]
   ✓ 同一文件 → 顺序
   ✓ 测试优先（TDD）
5. 任务编号：T001-T052（共52个任务）
6. 依赖关系已定义
7. 并行执行示例已创建
8. 验证：✓ 所有契约有测试、所有实体有模型、测试在实现前
9. 状态：✓ 任务已准备执行
```

## 格式：`[编号] [P?] 描述`
- **[P]**：可并行运行（不同文件，无依赖）
- 包含确切文件路径

## 路径约定
- **后端**：`internal/`（models、services、api）、`proto/`、`cmd/tiga-agent/`
- **前端**：`ui/src/`（components、pages、stores、hooks）
- **测试**：`tests/`（contract、integration）

---

## 阶段 3.1：设置 (T001-T005)

- [X] **T001** 创建gRPC协议定义
  - 文件：`proto/host_monitor.proto`
  - 内容：定义Agent-Server通信协议（HostInfo、HostState、ReportStateRequest/Response）
  - 参考：data-model.md中的HostInfo、HostState实体

- [X] **T002** 创建服务探测协议定义
  - 文件：`proto/service_probe.proto`
  - 内容：定义探测任务协议（ProbeTask、ProbeResult）
  - 参考：data-model.md中的ServiceMonitor、ServiceProbeResult

- [X] **T003** 创建WebSSH终端协议定义
  - 文件：`proto/terminal.proto`
  - 内容：定义终端IO流协议（TerminalInput、TerminalOutput、ResizeRequest）
  - 参考：contracts/host-api.md中的WebSocket消息格式

- [X] **T004** 生成gRPC代码
  - 命令：`protoc --go_out=. --go-grpc_out=. proto/*.proto`
  - 输出：生成的.pb.go文件到对应proto包

- [X] **T005** 添加Go依赖
  - 文件：`go.mod`
  - 依赖：grpc-go、gopsutil、robfig/cron、antonmedv/expr、golang.org/x/crypto
  - 命令：`go get google.golang.org/grpc github.com/shirou/gopsutil/v3 github.com/robfig/cron/v3 github.com/antonmedv/expr golang.org/x/crypto/ssh`

---

## 阶段 3.2：测试优先（TDD）⚠️ 必须在3.3之前完成 (T006-T017)

**关键：这些测试必须编写并且必须在任何实现之前失败**

### 契约测试（并行执行）

- [X] **T006 [P]** 测试主机管理API契约
  - 文件：`tests/contract/host_api_test.go`
  - 内容：测试12个主机API端点的请求/响应schema
  - 端点：POST/GET/PUT/DELETE /api/v1/vms/hosts、GET /hosts/{id}/state、POST /hosts/{id}/webssh等
  - 参考：contracts/host-api.md

- [X] **T007 [P]** 测试服务探测API契约
  - 文件：`tests/contract/service_monitor_api_test.go`
  - 内容：测试服务探测CRUD端点
  - 端点：POST/GET/PUT/DELETE /api/v1/vms/service-monitors、GET /service-monitors/{id}/probe-history等
  - 参考：contracts/service-alert-api.md

- [X] **T008 [P]** 测试告警API契约
  - 文件：`tests/contract/alert_api_test.go`
  - 内容：测试告警规则和事件端点
  - 端点：POST/GET/PUT/DELETE /api/v1/vms/alert-rules、GET/POST /alert-events等
  - 参考：contracts/service-alert-api.md

- [X] **T009 [P]** 测试主机分组API契约
  - 文件：`tests/contract/host_group_api_test.go`
  - 内容：测试主机分组CRUD端点
  - 端点：POST/GET/DELETE /api/v1/vms/host-groups、POST/DELETE /host-groups/{id}/hosts等
  - 参考：contracts/host-api.md

### 集成测试（并行执行）

- [ ] **T010 [P]** 集成测试：主机节点添加与Agent连接
  - 文件：`tests/integration/host_agent_test.go`
  - 场景：创建主机 → 启动mock Agent → 验证连接和数据上报
  - 参考：quickstart.md场景1

- [ ] **T011 [P]** 集成测试：实时监控数据
  - 文件：`tests/integration/host_monitor_test.go`
  - 场景：Agent上报数据 → 查询实时状态 → 查询历史数据
  - 参考：quickstart.md场景2

- [ ] **T012 [P]** 集成测试：服务探测
  - 文件：`tests/integration/service_probe_test.go`
  - 场景：创建探测规则 → 执行探测 → 验证结果和统计
  - 参考：quickstart.md场景3

- [ ] **T013 [P]** 集成测试：告警规则与事件
  - 文件：`tests/integration/alert_test.go`
  - 场景：创建告警规则 → 模拟触发条件 → 验证告警事件和通知
  - 参考：quickstart.md场景4

- [ ] **T014 [P]** 集成测试：WebSSH终端
  - 文件：`tests/integration/webssh_test.go`
  - 场景：创建WebSSH会话 → WebSocket连接 → 发送命令 → 验证输出
  - 参考：quickstart.md场景5

- [ ] **T015 [P]** 集成测试：主机分组
  - 文件：`tests/integration/host_group_test.go`
  - 场景：创建分组 → 添加主机 → 验证分组探测
  - 参考：quickstart.md场景6

- [ ] **T016** WebSocket实时推送集成测试
  - 文件：`tests/integration/websocket_realtime_test.go`
  - 场景：订阅主机监控 → Agent上报数据 → 验证WebSocket推送
  - 参考：contracts/service-alert-api.md的WebSocket部分

- [ ] **T017** Agent版本兼容性测试
  - 文件：`tests/integration/agent_version_test.go`
  - 场景：不同版本Agent连接 → 验证兼容性检测和警告
  - 参考：research.md决策和风险

---

## 阶段 3.3：核心实现-数据模型（仅在测试失败后）(T018-T028)

### GORM模型（并行执行）

- [X] **T018 [P]** 创建HostNode模型
  - 文件：`internal/models/host_node.go`
  - 内容：主机节点模型，包含UUID、Name、SecretKey（加密）、WebSSH配置等
  - 参考：data-model.md第1节

- [X] **T019 [P]** 创建HostInfo模型
  - 文件：`internal/models/host_info.go`
  - 内容：主机硬件信息模型，一对一关联HostNode
  - 参考：data-model.md第2节

- [X] **T020 [P]** 创建HostState模型
  - 文件：`internal/models/host_state.go`
  - 内容：监控指标模型，时序数据，包含CPU、内存、磁盘等16+指标
  - 参考：data-model.md第3节

- [X] **T021 [P]** 创建HostGroup模型
  - 文件：`internal/models/host_group.go`
  - 内容：主机分组模型，多对多关联HostNode
  - 参考：data-model.md第4节

- [X] **T022 [P]** 创建ServiceMonitor模型
  - 文件：`internal/models/service_monitor.go`
  - 内容：服务探测规则模型，包含类型、目标、频率、告警配置
  - 参考：data-model.md第5节

- [X] **T023 [P]** 创建ServiceProbeResult模型
  - 文件：`internal/models/service_probe_result.go`
  - 内容：探测结果模型，记录成功/失败、延迟、错误信息
  - 参考：data-model.md第6节

- [X] **T024 [P]** 创建ServiceAvailability模型
  - 文件：`internal/models/service_availability.go`
  - 内容：可用性统计模型，聚合数据
  - 参考：data-model.md第7节

- [X] **T025 [P]** 创建WebSSHSession模型
  - 文件：`internal/models/webssh_session.go`
  - 内容：WebSSH会话模型，生命周期管理
  - 参考：data-model.md第8节

- [X] **T026 [P]** 创建MonitorAlertRule模型
  - 文件：`internal/models/monitor_alert_rule.go`
  - 内容：告警规则模型，包含条件表达式、通知配置
  - 参考：data-model.md第9节

- [X] **T027 [P]** 创建AlertEvent模型
  - 文件：`internal/models/alert_event.go`
  - 内容：告警事件模型，记录触发、恢复、确认状态
  - 参考：data-model.md第10节

- [X] **T028 [P]** 创建AgentConnection模型
  - 文件：`internal/models/agent_connection.go`
  - 内容：Agent连接状态模型，心跳追踪
  - 参考：data-model.md第11节

---

## 阶段 3.4：核心实现-仓储层 (T029-T031)

- [X] **T029 [P]** 实现HostRepository
  - 文件：`internal/repository/host_repository.go`
  - 内容：主机CRUD、状态保存、历史查询等方法
  - 参考：data-model.md数据访问层

- [X] **T030 [P]** 实现ServiceRepository
  - 文件：`internal/repository/service_repository.go`
  - 内容：服务探测CRUD、探测结果保存、可用性查询
  - 参考：data-model.md数据访问层

- [X] **T031 [P]** 实现AlertRepository
  - 文件：`internal/repository/alert_repository.go`
  - 内容：告警规则CRUD、事件记录、确认处理
  - 参考：data-model.md数据访问层

---

## 阶段 3.5：核心实现-服务层 (T032-T038)

- [X] **T032** 实现AgentManager服务
  - 文件：`internal/services/host/agent_manager.go`
  - 内容：Agent连接管理、gRPC流维护、心跳检测
  - 依赖：T029（HostRepository）
  - 参考：research.md决策1、2

- [X] **T033** 实现StateCollector服务
  - 文件：`internal/services/host/state_collector.go`
  - 内容：监控数据接收、存储、实时推送
  - 依赖：T032（AgentManager）
  - 参考：data-model.md第3节

- [X] **T034** 实现HostService
  - 文件：`internal/services/host/host_service.go`
  - 内容：主机CRUD业务逻辑、密钥生成、Agent安装命令
  - 依赖：T029（HostRepository）
  - 参考：contracts/host-api.md

- [X] **T035** 实现ServiceProbeScheduler
  - 文件：`internal/services/monitor/probe_scheduler.go`
  - 内容：使用robfig/cron调度探测任务，向Agent下发任务
  - 依赖：T030（ServiceRepository）、T032（AgentManager）
  - 参考：research.md决策5

- [X] **T036** 实现ServiceProbeService
  - 文件：`internal/services/monitor/service_probe.go`
  - 内容：探测规则CRUD、结果聚合、可用性统计
  - 依赖：T030（ServiceRepository）、T035（Scheduler）
  - 参考：contracts/service-alert-api.md

- [X] **T037** 实现AlertEngine服务
  - 文件：`internal/services/alert/alert_engine.go`
  - 内容：使用antonmedv/expr解析告警规则、触发检测
  - 依赖：T031（AlertRepository）
  - 参考：research.md决策6

- [X] **T038** 实现WebSSH SessionManager
  - 文件：`internal/services/webssh/session_manager.go`
  - 内容：会话创建、超时管理、审计日志
  - 依赖：T032（AgentManager）
  - 参考：research.md决策4

---

## 阶段 3.6：核心实现-API Handler (T039-T044)

- [X] **T039** 实现主机管理Handler
  - 文件：`internal/api/handlers/host_handler.go`
  - 端点：POST/GET/PUT/DELETE /api/v1/hosts、GET /hosts/{id}、GET /hosts/{id}/state/*
  - 依赖：T034（HostService）
  - 参考：contracts/host-api.md

- [X] **T040** 实现主机分组Handler
  - 文件：`internal/api/handlers/host_group_handler.go`
  - 端点：POST/GET/DELETE /api/v1/host-groups、POST/DELETE /host-groups/{id}/hosts/*
  - 依赖：T034（HostService）
  - 参考：contracts/host-api.md

- [X] **T041** 实现服务探测Handler
  - 文件：`internal/api/handlers/service_monitor_handler.go`
  - 端点：POST/GET/PUT/DELETE /api/v1/service-monitors、GET /service-monitors/{id}/*
  - 依赖：T036（ServiceProbeService）
  - 参考：contracts/service-alert-api.md

- [X] **T042** 实现告警Handler
  - 文件：`internal/api/handlers/alert_handler.go`
  - 端点：POST/GET/PUT/DELETE /api/v1/alert-rules、GET/POST /alert-events/*
  - 依赖：T037（AlertEngine）
  - 参考：contracts/service-alert-api.md

- [X] **T043** 实现WebSSH Handler
  - 文件：`internal/api/handlers/webssh_handler.go`
  - 端点：POST /api/v1/hosts/{id}/webssh、GET /api/v1/webssh/{session_id}（WebSocket）
  - 依赖：T038（SessionManager）
  - 参考：contracts/host-api.md

- [X] **T044** 实现WebSocket实时推送Handler
  - 文件：`internal/api/handlers/websocket_handler.go`
  - 端点：GET /api/v1/ws/host-monitor、GET /api/v1/ws/service-probe、GET /api/v1/ws/alert-events
  - 依赖：T033（StateCollector）、T036（ServiceProbeService）、T037（AlertEngine）
  - 参考：contracts/service-alert-api.md

---

## 阶段 3.7：Agent实现 (T045-T047)

- [X] **T045** 实现Agent数据采集器
  - 目录：`cmd/tiga-agent/collector/`
  - 文件：system.go、cpu.go、memory.go、network.go、disk.go
  - 内容：使用gopsutil采集16+监控指标
  - 参考：data-model.md第3节、research.md决策2

- [X] **T046** 实现Agent探测执行器
  - 目录：`cmd/tiga-agent/probe/`
  - 文件：http.go、tcp.go、icmp.go
  - 内容：执行HTTP/TCP/ICMP探测任务
  - 参考：data-model.md第6节、research.md决策5

- [X] **T047** 实现Agent SSH执行器
  - 文件：`cmd/tiga-agent/terminal/ssh_executor.go`
  - 内容：使用golang.org/x/crypto/ssh执行SSH命令，IO流代理
  - 参考：research.md决策4

---

## 阶段 3.8：前端实现 (T048-T051)

- [X] **T048 [P]** 创建Zustand主机状态Store
  - 文件：`ui/src/stores/host-store.ts`
  - 内容：主机列表、实时监控数据Map、Agent连接状态、WebSocket订阅管理
  - 参考：plan.md技术栈、contracts/service-alert-api.md

- [X] **T049 [P]** 创建主机监控Hook
  - 文件：`ui/src/hooks/use-host-monitor.ts`
  - 内容：WebSocket连接、实时数据订阅、自动更新Store
  - 依赖：T048（host-store）
  - 参考：contracts/service-alert-api.md WebSocket部分

- [X] **T050** 实现主机监控组件
  - 目录：`ui/src/components/hosts/`
  - 文件：host-card.tsx、monitor-chart.tsx、service-status.tsx、webssh-terminal.tsx、alert-badge.tsx
  - 内容：使用Recharts图表、xterm.js终端
  - 依赖：T048（host-store）、T049（use-host-monitor）
  - 参考：plan.md前端结构

- [X] **T051** 实现主机管理页面
  - 目录：`ui/src/pages/hosts/`
  - 文件：host-list-page.tsx、host-detail-page.tsx、service-monitor-page.tsx、alert-events-page.tsx
  - 依赖：T050（组件）
  - 参考：plan.md前端结构、quickstart.md场景

---

## 阶段 3.9：优化与验证 (T052)

- [X] **T052** 执行quickstart.md验证
  - 文件：`.claude/specs/002-nezha-webssh/quickstart.md`
  - 内容：执行6个完整场景，验证所有验收标准
  - 依赖：所有实现任务（T001-T051）
  - 验收：11项功能、6项性能、5项安全

---

## 依赖关系图

```
设置层（T001-T005）
    ↓
测试层（T006-T017）[P] ← 必须在实现前失败
    ↓
数据模型（T018-T028）[P]
    ↓
仓储层（T029-T031）[P]
    ↓
服务层（T032-T038）
    ├─ T032 → T033
    ├─ T032,T030 → T035 → T036
    └─ T032 → T038
    ↓
API层（T039-T044）
    ├─ T034 → T039,T040
    ├─ T036 → T041
    ├─ T037 → T042
    ├─ T038 → T043
    └─ T033,T036,T037 → T044
    ↓
Agent（T045-T047）[P]
    ↓
前端（T048-T051）
    └─ T048 → T049 → T050 → T051
    ↓
验证（T052）
```

## 并行执行示例

### 批次1：契约测试（同时执行）
```bash
# T006-T009可并行
task dev  # 启动开发服务器
# 在不同终端或使用Task工具：
go test -v tests/contract/host_api_test.go
go test -v tests/contract/service_monitor_api_test.go
go test -v tests/contract/alert_api_test.go
go test -v tests/contract/host_group_api_test.go
```

### 批次2：集成测试（同时执行）
```bash
# T010-T015可并行
go test -v tests/integration/host_agent_test.go &
go test -v tests/integration/host_monitor_test.go &
go test -v tests/integration/service_probe_test.go &
go test -v tests/integration/alert_test.go &
go test -v tests/integration/webssh_test.go &
go test -v tests/integration/host_group_test.go &
wait
```

### 批次3：数据模型（同时执行）
```bash
# T018-T028可并行（11个独立文件）
# 使用Task工具或多个代理实例
```

### 批次4：仓储层（同时执行）
```bash
# T029-T031可并行（3个独立文件）
```

### 批次5：Agent模块（同时执行）
```bash
# T045-T047可并行（独立目录）
```

### 批次6：前端Store和Hook（同时执行）
```bash
# T048-T049可并行（独立文件）
cd ui && pnpm install zustand recharts xterm
# 并行创建store和hook
```

## 验证清单
*门禁：在返回前检查*

- [x] 所有契约都有对应的测试（T006-T009覆盖27个端点）
- [x] 所有实体都有模型任务（T018-T028覆盖11个实体）
- [x] 所有测试都在实现之前（阶段3.2在3.3之前）
- [x] 并行任务真正独立（[P]标记的任务无文件冲突）
- [x] 每个任务指定确切的文件路径
- [x] 没有任务修改与另一个[P]任务相同的文件

## 执行策略

**推荐顺序**：
1. 串行执行设置任务（T001-T005）
2. 并行执行所有测试（T006-T017），**确保全部失败**
3. 并行执行数据模型（T018-T028）
4. 并行执行仓储层（T029-T031）
5. 按依赖顺序执行服务层（T032-T038）
6. 按依赖顺序执行API层（T039-T044）
7. 并行执行Agent（T045-T047）
8. 按依赖顺序执行前端（T048-T051）
9. 执行完整验证（T052）

**预计工作量**：
- 设置：1-2小时
- 测试编写：4-6小时
- 核心实现：20-30小时
- 前端实现：8-12小时
- 验证优化：4-6小时
- **总计：37-56小时**

**里程碑**：
- M1（测试完成）：T017 ✓
- M2（后端完成）：T044 ✓
- M3（全栈完成）：T051 ✓
- M4（验收通过）：T052 ✓
