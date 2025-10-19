import { Check, ChevronsUpDown, Loader2, Server } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Badge } from '@/components/ui/badge'
import { useCluster } from '@/contexts/cluster-context'
import { useState } from 'react'
import { Cluster } from '@/types/api'

interface ClusterSelectorProps {
  className?: string
  variant?: 'default' | 'compact'
}

const getHealthStatusColor = (
  status: Cluster['health_status']
): string => {
  switch (status) {
    case 'healthy':
      return 'bg-green-500'
    case 'unhealthy':
      return 'bg-red-500'
    default:
      return 'bg-gray-400'
  }
}

const getHealthStatusText = (
  status: Cluster['health_status']
): string => {
  switch (status) {
    case 'healthy':
      return 'Healthy'
    case 'unhealthy':
      return 'Unhealthy'
    default:
      return 'Unknown'
  }
}

export function ClusterSelector({ className, variant = 'default' }: ClusterSelectorProps) {
  const { clusters, currentCluster, setCurrentCluster, isLoading, isSwitching } =
    useCluster()
  const [open, setOpen] = useState(false)

  const selectedCluster = clusters.find((c) => c.name === currentCluster)

  if (isLoading) {
    return (
      <div className={cn('flex items-center gap-2', className)}>
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-sm text-muted-foreground">Loading clusters...</span>
      </div>
    )
  }

  if (clusters.length === 0) {
    return (
      <div className={cn('flex items-center gap-2', className)}>
        <Server className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm text-muted-foreground">No clusters</span>
      </div>
    )
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn(
            'justify-between',
            variant === 'compact' ? 'h-8' : 'w-[250px]',
            className
          )}
          disabled={isSwitching}
        >
          <div className="flex items-center gap-2 overflow-hidden">
            {isSwitching ? (
              <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
            ) : (
              <Server className="h-4 w-4 shrink-0 text-muted-foreground" />
            )}
            {selectedCluster ? (
              <div className="flex items-center gap-2 overflow-hidden">
                <span className="truncate">
                  {variant === 'compact' && selectedCluster.name.length > 15
                    ? `${selectedCluster.name.slice(0, 12)}...`
                    : selectedCluster.name}
                </span>
                <div
                  className={cn(
                    'h-2 w-2 rounded-full shrink-0',
                    getHealthStatusColor(selectedCluster.health_status)
                  )}
                />
              </div>
            ) : (
              <span className="text-muted-foreground">Select cluster...</span>
            )}
          </div>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[280px] p-0" align="start">
        <Command>
          <CommandInput placeholder="Search cluster..." />
          <CommandList>
            <CommandEmpty>No cluster found.</CommandEmpty>
            <CommandGroup>
              {clusters.map((cluster) => (
                <CommandItem
                  key={cluster.id}
                  value={cluster.name}
                  onSelect={() => {
                    setCurrentCluster(cluster.name)
                    setOpen(false)
                  }}
                >
                  <Check
                    className={cn(
                      'mr-2 h-4 w-4',
                      currentCluster === cluster.name
                        ? 'opacity-100'
                        : 'opacity-0'
                    )}
                  />
                  <div className="flex flex-1 items-center justify-between gap-2">
                    <div className="flex flex-col gap-1 overflow-hidden">
                      <div className="flex items-center gap-2">
                        <span className="truncate font-medium">
                          {cluster.name}
                        </span>
                        {cluster.is_default && (
                          <Badge variant="secondary" className="text-xs">
                            Default
                          </Badge>
                        )}
                      </div>
                      {cluster.description && (
                        <span className="truncate text-xs text-muted-foreground">
                          {cluster.description}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <div
                        className={cn(
                          'h-2 w-2 rounded-full',
                          getHealthStatusColor(cluster.health_status)
                        )}
                        title={getHealthStatusText(cluster.health_status)}
                      />
                      {cluster.node_count > 0 && (
                        <span className="text-xs text-muted-foreground">
                          {cluster.node_count}N
                        </span>
                      )}
                    </div>
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
