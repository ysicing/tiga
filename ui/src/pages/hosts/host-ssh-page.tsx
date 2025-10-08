import { useCallback, useEffect, useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { ArrowLeft, Loader2 } from 'lucide-react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import { toast } from 'sonner';
import { devopsAPI } from '@/lib/api-client';

type WebSSHMessage = {
  type: string;
  data?: any;
  session_id?: string;
  timestamp?: number;
};

const encodeBase64 = (value: string) => {
  if (typeof window === 'undefined') return '';
  return window.btoa(unescape(encodeURIComponent(value)));
};

const decodeBase64 = (value: string) => {
  if (typeof window === 'undefined') return '';
  return decodeURIComponent(escape(window.atob(value)));
};

type ConnectionStatus = 'idle' | 'connecting' | 'connected' | 'disconnected' | 'error';

export function HostSSHPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const terminalContainerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const inputDisposableRef = useRef<{ dispose: () => void } | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const sessionIdRef = useRef<string | null>(null);
  const cleanupInProgress = useRef(false);
  const [status, setStatus] = useState<ConnectionStatus>('idle');

  const { data: hostResponse, isLoading, isError } = useQuery({
    queryKey: ['host', id],
    enabled: !!id,
    queryFn: async () => {
      if (!id) throw new Error('Missing host ID');
      return devopsAPI.vms.hosts.get(id);
    },
    staleTime: 10_000,
  });

  const host = hostResponse?.data;

  const sendMessage = useCallback((type: string, payload?: Record<string, unknown>) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      return;
    }

    const message: WebSSHMessage = {
      type,
      data: payload,
      timestamp: Date.now(),
    };

    if (sessionIdRef.current) {
      message.session_id = sessionIdRef.current;
    }

    ws.send(JSON.stringify(message));
  }, []);

  const sendResize = useCallback(() => {
    const term = terminalRef.current;
    if (!term) return;
    sendMessage('resize', { cols: term.cols, rows: term.rows });
  }, [sendMessage]);

  const closeRemoteSession = useCallback(
    async (reason = 'client closed') => {
      if (cleanupInProgress.current) {
        return;
      }
      cleanupInProgress.current = true;

      try {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          sendMessage('command', { command: 'close' });
          wsRef.current.close();
        }
        wsRef.current = null;

        if (sessionIdRef.current) {
          try {
            await devopsAPI.vms.webssh.closeSession(sessionIdRef.current);
          } catch (err) {
            logDebug('Failed to close WebSSH session', err);
          }
      }
    } finally {
      if (inputDisposableRef.current) {
        inputDisposableRef.current.dispose();
        inputDisposableRef.current = null;
      }
      sessionIdRef.current = null;
      setStatus((prev) => (reason === 'client closed' ? 'disconnected' : prev));
      cleanupInProgress.current = false;
    }
    },
    [sendMessage]
  );

  const handleServerMessage = useCallback(
    (event: MessageEvent) => {
      let msg: WebSSHMessage;
      try {
        msg = JSON.parse(event.data);
      } catch {
        logDebug('Invalid WebSSH message', event.data);
        return;
      }

      switch (msg.type) {
        case 'connected': {
          setStatus('connected');
          sendResize();
          if (terminalRef.current) {
            terminalRef.current.writeln('\r\n\x1b[1;32m=== WebSSH Connected ===\x1b[0m\r\n');
          }
          break;
        }
        case 'output': {
          const output = decodeBase64(msg.data?.output ?? '');
          if (terminalRef.current) {
            terminalRef.current.write(output);
          }
          break;
        }
        case 'error': {
          const message = msg.data?.message || '终端发生错误';
          toast.error(message);
          setStatus('error');
          break;
        }
        case 'info': {
          if (msg.data?.message && terminalRef.current) {
            terminalRef.current.writeln(`\r\n\x1b[1;34m${msg.data.message}\x1b[0m\r\n`);
          }
          break;
        }
        case 'ping': {
          sendMessage('pong');
          break;
        }
        case 'disconnected': {
          setStatus('disconnected');
          break;
        }
        default:
          logDebug('Unhandled WebSSH message', msg);
      }
    },
    [sendMessage, sendResize]
  );

  const openWebSocket = useCallback(
    (url: string) => {
      const websocket = new WebSocket(url);
      wsRef.current = websocket;

      websocket.onopen = () => {
        setStatus('connecting');
      };

      websocket.onmessage = handleServerMessage;

      websocket.onerror = (event) => {
        console.error('[WebSSH] WebSocket error', event);
        toast.error('WebSSH 连接出现异常');
        setStatus('error');
      };

      websocket.onclose = () => {
        setStatus((prev) => (prev === 'error' ? prev : 'disconnected'));
      };
    },
    [handleServerMessage]
  );

  useEffect(() => {
    if (!terminalContainerRef.current) {
      return;
    }

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#d4d4d4',
        black: '#000000',
        red: '#cd3131',
        green: '#0dbc79',
        yellow: '#e5e510',
        blue: '#2472c8',
        magenta: '#bc3fbc',
        cyan: '#11a8cd',
        white: '#e5e5e5',
        brightBlack: '#666666',
        brightRed: '#f14c4c',
        brightGreen: '#23d18b',
        brightYellow: '#f5f543',
        brightBlue: '#3b8eea',
        brightMagenta: '#d670d6',
        brightCyan: '#29b8db',
        brightWhite: '#e5e5e5',
      },
      scrollback: 5000,
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(new WebLinksAddon());

    term.open(terminalContainerRef.current);
    fitAddon.fit();

    terminalRef.current = term;
    fitAddonRef.current = fitAddon;

    const handleResize = () => {
      fitAddon.fit();
      sendResize();
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      term.dispose();
      terminalRef.current = null;
      fitAddonRef.current = null;
    };
  }, [sendResize]);

  useEffect(() => {
    if (!host || !host.online || !terminalRef.current) {
      return;
    }

    let cancelled = false;
    setStatus('connecting');

    const initSession = async () => {
      try {
        const term = terminalRef.current!;
        const response = await devopsAPI.vms.webssh.createSession({
          host_id: host.id,
          width: term.cols,
          height: term.rows,
        });

        if (cancelled) return;

        if (response.code !== 0) {
          throw new Error(response.message || '创建 WebSSH 会话失败');
        }

        const { websocket_url: websocketUrl, session_id: sessionId } = response.data ?? {};
        if (!websocketUrl || !sessionId) {
          throw new Error('返回的会话信息不完整');
        }

        sessionIdRef.current = sessionId;
        openWebSocket(websocketUrl);

        // Forward terminal input
        inputDisposableRef.current = term.onData((data) => {
          sendMessage('input', { input: encodeBase64(data) });
        });
      } catch (error) {
        if (cancelled) return;
        console.error('[WebSSH] Failed to initialize session', error);
        toast.error(error instanceof Error ? error.message : '创建 WebSSH 会话失败');
        setStatus('error');
        closeRemoteSession('error').catch(() => undefined);
      }
    };

    initSession();

    return () => {
      cancelled = true;
      closeRemoteSession();
    };
  }, [host, closeRemoteSession, openWebSocket, sendMessage]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isError || !host) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground mb-4">{isError ? '加载主机信息失败' : '主机未找到'}</p>
        <Button onClick={() => navigate('/vms/hosts')} className="mt-4">
          返回列表
        </Button>
      </div>
    );
  }

  if (!host.online) {
    return (
      <div className="text-center py-12">
        <p>主机离线，无法建立终端连接</p>
        <Button onClick={() => navigate(`/vms/hosts/${id}`)} className="mt-4">
          返回主机详情
        </Button>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b p-4">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(`/vms/hosts/${id}`)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-xl font-bold">WebSSH Terminal</h1>
            <p className="text-sm text-muted-foreground">
              {host.name}
              {host.host_info && ` · ${host.host_info.platform} ${host.host_info.arch}`}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <span
            className={`h-2 w-2 rounded-full ${
              status === 'connected' ? 'bg-emerald-500' : status === 'error' ? 'bg-red-500' : 'bg-yellow-500'
            }`}
          />
          <span>
            {status === 'connected'
              ? '已连接'
              : status === 'connecting'
              ? '连接中...'
              : status === 'error'
              ? '连接异常'
              : '未连接'}
          </span>
        </div>
      </div>

      <div className="flex-1 bg-[#1e1e1e] p-4">
        <div ref={terminalContainerRef} className="h-full" />
      </div>

      <div className="border-t p-2 text-center text-xs text-muted-foreground">
        提示：支持复制 / 粘贴、窗口大小自动调整。连接关闭后请重新打开终端。
      </div>
    </div>
  );
}

function logDebug(message: string, payload?: unknown) {
  if (process.env.NODE_ENV !== 'production') {
    console.debug(`[WebSSH] ${message}`, payload);
  }
}
