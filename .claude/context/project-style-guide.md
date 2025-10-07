---
created: 2025-10-06T23:34:01+0800
last_updated: 2025-10-06T23:34:01+0800
version: 1.0
author: Claude Code PM System
---

# Tiga 项目风格指南

## 📋 代码风格总则

### 核心原则
1. **可读性优先**: 代码首先是给人读的，其次才是给机器执行的
2. **一致性**: 保持项目内部风格一致
3. **简洁明了**: 避免过度设计和不必要的复杂性
4. **自解释**: 好的命名胜过注释
5. **遵循惯例**: 遵循 Go 和 TypeScript 社区惯例

## 🔷 Go 代码风格

### 1. 命名约定

#### 包名 (Package)
```go
// ✅ 好的包名：小写、简短、有意义
package models
package handlers
package repository

// ❌ 避免
package Models          // 不要大写
package user_service   // 不要下划线
package util           // 过于通用
```

#### 文件名
```go
// ✅ 使用 snake_case
user_handler.go
cluster_manager.go
instance_service.go

// ❌ 避免
UserHandler.go         // 不要 PascalCase
user-handler.go        // 不要 kebab-case
```

#### 类型名
```go
// ✅ 使用 PascalCase，简洁有意义
type User struct { ... }
type ClusterManager struct { ... }
type ServiceManager interface { ... }

// ❌ 避免
type user struct { ... }              // 导出类型应大写
type IServiceManager interface { ... } // 不要 I 前缀
type ServiceManagerInterface interface { ... } // 不要 Interface 后缀
```

#### 函数和方法名
```go
// ✅ 使用 PascalCase（导出）或 camelCase（私有）
func GetUser(id uint) (*User, error) { ... }
func (s *UserService) CreateUser(user *User) error { ... }
func parseConfig(path string) (*Config, error) { ... }

// ❌ 避免
func get_user(id uint) { ... }       // 不要 snake_case
func GetUserById(id uint) { ... }    // 避免冗余（ID 已经说明了 by id）
```

#### 变量名
```go
// ✅ 短小精悍，上下文清晰
user := getUser(id)
for i, v := range items { ... }
db := database.Connect()

// ✅ 缩写要一致
var userID uint        // 不是 userId
var httpClient *http.Client // 不是 HTTPClient
var urlPath string     // 不是 URLPath

// ❌ 避免
userData := getUser(id)    // data 是多余的
userObj := getUser(id)     // obj 是多余的
```

#### 常量名
```go
// ✅ 使用 PascalCase 或 camelCase
const MaxRetries = 3
const defaultTimeout = 30 * time.Second

// ✅ 枚举使用有意义的前缀
const (
    StatusPending  = "pending"
    StatusRunning  = "running"
    StatusComplete = "complete"
)

// ❌ 避免
const MAX_RETRIES = 3      // 不要全大写（除非特殊情况）
const kMaxRetries = 3      // 不要 k 前缀
```

### 2. 代码组织

#### 结构体定义
```go
// ✅ 好的结构体组织
type User struct {
    // 导出字段在前
    ID        uint      `gorm:"primaryKey" json:"id"`
    Username  string    `gorm:"uniqueIndex" json:"username"`
    Email     string    `gorm:"uniqueIndex" json:"email"`
    CreatedAt time.Time `json:"created_at"`

    // 私有字段在后
    passwordHash string
}

// ✅ 相关字段分组，用空行分隔
type Instance struct {
    // 基础信息
    ID   uint   `json:"id"`
    Name string `json:"name"`
    Type string `json:"type"`

    // 连接信息
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Username string `json:"username"`
    Password string `json:"-"` // 敏感信息不序列化

    // 时间戳
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

#### 函数组织
```go
// ✅ 构造函数在前
func NewUserService(repo repository.UserRepository) *UserService {
    return &UserService{repo: repo}
}

// ✅ 公共方法（导出）
func (s *UserService) CreateUser(user *User) error { ... }
func (s *UserService) GetUser(id uint) (*User, error) { ... }
func (s *UserService) UpdateUser(user *User) error { ... }

// ✅ 私有方法（辅助函数）
func (s *UserService) validateUser(user *User) error { ... }
func (s *UserService) hashPassword(password string) (string, error) { ... }
```

### 3. 错误处理

```go
// ✅ 好的错误处理
func GetUser(id uint) (*User, error) {
    user, err := db.Find(id)
    if err != nil {
        return nil, fmt.Errorf("failed to get user %d: %w", id, err)
    }
    return user, nil
}

// ✅ 使用 errors.Is 和 errors.As
if errors.Is(err, ErrNotFound) {
    return nil, fmt.Errorf("user not found")
}

// ❌ 避免忽略错误
db.Find(id)  // 没有检查 err

