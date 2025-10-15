import { createBrowserRouter, Navigate } from 'react-router-dom'

import {
  InstallGuard,
  PreventReinstallGuard,
} from './components/guards/install-guard'
import { ProtectedRoute } from './components/protected-route'
import { DbsLayout } from './layouts/dbs-layout'
// Layouts
import { DevOpsLayout } from './layouts/devops-layout'
import { DockerLayout } from './layouts/docker-layout'
import { K8sLayout } from './layouts/k8s-layout'
import { MinIOLayout } from './layouts/minio-layout'
import { VMsLayout } from './layouts/vms-layout'
import { WebServerLayout } from './layouts/webserver-layout'
// DevOps Platform Pages
import AlertsPage from './pages/alerts'
import { CRListPage } from './pages/cr-list-page'
import DatabaseManagementPage from './pages/database-management'
import { InstanceDetail } from './pages/database/instance-detail'
import { InstanceForm } from './pages/database/instance-form'
import { DatabaseInstanceList } from './pages/database/instance-list'
// Database Pages
import { DbsOverview } from './pages/dbs-overview'
// Docker Pages
import { DockerOverview } from './pages/docker-overview'
import { AlertEventsPage } from './pages/hosts/alert-events-page'
import AlertRulesPage from './pages/hosts/alert-rules-page'
import { HostDetailPage as NewHostDetailPage } from './pages/hosts/host-detail-page'
import { HostEditPage } from './pages/hosts/host-edit-page'
// VMs/Host Management Pages
import { HostListPage } from './pages/hosts/host-list-page'
import { HostSSHPage } from './pages/hosts/host-ssh-page'
import ServiceMonitorDetailPage from './pages/hosts/service-monitor-detail'
import ServiceMonitorListPage from './pages/hosts/service-monitor-list'
import { ServiceMonitorPage } from './pages/hosts/service-monitor-page'
import InstallPage from './pages/install'
import { LoginPage } from './pages/login'
import MinIOManagementPage from './pages/minio-management'
import MinioFilesPage from './pages/minio/files-page'
import MinioInstancesPage from './pages/minio/instances-page'
import MinioUsersPage from './pages/minio/users-page'
import { Overview } from './pages/overview'
import { OverviewDashboard } from './pages/overview-dashboard-new'
import { ResourceDetail } from './pages/resource-detail'
import { ResourceList } from './pages/resource-list'
import RoleFormPage from './pages/role-form'
import RolesPage from './pages/roles'
import { SettingsPage } from './pages/settings'
import UserFormPage from './pages/user-form'
import UsersPage from './pages/users'
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
        element: <Navigate to="hosts" replace />,
      },
      // 主机节点管理
      {
        path: 'hosts',
        element: <HostListPage />,
      },
      {
        path: 'hosts/:id',
        element: <NewHostDetailPage />,
      },
      {
        path: 'hosts/:id/edit',
        element: <HostEditPage />,
      },
      {
        path: 'hosts/:id/ssh',
        element: <HostSSHPage />,
      },
      // 服务监控管理
      {
        path: 'service-monitors',
        children: [
          {
            index: true,
            element: <ServiceMonitorListPage />,
          },
          {
            path: 'overview',
            element: <Navigate to="list" replace />,
          },
          {
            path: 'list',
            element: <ServiceMonitorListPage />,
          },
          {
            path: 'new',
            element: <ServiceMonitorPage />,
          },
          {
            path: ':id',
            element: <ServiceMonitorDetailPage />,
          },
          {
            path: ':id/edit',
            element: <ServiceMonitorPage />,
          },
        ],
      },
      // 告警事件管理
      {
        path: 'alert-events',
        element: <AlertEventsPage />,
      },
      // 告警规则管理
      {
        path: 'alert-rules',
        element: <AlertRulesPage />,
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
  // 数据库 子系统 - 数据库管理（MySQL, PostgreSQL, Redis）
  {
    path: '/dbs',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <DbsLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="instances" replace />,
      },
      {
        path: 'instances',
        element: <DatabaseInstanceList />,
      },
      {
        path: 'new',
        element: <InstanceForm />,
      },
      {
        path: 'instances/new',
        element: <InstanceForm />,
      },
      {
        path: 'instances/:id',
        element: <InstanceDetail />,
      },
      {
        path: 'mysql',
        element: <DbsOverview type="mysql" />,
      },
      {
        path: 'postgresql',
        element: <DbsOverview type="postgresql" />,
      },
      {
        path: 'redis',
        element: <DbsOverview type="redis" />,
      },
    ],
  },
  // MinIO 子系统 - 统一路由结构
  {
    path: '/minio',
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
        element: <Navigate to="instances" replace />,
      },
      {
        path: 'instances',
        element: <MinioInstancesPage />,
      },
      {
        path: ':instanceId/overview',
        element: <MinIOManagementPage />,
      },
      {
        path: ':instanceId/files',
        element: <MinioFilesPage />,
      },
      {
        path: ':instanceId/users',
        element: <MinioUsersPage />,
      },
      {
        path: ':instanceId/policies',
        element: <div>MinIO Policies 管理页面(待实现)</div>,
      },
      {
        path: ':instanceId/metrics',
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
  // Storage 子系统 - MinIO 对象存储（重定向到 /minio）
  {
    path: '/storage',
    element: <Navigate to="/minio/instances" replace />,
  },
  {
    path: '/storage/*',
    element: <Navigate to="/minio/instances" replace />,
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
