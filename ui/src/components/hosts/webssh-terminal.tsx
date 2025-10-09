import { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { X, Maximize2, Minimize2, RefreshCw } from 'lucide-react';

interface WebSSHTerminalProps {
  sessionId: string;
  hostName?: string;
  onClose?: () => void;
}

// Connection states
enum ConnectionState {
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  DISCONNECTED = 'disconnected',
  RECONNECTING = 'reconnecting',
  FAILED = 'failed',
}

// Reconnection configuration
const RECONNECT_CONFIG = {
  maxAttempts: 5,
  initialDelay: 1000, // 1 second
  maxDelay: 30000, // 30 seconds
  backoffMultiplier: 2,
};

const encodeBase64 = (value: string) => {
  if (typeof window === 'undefined') return '';
  return window.btoa(unescape(encodeURIComponent(value)));
};

const decodeBase64 = (value: string) => {
  if (typeof window === 'undefined') return '';
  return decodeURIComponent(escape(window.atob(value)));
};

export function WebSSHTerminal({
  sessionId,
  hostName = 'SSH Session',
  onClose,
}: WebSSHTerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalInstanceRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);

  const [isFullscreen, setIsFullscreen] = useState(false);
  const [connectionState, setConnectionState] = useState<ConnectionState>(
    ConnectionState.CONNECTING
  );
  const [reconnectDelay, setReconnectDelay] = useState(0);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // Calculate exponential backoff delay
  const getReconnectDelay = useCallback((attempt: number): number => {
    const delay = Math.min(
      RECONNECT_CONFIG.initialDelay * Math.pow(RECONNECT_CONFIG.backoffMultiplier, attempt),
      RECONNECT_CONFIG.maxDelay
    );
    return delay;
  }, []);

  // Connect to WebSocket
  const connectWebSocket = useCallback(() => {
    if (!terminalInstanceRef.current) return;

    const terminal = terminalInstanceRef.current;

    // Clear any existing reconnect timeout
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    // Close existing connection if any
    if (wsRef.current) {
      wsRef.current.onclose = null; // Prevent triggering reconnect
      wsRef.current.close();
      wsRef.current = null;
    }

    setConnectionState(
      reconnectAttemptsRef.current > 0
        ? ConnectionState.RECONNECTING
        : ConnectionState.CONNECTING
    );
    setErrorMessage(null);

    // Connect to WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsURL = `${protocol}//${window.location.host}/api/v1/vms/webssh/${sessionId}`;
    const ws = new WebSocket(wsURL);

    ws.onopen = () => {
      console.log('[WebSSH] Connected');
      setConnectionState(ConnectionState.CONNECTED);
      setErrorMessage(null);
      reconnectAttemptsRef.current = 0; // Reset on successful connection

      if (reconnectAttemptsRef.current > 0) {
        terminal.writeln('\r\n\x1b[1;32m=== Reconnected ===\x1b[0m\r\n');
      }
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);

        switch (msg.type) {
          case 'connected':
            terminal.writeln('\r\n\x1b[1;32m=== WebSSH Connected ===\x1b[0m\r\n');
            break;
          case 'output':
            const output = decodeBase64(msg.data?.output ?? '');
            terminal.write(output);
            break;
          case 'error':
            const errorMsg = msg.data?.message || 'Terminal error';
            terminal.writeln(`\r\n\x1b[1;31m[Error] ${errorMsg}\x1b[0m\r\n`);
            setErrorMessage(errorMsg);

            // Check if error suggests reconnection
            if (errorMsg.includes('interrupted') || errorMsg.includes('reconnecting')) {
              setConnectionState(ConnectionState.DISCONNECTED);
            }
            break;
          case 'ping':
            // Respond to ping with pong
            if (ws.readyState === WebSocket.OPEN) {
              ws.send(JSON.stringify({ type: 'pong', timestamp: Date.now() }));
            }
            break;
          case 'disconnected':
            setConnectionState(ConnectionState.DISCONNECTED);
            terminal.writeln('\r\n\x1b[33mConnection closed\x1b[0m\r\n');
            break;
        }
      } catch (err) {
        console.error('[WebSSH] Failed to parse message:', err);
      }
    };

    ws.onerror = (error) => {
      console.error('[WebSSH] Error:', error);
      terminal.writeln('\r\n\x1b[31mConnection error\x1b[0m\r\n');
      setErrorMessage('Connection error occurred');
    };

    ws.onclose = (event) => {
      console.log('[WebSSH] Disconnected', event.code, event.reason);
      setConnectionState(ConnectionState.DISCONNECTED);
      terminal.writeln('\r\n\x1b[33mConnection closed\x1b[0m\r\n');

      // Attempt reconnection if not manually closed and not exceeded max attempts
      if (
        event.code !== 1000 && // 1000 = normal closure
        reconnectAttemptsRef.current < RECONNECT_CONFIG.maxAttempts
      ) {
        const delay = getReconnectDelay(reconnectAttemptsRef.current);
        setReconnectDelay(delay);
        reconnectAttemptsRef.current += 1;

        terminal.writeln(
          `\x1b[33mReconnecting in ${delay / 1000}s... (Attempt ${reconnectAttemptsRef.current}/${RECONNECT_CONFIG.maxAttempts})\x1b[0m\r\n`
        );

        reconnectTimeoutRef.current = setTimeout(() => {
          connectWebSocket();
        }, delay);
      } else if (reconnectAttemptsRef.current >= RECONNECT_CONFIG.maxAttempts) {
        setConnectionState(ConnectionState.FAILED);
        setErrorMessage('Maximum reconnection attempts reached');
        terminal.writeln(
          `\r\n\x1b[1;31m=== Connection Failed ===\x1b[0m\r\n\x1b[31mMaximum reconnection attempts reached. Please refresh or reconnect manually.\x1b[0m\r\n`
        );
      }
    };

    wsRef.current = ws;

    // Send terminal input to server
    const onDataHandler = (data: string) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(
          JSON.stringify({
            type: 'input',
            data: {
              input: encodeBase64(data),
            },
            timestamp: Date.now(),
          })
        );
      }
    };

    // Remove old listeners and add new one
    terminal.onData(onDataHandler);

    // Handle window resize
    const handleResize = () => {
      fitAddonRef.current?.fit();
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(
          JSON.stringify({
            type: 'resize',
            data: {
              cols: terminal.cols,
              rows: terminal.rows,
            },
            timestamp: Date.now(),
          })
        );
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, [sessionId, getReconnectDelay]);

  // Manual reconnect
  const handleManualReconnect = () => {
    reconnectAttemptsRef.current = 0;
    setErrorMessage(null);
    connectWebSocket();
  };

  // Initialize terminal
  useEffect(() => {
    if (!terminalRef.current) return;

    // Initialize terminal
    const terminal = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
      },
      rows: 24,
      cols: 80,
      scrollback: 10000,
    });

    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);

    terminal.open(terminalRef.current);
    fitAddon.fit();

    terminalInstanceRef.current = terminal;
    fitAddonRef.current = fitAddon;

    // Initial connection
    connectWebSocket();

    // Cleanup
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.close();
      }
      terminal.dispose();
    };
  }, [sessionId, connectWebSocket]);

  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen);
    setTimeout(() => {
      fitAddonRef.current?.fit();
    }, 100);
  };

  const handleClose = () => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    if (wsRef.current) {
      wsRef.current.close(1000); // Normal closure
    }
    onClose?.();
  };

  // Render connection status indicator
  const getStatusColor = () => {
    switch (connectionState) {
      case ConnectionState.CONNECTED:
        return 'bg-green-500';
      case ConnectionState.CONNECTING:
      case ConnectionState.RECONNECTING:
        return 'bg-yellow-500 animate-pulse';
      case ConnectionState.DISCONNECTED:
        return 'bg-orange-500';
      case ConnectionState.FAILED:
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  const getStatusText = () => {
    switch (connectionState) {
      case ConnectionState.CONNECTED:
        return '已连接';
      case ConnectionState.CONNECTING:
        return '连接中...';
      case ConnectionState.RECONNECTING:
        return `重连中... (${reconnectAttemptsRef.current}/${RECONNECT_CONFIG.maxAttempts})`;
      case ConnectionState.DISCONNECTED:
        return '已断开';
      case ConnectionState.FAILED:
        return '连接失败';
      default:
        return '未知状态';
    }
  };

  return (
    <div className={`${isFullscreen ? 'fixed inset-0 z-50' : 'flex flex-col h-full'}`}>
      {/* Reconnection Alert */}
      {(connectionState === ConnectionState.RECONNECTING ||
        connectionState === ConnectionState.FAILED) && (
        <Alert variant={connectionState === ConnectionState.FAILED ? 'destructive' : 'default'} className="mb-2">
          <AlertDescription className="flex items-center justify-between">
            <span>
              {connectionState === ConnectionState.RECONNECTING
                ? `正在重连... (尝试 ${reconnectAttemptsRef.current}/${RECONNECT_CONFIG.maxAttempts}, ${reconnectDelay / 1000}秒后)`
                : errorMessage || '连接失败，已达到最大重试次数'}
            </span>
            {connectionState === ConnectionState.FAILED && (
              <Button size="sm" variant="outline" onClick={handleManualReconnect}>
                <RefreshCw className="h-4 w-4 mr-2" />
                手动重连
              </Button>
            )}
          </AlertDescription>
        </Alert>
      )}

      <Card className={`flex flex-col h-full ${isFullscreen ? 'rounded-none' : ''}`}>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-base font-medium flex items-center gap-2">
            <span className={`h-2 w-2 rounded-full ${getStatusColor()}`} title={getStatusText()} />
            {hostName}
            <span className="text-xs text-muted-foreground">{getStatusText()}</span>
          </CardTitle>
          <div className="flex items-center gap-2">
            {connectionState === ConnectionState.DISCONNECTED && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleManualReconnect}
                title="重新连接"
              >
                <RefreshCw className="h-4 w-4" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={toggleFullscreen}
              title={isFullscreen ? '退出全屏' : '全屏'}
            >
              {isFullscreen ? (
                <Minimize2 className="h-4 w-4" />
              ) : (
                <Maximize2 className="h-4 w-4" />
              )}
            </Button>
            <Button variant="ghost" size="sm" onClick={handleClose} title="关闭终端">
              <X className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent className="flex-1 p-0 overflow-hidden">
          <div ref={terminalRef} className="w-full h-full bg-[#1e1e1e]" />
        </CardContent>
      </Card>
    </div>
  );
}
