import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { minioApi } from '../services/minio-api'

export function useFiles(instanceId: string, bucket: string, prefix?: string) {
  const queryClient = useQueryClient()

  const list = useQuery({
    queryKey: ['minio', 'files', instanceId, bucket, prefix || ''],
    queryFn: () => minioApi.files.list(instanceId, bucket, prefix),
    enabled: !!instanceId && !!bucket,
  })

  const upload = useMutation({
    mutationFn: ({ key, file }: { key: string; file: File }) =>
      minioApi.files.upload(instanceId, bucket, key, file),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['minio', 'files', instanceId, bucket, prefix || ''],
      })
    },
  })

  const remove = useMutation({
    mutationFn: (keys: string[]) =>
      minioApi.files.delete(instanceId, bucket, keys),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['minio', 'files', instanceId, bucket, prefix || ''],
      })
    },
  })

  const getDownloadUrl = (key: string) =>
    minioApi.files.downloadUrl(instanceId, bucket, key)
  const getPreviewUrl = (key: string) =>
    minioApi.files.previewUrl(instanceId, bucket, key)

  return { list, upload, remove, getDownloadUrl, getPreviewUrl }
}
