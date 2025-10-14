import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import * as z from 'zod'
import { IconArrowLeft, IconDatabase } from '@tabler/icons-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useCreateInstance } from '@/services/database-api'
import { toast } from 'sonner'

const instanceFormSchema = z.object({
  name: z.string().min(1, '实例名称不能为空').max(100, '实例名称过长'),
  type: z.enum(['mysql', 'postgresql', 'redis']),
  host: z.string().min(1, '主机地址不能为空'),
  port: z.string().min(1, '端口号不能为空'),
  username: z.string().optional(),
  password: z.string().optional(),
  ssl_mode: z.string().optional(),
  description: z.string().optional(),
})

type InstanceFormValues = z.infer<typeof instanceFormSchema>

export function InstanceForm() {
  const navigate = useNavigate()
  const createMutation = useCreateInstance()
  const [showPassword, setShowPassword] = useState(false)

  const form = useForm<InstanceFormValues>({
    resolver: zodResolver(instanceFormSchema),
    defaultValues: {
      name: '',
      type: 'mysql',
      host: '',
      port: '3306',
      username: 'root',  // MySQL default username
      password: '',
      ssl_mode: '',
      description: '',
    },
  })

  const selectedType = form.watch('type')

  // 根据数据库类型设置默认端口和用户名
  const handleTypeChange = (type: 'mysql' | 'postgresql' | 'redis') => {
    const defaultPorts = {
      mysql: '3306',
      postgresql: '5432',
      redis: '6379',
    }
    const defaultUsernames = {
      mysql: 'root',
      postgresql: 'postgres',
      redis: '',  // Redis 不需要用户名
    }
    form.setValue('port', defaultPorts[type])
    form.setValue('username', defaultUsernames[type])
  }

  const onSubmit = async (values: InstanceFormValues) => {
    try {
      // Convert port string to number and remove empty optional fields
      const payload: any = {
        name: values.name,
        type: values.type,
        host: values.host,
        port: parseInt(values.port, 10)
      }

      // Only include optional fields if they have values
      if (values.username) payload.username = values.username
      if (values.password) payload.password = values.password
      if (values.ssl_mode) payload.ssl_mode = values.ssl_mode
      if (values.description) payload.description = values.description

      await createMutation.mutateAsync(payload)
      toast.success('数据库实例创建成功')
      navigate('/dbs/instances')
    } catch (error: any) {
      // Error from apiClient is a standard Error object with message
      const errorMessage = error?.message || error?.toString() || '创建实例失败'
      console.error('创建实例失败:', error)
      toast.error(errorMessage)
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" onClick={() => navigate('/dbs/instances')}>
          <IconArrowLeft className="w-4 h-4 mr-2" />
          返回
        </Button>
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <IconDatabase className="w-8 h-8" />
            新建数据库实例
          </h1>
          <p className="text-muted-foreground mt-2">添加 MySQL、PostgreSQL 或 Redis 数据库实例</p>
        </div>
      </div>

      {/* Form */}
      <Card>
        <CardHeader>
          <CardTitle>实例配置</CardTitle>
          <CardDescription>填写数据库连接信息</CardDescription>
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
                        <Input placeholder="例如: 生产环境MySQL" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* 数据库类型 */}
                <FormField
                  control={form.control}
                  name="type"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>数据库类型 *</FormLabel>
                      <Select
                        onValueChange={(value: 'mysql' | 'postgresql' | 'redis') => {
                          field.onChange(value)
                          handleTypeChange(value)
                        }}
                        defaultValue={field.value}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue placeholder="选择数据库类型" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectItem value="mysql">MySQL</SelectItem>
                          <SelectItem value="postgresql">PostgreSQL</SelectItem>
                          <SelectItem value="redis">Redis</SelectItem>
                        </SelectContent>
                      </Select>
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
                        <Input placeholder="例如: localhost 或 192.168.1.100" {...field} />
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
                        <Input type="number" placeholder="3306" {...field} />
                      </FormControl>
                      <FormDescription>
                        默认端口: MySQL(3306), PostgreSQL(5432), Redis(6379)
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* 用户名 */}
                {selectedType !== 'redis' && (
                  <FormField
                    control={form.control}
                    name="username"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>用户名</FormLabel>
                        <FormControl>
                          <Input placeholder={selectedType === 'mysql' ? 'root' : 'postgres'} {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}

                {/* 密码 */}
                <FormField
                  control={form.control}
                  name="password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>密码</FormLabel>
                      <FormControl>
                        <Input
                          type={showPassword ? 'text' : 'password'}
                          placeholder="数据库密码"
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>
                        <button
                          type="button"
                          onClick={() => setShowPassword(!showPassword)}
                          className="text-xs text-primary hover:underline"
                        >
                          {showPassword ? '隐藏' : '显示'}密码
                        </button>
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* SSL模式 (仅PostgreSQL) */}
                {selectedType === 'postgresql' && (
                  <FormField
                    control={form.control}
                    name="ssl_mode"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>SSL 模式</FormLabel>
                        <Select onValueChange={field.onChange} defaultValue={field.value}>
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="选择SSL模式" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="disable">禁用</SelectItem>
                            <SelectItem value="require">要求</SelectItem>
                            <SelectItem value="verify-ca">验证CA</SelectItem>
                            <SelectItem value="verify-full">完全验证</SelectItem>
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
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

              {/* 操作按钮 */}
              <div className="flex justify-end gap-4">
                <Button type="button" variant="outline" onClick={() => navigate('/dbs/instances')}>
                  取消
                </Button>
                <Button type="submit" disabled={createMutation.isPending}>
                  {createMutation.isPending ? '创建中...' : '创建实例'}
                </Button>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  )
}
