import React, { useRef } from 'react'

import { useUpload } from '@/hooks/use-upload'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'

type Props = {
  instanceId: string
  bucket: string
  onComplete?: () => void
}

export const FileUploader: React.FC<Props> = ({
  instanceId,
  bucket,
  onComplete,
}) => {
  const inputRef = useRef<HTMLInputElement>(null)
  const { queue, enqueue, pause, resume, cancel } = useUpload(
    instanceId,
    bucket,
    5
  )

  const onFilesSelected = (files: FileList | null) => {
    if (!files) return
    const items = Array.from(files).map((f) => ({ file: f, key: f.name }))
    enqueue(items)
  }

  const onDrop: React.DragEventHandler<HTMLDivElement> = (e) => {
    e.preventDefault()
    onFilesSelected(e.dataTransfer.files)
  }

  const onDragOver: React.DragEventHandler<HTMLDivElement> = (e) => {
    e.preventDefault()
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Upload Files</CardTitle>
      </CardHeader>
      <CardContent>
        <div
          className="border border-dashed rounded p-6 text-center hover:bg-muted cursor-pointer"
          onClick={() => inputRef.current?.click()}
          onDrop={onDrop}
          onDragOver={onDragOver}
        >
          Drag & drop files here, or click to select
          <input
            type="file"
            ref={inputRef}
            onChange={(e) => onFilesSelected(e.target.files)}
            className="hidden"
            multiple
          />
        </div>

        <div className="space-y-3 mt-4">
          {queue.map((item) => (
            <div key={item.key} className="border rounded p-3">
              <div className="flex items-center justify-between">
                <div>
                  <div className="font-medium">{item.key}</div>
                  <div className="text-xs text-muted-foreground">
                    {item.status} · {Math.round(item.speedBps / 1024)} KB/s ·
                    ETA {item.etaSeconds}s
                  </div>
                </div>
                <div className="space-x-2">
                  {item.status === 'uploading' && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => pause(item.key)}
                    >
                      Pause
                    </Button>
                  )}
                  {item.status === 'paused' && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => resume(item.key)}
                    >
                      Resume
                    </Button>
                  )}
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => cancel(item.key)}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
              <Progress value={item.progress} className="mt-2" />
              {item.error && (
                <div className="text-destructive text-sm mt-1">
                  {item.error}
                </div>
              )}
            </div>
          ))}
        </div>

        {onComplete && queue.every((i) => i.status === 'done') && (
          <div className="mt-4">
            <Button onClick={onComplete}>Done</Button>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
