# 快速开始:主机管理子系统验证

**功能分支**:`002-nezha-webssh`
**测试目标**:验证主机监控、服务探测和WebSSH功能

## 前置条件

1. **环境准备**:
   - Tiga Server已启动并可访问
   - 至少一台测试主机(Linux/Windows/macOS)可用于部署Agent
   - 测试主机与Server网络可达
   - 已创建管理员用户并获取JWT Token

2. **权限检查**:
   - 当前用户拥有`hosts:write`、`hosts:webssh`、`service_monitors:write`、`alert_rules:write`权限

3. **工具准备**:
   - curl或Postman(API测试)
   - 浏览器(WebSocket/WebSSH测试)
   - SSH客户端(Agent部署)

---

## 场景1:主机节点添加与Agent连接

### 步骤1:创建主机节点

**API请求**:
```bash
curl -X POST "http://localhost:12306/api/v1/hosts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-host-01",
    "note": "测试主机01",
    "enable_webssh": true,
    "ssh_port": 22,
    "ssh_user": "root"
  }'
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "name": "test-host-01",
    "secret_key": "encrypted_key_here",
    "agent_install_cmd": "curl -fsSL http://localhost:12306/agent/install.sh | bash -s -- --server http://localhost:12306 --uuid 550e8400-e29b-41d4-a716-446655440000 --key encrypted_key_here"
  }
}
```

**验证要点**:
- ✓ 返回状态码201 Created
- ✓ UUID唯一且不为空
- ✓ agent_install_cmd包含正确的Server地址、UUID和密钥

---

### 步骤2:部署Agent

**在测试主机上执行**:
```bash
# 复制agent_install_cmd并执行
curl -fsSL http://localhost:12306/agent/install.sh | bash -s -- --server http://localhost:12306 --uuid 550e8400-e29b-41d4-a716-446655440000 --key encrypted_key_here
```

**期望结果**:
```
[INFO] Downloading Tiga Agent...
[INFO] Installing Agent to /usr/local/bin/tiga-agent...
[INFO] Creating systemd service...
[INFO] Starting Tiga Agent...
[SUCCESS] Agent installed and started successfully
[INFO] Agent version: 1.0.0
[INFO] Server: http://localhost:12306
```

**验证要点**:
- ✓ Agent二进制下载成功
- ✓ systemd服务启动成功
- ✓ Agent进程运行中(`systemctl status tiga-agent`)

---

### 步骤3:验证Agent连接

**API请求**:
```bash
curl -X GET "http://localhost:12306/api/v1/hosts/1" \
  -H "Authorization: Bearer $TOKEN"
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "name": "test-host-01",
    "online": true,
    "last_active": "2025-10-07T10:05:00Z",
    "host_info": {
      "platform": "linux",
      "platform_version": "Ubuntu 22.04",
      "arch": "amd64",
      "cpu_cores": 4,
      "mem_total": 8589934592
    },
    "agent_connection": {
      "status": "online",
      "connected_at": "2025-10-07T10:00:00Z",
      "last_heartbeat": "2025-10-07T10:05:00Z",
      "agent_version": "1.0.0"
    }
  }
}
```

**验证要点**:
- ✓ online字段为true
- ✓ host_info包含正确的系统信息
- ✓ agent_connection.status为"online"
- ✓ last_heartbeat在最近60秒内

---

## 场景2:实时监控数据

### 步骤1:获取实时状态

**API请求**:
```bash
curl -X GET "http://localhost:12306/api/v1/hosts/1/state/current" \
  -H "Authorization: Bearer $TOKEN"
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "timestamp": "2025-10-07T10:05:00Z",
    "cpu_usage": 25.3,
    "mem_usage": 45.8,
    "disk_usage": 32.1,
    "net_in_speed": 102400,
    "net_out_speed": 51200,
    "load_1": 1.5,
    "tcp_conn_count": 120,
    "process_count": 180,
    "uptime": 864000
  }
}
```

**验证要点**:
- ✓ 所有监控指标数值在合理范围内
- ✓ cpu_usage在0-100之间
- ✓ timestamp为当前时间(±30秒)

---

### 步骤2:订阅实时数据推送

