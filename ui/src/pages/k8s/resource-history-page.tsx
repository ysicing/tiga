import { useState } from 'react'
import {
  ResourceHistory,
  ResourceHistoryFilters,
  useResourceHistory,
} from '@/services/k8s-api'
import {
  IconCheck,
  IconFilter,
  IconRefresh,
  IconX,
} from '@tabler/icons-react'
import { format } from 'date-fns'
import { useNavigate, useParams } from 'react-router-dom'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const operationTypeColors: Record<
  string,
  'default' | 'secondary' | 'destructive' | 'outline'
> = {
  create: 'default',
  update: 'secondary',
  delete: 'destructive',
  apply: 'outline',
}

const operationTypeLabels: Record<string, string> = {
  create: '创建',
  update: '更新',
  delete: '删除',
  apply: '应用',
}

export function ResourceHistoryPage() {
  const navigate = useNavigate()
  const { clusterId } = useParams<{ clusterId: string }>()

  const [filters, setFilters] = useState<ResourceHistoryFilters>({
    page: 1,
    page_size: 20,
  })

  const { data, isLoading, error, refetch } = useResourceHistory(
    clusterId || '',
    filters
  )

  const histories = Array.isArray(data?.data) ? data.data : []
  const total = data?.total || 0
  const currentPage = data?.page || 1
  const pageSize = data?.page_size || 20
  const totalPages = Math.ceil(total / pageSize)

  const handleFilterChange = (key: keyof ResourceHistoryFilters, value: any) => {
    setFilters((prev) => ({
      ...prev,
      [key]: value,
      page: 1, // Reset to first page when filters change
    }))
  }

  const clearFilters = () => {
    setFilters({ page: 1, page_size: 20 })
  }

  const hasActiveFilters = Object.keys(filters).some(
    (key) =>
      key !== 'page' &&
      key !== 'page_size' &&
      filters[key as keyof ResourceHistoryFilters]
  )

  if (!clusterId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">缺少集群 ID</CardTitle>
          <CardDescription>请从集群列表选择一个集群</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-10 w-32" />
        </div>
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-32" />
          </CardHeader>
          <CardContent className="space-y-4">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">加载失败</CardTitle>
          <CardDescription>
            {(error as any)?.message || '无法加载资源历史'}
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">资源操作历史</h1>
          <p className="text-muted-foreground mt-2">
            查看和管理 Kubernetes 资源的所有操作记录
          </p>
        </div>
        <Button onClick={() => refetch()} variant="outline">
          <IconRefresh className="w-4 h-4 mr-2" />
          刷新
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <IconFilter className="w-5 h-5" />
              过滤条件
            </CardTitle>
            {hasActiveFilters && (
              <Button
                variant="ghost"
                size="sm"
                onClick={clearFilters}
              >
                <IconX className="w-4 h-4 mr-1" />
                清除过滤
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4">
            <Input
              placeholder="资源类型 (如: deployment)"
              value={filters.resource_type || ''}
              onChange={(e) =>
                handleFilterChange('resource_type', e.target.value)
              }
            />
            <Input
              placeholder="资源名称"
              value={filters.resource_name || ''}
              onChange={(e) =>
                handleFilterChange('resource_name', e.target.value)
              }
            />
            <Input
              placeholder="命名空间"
              value={filters.namespace || ''}
              onChange={(e) => handleFilterChange('namespace', e.target.value)}
            />
            <Select
              value={filters.operation_type || 'all'}
              onValueChange={(value) =>
                handleFilterChange(
                  'operation_type',
                  value === 'all' ? undefined : value
                )
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="操作类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部类型</SelectItem>
                <SelectItem value="create">创建</SelectItem>
                <SelectItem value="update">更新</SelectItem>
                <SelectItem value="delete">删除</SelectItem>
                <SelectItem value="apply">应用</SelectItem>
              </SelectContent>
            </Select>
            <Select
              value={
                filters.success === undefined
                  ? 'all'
                  : filters.success
                    ? 'success'
                    : 'failed'
              }
              onValueChange={(value) =>
                handleFilterChange(
                  'success',
                  value === 'all'
                    ? undefined
                    : value === 'success'
                )
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="success">成功</SelectItem>
                <SelectItem value="failed">失败</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Results */}
      <Card>
        <CardHeader>
          <CardTitle>
            操作记录
            <span className="text-muted-foreground text-sm font-normal ml-2">
              (共 {total} 条)
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {histories.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              暂无操作记录
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>资源</TableHead>
                    <TableHead>操作类型</TableHead>
                    <TableHead>操作者</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>时间</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {histories.map((history: ResourceHistory) => (
                    <TableRow key={history.id}>
                      <TableCell>
                        <div>
                          <div className="font-medium">
                            {history.resource_type}/{history.resource_name}
                          </div>
                          {history.namespace && (
                            <div className="text-xs text-muted-foreground">
                              命名空间: {history.namespace}
                            </div>
                          )}
                          {history.api_group && (
                            <div className="text-xs text-muted-foreground">
                              API: {history.api_group}/{history.api_version}
                            </div>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            operationTypeColors[history.operation_type] ||
                            'secondary'
                          }
                        >
                          {operationTypeLabels[history.operation_type] ||
                            history.operation_type}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div>
                          <div className="font-medium">
                            {history.operator_name}
                          </div>
                          <div className="text-xs text-muted-foreground font-mono">
                            {history.operator_id.substring(0, 8)}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        {history.success ? (
                          <Badge className="bg-green-500">
                            <IconCheck className="w-3 h-3 mr-1" /> 成功
                          </Badge>
                        ) : (
                          <Badge variant="destructive">
                            <IconX className="w-3 h-3 mr-1" /> 失败
                          </Badge>
                        )}
                        {history.error_message && (
                          <div className="text-xs text-destructive mt-1">
                            {history.error_message}
                          </div>
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="text-sm">
                          {format(
                            new Date(history.created_at),
                            'yyyy-MM-dd HH:mm:ss'
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            navigate(
                              `/k8s/clusters/${clusterId}/resource-history/${history.id}`
                            )
                          }
                        >
                          查看详情
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4">
                  <div className="text-sm text-muted-foreground">
                    第 {currentPage} 页，共 {totalPages} 页
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={currentPage === 1}
                      onClick={() =>
                        setFilters((prev) => ({
                          ...prev,
                          page: currentPage - 1,
                        }))
                      }
                    >
                      上一页
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={currentPage === totalPages}
                      onClick={() =>
                        setFilters((prev) => ({
                          ...prev,
                          page: currentPage + 1,
                        }))
                      }
                    >
                      下一页
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
