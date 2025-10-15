# API契约:主机管理

**版本**:v1
**基础路径**:`/api/v1/hosts`

## 1. 主机节点管理

### 1.1 创建主机节点

**端点**:`POST /api/v1/hosts`

**请求**:
```json
{
  "name": "prod-server-01",
  "note": "生产环境Web服务器",
  "public_note": "公开备注信息",
  "display_index": 100,
  "hide_for_guest": false,
  "enable_webssh": true,
  "ssh_port": 22,
  "ssh_user": "root",
  "group_ids": [1, 2, 3]
}
```

**响应**(201 Created):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "name": "prod-server-01",
    "secret_key": "encrypted_secret_key_here",
    "agent_install_cmd": "curl -fsSL https://tiga.example.com/agent/install.sh | bash -s -- --server tiga.example.com:12307 --uuid 550e8400-e29b-41d4-a716-446655440000 --key encrypted_secret_key_here",
    "note": "生产环境Web服务器",
    "public_note": "公开备注信息",
    "display_index": 100,
    "hide_for_guest": false,
    "enable_webssh": true,
    "ssh_port": 22,
    "ssh_user": "root",
    "group_ids": [1, 2, 3],
    "created_at": "2025-10-07T10:00:00Z",
    "updated_at": "2025-10-07T10:00:00Z"
  }
}
```

---

### 1.2 获取主机列表

**端点**:`GET /api/v1/hosts`

**查询参数**:
- `page`:页码(默认1)
- `page_size`:每页数量(默认20,最大100)
- `group_id`:分组ID过滤
- `online`:在线状态过滤(true/false)
- `search`:搜索关键词(主机名/备注)
- `sort`:排序字段(display_index/-display_index/name/-name/created_at/-created_at)

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "uuid": "550e8400-e29b-41d4-a716-446655440000",
        "name": "prod-server-01",
        "note": "生产环境Web服务器",
        "public_note": "公开备注信息",
        "display_index": 100,
        "hide_for_guest": false,
        "enable_webssh": true,
        "group_ids": [1, 2],
        "online": true,
        "last_active": "2025-10-07T10:05:00Z",
        "host_info": {
          "platform": "linux",
          "platform_version": "Ubuntu 22.04",
          "arch": "amd64",
          "cpu_model": "Intel Xeon E5-2680",
          "cpu_cores": 16,
          "mem_total": 34359738368,
          "disk_total": 1099511627776
        },
        "current_state": {
          "cpu_usage": 35.5,
          "mem_usage": 68.2,
          "disk_usage": 45.8,
          "net_in_speed": 1024000,
          "net_out_speed": 512000
        },
        "created_at": "2025-10-07T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 1.3 获取主机详情

**端点**:`GET /api/v1/hosts/{id}`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "name": "prod-server-01",
    "note": "生产环境Web服务器",
    "public_note": "公开备注信息",
    "display_index": 100,
    "hide_for_guest": false,
    "enable_webssh": true,
    "ssh_port": 22,
    "ssh_user": "root",
    "group_ids": [1, 2],
    "online": true,
    "last_active": "2025-10-07T10:05:00Z",
    "host_info": {
      "platform": "linux",
      "platform_version": "Ubuntu 22.04",
      "arch": "amd64",
      "virtualization": "kvm",
      "cpu_model": "Intel Xeon E5-2680",
      "cpu_cores": 16,
      "mem_total": 34359738368,
      "disk_total": 1099511627776,
      "swap_total": 4294967296,
      "agent_version": "1.0.0",
      "boot_time": 1696320000
    },
    "agent_connection": {
      "status": "online",
      "connected_at": "2025-10-07T09:00:00Z",
      "last_heartbeat": "2025-10-07T10:05:00Z",
      "agent_version": "1.0.0",
      "ip_address": "192.168.1.100"
    },
    "created_at": "2025-10-07T10:00:00Z",
    "updated_at": "2025-10-07T10:00:00Z"
  }
}
```

---

### 1.4 更新主机节点

**端点**:`PUT /api/v1/hosts/{id}`

**请求**:
```json
{
  "name": "prod-server-01-updated",
  "note": "更新后的备注",
  "public_note": "更新后的公开备注",
  "display_index": 200,
  "hide_for_guest": true,
  "enable_webssh": true,
  "ssh_port": 2222,
  "ssh_user": "admin",
  "group_ids": [1, 3]
}
```

**响应**(200 OK):返回更新后的主机对象

---

### 1.5 删除主机节点

**端点**:`DELETE /api/v1/hosts/{id}`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "主机已删除"
}
```

---

## 2. 主机监控数据

### 2.1 获取实时状态

**端点**:`GET /api/v1/hosts/{id}/state/current`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "timestamp": "2025-10-07T10:05:00Z",
    "cpu_usage": 35.5,
    "load_1": 2.5,
    "load_5": 2.1,
    "load_15": 1.8,
    "mem_used": 23456789012,
    "mem_usage": 68.2,
    "swap_used": 1048576000,
    "disk_used": 503316480000,
    "disk_usage": 45.8,
    "net_in_transfer": 1234567890123,
    "net_out_transfer": 987654321098,
    "net_in_speed": 1024000,
    "net_out_speed": 512000,
    "tcp_conn_count": 150,
    "udp_conn_count": 50,
    "process_count": 200,
    "uptime": 864000,
    "temperatures": [
      {"name": "CPU", "temperature": 55.5},
      {"name": "Disk", "temperature": 42.0}
    ],
    "gpu_usage": 25.3
  }
}
```

