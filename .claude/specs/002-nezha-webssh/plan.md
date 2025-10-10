# 实施计划：主机管理子系统-Nezha监控与WebSSH集成

**分支**：`002-nezha-webssh` | **日期**：2025-10-07 | **规格**：[spec.md](./spec.md)
**输入**：来自 `.claude/specs/002-nezha-webssh/spec.md` 的功能规格

## 执行流程（/spec-kit:plan 命令范围）
```
1. 从输入路径加载功能规格
   ✓ 已完成：规格文件已加载
2. 填充技术上下文（扫描需要澄清的内容）
   ✓ 已完成：基于Nezha项目参考，所有技术决策已明确
3. 根据章程文档内容填充章程检查部分
   ✓ 已完成：章程检查通过
4. 评估章程检查部分
   ✓ 已完成：无违规项
5. 执行阶段 0 → research.md
   ✓ 已完成：技术研究文档已生成
6. 执行阶段 1 → contracts、data-model.md、quickstart.md、CLAUDE.md
   ✓ 已完成：所有阶段1文档已生成
7. 重新评估章程检查部分
   ✓ 已完成：设计后无新违规
8. 规划阶段 2 → 描述任务生成方法（不要创建 tasks.md）
   ✓ 已完成：任务规划策略已定义
9. 停止 - 准备执行 /spec-kit:tasks 命令
   ✓ 就绪：可执行下一阶段
```

**重要**：/spec-kit:plan 命令在步骤 8 停止。阶段 2-4 由其他命令执行：
- 阶段 2：/spec-kit:tasks 命令创建 tasks.md
- 阶段 3-4：实施执行（手动或通过工具）

## 摘要

基于Nezha项目的成熟架构，实现主机管理子系统，提供：
- **Agent-Server双向通信**：轻量级Agent通过gRPC持久连接上报监控数据
- **实时监控**：CPU、内存、磁盘、网络等16+指标实时采集和展示
- **服务探测**：HTTP/TCP/ICMP分布式探测，Cron调度，可用性统计
- **WebSSH终端**：浏览器内SSH终端，gRPC流+WebSocket代理架构
- **灵活告警**：表达式引擎驱动的告警规则，多通知渠道集成

技术方案：gRPC双向流（Agent-Server通信）、WebSocket（浏览器实时数据）、robfig/cron（探测调度）、antonmedv/expr（告警引擎）、Zustand（前端状态）、xterm.js（终端UI）。

## 技术上下文

**语言/版本**：
- 后端：Go 1.24+（已有）
- Agent：Go 1.24+（独立二进制）
- 前端：React 19、TypeScript 5.x（已有）

**主要依赖**：
- 后端：Gin、GORM、grpc-go、gorilla/websocket、robfig/cron、antonmedv/expr
- Agent：grpc-go、gopsutil、golang.org/x/crypto/ssh
- 前端：Zustand、Recharts、xterm.js

**存储**：
- 关系数据库：SQLite/PostgreSQL/MySQL（已有GORM抽象）
- 时序数据：短期存储在关系数据库，可选Prometheus长期存储

**测试**：
- Go：内置testing包 + testcontainers-go（已有）
- 前端：Vitest + React Testing Library（已有）

**目标平台**：
- Server：Linux服务器（现有部署环境）
- Agent：Linux/Windows/macOS跨平台编译
- 浏览器：现代浏览器（Chrome、Firefox、Safari、Edge）

**项目类型**：Web应用（前端+后端）

**性能目标**：
- Agent资源占用：内存<30MB，CPU<5%（空闲<1%）
- 监控数据延迟：<30秒
- 服务探测并发：>1000个规则
- WebSSH延迟：<200ms（广域网）
- API响应时间：<500ms

**约束**：
- Agent必须轻量级，避免影响生产主机性能
- 支持大规模部署（数百台主机）
- 数据保留策略：7-30天监控数据
- 安全性：TLS加密、RBAC权限、审计日志

**规模/范围**：
- 支持100+ Agent并发连接
- 1000+ 服务探测规则
- 10+ 并发WebSSH会话
- 7天历史数据快速查询

## 章程检查
*门禁：必须在阶段 0 研究之前通过。在阶段 1 设计后重新检查。*

### 原则1：安全优先设计
- ✓ Agent连接需要密钥验证（HostNode.SecretKey加密存储）
- ✓ gRPC连接强制TLS加密
- ✓ WebSSH访问基于RBAC权限控制（hosts:webssh权限）
- ✓ 所有API端点需要JWT认证
- ✓ WebSSH会话操作审计日志（WebSSHSession记录）
- ✓ 敏感数据加密存储（AES-256）

