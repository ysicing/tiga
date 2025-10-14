# 数据模型设计：数据库管理系统

**分支**: `003-nosql-sql` | **日期**: 2025-10-10
**依据**: [spec.md](./spec.md) 关键实体, [research.md](./research.md) 技术决策

---

## 模型概览

本功能需要6个核心实体模型,存储在Tiga应用数据库(SQLite/PostgreSQL/MySQL):

| 模型 | 表名 | 用途 | 关系 |
|------|------|------|------|
| DatabaseInstance | db_instances | 数据库实例元数据 | 1:N → Database/DatabaseUser |
| Database | databases | 数据库信息 | N:1 → DatabaseInstance, 1:N → PermissionPolicy |
| DatabaseUser | db_users | 数据库用户 | N:1 → DatabaseInstance, 1:N → PermissionPolicy |
| PermissionPolicy | db_permissions | 权限策略 | N:1 → DatabaseUser, N:1 → Database |
| QuerySession | db_query_sessions | 查询会话记录 | N:1 → DatabaseInstance |
| DatabaseAuditLog | db_audit_logs | 审计日志 | N:1 → DatabaseInstance |

---

## 实体1: DatabaseInstance (数据库实例)

**用途**: 存储数据库实例的连接信息和状态

### 字段定义
```go
type DatabaseInstance struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`

    // 基本信息
    Name        string `gorm:"size:100;not null;uniqueIndex" json:"name"` // 实例名称(唯一)
    Type        string `gorm:"size:20;not null;index" json:"type"`        // mysql|postgresql|redis
    Host        string `gorm:"size:255;not null" json:"host"`             // 主机地址
    Port        int    `gorm:"not null" json:"port"`                      // 端口
    Username    string `gorm:"size:100" json:"username"`                  // 管理员用户名
    Password    string `gorm:"size:500" json:"-"`                         // 加密后的密码(JSON忽略)
    SSLMode     string `gorm:"size:20;default:disable" json:"ssl_mode"`   // disable|require|verify-ca
    Description string `gorm:"size:500" json:"description"`               // 描述

    // 状态和元信息
    Status        string    `gorm:"size:20;default:pending" json:"status"`  // pending|online|offline|error
    LastCheckAt   time.Time `json:"last_check_at"`                          // 最后连接检查时间
    Version       string    `gorm:"size:50" json:"version"`                 // 数据库版本(从连接获取)
    Uptime        int64     `json:"uptime"`                                 // 运行时长(秒,从连接获取)

    // 关联
    Databases []Database     `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE" json:"-"`
    Users     []DatabaseUser `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE" json:"-"`
}
```

### 验证规则
- Name: 必填,1-100字符,唯一
- Type: 枚举值 `mysql|postgresql|redis`
- Host: 必填,支持IP或域名
- Port: 必填,1-65535范围
- Password: 存储前使用AES-256加密(使用 `pkg/crypto`)

### 索引
```sql
CREATE INDEX idx_instances_type ON db_instances(type);
CREATE INDEX idx_instances_status ON db_instances(status);
CREATE UNIQUE INDEX idx_instances_name ON db_instances(name);
```

---

## 实体2: Database (数据库)

**用途**: 存储实例中的数据库信息

### 字段定义
```go
type Database struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`

    // 关联实例
    InstanceID uint             `gorm:"not null;index" json:"instance_id"`
    Instance   DatabaseInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`

    // 数据库信息
    Name      string `gorm:"size:100;not null" json:"name"`     // 数据库名称
    Charset   string `gorm:"size:50" json:"charset"`            // 字符集(MySQL/PG)
    Collation string `gorm:"size:50" json:"collation"`          // 排序规则(MySQL)
    Owner     string `gorm:"size:100" json:"owner"`             // 所有者(PostgreSQL)
    Size      int64  `json:"size"`                              // 大小(字节)
    TableCount int   `json:"table_count"`                       // 表数量(仅展示)

    // Redis特殊字段
    DBNumber int    `gorm:"default:-1" json:"db_number"`        // Redis DB编号(0-15,-1表示非Redis)
    KeyCount int    `json:"key_count"`                          // Redis key数量

    // 关联
    Permissions []PermissionPolicy `gorm:"foreignKey:DatabaseID;constraint:OnDelete:CASCADE" json:"-"`
}
```

### 验证规则
- Name: 必填,1-100字符
- Name唯一性: 同一实例下唯一(复合唯一索引)
- DBNumber: Redis实例时必填(0-15),其他类型为-1

### 索引
```sql
CREATE UNIQUE INDEX idx_databases_instance_name ON databases(instance_id, name);
CREATE INDEX idx_databases_instance ON databases(instance_id);
```

---

## 实体3: DatabaseUser (数据库用户)

**用途**: 存储数据库层面的用户账户

### 字段定义
```go
type DatabaseUser struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`

    // 关联实例
    InstanceID uint             `gorm:"not null;index" json:"instance_id"`
    Instance   DatabaseInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`

    // 用户信息
    Username    string `gorm:"size:100;not null" json:"username"`      // 用户名
    Password    string `gorm:"size:500" json:"-"`                      // 加密密码(JSON忽略)
    Host        string `gorm:"size:255;default:%" json:"host"`         // 允许连接的主机(MySQL)
    Description string `gorm:"size:500" json:"description"`            // 描述

    // 状态
    IsActive bool      `gorm:"default:true" json:"is_active"`          // 是否激活
    LastLoginAt *time.Time `json:"last_login_at"`                      // 最后登录时间

    // 关联
    Permissions []PermissionPolicy `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
}
```

