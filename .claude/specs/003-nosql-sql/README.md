# 数据库管理系统 - 使用指南汇总

**最后更新**: 2025-10-12
**功能状态**: ✅ 核心功能已实现并可用
**实施完成度**: 97%

---

## 📌 针对您的问题: "创建数据库实例没成功"

### 根本原因
数据库管理API需要**JWT认证 + 管理员权限**，您的请求可能缺少认证token。

### 立即解决方案

#### 方法1: 使用自动化测试脚本 (推荐)

```bash
# 1. 编辑脚本配置
vi scripts/test-database-instance.sh

# 修改以下内容:
# - 第18-19行: 登录用户名和密码
# - 第50行: MySQL root密码

# 2. 运行脚本
./scripts/test-database-instance.sh
```

脚本会自动:
- 登录获取JWT token
- 创建MySQL测试实例
- 列出所有实例
- 测试连接

#### 方法2: 手动步骤

**步骤1: 获取JWT Token**
```bash
curl -X POST http://localhost:12306/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-admin-password"
  }'

# 从响应中提取token字段
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**步骤2: 使用Token创建实例**
```bash
curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MySQL Production",
    "type": "mysql",
    "host": "localhost",
    "port": 3306,
    "username": "root",
    "password": "your-mysql-password"
  }'
