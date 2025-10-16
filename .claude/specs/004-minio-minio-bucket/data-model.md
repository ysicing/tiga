# 数据模型：MinIO 对象存储管理系统

**功能**：004-minio-minio-bucket | **日期**：2025-10-14
**参考**：[spec.md](./spec.md) | [research.md](./research.md)

## 模型概述

本文档定义 MinIO 管理系统的数据模型，所有模型使用 GORM 映射到关系数据库（SQLite/PostgreSQL/MySQL）。

## 实体关系图

```
MinIOInstance (1) ----< (N) MinIOUser
                |
                ----< (N) BucketPermission >---- (N) MinIOUser
                |
                ----< (N) MinIOShareLink
                |
                ----< (N) MinIOAuditLog

MinIOUser (1) ----< (N) BucketPermission
          (1) ----< (N) MinIOShareLink
```

## 核心实体

### 1. MinIOInstance（MinIO 实例）

**用途**：存储 MinIO 服务器连接信息和元数据

**字段**：
```go
type MinIOInstance struct {
    ID          uint      `gorm:"primarykey"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`

    // 基本信息
    Name        string `gorm:"size:100;not null;index"`        // 实例名称
    Description string `gorm:"size:500"`                       // 描述
    Endpoint    string `gorm:"size:255;not null;uniqueIndex"`  // MinIO 端点 (host:port)
    UseSSL      bool   `gorm:"default:true"`                   // 是否使用 HTTPS

    // 凭据（加密存储）
    AccessKey   string `gorm:"size:255;not null"`  // 访问密钥
    SecretKey   string `gorm:"size:255;not null"`  // 密钥（AES-256-GCM 加密）

    // 状态和元数据
    Status      string    `gorm:"size:20;index"`         // online/offline/error
    Version     string    `gorm:"size:50"`               // MinIO 版本
    TotalSize   int64     `gorm:"default:0"`             // 总存储空间（字节）
    UsedSize    int64     `gorm:"default:0"`             // 已用空间（字节）
    LastChecked time.Time                                // 最后健康检查时间

    // 关联
    Users       []MinIOUser       `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE"`
    Permissions []BucketPermission `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE"`
    ShareLinks  []MinIOShareLink  `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE"`
    AuditLogs   []MinIOAuditLog   `gorm:"foreignKey:InstanceID;constraint:OnDelete:CASCADE"`
}
```

**索引**：
- `idx_minio_instances_deleted_at`：软删除查询
- `idx_minio_instances_name`：名称搜索
- `idx_minio_instances_endpoint`：唯一端点
- `idx_minio_instances_status`：状态筛选

**验证规则**：
- Name：必填，长度 1-100
- Endpoint：必填，格式 host:port，唯一
- AccessKey/SecretKey：必填
- SecretKey：创建时加密存储（pkg/crypto）

### 2. MinIOUser（MinIO 用户）

**用途**：存储 MinIO 用户账户信息

**字段**：
```go
type MinIOUser struct {
    ID         uint      `gorm:"primarykey"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
    DeletedAt  gorm.DeletedAt `gorm:"index"`

    // 关联实例
    InstanceID uint          `gorm:"not null;index"`
    Instance   MinIOInstance `gorm:"foreignKey:InstanceID"`

    // 用户信息
    Username   string `gorm:"size:100;not null;index:idx_instance_username,unique"` // MinIO 用户名
    AccessKey  string `gorm:"size:255;not null"`                                     // 访问密钥
    SecretKey  string `gorm:"size:255;not null"`                                     // 密钥（加密）
    Status     string `gorm:"size:20;default:active"`                                // active/disabled
    Description string `gorm:"size:500"`                                             // 用户描述

    // 关联
    Permissions []BucketPermission `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
    ShareLinks  []MinIOShareLink   `gorm:"foreignKey:CreatedBy"`
}
```

**索引**：
- `idx_instance_username`：复合唯一索引（InstanceID + Username）
- `idx_minio_users_deleted_at`：软删除

**验证规则**：
- Username：必填，长度 3-100，同实例内唯一
- SecretKey：创建时加密，仅返回一次
- Status：active/disabled

### 3. BucketPermission（Bucket 权限）

**用途**：定义用户对 Bucket 的访问权限

**字段**：
```go
type BucketPermission struct {
    ID         uint      `gorm:"primarykey"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
    DeletedAt  gorm.DeletedAt `gorm:"index"`

    // 关联
    InstanceID uint          `gorm:"not null;index"`
    Instance   MinIOInstance `gorm:"foreignKey:InstanceID"`

    UserID     uint       `gorm:"not null;index:idx_user_bucket,unique"`
    User       MinIOUser  `gorm:"foreignKey:UserID"`

    // 权限范围
    BucketName string `gorm:"size:100;not null;index:idx_user_bucket,unique"` // Bucket 名称
    Prefix     string `gorm:"size:500;default:''"`                             // 前缀（目录级权限）

    // 权限类型
    Permission string `gorm:"size:20;not null"` // readonly/writeonly/readwrite

    // 元数据
    GrantedBy  uint   `gorm:"not null"`         // 授权者用户ID
    Description string `gorm:"size:500"`        // 权限描述
}
```

**索引**：
- `idx_user_bucket`：复合唯一索引（UserID + BucketName + Prefix）
- `idx_bucket_permissions_instance_id`：实例查询
- `idx_bucket_permissions_deleted_at`：软删除

**验证规则**：
- BucketName：必填，符合 S3 命名规范
- Permission：readonly/writeonly/readwrite
- Prefix：可选，目录级权限（如 "team-a/" 或 "projects/2024/"）
- 同用户对同 Bucket 同前缀的权限唯一

### 4. MinIOShareLink（分享链接）

**用途**：记录分享链接创建信息（用于审计）

**字段**：
```go
type MinIOShareLink struct {
    ID         uint      `gorm:"primarykey"`
    CreatedAt  time.Time
    UpdatedAt  time.Time

    // 关联
    InstanceID uint          `gorm:"not null;index"`
    Instance   MinIOInstance `gorm:"foreignKey:InstanceID"`

    // 文件信息
    BucketName string `gorm:"size:100;not null;index"` // Bucket 名称
    ObjectKey  string `gorm:"size:1000;not null"`      // 对象键（完整路径）

    // 分享信息
    Token      string    `gorm:"size:100;uniqueIndex"` // 分享令牌（可选，用于撤销）
    ExpiresAt  time.Time `gorm:"index"`                // 过期时间
    Status     string    `gorm:"size:20;default:active"` // active/revoked/expired

    // 创建者
    CreatedBy  uint      `gorm:"not null;index"`
    Creator    MinIOUser `gorm:"foreignKey:CreatedBy"`

    // 访问统计（可选）
    AccessCount int `gorm:"default:0"` // 访问次数（需要额外实现追踪）
}
```

**索引**：
- `idx_minio_share_links_instance_id`：实例查询
- `idx_minio_share_links_bucket_name`：Bucket 筛选
- `idx_minio_share_links_created_by`：创建者查询
- `idx_minio_share_links_expires_at`：过期清理
- `idx_minio_share_links_token`：唯一令牌

**验证规则**：
- ExpiresAt：必须晚于当前时间
- Status：active/revoked/expired
- Token：生成唯一标识（UUID）

**说明**：本表主要用于审计，实际分享使用 MinIO presigned URL，无需额外验证。

### 5. MinIOAuditLog（审计日志）

**用途**：记录所有 MinIO 管理操作

**字段**：
```go
type MinIOAuditLog struct {
    ID         uint      `gorm:"primarykey"`
    CreatedAt  time.Time `gorm:"index"`

    // 关联实例
    InstanceID uint          `gorm:"not null;index"`
    Instance   MinIOInstance `gorm:"foreignKey:InstanceID"`

    // 操作信息
    OperationType string `gorm:"size:50;not null;index"` // 操作类型
    ResourceType  string `gorm:"size:50;not null"`       // 资源类型
    ResourceName  string `gorm:"size:255;index"`         // 资源名称
    Action        string `gorm:"size:500;not null"`      // 操作描述

    // 操作者
    OperatorID    uint   `gorm:"not null;index"`         // 操作者用户ID
    OperatorName  string `gorm:"size:100"`               // 操作者用户名
    ClientIP      string `gorm:"size:45"`                // 客户端IP

    // 操作结果
    Status        string `gorm:"size:20;not null"` // success/failed
    ErrorMessage  string `gorm:"type:text"`        // 错误信息（如果失败）

    // 额外信息
    Details       string `gorm:"type:text"` // JSON 格式的详细信息
}
```

**操作类型（OperationType）**：
- `instance_create/instance_update/instance_delete/instance_test`
- `bucket_create/bucket_delete/bucket_update_policy`
- `user_create/user_delete`
- `permission_grant/permission_revoke`
- `file_upload/file_delete`
- `share_create/share_revoke`

**资源类型（ResourceType）**：
- `instance/bucket/user/permission/file/share`

**索引**：
- `idx_minio_audit_logs_created_at`：时间查询
- `idx_minio_audit_logs_instance_id`：实例筛选
- `idx_minio_audit_logs_operation_type`：操作类型筛选
- `idx_minio_audit_logs_operator_id`：操作者查询
- `idx_minio_audit_logs_resource_name`：资源搜索

**清理策略**：定时任务每天 2AM 删除 90 天前的日志

## 状态转换

### MinIOInstance.Status
```
online  ←→ offline  →  error
```
- online：连接正常
- offline：暂时离线（可恢复）
- error：连接错误（需要检查配置）

### MinIOUser.Status
```
active  ←→  disabled
```
- active：正常使用
- disabled：已禁用（保留用户但无法访问）

### MinIOShareLink.Status
```
active  →  expired  or  revoked
```
- active：有效
- expired：已过期（自动）
- revoked：已撤销（手动）

## 数据迁移

**GORM AutoMigrate**：
```go
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &models.MinIOInstance{},
        &models.MinIOUser{},
        &models.BucketPermission{},
        &models.MinIOShareLink{},
        &models.MinIOAuditLog{},
    )
}
```

**手动索引创建**（如需要）：
```go
// 复合索引
db.Exec("CREATE UNIQUE INDEX idx_instance_username ON minio_users(instance_id, username, deleted_at)")
db.Exec("CREATE UNIQUE INDEX idx_user_bucket ON bucket_permissions(user_id, bucket_name, prefix, deleted_at)")
```

## 加密字段

**加密字段**：
- `MinIOInstance.SecretKey`
- `MinIOUser.SecretKey`

**加密方法**：
```go
// 保存前加密
func (m *MinIOInstance) BeforeCreate(tx *gorm.DB) error {
    encrypted, err := crypto.Encrypt(m.SecretKey)
    if err != nil {
        return err
    }
    m.SecretKey = encrypted
    return nil
}

// 查询后解密
func (m *MinIOInstance) AfterFind(tx *gorm.DB) error {
    decrypted, err := crypto.Decrypt(m.SecretKey)
    if err != nil {
        return err
    }
    m.SecretKey = decrypted
    return nil
}
```

**注意**：SecretKey 仅在创建时返回给用户，后续不可查询原文。

## 性能优化

**分区表**（可选，大规模部署）：
- `MinIOAuditLog`：按月分区（审计日志量大）

**缓存策略**：
- MinIO Client 缓存（InstanceID → *minio.Client）
- Presigned URL 缓存（15 分钟 TTL）

**查询优化**：
- 避免 N+1 查询：使用 GORM Preload
- 审计日志查询：使用时间范围 + 复合索引

---
*数据模型版本：v1.0*
*参考 GORM 文档：https://gorm.io/docs*
