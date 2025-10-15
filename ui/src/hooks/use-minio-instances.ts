import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { minioApi, MinioInstanceCreate } from '../services/minio-api'

export function useMinioInstances() {
  const queryClient = useQueryClient()

  const list = useQuery({
    queryKey: ['minio', 'instances'],
    queryFn: () => minioApi.instances.list(),
  })

  const create = useMutation({
    mutationFn: (data: MinioInstanceCreate) => minioApi.instances.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['minio', 'instances'] })
    },
  })

  const update = useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: Partial<MinioInstanceCreate>
    }) => minioApi.instances.update(id, data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ['minio', 'instances'] })
      queryClient.invalidateQueries({
        queryKey: ['minio', 'instances', vars.id],
      })
    },
  })

  const remove = useMutation({
    mutationFn: (id: string) => minioApi.instances.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['minio', 'instances'] })
    },
  })

  const test = useMutation({
    mutationFn: (id: string) => minioApi.instances.test(id),
  })

  return { list, create, update, remove, test }
}
