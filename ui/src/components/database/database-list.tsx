import { useState } from 'react'
import {
  useCreateDatabase,
  useDatabases,
  useDeleteDatabase,
} from '@/services/database-api'
import { IconAlertCircle, IconPlus, IconTrash } from '@tabler/icons-react'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

interface DatabaseListProps {
  instanceId: number
  instanceType: string
}

export function DatabaseList({ instanceId, instanceType }: DatabaseListProps) {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [isDeleteOpen, setIsDeleteOpen] = useState(false)
  const [selectedDb, setSelectedDb] = useState<any>(null)
  const [confirmName, setConfirmName] = useState('')

  const { data, isLoading } = useDatabases(instanceId)
  const createMutation = useCreateDatabase()
  const deleteMutation = useDeleteDatabase()

  const databases = data?.data || []

  const [formData, setFormData] = useState({
    name: '',
    charset: instanceType === 'mysql' ? 'utf8mb4' : 'UTF8',
    collation: instanceType === 'mysql' ? 'utf8mb4_unicode_ci' : '',
    owner: instanceType === 'postgresql' ? 'postgres' : '',
  })

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await createMutation.mutateAsync({ instanceId, data: formData })
      toast.success(`数据库 ${formData.name} 已创建`)
      setIsCreateOpen(false)
      setFormData({
        name: '',
        charset: instanceType === 'mysql' ? 'utf8mb4' : 'UTF8',
        collation: instanceType === 'mysql' ? 'utf8mb4_unicode_ci' : '',
        owner: instanceType === 'postgresql' ? 'postgres' : '',
      })
    } catch (error: any) {
      toast.error(error?.response?.data?.error || '无法创建数据库')
    }
  }

  const handleDelete = async () => {
    if (!selectedDb || confirmName !== selectedDb.name) {
      toast.error('请正确输入数据库名称确认删除')
      return
    }

    try {
      await deleteMutation.mutateAsync({ id: selectedDb.id, confirmName })
      toast.success(`数据库 ${selectedDb.name} 已删除`)
      setIsDeleteOpen(false)
      setSelectedDb(null)
      setConfirmName('')
    } catch (error: any) {
      toast.error(error?.response?.data?.error || '无法删除数据库')
    }
  }

  if (isLoading) {
    return <Skeleton className="h-64" />
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">数据库列表</h3>
        <Button size="sm" onClick={() => setIsCreateOpen(true)}>
          <IconPlus className="w-4 h-4 mr-2" />
          新建数据库
        </Button>
      </div>

      {databases.length === 0 ? (
        <Alert>
          <IconAlertCircle className="h-4 w-4" />
          <AlertDescription>
            暂无数据库，点击"新建数据库"开始添加
          </AlertDescription>
        </Alert>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              {instanceType !== 'redis' && <TableHead>字符集</TableHead>}
              {instanceType === 'mysql' && <TableHead>排序规则</TableHead>}
              {instanceType === 'postgresql' && <TableHead>所有者</TableHead>}
              {instanceType === 'redis' && <TableHead>DB编号</TableHead>}
              {instanceType === 'redis' && <TableHead>Key数量</TableHead>}
              <TableHead>创建时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {databases.map((db) => (
              <TableRow key={db.id}>
                <TableCell className="font-medium">{db.name}</TableCell>
                {instanceType !== 'redis' && (
                  <TableCell>{db.charset}</TableCell>
                )}
                {instanceType === 'mysql' && (
                  <TableCell>{db.collation}</TableCell>
                )}
                {instanceType === 'postgresql' && (
                  <TableCell>{db.owner}</TableCell>
                )}
                {instanceType === 'redis' && (
                  <TableCell>{db.db_number}</TableCell>
                )}
                {instanceType === 'redis' && (
                  <TableCell>{db.key_count || 0}</TableCell>
                )}
                <TableCell>
                  {new Date(db.created_at).toLocaleString()}
                </TableCell>
                <TableCell className="text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setSelectedDb(db)
                      setIsDeleteOpen(true)
                    }}
                  >
                    <IconTrash className="w-4 h-4 text-destructive" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* Create Dialog */}
      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent>
          <form onSubmit={handleCreate}>
            <DialogHeader>
              <DialogTitle>新建数据库</DialogTitle>
              <DialogDescription>创建一个新的数据库</DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="name">数据库名称 *</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  required
                />
              </div>
              {instanceType !== 'redis' && (
                <div className="space-y-2">
                  <Label htmlFor="charset">字符集</Label>
                  <Input
                    id="charset"
                    value={formData.charset}
                    onChange={(e) =>
                      setFormData({ ...formData, charset: e.target.value })
                    }
                  />
                </div>
              )}
              {instanceType === 'mysql' && (
                <div className="space-y-2">
                  <Label htmlFor="collation">排序规则</Label>
                  <Input
                    id="collation"
                    value={formData.collation}
                    onChange={(e) =>
                      setFormData({ ...formData, collation: e.target.value })
                    }
                  />
                </div>
              )}
              {instanceType === 'postgresql' && (
                <div className="space-y-2">
                  <Label htmlFor="owner">所有者</Label>
                  <Input
                    id="owner"
                    value={formData.owner}
                    onChange={(e) =>
                      setFormData({ ...formData, owner: e.target.value })
                    }
                  />
                </div>
              )}
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsCreateOpen(false)}
              >
                取消
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? '创建中...' : '创建'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除数据库</DialogTitle>
            <DialogDescription>
              此操作将永久删除数据库 <strong>{selectedDb?.name}</strong>。
              请输入数据库名称确认删除：
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input
              value={confirmName}
              onChange={(e) => setConfirmName(e.target.value)}
              placeholder={selectedDb?.name}
            />
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsDeleteOpen(false)
                setSelectedDb(null)
                setConfirmName('')
              }}
            >
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={
                deleteMutation.isPending || confirmName !== selectedDb?.name
              }
            >
              {deleteMutation.isPending ? '删除中...' : '删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
