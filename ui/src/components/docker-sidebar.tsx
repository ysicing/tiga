import Logo from '@/assets/logo.png'
import {
  IconBox,
  IconChevronDown,
  IconChevronRight,
  IconContainer,
  IconDatabase,
  IconLayoutDashboard,
  IconNetwork,
  IconServer,
  IconStack,
  IconVideo,
} from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useLocation, useParams } from 'react-router-dom'

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
import { Badge } from '@/components/ui/badge'
import { useDockerInstances } from '@/services/docker-api'
import { Skeleton } from '@/components/ui/skeleton'

export function DockerSidebar() {
  const { t } = useTranslation()
  const location = useLocation()
  const { id: currentInstanceId } = useParams<{ id: string }>()
  const { isMobile, setOpenMobile } = useSidebar()
  const [expandedInstances, setExpandedInstances] = useState<Set<string>>(
    new Set(currentInstanceId ? [currentInstanceId] : [])
  )

  const { data, isLoading } = useDockerInstances()
  const instances = data?.data || []

  const handleMenuItemClick = () => {
    if (isMobile) {
      setOpenMobile(false)
    }
  }

  const toggleInstance = (instanceId: string) => {
    setExpandedInstances((prev) => {
      const next = new Set(prev)
      if (next.has(instanceId)) {
        next.delete(instanceId)
      } else {
        next.add(instanceId)
      }
      return next
    })
  }

  const getInstanceSubMenu = (instanceId: string) => [
    {
      title: 'docker.containers',
      icon: IconBox,
      path: `/docker/instances/${instanceId}/containers`,
    },
    {
      title: 'docker.images',
      icon: IconStack,
      path: `/docker/instances/${instanceId}/images`,
    },
    {
      title: 'docker.networks',
      icon: IconNetwork,
      path: `/docker/instances/${instanceId}/networks`,
    },
    {
      title: 'docker.volumes',
      icon: IconDatabase,
      path: `/docker/instances/${instanceId}/volumes`,
    },
    {
      title: 'docker.recordings',
      icon: IconVideo,
      path: `/docker/instances/${instanceId}/recordings`,
    },
  ]

  const isActive = (path: string) => {
    return location.pathname === path || location.pathname.startsWith(path + '/')
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

        {/* Docker 实例 */}
        <SidebarGroup>
          <SidebarGroupLabel>
            {t('docker.title', 'Docker 管理')}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  asChild
                  isActive={location.pathname === '/docker'}
                >
                  <Link to="/docker" onClick={handleMenuItemClick}>
                    <IconContainer className="h-4 w-4" />
                    <span>{t('docker.instances', '实例列表')}</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {/* Docker 实例列表 */}
        <SidebarGroup>
          <SidebarGroupLabel>
            {t('docker.instances', 'Docker 实例')}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {isLoading ? (
                <div className="space-y-2 px-2">
                  <Skeleton className="h-8 w-full" />
                  <Skeleton className="h-8 w-full" />
                </div>
              ) : instances.length === 0 ? (
                <div className="px-2 py-4 text-sm text-muted-foreground text-center">
                  {t('docker.noInstances', '暂无实例')}
                </div>
              ) : (
                instances.map((instance) => {
                  const isExpanded = expandedInstances.has(instance.id)
                  const subMenuItems = getInstanceSubMenu(instance.id)

                  return (
                    <div key={instance.id}>
                      {/* 实例主项 */}
                      <SidebarMenuItem>
                        <SidebarMenuButton
                          onClick={() => toggleInstance(instance.id)}
                          className="w-full justify-between"
                          isActive={currentInstanceId === instance.id}
                        >
                          <div className="flex items-center gap-2 flex-1 min-w-0">
                            <IconServer className="h-4 w-4 flex-shrink-0" />
                            <span className="truncate">{instance.name}</span>
                            <Badge
                              variant={
                                instance.status === 'online'
                                  ? 'default'
                                  : 'secondary'
                              }
                              className="ml-auto flex-shrink-0 text-xs"
                            >
                              {instance.status === 'online' ? '在线' : '离线'}
                            </Badge>
                          </div>
                          {isExpanded ? (
                            <IconChevronDown className="h-4 w-4 flex-shrink-0" />
                          ) : (
                            <IconChevronRight className="h-4 w-4 flex-shrink-0" />
                          )}
                        </SidebarMenuButton>
                      </SidebarMenuItem>

                      {/* 子菜单 */}
                      {isExpanded && (
                        <div className="ml-4 border-l border-border">
                          {subMenuItems.map((item) => {
                            const IconComponent = item.icon
                            return (
                              <SidebarMenuItem key={item.path}>
                                <SidebarMenuButton
                                  asChild
                                  isActive={isActive(item.path)}
                                  tooltip={t(item.title)}
                                >
                                  <Link
                                    to={item.path}
                                    onClick={handleMenuItemClick}
                                  >
                                    <IconComponent className="h-4 w-4" />
                                    <span>{t(item.title)}</span>
                                  </Link>
                                </SidebarMenuButton>
                              </SidebarMenuItem>
                            )
                          })}
                        </div>
                      )}
                    </div>
                  )
                })
              )}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
