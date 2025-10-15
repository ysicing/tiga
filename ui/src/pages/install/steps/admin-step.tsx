import { useState } from 'react'
import { useInstall } from '@/contexts/install-context'
import { zodResolver } from '@hookform/resolvers/zod'
import { Eye, EyeOff } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  AdminAccountSchema,
  type AdminAccount,
} from '@/lib/schemas/install-schemas'
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
import { Progress } from '@/components/ui/progress'

export function AdminStep() {
  const { t } = useTranslation()
  const { state, setAdmin, goToNextStep, goToPreviousStep } = useInstall()
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)

  const form = useForm<AdminAccount>({
    resolver: zodResolver(AdminAccountSchema),
    defaultValues: state.admin
      ? { ...state.admin, confirm_password: '' }
      : {
          username: '',
          password: '',
          confirm_password: '',
          email: '',
        },
  })

  const password = form.watch('password')

  // Calculate password strength
  const getPasswordStrength = (
    pwd: string
  ): { score: number; label: string; color: string } => {
    if (!pwd) return { score: 0, label: '', color: '' }

    let score = 0
    if (pwd.length >= 8) score += 25
    if (/[a-z]/.test(pwd)) score += 25
    if (/[A-Z]/.test(pwd)) score += 25
    if (/[0-9]/.test(pwd)) score += 25

    if (score <= 25)
      return {
        score,
        label: t('install.admin.passwordWeak'),
        color: 'bg-red-500',
      }
    if (score <= 50)
      return {
        score,
        label: t('install.admin.passwordFair'),
        color: 'bg-orange-500',
      }
    if (score <= 75)
      return {
        score,
        label: t('install.admin.passwordGood'),
        color: 'bg-yellow-500',
      }
    return {
      score,
      label: t('install.admin.passwordStrong'),
      color: 'bg-green-500',
    }
  }

  const passwordStrength = getPasswordStrength(password)

  const onSubmit = (values: AdminAccount) => {
    const { confirm_password, ...adminData } = values
    setAdmin(adminData)
    goToNextStep()
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{t('install.admin.title')}</h2>
        <p className="text-muted-foreground">
          {t('install.admin.description')}
        </p>
      </div>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
          {/* Username */}
          <FormField
            control={form.control}
            name="username"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.admin.username')}</FormLabel>
                <FormControl>
                  <Input placeholder="admin" {...field} />
                </FormControl>
                <FormDescription>
                  {t('install.admin.usernameHelp')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Email */}
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.admin.email')}</FormLabel>
                <FormControl>
                  <Input
                    type="email"
                    placeholder="admin@example.com"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* Password */}
          <FormField
            control={form.control}
            name="password"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.admin.password')}</FormLabel>
                <FormControl>
                  <div className="relative">
                    <Input
                      type={showPassword ? 'text' : 'password'}
                      placeholder="••••••••"
                      {...field}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                      onClick={() => setShowPassword(!showPassword)}
                    >
                      {showPassword ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </FormControl>
                <FormDescription>
                  {t('install.admin.passwordHelp')}
                </FormDescription>
                <FormMessage />

                {/* Password Strength Indicator */}
                {password && (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">
                        {t('install.admin.passwordStrength')}:
                      </span>
                      <span className="font-medium">
                        {passwordStrength.label}
                      </span>
                    </div>
                    <Progress value={passwordStrength.score} className="h-2" />
                  </div>
                )}
              </FormItem>
            )}
          />

          {/* Confirm Password */}
          <FormField
            control={form.control}
            name="confirm_password"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('install.admin.confirmPassword')}</FormLabel>
                <FormControl>
                  <div className="relative">
                    <Input
                      type={showConfirmPassword ? 'text' : 'password'}
                      placeholder="••••••••"
                      {...field}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                      onClick={() =>
                        setShowConfirmPassword(!showConfirmPassword)
                      }
                    >
                      {showConfirmPassword ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </FormControl>
                <FormMessage />
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
