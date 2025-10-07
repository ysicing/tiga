---
created: 2025-10-06T23:34:01+0800
last_updated: 2025-10-06T23:34:01+0800
version: 1.0
author: Claude Code PM System
---

# 系统架构模式与设计决策

## 🏛️ 整体架构模式

### 1. 后端架构：分层架构 (Layered Architecture)

```
┌─────────────────────────────────────────┐
│          HTTP Handlers Layer            │  ← API 路由和请求处理
│  (internal/api/handlers, pkg/handlers)  │
├─────────────────────────────────────────┤
│         Middleware Layer                │  ← 认证、RBAC、审计、日志
│   (internal/api/middleware,             │
│    pkg/middleware)                      │
├─────────────────────────────────────────┤
│         Service Layer                   │  ← 业务逻辑
│    (internal/services)                  │
├─────────────────────────────────────────┤
│       Repository Layer                  │  ← 数据访问抽象
│    (internal/repository)                │
├─────────────────────────────────────────┤
│         Model Layer                     │  ← 数据模型（GORM）
│    (internal/models)                    │
├─────────────────────────────────────────┤
│        Database Layer                   │  ← SQLite/PostgreSQL/MySQL
│  (SQLite, PostgreSQL, MySQL)            │
└─────────────────────────────────────────┘
```

**优势**:
- 清晰的职责分离
- 易于测试（每层可独立测试）
- 易于维护和扩展
- 降低耦合度

### 2. 前端架构：基于组件的架构 (Component-Based Architecture)

```
┌─────────────────────────────────────────┐
│           App.tsx (Root)                │
├─────────────────────────────────────────┤
│        Routes Configuration             │  ← 路由定义
│         (routes.tsx)                    │
├─────────────────────────────────────────┤
│          Layouts Layer                  │  ← 子系统布局包装器
│  (DevOps, VMs, K8s, Middleware, etc.)  │
├─────────────────────────────────────────┤
│           Pages Layer                   │  ← 页面级组件
│  (List, Detail, Form, Dashboard)       │
├─────────────────────────────────────────┤
│        Components Layer                 │  ← 可复用业务组件
│  (Tables, Charts, Editors, Selectors)  │
├─────────────────────────────────────────┤
│        UI Components Layer              │  ← 基础 UI 组件
│    (Button, Card, Dialog, Input)       │
├─────────────────────────────────────────┤
│      Services/API Layer                 │  ← API 调用
│    (axios, TanStack Query)             │
└─────────────────────────────────────────┘
```

**优势**:
- 组件高度可复用
- 单向数据流
- 易于维护
- 利于团队协作

## 🎨 核心设计模式

### 1. 仓储模式 (Repository Pattern)

**位置**: `internal/repository/`

**定义**:
```go
type UserRepository interface {
    Create(user *models.User) error
    GetByID(id uint) (*models.User, error)
    Update(user *models.User) error
    Delete(id uint) error
    List(page, pageSize int) ([]models.User, int64, error)
}

type userRepository struct {
    db *gorm.DB
}
```

**应用场景**:
- 用户数据访问 (`user_repo.go`)
- 实例数据访问 (`instance_repo.go`)
- 告警数据访问 (`alert_repo.go`)
- 审计日志数据访问 (`audit_repo.go`)

**优势**:
- 数据访问逻辑与业务逻辑分离
- 易于单元测试（可 Mock）
- 支持缓存层透明集成
- 便于切换数据源

**示例**:
```go
// 服务层调用仓储
type UserService struct {
    repo repository.UserRepository
}

func (s *UserService) CreateUser(user *models.User) error {
    // 业务逻辑
    if err := s.validateUser(user); err != nil {
        return err
    }

    // 通过仓储访问数据库
    return s.repo.Create(user)
}
```

### 2. 管理器模式 (Manager Pattern)

**位置**: `internal/services/managers/`

