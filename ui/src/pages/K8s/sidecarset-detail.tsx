import { useEffect, useState, useMemo } from 'react'
import { IconRefresh, IconTrash } from '@tabler/icons-react'
import * as yaml from 'js-yaml'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  deleteResource,
  updateResource,
  useResource,
  useResources,
} from '@/lib/api'
import { getSidecarSetStatus } from '@/lib/k8s'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ResponsiveTabs } from '@/components/ui/responsive-tabs'
import { DeleteConfirmationDialog } from '@/components/delete-confirmation-dialog'
import { EventTable } from '@/components/event-table'
import { LabelsAnno } from '@/components/lables-anno'
import { PodTable } from '@/components/pod-table'
import { YamlEditor } from '@/components/yaml-editor'
import { SidecarSet } from '@/types/k8s'

export function SidecarSetDetail(props: { name: string }) {
  const { name } = props
  const { t } = useTranslation()
  const [yamlContent, setYamlContent] = useState('')
  const [isSavingYaml, setIsSavingYaml] = useState(false)
  const [refreshKey, setRefreshKey] = useState(0)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const navigate = useNavigate()

  // SidecarSet is cluster-scoped
  const {
    data: sidecarSet,
    isLoading,
    isError,
    error,
    refetch,
  } = useResource('sidecarsets', name, '')

  const labelSelector = (sidecarSet as SidecarSet)?.spec?.selector?.matchLabels
    ? Object.entries((sidecarSet as SidecarSet).spec.selector.matchLabels!)
        .map(([key, value]) => `${key}=${value}`)
        .join(',')
    : undefined

  // For SidecarSet, pods can be in any namespace
  const { data: allPods } = useResources('pods', '', {
    labelSelector,
    disable: !(sidecarSet as SidecarSet)?.spec?.selector?.matchLabels,
  })

  useEffect(() => {
    if (sidecarSet) {
      setYamlContent(yaml.dump(sidecarSet, { indent: 2 }))
    }
  }, [sidecarSet])

  const handleRefresh = () => {
    setRefreshKey((prev) => prev + 1)
    refetch()
  }

  const handleSaveYaml = async (content: SidecarSet) => {
    setIsSavingYaml(true)
    try {
      await updateResource('sidecarsets', name, '', content)
      toast.success(t('common.yamlSaved'))
    } catch (error) {
      console.error('Failed to save YAML:', error)
      toast.error(t('common.yamlSaveFailed'))
    } finally {
      setIsSavingYaml(false)
    }
  }

  const handleDelete = async () => {
    setIsDeleting(true)
    try {
      await deleteResource('sidecarsets', name, '')
      toast.success(t('common.deleteSuccess'))
      navigate('/k8s/sidecarsets')
    } catch (error) {
      console.error('Failed to delete:', error)
      toast.error(t('common.deleteFailed'))
      setIsDeleting(false)
    }
  }

  if (isLoading) {
    return <div>{t('common.loading')}</div>
  }

  if (isError) {
    return <div>Error: {(error as Error)?.message}</div>
  }

  if (!sidecarSet) {
    return <div>{t('common.notFound')}</div>
  }

  const typedSidecarSet = sidecarSet as SidecarSet
  const status = getSidecarSetStatus(typedSidecarSet)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {typedSidecarSet.metadata.name}
          </h1>
          <p className="text-muted-foreground">
            SidecarSet (Cluster-scoped)
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={handleRefresh} variant="outline" size="sm">
            <IconRefresh className="mr-2 h-4 w-4" />
            {t('common.refresh')}
          </Button>
          <Button
            onClick={() => setIsDeleteDialogOpen(true)}
            variant="destructive"
            size="sm"
          >
            <IconTrash className="mr-2 h-4 w-4" />
            {t('common.delete')}
          </Button>
        </div>
      </div>

      {/* Status Card */}
      <Card>
        <CardHeader>
          <CardTitle>{t('common.status')}</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                {t('common.status')}
              </dt>
              <dd className="mt-1">
                <Badge variant="outline">{status}</Badge>
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                {t('openkruise.sidecarsets.matched')}
              </dt>
              <dd className="mt-1 text-sm">
                {typedSidecarSet.status?.matchedPods || 0}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                {t('openkruise.sidecarsets.updated')}
              </dt>
              <dd className="mt-1 text-sm">
                {typedSidecarSet.status?.updatedPods || 0}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                {t('openkruise.sidecarsets.ready')}
              </dt>
              <dd className="mt-1 text-sm">
                {typedSidecarSet.status?.readyPods || 0}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                {t('common.created')}
              </dt>
              <dd className="mt-1 text-sm">
                {formatDate(typedSidecarSet.metadata.creationTimestamp || '')}
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      {/* Tabs */}
      <ResponsiveTabs
        defaultValue="pods"
        tabs={[
          {
            value: 'pods',
            label: t('nav.pods'),
            content: (
              <PodTable
                key={`pods-${refreshKey}`}
                pods={allPods || []}
                showNamespace={true}
              />
            ),
          },
          {
            value: 'events',
            label: t('common.events'),
            content: (
              <EventTable
                key={`events-${refreshKey}`}
                namespace=""
                fieldSelector={`involvedObject.name=${name},involvedObject.kind=SidecarSet`}
              />
            ),
          },
          {
            value: 'labels',
            label: t('common.labels'),
            content: (
              <LabelsAnno
                labels={typedSidecarSet.metadata.labels || {}}
                annotations={typedSidecarSet.metadata.annotations || {}}
              />
            ),
          },
          {
            value: 'yaml',
            label: 'YAML',
            content: (
              <YamlEditor
                key={`yaml-${refreshKey}`}
                value={yamlContent}
                onChange={setYamlContent}
                onSave={handleSaveYaml}
                isSaving={isSavingYaml}
              />
            ),
          },
        ]}
      />

      {/* Delete Dialog */}
      <DeleteConfirmationDialog
        open={isDeleteDialogOpen}
        onOpenChange={setIsDeleteDialogOpen}
        onConfirm={handleDelete}
        title={t('common.deleteConfirmation')}
        description={t('common.deleteWarning', {
          name: typedSidecarSet.metadata.name,
        })}
        isDeleting={isDeleting}
      />
    </div>
  )
}
