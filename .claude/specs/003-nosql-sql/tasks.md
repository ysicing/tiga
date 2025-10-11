# 任务：数据库管理系统

**输入**：来自 `.claude/specs/003-nosql-sql/` 的设计文档
**前提条件**：plan.md、research.md、data-model.md、contracts/database-api.yaml、quickstart.md
**分支**：`003-nosql-sql`

## 执行流程
```
1. 已加载 plan.md ✅
   → 技术栈：Go 1.24+ (后端) + TypeScript 5.x/React 19 (前端)
   → 框架：Gin, GORM, Vite, TailwindCSS, Radix UI
   → 结构：Web应用 (internal/, pkg/, ui/src/)
2. 已加载设计文档 ✅
   → data-model.md：6个实体（DatabaseInstance, Database, DatabaseUser, PermissionPolicy, QuerySession, DatabaseAuditLog）
   → contracts/database-api.yaml：25个API端点，6个API组
   → research.md：5个技术决策（驱动、安全、ACL、超时、审计）
   → quickstart.md：5个集成测试场景
3. 任务生成策略 ✅
   → 设置：3个任务（依赖安装、目录结构、安全配置）
   → 测试优先：13个任务（契约测试6 + 集成测试5 + 安全测试2）
   → 核心实现：27个任务（模型6 + 仓储5 + 服务8 + API处理器6 + 前端2）
   → 集成：4个任务（路由、中间件、迁移、调度器）
   → 优化：6个任务（单元测试、性能、文档、代码审查）
4. 并行标记 ✅
   → 不同文件 = [P]（模型、契约测试、前端组件）
   → 同一文件 = 顺序（API处理器共享routes.go）
5. 任务编号：T001-T053（共53个任务）
6. 依赖关系已定义 ✅
7. 并行执行示例已创建 ✅
```

## 格式：`[编号] [P?] 描述`
- **[P]**：可以并行运行（不同文件，无依赖关系）
- 在描述中包含确切的文件路径

## 路径约定
- **后端**：`internal/`（API层、模型、仓储、服务）、`pkg/`（共享库）
- **前端**：`ui/src/`（页面、组件、服务、类型）
- **测试**：`tests/`（contract/、integration/）
- 所有路径基于仓库根目录 `/Users/ysicing/go/src/github.com/ysicing/tiga/`

---

## 阶段 3.1：设置

- [ ] **T001** 根据plan.md创建数据库管理目录结构
  - 创建 `internal/api/handlers/database/`
  - 创建 `internal/models/`（数据库实体）
  - 创建 `internal/repository/database/`
  - 创建 `internal/services/database/`
  - 创建 `pkg/dbdriver/`
  - 创建 `ui/src/pages/database/`
  - 创建 `ui/src/components/database/`
  - 创建 `ui/src/services/database.ts`
  - 创建 `ui/src/types/database.ts`
  - 创建 `tests/contract/`
  - 创建 `tests/integration/database/`

- [ ] **T002** 安装Go依赖（数据库驱动和SQL解析器）
  - 运行 `go get github.com/go-sql-driver/mysql@v1.8.1`
  - 运行 `go get github.com/lib/pq@v1.10.9`
  - 运行 `go get github.com/redis/go-redis/v9@v9.5.1`
  - 运行 `go get github.com/xwb1989/sqlparser@latest`
  - 更新 `go.mod` 和 `go.sum`

- [ ] **T003** [P] 配置环境变量和安全密钥
  - 在 `config.yaml` 添加 `database_management` 配置段
  - 生成 `DB_CREDENTIAL_KEY` 32字节密钥（用于AES-256加密）
  - 在 `.env.example` 添加数据库管理相关环境变量
  - 更新 `internal/config/config.go` 添加DatabaseConfig结构体

---

## 阶段 3.2：测试优先（TDD）⚠️ 必须在 3.3 之前完成
**关键：这些测试必须编写并且必须在任何实现之前失败**

### 契约测试（基于 contracts/database-api.yaml）

- [ ] **T004** [P] 在 `tests/contract/instance_contract_test.go` 中测试实例管理API契约
  - 测试 `GET /api/v1/database/instances` 返回200和实例列表
  - 测试 `POST /api/v1/database/instances` 创建实例返回201
  - 测试 `GET /api/v1/database/instances/{id}` 返回实例详情
  - 测试 `DELETE /api/v1/database/instances/{id}` 删除实例返回200
  - 测试 `POST /api/v1/database/instances/{id}/test` 测试连接返回状态

