---
created: 2025-10-06T23:34:01+0800
last_updated: 2025-10-06T23:34:01+0800
version: 1.0
author: Claude Code PM System
---

# 技术栈与依赖项

## 🎯 技术栈概览

### 后端技术栈
- **语言**: Go 1.24.3
- **Web 框架**: Gin 1.11.0
- **ORM**: GORM (glebarez/sqlite, go-sql-driver/mysql, lib/pq)
- **日志**: Logrus 1.9.3 (最近从 klog 迁移)
- **API 文档**: Swagger/Swaggo 1.16.6

### 前端技术栈
- **语言**: TypeScript 5+
- **框架**: React 19
- **构建工具**: Vite
- **UI 库**: Radix UI + shadcn/ui
- **样式**: TailwindCSS 4.1.13
- **状态管理**: React Context + TanStack Query
- **图表**: Recharts
- **编辑器**: Monaco Editor
- **终端**: xterm.js
- **图标**: Tabler Icons React + Lucide React

### 数据库支持
- SQLite (默认，开发环境)
- PostgreSQL (生产推荐)
- MySQL

### 基础设施
- **容器化**: Docker
- **编排**: Kubernetes
- **监控**: Prometheus
- **对象存储**: MinIO
- **缓存**: Redis

## 📦 Go 依赖项详解

### 核心框架和工具

#### Web 框架
```go
github.com/gin-gonic/gin v1.11.0          // HTTP Web 框架
github.com/gin-contrib/gzip v1.2.3        // Gzip 压缩中间件
```

#### ORM 和数据库驱动
```go
// SQLite
github.com/glebarez/sqlite v1.11.0        // 纯 Go SQLite 驱动
github.com/mattn/go-sqlite3 v1.14.32      // CGO SQLite 驱动

// MySQL
github.com/go-sql-driver/mysql v1.9.3     // MySQL 驱动

// PostgreSQL
github.com/lib/pq v1.10.9                 // PostgreSQL 驱动

// GORM (通过 glebarez/sqlite 间接依赖)
gorm.io/gorm                               // ORM 库
```

#### Kubernetes 客户端
```go
k8s.io/client-go                          // Kubernetes Go 客户端
k8s.io/api                                // Kubernetes API 类型
k8s.io/apimachinery                       // Kubernetes API 工具
github.com/go-logr/logr v1.4.3           // 结构化日志接口（K8s 使用）
```

### 认证与安全

```go
github.com/golang-jwt/jwt/v5 v5.3.0       // JWT 令牌生成和验证
github.com/google/uuid v1.6.0             // UUID 生成（Session、ID 等）
golang.org/x/crypto                        // 密码哈希和加密（间接依赖）
```

### 日志系统

```go
github.com/sirupsen/logrus v1.9.3         // 结构化日志库（主要日志系统）
github.com/go-logr/logr v1.4.3            // Kubernetes 日志接口
// 注意：最近从 klog 迁移到 logrus
```

### 缓存

```go
github.com/hashicorp/golang-lru/v2 v2.0.7 // LRU 缓存实现
github.com/redis/go-redis/v9 v9.14.0      // Redis 客户端
```

### 监控和指标

```go
github.com/prometheus/client_golang v1.23.2  // Prometheus 客户端
github.com/prometheus/common v0.66.1         // Prometheus 公共库
```

### 对象存储（MinIO）

```go
github.com/minio/minio-go/v7 v7.0.95       // MinIO Go SDK
github.com/minio/madmin-go/v3 v3.0.110     // MinIO 管理 API
```

### Docker 客户端

```go
github.com/docker/docker v28.0.1+incompatible  // Docker Engine API
```

### 工具库

```go
github.com/samber/lo v1.51.0               // 函数式编程工具（Map、Filter 等）
github.com/blang/semver/v4 v4.0.0         // 语义化版本处理
```

### API 文档（Swagger）

```go
github.com/swaggo/swag v1.16.6            // Swagger 生成器
github.com/swaggo/gin-swagger v1.6.1      // Gin Swagger 中间件
github.com/swaggo/files v1.0.1            // Swagger 静态文件
```

### 测试

```go
github.com/stretchr/testify v1.11.1                // 测试断言和 Mock
github.com/testcontainers/testcontainers-go v0.33.0 // 集成测试容器
github.com/bytedance/mockey v1.2.14                 // Mock 框架
```

