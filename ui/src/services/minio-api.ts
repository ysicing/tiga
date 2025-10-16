import { API_BASE_URL, apiClient } from '../lib/api-client'

export interface MinioInstanceCreate {
  name: string
  description?: string
  host: string
  port: number
  use_ssl?: boolean
  access_key: string
  secret_key: string
  owner_id: string
  labels?: Record<string, any>
}

export const minioApi = {
  // Instances
  instances: {
    list: () => apiClient.get('/minio/instances'),
    get: (id: string) => apiClient.get(`/minio/instances/${id}`),
    create: (data: MinioInstanceCreate) =>
      apiClient.post('/minio/instances', data),
    update: (id: string, data: Partial<MinioInstanceCreate>) =>
      apiClient.put(`/minio/instances/${id}`, data),
    delete: (id: string) => apiClient.delete(`/minio/instances/${id}`),
    test: (id: string) => apiClient.post(`/minio/instances/${id}/test`),
  },

  // Buckets
  buckets: {
    list: (instanceId: string) =>
      apiClient.get(`/minio/instances/${instanceId}/buckets`),
    get: (instanceId: string, bucket: string) =>
      apiClient.get(`/minio/instances/${instanceId}/buckets/${bucket}`),
    create: (instanceId: string, name: string, location?: string) =>
      apiClient.post(`/minio/instances/${instanceId}/buckets`, {
        name,
        location,
      }),
    delete: (instanceId: string, name: string) =>
      apiClient.delete(`/minio/instances/${instanceId}/buckets/${name}`),
    updatePolicy: (instanceId: string, name: string, policy: any) =>
      apiClient.put(`/minio/instances/${instanceId}/buckets/${name}/policy`, {
        policy,
      }),
  },

  // Files (generic file API)
  files: {
    list: (instanceId: string, bucket: string, prefix?: string) =>
      apiClient.get(`/minio/instances/${instanceId}/files`, { bucket, prefix }),
    upload: (instanceId: string, bucket: string, key: string, file: File) => {
      const form = new FormData()
      form.append('bucket', bucket)
      form.append('name', key)
      form.append('file', file)
      return apiClient.postForm(`/minio/instances/${instanceId}/files`, form)
    },
    downloadUrl: (instanceId: string, bucket: string, key: string) =>
      apiClient.get(`/minio/instances/${instanceId}/files/download`, {
        bucket,
        key,
      }),
    previewUrl: (instanceId: string, bucket: string, key: string) =>
      apiClient.get(`/minio/instances/${instanceId}/files/preview`, {
        bucket,
        key,
      }),
    delete: async (instanceId: string, bucket: string, keys: string[]) => {
      const res = await fetch(
        `${API_BASE_URL}/minio/instances/${instanceId}/files`,
        {
          method: 'DELETE',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ bucket, keys }),
        }
      )
      if (!res.ok) {
        throw new Error(`Failed to delete files: ${res.status}`)
      }
      return res.json()
    },
  },

  // Users
  users: {
    list: (instanceId: string) =>
      apiClient.get(`/minio/instances/${instanceId}/users`),
    create: (instanceId: string, accessKey?: string, secretKey?: string) =>
      apiClient.post(`/minio/instances/${instanceId}/users`, {
        access_key: accessKey,
        secret_key: secretKey,
      }),
    delete: (instanceId: string, username: string) =>
      apiClient.delete(`/minio/instances/${instanceId}/users/${username}`),
  },

  // Permissions
  permissions: {
    grant: (
      instanceId: string,
      user: string,
      bucket: string,
      permission: 'readonly' | 'writeonly' | 'readwrite',
      prefix?: string
    ) =>
      apiClient.post(`/minio/permissions`, {
        instance_id: instanceId,
        user,
        bucket,
        permission,
        prefix,
      }),
    list: (instanceId: string, user?: string, bucket?: string) =>
      apiClient.get(`/minio/permissions`, {
        instance_id: instanceId,
        user,
        bucket,
      }),
    revoke: (id: string, instanceId: string, user: string) =>
      apiClient.delete(
        `/minio/permissions/${id}?instance_id=${encodeURIComponent(instanceId)}&user=${encodeURIComponent(user)}`
      ),
  },

  // Shares
  shares: {
    create: (
      instanceId: string,
      bucket: string,
      key: string,
      expiry: '1h' | '1d' | '7d' | '30d'
    ) =>
      apiClient.post(`/minio/shares`, {
        instance_id: instanceId,
        bucket,
        key,
        expiry,
      }),
    list: (instanceId?: string) =>
      instanceId
        ? apiClient.get(`/minio/shares`, { instance_id: instanceId })
        : apiClient.get(`/minio/shares`),
    revoke: (id: string) => apiClient.delete(`/minio/shares/${id}`),
  },
}
