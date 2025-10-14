# 新建数据库实例404错误修复报告

**问题**: 点击"新建实例"按钮报404错误
**发现时间**: 2025-10-11
**修复状态**: ✅ 已完成

---

## 问题描述

用户在数据库实例列表页面点击"新建实例"按钮时,浏览器显示:

```
Unexpected Application Error!
404 Not Found
```

---

## 根因分析

### 问题1: 路由路径不匹配

**导航路径** (`instance-list.tsx:130`):
```typescript
<Button onClick={() => navigate('/database/instances/new')}>
```

**实际路由配置** (`routes.tsx:210-249`):
```typescript
{
  path: '/dbs',  // ← 路径是 /dbs 而非 /database
  children: [
    {
      path: 'instances',
      element: <DatabaseInstanceList />,
    },
    {
      path: 'instances/:id',
      element: <InstanceDetail />,
    },
    // ❌ 缺少 'instances/new' 路由
  ],
}
```

**错误**:
1. 导航路径使用 `/database/instances/new`
2. 路由配置中实际路径是 `/dbs/instances`
3. 缺少 `/dbs/instances/new` 路由配置

### 问题2: 缺少表单页面组件

项目中不存在新建实例的表单页面:

```bash
$ find ui/src/pages/database -name "*form*"
(无结果)
```

只有两个页面:
- `instance-list.tsx` - 实例列表
- `instance-detail.tsx` - 实例详情

---

## 修复方案

### 修复1: 创建实例表单页面

**新建文件**: `ui/src/pages/database/instance-form.tsx` (265行)

**核心功能**:

1. **表单验证** (使用 Zod + React Hook Form):
```typescript
const instanceFormSchema = z.object({
  name: z.string().min(1).max(100),
  type: z.enum(['mysql', 'postgresql', 'redis']),
  host: z.string().min(1),
  port: z.number().min(1).max(65535),
  username: z.string().optional(),
  password: z.string().optional(),
  ssl_mode: z.string().optional(),
  description: z.string().optional(),
})
```

2. **智能默认值** (根据数据库类型):
```typescript
const handleTypeChange = (type: 'mysql' | 'postgresql' | 'redis') => {
  const defaultPorts = {
    mysql: 3306,
    postgresql: 5432,
    redis: 6379,
  }
  form.setValue('port', defaultPorts[type])
}
```

3. **条件渲染**:
- Redis: 隐藏用户名字段
- PostgreSQL: 显示SSL模式选择
- 密码显示/隐藏切换

4. **表单提交**:
```typescript
const onSubmit = async (values: InstanceFormValues) => {
  try {
    await createMutation.mutateAsync(values)
    toast.success('数据库实例创建成功')
    navigate('/dbs/instances')  // 返回列表
  } catch (error: any) {
    toast.error(error?.response?.data?.error || '创建实例失败')
  }
}
```

### 修复2: 更新路由配置

**文件**: `ui/src/routes.tsx`

**导入表单组件** (routes.tsx:48):
```typescript
import { InstanceForm } from './pages/database/instance-form'
```

**添加路由** (routes.tsx:229-232):
```typescript
{
  path: 'instances/new',
  element: <InstanceForm />,
},
```

**完整路由层级**:
```
/dbs
  /instances          → <DatabaseInstanceList />
  /instances/new      → <InstanceForm />        ← 新增
  /instances/:id      → <InstanceDetail />
```

**注意**: `instances/new` 必须在 `instances/:id` 之前,避免被`:id`路由匹配

### 修复3: 统一导航路径

**修复文件1**: `instance-list.tsx:130`

**Before**:
```typescript
<Button onClick={() => navigate('/database/instances/new')}>
```

**After**:
```typescript
<Button onClick={() => navigate('/dbs/instances/new')}>
```

**修复文件2**: `instance-detail.tsx:34, 53`

修复返回按钮导航路径 (共2处):

**Before**:
```typescript
onClick={() => navigate('/database/instances')}
```

**After**:
```typescript
onClick={() => navigate('/dbs/instances')}
```

---

## 表单UI设计

### 布局结构

```
┌─────────────────────────────────────────┐
│ ← 返回    新建数据库实例                │
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ 实例配置                            │ │
│ │                                     │ │
│ │ [实例名称*]      [数据库类型*]      │ │
│ │ [主机地址*]      [端口号*]          │ │
│ │ [用户名]         [密码]             │ │
│ │ [SSL模式]        (仅PostgreSQL)     │ │
│ │ [描述]                              │ │
│ │                                     │ │
│ │              [取消] [创建实例]      │ │
│ └─────────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

### 字段说明

| 字段 | 必填 | 类型 | 说明 |
|------|------|------|------|
| 实例名称 | ✅ | 文本 | 1-100字符 |
| 数据库类型 | ✅ | 下拉 | MySQL/PostgreSQL/Redis |
| 主机地址 | ✅ | 文本 | IP或域名 |
| 端口号 | ✅ | 数字 | 1-65535 |
| 用户名 | ❌ | 文本 | Redis不显示 |
| 密码 | ❌ | 密码 | 支持显示/隐藏 |
| SSL模式 | ❌ | 下拉 | 仅PostgreSQL |
| 描述 | ❌ | 文本域 | 可选备注 |

---

## 验证测试

### 编译测试

```bash
✅ cd ui && pnpm build
✓ built in 10.44s