**定义**:
```go
type ServiceManager interface {
    GetType() string
    Connect(instance *models.Instance) error
    Disconnect(instance *models.Instance) error
    GetHealth(instance *models.Instance) (*HealthStatus, error)
    GetMetrics(instance *models.Instance) (*Metrics, error)
}

// 具体管理器实现
type MinIOManager struct {
    // MinIO 特定字段
}

type MySQLManager struct {
    // MySQL 特定字段
}
```

**应用场景**:
- MinIO 管理器 (`minio_manager.go`)
- MySQL 管理器 (`mysql_manager.go`)
- PostgreSQL 管理器 (`postgresql_manager.go`)
- Redis 管理器 (`redis_manager.go`)
- Docker 管理器 (`docker_manager.go`)

**协调器**:
```go
type ManagerCoordinator struct {
    managers map[string]ServiceManager
}

func (c *ManagerCoordinator) GetManager(instanceType string) ServiceManager {
    return c.managers[instanceType]
}
```

**优势**:
- 统一的实例管理接口
- 易于扩展新的实例类型
- 职责清晰
- 便于维护

### 3. 中间件模式 (Middleware Pattern)

**位置**: `internal/api/middleware/`, `pkg/middleware/`

**实现**:
```go
type Middleware func(gin.HandlerFunc) gin.HandlerFunc

// 认证中间件
func AuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)
        if !validateToken(token) {
            c.AbortWithStatus(401)
            return
        }
        c.Next()
    }
}

// RBAC 中间件
func RequireAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        user := getCurrentUser(c)
        if !user.IsAdmin() {
            c.AbortWithStatus(403)
            return
        }
        c.Next()
    }
}
```

**中间件链**:
```
Request → CORS → Logger → Auth → RBAC → RateLimit → Audit → Handler
```

**应用的中间件**:
- CORS 处理 (`pkg/middleware/cors.go`)
- 请求日志 (`pkg/middleware/logger.go`)
- JWT 认证 (`internal/api/middleware/auth.go`)
- RBAC 权限检查 (`internal/api/middleware/rbac.go`)
- 审计日志 (`internal/api/middleware/audit.go`)
- 集群上下文 (`pkg/middleware/cluster.go`)

### 4. 工厂模式 (Factory Pattern)

**应用场景**:

#### Kubernetes 客户端工厂
```go
// pkg/kube/client.go
func GetK8sClient(clusterID uint) (*kubernetes.Clientset, error) {
    // 从缓存获取或创建新客户端
    if client, exists := clientCache[clusterID]; exists {
        return client, nil
    }

    // 创建新客户端
    client := createClient(clusterID)
    clientCache[clusterID] = client
    return client, nil
}
```

#### 通知器工厂
```go
// internal/services/notification/
func CreateNotifier(notifierType string) (Notifier, error) {
    switch notifierType {
    case "email":
        return &EmailNotifier{}, nil
    case "webhook":
        return &WebhookNotifier{}, nil
    case "dingtalk":
        return &DingTalkNotifier{}, nil
    default:
        return nil, errors.New("unknown notifier type")
    }
}
```

### 5. 观察者模式 (Observer Pattern)

**应用场景**:

#### 告警事件处理
```go
// internal/services/alert/processor.go
type AlertProcessor struct {
    notifiers []notification.Notifier
}

func (p *AlertProcessor) NotifyAlertEvent(event *models.AlertEvent) {
    for _, notifier := range p.notifiers {
        go notifier.Send(event)  // 异步通知
    }
}
```

#### WebSocket 日志流
```go
// pkg/handlers/logs_handler.go
// 日志流订阅者接收实时日志更新
```

### 6. 策略模式 (Strategy Pattern)

**应用场景**:

#### 数据库策略
```go
// internal/db/database.go
func InitDatabase(config *Config) (*gorm.DB, error) {
    var dialector gorm.Dialector

    switch config.DBType {
    case "sqlite":
        dialector = sqlite.Open(config.DBPath)
    case "postgres":
        dialector = postgres.Open(config.DSN)
    case "mysql":
        dialector = mysql.Open(config.DSN)
    }

    return gorm.Open(dialector, &gorm.Config{})
}
```

