import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  ArrowLeft,
  Database,
  Server,
  Activity,
  AlertTriangle,
  Settings,
  BarChart3,
  RefreshCw,
  Play,
  Square,
  Trash2,
} from 'lucide-react';

export default function InstanceDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [isRefreshing, setIsRefreshing] = useState(false);

  // TODO: Fetch real data from API
  const instance = {
    id,
    name: 'mysql-prod-01',
    type: 'mysql',
    environment: 'production',
    status: 'running',
    health: 'healthy',
    host: '10.0.1.10',
    port: 3306,
    version: '8.0.35',
    description: 'Production MySQL database for main application',
    tags: ['production', 'primary', 'critical'],
    config: {
      max_connections: 1000,
      innodb_buffer_pool_size: '8G',
      slow_query_log: 'ON',
    },
    metrics: {
      cpu_usage: 45.2,
      memory_usage: 62.8,
      disk_usage: 38.5,
      connections: 245,
      queries_per_second: 1234,
    },
    createdAt: '2024-01-15T10:30:00Z',
    updatedAt: '2024-03-01T14:20:00Z',
  };

  const handleRefresh = async () => {
    setIsRefreshing(true);
    // TODO: Call API to refresh instance status
    setTimeout(() => setIsRefreshing(false), 1000);
  };

  const getHealthColor = (health: string) => {
    const colors: Record<string, string> = {
      healthy: 'bg-green-500',
      unhealthy: 'bg-red-500',
      degraded: 'bg-yellow-500',
      unknown: 'bg-gray-500',
    };
    return colors[health] || colors.unknown;
  };

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate('/instances')}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-3">
              {instance.type === 'mysql' || instance.type === 'postgres' || instance.type === 'redis' ? (
                <Database className="h-6 w-6" />
              ) : (
                <Server className="h-6 w-6" />
              )}
              <h2 className="text-3xl font-bold tracking-tight">{instance.name}</h2>
              <Badge variant="outline">{instance.type.toUpperCase()}</Badge>
              <div className="flex items-center gap-2">
                <div className={`h-3 w-3 rounded-full ${getHealthColor(instance.health)}`} />
                <span className="text-sm capitalize">{instance.health}</span>
              </div>
            </div>
            <p className="text-sm text-muted-foreground mt-1">{instance.description}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon" onClick={handleRefresh} disabled={isRefreshing}>
            <RefreshCw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          </Button>
          <Button variant="outline" size="icon">
            {instance.status === 'running' ? <Square className="h-4 w-4" /> : <Play className="h-4 w-4" />}
          </Button>
          <Button variant="outline" size="icon" onClick={() => navigate(`/instances/${id}/edit`)}>
            <Settings className="h-4 w-4" />
          </Button>
          <Button variant="destructive" size="icon">
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">CPU Usage</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{instance.metrics.cpu_usage}%</div>
            <p className="text-xs text-muted-foreground">Current utilization</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Memory Usage</CardTitle>
            <BarChart3 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{instance.metrics.memory_usage}%</div>
            <p className="text-xs text-muted-foreground">Of allocated memory</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Disk Usage</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{instance.metrics.disk_usage}%</div>
            <p className="text-xs text-muted-foreground">Storage capacity</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Connections</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{instance.metrics.connections}</div>
            <p className="text-xs text-muted-foreground">Active connections</p>
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="metrics">Metrics</TabsTrigger>
          <TabsTrigger value="alerts">Alerts</TabsTrigger>
          <TabsTrigger value="audit">Audit Logs</TabsTrigger>
          <TabsTrigger value="config">Configuration</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Instance Information</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Instance ID:</span>
                  <span className="text-sm text-muted-foreground font-mono">{instance.id}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Type:</span>
                  <Badge variant="outline">{instance.type.toUpperCase()}</Badge>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Environment:</span>
                  <span className="text-sm capitalize">{instance.environment}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Version:</span>
                  <span className="text-sm text-muted-foreground">{instance.version}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Host:</span>
                  <span className="text-sm text-muted-foreground font-mono">{instance.host}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Port:</span>
                  <span className="text-sm text-muted-foreground font-mono">{instance.port}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm font-medium">Tags:</span>
                  <div className="flex gap-1">
                    {instance.tags.map((tag) => (
                      <Badge key={tag} variant="secondary" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Health Status</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex justify-between items-center">
                  <span className="text-sm font-medium">Overall Health:</span>
                  <div className="flex items-center gap-2">
                    <div className={`h-3 w-3 rounded-full ${getHealthColor(instance.health)}`} />
                    <span className="text-sm capitalize">{instance.health}</span>
                  </div>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Status:</span>
                  <Badge>{instance.status}</Badge>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Last Checked:</span>
                  <span className="text-sm text-muted-foreground">2 minutes ago</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Uptime:</span>
                  <span className="text-sm text-muted-foreground">45 days, 12 hours</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Response Time:</span>
                  <span className="text-sm text-muted-foreground">12ms</span>
                </div>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Performance Metrics</CardTitle>
              <CardDescription>Real-time performance indicators</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                Performance charts will be displayed here
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="metrics" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Metrics Dashboard</CardTitle>
              <CardDescription>Detailed metrics and statistics</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="h-[400px] flex items-center justify-center text-muted-foreground">
                Metrics dashboard will be displayed here
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="alerts" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Alert Rules</CardTitle>
              <CardDescription>Configure alert rules for this instance</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="flex items-center justify-between p-4 border rounded-lg">
                  <div className="flex items-center gap-3">
                    <AlertTriangle className="h-5 w-5 text-yellow-500" />
                    <div>
                      <p className="font-medium">High CPU Usage</p>
                      <p className="text-sm text-muted-foreground">Trigger when CPU &gt; 80%</p>
                    </div>
                  </div>
                  <Badge>Active</Badge>
                </div>
                <div className="flex items-center justify-between p-4 border rounded-lg">
                  <div className="flex items-center gap-3">
                    <AlertTriangle className="h-5 w-5 text-red-500" />
                    <div>
                      <p className="font-medium">Memory Exhaustion</p>
                      <p className="text-sm text-muted-foreground">Trigger when memory &gt; 90%</p>
                    </div>
                  </div>
                  <Badge>Active</Badge>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="audit" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Audit Logs</CardTitle>
              <CardDescription>Activity history for this instance</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {[
                  { action: 'Instance Started', user: 'system', time: '2 hours ago' },
                  { action: 'Configuration Updated', user: 'admin', time: '5 hours ago' },
                  { action: 'Health Check Passed', user: 'system', time: '1 day ago' },
                  { action: 'Instance Created', user: 'admin', time: '45 days ago' },
                ].map((log, idx) => (
                  <div key={idx} className="flex justify-between items-center p-3 border rounded">
                    <div>
                      <p className="font-medium text-sm">{log.action}</p>
                      <p className="text-xs text-muted-foreground">by {log.user}</p>
                    </div>
                    <span className="text-sm text-muted-foreground">{log.time}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="config" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Configuration</CardTitle>
              <CardDescription>Instance configuration settings</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {Object.entries(instance.config).map(([key, value]) => (
                  <div key={key} className="flex justify-between p-3 border rounded">
                    <span className="font-medium font-mono text-sm">{key}</span>
                    <span className="text-sm text-muted-foreground font-mono">{value}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
