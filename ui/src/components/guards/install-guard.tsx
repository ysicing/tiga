import { ReactNode, useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { Navigate } from 'react-router-dom'

interface InitCheckResponse {
  initialized: boolean
  step: number
}

/**
 * InstallGuard - 路由守卫，确保未初始化时重定向到 /install
 */
export function InstallGuard({ children }: { children: ReactNode }) {
  const [isInstalled, setIsInstalled] = useState<boolean | null>(null)

  useEffect(() => {
    fetch('/api/v1/init_check')
      .then((res) => res.json())
      .then((data: InitCheckResponse) => {
        setIsInstalled(data.initialized)
      })
      .catch(() => {
        // 如果检查失败，假设未初始化
        setIsInstalled(false)
      })
  }, [])

  // 加载中
  if (isInstalled === null) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    )
  }

  // 未初始化，重定向到安装页面
  if (!isInstalled) {
    return <Navigate to="/install" replace />
  }

  // 已初始化，允许访问
  return <>{children}</>
}

/**
 * PreventReinstallGuard - 防止已初始化后再次访问 /install
 */
export function PreventReinstallGuard({ children }: { children: ReactNode }) {
  const [isInstalled, setIsInstalled] = useState<boolean | null>(null)

  useEffect(() => {
    fetch('/api/v1/init_check')
      .then((res) => res.json())
      .then((data: InitCheckResponse) => {
        setIsInstalled(data.initialized)
      })
      .catch(() => {
        setIsInstalled(false)
      })
  }, [])

  // 加载中
  if (isInstalled === null) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    )
  }

  // 已初始化，重定向到登录页
  if (isInstalled) {
    return <Navigate to="/login" replace />
  }

  // 未初始化，允许访问安装页面
  return <>{children}</>
}
