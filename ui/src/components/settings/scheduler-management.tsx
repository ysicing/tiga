import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  Play,
  Clock,
  CheckCircle2,
  XCircle,
  AlertCircle,
  TrendingUp,
} from 'lucide-react'

import { schedulerService, type SchedulerTask } from '@/services/scheduler-service'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

export function SchedulerManagement() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedTask, setSelectedTask] = useState<SchedulerTask | null>(null)
  const [showHistory, setShowHistory] = useState(false)

  // Fetch tasks
  const { data: tasks, isLoading } = useQuery({
    queryKey: ['scheduler', 'tasks'],
    queryFn: schedulerService.getTasks,
  })

  // Trigger task mutation
  const triggerMutation = useMutation({
    mutationFn: (uid: string) => schedulerService.triggerTask(uid),
    onSuccess: (data) => {
      const executionInfo = data.execution_uid
        ? `Execution UID: ${data.execution_uid}`
        : data.message || 'Task triggered successfully'

      toast.success(
        t('scheduler.trigger_success', 'Task Triggered'),
        {
          description: executionInfo,
        }
      )
      queryClient.invalidateQueries({ queryKey: ['scheduler'] })
    },
    onError: (error: any) => {
      toast.error(
        t('scheduler.trigger_error', 'Trigger Failed'),
        {
          description: error.message,
        }
      )
    },
  })

  const formatNextRun = (nextRun?: string) => {
    if (!nextRun) return '-'
    const date = new Date(nextRun)
    const now = new Date()
    const diffMs = date.getTime() - now.getTime()
    const diffMins = Math.floor(diffMs / 60000)

    if (diffMins < 0) return t('scheduler.overdue', 'Overdue')
    if (diffMins < 60) return t('scheduler.in_minutes', { minutes: diffMins })
    const diffHours = Math.floor(diffMins / 60)
    if (diffHours < 24) return t('scheduler.in_hours', { hours: diffHours })
    const diffDays = Math.floor(diffHours / 24)
    return t('scheduler.in_days', { days: diffDays })
  }

  const getSuccessRate = (task: SchedulerTask) => {
    if (task.total_executions === 0) return '0.0'
    return ((task.success_executions / task.total_executions) * 100).toFixed(1)
  }

  const getSuccessRateBadge = (rate: number) => {
    if (rate >= 90) return 'default'
    if (rate >= 70) return 'secondary'
    return 'destructive'
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-96" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t('scheduler.total_tasks', 'Total Tasks')}
            </CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{tasks?.length || 0}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t('scheduler.enabled_tasks', 'Enabled')}
            </CardTitle>
            <CheckCircle2 className="h-4 w-4 text-green-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {tasks?.filter((t) => t.enabled).length || 0}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t('scheduler.total_executions', 'Total Executions')}
            </CardTitle>
            <TrendingUp className="h-4 w-4 text-blue-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {tasks?.reduce((sum, t) => sum + t.total_executions, 0) || 0}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t('scheduler.avg_success_rate', 'Avg Success Rate')}
            </CardTitle>
            <AlertCircle className="h-4 w-4 text-yellow-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {tasks && tasks.length > 0
                ? (
                    tasks.reduce(
                      (sum, t) => sum + parseFloat(getSuccessRate(t)),
                      0
                    ) / tasks.length
                  ).toFixed(1)
                : '0.0'}
              %
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tasks Table */}
      <Card>
        <CardHeader>
          <CardTitle>{t('scheduler.scheduled_tasks', 'Scheduled Tasks')}</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('scheduler.name', 'Name')}</TableHead>
                <TableHead>{t('scheduler.description', 'Description')}</TableHead>
                <TableHead>{t('scheduler.schedule', 'Schedule')}</TableHead>
                <TableHead>{t('scheduler.next_run', 'Next Run')}</TableHead>
                <TableHead>{t('scheduler.executions', 'Executions')}</TableHead>
                <TableHead>{t('scheduler.success_rate', 'Success Rate')}</TableHead>
                <TableHead>{t('scheduler.status', 'Status')}</TableHead>
                <TableHead>{t('scheduler.actions', 'Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tasks?.map((task) => {
                const successRate = parseFloat(getSuccessRate(task))
                return (
                  <TableRow key={task.uid}>
                    <TableCell className="font-medium">{task.name}</TableCell>
                    <TableCell className="max-w-xs truncate">
                      {task.description || '-'}
                    </TableCell>
                    <TableCell>
                      {task.is_recurring ? (
                        <code className="text-xs bg-muted px-1 py-0.5 rounded">
                          {task.cron_expr}
                        </code>
                      ) : (
                        t('scheduler.one_time', 'One-time')
                      )}
                    </TableCell>
                    <TableCell>{formatNextRun(task.next_run)}</TableCell>
                    <TableCell>
                      <div className="flex gap-2 text-xs">
                        <span className="text-green-600">
                          ✓ {task.success_executions}
                        </span>
                        <span className="text-red-600">
                          ✗ {task.failure_executions}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={getSuccessRateBadge(successRate)}>
                        {successRate}%
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {task.enabled ? (
                        <Badge variant="default">
                          {t('scheduler.enabled', 'Enabled')}
                        </Badge>
                      ) : (
                        <Badge variant="secondary">
                          {t('scheduler.disabled', 'Disabled')}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => triggerMutation.mutate(task.uid)}
                          disabled={
                            !task.enabled || triggerMutation.isPending
                          }
                        >
                          <Play className="h-3 w-3 mr-1" />
                          {t('scheduler.trigger', 'Trigger')}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setSelectedTask(task)
                            setShowHistory(true)
                          }}
                        >
                          {t('scheduler.history', 'History')}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Execution History Dialog */}
      <ExecutionHistoryDialog
        task={selectedTask}
        open={showHistory}
        onOpenChange={setShowHistory}
      />
    </div>
  )
}