- [ ] **T005** [P] 在 `tests/contract/database_contract_test.go` 中测试数据库操作API契约
  - 测试 `GET /api/v1/database/instances/{id}/databases` 返回数据库列表
  - 测试 `POST /api/v1/database/instances/{id}/databases` 创建数据库返回201
  - 测试 `DELETE /api/v1/database/databases/{id}` 带confirm_name删除数据库

- [ ] **T006** [P] 在 `tests/contract/user_contract_test.go` 中测试用户管理API契约
  - 测试 `GET /api/v1/database/instances/{id}/users` 返回用户列表
  - 测试 `POST /api/v1/database/instances/{id}/users` 创建用户返回201
  - 测试 `PATCH /api/v1/database/users/{id}` 修改密码返回200
  - 测试 `DELETE /api/v1/database/users/{id}` 删除用户返回200

- [ ] **T007** [P] 在 `tests/contract/permission_contract_test.go` 中测试权限管理API契约
  - 测试 `POST /api/v1/database/permissions` 授予权限返回201
  - 测试 `DELETE /api/v1/database/permissions/{id}` 撤销权限返回200
  - 测试 `GET /api/v1/database/users/{id}/permissions` 返回用户权限列表

- [ ] **T008** [P] 在 `tests/contract/query_contract_test.go` 中测试查询执行API契约
  - 测试 `POST /api/v1/database/instances/{id}/query` 执行SELECT返回QueryResult
  - 测试执行DDL返回400错误 "DDL operations are forbidden"
  - 测试执行无WHERE的DELETE返回400错误

- [ ] **T009** [P] 在 `tests/contract/audit_contract_test.go` 中测试审计日志API契约
  - 测试 `GET /api/v1/database/audit-logs` 返回分页日志
  - 测试带instance_id过滤参数返回过滤结果
  - 测试带operator、action、date过滤器返回正确结果

### 集成测试（基于 quickstart.md 场景）

- [ ] **T010** [P] 在 `tests/integration/database/mysql_instance_test.go` 中测试MySQL实例连接
  - 使用testcontainers启动MySQL 8.0实例
  - 测试创建实例、测试连接、列出数据库
  - 验证版本信息和uptime字段正确返回

- [ ] **T011** [P] 在 `tests/integration/database/postgres_user_test.go` 中测试PostgreSQL用户和权限
  - 使用testcontainers启动PostgreSQL 15实例
  - 测试创建数据库、创建用户、授予只读权限
  - 验证权限策略正确存储（readonly角色）

- [ ] **T012** [P] 在 `tests/integration/database/security_filter_test.go` 中测试SQL安全过滤
  - 测试DDL语句被拦截（DROP、TRUNCATE、ALTER）
  - 测试无WHERE的UPDATE/DELETE被拦截
  - 测试危险函数被拦截（LOAD_FILE、INTO OUTFILE）
  - 验证SELECT语句可正常执行

- [ ] **T013** [P] 在 `tests/integration/database/redis_acl_test.go` 中测试Redis ACL映射
  - 使用testcontainers启动Redis 7实例
  - 测试创建只读用户映射到@read命令
  - 测试创建管理用户排除危险命令（FLUSHDB、FLUSHALL）
  - 验证ACL规则正确应用

- [ ] **T014** [P] 在 `tests/integration/database/query_limit_test.go` 中测试查询限制
  - 测试查询超时30秒（使用长时间查询）
  - 测试10MB响应大小限制和截断提示
  - 验证truncated字段和message正确返回

### 安全测试

- [ ] **T015** [P] 在 `tests/integration/database/credential_encryption_test.go` 中测试凭据加密
  - 测试创建实例时密码AES-256加密存储
  - 验证数据库中密码字段不为明文
  - 测试解密后能正常连接数据库

- [ ] **T016** [P] 在 `tests/integration/database/audit_log_test.go` 中测试审计日志记录
  - 测试所有操作都记录审计日志（创建实例、执行查询、授予权限）
  - 验证日志包含operator、action、target_type、client_ip
  - 测试查询被拦截时记录blocked日志

---

## 阶段 3.3：核心实现（仅在测试失败后）

### 数据模型层（基于 data-model.md）

