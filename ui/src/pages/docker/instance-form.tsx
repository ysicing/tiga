import { useNavigate, useParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { toast } from 'sonner'
import { IconArrowLeft, IconBrandDocker } from '@tabler/icons-react'

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
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import {
  useCreateDockerInstance,
  useUpdateDockerInstance,
  useDockerInstance,
} from '@/services/docker-api'

const instanceFormSchema = z.object({
  name: z.string().min(1, '实例名称不能为空').max(100, '实例名称过长'),
  agent_id: z.string().min(1, 'Agent ID 不能为空'),
  host: z.string().min(1, '主机地址不能为空'),
  port: z.string().min(1, '端口号不能为空'),
  description: z.string().optional(),
  tls_enabled: z.boolean().optional(),
  tls_ca: z.string().optional(),
  tls_cert: z.string().optional(),
  tls_key: z.string().optional(),
})

type InstanceFormValues = z.infer<typeof instanceFormSchema>

export function DockerInstanceForm() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEditMode = Boolean(id)

  const { data: instanceData } = useDockerInstance(id!)
  const createMutation = useCreateDockerInstance()
  const updateMutation = useUpdateDockerInstance()

  const form = useForm<InstanceFormValues>({
    resolver: zodResolver(instanceFormSchema),
    defaultValues: {
      name: instanceData?.data?.name || '',
      agent_id: instanceData?.data?.agent_id || '',
      host: instanceData?.data?.host || '',
      port: instanceData?.data?.port?.toString() || '2375',
      description: instanceData?.data?.description || '',
      tls_enabled: false,
      tls_ca: '',
      tls_cert: '',
      tls_key: '',
    },
  })

  const tls_enabled = form.watch('tls_enabled')

  const onSubmit = async (values: InstanceFormValues) => {
    try {
      const payload: any = {
        name: values.name,
        agent_id: values.agent_id,
        host: values.host,
        port: parseInt(values.port, 10),
      }

      // Only include optional fields if they have values
      if (values.description) payload.description = values.description
      if (values.tls_enabled) {
        payload.tls_enabled = values.tls_enabled
        if (values.tls_ca) payload.tls_ca = values.tls_ca
        if (values.tls_cert) payload.tls_cert = values.tls_cert
        if (values.tls_key) payload.tls_key = values.tls_key
      }

      if (isEditMode && id) {
        await updateMutation.mutateAsync({ id, data: payload })
        toast.success('Docker 实例更新成功')
      } else {
        await createMutation.mutateAsync(payload)
        toast.success('Docker 实例创建成功')
      }
      navigate('/docker/instances')
    } catch (error: any) {
      const errorMessage =
        error?.message || error?.toString() || '操作失败'
      console.error('操作失败:', error)
      toast.error(errorMessage)
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" onClick={() => navigate('/docker/instances')}>
          <IconArrowLeft className="w-4 h-4 mr-2" />
          返回
        </Button>
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <IconBrandDocker className="w-8 h-8" />
            {isEditMode ? '编辑' : '新建'} Docker 实例
          </h1>
          <p className="text-muted-foreground mt-2">
            {isEditMode
              ? '修改 Docker 实例配置'
              : '添加新的 Docker 实例进行管理'}
          </p>
        </div>
      </div>

      {/* Form */}
      <Card>
        <CardHeader>
          <CardTitle>实例配置</CardTitle>
          <CardDescription>填写 Docker 实例连接信息</CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* 实例名称 */}
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>实例名称 *</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="例如: 生产环境Docker"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* Agent ID */}
                <FormField
                  control={form.control}
                  name="agent_id"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Agent ID *</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="关联的 Agent UUID"
                          {...field}
                          disabled={isEditMode}
                        />
                      </FormControl>
                      <FormDescription>
                        关联到主机节点的 Agent ID（创建后不可修改）
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* 主机地址 */}
                <FormField
                  control={form.control}
                  name="host"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>主机地址 *</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="例如: localhost 或 192.168.1.100"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* 端口号 */}
                <FormField
                  control={form.control}
                  name="port"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>端口号 *</FormLabel>
                      <FormControl>
                        <Input type="number" placeholder="2375" {...field} />
                      </FormControl>
                      <FormDescription>
                        Docker API 端口 (默认: 2375 无TLS, 2376 有TLS)
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              {/* 描述 */}
              <FormField
                control={form.control}
                name="description"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>描述</FormLabel>
                    <FormControl>
                      <Textarea
                        placeholder="可选的实例描述信息..."
                        className="resize-none"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* TLS 配置 */}
              <div className="space-y-4 border-t pt-4">
                <FormField
                  control={form.control}
                  name="tls_enabled"
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-start space-x-3 space-y-0">
                      <FormControl>
                        <Checkbox
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                      <div className="space-y-1 leading-none">
                        <FormLabel>启用 TLS</FormLabel>
                        <FormDescription>
                          使用 TLS 加密连接到 Docker API（推荐生产环境使用）
                        </FormDescription>
                      </div>
                    </FormItem>
                  )}
                />

                {tls_enabled && (
                  <div className="space-y-4 pl-6 border-l-2 border-muted">
                    <FormField
                      control={form.control}
                      name="tls_ca"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>CA 证书</FormLabel>
                          <FormControl>
                            <Textarea
                              placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                              className="font-mono text-xs resize-none"
                              rows={4}
                              {...field}
                            />
                          </FormControl>
                          <FormDescription>
                            PEM 格式的 CA 证书内容
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="tls_cert"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>客户端证书</FormLabel>
                          <FormControl>
                            <Textarea
                              placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                              className="font-mono text-xs resize-none"
                              rows={4}
                              {...field}
                            />
                          </FormControl>
                          <FormDescription>
                            PEM 格式的客户端证书内容
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="tls_key"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>客户端密钥</FormLabel>
                          <FormControl>
                            <Textarea
                              placeholder="-----BEGIN RSA PRIVATE KEY-----&#10;...&#10;-----END RSA PRIVATE KEY-----"
                              className="font-mono text-xs resize-none"
                              rows={4}
                              {...field}
                            />
                          </FormControl>
                          <FormDescription>
                            PEM 格式的客户端私钥内容
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                )}
              </div>

              {/* 操作按钮 */}
              <div className="flex justify-end gap-4">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => navigate('/docker/instances')}
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
                    ? isEditMode
                      ? '更新中...'
                      : '创建中...'
                    : isEditMode
                      ? '更新实例'
                      : '创建实例'}
                </Button>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  )
}