### 7. 单例模式 (Singleton Pattern)

**应用场景**:

#### 集群管理器
```go
// pkg/cluster/cluster_manager.go
var (
    clusterManagerInstance *ClusterManager
    once                   sync.Once
)

func GetClusterManager() *ClusterManager {
    once.Do(func() {
        clusterManagerInstance = &ClusterManager{
            clusters: make(map[uint]*Cluster),
        }
    })
    return clusterManagerInstance
}
```

#### JWT 管理器
```go
// internal/services/auth/jwt_manager.go
// JWT 管理器作为单例，确保密钥一致性
```

## 🔄 数据流模式

### 1. 后端请求处理流程

```
HTTP Request
    ↓
[CORS Middleware] → 跨域请求处理
    ↓
[Logger Middleware] → 请求日志记录
    ↓
[Auth Middleware] → JWT 验证
    ↓
[RBAC Middleware] → 权限检查
    ↓
[Audit Middleware] → 审计日志记录
    ↓
[Handler] → 请求参数验证和解析
    ↓
[Service] → 业务逻辑处理
    ↓
[Repository] → 数据库操作
    ↓
[Database] → 数据持久化
    ↓
[Repository] → 返回数据
    ↓
[Service] → 业务数据转换
    ↓
[Handler] → 响应格式化
    ↓
HTTP Response
```

### 2. 前端数据流

```
User Action
    ↓
[Event Handler] → 组件事件处理
    ↓
[Service/API] → 发起 HTTP 请求
    ↓
[Axios Interceptor] → 添加认证 Token
    ↓
Backend API
    ↓
[TanStack Query] → 缓存管理和状态更新
    ↓
[Component State] → 组件状态更新
    ↓
[Re-render] → UI 更新
    ↓
User sees updated UI
```

### 3. Kubernetes 资源操作流程

```
UI (YAML Editor)
    ↓
[Frontend Service] → 构建 API 请求
    ↓
[Backend Handler] → /api/v1/cluster/:id/resources/apply
    ↓
[Cluster Middleware] → 获取 K8s Client
    ↓
[RBAC Middleware] → K8s 权限检查
    ↓
[Resource Apply Handler] → 应用 YAML
    ↓
[K8s Client] → 调用 Kubernetes API
    ↓
[Resource History] → 记录历史版本
    ↓
[Audit Log] → 记录操作日志
    ↓
Response → 返回结果
```

## 🎯 架构决策记录 (ADR)

### ADR-001: 选择 Gin 作为 Web 框架
**日期**: 项目初始
**状态**: 已采纳
**理由**:
- 高性能（基于 httprouter）
- 简洁的 API
- 中间件支持
- 活跃社区

### ADR-002: 采用仓储模式
**日期**: 项目初始
**状态**: 已采纳
**理由**:
- 数据访问逻辑分离
- 易于测试
- 支持缓存层
- 便于切换数据源

### ADR-003: 从 klog 迁移到 logrus
**日期**: 2025-10
**状态**: 已采纳
**理由**:
- 更好的结构化日志支持
- 更灵活的日志级别控制
- 更好的开发体验
- 与 Kubernetes 日志系统解耦

### ADR-004: 使用 Web 安装向导
**日期**: 2025-10
**状态**: 已采纳
**理由**:
- 更好的用户体验
- 图形化配置界面
- 实时验证
- 降低安装门槛

### ADR-005: 多子系统架构（基于 Gaea 设计模式）
**日期**: 2025-10
**状态**: 已采纳
**理由**:
- 清晰的功能划分
- 独立的导航和布局
- 易于扩展新子系统
- 更好的用户体验

## 🔐 安全架构模式

### 1. 认证流程

```
用户登录
    ↓
[Login Handler] → 验证用户名密码
    ↓
[JWT Manager] → 生成 Access Token + Refresh Token
    ↓
[Session Service] → 创建会话记录
    ↓
返回 Tokens → 前端存储
    ↓
后续请求携带 Access Token
    ↓
[Auth Middleware] → 验证 Token
    ↓
[Context] → 设置当前用户
    ↓
业务处理
```