- [ ] **T017** [P] 在 `internal/models/db_instance.go` 中创建DatabaseInstance模型
  - 定义DatabaseInstance结构体（ID、Name、Type、Host、Port等）
  - 添加GORM标签（主键、索引、唯一约束）
  - 实现Password字段的加密/解密方法（BeforeSave/AfterFind钩子）
  - 添加验证规则（Name必填、Type枚举、Port范围1-65535）

- [ ] **T018** [P] 在 `internal/models/db_database.go` 中创建Database模型
  - 定义Database结构体（ID、InstanceID、Name、Charset等）
  - 添加外键关联到DatabaseInstance
  - 支持MySQL/PostgreSQL字段（Charset、Collation、Owner）
  - 支持Redis字段（DBNumber、KeyCount）

- [ ] **T019** [P] 在 `internal/models/db_user.go` 中创建DatabaseUser模型
  - 定义DatabaseUser结构体（ID、InstanceID、Username、Password等）
  - 添加外键关联到DatabaseInstance
  - 实现Password加密存储（BeforeSave钩子）
  - 添加唯一索引（instance_id + username）

- [ ] **T020** [P] 在 `internal/models/db_permission.go` 中创建PermissionPolicy模型
  - 定义PermissionPolicy结构体（UserID、DatabaseID、Role等）
  - 添加外键关联到DatabaseUser和Database
  - 支持Role枚举（readonly、readwrite）
  - 添加软删除（RevokedAt字段）
  - 添加唯一索引（user_id + database_id + role WHERE revoked_at IS NULL）

- [ ] **T021** [P] 在 `internal/models/db_query_session.go` 中创建QuerySession模型
  - 定义QuerySession结构体（ID、InstanceID、ExecutedBy、QuerySQL等）
  - 添加外键关联到DatabaseInstance
  - 添加Status枚举（success、error、timeout、truncated）
  - 添加QueryType枚举（SELECT、INSERT、UPDATE、DELETE、REDIS_CMD）

- [ ] **T022** [P] 在 `internal/models/db_audit_log.go` 中创建DatabaseAuditLog模型
  - 定义DatabaseAuditLog结构体（ID、InstanceID、Operator、Action等）
  - 添加可空外键关联到DatabaseInstance
  - 支持Action点分命名空间（instance.create、database.delete等）
  - 添加Details字段（JSON格式）

### 仓储层（Repository Pattern）

- [ ] **T023** [P] 在 `internal/repository/database/instance.go` 中实现InstanceRepository
  - 实现Create、GetByID、List、Update、Delete方法
  - 实现GetByName查询（唯一名称）
  - 实现ListByType过滤（mysql、postgresql、redis）
  - 使用GORM处理关联加载（Databases、Users）

- [ ] **T024** [P] 在 `internal/repository/database/database.go` 中实现DatabaseRepository
  - 实现Create、GetByID、Delete方法
  - 实现ListByInstance查询（按实例ID）
  - 实现CheckUniqueName验证（同实例下唯一）

- [ ] **T025** [P] 在 `internal/repository/database/user.go` 中实现UserRepository
  - 实现Create、GetByID、Update、Delete方法
  - 实现ListByInstance查询
  - 实现GetByUsername查询（instance_id + username）

- [ ] **T026** [P] 在 `internal/repository/database/permission.go` 中实现PermissionRepository
  - 实现Grant、Revoke、ListByUser、ListByDatabase方法
  - 实现软删除（设置RevokedAt而非物理删除）
  - 实现CheckExisting验证（避免重复授权）

- [ ] **T027** [P] 在 `internal/repository/database/audit.go` 中实现AuditLogRepository
  - 实现Create、List方法
  - 实现Filter方法（支持instance_id、operator、action、date范围过滤）
  - 实现分页查询（page、page_size）
  - 实现DeleteOldLogs方法（删除90天前日志）

### 驱动层（database/sql封装）

- [ ] **T028** [P] 在 `pkg/dbdriver/driver.go` 中定义DatabaseDriver接口
  - 定义Connect、Disconnect、Ping方法
  - 定义ListDatabases、CreateDatabase、DeleteDatabase方法
  - 定义ListUsers、CreateUser、DeleteUser方法
  - 定义ExecuteQuery方法（返回QueryResult）
  - 定义GetVersion、GetUptime方法

