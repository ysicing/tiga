import Logo from '@/assets/logo.png'
import {
  IconActivity,
  IconFile,
  IconFileText,
  IconLayoutDashboard,
  IconServer,
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

interface MinIOSidebarProps {
  instanceId?: string
}

export function MinIOSidebar({ instanceId }: MinIOSidebarProps) {
  const { t } = useTranslation()
  const location = useLocation()
  const { isMobile, setOpenMobile } = useSidebar()

  // 实例管理菜单（总是显示）
  const managementItems = [
    {
      title: t('minio.instances'),
      icon: IconServer,
      path: '/minio/instances',
    },
  ]

  // 实例详情菜单（仅当有 instanceId 时显示）
  const instanceMenuItems = instanceId
    ? [
        {
          title: t('minio.overview'),
          icon: IconLayoutDashboard,
          path: `/minio/${instanceId}/overview`,
        },
        {
          title: t('minio.files'),
          icon: IconFile,
          path: `/minio/${instanceId}/files`,
        },
        {
          title: t('minio.users'),
          icon: IconUsers,
          path: `/minio/${instanceId}/users`,
        },
        {
          title: t('minio.policies'),
          icon: IconFileText,
          path: `/minio/${instanceId}/policies`,
        },
        {
          title: t('minio.metrics'),
          icon: IconActivity,
          path: `/minio/${instanceId}/metrics`,
        },
      ]
    : []

  const isActive = (path: string) => location.pathname === path

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

        {/* MinIO 实例管理 */}
        <SidebarGroup>
          <SidebarGroupLabel>{t('minio.management')}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {managementItems.map((item) => {
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

        {/* 实例详情菜单 */}
        {instanceId && (
          <SidebarGroup>
            <SidebarGroupLabel>
              {t('minio.instancePrefix')}: {instanceId.slice(0, 8)}...
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {instanceMenuItems.map((item) => {
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
        )}
      </SidebarContent>
    </Sidebar>
  )
}
