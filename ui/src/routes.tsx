import { createBrowserRouter, Navigate, useParams } from 'react-router-dom'

import {
  InstallGuard,
  PreventReinstallGuard,
} from './components/guards/install-guard'
import { ProtectedRoute } from './components/protected-route'
import { DbsLayout } from './layouts/dbs-layout'
// Layouts
import { SystemLayout } from './layouts/system-layout'
import { DockerLayout } from './layouts/docker-layout'
import { K8sLayout } from './layouts/k8s-layout'
import { MinIOLayout } from './layouts/minio-layout'
import { VMsLayout } from './layouts/vms-layout'
import { WebServerLayout } from './layouts/webserver-layout'
// System Management Pages
import { SchedulerPage } from './pages/scheduler-page'
import { AuditPage } from './pages/audit-page'
import { SystemSettingsPage } from './pages/system-settings-page'
import { CRListPage } from './pages/k8s/cr-list-page'
import { CRDYAMLEditorPage } from './pages/k8s/crd-yaml-editor-page'
import DatabaseManagementPage from './pages/database-management'
import { InstanceDetail } from './pages/database/instance-detail'
import { InstanceForm } from './pages/database/instance-form'
import { DatabaseInstanceList } from './pages/database/instance-list'
// Database Pages
import { DbsOverview } from './pages/dbs-overview'
// Docker Pages
import { DockerOverview } from './pages/docker-overview'
import { DockerInstanceDetail } from './pages/docker/instance-detail'
import { DockerInstanceForm } from './pages/docker/instance-form'
import { ContainersPage } from './pages/docker/containers-page'
import { ImagesPage } from './pages/docker/images-page'
import { NetworksPage } from './pages/docker/networks-page'
import TerminalRecordingsPage from './pages/docker/terminal-recordings-page'
import TerminalRecordingPlayerPage from './pages/docker/terminal-recording-player-page'
import { VolumesPage } from './pages/docker/volumes-page'
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
import { Overview } from './pages/k8s/overview'
import { OverviewDashboard } from './pages/overview-dashboard-new'
import { ResourceDetail } from './pages/k8s/resource-detail'
import { ResourceList } from './pages/k8s/resource-list'
// K8s Cluster Management Pages
import { ClusterFormPage } from './pages/k8s/cluster-form-page'
import { ClusterListPage } from './pages/k8s/cluster-list-page'
import { ClusterDetailPage } from './pages/k8s/clusters/cluster-detail-page'
import { ResourceHistoryPage } from './pages/k8s/resource-history-page'
import { ResourceHistoryDetailPage } from './pages/k8s/resource-history-detail-page'
import { SearchPage } from './pages/k8s/search-page'
import { MonitoringPage } from './pages/k8s/monitoring-page'
import { OpenKruisePage as OpenKruiseOverviewPage } from './pages/k8s/openkruise-page'
import { TailscalePage } from './pages/k8s/tailscale-page'
import { SystemUpgradePage } from './pages/k8s/system-upgrade-page'
import { TraefikPage } from './pages/k8s/traefik-page'
// OpenKruise Pages
import { CloneSetListPage } from './pages/k8s/cloneset-list-page'
import { CloneSetDetail } from './pages/k8s/cloneset-detail'
import { AdvancedDaemonSetListPage } from './pages/k8s/advanced-daemonset-list-page'
import { AdvancedDaemonSetDetail } from './pages/k8s/advanced-daemonset-detail'
// import { BroadcastJobListPage } from './pages/k8s/broadcastjob-list-page'
// import { SidecarSetListPage } from './pages/k8s/sidecarset-list-page'
// import { ImagePullJobListPage } from './pages/k8s/imagepulljob-list-page'
// import { NodeImageListPage } from './pages/k8s/nodeimage-list-page'
// import { UnitedDeploymentListPage } from './pages/k8s/uniteddeployment-list-page'
// import { WorkloadSpreadListPage } from './pages/k8s/workloadspread-list-page'
// import { ContainerRecreateRequestListPage } from './pages/k8s/containerrecreate-list-page'
// import { ResourceDistributionListPage } from './pages/k8s/resourcedistribution-list-page'
// import { PersistentPodStateListPage } from './pages/k8s/persistentpodstate-list-page'
// import { PodProbeMarkerListPage } from './pages/k8s/podprobemarker-list-page'
// import { PodUnavailableBudgetListPage } from './pages/k8s/podunavailablebudget-list-page'
// Traefik Pages
import { IngressRouteListPage } from './pages/k8s/ingressroute-list-page'
import { IngressRouteDetail } from './pages/k8s/ingressroute-detail'
import { MiddlewareListPage } from './pages/k8s/middleware-list-page'
import { MiddlewareDetail } from './pages/k8s/middleware-detail'
// Tailscale Pages
import ConnectorListPage from './pages/k8s/connector-list-page'
import { ConnectorDetail } from './pages/k8s/connector-detail'
import ProxyClassListPage from './pages/k8s/proxyclass-list-page'
import { ProxyClassDetail } from './pages/k8s/proxyclass-detail'
// System Upgrade Pages
import UpgradePlansListPage from './pages/k8s/upgrade-plans-list-page'
import UpgradePlanDetail from './pages/k8s/upgrade-plan-detail'
import UserFormPage from './pages/user-form'
import UsersPage from './pages/users'
// WebServer Pages
import { WebServerOverview } from './pages/webserver-overview'
// Recording Pages
import { RecordingListPage } from './pages/recordings/recording-list-page'
import { RecordingDetailPage } from './pages/recordings/recording-detail-page'
import { RecordingPlayerPage } from './pages/recordings/recording-player-page'

