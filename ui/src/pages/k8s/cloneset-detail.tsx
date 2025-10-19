import { useEffect, useState, useMemo } from 'react'
import {
  IconLoader,
  IconRefresh,
  IconReload,
  IconScale,
  IconTrash,
} from '@tabler/icons-react'
import * as yaml from 'js-yaml'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  deleteResource,
  updateResource,
  useResource,
  useResources,
  restartCloneSet,
} from '@/lib/api'
import { getCloneSetStatus, toSimpleContainer, isOpenKruiseResourceRestartable } from '@/lib/k8s'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { ResponsiveTabs } from '@/components/ui/responsive-tabs'
import { ContainerTable } from '@/components/container-table'
import { DeleteConfirmationDialog } from '@/components/delete-confirmation-dialog'
import { DeploymentStatusIcon } from '@/components/deployment-status-icon'
import { EventTable } from '@/components/event-table'
import { LabelsAnno } from '@/components/lables-anno'
import { LogViewer } from '@/components/log-viewer'
import { PodMonitoring } from '@/components/pod-monitoring'
import { PodTable } from '@/components/pod-table'
import { ServiceTable } from '@/components/service-table'
import { Terminal } from '@/components/terminal'
import { VolumeTable } from '@/components/volume-table'
import { YamlEditor } from '@/components/yaml-editor'
import { CloneSet } from '@/types/k8s'
import type { PodWithMetrics } from '@/types/api'

