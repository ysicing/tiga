import React, { useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

type Props = {
  onCreate: (expiry: '1h' | '1d' | '7d' | '30d') => Promise<{ url: string }>
}

export const ShareDialog: React.FC<Props> = ({ onCreate }) => {
  const [open, setOpen] = useState(false)
  const [expiry, setExpiry] = useState<'1h' | '1d' | '7d' | '30d'>('7d')
  const [url, setUrl] = useState<string>('')

  const handleCreate = async () => {
    const res = await onCreate(expiry)
    setUrl(res.url)
  }

  const copy = async () => {
    await navigator.clipboard.writeText(url)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">Share</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Share Link</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label>Expiry</Label>
            <Select value={expiry} onValueChange={(v) => setExpiry(v as any)}>
              <SelectTrigger>
                <SelectValue placeholder="Select expiry" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="1h">1 hour</SelectItem>
                <SelectItem value="1d">1 day</SelectItem>
                <SelectItem value="7d">7 days</SelectItem>
                <SelectItem value="30d">30 days</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {url && (
            <div>
              <Label>Share URL</Label>
              <div className="flex items-center space-x-2 mt-1">
                <input
                  className="flex-1 rounded border px-2 py-1"
                  readOnly
                  value={url}
                />
                <Button onClick={copy}>Copy</Button>
              </div>
            </div>
          )}
        </div>
        <DialogFooter>
          <Button onClick={handleCreate}>Generate</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
