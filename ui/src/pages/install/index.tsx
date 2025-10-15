import { useEffect } from 'react'
import { InstallProvider, useInstall } from '@/contexts/install-context'
import { installApi } from '@/services/install-api'
import { Languages } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import { ProgressIndicator } from './components/progress-indicator'
import { AdminStep } from './steps/admin-step'
import { ConfirmStep } from './steps/confirm-step'
import { DatabaseStep } from './steps/database-step'
import { SettingsStep } from './steps/settings-step'

function InstallContent() {
  const { state } = useInstall()
  const navigate = useNavigate()
  const { i18n, t } = useTranslation()

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng)
  }

  useEffect(() => {
    // 检查是否已初始化
    installApi.checkStatus().then((status) => {
      if (status.installed) {
        navigate('/login')
      }
    })
  }, [navigate])

  const renderStep = () => {
    switch (state.currentStep) {
      case 0:
        return <DatabaseStep />
      case 1:
        return <AdminStep />
      case 2:
        return <SettingsStep />
      case 3:
        return <ConfirmStep />
      default:
        return <DatabaseStep />
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-background to-muted/20 flex items-center justify-center p-4">
      {/* Language Switcher - Top Right */}
      <div className="fixed top-4 right-4">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="icon">
              <Languages className="h-[1.2rem] w-[1.2rem]" />
              <span className="sr-only">{t('common.changeLanguage')}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => changeLanguage('zh-CN')}>
              中文（简体）
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => changeLanguage('en')}>
              English
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <Card className="w-full max-w-3xl">
        <CardHeader>
          <div className="text-center mb-6">
            <h1 className="text-3xl font-bold">{t('install.title')}</h1>
            <p className="text-muted-foreground mt-2">{t('install.welcome')}</p>
          </div>

          <ProgressIndicator currentStep={state.currentStep} totalSteps={4} />
        </CardHeader>

        <CardContent className="pt-6">{renderStep()}</CardContent>
      </Card>
    </div>
  )
}

export default function InstallPage() {
  return (
    <InstallProvider>
      <InstallContent />
    </InstallProvider>
  )
}
