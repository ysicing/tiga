import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Activity, Database, Server, AlertTriangle, Users, Clock } from 'lucide-react';

interface StatCardProps {
  title: string;
  value: string | number;
  description?: string;
  icon: React.ReactNode;
  trend?: {
    value: number;
    isPositive: boolean;
  };
}

function StatCard({ title, value, description, icon, trend }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {icon}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {description && <p className="text-xs text-muted-foreground">{description}</p>}
        {trend && (
          <p className={`text-xs ${trend.isPositive ? 'text-green-600' : 'text-red-600'}`}>
            {trend.isPositive ? '↑' : '↓'} {Math.abs(trend.value)}% from last week
          </p>
        )}
      </CardContent>
    </Card>
  );
}

export default function DashboardPage() {
  // TODO: Fetch real data from API
  const stats = {
    totalInstances: 42,
    runningInstances: 38,
    activeAlerts: 3,
    totalUsers: 12,
  };

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Instances"
          value={stats.totalInstances}
          description="Across all service types"
          icon={<Server className="h-4 w-4 text-muted-foreground" />}
          trend={{ value: 12, isPositive: true }}
        />
        <StatCard
          title="Running Instances"
          value={stats.runningInstances}
          description={`${((stats.runningInstances / stats.totalInstances) * 100).toFixed(1)}% healthy`}
          icon={<Activity className="h-4 w-4 text-muted-foreground" />}
          trend={{ value: 5, isPositive: true }}
        />
        <StatCard
          title="Active Alerts"
          value={stats.activeAlerts}
          description="Requires attention"
          icon={<AlertTriangle className="h-4 w-4 text-muted-foreground" />}
          trend={{ value: 8, isPositive: false }}
        />
        <StatCard
          title="Total Users"
          value={stats.totalUsers}
          description="Active team members"
          icon={<Users className="h-4 w-4 text-muted-foreground" />}
        />
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        <Card className="col-span-4">
          <CardHeader>
            <CardTitle>Instance Status Overview</CardTitle>
            <CardDescription>Real-time status of all managed instances</CardDescription>
          </CardHeader>
          <CardContent className="pl-2">
            {/* TODO: Add chart here */}
            <div className="h-[300px] flex items-center justify-center text-muted-foreground">
              Instance status chart will be displayed here
            </div>
          </CardContent>
        </Card>

        <Card className="col-span-3">
          <CardHeader>
            <CardTitle>Recent Alerts</CardTitle>
            <CardDescription>Latest alerts and notifications</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-8">
              {/* TODO: Fetch real alerts */}
              <div className="flex items-center">
                <AlertTriangle className="mr-2 h-4 w-4 text-yellow-500" />
                <div className="ml-4 space-y-1">
                  <p className="text-sm font-medium leading-none">High CPU Usage</p>
                  <p className="text-sm text-muted-foreground">MySQL instance mysql-prod-01</p>
                </div>
                <div className="ml-auto font-medium">
                  <Clock className="h-4 w-4 inline mr-1" />
                  5m ago
                </div>
              </div>
              <div className="flex items-center">
                <AlertTriangle className="mr-2 h-4 w-4 text-red-500" />
                <div className="ml-4 space-y-1">
                  <p className="text-sm font-medium leading-none">Instance Down</p>
                  <p className="text-sm text-muted-foreground">Redis instance redis-cache-02</p>
                </div>
                <div className="ml-auto font-medium">
                  <Clock className="h-4 w-4 inline mr-1" />
                  12m ago
                </div>
              </div>
              <div className="flex items-center">
                <AlertTriangle className="mr-2 h-4 w-4 text-yellow-500" />
                <div className="ml-4 space-y-1">
                  <p className="text-sm font-medium leading-none">Low Disk Space</p>
                  <p className="text-sm text-muted-foreground">MinIO instance minio-storage</p>
                </div>
                <div className="ml-auto font-medium">
                  <Clock className="h-4 w-4 inline mr-1" />
                  1h ago
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Instances by Type</CardTitle>
            <CardDescription>Distribution of managed services</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[
                { type: 'MySQL', count: 8, icon: <Database className="h-4 w-4" /> },
                { type: 'PostgreSQL', count: 6, icon: <Database className="h-4 w-4" /> },
                { type: 'Redis', count: 12, icon: <Database className="h-4 w-4" /> },
                { type: 'MinIO', count: 4, icon: <Server className="h-4 w-4" /> },
                { type: 'Docker', count: 8, icon: <Server className="h-4 w-4" /> },
                { type: 'Kubernetes', count: 3, icon: <Server className="h-4 w-4" /> },
                { type: 'Caddy', count: 1, icon: <Server className="h-4 w-4" /> },
              ].map((item) => (
                <div key={item.type} className="flex items-center">
                  {item.icon}
                  <div className="ml-4 space-y-1 flex-1">
                    <p className="text-sm font-medium leading-none">{item.type}</p>
                    <p className="text-sm text-muted-foreground">{item.count} instances</p>
                  </div>
                  <div className="ml-auto font-medium">{item.count}</div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
            <CardDescription>Latest system events and changes</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[
                { action: 'Instance Created', target: 'mysql-prod-03', user: 'admin', time: '2m ago' },
                { action: 'Alert Resolved', target: 'redis-cache-01', user: 'system', time: '15m ago' },
                { action: 'Instance Updated', target: 'postgres-main', user: 'john', time: '1h ago' },
                { action: 'User Added', target: 'jane@example.com', user: 'admin', time: '2h ago' },
                { action: 'Health Check Failed', target: 'minio-backup', user: 'system', time: '3h ago' },
              ].map((activity, idx) => (
                <div key={idx} className="flex items-center">
                  <div className="ml-4 space-y-1 flex-1">
                    <p className="text-sm font-medium leading-none">{activity.action}</p>
                    <p className="text-sm text-muted-foreground">
                      {activity.target} by {activity.user}
                    </p>
                  </div>
                  <div className="ml-auto text-sm text-muted-foreground">{activity.time}</div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
