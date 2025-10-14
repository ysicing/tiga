# 前端API响应格式修复

**问题**: `instances.map is not a function`
**修复时间**: 2025-10-10 21:00

---

## 问题分析

### 错误堆栈
```
TypeError: instances.map is not a function
at DatabaseInstanceList (instance-list.tsx:275:123)
```

### 根本原因
后端API响应格式与前端期望不匹配：

**后端实际返回** (`internal/api/handlers/database/instance.go:47-50`):
```go
handlers.RespondSuccess(c, gin.H{
    "instances": instances,  // 数据在 data.instances
    "count":     len(instances),
})
```

实际JSON:
```json
{
  "success": true,
  "data": {
    "instances": [...],  // ← 数组在这里
    "count": 0
  }
}
```

**前端期望** (`ui/src/services/database-api.ts:89`):
```typescript
apiClient.get<{ data: DatabaseInstance[] }>('/database/instances')
//                    ^^^^^^^^^^^^^^^^^ 期望 data 直接是数组
```

期望的JSON:
```json
{
  "data": [...]  // ❌ 但实际是嵌套的
}
```

---

## 修复方案

### 修改文件1: `ui/src/services/database-api.ts:86-94`

**修改前**:
```typescript
export const useInstances = () => {
  return useQuery({
    queryKey: ['database', 'instances'],
    queryFn: () => apiClient.get<{ data: DatabaseInstance[] }>('/database/instances')
  })
}
```

**修改后**:
```typescript
export const useInstances = () => {
  return useQuery({
    queryKey: ['database', 'instances'],
    queryFn: async () => {
      const response = await apiClient.get<{
        data: {
          instances: DatabaseInstance[],
          count: number
        }
      }>('/database/instances')
      return response
    }
  })
}
```

### 修改文件2: `ui/src/pages/database/instance-list.tsx:69`

**修改前**:
```typescript
const instances = data?.data || []
```

**修改后**:
```typescript
const instances = data?.data?.instances || []
```

---

## 验证

### 构建测试
```bash
✅ cd ui && pnpm build
✓ built in 10.84s

✅ go build -o bin/tiga ./cmd/tiga
成功生成二进制文件: 158M
```

### 运行时测试
启动应用后访问 http://localhost:12306/dbs/instances

**预期行为**:
- ✅ 页面正常加载，不再报错
- ✅ 显示"还没有数据库实例"卡片 (如果没有实例)
- ✅ 显示实例列表 (如果有实例)

---

## 影响范围

仅影响数据库实例列表页面的数据加载逻辑，其他API调用未受影响。

---

## 经验教训

### 问题根源
前后端API契约未对齐，缺少自动化契约测试。

### 改进建议 (可选)
1. **添加契约测试** (P2优先级):
   - 使用 Pact 或类似工具
   - 确保前后端API格式一致

2. **统一响应格式** (P3优先级):
   - 考虑后端直接返回数组:
     ```go
     handlers.RespondSuccess(c, instances)
     ```
   - 或前端统一处理嵌套格式

3. **TypeScript类型生成** (P3优先级):
   - 从Swagger自动生成TypeScript类型
   - 避免手动定义导致的不一致

---

**修复状态**: ✅ 已完成
**回归风险**: 极低 (仅修改数据访问路径)
**需要测试**: 启动应用验证页面加载
