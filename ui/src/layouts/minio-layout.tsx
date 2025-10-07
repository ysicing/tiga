import { Outlet, useParams } from 'react-router-dom'
import { SidebarProvider } from '@/components/ui/sidebar'
import { SiteHeader } from '@/components/site-header'
import { MinIOSidebar } from '@/components/minio-sidebar'
import { ClusterProvider } from '@/contexts/cluster-context'
import { GlobalSearchProvider, useGlobalSearch } from '@/components/global-search-provider'
import { GlobalSearch } from '@/components/global-search'

function GlobalSearchWrapper() {
  const { isOpen, closeSearch, openSearch } = useGlobalSearch()
  return <GlobalSearch open={isOpen} onOpenChange={(open) => open ? openSearch() : closeSearch()} />
}

export function MinIOLayout() {
  const { instanceId } = useParams<{ instanceId: string }>()

  return (
    <ClusterProvider>
      <GlobalSearchProvider>
        <SidebarProvider>
          <MinIOSidebar instanceId={instanceId!} />
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
