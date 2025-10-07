import { useState } from 'react'
import { useAuth } from '@/contexts/auth-context'
import { Plus, Settings } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { useIsMobile } from '@/hooks/use-mobile'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'

import { CreateResourceDialog } from './create-resource-dialog'
import { DynamicBreadcrumb } from './dynamic-breadcrumb'
import { LanguageToggle } from './language-toggle'
import { ModeToggle } from './mode-toggle'
import { Search } from './search'
import { UserMenu } from './user-menu'

interface SiteHeaderProps {
  showSearch?: boolean
}

export function SiteHeader({ showSearch = false }: SiteHeaderProps) {
  const isMobile = useIsMobile()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [createDialogOpen, setCreateDialogOpen] = useState(false)

  return (
    <>
      <header className="sticky top-0 z-50 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
        <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
          <SidebarTrigger className="-ml-1" />
          <Separator
            orientation="vertical"
            className="mx-2 data-[orientation=vertical]:h-4"
          />
          <DynamicBreadcrumb />
          <div className="ml-auto flex items-center gap-2">
            {showSearch && <Search />}
            <Plus
              className="h-5 w-5 cursor-pointer text-muted-foreground hover:text-foreground"
              onClick={() => setCreateDialogOpen(true)}
              aria-label="Create new resource"
            />
            {!isMobile && (
              <>
                <Separator
                  orientation="vertical"
                  className="mx-2 data-[orientation=vertical]:h-4"
                />
                {user?.isAdmin() && (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => navigate('/settings')}
                    className="hidden sm:flex"
                  >
                    <Settings className="h-5 w-5" />
                    <span className="sr-only">Settings</span>
                  </Button>
                )}
                <LanguageToggle />
                <ModeToggle />
              </>
            )}
            <UserMenu />
          </div>
        </div>
      </header>

      <CreateResourceDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />
    </>
  )
}
