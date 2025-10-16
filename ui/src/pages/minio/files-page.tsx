import { useState } from 'react'
import { minioApi } from '@/services/minio-api'
import { useParams } from 'react-router-dom'

import { useBuckets } from '@/hooks/use-buckets'
import { useFiles } from '@/hooks/use-files'
import { Button } from '@/components/ui/button'
import { FileUploader } from '@/components/minio/file-uploader'
import { ShareDialog } from '@/components/minio/share-dialog'

export default function MinioFilesPage() {
  const { instanceId } = useParams<{ instanceId: string }>()
  const [bucket, setBucket] = useState<string>('')
  const [prefix, setPrefix] = useState<string>('')
  const { list: bucketList } = useBuckets(instanceId!)
  const buckets = ((bucketList.data as any)?.data || []) as any[]
  const { list: fileList, remove } = useFiles(instanceId!, bucket || '', prefix)
  const files = ((fileList.data as any)?.data?.objects || []) as any[]

  return (
    <div className="space-y-4">
      <div className="flex items-center space-x-2">
        <select
          className="border rounded px-2 py-1"
          value={bucket}
          onChange={(e) => setBucket(e.target.value)}
        >
          <option value="">Select bucket</option>
          {buckets.map((b) => (
            <option key={b.name} value={b.name}>
              {b.name}
            </option>
          ))}
        </select>
        <input
          className="border rounded px-2 py-1"
          placeholder="prefix/"
          value={prefix}
          onChange={(e) => setPrefix(e.target.value)}
        />
      </div>

      {instanceId && bucket && (
        <FileUploader
          instanceId={instanceId}
          bucket={bucket}
          onComplete={() => fileList.refetch()}
        />
      )}

      <div className="border rounded">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-muted">
              <th className="text-left p-2">Key</th>
              <th className="text-left p-2">Size</th>
              <th className="text-left p-2">Last Modified</th>
              <th className="text-right p-2">Actions</th>
            </tr>
          </thead>
          <tbody>
            {files.map((f) => (
              <tr key={f.key} className="border-t">
                <td className="p-2">{f.key}</td>
                <td className="p-2">{f.size}</td>
                <td className="p-2">{f.last_modified}</td>
                <td className="p-2 text-right space-x-2">
                  <ShareDialog
                    onCreate={async (expiry) => {
                      const res = (await minioApi.shares.create(
                        instanceId!,
                        bucket,
                        f.key,
                        expiry
                      )) as any
                      return { url: res?.data?.url || '' }
                    }}
                  />
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => remove.mutate([f.key])}
                  >
                    Delete
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
