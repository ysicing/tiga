import Logo from '@/assets/logo.png'
import {
  IconLayoutDashboard,
  IconSettings,
  IconUsers,
  IconClock,
  IconFileText,
  IconPlayerRecord,
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
    title: 'system.scheduler',
    defaultTitle: '定时任务',
    icon: IconClock,
    path: '/system/scheduler',
  },
  {
    title: 'system.audit',
    defaultTitle: '审计日志',
    icon: IconFileText,
    path: '/system/audit',
  },
  {
    title: 'system.recordings',
    defaultTitle: '终端录制',
    icon: IconPlayerRecord,
    path: '/system/recordings',
  },
  {
    title: 'system.users',
    defaultTitle: '用户管理',
    icon: IconUsers,
    path: '/system/users',
  },
  {
    title: 'system.settings',
    defaultTitle: '系统设置',
    icon: IconSettings,
    path: '/system/settings',
  },
]

export function SystemSidebar() {
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

        {/* 系统管理菜单 */}
        <SidebarGroup>
          <SidebarGroupLabel>{t('system.title', '系统管理')}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {menuItems.map((item) => {
                const IconComponent = item.icon
                return (
                  <SidebarMenuItem key={item.path}>
                    <SidebarMenuButton
                      asChild
                      isActive={isActive(item.path)}
                      tooltip={t(item.title, item.defaultTitle)}
                    >
                      <Link to={item.path} onClick={handleMenuItemClick}>
                        <IconComponent className="h-4 w-4" />
                        <span>{t(item.title, item.defaultTitle)}</span>
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
