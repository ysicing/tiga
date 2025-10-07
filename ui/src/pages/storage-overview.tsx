import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { IconBucket, IconUsers, IconShield, IconChartBar, IconAlertCircle } from '@tabler/icons-react'

export function StorageOverview() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const sections = [
    {
      id: 'buckets',
      title: 'storage.buckets',
      description: 'storage.buckets.description',
      icon: IconBucket,
      path: '/storage/buckets',
      color: 'bg-blue-500',
    },
    {
      id: 'users',
      title: 'storage.users',
      description: 'storage.users.description',
      icon: IconUsers,
      path: '/storage/users',
      color: 'bg-green-500',
    },
    {
      id: 'policies',
      title: 'storage.policies',
      description: 'storage.policies.description',
      icon: IconShield,
      path: '/storage/policies',
      color: 'bg-purple-500',
    },
    {
      id: 'metrics',
      title: 'storage.metrics',
      description: 'storage.metrics.description',
      icon: IconChartBar,
      path: '/storage/metrics',
      color: 'bg-orange-500',
    },
  ]

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t('storage.overview', '对象存储概览')}</h1>
          <p className="text-muted-foreground mt-1">
            {t('storage.overview.description', 'MinIO 对象存储服务管理')}
          </p>
        </div>
      </div>

      <Alert>
        <IconAlertCircle className="h-4 w-4" />
        <AlertTitle>{t('common.comingSoon', '即将推出')}</AlertTitle>
        <AlertDescription>
          {t('storage.comingSoon.description', 'MinIO 对象存储管理功能正在开发中，敬请期待。')}
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
