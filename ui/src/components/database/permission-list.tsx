import { IconAlertCircle } from '@tabler/icons-react'

import { Alert, AlertDescription } from '@/components/ui/alert'

export function PermissionList({
  instanceId: _instanceId,
}: {
  instanceId: number
}) {
  return (
    <Alert>
      <IconAlertCircle className="h-4 w-4" />
      <AlertDescription>权限管理功能开发中</AlertDescription>
    </Alert>
  )
}
