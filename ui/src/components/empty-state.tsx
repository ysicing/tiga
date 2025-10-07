import { Button } from '@/components/ui/button'
import { PackageX } from 'lucide-react'
import { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export interface EmptyStateProps {
  message: string
  actionText?: string
  onAction?: () => void
  icon?: ReactNode
  className?: string
}

export function EmptyState({
  message,
  actionText,
  onAction,
  icon,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center py-12 px-4 text-center',
        className
      )}
    >
      <div className="mb-4 text-muted-foreground">
        {icon || <PackageX className="h-16 w-16" />}
      </div>
      <p className="text-lg font-medium text-muted-foreground mb-4">{message}</p>
      {onAction && actionText && (
        <Button onClick={onAction} variant="default">
          {actionText}
        </Button>
      )}
    </div>
  )
}
