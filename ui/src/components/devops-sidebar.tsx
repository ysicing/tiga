import Logo from '@/assets/logo.png'
import {
  IconBell,
  IconLayoutDashboard,
  IconSettings,
  IconShield,
  IconUsers,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { Link, useLocation } from 'react-router-dom'

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'

const menuItems = [
  {
    title: 'devops.alerts',
    icon: IconBell,
    path: '/devops/alerts',
  },
  {
    title: 'devops.users',
    icon: IconUsers,
    path: '/devops/users',
  },
  {
    title: 'devops.roles',
    icon: IconShield,
    path: '/devops/roles',
  },
  {
    title: 'devops.settings',
    icon: IconSettings,
    path: '/devops/settings',
  },
]

export function DevOpsSidebar() {
  const { t } = useTranslation()
  const location = useLocation()
  const { isMobile, setOpenMobile } = useSidebar()

  const isActive = (path: string) => location.pathname.startsWith(path)

  const handleMenuItemClick = () => {
    if (isMobile) {
      setOpenMobile(false)
    }
  }

  return (
    <Sidebar collapsible="offcanvas">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild>
              <Link to="/" onClick={handleMenuItemClick}>
                <img src={Logo} alt="tiga Logo" className="h-8 w-8" />
                <span className="text-base font-semibold">tiga</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        {/* 返回总览 */}
        <SidebarGroup>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton asChild isActive={location.pathname === '/'}>
                <Link to="/" onClick={handleMenuItemClick}>
                  <IconLayoutDashboard className="h-4 w-4" />
                  <span>{t('nav.backToOverview')}</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>

        {/* DevOps 菜单 */}
        <SidebarGroup>
          <SidebarGroupLabel>DevOps 管理</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {menuItems.map((item) => {
                const IconComponent = item.icon
                return (
                  <SidebarMenuItem key={item.path}>
                    <SidebarMenuButton
                      asChild
                      isActive={isActive(item.path)}
                      tooltip={t(item.title)}
                    >
                      <Link to={item.path} onClick={handleMenuItemClick}>
                        <IconComponent className="h-4 w-4" />
                        <span>{t(item.title)}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
