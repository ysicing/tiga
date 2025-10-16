import { useInstance } from '@/services/database-api'
import {
  IconArrowLeft,
  IconDatabase,
  IconKey,
  IconTerminal,
  IconUsers,
} from '@tabler/icons-react'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DatabaseList } from '@/components/database/database-list'
import { PermissionList } from '@/components/database/permission-list'
import { QueryConsole } from '@/components/database/query-console'
import { UserList } from '@/components/database/user-list'

export function InstanceDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const instanceId = parseInt(id || '0')

  const { data, isLoading, error } = useInstance(instanceId)
  const instance = data?.data

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-20" />
        <Skeleton className="h-96" />
      </div>
    )
  }

  if (error || !instance) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" onClick={() => navigate('/dbs/instances')}>
          <IconArrowLeft className="w-4 h-4 mr-2" />
          返回
        </Button>
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive">加载失败</CardTitle>
            <CardDescription>无法加载实例详情</CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate('/dbs/instances')}>
            <IconArrowLeft className="w-4 h-4 mr-2" />
            返回
          </Button>
          <div>
            <h1 className="text-3xl font-bold">{instance.name}</h1>
            <p className="text-muted-foreground mt-1">
              {instance.type.toUpperCase()} • {instance.host}:{instance.port}
            </p>
          </div>
        </div>
        <Badge
          className={
            instance.status === 'online'
              ? 'bg-green-500'
              : instance.status === 'offline'
                ? 'bg-gray-500'
                : 'bg-yellow-500'
          }
        >
          {instance.status}
        </Badge>
      </div>

      {/* Instance Info Card */}
      <Card>
        <CardHeader>
          <CardTitle>实例信息</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <div className="text-sm text-muted-foreground">类型</div>
              <div className="font-medium">{instance.type.toUpperCase()}</div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">版本</div>
              <div className="font-medium">{instance.version || 'N/A'}</div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">用户名</div>
              <div className="font-medium">{instance.username}</div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">SSL 模式</div>
              <div className="font-medium">
                {instance.ssl_mode || 'disable'}
              </div>
            </div>
          </div>
          {instance.description && (
            <div className="mt-4">
              <div className="text-sm text-muted-foreground">描述</div>
              <div className="mt-1">{instance.description}</div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Tabs */}
      <Tabs defaultValue="databases" className="space-y-4">
        <TabsList>
          <TabsTrigger value="databases">
            <IconDatabase className="w-4 h-4 mr-2" />
            数据库
          </TabsTrigger>
          <TabsTrigger value="users">
            <IconUsers className="w-4 h-4 mr-2" />
            用户
          </TabsTrigger>
          <TabsTrigger value="permissions">
            <IconKey className="w-4 h-4 mr-2" />
            权限
          </TabsTrigger>
          <TabsTrigger value="query">
            <IconTerminal className="w-4 h-4 mr-2" />
            查询控制台
          </TabsTrigger>
        </TabsList>

        <TabsContent value="databases">
          <DatabaseList instanceId={instanceId} instanceType={instance.type} />
        </TabsContent>

        <TabsContent value="users">
          <UserList instanceId={instanceId} />
        </TabsContent>

        <TabsContent value="permissions">
          <PermissionList instanceId={instanceId} />
        </TabsContent>

        <TabsContent value="query">
          <QueryConsole instanceId={instanceId} instanceType={instance.type} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
