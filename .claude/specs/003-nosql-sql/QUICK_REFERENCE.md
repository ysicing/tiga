# 数据库管理系统 - 快速参考卡

> **立即开始**: 解决"创建数据库实例没成功"问题的最快方法

## 🚀 最快测试方法

### 一键测试脚本

```bash
# 1. 编辑凭据
vi scripts/test-database-instance.sh

# 2. 更新这些行:
#    第18-19行: 登录用户名密码
#    第50行: MySQL root密码

# 3. 运行
./scripts/test-database-instance.sh
```

**脚本自动完成**:
- ✅ 登录获取JWT token
- ✅ 创建MySQL测试实例
- ✅ 列出所有实例
- ✅ 测试连接

---

## 🔑 核心问题: 认证要求

**为什么创建失败?**
数据库管理API需要**JWT token + 管理员权限**

### 正确的请求格式

```bash
# ❌ 错误 - 缺少认证
curl -X POST http://localhost:12306/api/v1/database/instances -d '{...}'

# ✅ 正确 - 包含JWT token
curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{...}'
```

---

## 📋 3步手动流程

### 步骤1: 获取Token

```bash
curl -X POST http://localhost:12306/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}'
```

**提取token** (从响应的 `data.token` 字段):
```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 步骤2: 创建实例

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

### 步骤3: 验证

```bash
curl http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer $TOKEN"
```

---

## 🔧 常见错误速查

| 错误信息 | 原因 | 解决方法 |
|---------|------|---------|
| `authorization header required` | 没有token | 先登录获取token |
| `insufficient permissions` | 不是管理员 | 使用管理员账号登录 |
| `connection test failed` | 数据库连不上 | 检查数据库服务是否运行 |
| `encryption service not initialised` | 缺少加密密钥 | config.yaml添加encryption_key |
| `instance name is required` | 请求参数不完整 | 检查必填字段 |

---

## 📦 必填字段清单

创建实例的**所有必填字段**:

```json
{
  "name": "实例名称(唯一)",
  "type": "mysql|postgresql|redis",
  "host": "主机地址",
  "port": 端口号(数字),
  "username": "用户名",
  "password": "密码"
}
```

**可选字段**:
- `ssl_mode`: SSL模式 (disable|require|verify-ca|verify-full)
- `description`: 实例描述

---

## 🧪 快速连接测试

**测试数据库是否可连接** (在创建实例前):

```bash
# MySQL
mysql -h localhost -P 3306 -u root -p

# PostgreSQL
psql -h localhost -p 5432 -U postgres

# Redis
redis-cli -h localhost -p 6379 -a password ping
```

---

## 📚 完整文档索引

- **故障排查**: [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
- **实施报告**: [IMPLEMENTATION_REPORT.md](./IMPLEMENTATION_REPORT.md)
- **详细快速开始**: [quickstart.md](./quickstart.md)
- **数据模型**: [data-model.md](./data-model.md)
- **API规范**: [contracts/database-api.yaml](./contracts/database-api.yaml)

---

## 🎯 快速检查列表

创建实例前确认:

- [ ] 应用正在运行 (`ps aux | grep tiga`)
- [ ] 端口监听正常 (`lsof -i:12306`)
- [ ] 已获取JWT token
- [ ] Token在Authorization header中
- [ ] 使用管理员账号
- [ ] 目标数据库服务运行中
- [ ] 用户名密码正确
- [ ] 主机地址和端口正确

---

## 💡 提示

1. **测试脚本最快**: 如果只是测试功能，直接用 `scripts/test-database-instance.sh`
2. **Token有效期**: JWT token默认24小时有效，过期需要重新登录
3. **密码安全**: 密码使用AES-256加密存储在数据库中
4. **错误日志**: 启用debug日志查看详细错误 (`LOG_LEVEL=debug`)
5. **API文档**: 运行 `./scripts/generate-swagger.sh` 生成Swagger文档

---

**还有问题?** 查看 [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) 获取详细诊断步骤
