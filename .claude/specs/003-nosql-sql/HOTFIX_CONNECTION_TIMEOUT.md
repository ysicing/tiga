# 紧急修复: 数据库连接超时问题

**日期**: 2025-10-12
**问题**: 创建数据库实例时响应超时 (75秒返回500错误)
**影响**: MySQL 和 PostgreSQL 实例创建
**状态**: ✅ 已修复并重新编译

---

## 🔴 问题分析

### 症状
用户在前端创建MySQL数据库实例时，请求花费**1分15秒**后返回500错误：

```
ERRO[1493] [GIN] 2025/10/12 - 09:11:18 | 500 | 1m15.003080625s | ::1 |
POST /api/v1/database/instances | user=admin
```

### 根本原因
在 `internal/services/database/manager.go` 的 `CreateInstance` 方法中，第63行会调用 `performConnectionTest()` 来验证数据库连接。

当用户提供的**数据库地址不可达**时（例如数据库服务未运行、网络不通、端口错误等），MySQL/PostgreSQL 驱动会等待**默认的TCP连接超时**，这个值通常是操作系统级别的，可能长达**75秒或更久**。

### 技术细节

**MySQL 驱动 (pkg/dbdriver/mysql.go:35)**:
```go
// 修复前 - 没有超时参数
dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=false&charset=utf8mb4",
    username, cfg.Password, cfg.Host, cfg.Port, sanitizeDatabaseName(cfg.Database))
```

**PostgreSQL 驱动 (pkg/dbdriver/postgres.go:40)**:
```go
// 修复前 - 没有 connect_timeout
dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
    cfg.Host, cfg.Port, username, cfg.Password, cfg.Database, sslmode)
```

**Redis 驱动已正确设置超时**:
```go
// pkg/dbdriver/redis.go:50-52
DialTimeout:  5 * time.Second,
ReadTimeout:  30 * time.Second,
WriteTimeout: 30 * time.Second,
```

---

## ✅ 修复方案

### 修改内容

#### 1. MySQL 驱动超时参数

**文件**: `pkg/dbdriver/mysql.go:35`

**修改后**:
```go
dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=false&charset=utf8mb4&timeout=10s&readTimeout=30s&writeTimeout=30s",
    username, cfg.Password, cfg.Host, cfg.Port, sanitizeDatabaseName(cfg.Database))
```

**新增参数**:
- `timeout=10s`: 建立连接的最大等待时间（10秒）
- `readTimeout=30s`: 读取数据的超时时间（30秒）
- `writeTimeout=30s`: 写入数据的超时时间（30秒）

#### 2. PostgreSQL 驱动超时参数

**文件**: `pkg/dbdriver/postgres.go:40`

**修改后**:
```go
dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=10",
    cfg.Host, cfg.Port, username, cfg.Password, cfg.Database, sslmode)
```

**新增参数**:
- `connect_timeout=10`: 连接超时时间（10秒）

---

## 📊 影响评估

### 修复前
- ❌ 数据库不可达时等待 **75秒+**
- ❌ 用户体验极差（长时间无响应）
- ❌ 可能导致多个并发请求堆积
- ❌ 前端可能提前超时但后端仍在等待

### 修复后
- ✅ 连接失败在 **10秒内**快速返回错误
- ✅ 用户能及时收到错误反馈
- ✅ 减少服务器资源占用
- ✅ 前端超时设置更合理 (可设置为15秒)

### 超时时间选择理由

**连接超时 (10秒)**:
- 局域网连接通常 <1秒
- 跨区域网络连接通常 2-5秒
- 10秒足够应对网络抖动
- 不会让用户等待太久

**读写超时 (30秒)**:
- 查询执行时间限制（与 `config.yaml` 中 `query_timeout_seconds: 30` 一致）
- 足够处理大多数正常查询
- 防止慢查询阻塞连接池

---

## 🧪 验证步骤

### 1. 测试连接超时

**模拟不可达的数据库**:
```bash
# 使用一个未监听的端口
curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Unreachable MySQL",
    "type": "mysql",
    "host": "192.0.2.1",
    "port": 3306,
    "username": "root",
    "password": "test"
  }'

# 预期: 10秒内返回 "connection test failed" 错误
```

