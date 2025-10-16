import { useState } from 'react'
import {
  Database,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Search,
  Server,
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

interface Instance {
  id: string
  name: string
  type: string
  environment: string
  status: string
  health: string
  host: string
  port: number
  version: string
  createdAt: string
}

const getStatusBadge = (status: string) => {
  const variants: Record<
    string,
    'default' | 'secondary' | 'destructive' | 'outline'
  > = {
    running: 'default',
    stopped: 'secondary',
    failed: 'destructive',
    pending: 'outline',
  }
  return <Badge variant={variants[status] || 'outline'}>{status}</Badge>
}

const getHealthBadge = (health: string) => {
  const colors: Record<string, string> = {
    healthy: 'bg-green-500',
    unhealthy: 'bg-red-500',
    degraded: 'bg-yellow-500',
    unknown: 'bg-gray-500',
  }
  return (
    <div className="flex items-center gap-2">
      <div
        className={`h-2 w-2 rounded-full ${colors[health] || colors.unknown}`}
      />
      <span className="capitalize">{health}</span>
    </div>
  )
}

const getServiceIcon = (type: string) => {
  const iconClass = 'h-5 w-5'
  switch (type.toLowerCase()) {
    case 'mysql':
    case 'postgres':
    case 'redis':
      return <Database className={iconClass} />
    default:
      return <Server className={iconClass} />
  }
}

export default function InstanceListPage() {
  const navigate = useNavigate()
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedType, setSelectedType] = useState<string>('all')

  // TODO: Fetch real data from API
  const instances: Instance[] = [
    {
      id: '1',
      name: 'mysql-prod-01',
      type: 'mysql',
      environment: 'production',
      status: 'running',
      health: 'healthy',
      host: '10.0.1.10',
      port: 3306,
      version: '8.0.35',
      createdAt: '2024-01-15T10:30:00Z',
    },
    {
      id: '2',
      name: 'postgres-main',
      type: 'postgres',
      environment: 'production',
      status: 'running',
      health: 'healthy',
      host: '10.0.1.11',
      port: 5432,
      version: '16.1',
      createdAt: '2024-01-10T14:20:00Z',
    },
    {
      id: '3',
      name: 'redis-cache-01',
      type: 'redis',
      environment: 'production',
      status: 'running',
      health: 'degraded',
      host: '10.0.1.12',
      port: 6379,
      version: '7.2.3',
      createdAt: '2024-02-01T09:15:00Z',
    },
    {
      id: '4',
      name: 'minio-storage',
      type: 'minio',
      environment: 'production',
      status: 'running',
      health: 'healthy',
      host: '10.0.1.13',
      port: 9000,
      version: 'RELEASE.2024-01-01',
      createdAt: '2024-01-20T11:00:00Z',
    },
    {
      id: '5',
      name: 'redis-cache-02',
      type: 'redis',
      environment: 'staging',
      status: 'stopped',
      health: 'unhealthy',
      host: '10.0.2.10',
      port: 6379,
      version: '7.2.3',
      createdAt: '2024-02-05T16:45:00Z',
    },
  ]

  const serviceTypes = [
    'all',
    'mysql',
    'postgres',
    'redis',
    'minio',
    'docker',
    'k8s',
    'caddy',
  ]

  const filteredInstances = instances.filter((instance) => {
    const matchesSearch =
      instance.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      instance.host.includes(searchTerm)
    const matchesType = selectedType === 'all' || instance.type === selectedType
    return matchesSearch && matchesType
  })

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">Instances</h2>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon">
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button onClick={() => navigate('/dbs/instances/new')}>
            <Plus className="mr-2 h-4 w-4" />
            Add Instance
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Manage Instances</CardTitle>
          <CardDescription>
            View and manage all service instances
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4 mb-4">
            <div className="relative flex-1">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search instances..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-8"
              />
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline">
                  Filter:{' '}
                  {selectedType === 'all'
                    ? 'All Types'
                    : selectedType.toUpperCase()}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>Service Type</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {serviceTypes.map((type) => (
                  <DropdownMenuItem
                    key={type}
                    onClick={() => setSelectedType(type)}
                  >
                    {type === 'all' ? 'All Types' : type.toUpperCase()}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Environment</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Health</TableHead>
                <TableHead>Connection</TableHead>
                <TableHead>Version</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredInstances.map((instance) => (
                <TableRow
                  key={instance.id}
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => navigate(`/dbs/instances/${instance.id}`)}
                >
                  <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                      {getServiceIcon(instance.type)}
                      {instance.name}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">
                      {instance.type.toUpperCase()}
                    </Badge>
                  </TableCell>
                  <TableCell className="capitalize">
                    {instance.environment}
                  </TableCell>
                  <TableCell>{getStatusBadge(instance.status)}</TableCell>
                  <TableCell>{getHealthBadge(instance.health)}</TableCell>
                  <TableCell className="font-mono text-sm">
                    {instance.host}:{instance.port}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {instance.version}
                  </TableCell>
                  <TableCell className="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        asChild
                        onClick={(e) => e.stopPropagation()}
                      >
                        <Button variant="ghost" className="h-8 w-8 p-0">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuLabel>Actions</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={(e) => {
                            e.stopPropagation()
                            navigate(`/dbs/instances/${instance.id}`)
                          }}
                        >
                          View Details
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={(e) => {
                            e.stopPropagation()
                            navigate(`/dbs/instances/${instance.id}`)
                          }}
                        >
                          Edit
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={(e) => {
                            e.stopPropagation()
                            navigate(`/database/${instance.id}`)
                          }}
                        >
                          View Metrics
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={(e) => e.stopPropagation()}
                          className="text-destructive"
                        >
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {filteredInstances.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              No instances found matching your criteria
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
