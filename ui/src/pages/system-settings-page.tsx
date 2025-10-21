import { useTranslation } from 'react-i18next'

import { ResponsiveTabs } from '@/components/ui/responsive-tabs'
import { ClusterManagement } from '@/components/settings/cluster-management'
import { OAuthProviderManagement } from '@/components/settings/oauth-provider-management'
import { RBACManagement } from '@/components/settings/rbac-management'

export function SystemSettingsPage() {
  const { t } = useTranslation()

  return (
    <div className="space-y-2">
      <div className="mb-4">
        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-3xl">{t('system.settings.title', '全局配置')}</h1>
        </div>
        <p className="text-muted-foreground">
          {t('system.settings.description', '管理集群、OAuth 和权限配置')}
        </p>
      </div>

      <ResponsiveTabs
        tabs={[
          {
            value: 'clusters',
            label: t('system.settings.tabs.clusters', 'Cluster'),
            content: <ClusterManagement />,
          },
          {
            value: 'oauth',
            label: t('system.settings.tabs.oauth', 'OAuth'),
            content: <OAuthProviderManagement />,
          },
          {
            value: 'rbac',
            label: t('system.settings.tabs.rbac', 'RBAC'),
            content: <RBACManagement />,
          },
        ]}
      />
    </div>
  )
}
