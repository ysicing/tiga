# 实施计划：K8s子系统完整实现（从Kite迁移）

**分支**：`005-k8s-kite-k8s` | **日期**：2025-10-17 | **规格**：[spec.md](./spec.md)
**输入**：来自 `.claude/specs/005-k8s-kite-k8s/spec.md` 的功能规格

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

将 Kite 项目中经过验证的 K8s 管理功能完整迁移到 Tiga，打造功能完善的 K8s 子系统。核心功能包括：

**主要需求**：
- 多集群统一管理（集群列表、切换、独立上下文、数据库表自增ID标识）
- 高级 CRD 支持（OpenKruise、Tailscale、Traefik、K3s Upgrade Controller）
- Prometheus 智能监控（异步自动发现、一次性尝试、手动重新检测）
- 资源增强（关系可视化、缓存机制、通用 CRUD）
- 终端增强（节点终端、Pod 终端、WebSocket）
- 全局搜索（跨命名空间、模糊匹配、1秒响应）
- 集群上下文和权限（只读模式、RBAC、审计日志）

**技术方法**：
- 通用 CRD 处理器模式（unstructured.Unstructured、CRD 存在性检查）
- 异步后台服务（Prometheus 发现、30秒超时、无重试机制）
- 集群级别缓存（K8s Client 实例、工作负载资源）
- 依赖 Kubernetes API Server 默认行为（ResourceVersion 冲突检测）
- 无硬性集群数量限制（仅受数据库和服务器资源约束）

## 技术上下文

**语言/版本**：Go 1.24+、TypeScript 5.x（前端）
**主要依赖**：
- 后端：Gin v1.10.0、GORM v1.25.0、client-go v0.31.4、OpenKruise SDK v1.8.0
- 前端：React 19、TailwindCSS、Radix UI、TanStack Query
**存储**：
- 元数据：GORM（SQLite/PostgreSQL/MySQL）
- 缓存：内存缓存（K8s Client 实例、工作负载资源，5 分钟有效期）
**测试**：
- 后端：Go testing、testcontainers-go（集成测试）、testify（断言库）
- 前端：Vitest、Testing Library
**目标平台**：Linux 服务器（amd64/arm64）、浏览器（Chrome 90+、Firefox 88+、Safari 14+、Edge 90+）
**项目类型**：Web 应用（前端 + 后端）
**性能目标**：
- API 响应：资源列表<500ms、全局搜索<1s、Prometheus 查询<2s
- 缓存命中率：>70%
- WebSocket 终端延迟：<100ms
**约束**：
- Kubernetes 1.24+ 兼容
- 支持 OpenKruise 1.0+、Traefik 2.10+、Tailscale Operator 1.52+、K3s Upgrade Controller v0.13+（可选依赖）
- 系统可用性：99.9%（排除K8s集群故障）
- API 错误率：<0.1%
**规模/范围**：
- 无硬性集群数量限制（建议：小规模<10、中等10-50、大规模50+）
- 支持 1000+ 资源的集群（全局搜索 1 秒响应）
- 25 个工作日实施（5 个阶段）
- 60 个功能需求（FR-0到FR-6）

## 章程检查
*门禁：必须在阶段 0 研究之前通过。在阶段 1 设计后重新检查。*

### 原则 1：安全优先设计 ✅

**需求符合性**：
- FR-4.5：节点终端访问限制为管理员权限
- FR-6.1-6.2：只读模式支持，阻止所有修改请求
- FR-6.5-6.6：审计日志记录所有资源修改和节点终端访问
- FR-6.7：集群切换时验证用户对目标集群的访问权限
- Secret 数据默认隐藏（需要点击"显示"）

**实施计划**：
- 使用 Tiga 现有的 JWT + RBAC 中间件
- 所有 K8s 资源操作通过 client-go 的 RBAC 集成进行权限验证
- 审计日志包含集群名称、用户、时间戳、操作详情

### 原则 2：生产就绪性 ✅

**需求符合性**：
- 性能需求明确（资源列表<500ms、搜索<1s、WebSocket延迟<100ms）
- 可靠性需求：99.9% 可用性、<0.1% API 错误率
- 错误处理：CRD 不存在时返回清晰错误、连接失败时显示友好提示
- 边缘情况覆盖：并发冲突、缓存失效、超时、循环引用等 20+ 场景
- 测试策略：契约测试、集成测试、单元测试