---

### 2.2 获取历史监控数据

**端点**:`GET /api/v1/hosts/{id}/state/history`

**查询参数**:
- `start`:开始时间(RFC3339格式,必填)
- `end`:结束时间(RFC3339格式,必填)
- `interval`:数据间隔(auto/1m/5m/1h/1d,默认auto)
- `metrics`:指标列表(逗号分隔,如cpu_usage,mem_usage)

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "start": "2025-10-07T09:00:00Z",
    "end": "2025-10-07T10:00:00Z",
    "interval": "5m",
    "points": [
      {
        "timestamp": "2025-10-07T09:00:00Z",
        "cpu_usage": 30.2,
        "mem_usage": 65.8
      },
      {
        "timestamp": "2025-10-07T09:05:00Z",
        "cpu_usage": 32.5,
        "mem_usage": 66.2
      }
    ]
  }
}
```

---

## 3. 主机分组

### 3.1 创建分组

**端点**:`POST /api/v1/host-groups`

**请求**:
```json
{
  "name": "生产环境",
  "description": "所有生产环境服务器"
}
```

**响应**(201 Created):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "生产环境",
    "description": "所有生产环境服务器",
    "host_count": 0,
    "created_at": "2025-10-07T10:00:00Z",
    "updated_at": "2025-10-07T10:00:00Z"
  }
}
```

---

### 3.2 获取分组列表

**端点**:`GET /api/v1/host-groups`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "name": "生产环境",
      "description": "所有生产环境服务器",
      "host_count": 5,
      "created_at": "2025-10-07T10:00:00Z"
    }
  ]
}
```

---

### 3.3 批量添加主机到分组

**端点**:`POST /api/v1/host-groups/{id}/hosts`

**请求**:
```json
{
  "host_ids": [1, 2, 3, 4, 5]
}
```

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "已添加5个主机到分组"
}
```

---

### 3.4 从分组移除主机

**端点**:`DELETE /api/v1/host-groups/{id}/hosts/{host_id}`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "已从分组移除主机"
}
```

---

## 4. WebSSH

### 4.1 创建WebSSH会话

**端点**:`POST /api/v1/hosts/{id}/webssh`

**请求**:
```json
{
  "width": 80,
  "height": 24
}
```

**响应**(201 Created):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_550e8400-e29b-41d4-a716-446655440000",
    "websocket_url": "wss://tiga.example.com/api/v1/webssh/sess_550e8400-e29b-41d4-a716-446655440000",
    "host_id": 1,
    "host_name": "prod-server-01"
  }
}
```

---

### 4.2 WebSocket连接(终端数据流)

**端点**:`GET /api/v1/webssh/{session_id}` (WebSocket升级)

**WebSocket消息格式**:

**输入消息**(浏览器 → Server):
```json
{
  "type": "input",
  "data": "ls -la\n"
}
```

**调整窗口大小**:
```json
{
  "type": "resize",
  "cols": 120,
  "rows": 30
}
```

**输出消息**(Server → 浏览器):
```json
{
  "type": "output",
  "data": "total 48\ndrwxr-xr-x  6 root root 4096 Oct  7 10:00 .\n..."
}
```

**错误消息**:
```json
{
  "type": "error",
  "message": "SSH连接失败: Connection refused"
}
```

**会话关闭**:
```json
{
  "type": "close",
  "reason": "session timeout"
}
```

---

### 4.3 获取WebSSH会话列表

**端点**:`GET /api/v1/webssh/sessions`

**查询参数**:
- `host_id`:主机ID过滤
- `user_id`:用户ID过滤
- `status`:状态过滤(active/closed)

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "session_id": "sess_550e8400-e29b-41d4-a716-446655440000",
      "user_id": 1,
      "user_name": "admin",
      "host_id": 1,
      "host_name": "prod-server-01",
      "status": "active",
      "start_time": "2025-10-07T10:00:00Z",
      "last_active": "2025-10-07T10:05:00Z",
      "client_ip": "192.168.1.200"
    }
  ]
}
```

---

### 4.4 关闭WebSSH会话

**端点**:`DELETE /api/v1/webssh/{session_id}`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "会话已关闭"
}
```

---

## 错误响应

所有错误响应遵循统一格式:

```json
{
  "code": 40001,
  "message": "主机UUID已存在",
  "details": {
    "field": "uuid",
    "value": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**常见错误码**:
- `40001`:参数验证失败
- `40002`:主机不存在
- `40003`:WebSSH未启用
- `40101`:未授权访问
- `40301`:权限不足
- `40401`:资源不存在
- `50001`:服务器内部错误
- `50002`:Agent离线
- `50003`:WebSSH连接失败

---

## 认证与授权

所有API请求需要JWT认证:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

WebSocket连接通过查询参数传递Token:

```
wss://tiga.example.com/api/v1/webssh/{session_id}?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**权限要求**:
- 创建/更新/删除主机:需要`hosts:write`权限
- 查看主机信息:需要`hosts:read`权限
- 访问WebSSH:需要`hosts:webssh`权限
- 管理分组:需要`host_groups:write`权限