export function CloneSetDetail(props: { namespace: string; name: string }) {
  const { namespace, name } = props
  const { t } = useTranslation()
  const [scaleReplicas, setScaleReplicas] = useState<number>(1)
  const [yamlContent, setYamlContent] = useState('')
  const [isSavingYaml, setIsSavingYaml] = useState(false)
  const [isScalePopoverOpen, setIsScalePopoverOpen] = useState(false)
  const [isRestartPopoverOpen, setIsRestartPopoverOpen] = useState(false)
  const [refreshKey, setRefreshKey] = useState(0)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [refreshInterval, setRefreshInterval] = useState<number>(0)
  const navigate = useNavigate()

  // Fetch cloneset data
  const {
    data: cloneSet,
    isLoading: isLoadingCloneSet,
    isError: isCloneSetError,
    error: cloneSetError,
    refetch: refetchCloneSet,
  } = useResource('clonesets', name, namespace, {
    refreshInterval,
  })

  const labelSelector = cloneSet?.spec?.selector.matchLabels
    ? Object.entries(cloneSet.spec.selector.matchLabels)
        .map(([key, value]) => `${key}=${value}`)
        .join(',')
    : undefined
  const { data: relatedPods } = useResources('pods', namespace, {
    labelSelector,
    refreshInterval,
    disable: !cloneSet?.spec?.selector.matchLabels,
  })

  // Fetch all services in the namespace
  const { data: allServices } = useResources('services', namespace, {
    refreshInterval,
  })

  // Filter services that are related to this CloneSet
  const relatedServices = useMemo(() => {
    if (!allServices || !cloneSet?.spec?.selector?.matchLabels) {
      return []
    }

    const cloneSetLabels = cloneSet.spec.selector.matchLabels
    
    return allServices.filter(service => {
      if (!service.spec?.selector) {
        return false
      }

      // Check if service selector matches CloneSet's pod labels
      const serviceSelector = service.spec.selector
      return Object.entries(serviceSelector).every(([key, value]) => {
        return cloneSetLabels[key] === value
      })
    })
  }, [allServices, cloneSet?.spec?.selector?.matchLabels])

  useEffect(() => {
    if (cloneSet) {
      setYamlContent(yaml.dump(cloneSet, { indent: 2 }))
      setScaleReplicas(cloneSet.spec?.replicas || 1)
    }
  }, [cloneSet])

  // Auto-reset refresh interval when cloneSet reaches stable state
  useEffect(() => {
    if (cloneSet) {
      const status = getCloneSetStatus(cloneSet)
      const isStable =
        status === 'Available' ||
        status === 'Scaled Down' ||
        status === 'Paused'

      if (isStable) {
        const timer = setTimeout(() => {
          setRefreshInterval(0)
        }, 2000)
        return () => clearTimeout(timer)
      } else {
        setRefreshInterval(1000)
      }
    }
  }, [cloneSet, refreshInterval])

  const handleRefresh = () => {
    setRefreshKey((prev) => prev + 1)
    refetchCloneSet()
  }

  const handleSaveYaml = async (content: CloneSet) => {
    setIsSavingYaml(true)
    try {
      await updateResource('clonesets', name, namespace, content)
      toast.success(t('common.yamlSaved'))
      setRefreshInterval(1000)
    } catch (error) {
      console.error('Failed to save YAML:', error)
      toast.error(
        `${t('common.yamlSaveError')}: ${
          error instanceof Error ? error.message : t('common.unknownError')
        }`
      )
    } finally {
      setIsSavingYaml(false)
    }
  }

  const handleYamlChange = (content: string) => {
    setYamlContent(content)
  }

  const handleScale = async () => {
    try {
      if (!cloneSet) return
      
      const updatedCloneSet = { ...cloneSet }
      if (updatedCloneSet.spec) {
        updatedCloneSet.spec.replicas = scaleReplicas
      }
      
      await updateResource('clonesets', name, namespace, updatedCloneSet)
      toast.success(t('openkruise.clonesets.scaleSuccess', { replicas: scaleReplicas }))
      setIsScalePopoverOpen(false)
      setRefreshInterval(1000)
    } catch (error) {
      console.error('Failed to scale CloneSet:', error)
      toast.error(
        `${t('openkruise.clonesets.scaleError')}: ${
          error instanceof Error ? error.message : t('common.unknownError')
        }`
      )
    }
  }

  const handleRestart = async () => {
    try {
      if (!cloneSet) return
      
      // Use the dedicated restart API instead of generic update
      const result = await restartCloneSet(namespace, name)
      
      toast.success(`${t('openkruise.clonesets.restartSuccess')} - ${result.message}`)
      
      // Show restart timestamp if available
      if (result.restartedAt) {
        console.log(`CloneSet restarted at: ${result.restartedAt}`)
      }
      
      // Start polling for updates to show the restart in progress
      setRefreshInterval(1000)
      
      // Stop polling after 30 seconds
      setTimeout(() => {
        setRefreshInterval(0)
      }, 30000)
      
    } catch (error) {
      console.error('Failed to restart CloneSet:', error)
      toast.error(
        `${t('openkruise.clonesets.restartError')}: ${
          error instanceof Error ? error.message : t('common.unknownError')
        }`
      )
    }
  }

  const handleDelete = async () => {
    setIsDeleting(true)
    try {
      await deleteResource('clonesets', name, namespace)
      toast.success(t('openkruise.clonesets.deleteSuccess'))
      navigate(`/clonesets`)
    } catch (error) {
      toast.error(
        `${t('openkruise.clonesets.deleteError')}: ${
          error instanceof Error ? error.message : t('common.unknownError')
        }`
      )
    } finally {
      setIsDeleting(false)
      setIsDeleteDialogOpen(false)
    }
  }

  const handleContainerUpdate = async (updatedContainer: any) => {
    if (!cloneSet) return

    try {
      const updatedCloneSet = { ...cloneSet }

      if (updatedCloneSet.spec?.template?.spec?.containers) {
        const containerIndex =
          updatedCloneSet.spec.template.spec.containers.findIndex(
            (c) => c.name === updatedContainer.name
          )

        if (containerIndex >= 0) {
          updatedCloneSet.spec.template.spec.containers[containerIndex] =
            updatedContainer

          await updateResource('clonesets', name, namespace, updatedCloneSet)
          toast.success(
            t('openkruise.clonesets.containerUpdateSuccess', { name: updatedContainer.name })
          )
          setRefreshInterval(1000)
        }
      }
    } catch (error) {
      console.error('Failed to update container:', error)
      toast.error(
        `${t('openkruise.clonesets.containerUpdateError')}: ${
          error instanceof Error ? error.message : t('common.unknownError')
        }`
      )
    }
  }

  if (isLoadingCloneSet) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-center gap-2">
              <IconLoader className="animate-spin" />
              <span>{t('common.loading')}</span>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (isCloneSetError || !cloneSet) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="pt-6">
            <div className="text-center text-destructive">
              {t('common.errorLoading')}:{' '}
              {cloneSetError?.message || t('common.notFound')}
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  const { status } = cloneSet
  const readyReplicas = status?.readyReplicas || 0
  const totalReplicas = status?.replicas || 0

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{name}</h1>
          <p className="text-muted-foreground">
            {t('common.namespace')}: <span className="font-medium">{namespace}</span>
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handleRefresh}>
            <IconRefresh className="w-4 h-4" />
            {t('common.refresh')}
          </Button>
          <Popover
            open={isScalePopoverOpen}
            onOpenChange={setIsScalePopoverOpen}
          >
            <PopoverTrigger asChild>
              <Button variant="outline" size="sm">
                <IconScale className="w-4 h-4" />
                {t('common.scale')}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-80" align="end">
              <div className="space-y-4">
                <div className="space-y-2">
                  <h4 className="font-medium">{t('openkruise.clonesets.scaleCloneSet')}</h4>
                  <p className="text-sm text-muted-foreground">
                    {t('openkruise.clonesets.scaleDescription')}
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="replicas">{t('openkruise.clonesets.replicas')}</Label>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-9 w-9 p-0"
                      onClick={() =>
                        setScaleReplicas(Math.max(0, scaleReplicas - 1))
                      }
                      disabled={scaleReplicas <= 0}
                    >
                      -
                    </Button>
                    <Input
                      id="replicas"
                      type="number"
                      min="0"
                      value={scaleReplicas}
                      onChange={(e) =>
                        setScaleReplicas(parseInt(e.target.value) || 0)
                      }
                      className="text-center"
                    />
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-9 w-9 p-0"
                      onClick={() => setScaleReplicas(scaleReplicas + 1)}
                    >
                      +
                    </Button>
                  </div>
                </div>
                <Button onClick={handleScale} className="w-full">
                  <IconScale className="w-4 h-4 mr-2" />
                  {t('common.scale')}
                </Button>
              </div>
            </PopoverContent>
          </Popover>
          {isOpenKruiseResourceRestartable('clonesets') && (
            <Popover
              open={isRestartPopoverOpen}
              onOpenChange={setIsRestartPopoverOpen}
            >
              <PopoverTrigger asChild>
                <Button variant="outline" size="sm">
                  <IconReload className="w-4 h-4" />
                  {t('common.restart')}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-80" align="end">
                <div className="space-y-4">
                  <div className="space-y-2">
                    <h4 className="font-medium">{t('openkruise.clonesets.restartCloneSet')}</h4>
                    <p className="text-sm text-muted-foreground">
                      {t('openkruise.clonesets.restartDescription')}
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      onClick={() => setIsRestartPopoverOpen(false)}
                      className="flex-1"
                    >
                      {t('common.cancel')}
                    </Button>
                    <Button
                      onClick={() => {
                        handleRestart()
                        setIsRestartPopoverOpen(false)
                      }}
                      className="flex-1"
                    >
                      <IconReload className="w-4 h-4 mr-2" />
                      {t('common.restart')}
                    </Button>
                  </div>
                </div>
              </PopoverContent>
            </Popover>
          )}
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setIsDeleteDialogOpen(true)}
            disabled={isDeleting}
          >
            <IconTrash className="w-4 h-4" />
            {t('common.delete')}
          </Button>
        </div>
      </div>
      
      {/* Tabs */}
      <ResponsiveTabs
        tabs={[
          {
            value: 'overview',
            label: t('common.overview'),
            content: (
              <div className="space-y-4">
                {/* Status Overview */}
                <Card>
                  <CardHeader>
                    <CardTitle>{t('common.statusOverview')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
                      <div className="flex items-center gap-3">
                        <div className="flex items-center gap-2">
                          <DeploymentStatusIcon
                            status={getCloneSetStatus(cloneSet)}
                          />
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground">
                            {t('common.status')}
                          </p>
                          <p className="text-sm font-medium">
                            {getCloneSetStatus(cloneSet)}
                          </p>
                        </div>
                      </div>

                      <div>
                        <p className="text-xs text-muted-foreground">
                          {t('common.readyReplicas')}
                        </p>
                        <p className="text-sm font-medium">
                          {readyReplicas} / {totalReplicas}
                        </p>
                      </div>

                      <div>
                        <p className="text-xs text-muted-foreground">
                          {t('common.updatedReplicas')}
                        </p>
                        <p className="text-sm font-medium">
                          {status?.updatedReplicas || 0}
                        </p>
                      </div>

                      <div>
                        <p className="text-xs text-muted-foreground">
                          {t('common.availableReplicas')}
                        </p>
                        <p className="text-sm font-medium">
                          {status?.availableReplicas || 0}
                        </p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                
                {/* CloneSet Info */}
                <Card>
                  <CardHeader>
                    <CardTitle>{t('openkruise.clonesets.cloneSetInformation')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                      <div>
                        <Label className="text-xs text-muted-foreground">
                          {t('openkruise.clonesets.created')}
                        </Label>
                        <p className="text-sm">
                          {formatDate(
                            cloneSet.metadata?.creationTimestamp || '',
                            true
                          )}
                        </p>
                      </div>
                      <div>
                        <Label className="text-xs text-muted-foreground">
                          {t('openkruise.clonesets.updateStrategy')}
                        </Label>
                        <p className="text-sm">
                          {cloneSet.spec?.updateStrategy?.type || 'RollingUpdate'}
                        </p>
                      </div>
                      <div>
                        <Label className="text-xs text-muted-foreground">
                          {t('openkruise.clonesets.replicas')}
                        </Label>
                        <p className="text-sm">
                          {cloneSet.spec?.replicas || 0}
                        </p>
                      </div>
                      <div>
                        <Label className="text-xs text-muted-foreground">
                          {t('common.selector')}
                        </Label>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {Object.entries(
                            cloneSet.spec?.selector?.matchLabels || {}
                          ).map(([key, value]) => (
                            <Badge
                              key={key}
                              variant="secondary"
                              className="text-xs"
                            >
                              {key}: {value}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    </div>
                    <LabelsAnno
                      labels={cloneSet.metadata?.labels || {}}
                      annotations={cloneSet.metadata?.annotations || {}}
                    />
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>
                      {t('common.containers')} (
                      {cloneSet.spec?.template?.spec?.containers?.length || 0}
                      )
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-6">
                      <div className="space-y-4">
                        {cloneSet.spec?.template?.spec?.containers?.map(
                          (container) => (
                            <ContainerTable
                              key={container.name}
                              container={container}
                              onContainerUpdate={handleContainerUpdate}
                            />
                          )
                        )}
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Conditions */}
                {status?.conditions && (
                  <Card>
                    <CardHeader>
                      <CardTitle>{t('common.conditions')}</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {status.conditions.map((condition, index) => (
                          <div
                            key={index}
                            className="flex items-center gap-3 p-2 border rounded"
                          >
                            <Badge
                              variant={
                                condition.status === 'True'
                                  ? 'default'
                                  : 'secondary'
                              }
                            >
                              {condition.type}
                            </Badge>
                            <span className="text-sm">{condition.message}</span>
                            <span className="text-xs text-muted-foreground ml-auto">
                              {formatDate(
                                condition.lastTransitionTime ||
                                  condition.lastUpdateTime ||
                                  ''
                              )}
                            </span>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                )}
              </div>
            ),
          },
          {
            value: 'yaml',
            label: t('common.yaml'),
            content: (
              <YamlEditor<'clonesets'>
                key={refreshKey}
                value={yamlContent}
                title={t('common.yamlConfiguration')}
                onSave={handleSaveYaml}
                onChange={handleYamlChange}
                isSaving={isSavingYaml}
              />
            ),
          },
          ...(relatedPods
            ? [
                {
                  value: 'pods',
                  label: (
                    <>
                      {t('common.pods')}
                      {relatedPods && (
                        <Badge variant="secondary">{relatedPods.length}</Badge>
                      )}
                    </>
                  ),
                  content: (
                    <PodTable
                      pods={relatedPods as PodWithMetrics[]}
                      labelSelector={labelSelector}
                    />
                  ),
                },
                {
                  value: 'logs',
                  label: t('common.logs'),
                  content: (
                    <div className="space-y-6">
                      <LogViewer
                        namespace={namespace}
                        pods={relatedPods as PodWithMetrics[]}
                        containers={
                          relatedPods?.[0]?.spec?.containers?.map(
                            (container) => ({
                              name: container.name,
                              image: container.image || '',
                            })
                          ) || []
                        }
                      />
                    </div>
                  ),
                },
                {
                  value: 'terminal',
                  label: t('common.terminal'),
                  content: (
                    <div className="space-y-6">
                      {relatedPods && relatedPods.length > 0 && (
                        <Terminal
                          namespace={namespace}
                          pods={relatedPods as PodWithMetrics[]}
                          containers={
                            relatedPods[0].spec?.containers?.map(
                              (container) => ({
                                name: container.name,
                                image: container.image || '',
                              })
                            ) || []
                          }
                        />
                      )}
                    </div>
                  ),
                },
              ]
            : []),
          {
            value: 'services',
            label: (
              <>
                {t('common.services')}{' '}
                {relatedServices && (
                  <Badge variant="secondary">
                    {relatedServices.length}
                  </Badge>
                )}
              </>
            ),
            content: (
              <ServiceTable
                services={relatedServices}
              />
            ),
          },
          ...(cloneSet.spec?.template?.spec?.volumes
            ? [
                {
                  value: 'volumes',
                  label: (
                    <>
                       {t('common.volumes')}
                      <Badge variant="secondary">
                        {cloneSet.spec.template.spec.volumes.length}
                      </Badge>
                    </>
                  ),
                  content: (
                    <VolumeTable
                      namespace={namespace}
                      volumes={cloneSet.spec?.template?.spec?.volumes}
                      containers={cloneSet.spec?.template?.spec?.containers}
                      isLoading={isLoadingCloneSet}
                    />
                  ),
                },
              ]
            : []),
          {
            value: 'events',
            label: t('common.events'),
            content: (
              <EventTable
                resource="clonesets"
                name={name}
                namespace={namespace}
              />
            ),
          },
          {
            value: 'monitor',
            label: t('common.monitor'),
            content: (
              <PodMonitoring
                namespace={namespace}
                pods={relatedPods as PodWithMetrics[]}
                containers={toSimpleContainer(relatedPods?.[0]?.spec?.initContainers, relatedPods?.[0]?.spec?.containers)}
                labelSelector={labelSelector}
              />
            ),
          },
        ]}
      />

      <DeleteConfirmationDialog
        open={isDeleteDialogOpen}
        onOpenChange={setIsDeleteDialogOpen}
        onConfirm={handleDelete}
        resourceName={name}
        resourceType={t('openkruise.clonesets.title')}
        namespace={namespace}
        isDeleting={isDeleting}
      />
    </div>
  )
} 