// ❌ 避免空错误消息
return nil, err  // 应该包装错误提供上下文
```

### 4. 注释规范

```go
// ✅ 包注释（package 前）
// Package handlers provides HTTP request handlers for the API.
package handlers

// ✅ 导出类型/函数注释（简洁，以名字开头）
// User represents a user account in the system.
type User struct { ... }

// GetUser retrieves a user by ID.
// Returns ErrNotFound if the user does not exist.
func GetUser(id uint) (*User, error) { ... }

// ✅ 复杂逻辑注释
func ProcessData(data []byte) error {
    // Step 1: Validate input format
    if err := validate(data); err != nil {
        return err
    }

    // Step 2: Transform data structure
    transformed := transform(data)

    // Step 3: Persist to database
    return save(transformed)
}

// ❌ 避免冗余注释
// GetUser gets user  // 冗余！函数名已经说明了
func GetUser(id uint) (*User, error) { ... }
```

### 5. 代码格式

```go
// ✅ 使用 gofmt 和 goimports
// ✅ 使用 gci 对 import 分组排序

import (
    // 标准库
    "context"
    "fmt"
    "time"

    // 第三方库
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    // 项目内部包
    "github.com/ysicing/tiga/internal/models"
    "github.com/ysicing/tiga/internal/repository"
)
```

## 🔶 TypeScript/React 代码风格

### 1. 命名约定

#### 文件名
```typescript
// ✅ 组件文件使用 kebab-case
cluster-selector.tsx
pod-table.tsx
yaml-editor.tsx

// ✅ 工具文件使用 kebab-case
api-client.ts
utils.ts
types.ts

// ❌ 避免
ClusterSelector.tsx    // 不要 PascalCase（文件名）
cluster_selector.tsx   // 不要 snake_case
```

#### 组件名
```typescript
// ✅ 使用 PascalCase
export function ClusterSelector() { ... }
export const PodTable: React.FC<Props> = ({ ... }) => { ... }

// ❌ 避免
export function clusterSelector() { ... }  // 不要 camelCase
```

#### 变量和函数名
```typescript
// ✅ 使用 camelCase
const userName = 'Alice';
const fetchUserData = async (id: string) => { ... };

// ✅ 布尔值使用 is/has 前缀
const isLoading = true;
const hasPermission = false;

// ✅ 事件处理器使用 handle 前缀
const handleClick = () => { ... };
const handleSubmit = (e: FormEvent) => { ... };

// ❌ 避免
const UserName = 'Alice';              // 不要 PascalCase（变量）
const fetch_user_data = async () => { ... };  // 不要 snake_case
```

#### 类型和接口名
```typescript
// ✅ 使用 PascalCase
interface User { ... }
type ClusterInfo = { ... };

// ✅ Props 类型以 Props 结尾
interface ClusterSelectorProps { ... }
type PodTableProps = { ... };

// ❌ 避免
interface IUser { ... }           // 不要 I 前缀
interface UserInterface { ... }  // 不要 Interface 后缀
```

#### 常量名
```typescript
// ✅ 使用 UPPER_SNAKE_CASE
const MAX_RETRIES = 3;
const API_BASE_URL = 'https://api.example.com';

// ✅ 枚举使用 PascalCase
enum Status {
    Pending = 'pending',
    Running = 'running',
    Complete = 'complete',
}
```

### 2. 组件结构

```typescript
// ✅ 好的组件结构
import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { fetchUsers } from '@/services/api';
import type { User } from '@/types';

// Props 类型定义
interface UserListProps {
    onUserSelect?: (user: User) => void;
    showActions?: boolean;
}

// 组件定义
export function UserList({ onUserSelect, showActions = true }: UserListProps) {
    // Hooks（顺序：useState → useEffect → 自定义 hooks）
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(false);
    const { t } = useTranslation();

    // 副作用
    useEffect(() => {
        loadUsers();
    }, []);

    // 事件处理器
    const loadUsers = async () => {
        setLoading(true);
        try {
            const data = await fetchUsers();
            setUsers(data);
        } catch (error) {
            console.error('Failed to load users:', error);
        } finally {
            setLoading(false);
        }
    };

    const handleUserClick = (user: User) => {
        onUserSelect?.(user);
    };

    // 渲染
    if (loading) {
        return <div>Loading...</div>;
    }

    return (
        <div className="user-list">
            {users.map(user => (
                <div key={user.id} onClick={() => handleUserClick(user)}>
                    {user.name}
                </div>
            ))}
        </div>
    );
}
```

### 3. TypeScript 类型

```typescript
// ✅ 优先使用类型推导
const count = 0;  // 推导为 number
const message = 'Hello';  // 推导为 string

// ✅ 明确需要时添加类型
const users: User[] = [];
const config: Config = { ... };

// ✅ 使用 interface 定义对象类型
interface User {
    id: number;
    name: string;
    email: string;
}

// ✅ 使用 type 定义联合类型、交叉类型
type Status = 'pending' | 'running' | 'complete';
type UserWithRole = User & { role: string };

