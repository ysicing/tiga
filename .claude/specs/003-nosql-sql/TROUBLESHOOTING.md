# 数据库实例创建故障排查指南

## 问题诊断

如果数据库实例创建失败，请按以下步骤排查：

### 1. 检查应用是否运行

```bash
# 检查Tiga进程
ps aux | grep tiga | grep -v grep

# 检查端口是否监听
lsof -i:12306
```

### 2. 检查认证

数据库管理API需要**管理员权限**。必须先登录获取JWT token。

#### 获取JWT Token

```bash
# 方法1: 使用默认管理员账号（如果已安装）
curl -X POST http://localhost:12306/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-admin-password"
  }'

# 方法2: 使用你创建的账号
curl -X POST http://localhost:12306/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your-username",
    "password": "your-password"
  }'
```

响应示例：
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {...}
  }
}
```

### 3. 使用Token创建实例

```bash
# 设置token变量
TOKEN="your-jwt-token-here"

# 创建MySQL实例
curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MySQL Production",
    "type": "mysql",
    "host": "localhost",
    "port": 3306,
    "username": "root",
    "password": "mysql-root-password",
    "ssl_mode": "disable",
    "description": "生产环境MySQL实例"
  }'
```

### 4. 常见错误及解决方案

#### 错误1: "authorization header or auth_token cookie required"

**原因**: 没有提供JWT token

**解决**:
```bash
# 确保在请求中包含Authorization header
curl -H "Authorization: Bearer YOUR_TOKEN" ...
```

#### 错误2: "connection test failed" 或请求超时

**原因**: 无法连接到目标数据库

**超时设置** (2025-10-12更新):
- MySQL/PostgreSQL连接超时: **10秒**
- Redis连接超时: **5秒**
- 查询执行超时: **30秒**

如果您的请求在10秒左右返回错误，这是**正常的快速失败行为**，表示数据库不可达。

**检查**:
```bash
# MySQL
mysql -h localhost -P 3306 -u root -p

# PostgreSQL
psql -h localhost -p 5432 -U postgres

# Redis
redis-cli -h localhost -p 6379 -a password ping
```

**可能原因**:
- 数据库服务未运行
- 端口不正确
- 用户名密码错误
- 防火墙阻止连接
- SSL模式配置错误
- 网络不可达（会在10秒内快速失败）

**网络连通性测试**:
```bash
# 测试TCP端口是否可达
telnet YOUR_HOST YOUR_PORT
# 或
nc -zv YOUR_HOST YOUR_PORT

# 检查网络路由
ping YOUR_HOST
traceroute YOUR_HOST
```

#### 错误3: "encryption service not initialised"

**原因**: 加密服务未初始化

**解决**:
```bash
# 检查config.yaml中是否有encryption_key
grep encryption_key config.yaml

# 如果没有，需要生成
openssl rand -base64 32
```

将生成的密钥添加到`config.yaml`:
```yaml
security:
  encryption_key: "生成的密钥"
```

#### 错误4: "instance name is required" 或其他验证错误

**原因**: 请求参数不完整

**必需字段**:
- `name`: 实例名称（唯一）
- `type`: 数据库类型（mysql|postgresql|redis）
- `host`: 主机地址
- `port`: 端口号（>0）
- `username`: 用户名
- `password`: 密码

### 5. 检查日志

```bash
# 如果应用在前台运行，直接查看输出

# 如果使用systemd
journalctl -u tiga -f

# 检查应用日志文件（如果配置了）
tail -f /var/log/tiga/app.log
```

### 6. 数据库连接测试

在创建实例之前，先手动测试数据库连接：

#### MySQL测试
```bash
# 测试连接
mysql -h YOUR_HOST -P YOUR_PORT -u YOUR_USERNAME -p

# 或使用telnet测试端口
telnet YOUR_HOST YOUR_PORT
```

#### PostgreSQL测试
```bash
# 测试连接
psql -h YOUR_HOST -p YOUR_PORT -U YOUR_USERNAME -d postgres

# 或使用pg_isready
pg_isready -h YOUR_HOST -p YOUR_PORT
```

#### Redis测试
```bash
# 测试连接
redis-cli -h YOUR_HOST -p YOUR_PORT -a YOUR_PASSWORD ping

# 应该返回 PONG
```

## 完整示例脚本

我已经创建了一个测试脚本：`scripts/test-database-instance.sh`

使用方法：

```bash
# 1. 修改脚本中的凭据
vi scripts/test-database-instance.sh

# 2. 运行脚本
./scripts/test-database-instance.sh
```

## 前端UI创建（如果已实现）

1. 登录Tiga Dashboard: http://localhost:12306
2. 导航到"数据库管理"模块
3. 点击"添加实例"按钮
4. 填写表单:
   - 实例名称
   - 数据库类型（MySQL/PostgreSQL/Redis）
   - 主机地址
   - 端口
   - 用户名
   - 密码
   - SSL模式（可选）
   - 描述（可选）
5. 点击"测试连接"验证
6. 点击"创建"保存

## API端点参考

### 实例管理
- `GET /api/v1/database/instances` - 列出所有实例
- `POST /api/v1/database/instances` - 创建实例
- `GET /api/v1/database/instances/{id}` - 获取实例详情
- `DELETE /api/v1/database/instances/{id}` - 删除实例
- `POST /api/v1/database/instances/{id}/test` - 测试连接

### 权限要求
所有数据库管理API需要：
1. **认证**: JWT token (Bearer token)
2. **授权**: Admin角色

## 调试技巧

### 启用调试日志

在`config.yaml`中设置：
```yaml
log:
  level: debug
  format: json
```

### 使用curl的详细模式

```bash
curl -v -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{...}'
```

### 检查数据库迁移

```bash
# 确保数据库表已创建
sqlite3 /path/to/tiga.db ".tables" | grep db_

# 应该看到:
# db_audit_logs  db_instances  db_permissions  db_query_sessions  db_users  databases
```

## 联系支持

如果问题仍然存在，请提供以下信息：

1. **错误信息**: 完整的错误响应
2. **应用日志**: 最近100行日志
3. **配置**: config.yaml（隐藏敏感信息）
4. **环境**:
   - 操作系统版本
   - Tiga版本
   - 目标数据库版本
5. **重现步骤**: 详细的操作步骤

在GitHub创建issue: https://github.com/ysicing/tiga/issues
