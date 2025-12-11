import { useCallback, useMemo } from 'react'
import { createColumnHelper } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

import { getSidecarSetStatus } from '@/lib/k8s'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { ResourceTable } from '@/components/resource-table'
import { SidecarSet } from '@/types/k8s'

export function SidecarSetListPage() {
  const { t } = useTranslation()

  const columnHelper = createColumnHelper<SidecarSet>()

  const columns = useMemo(
    () => [
      columnHelper.accessor('metadata.name', {
        header: t('common.name'),
        cell: ({ row }) => (
          <div className="font-medium text-blue-500 hover:underline">
            <Link
              to={`/k8s/sidecarsets/${row.original.metadata.name}`}
            >
              {row.original.metadata.name}
            </Link>
          </div>
        ),
      }),
      columnHelper.accessor((row) => row.status, {
        id: 'matched',
        header: t('openkruise.sidecarsets.matched', 'Matched'),
        cell: ({ row }) => {
          const matched = row.original.status?.matchedPods || 0
          return <div>{matched}</div>
        },
      }),
      columnHelper.accessor((row) => row.status, {
        id: 'updated',
        header: t('openkruise.sidecarsets.updated', 'Updated'),
        cell: ({ row }) => {
          const updated = row.original.status?.updatedPods || 0
          return <div>{updated}</div>
        },
      }),
      columnHelper.accessor((row) => row.status, {
        id: 'ready',
        header: t('openkruise.sidecarsets.ready', 'Ready'),
        cell: ({ row }) => {
          const ready = row.original.status?.readyPods || 0
          return <div>{ready}</div>
        },
      }),
      columnHelper.accessor('status.conditions', {
        header: t('common.status'),
        cell: ({ row }) => {
          const status = getSidecarSetStatus(row.original)
          const variantMap: Record<string, 'default' | 'destructive' | 'outline' | 'secondary'> = {
            'Ready': 'secondary',
            'NotReady': 'destructive',
            'Updating': 'default',
            'Unknown': 'outline',
          }
          return (
            <Badge variant={variantMap[status] || 'outline'}>
              {status}
            </Badge>
          )
        },
      }),
      columnHelper.accessor('metadata.creationTimestamp', {
        header: t('common.created'),
        cell: ({ getValue }) => {
          const dateStr = formatDate(getValue() || '')
          return (
            <span className="text-muted-foreground text-sm">{dateStr}</span>
          )
        },
      }),
    ],
    [columnHelper, t]
  )

  const sidecarSetSearchFilter = useCallback(
    (sidecarSet: SidecarSet, query: string) => {
      return sidecarSet.metadata.name.toLowerCase().includes(query)
    },
    []
  )

  return (
    <ResourceTable
      resourceName={t('openkruise.sidecarsets.title', 'SidecarSets')}
      resourceType="sidecarsets"
      columns={columns}
      searchQueryFilter={sidecarSetSearchFilter}
    />
  )
}
