import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  FileText,
  Filter,
  Calendar,
  User,
  Database,
  CheckCircle2,
  XCircle,
} from 'lucide-react'

import {
  auditService,
  type AuditEvent,
  SUBSYSTEMS,
  ACTIONS,
} from '@/services/audit-service'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export function AuditLogsManagement() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState({
    subsystem: '',
    action: '',
    resource_type: '',
    user_uid: '',
    success: undefined as boolean | undefined,
  })
  const [selectedEvent, setSelectedEvent] = useState<AuditEvent | null>(null)
  const [showDetail, setShowDetail] = useState(false)

  // Fetch events
  const { data: eventsData, isLoading } = useQuery({
    queryKey: ['audit', 'events', page, filters],
    queryFn: () =>
      auditService.getEvents({
        ...filters,
        page,
        page_size: 20,
      }),
  })

  // Fetch config
  const { data: config } = useQuery({
    queryKey: ['audit', 'config'],
    queryFn: auditService.getConfig,
  })

  const formatTimestamp = (timestamp: number) => {
    return new Date(timestamp).toLocaleString()
  }

  const getSubsystemBadge = (subsystem: string) => {
    const colors: Record<string, string> = {
      http: 'bg-blue-100 text-blue-800',
      kubernetes: 'bg-purple-100 text-purple-800',
      database: 'bg-green-100 text-green-800',
      minio: 'bg-orange-100 text-orange-800',
      scheduler: 'bg-pink-100 text-pink-800',
      auth: 'bg-red-100 text-red-800',
    }
    return colors[subsystem] || 'bg-gray-100 text-gray-800'
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
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t('audit.total_events', 'Total Events')}
            </CardTitle>
            <FileText className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {eventsData?.pagination.total || 0}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t('audit.retention_days', 'Retention Period')}
            </CardTitle>
            <Calendar className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {config?.retention_days || 0}{' '}
              <span className="text-sm font-normal">
                {t('audit.days', 'days')}
              </span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">
              {t('audit.max_object_size', 'Max Object Size')}
            </CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {config?.max_object_bytes ? Math.floor(config.max_object_bytes / 1024) : 64}{' '}
              <span className="text-sm font-normal">KB</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Filter className="h-5 w-5" />
            {t('audit.filters', 'Filters')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-5">
            <div className="space-y-2">
              <Label>{t('audit.subsystem', 'Subsystem')}</Label>
              <Select
                value={filters.subsystem}
                onValueChange={(value) =>
                  setFilters({ ...filters, subsystem: value })
                }
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={t('audit.all_subsystems', 'All Subsystems')}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">
                    {t('audit.all_subsystems', 'All Subsystems')}
                  </SelectItem>
                  {SUBSYSTEMS.map((sub) => (
                    <SelectItem key={sub} value={sub}>
                      {sub}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>{t('audit.action', 'Action')}</Label>
              <Select
                value={filters.action}
                onValueChange={(value) =>
                  setFilters({ ...filters, action: value })
                }
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={t('audit.all_actions', 'All Actions')}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">
                    {t('audit.all_actions', 'All Actions')}
                  </SelectItem>
                  {ACTIONS.map((action) => (
                    <SelectItem key={action} value={action}>
                      {action}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>{t('audit.resource_type', 'Resource Type')}</Label>
              <Input
                placeholder={t('audit.resource_type_placeholder', 'e.g. cluster')}
                value={filters.resource_type}
                onChange={(e) =>
                  setFilters({ ...filters, resource_type: e.target.value })
                }
              />
            </div>

            <div className="space-y-2">
              <Label>{t('audit.user', 'User')}</Label>
              <Input
                placeholder={t('audit.user_uid_placeholder', 'User UID')}
                value={filters.user_uid}
                onChange={(e) =>
                  setFilters({ ...filters, user_uid: e.target.value })
                }
              />
            </div>

            <div className="space-y-2">
              <Label>{t('audit.status', 'Status')}</Label>
              <Select
                value={
                  filters.success === undefined
                    ? ''
                    : filters.success
                      ? 'success'
                      : 'failure'
                }
                onValueChange={(value) =>
                  setFilters({
                    ...filters,
                    success:
                      value === ''
                        ? undefined
                        : value === 'success'
                          ? true
                          : false,
                  })
                }
              >
                <SelectTrigger>
                  <SelectValue
                    placeholder={t('audit.all_status', 'All Status')}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">
                    {t('audit.all_status', 'All Status')}
                  </SelectItem>
                  <SelectItem value="success">
                    {t('audit.success', 'Success')}
                  </SelectItem>
                  <SelectItem value="failure">
                    {t('audit.failure', 'Failure')}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="mt-4 flex gap-2">
            <Button
              variant="outline"
              onClick={() =>
                setFilters({
                  subsystem: '',
                  action: '',
                  resource_type: '',
                  user_uid: '',
                  success: undefined,
                })
              }
            >
              {t('audit.reset_filters', 'Reset Filters')}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Events Table */}
      <Card>
        <CardHeader>
          <CardTitle>{t('audit.events', 'Audit Events')}</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('audit.timestamp', 'Timestamp')}</TableHead>
                <TableHead>{t('audit.subsystem', 'Subsystem')}</TableHead>
                <TableHead>{t('audit.action', 'Action')}</TableHead>
                <TableHead>{t('audit.resource', 'Resource')}</TableHead>
                <TableHead>{t('audit.user', 'User')}</TableHead>
                <TableHead>{t('audit.status', 'Status')}</TableHead>
                <TableHead>{t('audit.actions', 'Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {eventsData?.data.map((event) => {
                // Extract flattened fields from nested structures
                const resourceName = event.resource_name || event.resource?.data?.name || event.resource?.identifier
                const userName = event.user_name || event.user?.username || event.user?.uid
                const success = event.success !== undefined ? event.success : !event.error_message

                return (
                <TableRow key={event.id}>
                  <TableCell className="text-xs">
                    {formatTimestamp(event.timestamp)}
                  </TableCell>
                  <TableCell>
                    <Badge className={getSubsystemBadge(event.subsystem)}>
                      {event.subsystem}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <code className="text-xs bg-muted px-1 py-0.5 rounded">
                      {event.action}
                    </code>
                  </TableCell>
                  <TableCell>
                    <div className="text-xs">
                      <div className="font-medium">{event.resource_type}</div>
                      {resourceName && (
                        <div className="text-muted-foreground truncate max-w-xs">
                          {resourceName}
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs">
                    <div className="flex items-center gap-1">
                      <User className="h-3 w-3" />
                      {userName || 'System'}
                    </div>
                  </TableCell>
                  <TableCell>
                    {success ? (
                      <Badge variant="default" className="gap-1">
                        <CheckCircle2 className="h-3 w-3" />
                        {t('audit.success', 'Success')}
                      </Badge>
                    ) : (
                      <Badge variant="destructive" className="gap-1">
                        <XCircle className="h-3 w-3" />
                        {t('audit.failure', 'Failure')}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setSelectedEvent(event)
                        setShowDetail(true)
                      }}
                    >
                      {t('audit.view_details', 'Details')}
                    </Button>
                  </TableCell>
                </TableRow>
                )
              })}
            </TableBody>
          </Table>

          {/* Pagination */}
          {eventsData && eventsData.pagination.total_pages > 1 && (
            <div className="flex justify-between items-center mt-4">
              <div className="text-sm text-muted-foreground">
                {t('audit.page', 'Page')} {eventsData.pagination.page} of{' '}
                {eventsData.pagination.total_pages}
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page === 1}
                  onClick={() => setPage(page - 1)}
                >
                  {t('audit.previous', 'Previous')}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= eventsData.pagination.total_pages}
                  onClick={() => setPage(page + 1)}
                >
                  {t('audit.next', 'Next')}
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Event Detail Dialog */}
      <EventDetailDialog
        event={selectedEvent}
        open={showDetail}
        onOpenChange={setShowDetail}
      />
    </div>
  )
}

// Event Detail Dialog Component
function EventDetailDialog({
  event,
  open,
  onOpenChange,
}: {
  event: AuditEvent | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()

  if (!event) return null

  // Extract computed fields from nested structures
  const resourceName = event.resource_name || event.resource?.data?.name || event.resource?.identifier
  const userName = event.user_name || event.user?.username || event.user?.uid
  const success = event.success !== undefined ? event.success : !event.error_message
  const eventMetadata = event.metadata || event.data

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t('audit.event_details', 'Event Details')}</DialogTitle>
          <DialogDescription>
            {t('audit.event_id', 'Event ID')}: {event.id}
          </DialogDescription>
        </DialogHeader>

        <div className="h-[600px] overflow-y-auto">
          <div className="space-y-6">
            {/* Basic Info */}
            <div className="grid gap-4 grid-cols-2">
              <div>
                <Label className="text-xs text-muted-foreground">
                  {t('audit.timestamp', 'Timestamp')}
                </Label>
                <div className="text-sm">
                  {new Date(event.timestamp).toLocaleString()}
                </div>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">
                  {t('audit.subsystem', 'Subsystem')}
                </Label>
                <div className="text-sm">{event.subsystem}</div>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">
                  {t('audit.action', 'Action')}
                </Label>
                <div className="text-sm">{event.action}</div>
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">
                  {t('audit.resource_type', 'Resource Type')}
                </Label>
                <div className="text-sm">{event.resource_type}</div>
              </div>
              {resourceName && (
                <div>
                  <Label className="text-xs text-muted-foreground">
                    {t('audit.resource_name', 'Resource Name')}
                  </Label>
                  <div className="text-sm">{resourceName}</div>
                </div>
              )}
              {userName && (
                <div>
                  <Label className="text-xs text-muted-foreground">
                    {t('audit.user', 'User')}
                  </Label>
                  <div className="text-sm">{userName}</div>
                </div>
              )}
              {event.client_ip && (
                <div>
                  <Label className="text-xs text-muted-foreground">
                    {t('audit.client_ip', 'Client IP')}
                  </Label>
                  <div className="text-sm font-mono">{event.client_ip}</div>
                </div>
              )}
              <div>
                <Label className="text-xs text-muted-foreground">
                  {t('audit.status', 'Status')}
                </Label>
                <div>
                  {success ? (
                    <Badge variant="default">
                      {t('audit.success', 'Success')}
                    </Badge>
                  ) : (
                    <Badge variant="destructive">
                      {t('audit.failure', 'Failure')}
                    </Badge>
                  )}
                </div>
              </div>
            </div>

            {/* Error Message */}
            {event.error_message && (
              <div>
                <Label className="text-xs text-muted-foreground">
                  {t('audit.error_message', 'Error Message')}
                </Label>
                <pre className="text-sm bg-destructive/10 p-3 rounded-md mt-1 overflow-x-auto">
                  {event.error_message}
                </pre>
              </div>
            )}

            {/* Diff Object */}
            {event.diff_object && (
              <div className="space-y-4">
                <Label>{t('audit.changes', 'Changes')}</Label>

                {event.diff_object.old_object && (
                  <div>
                    <Label className="text-xs text-muted-foreground">
                      {t('audit.old_object', 'Old Object')}
                      {event.diff_object.old_object_truncated && (
                        <Badge variant="secondary" className="ml-2">
                          {t('audit.truncated', 'Truncated')}
                        </Badge>
                      )}
                    </Label>
                    <pre className="text-xs bg-muted p-3 rounded-md mt-1 overflow-x-auto">
                      {JSON.stringify(event.diff_object.old_object, null, 2)}
                    </pre>
                  </div>
                )}

                {event.diff_object.new_object && (
                  <div>
                    <Label className="text-xs text-muted-foreground">
                      {t('audit.new_object', 'New Object')}
                      {event.diff_object.new_object_truncated && (
                        <Badge variant="secondary" className="ml-2">
                          {t('audit.truncated', 'Truncated')}
                        </Badge>
                      )}
                    </Label>
                    <pre className="text-xs bg-muted p-3 rounded-md mt-1 overflow-x-auto">
                      {JSON.stringify(event.diff_object.new_object, null, 2)}
                    </pre>
                  </div>
                )}

                {event.diff_object.truncated_fields &&
                  event.diff_object.truncated_fields.length > 0 && (
                    <div>
                      <Label className="text-xs text-muted-foreground">
                        {t('audit.truncated_fields', 'Truncated Fields')}
                      </Label>
                      <div className="flex flex-wrap gap-2 mt-1">
                        {event.diff_object.truncated_fields.map((field) => (
                          <Badge key={field} variant="outline">
                            {field}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
              </div>
            )}

            {/* Metadata */}
            {eventMetadata && Object.keys(eventMetadata).length > 0 && (
              <div>
                <Label className="text-xs text-muted-foreground">
                  {t('audit.metadata', 'Metadata')}
                </Label>
                <pre className="text-xs bg-muted p-3 rounded-md mt-1 overflow-x-auto">
                  {JSON.stringify(eventMetadata, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