**实施计划**：
- 使用 testcontainers-go 进行真实 K8s 集群集成测试
- 所有 API Handler 包含完整的错误处理和日志记录
- 缓存机制（5 分钟有效期）提升性能并支持手动刷新

### 原则 3：卓越用户体验 ✅

**需求符合性**：
- US-6：集群列表页面 → 点击集群 → 集群视图，流程清晰直观
- 界面顶部显示"当前集群"名称，避免操作错误
- 自动隐藏未安装的 CRD 菜单，避免无效操作
- Prometheus 异步发现（不阻塞用户操作）+ 状态提示（检测中/已发现/未发现）
- 资源关系可视化（US-4）、全局搜索（1秒响应）、节点终端（WebSocket 低延迟）

**实施计划**：
- 前端使用 Radix UI + TailwindCSS 保持一致的 UI 模式
- TanStack Query 实现实时数据更新和乐观 UI 更新
- 所有操作提供清晰的成功/错误反馈

### 原则 4：默认可观测性 ✅

**需求符合性**：
- FR-2：Prometheus 智能监控（异步自动发现、零配置集成）
- FR-0.7：集群列表显示健康状态（健康/警告/错误/不可用）
- FR-6.5-6.6：审计日志记录所有资源修改和节点终端访问
- 监控页面显示 CPU/内存使用历史图表

**实施计划**：
- 集成 Tiga 现有的 Prometheus 客户端（`pkg/prometheus/`）
- 集群健康检查机制（Phase 0）
- 审计日志系统（利用现有 `models.AuditLog`）

### 原则 5：开源承诺 ✅

**需求符合性**：
- 参考 Kite 开源项目（https://github.com/ysicing/kite）的设计模式
- 功能规格文档完整（用户故事、验收场景、FAQ）
- 架构决策明确（通用 CRD 处理器、异步发现、缓存策略）
- 实施计划包含详细的阶段划分和工作量估算

**实施计划**：
- 遵循 Tiga 项目的 Apache License 2.0 许可证
- 生成 API 文档（Swagger）
- 提供 quickstart.md 指导用户快速上手

### 门禁状态

✅ **所有原则符合** - 无违规项，可进入阶段 0 研究

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
# Web 应用程序结构（后端 Go + 前端 React）

# 后端（Go）
internal/
├── models/                    # 数据模型（扩展 Cluster 实体）
│   ├── cluster.go            # 扩展：健康状态、统计信息
│   └── audit_log.go          # 审计日志（已有）
├── repository/                # 仓储层
│   ├── cluster_repository.go # 集群 CRUD（已有）
│   └── resource_history_repository.go # 资源历史（已有）
├── services/                  # 业务逻辑层
│   ├── k8s_service.go        # K8s 服务（已有，需扩展）
│   ├── prometheus/           # Prometheus 服务（新增）
│   │   ├── discovery.go      # 异步自动发现
│   │   └── client.go         # 集群级别配置
│   └── k8s/                  # K8s 子服务（新增）
│       ├── cluster.go        # 集群管理
│       ├── cache.go          # 缓存服务
│       ├── search.go         # 全局搜索
│       └── relations.go      # 资源关系
└── api/                       # API 层
    └── handlers/
        ├── cluster/          # 集群管理 API（新增）
        └── k8s/              # K8s 资源 API（扩展）

pkg/                           # 可复用包
├── handlers/resources/        # K8s 资源处理器（扩展）
│   ├── kruise/               # OpenKruise CRD 处理器（新增）
│   ├── tailscale/            # Tailscale CRD 处理器（新增）
│   ├── traefik/              # Traefik CRD 处理器（新增）
│   └── k3s/                  # K3s Upgrade Controller 处理器（新增）
├── kube/                      # K8s 客户端工具（扩展）
│   ├── client.go             # 多集群 Client 缓存（扩展）
│   ├── terminal.go           # 节点终端（已有）
│   └── crd.go                # CRD 通用处理器（新增）
└── middleware/                # 中间件（扩展）
    ├── cluster_context.go    # 集群上下文中间件（新增）
    └── readonly.go           # 只读模式中间件（新增）

