import { useEffect, useState } from 'react'
import {
  useCluster,
  useCreateCluster,
  useUpdateCluster,
} from '@/services/k8s-api'
import { zodResolver } from '@hookform/resolvers/zod'
import { IconArrowLeft, IconCloud, IconUpload } from '@tabler/icons-react'
import { useForm } from 'react-hook-form'
import { useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

const clusterFormSchema = z.object({
  name: z.string().min(1, '集群名称不能为空').max(100, '集群名称过长'),
  description: z.string().optional(),
  config: z.string().optional(),
  in_cluster: z.boolean(),
  prometheus_url: z.string().url('Prometheus URL 格式不正确').optional().or(z.literal('')),
  enable: z.boolean(),
})

type ClusterFormValues = z.infer<typeof clusterFormSchema>

export function ClusterFormPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEditMode = Boolean(id)

  const createMutation = useCreateCluster()
  const updateMutation = useUpdateCluster()
  const { data: clusterData, isLoading: loadingCluster } = useCluster(id || '')

  const [uploadingFile, setUploadingFile] = useState(false)

  const form = useForm<ClusterFormValues>({
    resolver: zodResolver(clusterFormSchema),
    defaultValues: {
      name: '',
      description: '',
      config: '',
      in_cluster: false,
      prometheus_url: '',
      enable: true,
    },
  })

  const inCluster = form.watch('in_cluster')

  // Load existing cluster data in edit mode
  useEffect(() => {
    if (isEditMode && clusterData?.data) {
      const cluster = clusterData.data
      form.reset({
        name: cluster.name,
        description: cluster.description || '',
        config: cluster.config || '',
        in_cluster: cluster.in_cluster,
        prometheus_url: cluster.prometheus_url || '',
        enable: cluster.enable,
      })
    }
  }, [isEditMode, clusterData, form])

  const onSubmit = async (values: ClusterFormValues) => {
    try {
      // Validate kubeconfig if not in-cluster mode
      if (!values.in_cluster && !values.config) {
        toast.error('非集群内模式需要提供 Kubeconfig')
        return
      }

      // Prepare payload
      const payload: any = {
        name: values.name,
        description: values.description,
        in_cluster: values.in_cluster,
        enable: values.enable,
      }

      if (!values.in_cluster && values.config) {
        payload.config = values.config
      }
      if (values.prometheus_url) {
        payload.prometheus_url = values.prometheus_url
      }

      if (isEditMode && id) {
        await updateMutation.mutateAsync({ id, data: payload })
        toast.success('集群更新成功')
      } else {
        await createMutation.mutateAsync(payload)
        toast.success('集群创建成功')
      }

      navigate('/k8s/clusters')
    } catch (error: any) {
      const errorMessage =
        error?.response?.data?.error ||
        error?.message ||
        `${isEditMode ? '更新' : '创建'}集群失败`
      console.error('集群表单错误:', error)
      toast.error(errorMessage)
    }
  }

  const handleFileUpload = async (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const file = event.target.files?.[0]
    if (!file) return

    setUploadingFile(true)
    try {
      const text = await file.text()
      form.setValue('config', text)
      toast.success('Kubeconfig 已加载')
    } catch (error) {
      toast.error('无法读取文件')
    } finally {
      setUploadingFile(false)
    }
  }

  if (isEditMode && loadingCluster) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate('/k8s/clusters')}>
            <IconArrowLeft className="w-4 h-4 mr-2" />
            返回
          </Button>
          <div className="flex-1">
            <Skeleton className="h-8 w-64 mb-2" />
            <Skeleton className="h-4 w-96" />
          </div>
        </div>
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-32 mb-2" />
            <Skeleton className="h-4 w-64" />
          </CardHeader>
          <CardContent className="space-y-6">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="space-y-2">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-10 w-full" />
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" onClick={() => navigate('/k8s/clusters')}>
          <IconArrowLeft className="w-4 h-4 mr-2" />
          返回
        </Button>
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <IconCloud className="w-8 h-8" />
            {isEditMode ? '编辑集群' : '新建集群'}
          </h1>
          <p className="text-muted-foreground mt-2">
            {isEditMode
              ? '更新 Kubernetes 集群配置'
              : '添加新的 Kubernetes 集群'}
          </p>
        </div>
      </div>

      {/* Form */}
      <Card>
        <CardHeader>
          <CardTitle>集群配置</CardTitle>
          <CardDescription>
            {inCluster
              ? '配置集群内访问模式（需要应用运行在 Kubernetes 集群内）'
              : '配置 Kubeconfig 访问模式'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* 集群名称 */}
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>集群名称 *</FormLabel>
                      <FormControl>
                        <Input placeholder="例如: 生产环境集群" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* 启用状态 */}
                <FormField
                  control={form.control}
                  name="enable"
                  render={({ field }) => (
                    <FormItem className="flex items-center justify-between rounded-lg border p-4">
                      <div className="space-y-0.5">
                        <FormLabel>启用集群</FormLabel>
                        <FormDescription>
                          禁用后将不会连接此集群
                        </FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              </div>

              {/* 集群内模式 */}
              <FormField
                control={form.control}
                name="in_cluster"
                render={({ field }) => (
                  <FormItem className="flex items-center justify-between rounded-lg border p-4">
                    <div className="space-y-0.5">
                      <FormLabel>集群内模式</FormLabel>
                      <FormDescription>
                        应用运行在 Kubernetes 集群内时使用 ServiceAccount
                        自动认证
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />

              {/* Kubeconfig (仅在非集群内模式) */}
              {!inCluster && (
                <FormField
                  control={form.control}
                  name="config"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Kubeconfig *</FormLabel>
                      <FormControl>
                        <div className="space-y-2">
                          <Textarea
                            placeholder="粘贴 Kubeconfig YAML 内容..."
                            className="font-mono text-xs h-64 resize-none"
                            {...field}
                          />
                          <div className="flex items-center gap-2">
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              disabled={uploadingFile}
                              onClick={() =>
                                document.getElementById('kubeconfig-file')?.click()
                              }
                            >
                              <IconUpload className="w-4 h-4 mr-2" />
                              {uploadingFile ? '上传中...' : '上传文件'}
                            </Button>
                            <input
                              id="kubeconfig-file"
                              type="file"
                              accept=".yaml,.yml,.config"
                              className="hidden"
                              onChange={handleFileUpload}
                            />
                            <span className="text-xs text-muted-foreground">
                              支持 YAML 格式的 Kubeconfig 文件
                            </span>
                          </div>
                        </div>
                      </FormControl>
                      <FormDescription>
                        通常位于 ~/.kube/config 文件，或使用 kubectl config view
                        --raw 获取
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {/* Prometheus URL */}
              <FormField
                control={form.control}
                name="prometheus_url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Prometheus URL</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="例如: http://prometheus-server.monitoring:9090"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      可选，用于集成 Prometheus 监控指标（留空将尝试自动发现）
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* 描述 */}
              <FormField
                control={form.control}
                name="description"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>描述</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder="可选的集群描述信息..."
                        className="resize-none"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* 操作按钮 */}
              <div className="flex justify-end gap-4">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => navigate('/k8s/clusters')}
                >
                  取消
                </Button>
                <Button
                  type="submit"
                  disabled={
                    createMutation.isPending || updateMutation.isPending
                  }
                >
                  {createMutation.isPending || updateMutation.isPending
                    ? `${isEditMode ? '更新' : '创建'}中...`
                    : `${isEditMode ? '更新' : '创建'}集群`}
                </Button>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  )
}