- [ ] **T029** [P] 在 `pkg/dbdriver/mysql.go` 中实现MySQLDriver
  - 实现Connect（使用go-sql-driver/mysql）
  - 实现ListDatabases（查询INFORMATION_SCHEMA）
  - 实现CreateDatabase（CREATE DATABASE语句）
  - 实现CreateUser（CREATE USER + GRANT语句）
  - 实现ExecuteQuery（使用database/sql.QueryContext）
  - 配置连接池（MaxOpenConns=50、MaxIdleConns=10）

- [ ] **T030** [P] 在 `pkg/dbdriver/postgres.go` 中实现PostgresDriver
  - 实现Connect（使用lib/pq）
  - 实现ListDatabases（查询pg_database）
  - 实现CreateDatabase（CREATE DATABASE语句）
  - 实现CreateUser（CREATE ROLE + GRANT语句）
  - 实现ExecuteQuery（支持PostgreSQL语法）

- [ ] **T031** [P] 在 `pkg/dbdriver/redis.go` 中实现RedisDriver
  - 实现Connect（使用go-redis/v9）
  - 实现ListDatabases（INFO keyspace命令）
  - 实现CreateUser（ACL SETUSER命令）
  - 实现ExecuteQuery（支持Redis命令）
  - 实现ACL规则映射（readonly→@read、readwrite→@read+@write-@dangerous）

### 服务层（Business Logic）

- [ ] **T032** 在 `internal/services/database/security_filter.go` 中实现SQL安全过滤器
  - 使用xwb1989/sqlparser解析SQL
  - 实现ValidateSQL方法（检查DDL、危险DML、危险函数）
  - 实现DDL拦截（DROP、TRUNCATE、ALTER、CREATE INDEX等）
  - 实现无WHERE的UPDATE/DELETE拦截
  - 实现危险函数拦截（LOAD_FILE、INTO OUTFILE、DUMPFILE）
  - 添加性能测试（<2ms）

- [ ] **T033** 在 `internal/services/database/security_filter.go` 中实现Redis命令过滤器
  - 实现ValidateRedisCommand方法
  - 定义黑名单（FLUSHDB、FLUSHALL、SHUTDOWN、CONFIG、SAVE、BGSAVE）
  - 实现命令解析和匹配（大小写不敏感）

- [ ] **T034** 在 `internal/services/database/manager.go` 中实现DatabaseManager服务
  - 实现CreateInstance（加密密码、测试连接、保存到数据库）
  - 实现TestConnection（使用对应驱动Ping）
  - 实现GetInstance、ListInstances、DeleteInstance方法
  - 实现连接缓存（避免重复建连）

- [ ] **T035** 在 `internal/services/database/database_service.go` 中实现DatabaseService
  - 实现CreateDatabase（调用驱动CreateDatabase + 保存元数据）
  - 实现ListDatabases（从目标数据库查询 + 合并本地元数据）
  - 实现DeleteDatabase（二次确认 + 调用驱动 + 删除元数据）

- [ ] **T036** 在 `internal/services/database/user_service.go` 中实现UserService
  - 实现CreateUser（调用驱动CreateUser + 保存加密密码）
  - 实现UpdatePassword（验证旧密码 + 调用驱动 + 更新数据库）
  - 实现ListUsers、DeleteUser方法

- [ ] **T037** 在 `internal/services/database/permission_service.go` 中实现PermissionService
  - 实现GrantPermission（检查权限存在 + 调用驱动授权 + 保存策略）
  - 实现RevokePermission（软删除策略 + 调用驱动撤销）
  - 实现GetUserPermissions（查询有效权限）
  - 实现MySQL/PostgreSQL权限映射（readonly→SELECT、readwrite→ALL）
  - 实现Redis ACL映射（readonly→@read、readwrite→@read+@write）

- [ ] **T038** 在 `internal/services/database/query_executor.go` 中实现QueryExecutor服务
  - 实现ExecuteQuery方法（context 30秒超时）
  - 实现结果大小限制（10MB字节计数）
  - 实现结果截断（设置truncated=true和message）
  - 实现安全过滤（调用SecurityFilter）
  - 实现QuerySession记录（记录执行指标）

- [ ] **T039** 在 `internal/services/database/audit_logger.go` 中实现AuditLogger服务
  - 实现LogAction方法（记录操作到DatabaseAuditLog）
  - 实现ExtractClientIP（从context获取）
  - 实现FormatDetails（序列化为JSON）
  - 支持所有Action类型（instance.create、query.execute、permission.grant等）