### 其他依赖（间接）
- YAML 解析：`gopkg.in/yaml.v3`
- JSON 处理：Go 标准库 `encoding/json`
- HTTP 客户端：Go 标准库 `net/http`

## 🎨 前端依赖项详解

### 核心框架

```json
{
  "react": "^19.0.0",                     // React 框架
  "react-dom": "^19.0.0",                 // React DOM 渲染
  "react-router-dom": "^7.6.1",           // React 路由
  "vite": "^6.2.3",                       // 构建工具
  "typescript": "~5.8.4"                  // TypeScript
}
```

### UI 组件库（Radix UI）

```json
{
  "@radix-ui/react-avatar": "^1.1.10",
  "@radix-ui/react-checkbox": "^1.3.3",
  "@radix-ui/react-collapsible": "^1.1.12",
  "@radix-ui/react-dialog": "^1.1.15",
  "@radix-ui/react-dropdown-menu": "^2.1.16",
  "@radix-ui/react-label": "^2.1.7",
  "@radix-ui/react-popover": "^1.1.15",
  "@radix-ui/react-progress": "^1.1.7",
  "@radix-ui/react-select": "^2.2.6",
  "@radix-ui/react-separator": "^1.1.7",
  "@radix-ui/react-slot": "^1.2.3",
  "@radix-ui/react-switch": "^1.2.6",
  "@radix-ui/react-tabs": "^1.1.13",
  "@radix-ui/react-toggle": "^1.1.10",
  "@radix-ui/react-tooltip": "^1.2.6",
  // ... 更多 Radix 组件
}
```

### 样式和 UI

```json
{
  "tailwindcss": "^4.1.13",               // TailwindCSS
  "@tailwindcss/node": "^4.1.13",         // TailwindCSS Node 集成
  "class-variance-authority": "^0.7.1",   // CVA（组件变体管理）
  "clsx": "^2.1.1",                       // 条件类名
  "tailwind-merge": "^2.6.0"              // TailwindCSS 类名合并
}
```

### 图标

```json
{
  "@tabler/icons-react": "^3.29.0",       // Tabler 图标
  "lucide-react": "^0.511.2"              // Lucide 图标
}
```

### 状态管理和数据获取

```json
{
  "@tanstack/react-query": "^6.6.1",      // 服务端状态管理
  "axios": "^1.7.9",                      // HTTP 客户端
  "zustand": "^5.0.2"                     // 轻量级状态管理（如有使用）
}
```

### 表单处理

```json
{
  "react-hook-form": "^7.56.0",           // 表单管理
  "@hookform/resolvers": "^5.2.2",        // 表单验证解析器
  "zod": "^3.24.1"                        // Schema 验证
}
```

### 图表

```json
{
  "recharts": "^3.2.0"                    // 图表库
}
```

### 编辑器

```json
{
  "@monaco-editor/react": "^4.7.0"        // Monaco Editor（VS Code 编辑器）
}
```

### 终端

```json
{
  "@xterm/xterm": "^5.5.0",               // xterm.js 核心
  "@xterm/addon-fit": "^0.10.0",          // 自适应插件
  "@xterm/addon-search": "^0.15.0"        // 搜索插件
}
```

### 国际化

```json
{
  "i18next": "^24.3.2",                   // i18n 核心
  "react-i18next": "^16.2.1"              // React i18n 绑定
}
```

### 主题和样式

```json
{
  "next-themes": "^0.4.6"                 // 主题管理（支持深色/浅色模式）
}
```

### Toast 通知

```json
{
  "sonner": "^1.7.2"                      // Toast 通知组件
}
```

### 日期处理

```json
{
  "date-fns": "^4.1.0"                    // 日期工具库
}
```

### 开发工具

```json
{
  "@vitejs/plugin-react": "^4.4.1",       // Vite React 插件
  "eslint": "^9.35.0",                    // 代码检查
  "@typescript-eslint/eslint-plugin": "^8.42.0",
  "@typescript-eslint/parser": "^8.42.0",
  "prettier": "^3.4.2",                   // 代码格式化
  "prettier-plugin-tailwindcss": "^0.6.12"
}
```

## 🔧 开发工具和环境

### Go 工具链
- **Go 版本**: 1.24.3
- **构建工具**: `go build`
- **包管理**: `go mod`
- **测试**: `go test`
- **代码检查**: `golangci-lint`

