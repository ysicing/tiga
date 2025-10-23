# WebSocket协议契约：Docker容器终端

**功能分支**：`007-docker-docker-agent`
**创建日期**：2025-10-22
**协议版本**：v1.0.0
**状态**：草稿

---

## 概述

Docker容器Web终端通过WebSocket实现浏览器到容器的双向通信，用户可在浏览器中访问容器Shell。

**技术选型**（研究任务1决策）：
- 协议：WebSocket（复用K8s终端架构）
- 前端库：xterm.js + xterm-addon-fit + xterm-addon-attach
- 后端库：github.com/gorilla/websocket
- Agent端：docker exec -it <container> <shell>

**架构流程**：
```
浏览器 <--WebSocket--> Tiga Server <--gRPC流--> Agent <--docker exec--> 容器Shell
```

---

## 1. 终端会话创建

### 1.1 创建终端会话

**端点**：`POST /api/v1/docker/instances/:id/containers/:container_id/terminal`

**权限**：Operator

**请求体**：
```json
{
  "shell": "/bin/sh",
  "rows": 30,
  "cols": 120,
  "env": {
    "TERM": "xterm-256color"
  }
}
```

**参数说明**：
- `shell` (string, optional): Shell类型，默认 "/bin/sh"，可选 "/bin/bash"
- `rows` (integer, optional): 终端行数，默认30
- `cols` (integer, optional): 终端列数，默认120
- `env` (object, optional): 环境变量

**响应示例**：
```json
{
  "success": true,
  "data": {
    "session_id": "uuid",
    "ws_url": "ws://localhost:12306/api/v1/docker/terminal/uuid",
    "expires_at": "2025-10-22T11:00:00Z"
  }
}
```

**会话生命周期**：
- 创建后30分钟未连接WebSocket → 自动过期
- WebSocket连接后30分钟无活动 → 自动断开
- 客户端可发送心跳（ping）保持连接

**错误码**：
- `404 NOT_FOUND`: 容器不存在
- `400 BAD_REQUEST`: 容器未运行
- `503 SERVICE_UNAVAILABLE`: Docker实例离线

---

## 2. WebSocket连接

### 2.1 连接终端

**端点**：`WS /api/v1/docker/terminal/:session_id`

**查询参数**：
- `token` (string, required): JWT认证token（通过查询参数传递，因WebSocket不支持Authorization header）

**连接示例**：
```javascript
const ws = new WebSocket(
  `ws://localhost:12306/api/v1/docker/terminal/${sessionId}?token=${jwtToken}`
);

ws.onopen = () => {
  console.log('Terminal connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  handleMessage(message);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Terminal disconnected');
};
```

**连接握手**：
1. 客户端发起WebSocket连接
2. Server验证JWT token和session_id
3. Server通过Agent gRPC调用 `ExecContainer` 创建docker exec会话
4. 握手成功，返回101 Switching Protocols
5. Server发送欢迎消息（output类型）

**握手失败场景**：
- token无效 → 返回401，关闭连接
- session_id不存在或已过期 → 返回404，关闭连接
- 容器不存在或未运行 → 返回400，关闭连接

---

## 3. 消息协议

WebSocket消息使用JSON格式，包含两个方向：

### 3.1 客户端 → 服务器（输入消息）

**消息类型**：

#### (1) input - 终端输入

**格式**：
```json
{
  "type": "input",
  "data": "ls -la\n"
}
```

**说明**：
- `data` 字段为用户在终端输入的字符串
- 支持特殊字符（如 Ctrl+C 的 `\x03`，Tab 的 `\t`）
- 需要包含换行符 `\n` 触发命令执行

**示例**：
```javascript
// 发送命令
ws.send(JSON.stringify({
  type: 'input',
  data: 'ls -la\n'
}));