**WebSocket连接**:
```javascript
const ws = new WebSocket('ws://localhost:12306/api/v1/ws/host-monitor?token=' + TOKEN);

ws.onopen = () => {
  // 订阅主机1的监控数据
  ws.send(JSON.stringify({
    action: 'subscribe',
    host_ids: [1]
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
  /*
  {
    "type": "host_state",
    "host_id": 1,
    "data": {
      "timestamp": "2025-10-07T10:05:30Z",
      "cpu_usage": 26.1,
      "mem_usage": 45.9,
      ...
    }
  }
  */
};
```

**验证要点**:
- ✓ WebSocket连接成功
- ✓ 每30秒收到新的监控数据
- ✓ 数据格式正确且包含所有指标

---

### 步骤3:查询历史数据

**API请求**:
```bash
curl -X GET "http://localhost:12306/api/v1/hosts/1/state/history?start=2025-10-07T09:00:00Z&end=2025-10-07T10:00:00Z&interval=5m&metrics=cpu_usage,mem_usage" \
  -H "Authorization: Bearer $TOKEN"
```

**期望响应**:
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
        "cpu_usage": 20.5,
        "mem_usage": 42.3
      },
      {
        "timestamp": "2025-10-07T09:05:00Z",
        "cpu_usage": 22.1,
        "mem_usage": 43.1
      }
      // ... 更多数据点
    ]
  }
}
```

**验证要点**:
- ✓ 返回数据点数量符合时间范围和间隔
- ✓ 数据按时间升序排列
- ✓ 仅包含请求的metrics字段

---

## 场景3:服务探测

### 步骤1:创建HTTP服务探测

**API请求**:
```bash
curl -X POST "http://localhost:12306/api/v1/service-monitors" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试网站可用性",
    "enable": true,
    "type": 1,
    "target": "https://www.google.com",
    "duration": 60,
    "timeout": 10,
    "retry": 3,
    "execute_on": {
      "type": "hosts",
      "host_ids": [1]
    },
    "enable_alert": true,
    "fail_threshold": 3,
    "notification_id": 1
  }'
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "测试网站可用性",
    "enable": true,
    "type": 1,
    "target": "https://www.google.com",
    "duration": 60
  }
}
```

**验证要点**:
- ✓ 返回状态码201 Created
- ✓ 探测规则已创建并启用

---

### 步骤2:等待探测执行

**等待时间**:约60秒(探测频率)

**查看探测历史**:
```bash
curl -X GET "http://localhost:12306/api/v1/service-monitors/1/probe-history?limit=5" \
  -H "Authorization: Bearer $TOKEN"
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "service_monitor_id": 1,
      "host_id": 1,
      "host_name": "test-host-01",
      "timestamp": "2025-10-07T10:06:00Z",
      "success": true,
      "latency": 0.152,
      "status_code": 200
    }
  ]
}
```

**验证要点**:
- ✓ 探测结果已记录
- ✓ success字段为true
- ✓ status_code为200
- ✓ latency值合理(< timeout)

---

### 步骤3:查看可用性统计

**API请求**:
```bash
curl -X GET "http://localhost:12306/api/v1/service-monitors/1/availability?period=hour&start=2025-10-07T10:00:00Z&end=2025-10-07T11:00:00Z" \
  -H "Authorization: Bearer $TOKEN"
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "period": "hour",
      "start_time": "2025-10-07T10:00:00Z",
      "total_probes": 5,
      "success_probes": 5,
      "fail_probes": 0,
      "availability": 100.0,
      "avg_latency": 0.145
    }
  ]
}
```

**验证要点**:
- ✓ availability计算正确(success_probes / total_probes * 100)
- ✓ avg_latency为所有探测延迟的平均值

---

## 场景4:告警规则与事件

### 步骤1:创建告警规则

**API请求**:
```bash
curl -X POST "http://localhost:12306/api/v1/alert-rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CPU使用率告警",
    "enable": true,
    "type": "host_monitor",
    "target_ids": [1],
    "condition": "cpu_usage > 80",
    "duration": 60,
    "notification_group_id": 1,
    "alert_level": "warning",
    "silence_period": 300
  }'
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "CPU使用率告警",
    "enable": true,
    "condition": "cpu_usage > 80"
  }
}
```

---

### 步骤2:模拟告警触发

**在测试主机上执行CPU压力测试**:
```bash
# 使用stress或yes命令制造CPU压力
stress --cpu 4 --timeout 120s
# 或
yes > /dev/null &
yes > /dev/null &
```

**等待60秒(告警持续时长)**,然后查询告警事件:
```bash
curl -X GET "http://localhost:12306/api/v1/alert-events?status=firing" \
  -H "Authorization: Bearer $TOKEN"
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "alert_rule_name": "CPU使用率告警",
        "target_name": "test-host-01",
        "trigger_time": "2025-10-07T10:10:00Z",
        "status": "firing",
        "level": "warning",
        "title": "主机CPU使用率过高",
        "message": "主机test-host-01 CPU使用率85.2%,超过阈值80%",
        "current_value": {
          "cpu_usage": 85.2
        }
      }
    ]
  }
}
```

**验证要点**:
- ✓ 告警事件已触发
- ✓ status为"firing"
- ✓ current_value中cpu_usage > 80
- ✓ 收到通知(检查邮件/Webhook)

---

### 步骤3:确认告警

**API请求**:
```bash
curl -X POST "http://localhost:12306/api/v1/alert-events/1/acknowledge" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "note": "已知晓,正在处理"
  }'
