# Wire 依赖注入集成

## 概述

本项目已从手动依赖注入迁移到使用 Google Wire 进行编译时依赖注入。

## 什么是 Wire？

Wire 是 Google 开发的编译时依赖注入工具，具有以下特点：

- ✅ **编译时验证**：依赖错误在编译时发现，避免运行时崩溃
- ✅ **零运行时开销**：生成纯 Go 代码，无反射
- ✅ **可读性强**：生成的代码清晰易懂，便于调试
- ✅ **IDE 友好**：支持代码跳转和自动补全

## 项目结构

```
internal/app/
├── app.go           # Application 结构体定义和业务逻辑
├── wire.go          # Wire Provider 定义（构建标签 wireinject）
└── wire_gen.go      # Wire 自动生成的代码（不要手动编辑）
```

## 核心概念

### Provider Sets

Wire 通过 Provider Sets 组织依赖关系：

```go
// DatabaseSet 提供数据库相关依赖
var DatabaseSet = wire.NewSet(
    provideDatabaseConfig,
    db.NewDatabase,
    provideGormDB,
)

// RepositorySet 提供所有仓储层依赖
var RepositorySet = wire.NewSet(
    repository.NewUserRepository,
    wire.Bind(new(repository.UserRepositoryInterface), new(*repository.UserRepository)),
    // ... 其他 repositories
)

// ServiceSet 提供核心服务依赖
var ServiceSet = wire.NewSet(
    services.NewK8sService,
    notification.NewNotificationService,
    managers.NewManagerCoordinator,
    // ... 其他 services
)
```

### Injector 函数

`InitializeApplication` 是主注入函数：

```go
func InitializeApplication(
    ctx context.Context,
    cfg *config.Config,
    configPath string,
    installMode bool,
    staticFS embed.FS,
) (*Application, error)
```

Wire 会自动生成这个函数的实现，按正确顺序创建所有依赖。

## 处理循环依赖

项目中存在 `StateCollector` ↔ `AgentManager` 的循环依赖。解决方案：

```go
// 创建聚合类型
type HostMonitoringComponents struct {
    StateCollector *host.StateCollector
    AgentManager   *host.AgentManager
}

// Provider 函数处理循环引用
func provideHostMonitoringComponents(
    hostRepo repository.HostRepository,
    db *gorm.DB,
) *HostMonitoringComponents {
    stateCollector := host.NewStateCollector(hostRepo)
    agentManager := host.NewAgentManager(hostRepo, stateCollector, db)

    // 完成循环引用
    stateCollector.SetAgentManager(agentManager)

    return &HostMonitoringComponents{
        StateCollector: stateCollector,
        AgentManager:   agentManager,
    }
}
```

## 使用指南

### 生成 Wire 代码

```bash
# 单独生成
task wire

# 构建时自动生成（推荐）
task backend
task dev
```

### 添加新的依赖

1. **在 `wire.go` 中添加 Provider**：

```go
var MyServiceSet = wire.NewSet(
    myservice.NewMyService,
)
```

2. **在 `InitializeApplication` 中引用**：

```go
wire.Build(
    DatabaseSet,
    RepositorySet,
    ServiceSet,
    MyServiceSet,  // 新增
    // ...
    newWireApplication,
)
```

3. **重新生成代码**：

```bash
task wire
```

### 添加新的 Repository 接口

```go
// 1. 在 RepositorySet 中添加
var RepositorySet = wire.NewSet(
    repository.NewMyRepository,
    wire.Bind(new(repository.MyRepositoryInterface), new(*repository.MyRepository)),
)
```

## 与手动依赖注入的对比

### 之前（手动创建，195 行）

```go
func (a *Application) Initialize(ctx context.Context) error {
    userRepo := repository.NewUserRepository(a.db.DB)
    instanceRepo := repository.NewInstanceRepository(a.db.DB)
    metricsRepo := repository.NewMetricsRepository(a.db.DB)
    // ... 20+ 行手动创建

    managerFactory := managers.NewManagerFactory()
    a.coordinator = managers.NewManagerCoordinator(
        managerFactory,
        instanceRepo,
        metricsRepo,
        auditRepo,
    )
    // ... 更多手动连线

    // 手动处理循环依赖
    a.stateCollector = host.NewStateCollector(a.hostRepo)
    a.agentManager = host.NewAgentManager(a.hostRepo, a.stateCollector, a.db.DB)
    a.stateCollector.SetAgentManager(a.agentManager)  // 容易遗漏
    // ...
}
```

### 现在（Wire 自动注入）

```go
func NewApplication(cfg *config.Config, configPath string, installMode bool, staticFS embed.FS) (*Application, error) {
    // Wire 自动创建所有依赖
    app, err := InitializeApplication(context.Background(), cfg, configPath, installMode, staticFS)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize application with wire: %w", err)
    }

    // 仅需执行数据库迁移
    if err := app.db.AutoMigrate(); err != nil {
        return nil, fmt.Errorf("failed to migrate database: %w", err)
    }

    return app, nil
}
```

## 收益总结

1. **代码行数减少**：`Initialize()` 方法从 195 行减少到 120 行（-38%）
2. **依赖关系清晰**：通过 Provider Sets 明确展示依赖图
3. **编译时安全**：依赖缺失在编译时发现，不是运行时
4. **易于测试**：可以轻松创建测试用的 mock Provider
5. **易于维护**：添加新依赖只需更新 wire.go，Wire 自动处理连线

## 常见问题

### Q: wire_gen.go 应该提交到版本控制吗？

**A**: 是的。尽管它是自动生成的，但应该提交到 Git：
- 方便 Code Review
- CI/CD 环境无需安装 wire
- 保证构建的确定性

### Q: 如何调试 Wire 生成错误？

**A**: Wire 错误通常很明确：

```
wire: cannot find provider for *SomeType
```

解决方法：
1. 检查是否在 Provider Set 中定义了 `NewSomeType`
2. 检查是否在 `wire.Build()` 中引用了对应的 Set
3. 查看 Wire 文档：https://github.com/google/wire

### Q: Wire 是否支持运行时动态依赖？

**A**: 不支持。Wire 是**编译时**工具。如果需要运行时动态加载，可以：
- 在初始化后手动创建实例
- 使用 Provider 函数根据配置返回不同实现

## 参考资料

- [Wire 官方文档](https://github.com/google/wire/blob/main/docs/guide.md)
- [Wire 最佳实践](https://github.com/google/wire/blob/main/docs/best-practices.md)
- [项目 Phase 4 架构改进](./phase4-improvements.md)
