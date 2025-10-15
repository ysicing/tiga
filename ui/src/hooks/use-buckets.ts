import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { minioApi } from '../services/minio-api'

export function useBuckets(instanceId: string) {
  const queryClient = useQueryClient()

  const list = useQuery({
    queryKey: ['minio', 'buckets', instanceId],
    queryFn: () => minioApi.buckets.list(instanceId),
    enabled: !!instanceId,
  })

  const get = (name: string) =>
    useQuery({
      queryKey: ['minio', 'bucket', instanceId, name],
      queryFn: () => minioApi.buckets.get(instanceId, name),
      enabled: !!instanceId && !!name,
    })

  const create = useMutation({
    mutationFn: ({ name, location }: { name: string; location?: string }) =>
      minioApi.buckets.create(instanceId, name, location),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['minio', 'buckets', instanceId],
      })
    },
  })

  const remove = useMutation({
    mutationFn: (name: string) => minioApi.buckets.delete(instanceId, name),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['minio', 'buckets', instanceId],
      })
    },
  })

  const updatePolicy = useMutation({
    mutationFn: ({ name, policy }: { name: string; policy: any }) =>
      minioApi.buckets.updatePolicy(instanceId, name, policy),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({
        queryKey: ['minio', 'bucket', instanceId, vars.name],
      })
    },
  })

  return { list, get, create, remove, updatePolicy }
}
