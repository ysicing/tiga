import { useCallback, useMemo } from 'react'
import { createColumnHelper } from '@tanstack/react-table'
import { Secret } from 'kubernetes-types/core/v1'
import { Link } from 'react-router-dom'

import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { ResourceTable } from '@/components/resource-table'

export function SecretListPage() {
  // Define column helper outside of any hooks
  const columnHelper = createColumnHelper<Secret>()

  // Define columns for the secret table
  const columns = useMemo(
    () => [
      columnHelper.accessor('metadata.name', {
        header: 'Name',
        cell: ({ row }) => (
          <div className="font-medium text-blue-500 hover:underline">
            <Link
              to={`/secrets/${row.original.metadata!.namespace}/${
                row.original.metadata!.name
              }`}
            >
              {row.original.metadata!.name}
            </Link>
          </div>
        ),
      }),
      columnHelper.accessor('type', {
        header: 'Type',
        cell: ({ getValue }) => {
          const type = getValue() || 'Opaque'
          return <Badge variant="outline">{type}</Badge>
        },
      }),
      columnHelper.accessor('data', {
        header: 'Data Keys',
        cell: ({ getValue }) => {
          const data = getValue() || {}
          const keys = Object.keys(data)
          if (keys.length === 0) {
            return '-'
          }
          // Limit to first 5 keys for display
          return keys.length > 5 ? (
            <span className="text-muted-foreground">
              {keys.slice(0, 5).join(', ')}...
            </span>
          ) : (
            <span className="text-muted-foreground">{keys.join(', ')}</span>
          )
        },
      }),
      columnHelper.accessor('metadata.creationTimestamp', {
        header: 'Created',
        cell: ({ getValue }) => {
          const dateStr = formatDate(getValue() || '')

          return (
            <span className="text-muted-foreground text-sm">{dateStr}</span>
          )
        },
      }),
    ],
    [columnHelper]
  )

  // Custom filter for secret search
  const secretSearchFilter = useCallback((secret: Secret, query: string) => {
    const dataKeys = Object.keys(secret.data || {}).join(' ')
    const type = secret.type || ''

    return (
      secret.metadata!.name!.toLowerCase().includes(query) ||
      (secret.metadata!.namespace?.toLowerCase() || '').includes(query) ||
      type.toLowerCase().includes(query) ||
      dataKeys.toLowerCase().includes(query)
    )
  }, [])

  return (
    <ResourceTable
      resourceName="Secrets"
      columns={columns}
      clusterScope={false} // Secrets are namespace-scoped
      searchQueryFilter={secretSearchFilter}
    />
  )
}
