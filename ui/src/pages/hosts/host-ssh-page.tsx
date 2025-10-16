import { useCallback, useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Loader2 } from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'

import { devopsAPI } from '@/lib/api-client'
import { Button } from '@/components/ui/button'
import { WebSSHTerminal } from '@/components/hosts/webssh-terminal'

export function HostSSHPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [sessionId, setSessionId] = useState<string | null>(null)
  const [isCreating, setIsCreating] = useState(false)

  const {
    data: hostResponse,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ['host', id],
    enabled: !!id,
    queryFn: async () => {
      if (!id) throw new Error('Missing host ID')
      return devopsAPI.vms.hosts.get(id)
    },
    staleTime: 10_000,
  })

  const host = (hostResponse as any)?.data

  const createSession = useCallback(async () => {
    if (!host || !host.online || isCreating) return

    setIsCreating(true)
    try {
      const response = (await devopsAPI.vms.webssh.createSession({
        host_id: host.id,
      })) as any

      if (response.code !== 0) {
        throw new Error(response.message || '创建 WebSSH 会话失败')
      }

      const { session_id: newSessionId } = response.data ?? {}
      if (!newSessionId) {
        throw new Error('返回的会话信息不完整')
      }

      setSessionId(newSessionId)
    } catch (error) {
      console.error('[WebSSH] Failed to create session', error)
      toast.error(
        error instanceof Error ? error.message : '创建 WebSSH 会话失败'
      )
    } finally {
      setIsCreating(false)
    }
  }, [host, isCreating])

  useEffect(() => {
    if (host && host.online && !sessionId && !isCreating) {
      createSession()
    }
  }, [host, sessionId, isCreating, createSession])

  const handleClose = () => {
    navigate(`/vms/hosts/${id}`)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError || !host) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground mb-4">
          {isError ? '加载主机信息失败' : '主机未找到'}
        </p>
        <Button onClick={() => navigate('/vms/hosts')} className="mt-4">
          返回列表
        </Button>
      </div>
    )
  }

  if (!host.online) {
    return (
      <div className="text-center py-12">
        <p>主机离线，无法建立终端连接</p>
        <Button onClick={() => navigate(`/vms/hosts/${id}`)} className="mt-4">
          返回主机详情
        </Button>
      </div>
    )
  }

  if (!sessionId || isCreating) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        <span className="ml-2">正在创建终端会话...</span>
      </div>
    )
  }

  return (
    <div className="flex h-screen flex-col bg-background">
      <div className="flex items-center justify-between border-b p-4">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={handleClose}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-xl font-bold">WebSSH Terminal</h1>
            <p className="text-sm text-muted-foreground">
              {host.name}
              {host.host_info &&
                ` · ${host.host_info.platform} ${host.host_info.arch}`}
            </p>
          </div>
        </div>
      </div>

      <div className="flex-1 p-4">
        <WebSSHTerminal
          sessionId={sessionId}
          hostName={host.name || 'SSH Session'}
          onClose={handleClose}
        />
      </div>
    </div>
  )
}
