import { ClusterProvider } from '@/contexts/cluster-context'
import { Outlet } from 'react-router-dom'

import { SidebarProvider } from '@/components/ui/sidebar'
import { AppSidebar } from '@/components/app-sidebar'
import { GlobalSearch } from '@/components/global-search'
import {
  GlobalSearchProvider,
  useGlobalSearch,
} from '@/components/global-search-provider'
import { SiteHeader } from '@/components/site-header'

function GlobalSearchWrapper() {
  const { isOpen, closeSearch, openSearch } = useGlobalSearch()
  return (
    <GlobalSearch
      open={isOpen}
      onOpenChange={(open) => (open ? openSearch() : closeSearch())}
    />
  )
}

export function K8sLayout() {
  return (
    <ClusterProvider>
      <GlobalSearchProvider>
        <SidebarProvider>
          <AppSidebar variant="inset" />
          <div className="flex flex-1 flex-col">
            <SiteHeader showSearch={true} />
            <main className="flex-1 overflow-auto">
              <div className="container mx-auto p-6">
                <Outlet />
              </div>
            </main>
          </div>
        </SidebarProvider>
        <GlobalSearchWrapper />
      </GlobalSearchProvider>
    </ClusterProvider>
  )
}
