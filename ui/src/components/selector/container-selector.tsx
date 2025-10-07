import { useState } from 'react'
import { Check, ChevronsUpDown } from 'lucide-react'

import { SimpleContainer } from '@/types/k8s'
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

interface ContainerSelectorProps {
  containers: SimpleContainer
  selectedContainer?: string
  onContainerChange: (containerName?: string) => void
  placeholder?: string
  showAllOption?: boolean
}

export function ContainerSelector({
  containers,
  selectedContainer,
  onContainerChange,
  showAllOption = true,
  placeholder = 'Select container...',
}: ContainerSelectorProps) {
  const [open, setOpen] = useState(false)

  const allOption = { name: 'All Containers', image: '', init: false }
  const options = showAllOption ? [allOption, ...containers] : containers

  const selectedOption = selectedContainer
    ? containers.find((c) => c.name === selectedContainer)
    : allOption

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="max-w-[300px] justify-between"
        >
          {selectedOption ? selectedOption.name : placeholder}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="max-w-[300px] p-0">
        <Command>
          <CommandInput placeholder="Search containers..." />
          <CommandList>
            <CommandEmpty>No containers found.</CommandEmpty>
            <CommandGroup>
              {options.map((container) => (
                <CommandItem
                  key={container.name}
                  value={container.name}
                  onSelect={(currentValue) => {
                    const newValue =
                      currentValue === allOption.name ? undefined : currentValue
                    onContainerChange(newValue)
                    setOpen(false)
                  }}
                >
                  <Check
                    className={cn(
                      'mr-2 h-4 w-4',
                      selectedContainer === container.name ||
                        (!selectedContainer &&
                          container.name === allOption.name)
                        ? 'opacity-100'
                        : 'opacity-0'
                    )}
                  />
                  <div className="flex flex-col">
                    <span className="font-medium">
                      {container.name}
                      {container.init && (
                        <span className="text-xs text-muted-foreground ml-1">
                          (init)
                        </span>
                      )}
                    </span>
                    {container.image && (
                      <span className="text-xs text-muted-foreground">
                        {container.image}
                      </span>
                    )}
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
