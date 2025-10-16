import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { minioApi } from '../services/minio-api'

export function useShares(instanceId?: string) {
  const queryClient = useQueryClient()

  const list = useQuery({
    queryKey: ['minio', 'shares', instanceId || 'all'],
    queryFn: () => minioApi.shares.list(instanceId),
  })

  const create = useMutation({
    mutationFn: ({
      bucket,
      key,
      expiry,
    }: {
      bucket: string
      key: string
      expiry: '1h' | '1d' | '7d' | '30d'
    }) => minioApi.shares.create(instanceId!, bucket, key, expiry),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['minio', 'shares', instanceId || 'all'],
      })
    },
  })

  const revoke = useMutation({
    mutationFn: (id: string) => minioApi.shares.revoke(id),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['minio', 'shares', instanceId || 'all'],
      })
    },
  })

  return { list, create, revoke }
}
