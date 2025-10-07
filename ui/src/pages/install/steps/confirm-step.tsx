import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Loader2, Database, User, Settings, AlertCircle } from 'lucide-react'
import { useInstall } from '@/contexts/install-context'
import { installApi } from '@/services/install-api'

export function ConfirmStep() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { state, goToPreviousStep, clearState } = useInstall()
  const [isInstalling, setIsInstalling] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleFinalize = async () => {
    if (!state.database || !state.admin || !state.settings) {
      setError(t('install.confirm.incompleteData'))
      return
    }

    setIsInstalling(true)
    setError(null)

    try {
      const response = await installApi.finalize({
        database: state.database,
        admin: state.admin,
        settings: state.settings,
      })

      if (response.success && response.session_token) {
        // 保存 session token
        localStorage.setItem('auth_token', response.session_token)

        // 清除安装状态
        clearState()

        // 如果有重定向 URL，使用它（包含配置的端口）
        if (response.redirect_url) {
          // 如果需要重启，显示提示信息
          if (response.needs_restart && response.restart_message) {
            alert(response.restart_message)
          }
          // 重定向到登录页（使用完整 URL 包含端口）
          window.location.href = response.redirect_url
        } else {
          // 后备方案：跳转到登录页
          navigate('/login')
        }
      } else {
        setError(response.error || t('install.confirm.installationFailed'))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t('install.confirm.installationError'))
    } finally {
      setIsInstalling(false)
    }
  }

  if (!state.database || !state.admin || !state.settings) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>{t('install.confirm.incompleteData')}</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{t('install.confirm.title')}</h2>
        <p className="text-muted-foreground">{t('install.confirm.description')}</p>
      </div>

      {/* Database Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            {t('install.confirm.databaseConfig')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="grid grid-cols-2 gap-2">
            <div className="text-sm text-muted-foreground">{t('install.database.type')}:</div>
            <div className="text-sm font-medium">{state.database.type}</div>

            {state.database.type !== 'sqlite' && (
              <>
                <div className="text-sm text-muted-foreground">{t('install.database.host')}:</div>
                <div className="text-sm font-medium">{state.database.host}</div>

                <div className="text-sm text-muted-foreground">{t('install.database.port')}:</div>
                <div className="text-sm font-medium">{state.database.port}</div>

                <div className="text-sm text-muted-foreground">
                  {t('install.database.username')}:
                </div>
                <div className="text-sm font-medium">{state.database.username}</div>

                <div className="text-sm text-muted-foreground">
                  {t('install.database.password')}:
                </div>
                <div className="text-sm font-medium">********</div>
              </>
            )}

            <div className="text-sm text-muted-foreground">{t('install.database.database')}:</div>
            <div className="text-sm font-medium">{state.database.database}</div>
          </div>
        </CardContent>
      </Card>

      {/* Admin Account */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <User className="h-5 w-5" />
            {t('install.confirm.adminAccount')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="grid grid-cols-2 gap-2">
            <div className="text-sm text-muted-foreground">{t('install.admin.username')}:</div>
            <div className="text-sm font-medium">{state.admin.username}</div>

            <div className="text-sm text-muted-foreground">{t('install.admin.email')}:</div>
            <div className="text-sm font-medium">{state.admin.email}</div>

            <div className="text-sm text-muted-foreground">{t('install.admin.password')}:</div>
            <div className="text-sm font-medium">********</div>
          </div>
        </CardContent>
      </Card>

      {/* System Settings */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="h-5 w-5" />
            {t('install.confirm.systemSettings')}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <div className="grid grid-cols-2 gap-2">
            <div className="text-sm text-muted-foreground">{t('install.settings.appName')}:</div>
            <div className="text-sm font-medium">{state.settings.app_name}</div>

            <div className="text-sm text-muted-foreground">{t('install.settings.domain')}:</div>
            <div className="text-sm font-medium">{state.settings.domain}</div>

            <div className="text-sm text-muted-foreground">{t('install.settings.httpPort')}:</div>
            <div className="text-sm font-medium">{state.settings.http_port}</div>

            <div className="text-sm text-muted-foreground">{t('install.settings.language')}:</div>
            <div className="text-sm font-medium">
              {state.settings.language === 'zh-CN' ? '中文（简体）' : 'English'}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Error Message */}
      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Actions */}
      <div className="flex gap-4">
        <Button type="button" variant="outline" onClick={goToPreviousStep} disabled={isInstalling}>
          {t('install.actions.back')}
        </Button>

        <Button onClick={handleFinalize} disabled={isInstalling} className="ml-auto">
          {isInstalling && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {t('install.actions.complete')}
        </Button>
      </div>
    </div>
  )
}
