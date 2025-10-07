import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
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
import { SystemSettingsSchema, type SystemSettings, getDefaultSystemSettings } from '@/lib/schemas/install-schemas'
import { useInstall } from '@/contexts/install-context'

export function SettingsStep() {
  const { t } = useTranslation()
  const { state, setSettings, goToNextStep, goToPreviousStep } = useInstall()

  const form = useForm<SystemSettings>({
    resolver: zodResolver(SystemSettingsSchema),
    defaultValues: state.settings || getDefaultSystemSettings(),
  })

  const onSubmit = (values: SystemSettings) => {
    setSettings(values)
    goToNextStep()
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{t('install.settings.title')}</h2>
        <p className="text-muted-foreground">{t('install.settings.description')}</p>
      </div>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
          {/* Application Name */}
          <FormField
            control={form.control}
            name="app_name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.settings.appName')}</FormLabel>
                <FormControl>
                  <Input placeholder="Tiga Dashboard" {...field} />
                </FormControl>
                <FormDescription>{t('install.settings.appNameHelp')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Application Subtitle */}
          <FormField
            control={form.control}
            name="app_subtitle"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.settings.appSubtitle')}</FormLabel>
                <FormControl>
                  <Input placeholder="Access your DevOps dashboard" {...field} />
                </FormControl>
                <FormDescription>{t('install.settings.appSubtitleHelp')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Domain */}
          <FormField
            control={form.control}
            name="domain"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.settings.domain')}</FormLabel>
                <FormControl>
                  <Input placeholder="tiga.example.com" {...field} />
                </FormControl>
                <FormDescription>{t('install.settings.domainHelp')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* HTTP Port */}
          <FormField
            control={form.control}
            name="http_port"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.settings.httpPort')}</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    placeholder="12306"
                    {...field}
                    onChange={(e) => field.onChange(parseInt(e.target.value, 10))}
                  />
                </FormControl>
                <FormDescription>{t('install.settings.httpPortHelp')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Language */}
          <FormField
            control={form.control}
            name="language"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.settings.language')}</FormLabel>
                <Select onValueChange={field.onChange} defaultValue={field.value}>
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value="zh-CN">中文（简体）</SelectItem>
                    <SelectItem value="en-US">English</SelectItem>
                  </SelectContent>
                </Select>
                <FormDescription>{t('install.settings.languageHelp')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Enable Analytics */}
          <FormField
            control={form.control}
            name="enable_analytics"
            render={({ field }) => (
              <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-4">
                <FormControl>
                  <Checkbox
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
                <div className="space-y-1 leading-none">
                  <FormLabel>
                    {t('install.settings.enableAnalytics')}
                  </FormLabel>
                  <FormDescription>
                    {t('install.settings.enableAnalyticsHelp')}
                  </FormDescription>
                </div>
              </FormItem>
            )}
          />

          {/* Actions */}
          <div className="flex gap-4">
            <Button type="button" variant="outline" onClick={goToPreviousStep}>
              {t('install.actions.back')}
            </Button>

            <Button type="submit" className="ml-auto">
              {t('install.actions.next')}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}
