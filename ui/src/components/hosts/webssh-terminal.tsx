import { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { X, Maximize2, Minimize2 } from 'lucide-react';

interface WebSSHTerminalProps {
  sessionId: string;
  hostName?: string;
  onClose?: () => void;
}

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
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isConnected, setIsConnected] = useState(false);

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

    // Connect to WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsURL = `${protocol}//${window.location.host}/api/v1/vms/webssh/${sessionId}`;
    const ws = new WebSocket(wsURL);

    ws.onopen = () => {
      console.log('[WebSSH] Connected');
      setIsConnected(true);
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
            break;
          case 'ping':
            // Respond to ping with pong
            ws.send(JSON.stringify({ type: 'pong', timestamp: Date.now() }));
            break;
          case 'disconnected':
            setIsConnected(false);
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
    };

    ws.onclose = () => {
      console.log('[WebSSH] Disconnected');
      setIsConnected(false);
      terminal.writeln('\r\n\x1b[33mConnection closed\x1b[0m\r\n');
    };

    wsRef.current = ws;

    // Send terminal input to server
    terminal.onData((data) => {
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
    });

    // Handle window resize
    const handleResize = () => {
      fitAddon.fit();
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

    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize);
      ws.close();
      terminal.dispose();
    };
  }, [sessionId]);

  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen);
    setTimeout(() => {
      fitAddonRef.current?.fit();
    }, 100);
  };

  const handleClose = () => {
    if (wsRef.current) {
      wsRef.current.close();
    }
    onClose?.();
  };

  return (
    <Card
      className={`${
        isFullscreen ? 'fixed inset-0 z-50 rounded-none' : ''
      } flex flex-col h-full`}
    >
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-base font-medium flex items-center gap-2">
          <span className={`h-2 w-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
          {hostName}
        </CardTitle>
        <div className="flex items-center gap-2">
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
          <Button
            variant="ghost"
            size="sm"
            onClick={handleClose}
            title="关闭终端"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="flex-1 p-0 overflow-hidden">
        <div
          ref={terminalRef}
          className="w-full h-full bg-[#1e1e1e]"
        />
      </CardContent>
    </Card>
  );
}
