# 实施计划：[功能]

**分支**：`[###-feature-name]` | **日期**：[日期] | **规格**：[链接]
**输入**：来自 `.claude/specs/[###-feature-name]/spec.md` 的功能规格

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
[从功能规格中提取：主要需求 + 研究中的技术方法]

## 技术上下文
**语言/版本**：[例如，Python 3.11、Swift 5.9、Rust 1.75 或需要澄清]
**主要依赖**：[例如，FastAPI、UIKit、LLVM 或需要澄清]
**存储**：[如适用，例如，PostgreSQL、CoreData、文件或不适用]
**测试**：[例如，pytest、XCTest、cargo test 或需要澄清]
**目标平台**：[例如，Linux 服务器、iOS 15+、WASM 或需要澄清]
**项目类型**：[single/web/mobile - 决定源代码结构]
**性能目标**：[特定领域，例如，1000 req/s、10k lines/sec、60 fps 或需要澄清]
**约束**：[特定领域，例如，<200ms p95、<100MB 内存、支持离线或需要澄清]
**规模/范围**：[特定领域，例如，10k 用户、1M LOC、50 个屏幕或需要澄清]

## 章程检查
*门禁：必须在阶段 0 研究之前通过。在阶段 1 设计后重新检查。*

[基于章程文件确定的门禁]

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
<!--
  需要操作：将下面的占位符树替换为此功能的具体布局。
  删除未使用的选项并使用实际路径展开所选结构（例如，apps/admin、packages/something）。
  交付的计划不得包含选项标签。
-->
```
# [如未使用则删除] 选项 1：单一项目（默认）
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [如未使用则删除] 选项 2：Web 应用程序（检测到"前端"+"后端"时）
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [如未使用则删除] 选项 3：移动端 + API（检测到"iOS/Android"时）
api/
└── [与上面的后端相同]

ios/ 或 android/
└── [平台特定结构：功能模块、UI 流程、平台测试]
```

**结构决策**：[记录所选结构并引用上面捕获的实际目录]

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
- 从阶段 1 设计文档（契约、数据模型、快速启动）生成任务
- 每个契约 → 契约测试任务 [P]
- 每个实体 → 模型创建任务 [P]
- 每个用户故事 → 集成测试任务
- 使测试通过的实现任务

**排序策略**：
- TDD 顺序：测试在实现之前
- 依赖顺序：模型在服务之前在 UI 之前
- 为并行执行标记 [P]（独立文件）

**预计输出**：tasks.md 中的 25-30 个编号、有序的任务

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
- [x] 阶段 0：研究完成（/spec-kit:plan 命令） ✅ 2025-10-22
  - 已创建 `research.md`
  - 已解决 12 个技术决策点
- [x] 阶段 1：设计完成（/spec-kit:plan 命令） ✅ 2025-10-22
  - 已创建 `data-model.md`
  - 已创建 `contracts/agent_grpc.md`
  - 已创建 `contracts/api_rest.md`
  - 已创建 `contracts/websocket.md`
  - 已创建 `quickstart.md`
- [x] 阶段 2：任务规划方法已描述（/spec-kit:plan 命令） ✅ 2025-10-22
  - 已在 plan.md 中描述任务生成策略
- [ ] 阶段 3：任务已生成（/spec-kit:tasks 命令）
  - **下一步**：执行 `/spec-kit:tasks` 生成 `tasks.md`
- [ ] 阶段 4：实施完成
- [ ] 阶段 5：验证通过

**门禁状态**：
- [x] 初始章程检查：通过 ✅
- [x] 设计后章程检查：通过 ✅
- [x] 所有需要澄清的内容已解决 ✅
- [x] 复杂性偏差已记录：无偏差 ✅

**生成的文档**：
1. ✅ `spec.md` - 功能规格（已完成，包含 5 个已解决澄清）
2. ✅ `plan.md` - 实施计划（当前文件）
3. ✅ `research.md` - 技术研究（12 个决策点）
4. ✅ `data-model.md` - 数据模型设计（4 个实体）
5. ✅ `contracts/agent_grpc.md` - gRPC 协议契约
6. ✅ `contracts/api_rest.md` - REST API 契约
7. ✅ `contracts/websocket.md` - WebSocket 协议契约
8. ✅ `quickstart.md` - 验收场景指南（11 个场景）
9. ⏳ `tasks.md` - 任务清单（待 /spec-kit:tasks 命令生成）

**预计任务数量**：~60 个任务（20 个可并行）

**下一步操作**：
```bash
# 执行任务生成命令
/spec-kit:tasks
```

---
*基于章程 v1.0.0 - 参见 `.claude/memory/constitution.md`*
*规格版本：007-docker-docker-agent v1.0.0*
*完成时间：2025-10-22*