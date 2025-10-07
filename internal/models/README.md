# Models 使用指南

## 📋 BaseModel 标准化方案

为了统一数据模型的时间戳字段管理，项目提供了三种基础模型供选择使用。

## 🔧 基础模型类型

### 1. BaseModel - 标准模型（推荐）

**适用场景**：大部分业务模型
- ✅ 需要软删除功能
- ✅ 需要创建/更新时间
- ✅ 支持误删恢复

**字段**：
- `ID` (UUID) - 主键，自动生成
- `CreatedAt` - 创建时间，GORM 自动设置
- `UpdatedAt` - 更新时间，GORM 自动更新
- `DeletedAt` - 软删除时间，带索引

**使用示例**：
```go
type User struct {
    BaseModel
    Username string `gorm:"uniqueIndex;not null" json:"username"`
    Email    string `gorm:"uniqueIndex;not null" json:"email"`
    Status   string `gorm:"type:varchar(32);default:'active'" json:"status"`
}
```

**适用模型**：
- 用户 (User)
- 实例 (Instance)
- 集群 (Cluster)
- OAuth 提供商 (OAuthProvider)
- 告警规则 (Alert)
- 告警事件 (AlertEvent) - 建议迁移
- 后台任务 (BackgroundTask) - 建议迁移
- 备份 (Backup)
- 角色 (Role)

### 2. BaseModelWithoutSoftDelete - 无软删除模型

**适用场景**：临时数据、会话数据
- ❌ 不需要软删除
- ✅ 需要创建/更新时间
- ✅ 过期即删除

**字段**：
- `ID` (UUID) - 主键，自动生成
- `CreatedAt` - 创建时间
- `UpdatedAt` - 更新时间

**使用示例**：
```go
type Session struct {
    BaseModelWithoutSoftDelete
    Token     string    `gorm:"uniqueIndex;not null" json:"token"`
    UserID    uuid.UUID `gorm:"type:char(36);not null;index" json:"user_id"`
    ExpiresAt time.Time `json:"expires_at"`
}
```

**适用模型**：
- 会话 (Session)
- 临时令牌 (TempToken)
- 缓存条目 (CacheEntry)

### 3. AppendOnlyModel - 仅追加模型

**适用场景**：日志、审计、时序数据
- ❌ 不可修改
- ❌ 不可删除
- ✅ 只记录创建时间

**字段**：
- `ID` (UUID) - 主键，自动生成
- `CreatedAt` - 创建时间，带索引

**使用示例**：
```go
type AuditLog struct {
    AppendOnlyModel
    Action   string    `gorm:"type:varchar(128);not null;index" json:"action"`
    UserID   uuid.UUID `gorm:"type:char(36);index" json:"user_id"`
    Resource string    `gorm:"type:varchar(255)" json:"resource"`
    Details  JSONB     `gorm:"type:text" json:"details"`
}
```

**适用模型**：
- 审计日志 (AuditLog)
- 指标数据 (Metric)
- 事件日志 (Event)
- 操作记录 (OperationLog)

## 📊 选择指南

```
需要软删除？
├─ 是 → 使用 BaseModel
│      例：用户、实例、集群、告警
│
└─ 否 → 需要修改记录？
       ├─ 是 → 使用 BaseModelWithoutSoftDelete
       │      例：会话、临时数据
       │
       └─ 否 → 使用 AppendOnlyModel
              例：审计日志、指标、事件
```

## 🚀 新建模型最佳实践

### 1. 嵌入基础模型
```go
type NewModel struct {
    BaseModel  // 选择合适的基础模型

    // 业务字段
    Name   string `gorm:"type:varchar(255);not null" json:"name"`
    Status string `gorm:"type:varchar(32);default:'active'" json:"status"`
}
```

### 2. 指定表名（可选）
```go
func (NewModel) TableName() string {
    return "new_models"
}
```

### 3. BeforeCreate 钩子（如需额外逻辑）
```go
func (m *NewModel) BeforeCreate(tx *gorm.DB) error {
    // 先调用基础模型的钩子（生成 UUID）
    if err := m.BaseModel.BeforeCreate(tx); err != nil {
        return err
    }

    // 自定义逻辑
    if m.Status == "" {
        m.Status = "pending"
    }
    return nil
}
```

## 🔍 查询操作

### 标准查询（自动排除软删除）
```go
// 查询所有未删除记录
var users []User
db.Find(&users)

// 条件查询（自动排除软删除）
db.Where("status = ?", "active").Find(&users)
```

### 包含软删除记录
```go
// 查询所有记录（包括已软删除）
db.Unscoped().Find(&users)

// 仅查询软删除记录
db.Unscoped().Where("deleted_at IS NOT NULL").Find(&users)
```

### 硬删除
```go
// 软删除（默认）
db.Delete(&user)

// 硬删除（永久删除）
db.Unscoped().Delete(&user)
```

### 恢复软删除记录
```go
// 更新 deleted_at 为 NULL
db.Model(&User{}).Unscoped().Where("id = ?", userID).Update("deleted_at", nil)
```

## 📈 数据迁移示例

### 为现有模型添加软删除
```go
// 1. 修改模型定义
type AlertEvent struct {
    BaseModel  // 替换现有字段

    // ... 保留业务字段
}

// 2. 运行迁移（GORM 自动添加字段）
db.AutoMigrate(&AlertEvent{})

// 3. 已有数据的 DeletedAt 自动为 NULL（未删除状态）
```

## ⚠️ 注意事项

1. **UUID 自动生成**
   - 所有基础模型都会在 BeforeCreate 时自动生成 UUID
   - 无需手动设置 ID

2. **软删除索引**
   - `DeletedAt` 字段自动创建索引
   - 提高查询性能

3. **JSON 序列化**
   - `DeletedAt` 使用 `omitempty`，未删除时不会出现在 JSON 中
   - 已删除记录会显示删除时间

4. **外键关联**
   - 软删除的记录，关联查询仍然有效
   - 需要 `Unscoped()` 查询已删除的关联记录

5. **唯一索引**
   - 软删除不影响唯一索引
   - 已删除记录的唯一字段仍然占用索引空间

## 🎯 迁移优先级建议

### 高优先级（建议迁移）
- [ ] AlertEvent → BaseModel
- [ ] BackgroundTask → BaseModel
- [ ] InstanceSnapshot → BaseModel

### 低优先级（按需迁移）
- [ ] Event → BaseModel 或 AppendOnlyModel
- [ ] Metric → 保持现状或 AppendOnlyModel
- [ ] Session → BaseModelWithoutSoftDelete

### 不需要迁移
- [x] AuditLog - 已是 append-only
- [x] SystemConfig - 单例配置，不删除

## 📚 相关资源

- [GORM 软删除文档](https://gorm.io/docs/delete.html#Soft-Delete)
- [UUID 最佳实践](https://gorm.io/docs/data_types.html#UUID)
- [时间戳字段](https://gorm.io/docs/conventions.html#CreatedAt)