✅ go build -o bin/tiga ./cmd/tiga
成功生成二进制文件: 163M
```

### 功能验证 (待运行时测试)

**测试场景1**: 访问新建实例页面
- [ ] 访问 `/dbs/instances`
- [ ] 点击"新建实例"按钮
- [ ] 应跳转到 `/dbs/instances/new`
- [ ] 显示新建实例表单

**测试场景2**: 表单交互
- [ ] 选择MySQL类型,端口自动填充3306
- [ ] 选择PostgreSQL类型,端口自动填充5432,显示SSL模式
- [ ] 选择Redis类型,端口自动填充6379,隐藏用户名
- [ ] 密码字段支持显示/隐藏切换

**测试场景3**: 表单验证
- [ ] 实例名称为空时显示错误
- [ ] 端口号超出范围时显示错误
- [ ] 必填字段未填时无法提交

**测试场景4**: 提交流程
- [ ] 填写有效数据提交
- [ ] 显示"创建中..."加载状态
- [ ] 成功后显示 toast 提示
- [ ] 自动跳转回 `/dbs/instances` 列表
- [ ] 列表显示新创建的实例

**测试场景5**: 错误处理
- [ ] 后端返回错误时显示错误 toast
- [ ] 点击"取消"按钮返回列表

---

## 修改的文件

### 新增文件 (1个)

1. ✅ `ui/src/pages/database/instance-form.tsx` (265行)
   - 完整的实例创建表单
   - Zod schema验证
   - React Hook Form集成
   - 智能默认值

### 修改文件 (3个)

2. ✅ `ui/src/routes.tsx`
   - 导入 `InstanceForm` 组件
   - 添加 `/dbs/instances/new` 路由

3. ✅ `ui/src/pages/database/instance-list.tsx`
   - 修复导航路径: `/database/instances/new` → `/dbs/instances/new`

4. ✅ `ui/src/pages/database/instance-detail.tsx`
   - 修复返回按钮路径: `/database/instances` → `/dbs/instances` (2处)

---

## TypeScript错误修复

### 错误1: Zod enum配置

**错误**:
```typescript
type: z.enum(['mysql', 'postgresql', 'redis'], { required_error: '...' })
//                                              ^^^^^^^^^^^^^^^^^^^^^^^^ 不支持
```

**修复**:
```typescript
type: z.enum(['mysql', 'postgresql', 'redis'])
```

### 错误2: 端口号类型

**错误**:
```typescript
port: z.coerce.number()  // coerce导致类型推断问题
```

**修复**:
```typescript
port: z.number()  // 直接使用number类型
```

表单输入已设置 `type="number"`,自动转换为数字。

---

## 路由优先级说明

**路由顺序很重要**:

```typescript
// ✅ 正确顺序
{
  path: 'instances/new',    // 1. 精确匹配优先
  element: <InstanceForm />,
},
{
  path: 'instances/:id',    // 2. 参数匹配其次
  element: <InstanceDetail />,
}

// ❌ 错误顺序
{
  path: 'instances/:id',    // :id 会匹配 "new"
  element: <InstanceDetail />,
},
{
  path: 'instances/new',    // 永远不会被匹配
  element: <InstanceForm />,
}
```

**原因**: React Router按顺序匹配路由,`:id`会匹配任何值包括"new"。

---

## API集成

表单使用已有的 `useCreateInstance()` hook:

**API调用** (`database-api.ts:104-114`):
```typescript
export const useCreateInstance = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: Partial<DatabaseInstance>) =>
      apiClient.post<{ data: DatabaseInstance }>('/database/instances', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['database', 'instances'] })
    }
  })
}
```

**后端端点**: `POST /api/v1/database/instances`

**请求体示例**:
```json
{
  "name": "生产环境MySQL",
  "type": "mysql",
  "host": "localhost",
  "port": 3306,
  "username": "root",
  "password": "secret",
  "description": "主数据库实例"
}
```

---

## 用户体验改进

### Before (404错误)

```
用户点击"新建实例"
    ↓
导航到 /database/instances/new
    ↓
路由不匹配
    ↓
❌ 显示: "404 Not Found"
```

### After (正常流程)

```
用户点击"新建实例"
    ↓
导航到 /dbs/instances/new
    ↓
显示实例创建表单
    ↓
用户填写信息
    ↓
选择数据库类型
    ↓
端口号自动填充
    ↓
提交表单
    ↓
调用 POST /api/v1/database/instances
    ↓
✅ 成功: toast提示 + 跳转回列表
❌ 失败: toast显示错误信息
```

---

## 回归风险评估

**风险等级**: 🟢 低

**理由**:
1. ✅ 仅新增路由和页面,不影响现有功能
2. ✅ 路径修复不破坏其他导航
3. ✅ 编译通过,无TypeScript错误
4. ✅ 使用已有API,不涉及后端修改

**影响范围**:
- 仅影响数据库实例管理模块
- 其他模块(K8s、MinIO、VMs等)不受影响

**建议验证**:
- [ ] 新建实例流程端到端测试
- [ ] 实例列表和详情页导航测试
- [ ] 不同数据库类型表单测试

---

## 后续优化建议

### P3优先级

1. **连接测试功能**:
   - 添加"测试连接"按钮
   - 在提交前验证数据库可达性
   - 自动检测数据库版本

2. **表单预填充**:
   - 支持复制现有实例配置
   - URL参数预填充字段

3. **高级配置**:
   - 连接池设置
   - 超时时间配置
   - 自定义连接参数

---

## 总结

**修复状态**: ✅ 已完成
**问题根因**: 路由路径不一致 + 缺少表单组件
**解决方案**: 创建表单页面 + 统一路由路径
**影响范围**: 仅数据库实例管理模块
**回归风险**: 低
**下一步**: 运行时验证新建实例流程

---

**修复人**: Claude Code (Sonnet 4.5)
**修复时间**: 2025-10-11
**验证状态**: 编译通过,待运行时测试