### API处理器层（Handlers）

- [ ] **T040** 在 `internal/api/handlers/database/instance.go` 中实现实例管理处理器
  - 实现ListInstances处理器（GET /api/v1/database/instances）
  - 实现CreateInstance处理器（POST /api/v1/database/instances）
  - 实现GetInstance处理器（GET /api/v1/database/instances/{id}）
  - 实现DeleteInstance处理器（DELETE /api/v1/database/instances/{id}）
  - 实现TestConnection处理器（POST /api/v1/database/instances/{id}/test）
  - 添加Swagger注解

- [ ] **T041** 在 `internal/api/handlers/database/dbops.go` 中实现数据库操作处理器
  - 实现ListDatabases处理器（GET /api/v1/database/instances/{id}/databases）
  - 实现CreateDatabase处理器（POST /api/v1/database/instances/{id}/databases）
  - 实现DeleteDatabase处理器（DELETE /api/v1/database/databases/{id}）
  - 验证confirm_name参数（删除操作）

- [ ] **T042** 在 `internal/api/handlers/database/user.go` 中实现用户管理处理器
  - 实现ListUsers处理器（GET /api/v1/database/instances/{id}/users）
  - 实现CreateUser处理器（POST /api/v1/database/instances/{id}/users）
  - 实现UpdateUserPassword处理器（PATCH /api/v1/database/users/{id}）
  - 实现DeleteUser处理器（DELETE /api/v1/database/users/{id}）

- [ ] **T043** 在 `internal/api/handlers/database/permission.go` 中实现权限管理处理器
  - 实现GrantPermission处理器（POST /api/v1/database/permissions）
  - 实现RevokePermission处理器（DELETE /api/v1/database/permissions/{id}）
  - 实现GetUserPermissions处理器（GET /api/v1/database/users/{id}/permissions）

- [ ] **T044** 在 `internal/api/handlers/database/query.go` 中实现查询执行处理器
  - 实现ExecuteQuery处理器（POST /api/v1/database/instances/{id}/query）
  - 实现安全过滤调用（返回400错误如果拦截）
  - 实现查询结果序列化（columns、rows、row_count、duration）
  - 实现超时处理（返回timeout状态）

- [ ] **T045** 在 `internal/api/handlers/database/audit.go` 中实现审计日志处理器
  - 实现ListAuditLogs处理器（GET /api/v1/database/audit-logs）
  - 实现过滤参数解析（instance_id、operator、action、date范围）
  - 实现分页参数解析（page、page_size，默认1和50）

### 前端实现

- [ ] **T046** 在 `ui/src/pages/database/` 中实现数据库管理页面
  - 实现InstanceList.tsx（实例列表和创建表单）
  - 实现DatabaseList.tsx（数据库列表和操作）
  - 实现UserManagement.tsx（用户管理界面）
  - 实现PermissionManagement.tsx（权限授予和撤销）
  - 实现QueryConsole.tsx（SQL/Redis控制台，集成Monaco Editor）
  - 实现AuditLog.tsx（审计日志查询和筛选）
  - 使用TanStack Query管理服务端状态

- [ ] **T047** 在 `ui/src/components/database/` 中实现数据库UI组件
  - 实现InstanceCard.tsx（实例卡片，显示状态和版本）
  - 实现DatabaseTable.tsx（数据库表格，支持虚拟滚动）
  - 实现UserForm.tsx（用户创建和编辑表单）
  - 实现PermissionSelector.tsx（权限角色选择器）
  - 实现SQLEditor.tsx（集成Monaco Editor语法高亮）
  - 实现QueryResultTable.tsx（查询结果表格，react-window虚拟滚动）
  - 使用Radix UI组件库

---

## 阶段 3.4：集成

- [ ] **T048** 在 `internal/api/routes.go` 中注册数据库管理路由
  - 添加路由组 `/api/v1/database`
  - 注册所有处理器到对应路径
  - 应用认证中间件（JWT验证）
  - 应用RBAC中间件（admin角色要求）
  - 应用审计日志中间件

- [ ] **T049** 在 `internal/db/db.go` 中添加数据库迁移
  - 在AutoMigrate添加6个模型
  - 验证索引正确创建
  - 测试外键约束和级联删除