### 原则2：生产就绪性
- ✓ Agent自动重连机制（网络恢复后快速恢复）
- ✓ 错误处理和优雅降级（Agent离线主机标记offline）
- ✓ 数据保留策略（7-30天，自动清理）
- ✓ 版本兼容性检测（Agent版本验证）
- ✓ 性能监控（Agent资源占用<30MB）

### 原则3：卓越用户体验
- ✓ 响应式UI设计（TailwindCSS + Radix UI复用）
- ✓ 实时数据更新（WebSocket推送，无需手动刷新）
- ✓ 直观的监控图表（Recharts可视化）
- ✓ 友好的错误提示（统一响应格式）
- ✓ 国际化支持（i18n集成）

### 原则4：默认可观测性
- ✓ 开箱即用的监控仪表板
- ✓ 16+监控指标自动采集
- ✓ 实时告警和通知
- ✓ 详细的探测历史记录
- ✓ 可用性统计和趋势分析

### 原则5：开源承诺
- ✓ Apache License 2.0许可证（项目已有）
- ✓ 清晰的API文档（OpenAPI规范）
- ✓ 详细的实施计划和设计文档
- ✓ 参考开源项目Nezha的最佳实践

**门禁状态**：✓ 通过，无违规项

## 项目结构

### 文档（此功能）
```
specs/002-nezha-webssh/
├── plan.md              # 此文件（/spec-kit:plan 命令输出）
├── research.md          # 阶段 0 输出（/spec-kit:plan 命令）✓
├── data-model.md        # 阶段 1 输出（/spec-kit:plan 命令）✓
├── quickstart.md        # 阶段 1 输出（/spec-kit:plan 命令）✓
├── contracts/           # 阶段 1 输出（/spec-kit:plan 命令）✓
│   ├── host-api.md
│   └── service-alert-api.md
└── tasks.md             # 阶段 2 输出（/spec-kit:tasks 命令 - 待生成）
```

### 源代码（仓库根目录）

**选定结构**：Web 应用程序（前端+后端）

```
# 后端新增模块
internal/
├── models/
│   ├── host_node.go          # 主机节点模型
│   ├── host_info.go          # 主机信息
│   ├── host_state.go         # 主机状态
│   ├── host_group.go         # 主机分组
│   ├── service_monitor.go    # 服务探测规则
│   ├── service_probe_result.go
│   ├── service_availability.go
│   ├── webssh_session.go     # WebSSH会话
│   ├── monitor_alert_rule.go # 告警规则
│   ├── alert_event.go        # 告警事件
│   └── agent_connection.go   # Agent连接状态
├── repository/
│   ├── host_repository.go    # 主机仓储
│   ├── service_repository.go # 服务仓储
│   └── alert_repository.go   # 告警仓储
├── services/
│   ├── host/                 # 主机服务
│   │   ├── host_service.go
│   │   ├── agent_manager.go  # Agent连接管理
│   │   └── state_collector.go
│   ├── monitor/              # 监控服务
│   │   ├── service_probe.go  # 服务探测
│   │   └── probe_scheduler.go
│   ├── webssh/               # WebSSH服务
│   │   ├── session_manager.go
│   │   └── terminal_proxy.go
│   └── alert/                # 告警服务（扩展现有）
│       ├── monitor_alert.go
│       └── alert_engine.go
├── api/
│   └── handlers/
│       ├── host_handler.go   # 主机API处理器
│       ├── service_monitor_handler.go
│       ├── alert_handler.go
│       └── webssh_handler.go

# gRPC协议定义
proto/
├── host_monitor.proto        # 主机监控协议
├── service_probe.proto       # 服务探测协议
└── terminal.proto            # WebSSH终端协议

# Agent（独立项目或cmd子目录）
cmd/tiga-agent/
├── main.go
├── collector/                # 数据采集
│   ├── system.go
│   ├── cpu.go
│   ├── memory.go
│   └── network.go
├── probe/                    # 服务探测执行
│   ├── http.go
│   ├── tcp.go
│   └── icmp.go
├── terminal/                 # SSH执行器
│   └── ssh_executor.go
└── client/                   # gRPC客户端
    └── agent_client.go

# 前端新增模块
ui/src/
├── pages/
│   └── hosts/                # 主机管理页面
│       ├── host-list-page.tsx
│       ├── host-detail-page.tsx
│       ├── service-monitor-page.tsx
│       └── alert-events-page.tsx
├── components/
│   └── hosts/                # 主机组件
│       ├── host-card.tsx
│       ├── monitor-chart.tsx
│       ├── service-status.tsx
│       ├── webssh-terminal.tsx
│       └── alert-badge.tsx
├── services/
│   ├── host-service.ts       # 主机API客户端
│   ├── monitor-service.ts
│   └── webssh-service.ts
├── stores/
│   └── host-store.ts         # Zustand主机状态
└── hooks/
    ├── use-host-monitor.ts   # 实时监控Hook
    └── use-webssh.ts         # WebSSH Hook

# 测试
tests/
├── integration/
│   ├── agent_test.go         # Agent集成测试
│   ├── service_probe_test.go
│   └── webssh_test.go
└── contract/
    ├── host_api_test.go      # API契约测试
    └── service_api_test.go
```

