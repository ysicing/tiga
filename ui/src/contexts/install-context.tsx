import {
  createContext,
  ReactNode,
  useContext,
  useEffect,
  useState,
} from 'react'

import type {
  AdminAccount,
  CheckDBResponse,
  DatabaseConfig,
  SystemSettings,
} from '@/lib/schemas/install-schemas'

// T035: InstallContext 状态管理

export interface InstallState {
  currentStep: number
  database: DatabaseConfig | null
  admin: Omit<AdminAccount, 'confirm_password'> | null
  settings: SystemSettings | null
  isTestingConnection: boolean
  connectionTestResult: CheckDBResponse | null
}

interface InstallContextValue {
  state: InstallState
  setCurrentStep: (step: number) => void
  setDatabase: (database: DatabaseConfig) => void
  setAdmin: (admin: Omit<AdminAccount, 'confirm_password'>) => void
  setSettings: (settings: SystemSettings) => void
  setIsTestingConnection: (testing: boolean) => void
  setConnectionTestResult: (result: CheckDBResponse | null) => void
  clearState: () => void
  goToNextStep: () => void
  goToPreviousStep: () => void
}

const initialState: InstallState = {
  currentStep: 0,
  database: null,
  admin: null,
  settings: null,
  isTestingConnection: false,
  connectionTestResult: null,
}

const STORAGE_KEY = 'install-state'

const InstallContext = createContext<InstallContextValue | null>(null)

export function InstallProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<InstallState>(() => {
    // 从 sessionStorage 恢复状态
    if (typeof window !== 'undefined') {
      const saved = sessionStorage.getItem(STORAGE_KEY)
      if (saved) {
        try {
          return JSON.parse(saved)
        } catch {
          return initialState
        }
      }
    }
    return initialState
  })

  // 保存状态到 sessionStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify(state))
    }
  }, [state])

  const setCurrentStep = (step: number) => {
    setState((prev) => ({ ...prev, currentStep: step }))
  }

  const setDatabase = (database: DatabaseConfig) => {
    setState((prev) => ({ ...prev, database }))
  }

  const setAdmin = (admin: Omit<AdminAccount, 'confirm_password'>) => {
    setState((prev) => ({ ...prev, admin }))
  }

  const setSettings = (settings: SystemSettings) => {
    setState((prev) => ({ ...prev, settings }))
  }

  const setIsTestingConnection = (testing: boolean) => {
    setState((prev) => ({ ...prev, isTestingConnection: testing }))
  }

  const setConnectionTestResult = (result: CheckDBResponse | null) => {
    setState((prev) => ({ ...prev, connectionTestResult: result }))
  }

  const clearState = () => {
    setState(initialState)
    if (typeof window !== 'undefined') {
      sessionStorage.removeItem(STORAGE_KEY)
    }
  }

  const goToNextStep = () => {
    setState((prev) => ({ ...prev, currentStep: prev.currentStep + 1 }))
  }

  const goToPreviousStep = () => {
    setState((prev) => ({
      ...prev,
      currentStep: Math.max(0, prev.currentStep - 1),
    }))
  }

  const value: InstallContextValue = {
    state,
    setCurrentStep,
    setDatabase,
    setAdmin,
    setSettings,
    setIsTestingConnection,
    setConnectionTestResult,
    clearState,
    goToNextStep,
    goToPreviousStep,
  }

  return (
    <InstallContext.Provider value={value}>{children}</InstallContext.Provider>
  )
}

export function useInstall() {
  const context = useContext(InstallContext)
  if (!context) {
    throw new Error('useInstall must be used within InstallProvider')
  }
  return context
}
