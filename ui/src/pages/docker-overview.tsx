import {
  IconAlertCircle,
  IconBox,
  IconNetwork,
  IconStack,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function DockerOverview() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const sections = [
    {
      id: 'containers',
      title: 'docker.containers',
      description: 'docker.containers.description',
      icon: IconBox,
      path: '/docker/containers',
      color: 'bg-blue-500',
    },
    {
      id: 'images',
      title: 'docker.images',
      description: 'docker.images.description',
      icon: IconStack,
      path: '/docker/images',
      color: 'bg-green-500',
    },
    {
      id: 'networks',
      title: 'docker.networks',
      description: 'docker.networks.description',
      icon: IconNetwork,
      path: '/docker/networks',
      color: 'bg-purple-500',
    },
  ]

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">
            {t('docker.overview', 'Docker 概览')}
          </h1>
          <p className="text-muted-foreground mt-1">
            {t('docker.overview.description', '管理 Docker 容器、镜像和网络')}
          </p>
        </div>
      </div>

      <Alert>
        <IconAlertCircle className="h-4 w-4" />
        <AlertTitle>{t('common.comingSoon', '即将推出')}</AlertTitle>
        <AlertDescription>
          {t(
            'docker.comingSoon.description',
            'Docker 管理功能正在开发中，敬请期待。'
          )}
        </AlertDescription>
      </Alert>

      <div className="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
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