// Wrapper components for detail pages that need URL params
const CloneSetDetailWrapper = () => {
  const { namespace, name } = useParams<{ namespace: string; name: string }>()
  return <CloneSetDetail namespace={namespace!} name={name!} />
}

const AdvancedDaemonSetDetailWrapper = () => {
  const { namespace, name } = useParams<{ namespace: string; name: string }>()
  return <AdvancedDaemonSetDetail namespace={namespace!} name={name!} />
}

const IngressRouteDetailWrapper = () => {
  const { namespace, name } = useParams<{ namespace: string; name: string }>()
  return <IngressRouteDetail namespace={namespace!} name={name!} />
}

const MiddlewareDetailWrapper = () => {
  const { namespace, name } = useParams<{ namespace: string; name: string }>()
  return <MiddlewareDetail namespace={namespace!} name={name!} />
}

const ConnectorDetailWrapper = () => {
  const { name } = useParams<{ namespace: string; name: string }>()
  // Connector uses namespace/name format in single prop
  return <ConnectorDetail name={name!} />
}

const ProxyClassDetailWrapper = () => {
  const { name } = useParams<{ name: string }>()
  return <ProxyClassDetail name={name!} />
}

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
  // System Management 子系统 - 系统管理
  {
    path: '/system',
    element: (
      <InstallGuard>
        <ProtectedRoute>
          <SystemLayout />
        </ProtectedRoute>
      </InstallGuard>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="scheduler" replace />,
      },
      {
        path: 'scheduler',
        element: <SchedulerPage />,
      },
      {
        path: 'audit',
        element: <AuditPage />,
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
        path: 'users/:id',
        element: <div>用户详情页（待实现）</div>,
      },
      {
        path: 'users/:id/edit',
        element: <UserFormPage />,
      },
      {
        path: 'users/:id/roles',
        element: <div>用户角色管理页（待实现）</div>,
      },
      {
        path: 'settings',
        element: <SystemSettingsPage />,
      },
      {
        path: 'recordings',
        element: <RecordingListPage />,
      },
      {
        path: 'recordings/:id',
        element: <RecordingDetailPage />,
      },
      {
        path: 'recordings/:id/player',
        element: <RecordingPlayerPage />,
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
        element: <Navigate to="clusters" replace />,
      },
      {
        path: 'overview',
        element: <Overview />,
      },
      // Cluster Management routes (must be before generic resource routes)
      {
        path: 'clusters',
        element: <ClusterListPage />,
      },
      {
        path: 'clusters/new',
        element: <ClusterFormPage />,
      },
      {
        path: 'clusters/:id',
        element: <ClusterDetailPage />,
      },
      {
        path: 'clusters/:id/edit',
        element: <ClusterFormPage />,
      },
      // Search and Monitoring routes
      {
        path: 'search',
        element: <SearchPage />,
      },
      {
        path: 'monitoring',
        element: <MonitoringPage />,
      },
      {
        path: 'clusters/:clusterId/resource-history',
        element: <ResourceHistoryPage />,
      },
      {
        path: 'clusters/:clusterId/resource-history/:historyId',
        element: <ResourceHistoryDetailPage />,
      },
      {
        path: 'clusters/:clusterId/yaml-editor',
        element: <CRDYAMLEditorPage />,
      },
      // Advanced Features routes
      {
        path: 'advanced-features/openkruise',
        element: <OpenKruiseOverviewPage />,
      },
      {
        path: 'advanced-features/tailscale',
        element: <TailscalePage />,
      },
      {
        path: 'advanced-features/system-upgrade',
        element: <SystemUpgradePage />,
      },
      {
        path: 'advanced-features/traefik',
        element: <TraefikPage />,
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
      // OpenKruise routes (must be before generic routes)
      {
        path: 'clonesets',
        element: <CloneSetListPage />,
      },
      {
        path: 'clonesets/:namespace/:name',
        element: <CloneSetDetailWrapper />,
      },
      {
        path: 'daemonsets.apps.kruise.io',
        element: <AdvancedDaemonSetListPage />,
      },
      {
        path: 'daemonsets.apps.kruise.io/:namespace/:name',
        element: <AdvancedDaemonSetDetailWrapper />,
      },
      {
        path: 'broadcastjobs',
        element: <div>Coming Soon: BroadcastJobs</div>,
      },
      {
        path: 'sidecarsets',
        element: <div>Coming Soon: SidecarSets</div>,
      },
      {
        path: 'imagepulljobs',
        element: <div>Coming Soon: ImagePullJobs</div>,
      },
      {
        path: 'nodeimages',
        element: <div>Coming Soon: NodeImages</div>,
      },
      {
        path: 'uniteddeployments',
        element: <div>Coming Soon: UnitedDeployments</div>,
      },
      {
        path: 'workloadspreads',
        element: <div>Coming Soon: WorkloadSpreads</div>,
      },
      {
        path: 'containerrecreaterequests',
        element: <div>Coming Soon: ContainerRecreateRequests</div>,
      },
      {
        path: 'resourcedistributions',
        element: <div>Coming Soon: ResourceDistributions</div>,
      },
      {
        path: 'persistentpodstates',
        element: <div>Coming Soon: PersistentPodStates</div>,
      },
      {
        path: 'podprobemarkers',
        element: <div>Coming Soon: PodProbeMarkers</div>,
      },
      {
        path: 'podunavailablebudgets',
        element: <div>Coming Soon: PodUnavailableBudgets</div>,
      },
      // Traefik routes
      {
        path: 'ingressroutes',
        element: <IngressRouteListPage />,
      },
      {
        path: 'ingressroutes/:namespace/:name',
        element: <IngressRouteDetailWrapper />,
      },
      {
        path: 'middlewares',
        element: <MiddlewareListPage />,
      },
      {
        path: 'middlewares/:namespace/:name',
        element: <MiddlewareDetailWrapper />,
      },
      // Tailscale routes
      {
        path: 'connectors',
        element: <ConnectorListPage />,
      },
      {
        path: 'connectors/:namespace/:name',
        element: <ConnectorDetailWrapper />,
      },
      {
        path: 'proxyclasses',
        element: <ProxyClassListPage />,
      },
      {
        path: 'proxyclasses/:name',
        element: <ProxyClassDetailWrapper />,
      },
      // System Upgrade routes
      {
        path: 'plans',
        element: <UpgradePlansListPage />,
      },
      {
        path: 'plans/:namespace/:name',
        element: <UpgradePlanDetail />,
      },
      // Generic K8s resource routes
      // IMPORTANT: Order matters! More specific routes (3 params) must come before generic routes (2 params)
      {
        path: ':resource/:namespace/:name',
        element: <ResourceDetail />,
      },
      {
        path: ':resource/:name',
        element: <ResourceDetail />,
      },
      {
        path: ':resource',
        element: <ResourceList />,
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
        path: 'instances/new',
        element: <DockerInstanceForm />,
      },
      {
        path: 'instances/:id',
        children: [
          {
            index: true,
            element: <DockerInstanceDetail />,
          },
          {
            path: 'edit',
            element: <DockerInstanceForm />,
          },
          {
            path: 'containers',
            element: <ContainersPage />,
          },
          {
            path: 'images',
            element: <ImagesPage />,
          },
          {
            path: 'networks',
            element: <NetworksPage />,
          },
          {
            path: 'volumes',
            element: <VolumesPage />,
          },
          {
            path: 'recordings',
            element: <TerminalRecordingsPage />,
          },
          {
            path: 'recordings/:recordingId/play',
            element: <TerminalRecordingPlayerPage />,
          },
        ],
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
