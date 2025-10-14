# 技术研究文档：数据库管理系统

**分支**: `003-nosql-sql` | **日期**: 2025-10-10
**目的**: 解决实施计划中的技术未知项,为阶段1设计提供技术基础

---

## 研究1: 数据库驱动选择和最佳实践

### 决策: 使用database/sql + 官方驱动
**选择的技术栈**:
- **MySQL**: `github.com/go-sql-driver/mysql` v1.8+
- **PostgreSQL**: `github.com/lib/pq` v1.10+
- **Redis**: `github.com/redis/go-redis/v9` v9.5+

### 理由
1. **database/sql标准库优势**:
   - Go标准接口,易于mock测试
   - 内置连接池管理(MaxOpenConns, MaxIdleConns)
   - 原生支持context超时和取消
   - 参数化查询防SQL注入

2. **不使用GORM的原因**:
   - 数据库管理场景需要执行任意SQL,GORM的ORM层反而增加复杂度
   - 直接使用database/sql可以获取原始错误信息,便于诊断
   - 避免GORM的hook和自动迁移干扰用户SQL

3. **Redis驱动选择go-redis**:
   - 官方推荐,支持Redis 6.0+ ACL
   - 完善的Pipeline和Cluster支持
   - 类型安全的命令封装

### 连接池配置
```go
// MySQL/PostgreSQL推荐配置
db.SetMaxOpenConns(50)      // 最多50个并发连接(匹配性能目标)
db.SetMaxIdleConns(10)      // 空闲连接池10个(减少频繁建连)
db.SetConnMaxLifetime(5*time.Minute) // 连接最大生命周期5分钟(避免长连接问题)
db.SetConnMaxIdleTime(2*time.Minute) // 空闲连接2分钟关闭

// Redis推荐配置
redis.NewClient(&redis.Options{
    PoolSize:     50,
    MinIdleConns: 5,
    MaxRetries:   3,
    DialTimeout:  5 * time.Second,
    ReadTimeout:  30 * time.Second, // 匹配查询超时
})
```

### 考虑的替代方案
- **sqlx**: 提供便捷的结构体映射,但本场景不需要(直接返回JSON)
- **gorm**: 功能强大但过于重量级,不适合数据库管理工具

---

## 研究2: SQL安全过滤器实现

### 决策: 使用xwb1989/sqlparser + 自定义规则引擎

**技术选择**:
- 主引擎: `github.com/xwb1989/sqlparser` (vitess衍生版,轻量级)
- 辅助: 关键词白名单/黑名单

### 理由
1. **语法解析优于正则**:
   - sqlparser可准确识别SQL AST(抽象语法树)
   - 支持复杂语句(JOIN、子查询、CTE)
   - 避免正则误判(如注释中的DROP)

2. **性能可接受**:
   - 解析1KB SQL约0.5-2ms(远低于10ms目标)
   - 可缓存解析结果(相同SQL模板)

### 实现策略
```go
type SecurityFilter struct {
    bannedStatements []sqlparser.StatementType
    bannedKeywords   []string
}

func (f *SecurityFilter) Validate(sql string) error {
    stmt, err := sqlparser.Parse(sql)
    if err != nil {
        return fmt.Errorf("invalid SQL: %w", err)
    }

    // 检查1: DDL完全禁止
    switch stmt.(type) {
    case *sqlparser.DDL:
        return errors.New("DDL operations are forbidden")
    }

    // 检查2: 危险DML
    if update, ok := stmt.(*sqlparser.Update); ok {
        if update.Where == nil {
            return errors.New("UPDATE without WHERE is forbidden")
        }
    }
    if delete, ok := stmt.(*sqlparser.Delete); ok {
        if delete.Where == nil {
            return errors.New("DELETE without WHERE is forbidden")
        }
    }

    // 检查3: 危险函数
    bannedFuncs := []string{"LOAD_FILE", "INTO OUTFILE", "DUMPFILE"}
    for _, fn := range bannedFuncs {
        if strings.Contains(strings.ToUpper(sql), fn) {
            return fmt.Errorf("dangerous function %s is forbidden", fn)
        }
    }

    return nil
}
```

### Redis命令过滤
- 黑名单: `FLUSHDB`, `FLUSHALL`, `SHUTDOWN`, `CONFIG`, `SAVE`, `BGSAVE`
- 实现: 在go-redis客户端添加Hook拦截

### 考虑的替代方案
- **正则表达式**: 简单但易误判,不采用
- **pingcap/parser**: TiDB的parser,功能完善但依赖重(60MB+),过于庞大

---

## 研究3: Redis ACL权限映射

### 决策: 使用Redis ACL规则模板

**映射策略**:
- **只读角色** → `+@read -@write -@dangerous`
- **管理角色** → `+@read +@write -@dangerous -flushdb -flushall`

### 理由
1. **Redis 6.0+ ACL特性**:
   - 支持命令类别(@read, @write, @dangerous)
   - 支持键模式(~prefix:*)
   - 支持频道模式(&channel)

2. **类别定义**:
   - `@read`: GET, MGET, KEYS, SCAN, EXISTS等
   - `@write`: SET, DEL, INCR, LPUSH等
   - `@dangerous`: FLUSHDB, FLUSHALL, SHUTDOWN, CONFIG等

