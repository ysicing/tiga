import Logo from '@/assets/logo.png'
import {
  IconBell,
  IconBucket,
  IconCloud,
  IconContainer,
  IconDatabase,
  IconServer2,
  IconSettings,
  IconWorldWww,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Toaster } from '@/components/ui/sonner'
import { Footer } from '@/components/footer'
import { LanguageToggle } from '@/components/language-toggle'
import { ModeToggle } from '@/components/mode-toggle'
import { UserMenu } from '@/components/user-menu'

// Define main subsystems (not individual instance types)
const SUBSYSTEMS = [
  {
    id: 'devops',
    name: 'subsystem.devops',
    description: 'subsystem.devops.description',
    icon: IconSettings,
    path: '/devops',
    color: 'bg-blue-500',
  },
  {
    id: 'vms',
    name: 'subsystem.hosts',
    description: 'subsystem.hosts.description',
    icon: IconServer2,
    path: '/vms',
    color: 'bg-indigo-500',
  },
  {
    id: 'dbs',
    name: 'subsystem.dbs',
    description: 'subsystem.dbs.description',
    icon: IconDatabase,
    path: '/dbs',
    color: 'bg-green-500',
  },
  {
    id: 'minio',
    name: 'subsystem.minio',
    description: 'subsystem.minio.description',
    icon: IconBucket,
    path: '/storage',
    color: 'bg-pink-500',
  },
  {
    id: 'kubernetes',
    name: 'subsystem.kubernetes',
    description: 'subsystem.kubernetes.description',
    icon: IconCloud,
    path: '/k8s',
    color: 'bg-purple-500',
  },
  {
    id: 'docker',
    name: 'subsystem.docker',
    description: 'subsystem.docker.description',
    icon: IconContainer,
    path: '/docker',
    color: 'bg-cyan-500',
  },
  {
    id: 'caddy',
    name: 'subsystem.caddy',
    description: 'subsystem.caddy.description',
    icon: IconWorldWww,
    path: '/webserver',
    color: 'bg-teal-500',
  },
  {
    id: 'monitoring',
    name: 'subsystem.monitoring',
    description: 'subsystem.monitoring.description',
    icon: IconBell,
    path: '/monitoring',
    color: 'bg-orange-500',
  },
]

export function OverviewDashboard() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const handleCardClick = (path: string) => {
    navigate(path)
  }

  return (
    <>
      <div className="flex flex-col min-h-screen">
        {/* Header */}
        <header className="sticky top-0 z-50 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b">
          <div className="flex h-16 items-center px-6 gap-4">
            <div className="flex items-center gap-2">
              <img src={Logo} alt="tiga Logo" className="h-8 w-8" />
              <span className="text-lg font-semibold">tiga</span>
            </div>
            <div className="ml-auto flex items-center gap-4">
              <LanguageToggle />
              <ModeToggle />
              <Separator orientation="vertical" className="h-6" />
              <UserMenu />
            </div>
          </div>
        </header>

        {/* Main Content */}
        <main className="flex-1 bg-gradient-to-br from-background via-background to-muted/20">
          <div className="container mx-auto p-8">
            <div className="flex flex-col gap-6">
              <div className="flex items-center justify-between">
                <div>
                  <h1 className="text-3xl font-bold bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">
                    {t('overview.title')}
                  </h1>
                  <p className="text-muted-foreground mt-2 text-lg">
                    {t('overview.selectSubsystem')}
                  </p>
                </div>
              </div>

              <div className="grid gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3">
                {SUBSYSTEMS.map((subsystem) => {
                  const Icon = subsystem.icon
                  return (
                    <Card
                      key={subsystem.id}
                      className="cursor-pointer transition-all hover:shadow-lg hover:scale-105 overflow-hidden"
                      onClick={() => handleCardClick(subsystem.path)}
                    >
                      <div className={`h-2 ${subsystem.color}`} />
                      <CardHeader className="pb-4">
                        <CardTitle className="flex items-center gap-3">
                          <Icon className="h-6 w-6" />
                          <span>{t(subsystem.name)}</span>
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <p className="text-sm text-muted-foreground">
                          {t(subsystem.description)}
                        </p>
                      </CardContent>
                    </Card>
                  )
                })}
              </div>
            </div>
          </div>
        </main>

        {/* Footer */}
        <Footer />
      </div>
      <Toaster />
    </>
  )
}
