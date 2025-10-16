import React from 'react'
import { useForm } from 'react-hook-form'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

type FormValues = {
  user: string
  bucket: string
  prefix?: string
  permission: 'readonly' | 'writeonly' | 'readwrite'
}

type Props = {
  buckets: string[]
  onSubmit: (values: FormValues) => void
}

export const PermissionForm: React.FC<Props> = ({ buckets, onSubmit }) => {
  const { register, handleSubmit, setValue, watch } = useForm<FormValues>({
    defaultValues: { permission: 'readonly' },
  })

  const permission = watch('permission')
  const bucket = watch('bucket')

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <Label>User (AccessKey)</Label>
        <Input
          {...register('user', { required: true })}
          placeholder="User access key"
        />
      </div>
      <div>
        <Label>Bucket</Label>
        <Select onValueChange={(v) => setValue('bucket', v)} value={bucket}>
          <SelectTrigger>
            <SelectValue placeholder="Select bucket" />
          </SelectTrigger>
          <SelectContent>
            {buckets.map((b) => (
              <SelectItem key={b} value={b}>
                {b}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div>
        <Label>Prefix (optional)</Label>
        <Input {...register('prefix')} placeholder="path/within/bucket/" />
      </div>
      <div>
        <Label>Permission</Label>
        <Select
          onValueChange={(v) => setValue('permission', v as any)}
          value={permission}
        >
          <SelectTrigger>
            <SelectValue placeholder="Select permission" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="readonly">Readonly</SelectItem>
            <SelectItem value="writeonly">Writeonly</SelectItem>
            <SelectItem value="readwrite">Readwrite</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div>
        <Button type="submit">Grant</Button>
      </div>
    </form>
  )
}