// ✅ 使用泛型
function identity<T>(arg: T): T {
    return arg;
}

// ❌ 避免 any
const data: any = fetchData();  // 不好
const data: unknown = fetchData();  // 更好，需要类型守卫
```

### 4. React Hooks

```typescript
// ✅ 自定义 Hook 以 use 开头
function useUser(id: string) {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetchUser(id).then(setUser).finally(() => setLoading(false));
    }, [id]);

    return { user, loading };
}

// ✅ 依赖数组明确
useEffect(() => {
    fetchData(id);
}, [id]);  // 明确依赖

// ❌ 避免遗漏依赖
useEffect(() => {
    fetchData(id);
}, []);  // 错误：遗漏了 id 依赖
```

### 5. 注释规范

```typescript
/**
 * ClusterSelector component displays a dropdown for selecting Kubernetes clusters.
 *
 * @param clusters - Array of available clusters
 * @param onClusterChange - Callback when cluster selection changes
 * @returns React component
 *
 * @example
 * ```tsx
 * <ClusterSelector
 *   clusters={clusters}
 *   onClusterChange={(cluster) => console.log(cluster)}
 * />
 * ```
 */
export function ClusterSelector({ clusters, onClusterChange }: Props) {
    // ...
}

// ✅ 复杂逻辑注释
const processData = (data: Data[]) => {
    // Filter out invalid entries
    const validData = data.filter(item => item.isValid);

    // Group by category
    const grouped = validData.reduce((acc, item) => {
        // ...
    }, {});

    return grouped;
};
```

## 🎨 样式约定

### TailwindCSS
```tsx
// ✅ 使用 Tailwind 实用类
<div className="flex items-center gap-2 p-4 rounded-lg bg-white dark:bg-gray-800">
    <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
        Hello
    </span>
</div>

// ✅ 使用 clsx 或 cn 处理条件类名
import { cn } from '@/lib/utils';

<div className={cn(
    "base-class",
    isActive && "active-class",
    isFocused && "focus-class"
)}>
    Content
</div>

// ❌ 避免内联样式（除非动态值）
<div style={{ color: 'red' }}>Text</div>  // 不好
<div className="text-red-500">Text</div>  // 更好
```

## 📝 文档约定

### README.md
- 使用中文或英文（保持一致）
- 包含：项目简介、快速开始、功能特性、部署指南
- 使用 Markdown 格式，清晰的层次结构
- 包含徽章（Badge）和截图

### API 文档
- 使用 Swagger/OpenAPI 注释
- 包含完整的请求/响应示例
- 明确参数说明和错误码

### 代码注释
- 导出的类型、函数必须有注释
- 注释使用英文或中文（保持项目一致）
- 复杂逻辑添加说明性注释

## 🧪 测试约定

### Go 测试
```go
// ✅ 测试文件命名
user_service_test.go

// ✅ 测试函数命名
func TestUserService_CreateUser(t *testing.T) { ... }
func TestUserService_CreateUser_DuplicateEmail(t *testing.T) { ... }

// ✅ 使用表驱动测试
func TestValidateUser(t *testing.T) {
    tests := []struct {
        name    string
        user    *User
        wantErr bool
    }{
        {"valid user", &User{Email: "test@example.com"}, false},
        {"invalid email", &User{Email: "invalid"}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateUser(tt.user)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### React 测试
```typescript
// ✅ 测试文件放在 __tests__ 目录
__tests__/cluster-selector.test.tsx

// ✅ 测试描述清晰
describe('ClusterSelector', () => {
    it('renders cluster list', () => { ... });
    it('calls onClusterChange when cluster is selected', () => { ... });
    it('shows loading state', () => { ... });
});
```

## 🔧 工具配置

### Go
- **Linter**: golangci-lint
- **Formatter**: gofmt + goimports + gci
- **测试**: go test

### TypeScript
- **Linter**: ESLint
- **Formatter**: Prettier
- **类型检查**: TypeScript compiler

## 📌 最佳实践

### 通用
1. ✅ 保持函数简短（< 50 行）
2. ✅ 避免深层嵌套（< 3 层）
3. ✅ DRY 原则（Don't Repeat Yourself）
4. ✅ KISS 原则（Keep It Simple, Stupid）
5. ✅ 单一职责原则

### Go 特定
1. ✅ 接收错误，返回错误
2. ✅ 使用 context 传递取消信号
3. ✅ 避免全局变量
4. ✅ 使用 defer 清理资源

### React 特定
1. ✅ 保持组件职责单一
2. ✅ 提取自定义 Hooks 复用逻辑
3. ✅ 使用 memo 优化性能
4. ✅ 避免 props drilling，使用 Context

---

**风格指南版本**: 1.0
**最后更新**: 2025-10-06
**适用范围**: Tiga 项目全体代码
