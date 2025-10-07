import { createBrowserRouter, Navigate } from 'react-router-dom'

import { ProtectedRoute } from './components/protected-route'
import { InstallGuard, PreventReinstallGuard } from './components/guards/install-guard'
import { CRListPage } from './pages/cr-list-page'
import { LoginPage } from './pages/login'
import { Overview } from './pages/overview'
import { OverviewDashboard } from './pages/overview-dashboard-new'
import { ResourceDetail } from './pages/resource-detail'
import { ResourceList } from './pages/resource-list'
import { SettingsPage } from './pages/settings'
import InstallPage from './pages/install'

// Layouts
import { DevOpsLayout } from './layouts/devops-layout'
import { K8sLayout } from './layouts/k8s-layout'
import { MinIOLayout } from './layouts/minio-layout'
import { MiddlewareLayout } from './layouts/middleware-layout'
import { DockerLayout } from './layouts/docker-layout'
import { VMsLayout } from './layouts/vms-layout'
import { StorageLayout } from './layouts/storage-layout'
import { WebServerLayout } from './layouts/webserver-layout'

// DevOps Platform Pages
import HostsPage from './pages/hosts'
import HostFormPage from './pages/host-form'
import HostDetailPage from './pages/host-detail'
import HostMetricsPage from './pages/host-metrics'
import AlertsPage from './pages/alerts'
import UsersPage from './pages/users'
import UserFormPage from './pages/user-form'
import RolesPage from './pages/roles'
import RoleFormPage from './pages/role-form'
import MinIOManagementPage from './pages/minio-management'
import DatabaseManagementPage from './pages/database-management'

// Middleware Pages
import { MiddlewareOverview } from './pages/middleware-overview'

// Docker Pages
import { DockerOverview } from './pages/docker-overview'

// Storage Pages
import { StorageOverview } from './pages/storage-overview'

// WebServer Pages
import { WebServerOverview } from './pages/webserver-overview'

export const router = createBrowserRouter([
  {
    path: '/install',
    element: (
      <PreventReinstallGuard>
        <InstallPage />
      </PreventReinstallGuard>
    ),
  },
  {
    path: '/login',
    element: (
      <InstallGuard>
        <LoginPage />
      </InstallGuard>
    ),
  },
  // Overview Dashboard - 大屏总览,无侧边栏
  {
    path: '/',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <OverviewDashboard />
        </ProtectedRoute>
      </InstallGuard>
    ),
  },
  // VMs 子系统 - 主机/虚拟机管理
  {
    path: '/vms',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <VMsLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <HostsPage />,
      },
      {
        path: 'new',
        element: <HostFormPage />,
      },
      {
        path: ':id',
        element: <HostDetailPage />,
      },
      {
        path: ':id/edit',
        element: <HostFormPage />,
      },
      {
        path: ':id/metrics',
        element: <HostMetricsPage />,
      },
      {
        path: 'metrics',
        element: <div>VMs 整体监控页面(待实现)</div>,
      },
    ],
  },
  // DevOps 子系统
  {
    path: '/devops',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <DevOpsLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="alerts" replace />,
      },
      {
        path: 'alerts',
        element: <AlertsPage />,
      },
      {
        path: 'users',
        element: <UsersPage />,
      },
      {
        path: 'users/new',
        element: <UserFormPage />,
      },
      {
        path: 'users/:id/edit',
        element: <UserFormPage />,
      },
      {
        path: 'roles',
        element: <RolesPage />,
      },
      {
        path: 'roles/new',
        element: <RoleFormPage />,
      },
      {
        path: 'roles/:id/edit',
        element: <RoleFormPage />,
      },
      {
        path: 'settings',
        element: <SettingsPage />,
      },
    ],
  },
  // Middleware 子系统 - 中间件管理（MySQL, PostgreSQL, Redis）
  {
    path: '/middleware',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <MiddlewareLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="mysql" replace />,
      },
      {
        path: 'mysql',
        element: <MiddlewareOverview type="mysql" />,
      },
      {
        path: 'postgresql',
        element: <MiddlewareOverview type="postgresql" />,
      },
      {
        path: 'redis',
        element: <MiddlewareOverview type="redis" />,
      },
    ],
  },
  // MinIO 子系统
  {
    path: '/minio/:instanceId',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <MinIOLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="overview" replace />,
      },
      {
        path: 'overview',
        element: <MinIOManagementPage />,
      },
      {
        path: 'buckets',
        element: <div>MinIO Buckets 管理页面(待实现)</div>,
      },
      {
        path: 'users',
        element: <div>MinIO Users 管理页面(待实现)</div>,
      },
      {
        path: 'policies',
        element: <div>MinIO Policies 管理页面(待实现)</div>,
      },
      {
        path: 'metrics',
        element: <div>MinIO Metrics 监控页面(待实现)</div>,
      },
    ],
  },
  // Database 子系统
  {
    path: '/database/:instanceId',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <MinIOLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="overview" replace />,
      },
      {
        path: 'overview',
        element: <DatabaseManagementPage />,
      },
    ],
  },
  // Kubernetes 子系统
  {
    path: '/k8s',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <K8sLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="overview" replace />,
      },
      {
        path: 'overview',
        element: <Overview />,
      },
      // CRD routes
      {
        path: 'crds/:crd',
        element: <CRListPage />,
      },
      {
        path: 'crds/:resource/:namespace/:name',
        element: <ResourceDetail />,
      },
      {
        path: 'crds/:resource/:name',
        element: <ResourceDetail />,
      },
      // Generic K8s resource routes
      {
        path: ':resource/:name',
        element: <ResourceDetail />,
      },
      {
        path: ':resource',
        element: <ResourceList />,
      },
      {
        path: ':resource/:namespace/:name',
        element: <ResourceDetail />,
      },
    ],
  },
  // Docker 子系统
  {
    path: '/docker',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <DockerLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <DockerOverview />,
      },
      {
        path: 'containers',
        element: <div>Docker Containers 管理页面(待实现)</div>,
      },
      {
        path: 'images',
        element: <div>Docker Images 管理页面(待实现)</div>,
      },
      {
        path: 'networks',
        element: <div>Docker Networks 管理页面(待实现)</div>,
      },
    ],
  },
  // Storage 子系统 - MinIO 对象存储
  {
    path: '/storage',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <StorageLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <StorageOverview />,
      },
      {
        path: 'buckets',
        element: <div>Storage Buckets 管理页面(待实现)</div>,
      },
      {
        path: 'users',
        element: <div>Storage Users 管理页面(待实现)</div>,
      },
      {
        path: 'policies',
        element: <div>Storage Policies 管理页面(待实现)</div>,
      },
      {
        path: 'metrics',
        element: <div>Storage Metrics 监控页面(待实现)</div>,
      },
    ],
  },
  // WebServer 子系统 - Caddy Web 服务器
  {
    path: '/webserver',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <WebServerLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <WebServerOverview />,
      },
      {
        path: 'sites',
        element: <div>WebServer Sites 管理页面(待实现)</div>,
      },
      {
        path: 'config',
        element: <div>WebServer Config 配置页面(待实现)</div>,
      },
      {
        path: 'certificates',
        element: <div>WebServer Certificates 证书页面(待实现)</div>,
      },
      {
        path: 'metrics',
        element: <div>WebServer Metrics 监控页面(待实现)</div>,
      },
    ],
  },
])