### 前端工具链
- **包管理器**: pnpm (推荐)
- **构建工具**: Vite 6.2.3
- **类型检查**: TypeScript 5.8.4
- **代码检查**: ESLint 9.35.0
- **代码格式化**: Prettier 3.4.2

### 任务管理
- **工具**: Task (Taskfile.yml)
- **用途**: 构建、测试、开发任务自动化

### 容器化
- **Docker**: 用于构建和运行应用
- **Docker Compose**: 本地开发环境编排

### CI/CD
- **GitHub Actions**: 自动化测试和构建
- **脚本**: `scripts/` 目录下的 Bash 脚本

## 🏗️ 架构技术选型理由

### 为什么选择 Go？
- 高性能、低资源消耗
- 优秀的并发支持（Goroutines）
- 丰富的 Kubernetes 生态系统
- 静态编译，部署简单
- 强类型系统，代码健壮

### 为什么选择 React 19？
- 最新特性和性能优化
- 强大的生态系统
- 声明式 UI 范式
- 虚拟 DOM 性能优势
- 丰富的第三方库

### 为什么选择 Gin？
- 高性能（基于 httprouter）
- 简洁的 API
- 中间件支持完善
- 良好的社区支持
- 易于学习和使用

### 为什么选择 GORM？
- 功能丰富的 ORM
- 支持多种数据库
- 自动迁移
- 关联关系处理
- 活跃的社区

### 为什么选择 Vite？
- 极快的冷启动
- HMR（热模块替换）
- 优秀的 TypeScript 支持
- 现代化的构建输出
- 插件生态丰富

### 为什么选择 TailwindCSS？
- 实用优先的 CSS 框架
- 高度可定制
- 优秀的开发体验
- 自动优化（PurgeCSS）
- 一致的设计系统

### 为什么选择 Radix UI？
- 无样式的基础组件
- 完全可访问性（ARIA）
- 键盘导航支持
- 高度可定制
- 与 shadcn/ui 完美集成

## 📊 依赖版本策略

### 主要依赖版本固定
- Go 核心框架：使用精确版本
- React 生态：使用 caret 范围（^）
- UI 库：使用精确版本或 caret 范围

### 定期更新策略
- 安全更新：立即应用
- 功能更新：评估后应用
- 主要版本升级：充分测试后应用

## 🔍 依赖管理最佳实践

### Go 模块
```bash
# 添加依赖
go get github.com/some/package

# 更新依赖
go get -u github.com/some/package

# 清理未使用依赖
go mod tidy

# 验证依赖
go mod verify
```

### pnpm（前端）
```bash
# 安装依赖
pnpm install

# 添加依赖
pnpm add package-name

# 更新依赖
pnpm update

# 审计安全漏洞
pnpm audit
```

## 🚀 性能考虑

### 后端
- **并发处理**: Goroutines 和 Channels
- **数据库连接池**: GORM 自动管理
- **缓存**: LRU 缓存和 Redis
- **日志异步写入**: Logrus 支持

### 前端
- **代码分割**: Vite 自动处理
- **懒加载**: React.lazy 和 Suspense
- **虚拟滚动**: 长列表优化
- **图片优化**: 按需加载

## 🔐 安全依赖

### 后端安全
- JWT 验证（golang-jwt/jwt）
- 密码哈希（bcrypt 通过 golang.org/x/crypto）
- HTTPS 支持（Go 标准库）
- CORS 配置（gin-contrib/cors）

### 前端安全
- XSS 防护（React 自动转义）
- CSRF 防护（Token 验证）
- 安全的第三方库使用
- 定期安全审计

## 📚 技术文档资源

### Go 生态
- [Gin 文档](https://gin-gonic.com/)
- [GORM 文档](https://gorm.io/)
- [Kubernetes Client-Go](https://github.com/kubernetes/client-go)
- [Logrus 文档](https://github.com/sirupsen/logrus)

### React 生态
- [React 19 文档](https://react.dev/)
- [Vite 文档](https://vitejs.dev/)
- [TailwindCSS 文档](https://tailwindcss.com/)
- [Radix UI 文档](https://www.radix-ui.com/)
- [shadcn/ui 文档](https://ui.shadcn.com/)
- [TanStack Query 文档](https://tanstack.com/query/)

### 开发工具
- [Task 文档](https://taskfile.dev/)
- [Swagger/OpenAPI](https://swagger.io/)