// 发送Ctrl+C
ws.send(JSON.stringify({
  type: 'input',
  data: '\x03'
}));
```

---

#### (2) resize - 调整终端大小

**格式**：
```json
{
  "type": "resize",
  "rows": 40,
  "cols": 150
}
```

**说明**：
- 浏览器窗口大小变化时发送
- Server转发到Agent，Agent调用docker exec API调整TTY大小
- 必须发送，否则终端显示错乱

**示例**：
```javascript
// 监听浏览器窗口大小变化
window.addEventListener('resize', () => {
  const { rows, cols } = term.getFitAddon().proposeDimensions();
  ws.send(JSON.stringify({
    type: 'resize',
    rows: rows,
    cols: cols
  }));
});
```

---

#### (3) ping - 心跳保活

**格式**：
```json
{
  "type": "ping"
}
```

**说明**：
- 客户端每10秒发送一次心跳
- Server响应pong消息
- 连续3次无响应 → 客户端主动断开连接

**示例**：
```javascript
// 心跳定时器
setInterval(() => {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'ping' }));
  }
}, 10000);
```

---

### 3.2 服务器 → 客户端（输出消息）

**消息类型**：

#### (1) output - 终端输出

**格式**：
```json
{
  "type": "output",
  "data": "total 48\ndrwxr-xr-x 2 root root 4096 Oct 22 10:00 bin\n..."
}
```

**说明**：
- `data` 字段为容器Shell的输出（stdout/stderr合并）
- 包含ANSI转义序列（颜色、格式控制）
- xterm.js自动解析和渲染

**示例**：
```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'output') {
    term.write(message.data); // xterm.js写入终端
  }
};
```

---

#### (2) error - 错误消息

**格式**：
```json
{
  "type": "error",
  "code": "EXEC_FAILED",
  "message": "Failed to execute command: container is not running"
}
```

**说明**：
- 仅在发生错误时发送
- 发送后Server自动关闭连接
- 客户端应显示错误消息给用户

**错误码列表**：

| 错误码 | 描述 |
|--------|------|
| EXEC_FAILED | docker exec执行失败 |
| CONTAINER_STOPPED | 容器已停止 |
| AGENT_DISCONNECTED | Agent断开连接 |
| SESSION_TIMEOUT | 会话超时 |
| INTERNAL_ERROR | 内部错误 |

**示例**：
```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'error') {
    console.error(`Terminal error: ${message.code} - ${message.message}`);
    term.write(`\r\n\x1b[31mError: ${message.message}\x1b[0m\r\n`);
  }
};
```

---

#### (3) pong - 心跳响应

**格式**：
```json
{
  "type": "pong"
}
```

**说明**：
- Server对ping消息的响应
- 客户端收到pong后重置超时计数器

---

#### (4) exit - 会话结束

**格式**：
```json
{
  "type": "exit",
  "exit_code": 0
}
```

**说明**：
- 容器Shell进程退出时发送
- `exit_code=0` 表示正常退出，非零表示异常退出
- 发送后Server自动关闭连接

**示例**：
```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'exit') {
    term.write(`\r\n\x1b[32mSession ended (exit code: ${message.exit_code})\x1b[0m\r\n`);
  }
};
```

---

## 4. 完整前端实现示例

### 4.1 React + xterm.js集成

```typescript
import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

interface DockerTerminalProps {
  instanceId: string;
  containerId: string;
  jwtToken: string;
}

