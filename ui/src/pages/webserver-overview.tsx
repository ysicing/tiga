import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { IconWorldWww, IconFileCode, IconCertificate, IconChartBar, IconAlertCircle } from '@tabler/icons-react'

export function WebServerOverview() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const sections = [
    {
      id: 'sites',
      title: 'webserver.sites',
      description: 'webserver.sites.description',
      icon: IconWorldWww,
      path: '/webserver/sites',
      color: 'bg-blue-500',
    },
    {
      id: 'config',
      title: 'webserver.config',
      description: 'webserver.config.description',
      icon: IconFileCode,
      path: '/webserver/config',
      color: 'bg-green-500',
    },
    {
      id: 'certificates',
      title: 'webserver.certificates',
      description: 'webserver.certificates.description',
      icon: IconCertificate,
      path: '/webserver/certificates',
      color: 'bg-purple-500',
    },
    {
      id: 'metrics',
      title: 'webserver.metrics',
      description: 'webserver.metrics.description',
      icon: IconChartBar,
      path: '/webserver/metrics',
      color: 'bg-orange-500',
    },
  ]

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('webserver.overview', 'Web 服务器概览')}</h1>
          <p className="text-muted-foreground mt-1">
            {t('webserver.overview.description', 'Caddy Web 服务器和反向代理管理')}
          </p>
        </div>
      </div>

      <Alert>
        <IconAlertCircle className="h-4 w-4" />
        <AlertTitle>{t('common.comingSoon', '即将推出')}</AlertTitle>
        <AlertDescription>
          {t('webserver.comingSoon.description', 'Caddy Web 服务器管理功能正在开发中，敬请期待。')}
        </AlertDescription>
      </Alert>

      <div className="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-2">
        {sections.map((section) => {
          const Icon = section.icon
          return (
            <Card
              key={section.id}
              className="cursor-pointer transition-all hover:shadow-lg hover:scale-105 overflow-hidden opacity-60"
              onClick={() => navigate(section.path)}
            >
              <div className={`h-2 ${section.color}`} />
              <CardHeader className="pb-4">
                <CardTitle className="flex items-center gap-3">
                  <Icon className="h-6 w-6" />
                  <span>{t(section.title)}</span>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  {t(section.description)}
                </p>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