# 前端（React + TypeScript）
ui/src/
├── pages/k8s/                 # K8s 页面（扩展）
│   ├── clusters/             # 集群列表、详情（新增）
│   ├── resources/            # 资源管理（扩展）
│   │   ├── kruise/           # OpenKruise 资源页面（新增）
│   │   ├── tailscale/        # Tailscale 资源页面（新增）
│   │   └── traefik/          # Traefik 资源页面（新增）
│   ├── monitoring/           # Prometheus 监控（扩展）
│   └── search/               # 全局搜索（新增）
├── components/k8s/            # K8s 组件
│   ├── ClusterSelector.tsx   # 集群切换器（新增）
│   ├── ResourceRelations.tsx # 资源关系图（新增）
│   └── TerminalPanel.tsx     # 终端面板（扩展）
└── contexts/                  # 上下文
    └── ClusterContext.tsx    # 集群上下文（新增）

# 测试
tests/
├── contract/k8s/             # K8s API 契约测试（新增）
├── integration/k8s/          # K8s 集成测试（新增）
│   ├── cluster_test.go
│   ├── prometheus_test.go
│   ├── kruise_test.go
│   └── search_test.go
└── unit/k8s/                 # K8s 单元测试（新增）
    ├── cache_test.go
    └── relations_test.go
