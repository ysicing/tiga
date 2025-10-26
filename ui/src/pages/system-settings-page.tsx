import { useTranslation } from 'react-i18next'

import { ResponsiveTabs } from '@/components/ui/responsive-tabs'
import { OAuthProviderManagement } from '@/components/settings/oauth-provider-management'
import { VersionInfo } from '@/components/settings/version-info'

export function SystemSettingsPage() {
  const { t } = useTranslation()

  return (
    <div className="space-y-2">
      <div className="mb-4">
        <div className="flex items-center gap-3 mb-2">
          <h1 className="text-3xl">{t('system.settings.title', '全局配置')}</h1>
        </div>
        <p className="text-muted-foreground">
          {t('system.settings.description', '管理 OAuth 认证配置')}
        </p>
      </div>

      <ResponsiveTabs
        tabs={[
          {
            value: 'oauth',
            label: t('system.settings.tabs.oauth', 'OAuth'),
            content: <OAuthProviderManagement />,
          },
          {
            value: 'version',
            label: t('system.settings.tabs.version', '版本'),
            content: <VersionInfo />,
          },
        ]}
      />
    </div>
  )
}