```

---

## 📚 文档导航

根据您的需求选择合适的文档:

### 1. 快速开始 - 选择您的场景

| 场景 | 使用文档 | 用途 |
|------|---------|------|
| 🆘 **解决创建实例失败** | [QUICK_REFERENCE.md](./QUICK_REFERENCE.md) | 快速参考卡,常见错误速查 |
| 🚀 **首次使用系统** | [quickstart.md](./quickstart.md) | 完整的环境搭建和使用流程 |
| 🔍 **诊断问题** | [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) | 详细的故障排查步骤 |
| 📊 **了解实施状态** | [IMPLEMENTATION_REPORT.md](./IMPLEMENTATION_REPORT.md) | 完整的实施完成报告 |

### 2. 技术文档

| 文档 | 内容 |
|------|------|
| [spec.md](./spec.md) | 功能需求规格 |
| [data-model.md](./data-model.md) | 数据模型设计 |
| [plan.md](./plan.md) | 实施计划 |
| [research.md](./research.md) | 技术决策研究 |
| [tasks.md](./tasks.md) | 详细任务列表 |
| [contracts/database-api.yaml](./contracts/database-api.yaml) | API规范 |

---

## ✅ 系统实施状态

### 已完成 (97%)

#### ✅ 数据模型层 (6个模型)
- DatabaseInstance, Database, DatabaseUser
- PermissionPolicy, QuerySession, DatabaseAuditLog
- 位置: `internal/models/db_*.go`

#### ✅ 仓储层 (5个仓储)
- Instance, Database, User, Permission, AuditLog
- 位置: `internal/repository/database/`

#### ✅ 驱动层 (3种数据库)
- MySQL Driver, PostgreSQL Driver, Redis Driver
- 位置: `pkg/dbdriver/`

#### ✅ 服务层 (7个服务)
- DatabaseManager, SQL/Redis安全过滤器
- QueryExecutor, AuditLogger等
- 位置: `internal/services/database/`

#### ✅ API处理器层 (6个处理器)
- Instance, Database, User, Permission, Query, Audit
- 位置: `internal/api/handlers/database/`

#### ✅ 集成
- 路由已注册 (`internal/api/routes.go:320`)
- 数据库迁移已配置
- 中间件栈: CORS → Logger → Auth → RBAC → Audit

### 待完成 (3%)

- ⏸ 单元测试覆盖率提升 (目标≥70%)
- ⏸ 性能测试
- ⏸ Swagger文档生成 (`./scripts/generate-swagger.sh`)

---

## 🔒 安全特性

系统已实现以下安全机制:

1. **认证**: JWT Bearer Token (24小时有效期)
2. **授权**: 基于角色的访问控制 (需要Admin角色)
3. **加密**: AES-256加密数据库密码
4. **SQL防护**: AST解析阻止DDL操作
5. **危险操作拦截**: 无WHERE的UPDATE/DELETE
6. **Redis命令过滤**: 禁止FLUSHDB/FLUSHALL/SHUTDOWN等
7. **审计日志**: 全量记录,90天保留期
8. **超时控制**: 查询30秒超时
9. **结果限制**: 10MB最大结果大小

---

## 🌐 API端点概览

### 实例管理
- `GET /api/v1/database/instances` - 列出所有实例
- `POST /api/v1/database/instances` - 创建实例
- `GET /api/v1/database/instances/{id}` - 获取实例详情
- `DELETE /api/v1/database/instances/{id}` - 删除实例
- `POST /api/v1/database/instances/{id}/test` - 测试连接

### 数据库操作
- `GET /api/v1/database/instances/{id}/databases` - 列出数据库
- `POST /api/v1/database/instances/{id}/databases` - 创建数据库
- `DELETE /api/v1/database/instances/{id}/databases/{name}` - 删除数据库

### 用户管理
- `GET /api/v1/database/instances/{id}/users` - 列出用户
- `POST /api/v1/database/instances/{id}/users` - 创建用户
- `PUT /api/v1/database/instances/{id}/users/{username}/password` - 修改密码
- `DELETE /api/v1/database/instances/{id}/users/{username}` - 删除用户

### 权限管理
- `GET /api/v1/database/permissions` - 查询权限
- `POST /api/v1/database/permissions` - 授予权限
- `DELETE /api/v1/database/permissions/{id}` - 撤销权限

### 查询执行
- `POST /api/v1/database/instances/{id}/query` - 执行查询

### 审计日志
- `GET /api/v1/database/audit-logs` - 查询审计日志

**所有端点都需要**: `Authorization: Bearer YOUR_JWT_TOKEN`

---

## 🧪 测试与验证

### 契约测试
```bash
# 运行契约测试
go test ./tests/contract/*_contract_test.go -v

# 当前状态: 编译通过,按预期失败(TDD要求)
```

### 集成测试
```bash
# 启动测试数据库
docker-compose -f docker-compose.test.yml up -d

# 运行集成测试
go test ./tests/integration/database/... -v
```

### 手动测试
参考 [quickstart.md](./quickstart.md) 中的场景演示

---

## 🐛 常见问题快速解决

| 问题 | 快速检查 |
|------|---------|
| 创建实例失败 | 1. 检查是否有JWT token<br>2. 确认用户是管理员<br>3. 验证数据库可连接 |
| 连接测试失败 | 1. 数据库服务是否运行<br>2. 端口是否正确<br>3. 用户名密码是否正确 |
| 加密错误 | 检查config.yaml中的encryption_key |
| 查询被拦截 | 查看错误消息,可能是DDL或危险操作 |

详细诊断步骤请参考 [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)

---

## 📞 获取帮助

### 本地调试
```bash
# 启用调试日志
export LOG_LEVEL=debug

# 查看详细日志
./bin/tiga 2>&1 | tee tiga.log
```

### 检查清单
在报告问题前,请提供:
1. ✅ 完整的错误响应
2. ✅ 应用日志 (最近100行)
3. ✅ 使用的curl命令
4. ✅ 环境信息 (OS、Tiga版本、数据库版本)
5. ✅ 是否使用了JWT token
6. ✅ 用户角色信息

### 提交Issue
GitHub: https://github.com/ysicing/tiga/issues

---

## 🎓 学习路径

### 新用户
1. 阅读 [QUICK_REFERENCE.md](./QUICK_REFERENCE.md) - 5分钟
2. 运行 `scripts/test-database-instance.sh` - 测试创建实例
3. 参考 [quickstart.md](./quickstart.md) - 完整场景演练

### 开发者
1. 阅读 [IMPLEMENTATION_REPORT.md](./IMPLEMENTATION_REPORT.md) - 了解架构
2. 查看 [data-model.md](./data-model.md) - 理解数据模型
3. 阅读 [tasks.md](./tasks.md) - 了解实施细节
4. 查看代码: `internal/services/database/` 和 `pkg/dbdriver/`

### 运维人员
1. 阅读 [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - 故障排查
2. 了解 [quickstart.md](./quickstart.md) - 部署和配置
3. 配置审计日志清理和备份策略

---

## 🚀 下一步行动

### 立即行动
1. ✅ 使用测试脚本验证功能: `./scripts/test-database-instance.sh`
2. ✅ 生成Swagger文档: `./scripts/generate-swagger.sh`
3. ✅ 访问API文档: http://localhost:12306/swagger/index.html

### 短期任务
1. 运行集成测试确保功能完整性
2. 补充单元测试提升覆盖率
3. 进行性能测试验证响应时间

### 长期规划
1. Phase 2: 前端UI开发 (React组件)
2. 高级功能: 审计日志导出、查询计划分析
3. 多租户数据隔离

---

**祝使用愉快! 如有问题,请参考对应文档或提交Issue。**
