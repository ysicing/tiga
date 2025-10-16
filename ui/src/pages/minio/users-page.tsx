import React, { useMemo, useState } from 'react'
import { minioApi } from '@/services/minio-api'
import { useParams } from 'react-router-dom'

import { useBuckets } from '@/hooks/use-buckets'
import { Button } from '@/components/ui/button'
import { PermissionForm } from '@/components/minio/permission-form'

export default function MinioUsersPage() {
  const { instanceId } = useParams<{ instanceId: string }>()
  const [users, setUsers] = useState<any[]>([])
  const [permList, setPermList] = useState<any[]>([])
  const { list: bucketList } = useBuckets(instanceId!)
  const buckets = useMemo(
    () => (((bucketList.data as any)?.data || []) as any[]).map((b) => b.name),
    [bucketList.data]
  )

  const refresh = async () => {
    const ures = (await minioApi.users.list(instanceId!)) as any
    setUsers(ures?.data || [])
    const pres = (await minioApi.permissions.list(instanceId!)) as any
    setPermList(pres?.data || [])
  }

  React.useEffect(() => {
    if (instanceId) refresh()
  }, [instanceId])

  const createUser = async () => {
    await minioApi.users.create(instanceId!)
    await refresh()
  }

  const deleteUser = async (username: string) => {
    await minioApi.users.delete(instanceId!, username)
    await refresh()
  }

  // Build map user_id -> access_key for revoke
  const userAccessKeyMap = useMemo(() => {
    const m = new Map<string, string>()
    users.forEach((u: any) =>
      m.set(u.id || u.user_id || u.access_key, u.access_key)
    )
    return m
  }, [users])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Users</h1>
        <Button onClick={createUser}>Create User</Button>
      </div>

      <div className="border rounded">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-muted">
              <th className="text-left p-2">AccessKey</th>
              <th className="text-left p-2">Status</th>
              <th className="text-left p-2">Policy</th>
              <th className="text-right p-2">Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.access_key} className="border-t">
                <td className="p-2">{u.access_key}</td>
                <td className="p-2">{u.status}</td>
                <td className="p-2">{u.policy}</td>
                <td className="p-2 text-right">
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => deleteUser(u.access_key)}
                  >
                    Delete
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div>
        <h2 className="text-xl font-semibold mb-2">Grant Permission</h2>
        <PermissionForm
          buckets={buckets}
          onSubmit={async (v) => {
            await minioApi.permissions.grant(
              instanceId!,
              v.user,
              v.bucket,
              v.permission,
              v.prefix
            )
            await refresh()
          }}
        />
      </div>

      <div className="border rounded">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-muted">
              <th className="text-left p-2">ID</th>
              <th className="text-left p-2">User</th>
              <th className="text-left p-2">Bucket</th>
              <th className="text-left p-2">Prefix</th>
              <th className="text-left p-2">Permission</th>
              <th className="text-right p-2">Actions</th>
            </tr>
          </thead>
          <tbody>
            {permList.map((p) => (
              <tr key={p.id} className="border-t">
                <td className="p-2">{p.id}</td>
                <td className="p-2">{p.user_id}</td>
                <td className="p-2">{p.bucket_name}</td>
                <td className="p-2">{p.prefix}</td>
                <td className="p-2">{p.permission}</td>
                <td className="p-2 text-right">
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={async () => {
                      const accessKey = userAccessKeyMap.get(p.user_id) || ''
                      await minioApi.permissions.revoke(
                        p.id,
                        instanceId!,
                        accessKey
                      )
                      await refresh()
                    }}
                  >
                    Revoke
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