- [ ] **T050** 在 `internal/services/scheduler/` 中添加审计日志清理任务
  - 添加定时任务：每天凌晨2点执行
  - 实现批次删除逻辑（1000条/批，间隔100ms）
  - 删除90天前的审计日志
  - 添加任务执行日志

- [ ] **T051** 在前端添加数据库管理菜单和路由
  - 在 `ui/src/layouts/` 添加数据库子系统布局
  - 在导航菜单添加"数据库管理"入口
  - 配置前端路由（/database/instances、/database/query等）
  - 添加权限守卫（需要admin角色）

---

## 阶段 3.5：优化

- [ ] **T052** [P] 在 `tests/unit/` 中添加单元测试
  - 在 `tests/unit/security_filter_test.go` 测试SQL解析逻辑
  - 在 `tests/unit/credential_test.go` 测试密码加密/解密
  - 在 `tests/unit/acl_mapping_test.go` 测试Redis ACL规则生成
  - 目标覆盖率：≥70%

- [ ] **T053** [P] 性能测试和优化
  - 测试查询响应时间（目标<2秒，10MB数据）
  - 测试并发连接（目标支持50个实例并发查询）
  - 测试审计日志查询性能（90天数据，目标<2秒）
  - 优化索引和查询（如需要）

- [ ] **T054** [P] 更新API文档
  - 运行 `./scripts/generate-swagger.sh` 生成Swagger文档
  - 验证所有端点在Swagger UI正确显示
  - 更新 `CLAUDE.md` 添加数据库管理功能说明

- [ ] **T055** [P] 代码审查和重构
  - 移除重复代码（提取公共方法）
  - 检查错误处理完整性
  - 验证日志记录充分性
  - 运行 `task lint` 修复代码风格问题

- [ ] **T056** 执行quickstart.md手动验证
  - 启动Docker Compose测试环境
  - 执行所有curl示例命令
  - 验证前端UI所有功能
  - 验证契约测试全部通过
  - 验证集成测试全部通过

---

## 依赖关系

**关键依赖链**：
```
T001(目录结构) → T002(依赖安装) → T003(配置)
  ↓
T004-T016(测试) [必须先失败]
  ↓
T017-T022(模型) [P] → T023-T027(仓储) [P]
  ↓
T028-T031(驱动) [P] → T032-T033(安全过滤器)
  ↓
T034-T039(服务) → T040-T045(API处理器)
  ↓
T046-T047(前端) [P]
  ↓
T048-T051(集成)
  ↓
T052-T056(优化) [P]
```

**详细依赖**：
- T001-T003 无依赖（设置任务）
- T004-T016 依赖 T001（需要测试目录）
- T017-T022 依赖 T002（需要GORM）
- T023-T027 依赖 T017-T022（仓储依赖模型）
- T028-T031 依赖 T002（需要数据库驱动）
- T032 依赖 T002（需要sqlparser）
- T034-T039 依赖 T023-T033（服务依赖仓储和驱动）
- T040-T045 依赖 T034-T039（处理器依赖服务）
- T046-T047 依赖 T003（需要API端点可用）
- T048 依赖 T040-T045（路由注册依赖处理器）
- T049 依赖 T017-T022（迁移依赖模型定义）
- T050 依赖 T027（调度器依赖审计仓储）
- T051 依赖 T046（前端路由依赖页面组件）
- T052-T056 依赖所有实现任务完成

**阻塞关系**：
- T017阻塞T023（InstanceRepository需要DatabaseInstance模型）
- T032阻塞T038（QueryExecutor需要SecurityFilter）
- T034阻塞T040（Instance处理器需要DatabaseManager）
- T048阻塞T056（手动测试需要路由注册）

---

## 并行执行示例

### 并行组1：契约测试（T004-T009可同时运行）
```bash
# 在Claude Code中使用Task工具并行启动6个契约测试
Task: "在 tests/contract/instance_contract_test.go 中测试实例管理API契约"
Task: "在 tests/contract/database_contract_test.go 中测试数据库操作API契约"
Task: "在 tests/contract/user_contract_test.go 中测试用户管理API契约"
Task: "在 tests/contract/permission_contract_test.go 中测试权限管理API契约"
Task: "在 tests/contract/query_contract_test.go 中测试查询执行API契约"
Task: "在 tests/contract/audit_contract_test.go 中测试审计日志API契约"
```

