import React from 'react'
import { MinioInstanceCreate } from '@/services/minio-api'
import { useForm } from 'react-hook-form'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Props = {
  ownerId: string
  onSubmit: (data: MinioInstanceCreate) => void
  defaultValues?: Partial<MinioInstanceCreate>
}

export const InstanceForm: React.FC<Props> = ({
  ownerId,
  onSubmit,
  defaultValues,
}) => {
  const { register, handleSubmit } = useForm<MinioInstanceCreate>({
    defaultValues: {
      name: '',
      description: '',
      host: '',
      port: 9000,
      use_ssl: false,
      access_key: '',
      secret_key: '',
      owner_id: ownerId,
      ...defaultValues,
    },
  })

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <Label htmlFor="name">Name</Label>
        <Input id="name" {...register('name', { required: true })} />
      </div>
      <div>
        <Label htmlFor="description">Description</Label>
        <Input id="description" {...register('description')} />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label htmlFor="host">Endpoint Host</Label>
          <Input id="host" {...register('host', { required: true })} />
        </div>
        <div>
          <Label htmlFor="port">Port</Label>
          <Input
            id="port"
            type="number"
            {...register('port', { valueAsNumber: true })}
          />
        </div>
      </div>
      <div className="flex items-center space-x-2">
        <Checkbox id="use_ssl" {...register('use_ssl')} />
        <Label htmlFor="use_ssl">Use SSL</Label>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label htmlFor="access_key">Access Key</Label>
          <Input
            id="access_key"
            {...register('access_key', { required: true })}
          />
        </div>
        <div>
          <Label htmlFor="secret_key">Secret Key</Label>
          <Input
            id="secret_key"
            type="password"
            {...register('secret_key', { required: true })}
          />
        </div>
      </div>
      <div>
        <Button type="submit">Save</Button>
      </div>
    </form>
  )
}