```

**期望响应**:
```json
{
  "code": 0,
  "message": "告警已确认",
  "data": {
    "acknowledged": true,
    "ack_user": {
      "id": 1,
      "name": "admin"
    },
    "ack_time": "2025-10-07T10:12:00Z"
  }
}
```

---

### 步骤4:停止压力测试并验证恢复

**停止CPU压力**:
```bash
pkill stress
# 或
pkill yes
```

**等待监控数据恢复(约30-60秒)**,查询告警事件:
```bash
curl -X GET "http://localhost:12306/api/v1/alert-events/1" \
  -H "Authorization: Bearer $TOKEN"
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "status": "resolved",
    "trigger_time": "2025-10-07T10:10:00Z",
    "recover_time": "2025-10-07T10:13:00Z"
  }
}
```

**验证要点**:
- ✓ status变为"resolved"
- ✓ recover_time已设置
- ✓ 收到恢复通知

---

## 场景5:WebSSH终端

### 步骤1:创建WebSSH会话

**API请求**:
```bash
curl -X POST "http://localhost:12306/api/v1/hosts/1/webssh" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "width": 80,
    "height": 24
  }'
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "sess_550e8400-e29b-41d4-a716-446655440000",
    "websocket_url": "ws://localhost:12306/api/v1/webssh/sess_550e8400-e29b-41d4-a716-446655440000",
    "host_id": 1,
    "host_name": "test-host-01"
  }
}
```

**验证要点**:
- ✓ 返回状态码201 Created
- ✓ session_id为有效UUID
- ✓ websocket_url格式正确

---

### 步骤2:连接WebSocket并交互

**JavaScript代码**:
```javascript
const wsUrl = 'ws://localhost:12306/api/v1/webssh/sess_550e8400-e29b-41d4-a716-446655440000?token=' + TOKEN;
const ws = new WebSocket(wsUrl);