**结构决策**：
- 采用Web应用结构（frontend/ + backend/），复用现有项目架构
- Agent作为独立cmd子项目，可单独编译部署
- gRPC协议定义在proto/目录，前后端共享
- 前端按功能模块组织（pages/hosts、components/hosts）
- 测试分离为contract（契约测试）和integration（集成测试）

## 阶段 0：概述与研究

**已完成**：✓ research.md已生成

### 研究成果摘要

**决策1**：gRPC双向流式通信
- 理由：实时性、二进制协议高效、双向通信、类型安全
- 替代方案：WebSocket、HTTP长轮询、MQTT

**决策2**：独立二进制Agent + systemd服务
- 理由：轻量级、跨平台、自动重连、资源占用低
- 替代方案：容器化Agent、脚本Agent、Sidecar模式

**决策3**：混合存储（关系数据库 + 可选Prometheus）
- 理由：复用现有GORM、短期数据快速查询、长期数据可选外部存储
- 替代方案：仅Prometheus、仅关系数据库、InfluxDB

**决策4**：gRPC流代理 + WebSocket双层架构（WebSSH）
- 理由：安全性、实时性、会话管理、审计追溯
- 替代方案：直接SSH代理、VNC/RDP、Web-based IDE

**决策5**：Agent端分布式探测 + Server调度
- 理由：分布式探测、网络位置灵活、延迟准确、批量管理
- 替代方案：Server中心化探测、仅Agent自主探测

**决策6**：表达式引擎(antonmedv/expr)
- 理由：灵活规则定义、类型安全、可扩展、性能高
- 替代方案：硬编码规则、Lua/JS脚本、Prometheus AlertManager

**决策7**：WebSocket + Zustand实时数据
- 理由：Server主动推送、轻量级状态管理、选择性订阅
- 替代方案：HTTP轮询、SSE、GraphQL Subscription

**输出**：research.md，所有技术未知项已解决

## 阶段 1：设计与契约

**已完成**：✓ data-model.md、contracts/、quickstart.md、CLAUDE.md

### 1. 数据模型（data-model.md）

**核心实体**（11个）：
1. HostNode：主机节点元数据
2. HostInfo：主机硬件系统信息
3. HostState：实时监控指标（时序）
4. HostGroup：主机分组
5. ServiceMonitor：服务探测规则
6. ServiceProbeResult：探测结果
7. ServiceAvailability：可用性统计
8. WebSSHSession：SSH会话
9. MonitorAlertRule：告警规则
10. AlertEvent：告警事件
11. AgentConnection：Agent连接状态

**关键设计**：
- 元数据与监控数据分离（HostNode vs HostState）
- 时序数据优化（timestamp索引）
- 预聚合统计（ServiceAvailability）
- 完整生命周期记录（WebSSHSession、AlertEvent）

### 2. API契约（contracts/）

**已生成**：
- `host-api.md`：主机管理、监控数据、分组、WebSSH（12个端点）
- `service-alert-api.md`：服务探测、告警规则、事件、WebSocket订阅（15个端点）

**API特点**：
- RESTful设计（/api/v1/hosts、/service-monitors、/alert-rules）
- 统一响应格式（code、message、data）
- WebSocket实时推送（监控数据、探测结果、告警事件）
- 分页查询支持（page、page_size、过滤、排序）

### 3. 契约测试

**测试策略**：
- 每个端点对应一个契约测试
- 验证请求/响应schema
- 测试必须失败（实现前）
- 使用Go testing包 + httptest