### 验证规则
- Username: 必填,1-100字符
- Username唯一性: 同一实例下唯一(复合唯一索引)
- Password: 最小8位,包含字母和数字(前端验证+后端校验)

### 索引
```sql
CREATE UNIQUE INDEX idx_users_instance_username ON db_users(instance_id, username);
CREATE INDEX idx_users_instance ON db_users(instance_id);
CREATE INDEX idx_users_active ON db_users(is_active);
```

---

## 实体4: PermissionPolicy (权限策略)

**用途**: 定义用户对数据库的访问权限

### 字段定义
```go
type PermissionPolicy struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`

    // 关联用户和数据库
    UserID     uint         `gorm:"not null;index" json:"user_id"`
    User       DatabaseUser `gorm:"foreignKey:UserID" json:"user,omitempty"`
    DatabaseID uint         `gorm:"not null;index" json:"database_id"`
    Database   Database     `gorm:"foreignKey:DatabaseID" json:"database,omitempty"`

    // 权限类型
    Role string `gorm:"size:20;not null" json:"role"` // readonly|readwrite

    // 审计
    GrantedBy  string    `gorm:"size:100" json:"granted_by"`  // 授权人(Tiga用户名)
    GrantedAt  time.Time `json:"granted_at"`                  // 授权时间
    RevokedAt  *time.Time `json:"revoked_at"`                 // 撤销时间(NULL表示有效)
}
```

### 验证规则
- Role: 枚举值 `readonly|readwrite`
- 唯一性: 同一用户+数据库+角色组合唯一(复合唯一索引)
- 撤销逻辑: 软删除,设置RevokedAt时间而非物理删除

### 索引
```sql
CREATE UNIQUE INDEX idx_permissions_user_db_role ON db_permissions(user_id, database_id, role)
    WHERE revoked_at IS NULL;  -- 仅对有效权限建唯一索引