ws.onopen = () => {
  console.log('WebSSH connected');
  // 发送命令
  ws.send(JSON.stringify({
    type: 'input',
    data: 'ls -la\n'
  }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'output') {
    console.log('Terminal output:', msg.data);
    // 期望看到ls -la的输出
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};
```

**验证要点**:
- ✓ WebSocket连接成功
- ✓ 发送命令后收到输出
- ✓ 输出内容正确(如ls -la的文件列表)
- ✓ 支持交互式命令(如vim、top)

---

### 步骤3:调整终端大小

**WebSocket消息**:
```javascript
ws.send(JSON.stringify({
  type: 'resize',
  cols: 120,
  rows: 30
}));
```

**验证要点**:
- ✓ 调整后终端显示适应新尺寸
- ✓ 再次执行命令输出宽度正确

---

### 步骤4:关闭会话

**API请求**:
```bash
curl -X DELETE "http://localhost:12306/api/v1/webssh/sess_550e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer $TOKEN"
```

**期望响应**:
```json
{
  "code": 0,
  "message": "会话已关闭"
}
```

**验证要点**:
- ✓ WebSocket连接自动断开
- ✓ 收到close消息(type='close')
- ✓ 会话状态更新为"closed"

---

## 场景6:主机分组

### 步骤1:创建主机分组

**API请求**:
```bash
curl -X POST "http://localhost:12306/api/v1/host-groups" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试环境",
    "description": "所有测试环境服务器"
  }'
```

**期望响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "测试环境",
    "description": "所有测试环境服务器",
    "host_count": 0
  }
}
```

---

### 步骤2:添加主机到分组

**API请求**:
```bash
curl -X POST "http://localhost:12306/api/v1/host-groups/1/hosts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "host_ids": [1]
  }'
```

**期望响应**:
```json
{
  "code": 0,
  "message": "已添加1个主机到分组"
}
```

**验证要点**:
- ✓ 分组的host_count增加
- ✓ 主机详情中group_ids包含该分组ID

---

### 步骤3:基于分组的服务探测

**创建分组级探测规则**:
```bash
curl -X POST "http://localhost:12306/api/v1/service-monitors" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试环境内网服务",
    "enable": true,
    "type": 2,
    "target": "192.168.1.100:3306",
    "duration": 30,
    "execute_on": {
      "type": "group",
      "group_id": 1
    }
  }'
```

**验证要点**:
- ✓ 探测任务在分组内所有主机上执行
- ✓ 探测结果关联正确的host_id

---

## 验收标准

### 功能完整性
- [x] 主机节点CRUD操作正常
- [x] Agent部署和连接成功
- [x] 实时监控数据准确
- [x] 历史数据查询正确
- [x] 服务探测功能工作
- [x] 可用性统计准确
- [x] 告警规则触发正常
- [x] 告警通知发送成功
- [x] WebSSH终端可用
- [x] 主机分组功能正常
- [x] WebSocket实时推送工作

### 性能指标
- [ ] Agent资源占用:<30MB内存,<5% CPU
- [ ] 监控数据延迟:<30秒
- [ ] 服务探测并发:>100个规则
- [ ] WebSSH延迟:<200ms
- [ ] API响应时间:<500ms
- [ ] 历史数据查询:7天数据<2秒

### 安全性
- [ ] Agent连接需要有效密钥
- [ ] API访问需要JWT认证
- [ ] WebSSH需要权限验证
- [ ] 敏感数据加密存储
- [ ] 操作日志完整记录

---

## 清理步骤

测试完成后执行清理:

1. **删除服务探测规则**:
```bash
curl -X DELETE "http://localhost:12306/api/v1/service-monitors/1" -H "Authorization: Bearer $TOKEN"
curl -X DELETE "http://localhost:12306/api/v1/service-monitors/2" -H "Authorization: Bearer $TOKEN"
```

2. **删除告警规则**:
```bash
curl -X DELETE "http://localhost:12306/api/v1/alert-rules/1" -H "Authorization: Bearer $TOKEN"
```

3. **删除主机分组**:
```bash
curl -X DELETE "http://localhost:12306/api/v1/host-groups/1" -H "Authorization: Bearer $TOKEN"
```

4. **删除主机节点**:
```bash
curl -X DELETE "http://localhost:12306/api/v1/hosts/1" -H "Authorization: Bearer $TOKEN"
```

5. **卸载Agent**(在测试主机上):
```bash
systemctl stop tiga-agent
systemctl disable tiga-agent
rm -f /usr/local/bin/tiga-agent
rm -f /etc/systemd/system/tiga-agent.service
systemctl daemon-reload
```

---

## 故障排查

### Agent无法连接
- 检查网络连通性:`telnet <server_ip> 12306`
- 检查Agent日志:`journalctl -u tiga-agent -f`
- 验证密钥正确性
- 检查防火墙规则

### 监控数据不更新
- 确认Agent在线状态
- 检查Agent上报日志
- 验证数据库连接正常
- 检查Server端处理器运行

### WebSSH连接失败
- 确认主机已启用WebSSH
- 检查SSH端口和用户配置
- 验证用户权限
- 检查WebSocket连接日志

### 告警未触发
- 验证告警规则已启用
- 检查条件表达式正确性
- 确认duration已满足
- 检查通知组配置

---

**测试完成标志**:所有验收标准项全部通过,无功能缺陷和性能问题
