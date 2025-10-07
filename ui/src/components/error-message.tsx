import { useEffect, useState } from 'react'
import { RotateCcw, ShieldX, XCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { isRBACError, translateError } from '@/lib/utils'

import { Button } from './ui/button'

interface ErrorMessageProps {
  resourceName: string
  error: Error | unknown
  fallbackKey?: string
  className?: string
  refetch: () => void
}

export function ErrorMessage({
  resourceName,
  refetch,
  error,
  fallbackKey = 'common.error',
}: ErrorMessageProps) {
  const { t } = useTranslation()
  const [isRBAC, setIsRBAC] = useState(false)
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (error instanceof Error) {
      setIsRBAC(isRBACError(error.message))
      setMessage(translateError(error, t))
    } else {
      setMessage(t(fallbackKey))
    }
  }, [error, t, fallbackKey])

  if (!error) {
    return null
  }

  return (
    <div className="h-72 flex flex-col items-center justify-center">
      <div className="mb-4">
        {isRBAC ? (
          <ShieldX className="h-16 w-16 text-amber-500" />
        ) : (
          <XCircle className="h-16 w-16 text-red-500" />
        )}
      </div>
      <h3
        className={`text-lg font-medium mb-1 ${isRBAC ? 'text-amber-600' : 'text-red-500'}`}
      >
        {t('resourceTable.errorLoading', {
          resourceName: resourceName.toLowerCase(),
        })}
      </h3>
      <p className="text-muted-foreground mb-4">{message}</p>
      <Button variant="outline" onClick={() => refetch()}>
        <RotateCcw className="h-4 w-4 mr-2" />
        {t('resourceTable.tryAgain')}
      </Button>
    </div>
  )
}
