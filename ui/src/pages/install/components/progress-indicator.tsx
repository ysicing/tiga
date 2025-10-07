import { useTranslation } from 'react-i18next'
import { CheckCircle2 } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ProgressIndicatorProps {
  currentStep: number
  totalSteps?: number
}

const defaultSteps = [
  { key: 'database', label: 'install.steps.database' },
  { key: 'admin', label: 'install.steps.admin' },
  { key: 'settings', label: 'install.steps.settings' },
  { key: 'confirm', label: 'install.steps.confirm' },
]

export function ProgressIndicator({ currentStep, totalSteps = 4 }: ProgressIndicatorProps) {
  const { t } = useTranslation()

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-8">
        {defaultSteps.map((step, index) => {
          const isCompleted = index < currentStep
          const isCurrent = index === currentStep
          const isUpcoming = index > currentStep

          return (
            <div key={step.key} className="flex items-center flex-1">
              {/* Step Circle */}
              <div className="flex flex-col items-center">
                <div
                  className={cn(
                    'w-10 h-10 rounded-full flex items-center justify-center font-semibold transition-colors',
                    {
                      'bg-primary text-primary-foreground': isCurrent,
                      'bg-green-500 text-white': isCompleted,
                      'bg-muted text-muted-foreground': isUpcoming,
                    }
                  )}
                >
                  {isCompleted ? (
                    <CheckCircle2 className="h-6 w-6" />
                  ) : (
                    <span>{index + 1}</span>
                  )}
                </div>
                <div
                  className={cn('mt-2 text-sm font-medium text-center', {
                    'text-primary': isCurrent,
                    'text-muted-foreground': !isCurrent,
                  })}
                >
                  {t(step.label)}
                </div>
              </div>

              {/* Connector Line */}
              {index < totalSteps - 1 && (
                <div
                  className={cn('flex-1 h-1 mx-2 transition-colors', {
                    'bg-green-500': isCompleted,
                    'bg-muted': !isCompleted,
                  })}
                />
              )}
            </div>
          )
        })}
      </div>

      {/* Progress Bar */}
      <div className="w-full bg-muted rounded-full h-2">
        <div
          className="bg-primary h-2 rounded-full transition-all duration-300"
          style={{ width: `${((currentStep + 1) / totalSteps) * 100}%` }}
        />
      </div>
    </div>
  )
}