### 实现流程
```go
func (m *RedisManager) CreateUserWithRole(username, password, role string) error {
    var aclRule string
    switch role {
    case "readonly":
        // 只读:允许读命令,拒绝写和危险命令
        aclRule = fmt.Sprintf("ACL SETUSER %s on >%s ~* +@read -@write -@dangerous",
            username, password)
    case "readwrite":
        // 读写:允许读写,拒绝危险命令
        aclRule = fmt.Sprintf("ACL SETUSER %s on >%s ~* +@read +@write -@dangerous -flushdb -flushall",
            username, password)
    }
    _, err := m.client.Do(ctx, "ACL", "SETUSER", username, "on", ">"+password,
        "~*", "+@read", "+@write", "-@dangerous", "-flushdb", "-flushall").Result()
    return err
}
```

### 数据库级限制
Redis不支持数据库级权限,使用键模式模拟:
- DB 0 → 键模式 `~db0:*`
- DB 1 → 键模式 `~db1:*`
- 前端提示: "Redis权限基于键前缀,建议使用命名空间"

### 考虑的替代方案
- **自定义命令过滤**: 复杂且易遗漏,不如使用Redis原生ACL
- **代理层控制**: 增加架构复杂度,不采用

---

## 研究4: 查询超时和大结果集处理

### 决策: context.WithTimeout + 流式响应 + 虚拟滚动

**超时控制**:
```go
func (e *QueryExecutor) Execute(sql string) (*QueryResult, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    rows, err := e.db.QueryContext(ctx, sql)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return nil, errors.New("query timeout after 30 seconds")
        }
        return nil, err
    }
    defer rows.Close()

    return e.fetchRows(rows, 10*1024*1024) // 10MB限制
}
```

**响应大小限制**:
```go
func (e *QueryExecutor) fetchRows(rows *sql.Rows, maxBytes int64) (*QueryResult, error) {
    var result QueryResult
    var totalBytes int64

    for rows.Next() {
        var row map[string]interface{}
        if err := rows.Scan(/* 动态列扫描 */); err != nil {
            return nil, err
        }

        // 估算行大小(JSON序列化后)
        rowBytes := len(toJSON(row))
        if totalBytes + rowBytes > maxBytes {
            result.Truncated = true
            result.Message = "Result truncated: exceeded 10MB limit"
            break
        }

        totalBytes += rowBytes
        result.Rows = append(result.Rows, row)
    }

    result.RowCount = len(result.Rows)
    return &result, nil
}
```

**前端虚拟滚动**:
- 库选择: `react-window` (轻量,18KB)
- 策略: 每次渲染50行,滚动时动态加载
- 备选: 后端分页API(游标模式,适合超大数据集)

### 考虑的替代方案
- **完全分页**: 需要保存查询状态,复杂度高
- **SSE流式**: 适合实时更新,不适合批量数据传输

---

## 研究5: 审计日志自动清理

### 决策: 复用现有Scheduler + 批次物理删除

**调度策略**:
```go
// 复用 internal/services/scheduler.Scheduler
scheduler.AddTask("database_audit_cleanup",
    schedule.Every(1).Day().At("02:00"), // 每天凌晨2点
    func() {
        cleanupAuditLogs(90) // 保留90天
    })

func cleanupAuditLogs(retentionDays int) error {
    cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

    // 批次删除,每次1000条(避免长事务)
    for {
        result := db.Where("created_at < ?", cutoffDate).
            Limit(1000).
            Delete(&models.DatabaseAuditLog{})

        if result.Error != nil {
            return result.Error
        }
        if result.RowsAffected == 0 {
            break // 删除完成
        }
        time.Sleep(100 * time.Millisecond) // 避免CPU峰值
    }

    return nil
}
```

**性能优化**:
- 使用索引: `created_at` 字段建立索引
- 批次大小: 1000条(平衡性能和锁粒度)
- 执行时间: 凌晨2点(低峰期)

### 可选导出
Phase 2功能(v1.0不实现):
- 导出到Elasticsearch: 使用bulk API批量写入
- 导出到对象存储: 按月归档到MinIO/S3

### 考虑的替代方案
- **软删除**: 浪费存储空间,查询性能下降
- **独立cron**: 增加依赖,不如复用现有Scheduler

---

## 技术决策摘要

| 研究项 | 决策 | 关键依赖 |
|--------|------|----------|
| 数据库驱动 | database/sql + 官方驱动 | go-sql-driver/mysql, lib/pq, go-redis/v9 |
| SQL安全过滤 | xwb1989/sqlparser + 规则引擎 | xwb1989/sqlparser |
| Redis权限 | ACL规则模板(@read, @write) | Redis 6.0+ ACL |
| 查询超时 | context.WithTimeout(30s) | Go context标准库 |
| 结果限制 | 10MB字节计数 + 截断提示 | JSON序列化估算 |
| 虚拟滚动 | react-window | react-window@1.8+ |
| 审计清理 | Scheduler批次删除(90天) | 现有scheduler包 |

---

## 未解决的技术债务

**Phase 2计划**:
1. 审计日志导出到Elasticsearch(高级搜索)
2. 查询结果流式传输(SSE,适合大数据集)
3. SQL查询计划分析(EXPLAIN集成)
4. 多租户隔离(不同租户的数据库实例隔离)

**依赖项版本锁定**:
```
github.com/go-sql-driver/mysql v1.8.1
github.com/lib/pq v1.10.9
github.com/redis/go-redis/v9 v9.5.1
github.com/xwb1989/sqlparser v0.0.0-20180606152119-120387863bf2
react-window ^1.8.10
```

---

*研究完成日期: 2025-10-10*
*下一步: 阶段1设计与契约*
