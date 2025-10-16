import { useCallback, useEffect, useRef } from 'react'

import { HostState, useHostStore } from '../stores/host-store'

interface WebSocketMessage {
  action: 'subscribe' | 'unsubscribe' | 'state_update'
  host_ids?: string[]
  data?: HostState & { host_id: string }
}

interface UseHostMonitorOptions {
  hostIds?: string[] // Specific hosts to monitor, empty for all
  autoConnect?: boolean
  reconnectInterval?: number // in milliseconds
  maxReconnectAttempts?: number
}

export function useHostMonitor(options: UseHostMonitorOptions = {}) {
  const {
    hostIds = [],
    autoConnect = true,
    reconnectInterval = 3000,
    maxReconnectAttempts = 10,
  } = options

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | undefined>(undefined)
  const reconnectAttemptsRef = useRef(0)

  const { updateHostState, setWSSubscription, wsSubscription } = useHostStore()

  // Build WebSocket URL
  const getWebSocketURL = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    return `${protocol}//${host}/api/v1/vms/ws/hosts/monitor`
  }, [])

  // Send message to WebSocket
  const sendMessage = useCallback((message: WebSocketMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
    }
  }, [])

  // Subscribe to hosts
  const subscribe = useCallback(
    (hostIdList: string[]) => {
      sendMessage({
        action: 'subscribe',
        host_ids: hostIdList.length > 0 ? hostIdList : undefined, // undefined = all hosts
      })
      setWSSubscription({ hostIds: hostIdList })
    },
    [sendMessage, setWSSubscription]
  )

  // Unsubscribe from hosts
  const unsubscribe = useCallback(() => {
    sendMessage({
      action: 'unsubscribe',
    })
    setWSSubscription({ hostIds: [] })
  }, [sendMessage, setWSSubscription])

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return // Already connected
    }

    const wsURL = getWebSocketURL()
    const ws = new WebSocket(wsURL)

    ws.onopen = () => {
      console.log('[Host Monitor] WebSocket connected')
      reconnectAttemptsRef.current = 0
      setWSSubscription({
        connected: true,
        reconnecting: false,
        error: undefined,
      })

      // Subscribe to hosts after connection
      if (hostIds.length > 0 || autoConnect) {
        subscribe(hostIds)
      }
    }

    ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data)

        if (message.action === 'state_update' && message.data) {
          const { host_id, ...state } = message.data
          updateHostState(host_id, state)
        }
      } catch (error) {
        console.error('[Host Monitor] Failed to parse message:', error)
      }
    }

    ws.onerror = (error) => {
      console.error('[Host Monitor] WebSocket error:', error)
      setWSSubscription({
        error: 'WebSocket connection error',
      })
    }

    ws.onclose = () => {
      console.log('[Host Monitor] WebSocket closed')
      setWSSubscription({ connected: false })

      // Attempt to reconnect
      if (autoConnect && reconnectAttemptsRef.current < maxReconnectAttempts) {
        reconnectAttemptsRef.current++
        setWSSubscription({ reconnecting: true })

        reconnectTimeoutRef.current = setTimeout(() => {
          console.log(
            `[Host Monitor] Reconnecting (attempt ${reconnectAttemptsRef.current}/${maxReconnectAttempts})...`
          )
          connect()
        }, reconnectInterval)
      } else if (reconnectAttemptsRef.current >= maxReconnectAttempts) {
        setWSSubscription({
          error: 'Max reconnect attempts reached',
          reconnecting: false,
        })
      }
    }

    wsRef.current = ws
  }, [
    getWebSocketURL,
    hostIds,
    autoConnect,
    maxReconnectAttempts,
    reconnectInterval,
    setWSSubscription,
    subscribe,
    updateHostState,
  ])

  // Disconnect from WebSocket
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }

    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    setWSSubscription({
      connected: false,
      reconnecting: false,
      hostIds: [],
    })
  }, [setWSSubscription])

  // Auto-connect on mount
  useEffect(() => {
    if (autoConnect) {
      connect()
    }

    // Cleanup on unmount
    return () => {
      disconnect()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []) // Only run on mount/unmount

  // Update subscription when hostIds change
  useEffect(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN && hostIds.length > 0) {
      subscribe(hostIds)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hostIds])

  return {
    connected: wsSubscription.connected,
    reconnecting: wsSubscription.reconnecting,
    error: wsSubscription.error,
    connect,
    disconnect,
    subscribe,
    unsubscribe,
  }
}