export const DockerTerminal: React.FC<DockerTerminalProps> = ({
  instanceId,
  containerId,
  jwtToken,
}) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const [terminal, setTerminal] = useState<Terminal | null>(null);
  const [ws, setWs] = useState<WebSocket | null>(null);

  useEffect(() => {
    // 1. 创建终端实例
    const term = new Terminal({
      rows: 30,
      cols: 120,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
      },
      cursorBlink: true,
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    if (terminalRef.current) {
      term.open(terminalRef.current);
      fitAddon.fit();
    }

    setTerminal(term);

    // 2. 创建终端会话
    createTerminalSession()
      .then((session) => {
        // 3. 连接WebSocket
        const wsUrl = `ws://localhost:12306/api/v1/docker/terminal/${session.session_id}?token=${jwtToken}`;
        const websocket = new WebSocket(wsUrl);

        websocket.onopen = () => {
          console.log('Terminal connected');
        };

        websocket.onmessage = (event) => {
          const message = JSON.parse(event.data);

          switch (message.type) {
            case 'output':
              term.write(message.data);
              break;
            case 'error':
              term.write(`\r\n\x1b[31mError: ${message.message}\x1b[0m\r\n`);
              break;
            case 'exit':
              term.write(`\r\n\x1b[32mSession ended (exit code: ${message.exit_code})\x1b[0m\r\n`);
              websocket.close();
              break;
          }
        };

        websocket.onerror = (error) => {
          console.error('WebSocket error:', error);
          term.write('\r\n\x1b[31mConnection error\x1b[0m\r\n');
        };

        websocket.onclose = () => {
          console.log('Terminal disconnected');
          term.write('\r\n\x1b[33mConnection closed\x1b[0m\r\n');
        };

        setWs(websocket);

        // 4. 监听用户输入
        term.onData((data) => {
          if (websocket.readyState === WebSocket.OPEN) {
            websocket.send(JSON.stringify({
              type: 'input',
              data: data,
            }));
          }
        });

        // 5. 监听窗口大小变化
        const handleResize = () => {
          fitAddon.fit();
          const dimensions = fitAddon.proposeDimensions();
          if (dimensions && websocket.readyState === WebSocket.OPEN) {
            websocket.send(JSON.stringify({
              type: 'resize',
              rows: dimensions.rows,
              cols: dimensions.cols,
            }));
          }
        };

        window.addEventListener('resize', handleResize);

        // 6. 心跳保活
        const pingInterval = setInterval(() => {
          if (websocket.readyState === WebSocket.OPEN) {
            websocket.send(JSON.stringify({ type: 'ping' }));
          }
        }, 10000);

        // 清理
        return () => {
          window.removeEventListener('resize', handleResize);
          clearInterval(pingInterval);
          websocket.close();
          term.dispose();
        };
      })
      .catch((error) => {
        console.error('Failed to create terminal session:', error);
        term.write('\r\n\x1b[31mFailed to create terminal session\x1b[0m\r\n');
      });

    return () => {
      if (ws) ws.close();
      if (terminal) terminal.dispose();
    };
  }, [instanceId, containerId, jwtToken]);

  const createTerminalSession = async () => {
    const response = await fetch(
      `/api/v1/docker/instances/${instanceId}/containers/${containerId}/terminal`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${jwtToken}`,
        },
        body: JSON.stringify({
          shell: '/bin/sh',
          rows: 30,
          cols: 120,
        }),
      }
    );

    if (!response.ok) {
      throw new Error('Failed to create terminal session');
    }

    const data = await response.json();
    return data.data;
  };

  return (
    <div style={{ width: '100%', height: '100%' }}>
      <div ref={terminalRef} style={{ width: '100%', height: '100%' }} />
    </div>
  );
};
```

---

## 5. 后端实现要点

### 5.1 WebSocket处理器

```go
package handlers

import (
    "context"
    "encoding/json"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  8192,
    WriteBufferSize: 8192,
    CheckOrigin: func(r *http.Request) bool {
        return true // 生产环境需要检查Origin
    },
}

func (h *TerminalHandler) HandleDockerTerminal(c *gin.Context) {
    sessionID := c.Param("session_id")
    token := c.Query("token")

    // 1. 验证token
    claims, err := h.authService.ValidateToken(token)
    if err != nil {
        c.JSON(401, gin.H{"error": "Invalid token"})
        return
    }

    // 2. 验证session
    session, err := h.sessionService.GetSession(sessionID)
    if err != nil {
        c.JSON(404, gin.H{"error": "Session not found"})
        return
    }

    // 3. 升级到WebSocket
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    // 4. 通过Agent gRPC创建docker exec
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    stream, err := h.agentForwarder.ExecContainer(ctx)
    if err != nil {
        sendError(conn, "EXEC_FAILED", err.Error())
        return
    }

    // 发送启动命令
    stream.Send(&docker.ExecRequest{
        Request: &docker.ExecRequest_Start{
            Start: &docker.ExecStart{
                ContainerId: session.ContainerID,
                Cmd:         []string{session.Shell},
                Tty:         true,
                AttachStdin: true,
                AttachStdout: true,
                AttachStderr: true,
            },
        },
    })

    // 5. 双向转发：WebSocket <-> gRPC
    errChan := make(chan error, 2)

    // WebSocket -> gRPC
    go func() {
        for {
            var msg Message
            if err := conn.ReadJSON(&msg); err != nil {
                errChan <- err
                return
            }

            switch msg.Type {
            case "input":
                stream.Send(&docker.ExecRequest{
                    Request: &docker.ExecRequest_Input{
                        Input: &docker.ExecInput{
                            Data: []byte(msg.Data),
                        },
                    },
                })
            case "resize":
                stream.Send(&docker.ExecRequest{
                    Request: &docker.ExecRequest_Resize{
                        Resize: &docker.ExecResize{
                            Rows: int32(msg.Rows),
                            Cols: int32(msg.Cols),
                        },
                    },
                })
            case "ping":
                sendMessage(conn, Message{Type: "pong"})
            }
        }
    }()

    // gRPC -> WebSocket
    go func() {
        for {
            resp, err := stream.Recv()
            if err != nil {
                errChan <- err
                return
            }

            switch r := resp.Response.(type) {
            case *docker.ExecResponse_Output:
                sendMessage(conn, Message{
                    Type: "output",
                    Data: string(r.Output.Data),
                })
            case *docker.ExecResponse_Exit:
                sendMessage(conn, Message{
                    Type:     "exit",
                    ExitCode: int(r.Exit.ExitCode),
                })
                errChan <- nil
                return
            }
        }
    }()

    // 等待错误或退出
    <-errChan
}

type Message struct {
    Type     string `json:"type"`
    Data     string `json:"data,omitempty"`
    Rows     int    `json:"rows,omitempty"`
    Cols     int    `json:"cols,omitempty"`
    Code     string `json:"code,omitempty"`
    Message  string `json:"message,omitempty"`
    ExitCode int    `json:"exit_code,omitempty"`
}

func sendMessage(conn *websocket.Conn, msg Message) error {
    return conn.WriteJSON(msg)
}

func sendError(conn *websocket.Conn, code, message string) {
    sendMessage(conn, Message{
        Type:    "error",
        Code:    code,
        Message: message,
    })
    conn.Close()
}
```

---

## 6. 性能和限制

**并发连接限制**：
- 单Server最多支持100个并发终端连接
- 超出限制返回503错误

**会话超时**：
- 未连接：创建后30分钟
- 已连接：无活动30分钟自动断开

**消息大小限制**：
- 单条消息最大8KB
- 超出限制断开连接

**带宽限制**：
- 单连接最大带宽1MB/s
- 防止日志风暴占用所有带宽

---

## 7. 安全性

**认证**：
- JWT token通过查询参数传递（WebSocket不支持Authorization header）
- Token有效期检查
- Session归属检查（用户只能访问自己创建的session）

**授权**：
- 最低权限：Operator（操作员）
- Admin可访问所有容器终端

**防护措施**：
- 命令注入防护：仅允许预定义Shell（/bin/sh、/bin/bash）
- 超时保护：30分钟无活动自动断开
- 审计日志：记录所有终端访问（container_exec操作）

---

## 8. 故障处理

**Agent断开**：
- Server检测到Agent连接断开
- 发送error消息给客户端（AGENT_DISCONNECTED）
- 关闭WebSocket连接

**容器停止**：
- docker exec进程自动退出
- Server收到exit消息（exit_code非零）
- 转发给客户端并关闭连接

**网络抖动**：
- 心跳机制检测连接状态
- 连续3次ping无响应 → 客户端主动断开
- 客户端可重新创建session并连接

---

## 9. 测试覆盖

**契约测试**（tests/contract/docker/websocket_test.go）：
```go
func TestDockerTerminalWebSocket(t *testing.T) {
    // 1. 创建session
    session := createTerminalSession(t, instanceID, containerID)

    // 2. 连接WebSocket
    ws := connectWebSocket(t, session.SessionID, jwtToken)
    defer ws.Close()

    // 3. 发送input消息
    sendMessage(t, ws, Message{
        Type: "input",
        Data: "echo hello\n",
    })

    // 4. 接收output消息
    msg := receiveMessage(t, ws)
    assert.Equal(t, "output", msg.Type)
    assert.Contains(t, msg.Data, "hello")

    // 5. 发送resize消息
    sendMessage(t, ws, Message{
        Type: "resize",
        Rows: 40,
        Cols: 150,
    })

    // 6. 测试ping/pong
    sendMessage(t, ws, Message{Type: "ping"})
    msg = receiveMessage(t, ws)
    assert.Equal(t, "pong", msg.Type)

    // 7. 发送exit命令
    sendMessage(t, ws, Message{
        Type: "input",
        Data: "exit\n",
    })

    // 8. 接收exit消息
    msg = receiveMessage(t, ws)
    assert.Equal(t, "exit", msg.Type)
    assert.Equal(t, 0, msg.ExitCode)
}
```

---

**协议版本**：v1.0.0
**创建时间**：2025-10-22
**状态**：草稿，待契约测试验证
**参考实现**：`pkg/kube/terminal.go`（K8s节点终端）