// Execution History Dialog Component
function ExecutionHistoryDialog({
  task,
  open,
  onOpenChange,
}: {
  task: SchedulerTask | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()

  const { data: executions, isLoading } = useQuery({
    queryKey: ['scheduler', 'executions', task?.uid],
    queryFn: () =>
      schedulerService.getExecutions({
        task_uid: task?.uid,
        page: 1,
        page_size: 50,
      }),
    enabled: !!task && open,
  })

  const getStateBadge = (state: string) => {
    const variants: Record<string, any> = {
      success: 'default',
      running: 'secondary',
      pending: 'outline',
      failure: 'destructive',
      timeout: 'destructive',
      cancelled: 'secondary',
    }
    return variants[state] || 'outline'
  }

  const getStateIcon = (state: string) => {
    const icons: Record<string, any> = {
      success: <CheckCircle2 className="h-3 w-3" />,
      running: <Clock className="h-3 w-3 animate-spin" />,
      failure: <XCircle className="h-3 w-3" />,
      timeout: <AlertCircle className="h-3 w-3" />,
    }
    return icons[state] || null
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl">
        <DialogHeader>
          <DialogTitle>
            {t('scheduler.execution_history', 'Execution History')} -{' '}
            {task?.name}
          </DialogTitle>
          <DialogDescription>
            {t(
              'scheduler.execution_history_desc',
              'Recent execution records for this task'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="h-[500px] overflow-y-auto overflow-x-auto">
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-16" />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-24">{t('scheduler.execution_id', 'ID')}</TableHead>
                  <TableHead className="w-32">{t('scheduler.state', 'State')}</TableHead>
                  <TableHead className="w-48">{t('scheduler.started_at', 'Started')}</TableHead>
                  <TableHead className="w-24">{t('scheduler.duration', 'Duration')}</TableHead>
                  <TableHead className="w-28">{t('scheduler.trigger', 'Trigger')}</TableHead>
                  <TableHead className="min-w-[300px]">{t('scheduler.result', 'Result')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {executions?.data.map((exec) => (
                  <TableRow key={exec.id}>
                    <TableCell className="font-mono text-xs">
                      {exec.execution_uid.substring(0, 8)}
                    </TableCell>
                    <TableCell>
                      <Badge variant={getStateBadge(exec.state)}>
                        <span className="flex items-center gap-1">
                          {getStateIcon(exec.state)}
                          {exec.state}
                        </span>
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs whitespace-nowrap">
                      {new Date(exec.started_at).toLocaleString()}
                    </TableCell>
                    <TableCell className="whitespace-nowrap">
                      {exec.duration_ms
                        ? `${(exec.duration_ms / 1000).toFixed(2)}s`
                        : '-'}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{exec.triggered_by}</Badge>
                    </TableCell>
                    <TableCell>
                      {exec.error ? (
                        <span className="text-xs text-destructive block break-words">
                          {exec.error}
                        </span>
                      ) : exec.result ? (
                        <span className="text-xs text-muted-foreground block break-words">
                          {exec.result}
                        </span>
                      ) : (
                        '-'
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
