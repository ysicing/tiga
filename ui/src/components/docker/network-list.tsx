import { useState } from 'react'
import {
  Network,
  useNetworks,
  useDeleteNetwork,
} from '@/services/docker-api'
import {
  IconTrash,
  IconRefresh,
  IconSearch,
  IconNetwork,
} from '@tabler/icons-react'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
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
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

interface NetworkListProps {
  instanceId: string
}

const getScopeBadge = (scope: string) => {
  switch (scope.toLowerCase()) {
    case 'local':
      return <Badge variant="secondary">本地</Badge>
    case 'swarm':
      return <Badge className="bg-blue-500">Swarm</Badge>
    case 'global':
      return <Badge className="bg-purple-500">全局</Badge>
    default:
      return <Badge variant="outline">{scope}</Badge>
  }
}

const formatContainers = (containers?: Record<string, any>) => {
  if (!containers) return 0
  return Object.keys(containers).length
}

const formatSubnet = (network: Network) => {
  if (!network.ipam || !network.ipam.config || network.ipam.config.length === 0) {
    return '-'
  }
  return network.ipam.config.map((c) => c.subnet).filter(Boolean).join(', ') || '-'
}

export function NetworkList({ instanceId }: NetworkListProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<{
    id: string
    name: string
  } | null>(null)

  const { data, isLoading, error, refetch } = useNetworks(instanceId)
  const deleteMutation = useDeleteNetwork()

  const networks = data?.data?.networks || []
  const filteredNetworks = networks.filter((network) => {
    const name = network.name.toLowerCase()
    const driver = network.driver.toLowerCase()
    const search = searchTerm.toLowerCase()
    return name.includes(search) || driver.includes(search)
  })

  const handleDelete = async () => {
    if (!deleteTarget) return

    try {
      await deleteMutation.mutateAsync({
        instanceId,
        networkId: deleteTarget.id,
      })
      toast.success(`网络 ${deleteTarget.name} 已删除`)
      setDeleteTarget(null)
      refetch()
    } catch (error: any) {
      toast.error(error?.message || '无法删除网络')
    }
  }

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>网络列表</CardTitle>
              <CardDescription className="mt-1 flex items-center gap-2">
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                正在加载网络列表...
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[200px]">名称</TableHead>
                  <TableHead>驱动</TableHead>
                  <TableHead className="w-[100px]">范围</TableHead>
                  <TableHead>子网</TableHead>
                  <TableHead className="w-[100px]">容器数</TableHead>
                  <TableHead className="w-[150px]">属性</TableHead>
                  <TableHead className="text-right w-[80px]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[1, 2, 3, 4, 5].map((i) => (
                  <TableRow key={i}>
                    <TableCell>
                      <Skeleton className="h-4 w-32" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-4 w-16" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-6 w-16 rounded-full" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-4 w-28" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-4 w-8" />
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Skeleton className="h-5 w-16 rounded-full" />
                        <Skeleton className="h-5 w-16 rounded-full" />
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Skeleton className="h-8 w-8 rounded-md" />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">加载失败</CardTitle>
          <CardDescription>
            {(error as any)?.message || '无法加载网络列表'}
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>网络列表</CardTitle>
              <CardDescription className="mt-1">
                共 {networks.length} 个网络
                {filteredNetworks.length !== networks.length &&
                  ` (显示 ${filteredNetworks.length} 个)`}
              </CardDescription>
            </div>
            <div className="flex gap-2">
              <div className="relative">
                <IconSearch className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="搜索网络或驱动..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9 w-64"
                />
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => refetch()}
                disabled={isLoading}
              >
                <IconRefresh className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {filteredNetworks.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              {searchTerm ? '没有匹配的网络' : '还没有网络'}
            </div>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[200px]">名称</TableHead>
                    <TableHead>驱动</TableHead>
                    <TableHead className="w-[100px]">范围</TableHead>
                    <TableHead>子网</TableHead>
                    <TableHead className="w-[100px]">容器数</TableHead>
                    <TableHead className="w-[150px]">属性</TableHead>
                    <TableHead className="text-right w-[80px]">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredNetworks.map((network) => {
                    const containerCount = formatContainers(network.containers)
                    const isSystemNetwork = ['bridge', 'host', 'none'].includes(
                      network.name
                    )

                    return (
                      <TableRow key={network.id}>
                        <TableCell className="font-medium">
                          <div className="flex items-center gap-2">
                            <IconNetwork className="w-4 h-4 text-muted-foreground" />
                            <div>
                              <div className="font-mono text-sm">{network.name}</div>
                              <div className="text-xs text-muted-foreground truncate max-w-[180px]">
                                {network.id.substring(0, 12)}
                              </div>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="font-mono text-xs">{network.driver}</div>
                        </TableCell>
                        <TableCell>{getScopeBadge(network.scope)}</TableCell>
                        <TableCell className="font-mono text-xs">
                          {formatSubnet(network)}
                        </TableCell>
                        <TableCell className="text-center">
                          {containerCount > 0 ? (
                            <Badge variant="outline">{containerCount}</Badge>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-1">
                            {network.internal && (
                              <Badge variant="outline" className="text-xs">
                                内部
                              </Badge>
                            )}
                            {network.attachable && (
                              <Badge variant="outline" className="text-xs">
                                可附加
                              </Badge>
                            )}
                            {network.ingress && (
                              <Badge variant="outline" className="text-xs">
                                入口
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-8 w-8"
                              onClick={() =>
                                setDeleteTarget({
                                  id: network.id,
                                  name: network.name,
                                })
                              }
                              disabled={
                                isSystemNetwork ||
                                containerCount > 0 ||
                                deleteMutation.isPending
                              }
                              title={
                                isSystemNetwork
                                  ? '系统网络无法删除'
                                  : containerCount > 0
                                    ? '请先断开所有容器'
                                    : '删除网络'
                              }
                            >
                              <IconTrash className="w-4 h-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除网络？</AlertDialogTitle>
            <AlertDialogDescription>
              即将删除网络 <strong>{deleteTarget?.name}</strong>，此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