### 并行组2：集成测试（T010-T016可同时运行）
```bash
Task: "在 tests/integration/database/mysql_instance_test.go 中测试MySQL实例连接"
Task: "在 tests/integration/database/postgres_user_test.go 中测试PostgreSQL用户和权限"
Task: "在 tests/integration/database/security_filter_test.go 中测试SQL安全过滤"
Task: "在 tests/integration/database/redis_acl_test.go 中测试Redis ACL映射"
Task: "在 tests/integration/database/query_limit_test.go 中测试查询限制"
Task: "在 tests/integration/database/credential_encryption_test.go 中测试凭据加密"
Task: "在 tests/integration/database/audit_log_test.go 中测试审计日志记录"
```

### 并行组3：模型层（T017-T022可同时运行）
```bash
Task: "在 internal/models/db_instance.go 中创建DatabaseInstance模型"
Task: "在 internal/models/db_database.go 中创建Database模型"
Task: "在 internal/models/db_user.go 中创建DatabaseUser模型"
Task: "在 internal/models/db_permission.go 中创建PermissionPolicy模型"
Task: "在 internal/models/db_query_session.go 中创建QuerySession模型"
Task: "在 internal/models/db_audit_log.go 中创建DatabaseAuditLog模型"
```

### 并行组4：仓储层（T023-T027可同时运行）
```bash
Task: "在 internal/repository/database/instance.go 中实现InstanceRepository"
Task: "在 internal/repository/database/database.go 中实现DatabaseRepository"
Task: "在 internal/repository/database/user.go 中实现UserRepository"
Task: "在 internal/repository/database/permission.go 中实现PermissionRepository"
Task: "在 internal/repository/database/audit.go 中实现AuditLogRepository"
```

### 并行组5：驱动层（T028-T031可同时运行）
```bash
Task: "在 pkg/dbdriver/driver.go 中定义DatabaseDriver接口"
Task: "在 pkg/dbdriver/mysql.go 中实现MySQLDriver"
Task: "在 pkg/dbdriver/postgres.go 中实现PostgresDriver"
Task: "在 pkg/dbdriver/redis.go 中实现RedisDriver"
```

### 并行组6：优化任务（T052-T055可同时运行）
```bash
Task: "在 tests/unit/ 中添加单元测试"
Task: "性能测试和优化"
Task: "更新API文档"
Task: "代码审查和重构"
```

---

## 验证清单
*门禁：在开始实施前检查*

- [x] 所有契约都有对应的测试（6个契约 → 6个测试任务T004-T009）
- [x] 所有实体都有模型任务（6个实体 → 6个模型任务T017-T022）
- [x] 所有测试都在实现之前（阶段3.2在3.3之前）
- [x] 并行任务真正独立（不同文件的[P]任务无文件冲突）
- [x] 每个任务指定确切的文件路径（所有任务包含具体路径）
- [x] 没有任务修改与另一个[P]任务相同的文件（已验证）

---

## 注意事项

**TDD流程**：
1. ⚠️ 必须先运行契约测试（T004-T009）和集成测试（T010-T016），验证它们失败
2. 然后按顺序实现：模型 → 仓储 → 驱动 → 服务 → 处理器
3. 每完成一个任务，重新运行相关测试验证通过

**并行执行规则**：
- [P]标记的任务操作不同文件，可以并行执行
- 同一阶段的[P]任务可以一起启动
- 注意依赖关系，不能跨阶段并行

**提交策略**：
- 每完成一个任务提交一次
- 提交信息格式：`feat(database): [T###] 任务描述`
- 例如：`feat(database): [T017] 创建DatabaseInstance模型`

**测试覆盖率目标**：
- 后端：≥70%（单元测试 + 集成测试）
- 前端：组件测试覆盖核心交互

**性能目标**：
- 查询响应：<2秒（10MB数据）
- 并发连接：支持50个实例
- 审计日志查询：<2秒（90天数据）

---

**任务总数**：56个任务
**预计工作量**：
- 设置：3个任务（T001-T003）
- 测试优先：13个任务（T004-T016）
- 核心实现：31个任务（T017-T047）
- 集成：4个任务（T048-T051）
- 优化：5个任务（T052-T056）

**下一步**：开始执行T001，创建项目目录结构
