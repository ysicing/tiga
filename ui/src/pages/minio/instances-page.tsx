import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus } from 'lucide-react'

import { useMinioInstances } from '@/hooks/use-minio-instances'
import { useAuth } from '@/contexts/auth-context'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { InstanceForm } from '@/components/minio/instance-form'
import type { MinioInstanceCreate } from '@/services/minio-api'

export default function MinioInstancesPage() {
  const { user } = useAuth()
  const { list, create } = useMinioInstances()
  const items = (list.data as any)?.data || []
  const [isDialogOpen, setIsDialogOpen] = useState(false)

  const handleCreate = async (data: MinioInstanceCreate) => {
    try {
      await create.mutateAsync(data)
      setIsDialogOpen(false)
    } catch (error) {
      console.error('Failed to create instance:', error)
    }
  }

  return (
    <div className="container mx-auto p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">MinIO Instances</h1>
          <p className="text-muted-foreground">
            Manage MinIO object storage instances
          </p>
        </div>
        <Button onClick={() => setIsDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Add Instance
        </Button>
      </div>

      {items.length === 0 ? (
        <div className="border-2 border-dashed rounded-lg p-12 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-muted">
            <Plus className="h-6 w-6 text-muted-foreground" />
          </div>
          <h3 className="mt-4 text-lg font-semibold">No instances yet</h3>
          <p className="mt-2 text-sm text-muted-foreground max-w-sm mx-auto">
            Get started by adding your first MinIO instance. You'll be able to
            manage buckets, files, and users.
          </p>
          <Button onClick={() => setIsDialogOpen(true)} className="mt-6">
            <Plus className="h-4 w-4 mr-2" />
            Add Your First Instance
          </Button>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {items.map((instance: any) => (
            <div
              key={instance.id}
              className="border rounded-lg p-4 hover:shadow-md transition-shadow"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <h3 className="font-semibold text-lg">{instance.name}</h3>
                  {instance.description && (
                    <p className="text-sm text-muted-foreground mt-1">
                      {instance.description}
                    </p>
                  )}
                  <div className="mt-3 space-y-1">
                    <div className="flex items-center text-sm">
                      <span className="text-muted-foreground">Endpoint:</span>
                      <span className="ml-2 font-mono">
                        {instance.connection?.host}:{instance.connection?.port}
                      </span>
                    </div>
                    {instance.status && (
                      <div className="flex items-center text-sm">
                        <span className="text-muted-foreground">Status:</span>
                        <span
                          className={`ml-2 px-2 py-0.5 rounded text-xs font-medium ${
                            instance.status === 'active'
                              ? 'bg-green-100 text-green-700'
                              : 'bg-gray-100 text-gray-700'
                          }`}
                        >
                          {instance.status}
                        </span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
              <div className="mt-4 pt-4 border-t flex gap-2">
                <Button asChild variant="outline" size="sm" className="flex-1">
                  <Link to={`/minio/${instance.id}/overview`}>Open</Link>
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Add MinIO Instance</DialogTitle>
            <DialogDescription>
              Connect to a MinIO server by providing the endpoint and
              credentials.
            </DialogDescription>
          </DialogHeader>
          {user && (
            <InstanceForm ownerId={user.id} onSubmit={handleCreate} />
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
