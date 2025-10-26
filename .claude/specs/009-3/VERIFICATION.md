# 统一终端录制系统 - 验证清单

**生成时间**: 2025-10-26
**完成度**: 阶段 3.1 (设置) + 阶段 3.2 部分 (契约测试)

## ✅ 验证步骤

### 1. 代码编译验证

```bash
# 验证配置文件
go build ./internal/config/...
echo $?  # 应返回 0

# 验证迁移脚本
go build ./internal/db/...
echo $?  # 应返回 0

# 验证服务占位符
go build ./internal/services/recording/...
echo $?  # 应返回 0

# 验证契约测试
go build ./tests/contract/...
echo $?  # 应返回 0
```

### 2. Lint 检查

```bash
# 运行 golangci-lint
task lint

# 或直接运行
golangci-lint run ./internal/config/...
golangci-lint run ./internal/db/...
golangci-lint run ./internal/services/recording/...
golangci-lint run ./tests/contract/...
```

### 3. 配置结构验证

```bash
# 查看 RecordingConfig 定义
grep -A 30 "type RecordingConfig struct" internal/config/config.go

# 验证配置字段
grep -A 20 "RecordingConfig" internal/config/config.go | grep -E "(StorageType|BasePath|RetentionDays|CleanupSchedule)"
```

### 4. 迁移脚本验证

```bash
# 查看迁移函数
cat internal/db/migrations.go

# 验证无向后兼容逻辑
grep -i "backward" internal/db/migrations.go  # 应无结果
grep -i "deprecated" internal/db/migrations.go  # 应无结果
grep -i "migrate.*recording" internal/db/migrations.go  # 只有索引创建
```

### 5. 契约测试验证

```bash
# 列出所有契约测试文件
ls -lh tests/contract/*.go

# 统计测试用例数量
grep -r "t.Run(" tests/contract/ | wc -l

# 查看测试辅助工具
cat tests/contract/test_helper.go
```

### 6. 测试运行（跳过未实现功能）

```bash
# 运行契约测试（预期会跳过，因为实现未完成）
go test -v ./tests/contract/... -short

# 示例输出应包含：
# - SKIP: database setup not implemented yet
# - SKIP: router setup not implemented yet
```

### 7. Git 状态检查

```bash
# 查看修改的文件
git status

# 查看详细变更
git diff internal/config/config.go
git diff internal/db/migrations.go
git diff CLAUDE.md

# 查看新增文件
git ls-files --others --exclude-standard
```

## ✅ 已完成检查项

- [x] **配置扩展**: RecordingConfig 结构体包含所有必需字段
- [x] **迁移脚本**: 简化为仅创建索引，无复杂迁移逻辑
- [x] **服务占位符**: 3 个服务文件已创建（storage/cleanup/manager）
- [x] **契约测试**: 9 个 API 端点测试完整覆盖
- [x] **测试工具**: TestHelper 提供通用测试方法
- [x] **文档更新**: CLAUDE.md 反映最新进度和设计
- [x] **Lint 配置**: gofmt 和 goimports 已启用

## ⚠️ 预期的"问题"（正常）

以下情况是预期的，因为实现尚未开始：

1. **测试会跳过**: 契约测试会 Skip，提示"database setup not implemented"
2. **服务为空**: recording/ 目录下的服务文件只有占位符
3. **模型未扩展**: TerminalRecording 模型尚未添加新字段
4. **路由未注册**: API 端点尚未在路由中注册

这些都是 TDD（测试驱动开发）的正常状态 - **测试先行，实现在后**。

## 📊 代码质量指标

### 文件统计
```bash
# 配置代码行数
wc -l internal/config/config.go

# 迁移脚本行数（应显著减少）
wc -l internal/db/migrations.go

# 契约测试总行数
wc -l tests/contract/*.go | tail -1
```

### 测试覆盖

**契约测试覆盖**:
- ✅ 9/9 API 端点有测试
- ✅ 100% OpenAPI 规范覆盖
- ✅ 分页、过滤、排序测试
- ✅ 错误场景测试 (404/400/403)
- ✅ Asciinema v2 格式验证
- ✅ 文件下载和回放测试
- ✅ 异步清理任务测试

## 🔍 关键代码审查点

### 1. 配置完整性

检查 `internal/config/config.go`:
```go
// 应包含以下字段
type RecordingConfig struct {
    StorageType      string  // ✓
    BasePath         string  // ✓
    RetentionDays    int     // ✓
    CleanupSchedule  string  // ✓
    CleanupBatchSize int     // ✓
    MaxRecordingSize int64   // ✓
    MinIO MinIORecordingConfig // ✓
}
```

### 2. 迁移脚本简洁性

检查 `internal/db/migrations.go`:
- ❌ **不应包含**: 数据迁移逻辑、向后兼容代码
- ✅ **应包含**: 索引创建（type, cleanup, storage, user）
- ✅ **支持数据库**: PostgreSQL, MySQL, SQLite

### 3. 测试辅助工具健壮性

检查 `tests/contract/test_helper.go`:
- ✅ `MakeRequest()` - HTTP 请求封装
- ✅ `AssertJSONResponse()` - JSON 响应验证
- ✅ `AssertSuccessResponse()` - 成功响应断言
- ✅ `AssertErrorResponse()` - 错误响应断言
- ✅ `AssertPaginationStructure()` - 分页结构验证

## 📝 验证报告模板

验证完成后，填写以下报告：

```
## 验证报告

**执行人**: _______
**执行时间**: _______

### 编译验证
- [ ] config 包编译通过
- [ ] db 包编译通过
- [ ] recording 服务包编译通过
- [ ] contract 测试包编译通过

### Lint 检查
- [ ] 无 gofmt 错误
- [ ] 无 goimports 错误
- [ ] 无 govet 错误
- [ ] 无 staticcheck 错误

### 代码审查
- [ ] RecordingConfig 字段完整
- [ ] migrations.go 无向后兼容代码
- [ ] 测试辅助工具方法齐全
- [ ] 契约测试覆盖所有端点

### 文档检查
- [ ] CLAUDE.md 已更新
- [ ] tasks.md 标记已完成任务
- [ ] 配置示例正确

### 问题记录
_（无问题或列出发现的问题）_

### 签名
_______
```

## 🚀 下一步准备

在新会话中继续时，执行以下操作：

1. **加载上下文**:
   ```bash
   cat .claude/specs/009-3/tasks.md
   cat CLAUDE.md | grep -A 100 "统一终端录制系统"
   ```

2. **继续集成测试**:
   - 从 T014 开始（Docker 容器终端录制）
   - 使用 testcontainers-go 创建测试环境

3. **或开始核心实现**:
   - 如果集成测试复杂度过高，可先实现 T024-T026（数据模型）
   - 然后实现基础服务层，再完成集成测试

## 📞 需要帮助？

遇到问题时：
1. 检查 `.claude/specs/009-3/` 目录下的设计文档
2. 查看 `contracts/recording-api.yaml` 了解 API 规范
3. 参考 `quickstart.md` 了解测试场景
4. 运行 `task lint` 发现代码问题
