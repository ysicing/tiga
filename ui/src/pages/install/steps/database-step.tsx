import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { CheckCircle2, XCircle, Loader2, AlertTriangle } from 'lucide-react'
import { DatabaseConfigSchema, type DatabaseConfig } from '@/lib/schemas/install-schemas'
import { useInstall } from '@/contexts/install-context'
import { installApi } from '@/services/install-api'
import { Checkbox } from '@/components/ui/checkbox'

export function DatabaseStep() {
  const { t } = useTranslation()
  const { state, setDatabase, goToNextStep } = useInstall()
  const [testResult, setTestResult] = useState<{
    success: boolean
    message: string
    hasExistingData?: boolean
  } | null>(null)
  const [isTesting, setIsTesting] = useState(false)
  const [confirmReinstall, setConfirmReinstall] = useState(false)

  const form = useForm<DatabaseConfig>({
    resolver: zodResolver(DatabaseConfigSchema),
    defaultValues: state.database || {
      type: 'postgresql',
      host: '127.0.0.1',
      port: 5432,
      database: 'tiga',
      username: 'tiga',
      password: '',
      ssl_mode: 'disable',
    },
  })

  const dbType = form.watch('type')

  const handleTestConnection = async () => {
    const values = form.getValues()

    // 先验证表单
    const isValid = await form.trigger()
    if (!isValid) return

    setIsTesting(true)
    // 保存之前的 hasExistingData 状态
    const previousHasExistingData = testResult?.hasExistingData
    setTestResult(null)

    try {
      // 如果之前检测到存在数据且现在确认重新安装，则添加 confirm_reinstall 参数
      const requestData = previousHasExistingData && confirmReinstall
        ? { ...values, confirm_reinstall: true }
        : values

      const response = await installApi.checkDatabase(requestData)

      if (response.success) {
        if (response.has_existing_data && !confirmReinstall) {
          setTestResult({
            success: false,
            message: t('install.database.existingDataFound', 'Existing data found. Please confirm to reinstall.'),
            hasExistingData: true,
          })
        } else {
          setTestResult({
            success: true,
            message: response.has_existing_data
              ? t('install.database.willReinstall', 'Will reinstall with existing data')
              : t('install.database.connectionSuccess', 'Connection successful'),
            hasExistingData: response.has_existing_data,
          })
        }
      } else {
        setTestResult({
          success: false,
          message: response.error || t('install.database.connectionFailed'),
        })
      }
    } catch (error) {
      setTestResult({
        success: false,
        message: error instanceof Error ? error.message : t('install.database.connectionError'),
      })
    } finally {
      setIsTesting(false)
    }
  }

  const onSubmit = (values: DatabaseConfig) => {
    setDatabase(values)
    goToNextStep()
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{t('install.database.title')}</h2>
        <p className="text-muted-foreground">{t('install.database.description')}</p>
      </div>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
          {/* Database Type */}
          <FormField
            control={form.control}
            name="type"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.database.type')}</FormLabel>
                <Select
                  onValueChange={(value) => {
                    field.onChange(value)
                    // Auto-update port based on database type
                    if (value === 'mysql') {
                      form.setValue('port', 3306)
                    } else if (value === 'postgresql') {
                      form.setValue('port', 5432)
                    }
                  }}
                  defaultValue={field.value}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder={t('install.database.selectType')} />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value="postgresql">PostgreSQL</SelectItem>
                    <SelectItem value="mysql">MySQL</SelectItem>
                    <SelectItem value="sqlite">SQLite</SelectItem>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Host (not for SQLite) */}
          {dbType !== 'sqlite' && (
            <FormField
              control={form.control}
              name="host"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('install.database.host')}</FormLabel>
                  <FormControl>
                    <Input placeholder="127.0.0.1" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          {/* Port (not for SQLite) */}
          {dbType !== 'sqlite' && (
            <FormField
              control={form.control}
              name="port"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('install.database.port')}</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      placeholder={dbType === 'mysql' ? '3306' : '5432'}
                      {...field}
                      onChange={(e) => field.onChange(parseInt(e.target.value, 10))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          {/* Database Name */}
          <FormField
            control={form.control}
            name="database"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.database.database')}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={dbType === 'sqlite' ? '/path/to/tiga.db' : 'tiga'}
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {dbType === 'sqlite'
                    ? t('install.database.sqlitePath')
                    : t('install.database.databaseName')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Username (not for SQLite) */}
          {dbType !== 'sqlite' && (
            <FormField
              control={form.control}
              name="username"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('install.database.username')}</FormLabel>
                  <FormControl>
                    <Input placeholder="tiga" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          {/* Password (not for SQLite) */}
          {dbType !== 'sqlite' && (
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('install.database.password')}</FormLabel>
                  <FormControl>
                    <Input type="password" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          {/* Test Connection Result */}
          {testResult && (
            <Alert variant={testResult.success ? 'default' : 'destructive'}>
              <div className="flex items-start gap-2">
                {testResult.success ? (
                  <CheckCircle2 className="h-4 w-4 mt-0.5" />
                ) : testResult.hasExistingData ? (
                  <AlertTriangle className="h-4 w-4 mt-0.5 text-amber-500" />
                ) : (
                  <XCircle className="h-4 w-4 mt-0.5" />
                )}
                <div className="flex-1">
                  <AlertDescription>{testResult.message}</AlertDescription>
                  {testResult.hasExistingData && !testResult.success && (
                    <div className="mt-3 flex items-center space-x-2">
                      <Checkbox
                        id="confirm-reinstall"
                        checked={confirmReinstall}
                        onCheckedChange={(checked) => setConfirmReinstall(!!checked)}
                      />
                      <label
                        htmlFor="confirm-reinstall"
                        className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                      >
                        {t('install.database.confirmReinstall', 'I confirm to reinstall and overwrite existing data')}
                      </label>
                    </div>
                  )}
                </div>
              </div>
            </Alert>
          )}

          {/* Actions */}
          <div className="flex gap-4">
            <Button
              type="button"
              variant="outline"
              onClick={handleTestConnection}
              disabled={isTesting}
            >
              {isTesting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {testResult?.hasExistingData && confirmReinstall
                ? t('install.database.retestConnection', 'Retest Connection')
                : t('install.database.testConnection')}
            </Button>

            <Button
              type="submit"
              disabled={!testResult?.success}
              className="ml-auto"
            >
              {t('install.actions.next')}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}
