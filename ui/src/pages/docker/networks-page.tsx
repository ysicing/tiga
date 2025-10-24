import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { IconRefresh } from '@tabler/icons-react'

import { Button } from '@/components/ui/button'
import { NetworkList } from '@/components/docker/network-list'

export function NetworksPage() {
  const { t } = useTranslation()
  const { id: instanceId } = useParams<{ id: string }>()
  const [refreshKey, setRefreshKey] = useState(0)

  const handleRefresh = () => {
    setRefreshKey((prev) => prev + 1)
  }

  if (!instanceId) {
    return (
      <div className="flex items-center justify-center h-64 border border-dashed rounded-lg">
        <p className="text-muted-foreground">
          {t('docker.invalidInstance', '无效的实例 ID')}
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            {t('docker.networks', 'Docker 网络')}
          </h1>
          <p className="text-muted-foreground mt-1">
            {t('docker.networks.description', '管理 Docker 网络')}
          </p>
        </div>
        <Button onClick={handleRefresh} variant="outline" size="sm">
          <IconRefresh className="h-4 w-4 mr-2" />
          {t('common.refresh', '刷新')}
        </Button>
      </div>

      {/* Network List */}
      <NetworkList instanceId={instanceId} key={refreshKey} />
    </div>
  )
}
