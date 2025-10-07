import { useEffect, useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useHostStore } from '@/stores/host-store';
import { Button } from '@/components/ui/button';
import { ArrowLeft } from 'lucide-react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import { toast } from 'sonner';

export function HostSSHPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { hosts } = useHostStore();
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalInstanceRef = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [connected, setConnected] = useState(false);

  const host = hosts.find((h) => h.id === id);

  useEffect(() => {
    if (!terminalRef.current || !host || !host.host_info?.ssh_enabled) return;

    // Create terminal instance
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
      rows: 30,
      cols: 100,
    });

    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.open(terminalRef.current);
    fitAddon.fit();

    terminalInstanceRef.current = term;

    // Connect WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/vms/hosts/${id}/ssh/connect`;
    const websocket = new WebSocket(wsUrl);

    websocket.onopen = () => {
      setConnected(true);
      term.writeln('\x1b[1;32m=== WebSSH Terminal ===\x1b[0m');
      term.writeln(`\x1b[1;36mConnecting to ${host.name}...\x1b[0m`);
      term.writeln('');

      // Send authentication token
      const token = localStorage.getItem('token');
      if (token) {
        websocket.send(JSON.stringify({ type: 'auth', token }));
      }
    };

    websocket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'output') {
          term.write(data.data);
        } else if (data.type === 'error') {
          term.writeln(`\x1b[1;31mError: ${data.message}\x1b[0m`);
        }
      } catch {
        // Binary data, write directly
        term.write(event.data);
      }
    };

    websocket.onerror = (error) => {
      console.error('WebSocket error:', error);
      toast.error('SSH 连接失败');
      term.writeln('\x1b[1;31mConnection error\x1b[0m');
    };

    websocket.onclose = () => {
      setConnected(false);
      term.writeln('');
      term.writeln('\x1b[1;33mConnection closed\x1b[0m');
    };

    // Handle terminal input
    term.onData((data) => {
      if (websocket.readyState === WebSocket.OPEN) {
        websocket.send(JSON.stringify({ type: 'input', data }));
      }
    });

    wsRef.current = websocket;

    // Handle window resize
    const handleResize = () => {
      fitAddon.fit();
      if (websocket.readyState === WebSocket.OPEN) {
        websocket.send(
          JSON.stringify({
            type: 'resize',
            rows: term.rows,
            cols: term.cols,
          })
        );
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      websocket.close();
      term.dispose();
    };
  }, [id, host]);

  if (!host) {
    return (
      <div className="text-center py-12">
        <p>主机未找到</p>
        <Button onClick={() => navigate('/vms/hosts')} className="mt-4">
          返回列表
        </Button>
      </div>
    );
  }

  if (!host.host_info?.ssh_enabled) {
    return (
      <div className="text-center py-12">
        <p>此主机 SSH 服务未启用或未被 Agent 检测到</p>
        <Button onClick={() => navigate(`/vms/hosts/${id}`)} className="mt-4">
          返回主机详情
        </Button>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(`/vms/hosts/${id}`)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-xl font-bold">WebSSH Terminal</h1>
            <p className="text-sm text-muted-foreground">
              {host.name} - {host.host_info?.ssh_user || 'root'}@localhost:{host.host_info?.ssh_port || 22}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <div
            className={`h-2 w-2 rounded-full ${
              connected ? 'bg-green-500' : 'bg-red-500'
            }`}
          />
          <span className="text-sm text-muted-foreground">
            {connected ? '已连接' : '未连接'}
          </span>
        </div>
      </div>

      {/* Terminal */}
      <div className="flex-1 p-4 bg-[#1e1e1e]">
        <div ref={terminalRef} className="h-full" />
      </div>

      {/* Footer */}
      <div className="p-2 border-t text-xs text-muted-foreground text-center">
        提示：使用 Ctrl+C 可以发送中断信号 | 支持复制粘贴
      </div>
    </div>
  );
}
