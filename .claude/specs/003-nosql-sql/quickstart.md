# 快速启动指南：数据库管理系统

**功能**: 数据库管理系统 (MySQL/PostgreSQL/Redis)
**分支**: `003-nosql-sql`
**前置条件**: 已安装Docker和Docker Compose

---

## 环境准备

### 1. 启动测试数据库实例

创建 `docker-compose.test.yml`:
```yaml
version: '3.8'
services:
  mysql-test:
    image: mysql:8.0
    ports:
      - "3307:3306"
    environment:
      MYSQL_ROOT_PASSWORD: test123456
      MYSQL_DATABASE: testdb
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 3s
      retries: 3

  postgres-test:
    image: postgres:15
    ports:
      - "5433:5432"
    environment:
      POSTGRES_PASSWORD: test123456
      POSTGRES_DB: testdb
    healthcheck:
      test: ["CMD-SHELL", "pg_isready"]
      interval: 5s
      timeout: 3s
      retries: 3

  redis-test:
    image: redis:7-alpine
    ports:
      - "6380:6379"
    command: >
      redis-server
      --requirepass test123456
      --aclfile /usr/local/etc/redis/users.acl
    volumes:
      - ./redis-acl-init.acl:/usr/local/etc/redis/users.acl
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "test123456", "ping"]
      interval: 5s
      timeout: 3s
      retries: 3
```

创建 `redis-acl-init.acl`:
```
user default on >test123456 ~* &* +@all
```

启动测试实例:
```bash
docker-compose -f docker-compose.test.yml up -d

# 验证健康状态
docker-compose -f docker-compose.test.yml ps
```

### 2. 启动Tiga应用

```bash
# 确保在正确的分支
git checkout 003-nosql-sql

# 运行数据库迁移(自动创建新表)
task dev

# 或手动启动
go run cmd/tiga/main.go
```

---

## 首次配置

### 步骤1: 登录Tiga Dashboard
1. 访问 http://localhost:12306
2. 使用管理员账户登录(安装时创建的账户)
3. 进入"数据库管理"模块

### 步骤2: 添加测试实例

**添加MySQL实例**:
```bash
curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MySQL测试实例",
    "type": "mysql",
    "host": "localhost",
    "port": 3307,
    "username": "root",
    "password": "test123456",
    "description": "本地MySQL测试环境"
  }'
```

**添加PostgreSQL实例**:
```bash
curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "PostgreSQL测试实例",
    "type": "postgresql",
    "host": "localhost",
    "port": 5433,
    "username": "postgres",
    "password": "test123456",
    "description": "本地PostgreSQL测试环境"
  }'
```

**添加Redis实例**:
```bash
curl -X POST http://localhost:12306/api/v1/database/instances \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Redis测试实例",
    "type": "redis",
    "host": "localhost",
    "port": 6380,
    "username": "default",
    "password": "test123456",
    "description": "本地Redis测试环境"
  }'
```

### 步骤3: 测试连接

```bash
# 测试MySQL连接
curl -X POST http://localhost:12306/api/v1/database/instances/1/test \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 预期响应:
# {
#   "success": true,
#   "data": {
#     "status": "online",
#     "version": "8.0.35",
#     "uptime": 3600,
#     "message": "Connection successful"
#   }
# }
```

---

## 核心流程演示

### 场景1: 创建数据库和用户

**1. 列出MySQL实例的数据库**:
```bash
curl http://localhost:12306/api/v1/database/instances/1/databases \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**2. 创建新数据库**:
```bash
curl -X POST http://localhost:12306/api/v1/database/instances/1/databases \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "demo_app",
    "charset": "utf8mb4",
    "collation": "utf8mb4_unicode_ci"
  }'
```

**3. 创建数据库用户**:
```bash
curl -X POST http://localhost:12306/api/v1/database/instances/1/users \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "app_user",
    "password": "SecurePass123",
    "host": "%",
    "description": "应用只读用户"
  }'
```

**4. 授予只读权限**:
```bash
curl -X POST http://localhost:12306/api/v1/database/permissions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "database_id": 2,
    "role": "readonly"
  }'