### 2. RBAC 权限模型

```
User (用户)
    ↓
has many
    ↓
Roles (角色)
    ↓
has many
    ↓
Permissions (权限)
    ↓
对应
    ↓
Resources + Actions (资源 + 操作)
```

**权限检查**:
```go
// RBAC 中间件
func RBACMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        user := getCurrentUser(c)
        resource := c.Param("resource")
        action := c.Request.Method

        if !hasPermission(user, resource, action) {
            c.AbortWithStatus(403)
            return
        }
        c.Next()
    }
}
```

## 🧩 模块化设计

### 1. 前端模块化

```
ui/src/
├── pages/            # 页面模块（按功能划分）
├── components/       # 组件模块（按复用性划分）
├── layouts/          # 布局模块（按子系统划分）
├── services/         # API 服务模块
├── contexts/         # 状态管理模块
├── hooks/            # Hooks 模块
└── types/            # 类型定义模块
```

### 2. 后端模块化

```
internal/
├── api/              # API 模块
├── services/         # 业务逻辑模块
├── repository/       # 数据访问模块
├── models/           # 数据模型模块
└── install/          # 安装模块

pkg/
├── handlers/         # K8s 处理器模块
├── kube/             # K8s 客户端模块
├── cluster/          # 集群管理模块
├── auth/             # 认证模块
├── rbac/             # RBAC 模块
└── utils/            # 工具模块
```

## 🔄 缓存策略

### 1. 应用级缓存
- **LRU 缓存**: Kubernetes 客户端缓存
- **在内存缓存**: 集群信息缓存
- **TanStack Query**: 前端 API 响应缓存

### 2. 数据库级缓存
- **优化仓储**: `instance_repo_optimized.go`、`audit_repo_optimized.go`
- **查询缓存**: GORM 查询结果缓存

## 📊 性能优化模式

### 1. 后端性能优化
- **连接池**: GORM 数据库连接池
- **Goroutine 池**: 并发任务处理
- **异步处理**: 告警通知、审计日志
- **批量操作**: 批量查询和插入

### 2. 前端性能优化
- **代码分割**: Vite 动态导入
- **懒加载**: React.lazy + Suspense
- **虚拟滚动**: 长列表优化
- **Debounce/Throttle**: 输入和滚动优化
- **Memoization**: React.memo 和 useMemo

## 🧪 可测试性设计

### 1. 依赖注入
```go
// 服务层接受仓储接口注入
type UserService struct {
    repo repository.UserRepository  // 接口，易于 Mock
}

func NewUserService(repo repository.UserRepository) *UserService {
    return &UserService{repo: repo}
}
```

### 2. 接口抽象
```go
// 定义接口而非具体实现
type NotificationService interface {
    Send(message string) error
}

// 易于创建 Mock 实现用于测试
type MockNotificationService struct {
    SentMessages []string
}
```

## 🔮 未来架构演进方向

### 1. 微服务化（可选）
- 将实例管理器拆分为独立服务
- API Gateway 统一入口
- 服务间通信（gRPC）

### 2. 事件驱动架构
- 引入消息队列（Kafka/RabbitMQ）
- 事件溯源
- CQRS 模式

### 3. 插件系统
- 动态加载插件
- 插件 API 规范
- 插件市场

## 📝 架构原则

1. **单一职责原则 (SRP)**: 每个模块/类只负责一项功能
2. **开闭原则 (OCP)**: 对扩展开放，对修改关闭
3. **依赖倒置原则 (DIP)**: 依赖接口而非具体实现
4. **接口隔离原则 (ISP)**: 使用专门的接口
5. **Don't Repeat Yourself (DRY)**: 避免重复代码
6. **Keep It Simple, Stupid (KISS)**: 保持简单
7. **关注点分离 (SoC)**: 清晰的层次划分
