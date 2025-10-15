import React, { useState } from 'react'
import {
  AlertRule,
  AlertRuleService,
  AlertSeverity,
  AlertType,
} from '@/services/alert-rule'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertCircle,
  AlertTriangle,
  Edit,
  Info,
  Plus,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'

const AlertRulesPage: React.FC = () => {
  const queryClient = useQueryClient()
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const [deletingRuleId, setDeletingRuleId] = useState<string | null>(null)

  // Form state
  const [formData, setFormData] = useState({
    name: '',
    type: 'service' as AlertType,
    target_id: '',
    severity: 'warning' as AlertSeverity,
    condition: '',
    duration: 300,
    enabled: true,
    notify_channels: '',
  })

  // Fetch alert rules
  const { data: rulesData, isLoading } = useQuery({
    queryKey: ['alert-rules'],
    queryFn: () => AlertRuleService.listRules({ page: 1, page_size: 100 }),
  })

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: Partial<AlertRule>) => AlertRuleService.createRule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
      setIsCreateDialogOpen(false)
      resetForm()
      toast.success('告警规则创建成功')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.message || '创建告警规则失败')
    },
  })

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<AlertRule> }) =>
      AlertRuleService.updateRule(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
      setIsEditDialogOpen(false)
      setEditingRule(null)
      toast.success('告警规则更新成功')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.message || '更新告警规则失败')
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => AlertRuleService.deleteRule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alert-rules'] })
      setIsDeleteDialogOpen(false)
      setDeletingRuleId(null)
      toast.success('告警规则删除成功')
    },
    onError: (error: any) => {
      toast.error(error.response?.data?.message || '删除告警规则失败')
    },
  })

  const resetForm = () => {
    setFormData({
      name: '',
      type: 'service',
      target_id: '',
      severity: 'warning',
      condition: '',
      duration: 300,
      enabled: true,
      notify_channels: '',
    })
  }

  const handleCreate = () => {
    createMutation.mutate(formData)
  }

  const handleEdit = (rule: AlertRule) => {
    setEditingRule(rule)
    setFormData({
      name: rule.name,
      type: rule.type,
      target_id: rule.target_id,
      severity: rule.severity,
      condition: rule.condition,
      duration: rule.duration,
      enabled: rule.enabled,
      notify_channels: rule.notify_channels || '',
    })
    setIsEditDialogOpen(true)
  }

  const handleUpdate = () => {
    if (editingRule) {
      updateMutation.mutate({
        id: editingRule.id,
        data: formData,
      })
    }
  }

  const handleDelete = (id: string) => {
    setDeletingRuleId(id)
    setIsDeleteDialogOpen(true)
  }

  const confirmDelete = () => {
    if (deletingRuleId) {
      deleteMutation.mutate(deletingRuleId)
    }
  }

  const getSeverityBadge = (severity: AlertSeverity) => {
    switch (severity) {
      case 'critical':
        return (
          <Badge variant="destructive" className="flex items-center gap-1">
            <AlertCircle className="h-3 w-3" />
            严重
          </Badge>
        )
      case 'warning':
        return (
          <Badge className="bg-yellow-500 flex items-center gap-1">
            <AlertTriangle className="h-3 w-3" />
            警告
          </Badge>
        )
      case 'info':
        return (
          <Badge variant="outline" className="flex items-center gap-1">
            <Info className="h-3 w-3" />
            信息
          </Badge>
        )
      default:
        return <Badge variant="outline">{severity}</Badge>
    }
  }

  const getTypeBadge = (type: AlertType) => {
    return type === 'host' ? (
      <Badge variant="secondary">主机</Badge>
    ) : (
      <Badge variant="default">服务</Badge>
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" />
          <p className="mt-2 text-muted-foreground">加载中...</p>
        </div>
      </div>
    )
  }

  const rules = rulesData?.items || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">告警规则管理</h1>
          <p className="text-muted-foreground">配置主机和服务的告警规则</p>
        </div>
        <Button onClick={() => setIsCreateDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          创建规则
        </Button>
      </div>

      {/* Rules List */}
      <Card>
        <CardHeader>
          <CardTitle>告警规则列表</CardTitle>
          <CardDescription>共 {rules.length} 条规则</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>规则名称</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>严重程度</TableHead>
                <TableHead>触发条件</TableHead>
                <TableHead>持续时间</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rules.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8">
                    暂无告警规则
                  </TableCell>
                </TableRow>
              ) : (
                rules.map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell className="font-medium">{rule.name}</TableCell>
                    <TableCell>{getTypeBadge(rule.type)}</TableCell>
                    <TableCell>{getSeverityBadge(rule.severity)}</TableCell>
                    <TableCell>
                      <code className="text-xs bg-muted px-2 py-1 rounded">
                        {rule.condition}
                      </code>
                    </TableCell>
                    <TableCell>{rule.duration}秒</TableCell>
                    <TableCell>
                      {rule.enabled ? (
                        <Badge className="bg-green-500">启用</Badge>
                      ) : (
                        <Badge variant="outline">禁用</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEdit(rule)}
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDelete(rule.id)}
                        >
                          <Trash2 className="h-4 w-4 text-red-500" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Create Dialog */}
      <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>创建告警规则</DialogTitle>
            <DialogDescription>
              配置新的告警规则以监控主机或服务状态
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">规则名称</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder="例如: 服务可用率低于95%"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="type">规则类型</Label>
                <Select
                  value={formData.type}
                  onValueChange={(value: AlertType) =>
                    setFormData({ ...formData, type: value })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="host">主机</SelectItem>
                    <SelectItem value="service">服务</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="severity">严重程度</Label>
                <Select
                  value={formData.severity}
                  onValueChange={(value: AlertSeverity) =>
                    setFormData({ ...formData, severity: value })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="info">信息</SelectItem>
                    <SelectItem value="warning">警告</SelectItem>
                    <SelectItem value="critical">严重</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="target_id">目标ID</Label>
              <Input
                id="target_id"
                value={formData.target_id}
                onChange={(e) =>
                  setFormData({ ...formData, target_id: e.target.value })
                }
                placeholder="主机或服务的UUID"
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="condition">触发条件 (表达式)</Label>
              <Textarea
                id="condition"
                value={formData.condition}
                onChange={(e) =>
                  setFormData({ ...formData, condition: e.target.value })
                }
                placeholder="uptime_percentage < 95.0"
                rows={3}
              />
              <p className="text-xs text-muted-foreground">
                示例: uptime_percentage &lt; 99.9 && failed_checks &gt; 10
              </p>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="duration">持续时间 (秒)</Label>
              <Input
                id="duration"
                type="number"
                value={formData.duration}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    duration: parseInt(e.target.value) || 0,
                  })
                }
                placeholder="300"
              />
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="enabled"
                checked={formData.enabled}
                onCheckedChange={(checked) =>
                  setFormData({ ...formData, enabled: checked })
                }
              />
              <Label htmlFor="enabled">启用规则</Label>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsCreateDialogOpen(false)
                resetForm()
              }}
            >
              取消
            </Button>
            <Button onClick={handleCreate} disabled={createMutation.isPending}>
              {createMutation.isPending ? '创建中...' : '创建'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>编辑告警规则</DialogTitle>
            <DialogDescription>修改现有告警规则的配置</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">规则名称</Label>
              <Input
                id="edit-name"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="edit-type">规则类型</Label>
                <Select
                  value={formData.type}
                  onValueChange={(value: AlertType) =>
                    setFormData({ ...formData, type: value })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="host">主机</SelectItem>
                    <SelectItem value="service">服务</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="edit-severity">严重程度</Label>
                <Select
                  value={formData.severity}
                  onValueChange={(value: AlertSeverity) =>
                    setFormData({ ...formData, severity: value })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="info">信息</SelectItem>
                    <SelectItem value="warning">警告</SelectItem>
                    <SelectItem value="critical">严重</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="edit-condition">触发条件</Label>
              <Textarea
                id="edit-condition"
                value={formData.condition}
                onChange={(e) =>
                  setFormData({ ...formData, condition: e.target.value })
                }
                rows={3}
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="edit-duration">持续时间 (秒)</Label>
              <Input
                id="edit-duration"
                type="number"
                value={formData.duration}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    duration: parseInt(e.target.value) || 0,
                  })
                }
              />
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="edit-enabled"
                checked={formData.enabled}
                onCheckedChange={(checked) =>
                  setFormData({ ...formData, enabled: checked })
                }
              />
              <Label htmlFor="edit-enabled">启用规则</Label>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsEditDialogOpen(false)
                setEditingRule(null)
              }}
            >
              取消
            </Button>
            <Button onClick={handleUpdate} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? '保存中...' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              删除告警规则后，相关的告警事件也会被删除，此操作无法撤销。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setIsDeleteDialogOpen(false)
                setDeletingRuleId(null)
              }}
            >
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? '删除中...' : '确认删除'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

export default AlertRulesPage