```

### 场景2: 执行SQL查询

**执行SELECT查询**:
```bash
curl -X POST http://localhost:12306/api/v1/database/instances/1/query \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "database_name": "demo_app",
    "query": "SELECT * FROM information_schema.tables LIMIT 10"
  }'
```

**尝试执行DDL(应被拦截)**:
```bash
curl -X POST http://localhost:12306/api/v1/database/instances/1/query \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "database_name": "demo_app",
    "query": "DROP TABLE users"
  }'

# 预期响应:
# {
#   "success": false,
#   "error": "DDL operations are forbidden"
# }
```

### 场景3: Redis操作

**查看Redis数据库**:
```bash
curl http://localhost:12306/api/v1/database/instances/3/databases \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 响应: DB 0-15 及key数量
```

**执行Redis命令**:
```bash
curl -X POST http://localhost:12306/api/v1/database/instances/3/query \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "SET test:key \"hello\""
  }'

curl -X POST http://localhost:12306/api/v1/database/instances/3/query \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "GET test:key"
  }'
```

**尝试执行危险命令(应被拦截)**:
```bash
curl -X POST http://localhost:12306/api/v1/database/instances/3/query \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "FLUSHDB"
  }'

# 预期响应:
# {
#   "success": false,
#   "error": "Dangerous Redis command FLUSHDB is forbidden"
# }
```

### 场景4: 查看审计日志

```bash
curl "http://localhost:12306/api/v1/database/audit-logs?instance_id=1&page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# 响应: 包含所有操作记录
# {
#   "success": true,
#   "data": {
#     "logs": [...],
#     "total": 15,
#     "page": 1,
#     "page_size": 20
#   }
# }
```

---

## 验证步骤

### 契约测试验证

```bash
# 运行契约测试(初始应全部FAIL,因为无实现)
go test ./tests/contract/... -v

# 预期输出:
# === RUN   TestInstanceContract
# --- FAIL: TestInstanceContract (0.01s)
#     instance_contract_test.go:15: Expected 200, got connection refused
# ...
```

### 集成测试验证

```bash
# 运行集成测试(需要测试数据库实例运行)
go test ./tests/integration/database/... -v

# 预期场景:
# ✓ MySQL实例连接和数据库列表查询
# ✓ PostgreSQL用户创建和只读权限授予
# ✓ 危险SQL拦截测试
# ✓ Redis ACL权限映射
# ✓ 超大结果集截断测试
```

### 前端验证

1. **UI访问**: http://localhost:12306/database
2. **功能检查**:
   - [ ] 实例列表显示正常
   - [ ] SQL编辑器支持语法高亮
   - [ ] 查询结果表格支持虚拟滚动
   - [ ] 用户权限选择器正确显示角色
   - [ ] 审计日志筛选功能正常

---

## 常见问题

### Q1: 数据库连接失败
**原因**: 防火墙或网络问题
**解决**:
```bash
# 检查端口监听
netstat -an | grep 3307

# 测试端口连通性
telnet localhost 3307
```

### Q2: 密码加密失败
**原因**: 缺少环境变量 `DB_CREDENTIAL_KEY`
**解决**:
```bash
# 生成32字节密钥
openssl rand -hex 32

# 设置环境变量
export DB_CREDENTIAL_KEY="your-32-byte-hex-key"
```

### Q3: Redis ACL不生效
**原因**: Redis版本 < 6.0
**解决**: 升级Redis到6.0+或禁用ACL功能

---

## 清理环境

```bash
# 停止测试数据库实例
docker-compose -f docker-compose.test.yml down -v

# 删除测试数据
rm -rf ./data/database-test.db
```

---

## 下一步

- 阅读 [data-model.md](./data-model.md) 了解数据模型
- 阅读 [contracts/database-api.yaml](./contracts/database-api.yaml) 了解完整API
- 运行 `/spec-kit:tasks` 生成详细任务列表
- 开始实施: `/spec-kit:implement`

---

*快速启动指南版本: 1.0*
*最后更新: 2025-10-10*