**测试文件**（待生成）：
- `tests/contract/host_api_test.go`
- `tests/contract/service_api_test.go`
- `tests/contract/alert_api_test.go`
- `tests/contract/webssh_api_test.go`

### 4. 快速启动测试（quickstart.md）

**测试场景**（6个）：
1. 主机节点添加与Agent连接
2. 实时监控数据
3. 服务探测
4. 告警规则与事件
5. WebSSH终端
6. 主机分组

**验收标准**：
- 功能完整性：11项检查
- 性能指标：6项要求
- 安全性：5项验证

### 5. 代理文件更新（CLAUDE.md）

**已更新**：✓ 新增"主机管理子系统"章节

**输出**：data-model.md、contracts/*、quickstart.md、CLAUDE.md已更新

## 阶段 2：任务规划方法

*本节描述 /spec-kit:tasks 命令将执行的操作 - 不要在 /spec-kit:plan 期间执行*

**任务生成策略**：

1. **从阶段1设计文档提取任务**：
   - data-model.md → 数据模型创建任务（11个实体）
   - contracts/host-api.md → API端点实现任务（12个端点）
   - contracts/service-alert-api.md → API端点实现任务（15个端点）
   - quickstart.md → 集成测试任务（6个场景）

2. **TDD顺序**：
   - 先创建契约测试（测试失败）
   - 再实现数据模型
   - 再实现API Handler
   - 最后通过测试

3. **任务分类**：
   - **[P] 并行任务**：独立文件，可并发执行
     - 数据模型创建（11个实体）
     - 契约测试创建（4个测试文件）
     - 仓储实现（3个仓储）

   - **[S] 顺序任务**：有依赖关系
     - Agent gRPC协议定义 → Agent实现
     - 数据模型 → 仓储 → 服务层 → Handler
     - 前端组件 → 页面 → 集成测试

4. **任务优先级**：
   - P0（核心）：Agent连接、监控数据采集、API基础
   - P1（重要）：服务探测、告警规则、WebSSH
   - P2（增强）：主机分组、高级告警、Prometheus集成

**排序策略**：
- 阶段顺序：数据层 → 服务层 → API层 → 前端 → 集成测试
- 依赖顺序：模型 → 仓储 → 服务 → Handler
- 功能顺序：Agent连接 → 监控 → 探测 → WebSSH → 告警
- 并行标记：同一层级、无依赖的任务标记[P]

**预计任务数量**：约45-50个任务
- 数据模型：11个
- 仓储实现：3个
- 服务层：8个
- API Handler：10个
- 前端组件：6个
- 前端页面：4个
- 集成测试：6个

**预计输出**：tasks.md中的编号、有序任务列表

**重要**：此阶段由 /spec-kit:tasks 命令执行，而不是由 /spec-kit:plan 执行

## 阶段 3+：未来实施

*这些阶段超出了 /spec-kit:plan 命令的范围*

**阶段 3**：任务执行（/spec-kit:tasks 命令创建 tasks.md）
**阶段 4**：实施（按照章程原则执行 tasks.md）
**阶段 5**：验证（运行测试、执行 quickstart.md、性能验证）

## 复杂性跟踪

*仅在章程检查有必须证明合理的违规时填写*

**无违规项**：设计符合所有章程原则

## 进度跟踪

*此清单在执行流程期间更新*

**阶段状态**：
- [x] 阶段 0：研究完成（/spec-kit:plan 命令）✓
- [x] 阶段 1：设计完成（/spec-kit:plan 命令）✓
- [x] 阶段 2：任务规划完成（/spec-kit:plan 命令 - 仅描述方法）✓
- [ ] 阶段 3：任务已生成（/spec-kit:tasks 命令）
- [ ] 阶段 4：实施完成
- [ ] 阶段 5：验证通过

**门禁状态**：
- [x] 初始章程检查：通过 ✓
- [x] 设计后章程检查：通过 ✓
- [x] 所有需要澄清的内容已解决 ✓
- [x] 复杂性偏差已记录：无违规 ✓

**生成文档**：
- [x] research.md ✓
- [x] data-model.md ✓
- [x] contracts/host-api.md ✓
- [x] contracts/service-alert-api.md ✓
- [x] quickstart.md ✓
- [x] CLAUDE.md（已更新）✓
- [ ] tasks.md（待/spec-kit:tasks生成）

---

**计划阶段完成**：✓ 所有阶段0-2已完成，准备执行 `/spec-kit:tasks` 生成任务列表

*基于章程 v1.0.0 - 参见 `.claude/memory/constitution.md`*
