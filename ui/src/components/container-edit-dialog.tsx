import { useEffect, useState } from 'react'
import { DialogDescription } from '@radix-ui/react-dialog'
import { Container } from 'kubernetes-types/core/v1'

import { EnvironmentEditor, ImageEditor, ResourceEditor } from './editors'
import { Button } from './ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs'

interface ContainerEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  container: Container
  onSave: (updatedContainer: Container) => void
}

export function ContainerEditDialog({
  open,
  onOpenChange,
  container,
  onSave,
}: ContainerEditDialogProps) {
  const [editedContainer, setEditedContainer] = useState<Container>(container)

  useEffect(() => {
    setEditedContainer({ ...container })
  }, [container])

  const handleSave = () => {
    onSave(editedContainer)
    onOpenChange(false)
  }

  const handleUpdate = (updates: Partial<Container>) => {
    setEditedContainer((prev) => ({ ...prev, ...updates }))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="!max-w-4xl max-h-[90vh] overflow-y-auto sm:!max-w-4xl">
        <DialogHeader>
          <DialogTitle>Edit Container: {container.name}</DialogTitle>
          <DialogDescription className="text-sm text-muted-foreground">
            More complex changes can be made by modifying in YAML.
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="image" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="image">Image</TabsTrigger>
            <TabsTrigger value="resources">Resources</TabsTrigger>
            <TabsTrigger value="environment">Environment</TabsTrigger>
          </TabsList>

          <TabsContent value="image" className="space-y-4">
            <ImageEditor container={editedContainer} onUpdate={handleUpdate} />
          </TabsContent>

          <TabsContent value="resources" className="space-y-6">
            <ResourceEditor
              container={editedContainer}
              onUpdate={handleUpdate}
            />
          </TabsContent>

          <TabsContent value="environment" className="space-y-4">
            <EnvironmentEditor
              container={editedContainer}
              onUpdate={handleUpdate}
            />
          </TabsContent>
        </Tabs>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave}>Save Changes</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