### 2. 测试正常连接

**连接真实数据库**:
```bash
# 确保MySQL运行
docker run -d --name mysql-test \
  -e MYSQL_ROOT_PASSWORD=test123 \
  -p 3306:3306 \
  mysql:8.0

# 等待启动
sleep 10

# 创建实例
curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test MySQL",
    "type": "mysql",
    "host": "localhost",
    "port": 3306,
    "username": "root",
    "password": "test123"
  }'

# 预期: 1-2秒内成功创建
```

### 3. 测试PostgreSQL

```bash
# 测试不可达的PostgreSQL
curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Unreachable PG",
    "type": "postgresql",
    "host": "192.0.2.1",
    "port": 5432,
    "username": "postgres",
    "password": "test"
  }'

# 预期: 10秒内返回错误
```

---

## 📝 用户指南更新

### 错误消息改进

修复后，用户会看到更快的错误响应：

**修复前**:
- 等待 75+ 秒
- 可能看到 "Gateway Timeout" (如果前端超时)
- 难以判断是网络问题还是配置错误

**修复后**:
- 10秒内收到明确错误
- 错误消息: `"connection test failed: dial tcp 192.0.2.1:3306: i/o timeout"`
- 用户可以立即检查主机地址、端口、网络连接

### 故障排查建议

当看到 "connection test failed" 错误时，检查：

1. **数据库服务是否运行**
   ```bash
   # MySQL
   systemctl status mysql
   # 或
   docker ps | grep mysql

   # PostgreSQL
   systemctl status postgresql
   # 或
   docker ps | grep postgres
   ```

2. **端口是否正确**
   ```bash
   # 检查监听端口
   netstat -an | grep 3306
   lsof -i:3306
   ```

3. **网络连通性**
   ```bash
   # 测试TCP连接
   telnet localhost 3306
   # 或
   nc -zv localhost 3306
   ```

4. **防火墙规则**
   ```bash
   # Linux
   iptables -L -n | grep 3306

   # macOS
   sudo pfctl -s rules | grep 3306
   ```

5. **数据库凭据**
   ```bash
   # 手动测试登录
   mysql -h localhost -P 3306 -u root -p
   ```

---

## 🔄 部署清单

### 重新启动服务

修复已编译到新的二进制文件中：

```bash
# 1. 停止旧服务
pkill tiga

# 2. 启动新服务
./bin/tiga

# 或使用task
task dev
```

### 验证修复

```bash
# 查看日志确认新版本
tail -f /var/log/tiga/app.log

# 测试快速失败
time curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","type":"mysql","host":"192.0.2.1","port":3306,"username":"root","password":"test"}'

# 应该在 10-11 秒内完成
```

---

## 📚 相关文档

- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - 已更新错误排查步骤
- [QUICK_REFERENCE.md](./QUICK_REFERENCE.md) - 已更新常见错误说明
- [MySQL Driver Timeout Docs](https://github.com/go-sql-driver/mysql#timeout)
- [PostgreSQL Connection Strings](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)

---

## 🎯 总结

### 修复前后对比

| 指标 | 修复前 | 修复后 |
|-----|-------|-------|
| 连接超时 | 75+ 秒 | 10 秒 |
| 用户体验 | ❌ 长时间卡住 | ✅ 快速失败 |
| 错误反馈 | ❌ 延迟反馈 | ✅ 及时反馈 |
| 资源占用 | ❌ 连接堆积 | ✅ 快速释放 |

### 推荐配置

```yaml
# config.yaml
database_management:
  connection_timeout: 10        # 连接超时（秒）
  query_timeout_seconds: 30     # 查询超时（秒）
  max_result_bytes: 10485760    # 10MB 结果大小限制
```

---

**修复状态**: ✅ 已完成并测试
**二进制文件**: `bin/tiga` (2025-10-12 09:17)
**下一步**: 在生产环境部署并监控响应时间

如有疑问，请参考 [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) 或提交 Issue。