CREATE INDEX idx_permissions_user ON db_permissions(user_id);
CREATE INDEX idx_permissions_database ON db_permissions(database_id);
CREATE INDEX idx_permissions_revoked ON db_permissions(revoked_at);
```

---

## 实体5: QuerySession (查询会话)

**用途**: 记录查询执行的上下文和结果摘要

### 字段定义
```go
type QuerySession struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `json:"created_at"`

    // 关联实例和执行者
    InstanceID uint             `gorm:"not null;index" json:"instance_id"`
    Instance   DatabaseInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
    ExecutedBy string           `gorm:"size:100;not null;index" json:"executed_by"` // Tiga用户名
    DatabaseName string         `gorm:"size:100" json:"database_name"`              // 目标数据库

    // 查询信息
    QuerySQL    string `gorm:"type:text;not null" json:"query_sql"`     // SQL/命令文本
    QueryType   string `gorm:"size:20;index" json:"query_type"`         // SELECT|INSERT|UPDATE|DELETE|REDIS_CMD
    Status      string `gorm:"size:20;not null;index" json:"status"`    // success|error|timeout|truncated
    ErrorMsg    string `gorm:"type:text" json:"error_msg"`              // 错误消息

    // 执行指标
    StartedAt   time.Time `json:"started_at"`                           // 开始时间
    CompletedAt time.Time `json:"completed_at"`                         // 完成时间
    Duration    int       `json:"duration"`                             // 执行时长(毫秒)
    RowCount    int       `json:"row_count"`                            // 返回行数
    BytesReturned int64   `json:"bytes_returned"`                       // 返回字节数

    // 客户端信息
    ClientIP string `gorm:"size:50" json:"client_ip"`                  // 客户端IP
}
```

### 验证规则
- QuerySQL: 必填,最大100KB(防止超大SQL)
- Status: 枚举值 `success|error|timeout|truncated`
- QueryType: 枚举值 `SELECT|INSERT|UPDATE|DELETE|REDIS_CMD|OTHER`

### 索引
```sql
CREATE INDEX idx_sessions_instance ON db_query_sessions(instance_id);
CREATE INDEX idx_sessions_user ON db_query_sessions(executed_by);
CREATE INDEX idx_sessions_status ON db_query_sessions(status);
CREATE INDEX idx_sessions_created ON db_query_sessions(created_at DESC);  -- 时间倒序
```

### 数据保留
- 保留策略: 7天(独立于审计日志)
- 清理策略: 每天凌晨3点删除7天前的记录(与审计日志错峰)

---

## 实体6: DatabaseAuditLog (审计日志)

**用途**: 记录所有数据库管理操作

### 字段定义
```go
type DatabaseAuditLog struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `gorm:"index" json:"created_at"`

    // 关联实例和操作者
    InstanceID *uint            `gorm:"index" json:"instance_id"`                // 可为空(实例删除操作)
    Instance   DatabaseInstance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
    Operator   string           `gorm:"size:100;not null;index" json:"operator"` // Tiga用户名

    // 操作信息
    Action     string `gorm:"size:50;not null;index" json:"action"`     // 操作类型
    TargetType string `gorm:"size:50" json:"target_type"`               // instance|database|user|permission|query
    TargetName string `gorm:"size:255" json:"target_name"`              // 目标对象名称
    Details    string `gorm:"type:text" json:"details"`                 // 详细信息(JSON格式)

    // 结果和客户端
    Success  bool   `gorm:"not null;index" json:"success"`             // 操作是否成功
    ErrorMsg string `gorm:"type:text" json:"error_msg"`                // 错误消息(失败时)
    ClientIP string `gorm:"size:50" json:"client_ip"`                  // 客户端IP
}
```

### 审计操作类型(Action)
- `instance.create`: 创建实例
- `instance.update`: 更新实例
- `instance.delete`: 删除实例
- `instance.test_connection`: 测试连接
- `database.create`: 创建数据库
- `database.delete`: 删除数据库
- `user.create`: 创建用户
- `user.update`: 修改用户
- `user.delete`: 删除用户
- `permission.grant`: 授予权限
- `permission.revoke`: 撤销权限
- `query.execute`: 执行查询(仅记录摘要,详细信息在QuerySession)
- `query.blocked`: 危险查询被拦截

### 验证规则
- Action: 必填,使用点分命名空间
- TargetType: 枚举值 `instance|database|user|permission|query`
- Details: JSON格式(例如: `{"database_name":"test_db","charset":"utf8mb4"}`)

### 索引
```sql
CREATE INDEX idx_audit_instance ON db_audit_logs(instance_id);
CREATE INDEX idx_audit_operator ON db_audit_logs(operator);
CREATE INDEX idx_audit_action ON db_audit_logs(action);
CREATE INDEX idx_audit_created ON db_audit_logs(created_at DESC);  -- 时间倒序
CREATE INDEX idx_audit_success ON db_audit_logs(success);
```

### 数据保留
- 保留策略: 90天(符合规格FR-037)
- 清理策略: 每天凌晨2点批次删除(研究文档策略)

---

## 关系图

```
DatabaseInstance (1)───────(N) Database
       │                          │
       │                          │
       └────────(N) DatabaseUser  │
                     │            │
                     └───(N) PermissionPolicy (N)───┘

DatabaseInstance (1)───────(N) QuerySession
DatabaseInstance (1)───────(N) DatabaseAuditLog
```

---

## 数据迁移策略

### GORM AutoMigrate
```go
// 在 internal/db/db.go 中添加
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.DatabaseInstance{},
        &models.Database{},
        &models.DatabaseUser{},
        &models.PermissionPolicy{},
        &models.QuerySession{},
        &models.DatabaseAuditLog{},
    )
}
```

### 初始数据
无需预置数据,用户通过UI添加实例。

### 向后兼容
- 新表与现有表无冲突(独立命名空间 `db_*`)
- 删除实例时级联删除关联数据(`OnDelete:CASCADE`)

---

## 数据安全

### 敏感字段加密
- `DatabaseInstance.Password`: 使用 `pkg/crypto.Encrypt()` 加密存储
- `DatabaseUser.Password`: 同上
- 加密算法: AES-256-GCM
- 密钥管理: 从环境变量 `DB_CREDENTIAL_KEY` 读取(32字节)

### JSON序列化
- 密码字段添加 `json:"-"` 标签,永不输出到API
- 前端显示为 `******`
- 更新密码时需要新旧密码验证

---

*数据模型设计完成日期: 2025-10-10*
*下一步: API契约设计*
