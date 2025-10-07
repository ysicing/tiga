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
      terminal.writeln('Connected to SSH server...\r\n');
    };

    ws.onmessage = (event) => {
      // Receive output from server
      terminal.write(event.data);
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
            data: data,
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
            cols: terminal.cols,
            rows: terminal.rows,
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
        isFullscreen ? 'fixed inset-4 z-50' : ''
      } flex flex-col`}
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
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="flex-1 p-0">
        <div
          ref={terminalRef}
          className={`w-full ${
            isFullscreen ? 'h-full' : 'h-[400px]'
          } bg-[#1e1e1e]`}
        />
      </CardContent>
    </Card>
  );
}