```

**结构决策**：选择 Web 应用程序结构（选项 2），因为项目是全栈应用（Go 后端 + React 前端）。后端采用分层架构（models/repository/services/api），前端采用页面组件分离。K8s 子系统作为现有项目的扩展模块，复用 Tiga 现有的基础设施（Wire DI、JWT 认证、数据库、前端框架）。

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
- 从阶段 1 设计文档生成具体任务：
  - `contracts/cluster-api.md` → 7 个契约测试任务 [P]
  - `contracts/crd-api.md` → 13 个契约测试任务 [P]（OpenKruise 3个 + Tailscale 3个 + Traefik 9个）
  - `contracts/search-api.md` → 1 个契约测试任务 [P]
  - `data-model.md` → 2 个实体扩展任务（Cluster 扩展 + 配置扩展）
  - `quickstart.md` → 5 个集成测试任务（对应 5 个验收场景）

**任务分解**（预计 60-70 个任务）：

### Phase 0 任务（3 天）
1. **[P] 扩展 Cluster 模型**：添加健康状态、统计信息字段
2. **[P] 实现集群健康检查服务**：后台 Goroutine，60 秒间隔
3. **[P] 实现集群列表 API Handler**
4. **[P] 实现集群 CRUD API Handler**
5. **[P] 实现集群上下文中间件**：从 Header/Query 读取 cluster_id
6. **契约测试：集群管理 API**（7 个测试用例）

### Phase 1 任务（5 天）
7. **[P] 扩展配置结构**：添加 KubernetesConfig、PrometheusConfig、FeaturesConfig
8. **[P] 实现 Prometheus 发现服务**：异步后台任务、Service 检测、连通性测试
9. **[P] 实现 Prometheus 发现任务管理器**：启动、停止、状态跟踪
10. **[P] 实现 Prometheus API Handler**：手动配置、重新检测
11. **[P] 增强 Prometheus 客户端**：集群级别配置、URL 优先级
12. **契约测试：Prometheus 监控 API**（3 个测试用例）
13. **集成测试：Prometheus 异步发现**（对应验收场景 2）

### Phase 2 任务（7 天）
14. **[P] 实现通用 CRD 处理器框架**：`pkg/kube/crd.go`
15. **[P] 实现 CRD 检测 API**：扫描集群中的 CRD
16. **[P] 实现 OpenKruise CloneSet Handler**：List/Get/Create/Update/Delete/Scale/Restart
17. **[P] 实现 OpenKruise Advanced DaemonSet Handler**
18. **[P] 实现 OpenKruise Advanced StatefulSet Handler**
19. **[P] 实现 Tailscale Connector Handler**（集群级别）
20. **[P] 实现 Tailscale ProxyClass Handler**（集群级别）
21. **[P] 实现 Tailscale ProxyGroup Handler**（集群级别）
22. **[P] 实现 Traefik IngressRoute Handler**（命名空间级别）
23. **[P] 实现 Traefik Middleware Handler**
24. **[P] 实现 Traefik TLSOption Handler**
25. **[P] 实现其他 6 个 Traefik CRD Handler**（批量实现）
26. **[P] 实现 K3s Plan Handler**（命名空间级别）
27. **[P] 实现菜单动态显示逻辑**：根据 CRD 检测结果显示/隐藏菜单
28. **契约测试：CRD 资源管理 API**（13 个测试用例）
29. **集成测试：CloneSet 扩缩容**（对应验收场景 1）

### Phase 3 任务（6 天）
30. **[P] 实现通用资源处理器**：统一 CRUD 接口
31. **[P] 实现资源关系服务**：静态映射 + 递归查询（限制深度 3）
32. **[P] 实现工作负载缓存服务**：5 分钟有效期、集群级别缓存
33. **[P] 实现全局搜索服务**：并发查询、评分算法、结果限制 50
34. **[P] 实现缓存清理机制**：手动刷新、ResourceVersion 检测
35. **[P] 实现搜索 API Handler**
36. **契约测试：全局搜索 API**（1 个测试用例）
37. **集成测试：全局搜索**（对应验收场景 5）
38. **单元测试：资源关系服务**
39. **单元测试：缓存服务**

### Phase 4 任务（4 天）
40. **[P] 实现节点终端 Handler**：特权 Pod 创建、WebSocket 连接
41. **[P] 实现只读模式中间件**：阻止 POST/PUT/PATCH/DELETE
42. **[P] 增强审计日志**：包含集群名称、集群上下文
43. **[P] 实现 30 分钟超时清理**：自动断开终端会话、清理 Pod
44. **集成测试：节点终端访问**（对应验收场景 3）
45. **集成测试：只读模式**

### 前端任务（并行实施）
46. **[P] 实现集群列表页面**
47. **[P] 实现集群详情页面**
48. **[P] 实现集群切换器组件**
49. **[P] 实现 ClusterContext**
50. **[P] 实现 OpenKruise 资源页面**
51. **[P] 实现 Traefik 资源页面**
52. **[P] 实现全局搜索页面**
53. **[P] 实现资源关系图组件**
54. **[P] 实现 Prometheus 监控页面**（扩展现有）
55. **[P] 实现节点终端面板**（扩展现有）

**排序策略**：
- **TDD 顺序**：契约测试在实现之前创建（先写测试，后写实现）
- **依赖顺序**：
  - Phase 0（多集群基础）→ Phase 1（Prometheus）→ Phase 2（CRD）→ Phase 3（资源增强）→ Phase 4（终端和只读）
  - 模型扩展 → 服务实现 → API Handler → 前端页面
- **并行标记 [P]**：
  - 同一 Phase 内的任务可并行实施（独立文件）
  - 前端任务可与后端任务并行实施

**预计输出**：tasks.md 中的 55 个编号、有序的任务

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
- [x] 阶段 0：研究完成（/spec-kit:plan 命令）✅ 2025-10-17
- [x] 阶段 1：设计完成（/spec-kit:plan 命令）✅ 2025-10-17
- [x] 阶段 2：任务规划完成（/spec-kit:plan 命令 - 仅描述方法）✅ 2025-10-17
- [ ] 阶段 3：任务已生成（/spec-kit:tasks 命令）
- [ ] 阶段 4：实施完成
- [ ] 阶段 5：验证通过

**门禁状态**：
- [x] 初始章程检查：通过 ✅ 所有原则符合
- [x] 设计后章程检查：通过 ✅ 无新增违规
- [x] 所有需要澄清的内容已解决 ✅ 10 个技术决策已研究
- [x] 复杂性偏差已记录 ✅ 无违规项

**生成的文档**：
- ✅ `plan.md` - 实施计划（本文件）
- ✅ `research.md` - 技术研究（10 个决策）
- ✅ `data-model.md` - 数据模型（2 个实体）
- ✅ `contracts/cluster-api.md` - 集群管理 API（7 个端点）
- ✅ `contracts/crd-api.md` - CRD 资源管理 API（13 个 CRD 类型）
- ✅ `contracts/search-api.md` - 全局搜索 API（1 个端点）
- ✅ `quickstart.md` - 快速启动指南（5 个验收场景）

**下一步**：运行 `/spec-kit:tasks` 命令生成 `tasks.md`

---
*基于章程 v1.0.0 - 参见 `.claude/memory/constitution.md`*