import { useEffect, useRef, useState } from 'react'

import { API_BASE_URL } from '../lib/api-client'

export type UploadItem = {
  file: File
  key: string
  progress: number
  speedBps: number
  etaSeconds: number
  status: 'queued' | 'uploading' | 'paused' | 'error' | 'done'
  error?: string
}

export function useUpload(instanceId: string, bucket: string, concurrency = 5) {
  const [queue, setQueue] = useState<UploadItem[]>([])
  const active = useRef(0)
  const controllerMap = useRef<Map<string, XMLHttpRequest>>(new Map())

  const enqueue = (files: { file: File; key: string }[]) => {
    const items = files.map(({ file, key }) => ({
      file,
      key,
      progress: 0,
      speedBps: 0,
      etaSeconds: 0,
      status: 'queued' as const,
    }))
    setQueue((q) => [...q, ...items])
  }

  const startNext = () => {
    setQueue((q) => {
      if (active.current >= concurrency) return q
      const idx = q.findIndex((i) => i.status === 'queued')
      if (idx === -1) return q
      const item = q[idx]
      uploadItem(item, idx)
      const newQ = [...q]
      newQ[idx] = { ...item, status: 'uploading' }
      active.current += 1
      return newQ
    })
  }

  const uploadItem = (item: UploadItem, idx: number) => {
    const form = new FormData()
    form.append('bucket', bucket)
    form.append('name', item.key)
    form.append('file', item.file)

    const xhr = new XMLHttpRequest()
    controllerMap.current.set(item.key, xhr)
    let lastLoaded = 0
    let lastTime = Date.now()
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) {
        const progress = Math.round((e.loaded / e.total) * 100)
        const now = Date.now()
        const dt = (now - lastTime) / 1000
        const dBytes = e.loaded - lastLoaded
        const speed = dt > 0 ? dBytes / dt : 0
        const eta =
          speed > 0 ? Math.max(0, Math.round((e.total - e.loaded) / speed)) : 0
        lastLoaded = e.loaded
        lastTime = now
        setQueue((q) => {
          const nq = [...q]
          nq[idx] = { ...nq[idx], progress, speedBps: speed, etaSeconds: eta }
          return nq
        })
      }
    }
    xhr.onreadystatechange = () => {
      if (xhr.readyState === 4) {
        controllerMap.current.delete(item.key)
        active.current -= 1
        if (xhr.status >= 200 && xhr.status < 300) {
          setQueue((q) => {
            const nq = [...q]
            nq[idx] = { ...nq[idx], status: 'done', progress: 100 }
            return nq
          })
        } else {
          setQueue((q) => {
            const nq = [...q]
            nq[idx] = {
              ...nq[idx],
              status: 'error',
              error: `HTTP ${xhr.status}`,
            }
            return nq
          })
        }
        // trigger next
        startNext()
      }
    }
    xhr.open(
      'POST',
      `${API_BASE_URL}/minio/instances/${instanceId}/files`,
      true
    )
    xhr.withCredentials = true
    xhr.send(form)
  }

  useEffect(() => {
    if (active.current < concurrency) startNext()
  }, [queue, concurrency])

  const pause = (key: string) => {
    const xhr = controllerMap.current.get(key)
    if (xhr) {
      xhr.abort()
      controllerMap.current.delete(key)
      setQueue((q) =>
        q.map((i) => (i.key === key ? { ...i, status: 'paused' } : i))
      )
      active.current = Math.max(0, active.current - 1)
      startNext()
    }
  }

  const resume = (key: string) => {
    setQueue((q) =>
      q.map((i) =>
        i.key === key && i.status === 'paused' ? { ...i, status: 'queued' } : i
      )
    )
    startNext()
  }

  const cancel = (key: string) => {
    pause(key)
    setQueue((q) => q.filter((i) => i.key !== key))
  }

  return { queue, enqueue, pause, resume, cancel }
}
