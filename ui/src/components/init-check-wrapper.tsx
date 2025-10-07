import { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { useInitCheck } from '@/lib/api'

interface InitCheckWrapperProps {
  children: ReactNode
}

/**
 * Global initialization check wrapper
 *
 * This component wraps the entire application and checks if the system is initialized.
 * If not initialized, it redirects to the installation page.
 *
 * This is more efficient than using InitCheckRoute on individual routes because
 * it only makes one API call for the entire application instead of one per route.
 */
export function InitCheckWrapper({ children }: InitCheckWrapperProps) {
  const { data: initCheck, isLoading } = useInitCheck()
  const location = useLocation()

  // Show loading spinner while checking initialization status
  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-primary"></div>
      </div>
    )
  }

  // Allow access to /install route even if not initialized
  if (location.pathname === '/install') {
    return <>{children}</>
  }

  // Redirect to installation page if not initialized
  if (!initCheck?.initialized) {
    return <Navigate to="/install" replace />
  }

  // System is initialized, render the application
  return <>{children}</>
}
