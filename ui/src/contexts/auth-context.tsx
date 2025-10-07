/* eslint-disable react-refresh/only-export-components */
import {
  createContext,
  ReactNode,
  useContext,
  useEffect,
  useState,
} from 'react'

interface User {
  id: string
  username: string
  name: string
  avatar_url: string
  provider: string
  roles?: { name: string }[]

  isAdmin(): boolean
}

interface AuthContextType {
  user: User | null
  isLoading: boolean
  providers: string[]
  login: (provider?: string) => Promise<void>
  loginWithPassword: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  checkAuth: () => Promise<void>
  refreshToken: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

interface AuthProviderProps {
  children: ReactNode
}

const base = ''

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [providers, setProviders] = useState<string[]>([])

  const loadProviders = async () => {
    try {
      const response = await fetch(`${base}/api/auth/providers`)
      if (response.ok) {
        const data = await response.json()
        // Handle both wrapped and unwrapped responses
        const providerList = data.data?.providers || data.providers || []
        setProviders(providerList)
      }
    } catch (error) {
      console.error('Failed to load OAuth providers:', error)
    }
  }

  const checkAuth = async () => {
    try {
      const response = await fetch(`${base}/api/auth/user`, {
        credentials: 'include',
      })

      if (response.ok) {
        const data = await response.json()
        // Backend wraps response in { success: true, data: { user: {...} } }
        const user = data.data?.user as User
        if (user) {
          user.isAdmin = function () {
            return (
              this.roles?.some(
                (role: { name: string }) => role.name === 'admin'
              ) || false
            )
          }
          setUser(user)
        } else {
          setUser(null)
        }
      } else if (response.status === 401) {
        // Unauthorized - redirect to login
        setUser(null)
        setIsLoading(false)
        window.location.href = '/login'
        return
      } else {
        setUser(null)
      }
    } catch (error) {
      console.error('Auth check failed:', error)
      setUser(null)
    } finally {
      setIsLoading(false)
    }
  }

  const login = async (provider: string = 'github') => {
    try {
      const response = await fetch(
        `${base}/api/auth/login?provider=${provider}`,
        {
          credentials: 'include',
        }
      )

      if (response.ok) {
        const data = await response.json()
        window.location.href = data.auth_url
      } else {
        throw new Error('Failed to initiate login')
      }
    } catch (error) {
      console.error('Login failed:', error)
      throw error
    }
  }

  const loginWithPassword = async (username: string, password: string) => {
    try {
      const response = await fetch(`${base}/api/auth/login/password`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password }),
        credentials: 'include',
      })

      if (response.ok) {
        await checkAuth()
      } else {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Password login failed')
      }
    } catch (error) {
      console.error('Password login failed:', error)
      throw error
    }
  }

  const refreshToken = async () => {
    try {
      const response = await fetch(`${base}/api/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      })

      if (!response.ok) {
        throw new Error('Failed to refresh token')
      }
    } catch (error) {
      console.error('Token refresh failed:', error)
      // If refresh fails, redirect to login
      setUser(null)
      window.location.href = '/login'
    }
  }

  const logout = async () => {
    try {
      const response = await fetch(`${base}/api/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      })

      if (response.ok) {
        setUser(null)
        window.location.href = '/login'
      } else {
        throw new Error('Failed to logout')
      }
    } catch (error) {
      console.error('Logout failed:', error)
      throw error
    }
  }

  useEffect(() => {
    const initAuth = async () => {
      await loadProviders()
      await checkAuth()
    }
    initAuth()
  }, [])

  // Set up automatic token refresh
  useEffect(() => {
    if (!user) return
    const refreshKey = 'lastRefreshTokenAt'
    const lastRefreshAt = localStorage.getItem(refreshKey)
    const now = Date.now()

    // If the last refresh was more than 30 minutes ago, refresh immediately
    if (!lastRefreshAt || now - Number(lastRefreshAt) > 30 * 60 * 1000) {
      refreshToken()
      localStorage.setItem(refreshKey, String(now))
    }

    const refreshInterval = setInterval(
      () => {
        refreshToken()
        localStorage.setItem(refreshKey, String(Date.now()))
      },
      30 * 60 * 1000
    ) // Refresh every 30 minutes

    return () => clearInterval(refreshInterval)
  }, [user])

  const value = {
    user,
    isLoading,
    providers,
    login,
    loginWithPassword,
    logout,
    checkAuth,
    refreshToken,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
