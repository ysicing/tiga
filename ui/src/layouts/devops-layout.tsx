import { Outlet } from 'react-router-dom'

import { SidebarProvider } from '@/components/ui/sidebar'
import { DevOpsSidebar } from '@/components/devops-sidebar'
import { SiteHeader } from '@/components/site-header'

export function DevOpsLayout() {
  return (
    <SidebarProvider>
      <DevOpsSidebar />
      <div className="flex flex-1 flex-col">
        <SiteHeader />
        <main className="flex-1 overflow-auto">
          <div className="container mx-auto p-6">
            <Outlet />
          </div>
        </main>
      </div>
    </SidebarProvider>
  )
}
