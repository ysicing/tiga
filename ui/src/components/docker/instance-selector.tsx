import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, ChevronsUpDown } from 'lucide-react'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useDockerInstances } from '@/services/docker-api'
import { Badge } from '@/components/ui/badge'

interface DockerInstanceSelectorProps {
  value?: string
  onValueChange: (value: string) => void
}

export function DockerInstanceSelector({
  value,
  onValueChange,
}: DockerInstanceSelectorProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const { data } = useDockerInstances()

  const instances = data?.data || []
  const selectedInstance = instances.find((inst) => inst.id === value)

  // 自动选择第一个在线实例
  useEffect(() => {
    if (!value && instances.length > 0) {
      const onlineInstance = instances.find((inst) => inst.status === 'online')
      if (onlineInstance) {
        onValueChange(onlineInstance.id)
      } else if (instances[0]) {
        onValueChange(instances[0].id)
      }
    }
  }, [instances, value, onValueChange])

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-[300px] justify-between"
        >
          {selectedInstance ? (
            <span className="flex items-center gap-2">
              {selectedInstance.name}
              <Badge
                variant={
                  selectedInstance.status === 'online' ? 'default' : 'secondary'
                }
                className="ml-auto"
              >
                {selectedInstance.status}
              </Badge>
            </span>
          ) : (
            t('docker.selectInstance', '选择 Docker 实例...')
          )}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[300px] p-0">
        <Command>
          <CommandInput
            placeholder={t('docker.searchInstance', '搜索实例...')}
          />
          <CommandEmpty>
            {t('docker.noInstanceFound', '未找到实例')}
          </CommandEmpty>
          <CommandGroup>
            {instances.map((instance) => (
              <CommandItem
                key={instance.id}
                value={instance.id}
                onSelect={(currentValue) => {
                  onValueChange(currentValue)
                  setOpen(false)
                }}
              >
                <Check
                  className={cn(
                    'mr-2 h-4 w-4',
                    value === instance.id ? 'opacity-100' : 'opacity-0'
                  )}
                />
                <span className="flex-1">{instance.name}</span>
                <Badge
                  variant={
                    instance.status === 'online' ? 'default' : 'secondary'
                  }
                >
                  {instance.status}
                </Badge>
              </CommandItem>
            ))}
          </CommandGroup>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
