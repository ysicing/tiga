import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { AlertTriangle, Search, Plus, RefreshCw, CheckCircle, XCircle, Clock } from 'lucide-react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

interface Alert {
  id: string;
  title: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
  status: 'open' | 'acknowledged' | 'resolved';
  instance: string;
  message: string;
  createdAt: string;
  acknowledgedBy?: string;
  resolvedBy?: string;
}

const getSeverityBadge = (severity: string) => {
  const variants: Record<string, { variant: any; className: string }> = {
    info: { variant: 'secondary', className: 'bg-blue-500' },
    warning: { variant: 'outline', className: 'bg-yellow-500' },
    error: { variant: 'destructive', className: 'bg-orange-500' },
    critical: { variant: 'destructive', className: 'bg-red-600' },
  };
  const config = variants[severity] || variants.info;
  return (
    <Badge variant={config.variant} className={config.className}>
      {severity.toUpperCase()}
    </Badge>
  );
};

const getStatusIcon = (status: string) => {
  switch (status) {
    case 'open':
      return <AlertTriangle className="h-4 w-4 text-red-500" />;
    case 'acknowledged':
      return <Clock className="h-4 w-4 text-yellow-500" />;
    case 'resolved':
      return <CheckCircle className="h-4 w-4 text-green-500" />;
    default:
      return <XCircle className="h-4 w-4 text-gray-500" />;
  }
};

export default function AlertsPage() {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedSeverity, setSelectedSeverity] = useState<string>('all');

  // TODO: Fetch real data from API
  const alerts: Alert[] = [
    {
      id: '1',
      title: 'High CPU Usage',
      severity: 'warning',
      status: 'open',
      instance: 'mysql-prod-01',
      message: 'CPU usage exceeded 80% threshold (current: 85%)',
      createdAt: '2024-03-04T10:30:00Z',
    },
    {
      id: '2',
      title: 'Instance Down',
      severity: 'critical',
      status: 'acknowledged',
      instance: 'redis-cache-02',
      message: 'Instance is not responding to health checks',
      createdAt: '2024-03-04T09:15:00Z',
      acknowledgedBy: 'admin',
    },
    {
      id: '3',
      title: 'Low Disk Space',
      severity: 'warning',
      status: 'resolved',
      instance: 'minio-storage',
      message: 'Disk usage exceeded 85% threshold',
      createdAt: '2024-03-03T14:20:00Z',
      acknowledgedBy: 'john',
      resolvedBy: 'john',
    },
    {
      id: '4',
      title: 'High Memory Usage',
      severity: 'error',
      status: 'open',
      instance: 'postgres-main',
      message: 'Memory usage exceeded 90% threshold (current: 92%)',
      createdAt: '2024-03-04T11:45:00Z',
    },
    {
      id: '5',
      title: 'Slow Query Detected',
      severity: 'info',
      status: 'resolved',
      instance: 'mysql-prod-01',
      message: 'Query execution time exceeded 5 seconds',
      createdAt: '2024-03-02T16:30:00Z',
      resolvedBy: 'system',
    },
  ];

  const severityOptions = ['all', 'info', 'warning', 'error', 'critical'];

  const filteredAlerts = alerts.filter((alert) => {
    const matchesSearch =
      alert.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
      alert.instance.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesSeverity = selectedSeverity === 'all' || alert.severity === selectedSeverity;
    return matchesSearch && matchesSeverity;
  });

  const openAlerts = filteredAlerts.filter((a) => a.status === 'open');
  const acknowledgedAlerts = filteredAlerts.filter((a) => a.status === 'acknowledged');
  const resolvedAlerts = filteredAlerts.filter((a) => a.status === 'resolved');

  const AlertTable = ({ alerts }: { alerts: Alert[] }) => (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Status</TableHead>
          <TableHead>Title</TableHead>
          <TableHead>Severity</TableHead>
          <TableHead>Instance</TableHead>
          <TableHead>Message</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {alerts.map((alert) => (
          <TableRow key={alert.id}>
            <TableCell>{getStatusIcon(alert.status)}</TableCell>
            <TableCell className="font-medium">{alert.title}</TableCell>
            <TableCell>{getSeverityBadge(alert.severity)}</TableCell>
            <TableCell className="font-mono text-sm">{alert.instance}</TableCell>
            <TableCell className="max-w-md truncate">{alert.message}</TableCell>
            <TableCell className="text-sm text-muted-foreground">
              {new Date(alert.createdAt).toLocaleString()}
            </TableCell>
            <TableCell className="text-right">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm">
                    Actions
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel>Actions</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  {alert.status === 'open' && (
                    <DropdownMenuItem>Acknowledge</DropdownMenuItem>
                  )}
                  {(alert.status === 'open' || alert.status === 'acknowledged') && (
                    <DropdownMenuItem>Resolve</DropdownMenuItem>
                  )}
                  <DropdownMenuItem>View Details</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem className="text-destructive">Delete</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">Alerts</h2>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon">
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            New Alert Rule
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Open Alerts</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{openAlerts.length}</div>
            <p className="text-xs text-muted-foreground">Requires attention</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Acknowledged</CardTitle>
            <Clock className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{acknowledgedAlerts.length}</div>
            <p className="text-xs text-muted-foreground">In progress</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Resolved</CardTitle>
            <CheckCircle className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{resolvedAlerts.length}</div>
            <p className="text-xs text-muted-foreground">Last 24 hours</p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Alert Management</CardTitle>
          <CardDescription>Monitor and manage system alerts</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4 mb-4">
            <div className="relative flex-1">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search alerts..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-8"
              />
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline">
                  Severity: {selectedSeverity === 'all' ? 'All' : selectedSeverity.toUpperCase()}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>Filter by Severity</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {severityOptions.map((severity) => (
                  <DropdownMenuItem key={severity} onClick={() => setSelectedSeverity(severity)}>
                    {severity === 'all' ? 'All Severities' : severity.toUpperCase()}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          <Tabs defaultValue="open" className="space-y-4">
            <TabsList>
              <TabsTrigger value="open">
                Open ({openAlerts.length})
              </TabsTrigger>
              <TabsTrigger value="acknowledged">
                Acknowledged ({acknowledgedAlerts.length})
              </TabsTrigger>
              <TabsTrigger value="resolved">
                Resolved ({resolvedAlerts.length})
              </TabsTrigger>
              <TabsTrigger value="all">
                All ({filteredAlerts.length})
              </TabsTrigger>
            </TabsList>

            <TabsContent value="open">
              <AlertTable alerts={openAlerts} />
            </TabsContent>
            <TabsContent value="acknowledged">
              <AlertTable alerts={acknowledgedAlerts} />
            </TabsContent>
            <TabsContent value="resolved">
              <AlertTable alerts={resolvedAlerts} />
            </TabsContent>
            <TabsContent value="all">
              <AlertTable alerts={filteredAlerts} />
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  );
}
