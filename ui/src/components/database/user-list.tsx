import { useDatabaseUsers } from '@/services/database-api'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { IconAlertCircle } from '@tabler/icons-react'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

export function UserList({ instanceId }: { instanceId: number }) {
  const { data, isLoading } = useDatabaseUsers(instanceId)
  const users = data?.data || []

  if (isLoading) return <Skeleton className="h-64" />

  if (users.length === 0) {
    return (
      <Alert>
        <IconAlertCircle className="h-4 w-4" />
        <AlertDescription>暂无用户</AlertDescription>
      </Alert>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>用户名</TableHead>
          <TableHead>主机</TableHead>
          <TableHead>状态</TableHead>
          <TableHead>创建时间</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {users.map((user) => (
          <TableRow key={user.id}>
            <TableCell className="font-medium">{user.username}</TableCell>
            <TableCell>{user.host || '%'}</TableCell>
            <TableCell>{user.is_active ? '激活' : '禁用'}</TableCell>
            <TableCell>{new Date(user.created_at).toLocaleString()}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function PermissionList({ instanceId: _instanceId }: { instanceId: number }) {
  return (
    <Alert>
      <IconAlertCircle className="h-4 w-4" />
      <AlertDescription>权限管理功能开发中</AlertDescription>
    </Alert>
  )
}
