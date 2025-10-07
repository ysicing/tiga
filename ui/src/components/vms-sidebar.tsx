import { Link, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  IconServer2,
  IconLayoutDashboard,
  IconActivity,
  IconAlertTriangle,
  IconFolders,
} from '@tabler/icons-react'
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
import Logo from '@/assets/logo.png'

const menuItems = [
  {
    title: '主机节点',
    icon: IconServer2,
    path: '/vms/hosts',
  },
  {
    title: '服务监控',
    icon: IconActivity,
    path: '/vms/service-monitors',
  },
  {
    title: '告警事件',
    icon: IconAlertTriangle,
    path: '/vms/alert-events',
  },
  {
    title: '主机分组',
    icon: IconFolders,
    path: '/vms/host-groups',
  },
]

export function VMsSidebar() {
  const { t } = useTranslation()
  const location = useLocation()
  const { isMobile, setOpenMobile } = useSidebar()

  const isActive = (path: string) => {
    // 精确匹配子页面
    if (path === '/vms/hosts' && location.pathname.startsWith('/vms/hosts')) {
      return true
    }
    return location.pathname === path || location.pathname.startsWith(path + '/')
  }

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

        {/* VMs 菜单 */}
        <SidebarGroup>
          <SidebarGroupLabel>主机管理</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {menuItems.map((item) => {
                const IconComponent = item.icon
                return (
                  <SidebarMenuItem key={item.path}>
                    <SidebarMenuButton
                      asChild
                      isActive={isActive(item.path)}
                      tooltip={item.title}
                    >
                      <Link to={item.path} onClick={handleMenuItemClick}>
                        <IconComponent className="h-4 w-4" />
                        <span>{item.title}</span>
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
