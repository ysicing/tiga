# API契约:服务探测与告警

**版本**:v1
**基础路径**:`/api/v1`

## 1. 服务探测规则

### 1.1 创建服务探测规则

**端点**:`POST /api/v1/service-monitors`

**请求**:
```json
{
  "name": "网站可用性监控",
  "enable": true,
  "type": 1,
  "target": "https://example.com",
  "duration": 60,
  "timeout": 10,
  "retry": 3,
  "execute_on": {
    "type": "group",
    "group_id": 1
  },
  "skip_hosts": [2, 3],
  "enable_alert": true,
  "fail_threshold": 3,
  "notification_id": 1,
  "latency_alert": true,
  "max_latency": 2.0
}
```

**字段说明**:
- `type`:探测类型(1=HTTP, 2=TCP, 3=ICMP)
- `target`:目标地址(URL/IP:Port/IP)
- `duration`:探测频率(秒)
- `execute_on.type`:执行范围类型(all=所有主机/group=分组/hosts=指定主机)
- `skip_hosts`:跳过的主机ID列表
- `max_latency`:最大延迟阈值(秒)

**响应**(201 Created):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "网站可用性监控",
    "enable": true,
    "type": 1,
    "target": "https://example.com",
    "duration": 60,
    "timeout": 10,
    "retry": 3,
    "execute_on": {
      "type": "group",
      "group_id": 1
    },
    "skip_hosts": [2, 3],
    "enable_alert": true,
    "fail_threshold": 3,
    "notification_id": 1,
    "latency_alert": true,
    "max_latency": 2.0,
    "created_at": "2025-10-07T10:00:00Z",
    "updated_at": "2025-10-07T10:00:00Z"
  }
}
```

---

### 1.2 获取服务探测规则列表

**端点**:`GET /api/v1/service-monitors`

**查询参数**:
- `page`:页码
- `page_size`:每页数量
- `enable`:启用状态过滤
- `type`:探测类型过滤
- `search`:搜索关键词

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "网站可用性监控",
        "enable": true,
        "type": 1,
        "target": "https://example.com",
        "duration": 60,
        "current_status": "online",
        "availability_24h": 99.5,
        "avg_latency_24h": 0.35,
        "last_probe_time": "2025-10-07T10:05:00Z",
        "last_probe_result": "success",
        "created_at": "2025-10-07T09:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 1.3 获取服务探测详情

**端点**:`GET /api/v1/service-monitors/{id}`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "网站可用性监控",
    "enable": true,
    "type": 1,
    "target": "https://example.com",
    "duration": 60,
    "timeout": 10,
    "retry": 3,
    "execute_on": {
      "type": "group",
      "group_id": 1,
      "group_name": "生产环境"
    },
    "skip_hosts": [2, 3],
    "enable_alert": true,
    "fail_threshold": 3,
    "notification_id": 1,
    "latency_alert": true,
    "max_latency": 2.0,
    "statistics": {
      "availability_1h": 100.0,
      "availability_24h": 99.5,
      "availability_7d": 98.8,
      "avg_latency_1h": 0.32,
      "avg_latency_24h": 0.35,
      "avg_latency_7d": 0.38,
      "total_probes_24h": 1440,
      "success_probes_24h": 1433,
      "fail_probes_24h": 7
    },
    "created_at": "2025-10-07T09:00:00Z",
    "updated_at": "2025-10-07T09:00:00Z"
  }
}
```

---

### 1.4 更新服务探测规则

**端点**:`PUT /api/v1/service-monitors/{id}`

**请求**:同创建请求

**响应**(200 OK):返回更新后的规则对象

---

### 1.5 删除服务探测规则

**端点**:`DELETE /api/v1/service-monitors/{id}`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "服务探测规则已删除"
}
```

---

### 1.6 启用/禁用服务探测规则

**端点**:`PATCH /api/v1/service-monitors/{id}/toggle`

**请求**:
```json
{
  "enable": true
}
```

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "服务探测规则已启用"
}
```

---

## 2. 服务探测结果

### 2.1 获取探测历史记录

**端点**:`GET /api/v1/service-monitors/{id}/probe-history`

**查询参数**:
- `start`:开始时间(RFC3339)
- `end`:结束时间(RFC3339)
- `host_id`:主机ID过滤
- `status`:状态过滤(success/fail)
- `limit`:限制数量(默认100,最大1000)

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1001,
      "service_monitor_id": 1,
      "host_id": 1,
      "host_name": "prod-server-01",
      "timestamp": "2025-10-07T10:05:00Z",
      "success": true,
      "latency": 0.352,
      "status_code": 200,
      "error_message": null
    },
    {
      "id": 1002,
      "service_monitor_id": 1,
      "host_id": 2,
      "host_name": "prod-server-02",
      "timestamp": "2025-10-07T10:05:00Z",
      "success": false,
      "latency": 0.0,
      "status_code": 0,
      "error_message": "connection timeout"
    }
  ]
}
```

---

### 2.2 获取可用性统计

**端点**:`GET /api/v1/service-monitors/{id}/availability`

**查询参数**:
- `period`:统计周期(hour/day/week/month)
- `start`:开始时间(RFC3339)
- `end`:结束时间(RFC3339)

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "period": "hour",
      "start_time": "2025-10-07T09:00:00Z",
      "total_probes": 60,
      "success_probes": 60,
      "fail_probes": 0,
      "availability": 100.0,
      "avg_latency": 0.32,
      "min_latency": 0.28,
      "max_latency": 0.45
    },
    {
      "period": "hour",
      "start_time": "2025-10-07T10:00:00Z",
      "total_probes": 5,
      "success_probes": 4,
      "fail_probes": 1,
      "availability": 80.0,
      "avg_latency": 0.35,
      "min_latency": 0.30,
      "max_latency": 0.40
    }
  ]
}
```

---

## 3. 告警规则

### 3.1 创建告警规则

**端点**:`POST /api/v1/alert-rules`

**请求**:
```json
{
  "name": "CPU使用率告警",
  "enable": true,
  "type": "host_monitor",
  "target_ids": [1, 2, 3],
  "condition": "cpu_usage > 90",
  "duration": 300,
  "notification_group_id": 1,
  "alert_level": "warning",
  "silence_period": 600
}
```

**字段说明**:
- `type`:规则类型(host_monitor=主机监控/service_probe=服务探测)
- `condition`:条件表达式(支持cpu_usage/mem_usage/disk_usage/load等指标)
- `duration`:持续时长(秒,条件持续满足该时长才触发)
- `alert_level`:告警级别(info/warning/critical)
- `silence_period`:静默期(秒,避免重复告警)

**响应**(201 Created):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "CPU使用率告警",
    "enable": true,
    "type": "host_monitor",
    "target_ids": [1, 2, 3],
    "condition": "cpu_usage > 90",
    "duration": 300,
    "notification_group_id": 1,
    "alert_level": "warning",
    "silence_period": 600,
    "created_at": "2025-10-07T10:00:00Z",
    "updated_at": "2025-10-07T10:00:00Z"
  }
}
```

---

### 3.2 获取告警规则列表

**端点**:`GET /api/v1/alert-rules`

**查询参数**:
- `page`:页码
- `page_size`:每页数量
- `enable`:启用状态过滤
- `type`:规则类型过滤
- `alert_level`:告警级别过滤

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "name": "CPU使用率告警",
        "enable": true,
        "type": "host_monitor",
        "condition": "cpu_usage > 90",
        "alert_level": "warning",
        "active_events": 2,
        "total_events_24h": 5,
        "last_trigger_time": "2025-10-07T09:30:00Z",
        "created_at": "2025-10-07T09:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 3.3 更新告警规则

**端点**:`PUT /api/v1/alert-rules/{id}`

**请求**:同创建请求

**响应**(200 OK):返回更新后的规则对象

---

### 3.4 删除告警规则

**端点**:`DELETE /api/v1/alert-rules/{id}`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "告警规则已删除"
}
```

---

## 4. 告警事件

### 4.1 获取告警事件列表

**端点**:`GET /api/v1/alert-events`

**查询参数**:
- `page`:页码
- `page_size`:每页数量
- `alert_rule_id`:告警规则ID过滤
- `target_id`:对象ID过滤
- `target_type`:对象类型过滤(host/service)
- `status`:状态过滤(pending/firing/resolved/silenced)
- `level`:级别过滤(info/warning/critical)
- `acknowledged`:是否已确认(true/false)
- `start`:开始时间
- `end`:结束时间

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1,
        "alert_rule_id": 1,
        "alert_rule_name": "CPU使用率告警",
        "target_id": 1,
        "target_type": "host",
        "target_name": "prod-server-01",
        "trigger_time": "2025-10-07T09:30:00Z",
        "recover_time": null,
        "status": "firing",
        "level": "warning",
        "title": "主机CPU使用率过高",
        "message": "主机prod-server-01 CPU使用率92.5%,超过阈值90%",
        "current_value": {
          "cpu_usage": 92.5
        },
        "acknowledged": false,
        "ack_user": null,
        "ack_time": null,
        "ack_note": null
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 4.2 获取告警事件详情

**端点**:`GET /api/v1/alert-events/{id}`

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "alert_rule": {
      "id": 1,
      "name": "CPU使用率告警",
      "condition": "cpu_usage > 90",
      "duration": 300
    },
    "target": {
      "id": 1,
      "type": "host",
      "name": "prod-server-01",
      "uuid": "550e8400-e29b-41d4-a716-446655440000"
    },
    "trigger_time": "2025-10-07T09:30:00Z",
    "recover_time": null,
    "duration_seconds": 300,
    "status": "firing",
    "level": "warning",
    "title": "主机CPU使用率过高",
    "message": "主机prod-server-01 CPU使用率92.5%,超过阈值90%,持续5分钟",
    "current_value": {
      "cpu_usage": 92.5,
      "timestamp": "2025-10-07T09:35:00Z"
    },
    "history": [
      {
        "timestamp": "2025-10-07T09:30:00Z",
        "cpu_usage": 91.2
      },
      {
        "timestamp": "2025-10-07T09:35:00Z",
        "cpu_usage": 92.5
      }
    ],
    "acknowledged": false,
    "ack_user": null,
    "ack_time": null,
    "ack_note": null,
    "notifications": [
      {
        "id": 1,
        "channel": "email",
        "sent_at": "2025-10-07T09:35:10Z",
        "status": "delivered"
      }
    ]
  }
}
```

---

### 4.3 确认告警事件

**端点**:`POST /api/v1/alert-events/{id}/acknowledge`

**请求**:
```json
{
  "note": "已通知运维团队处理"
}
```

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "告警已确认",
  "data": {
    "id": 1,
    "acknowledged": true,
    "ack_user": {
      "id": 1,
      "name": "admin"
    },
    "ack_time": "2025-10-07T09:40:00Z",
    "ack_note": "已通知运维团队处理"
  }
}
```

---

### 4.4 静默告警事件

**端点**:`POST /api/v1/alert-events/{id}/silence`

**请求**:
```json
{
  "duration": 3600
}
```

**响应**(200 OK):
```json
{
  "code": 0,
  "message": "告警已静默",
  "data": {
    "id": 1,
    "status": "silenced",
    "silence_until": "2025-10-07T10:40:00Z"
  }
}
```

---

## 5. 实时WebSocket订阅

### 5.1 订阅主机监控数据

**端点**:`GET /api/v1/ws/host-monitor` (WebSocket升级)

**订阅消息**(浏览器 → Server):
```json
{
  "action": "subscribe",
  "host_ids": [1, 2, 3]
}
```

**取消订阅**:
```json
{
  "action": "unsubscribe",
  "host_ids": [1]
}
```

**推送消息**(Server → 浏览器):
```json
{
  "type": "host_state",
  "host_id": 1,
  "data": {
    "timestamp": "2025-10-07T10:05:00Z",
    "cpu_usage": 35.5,
    "mem_usage": 68.2,
    "disk_usage": 45.8,
    "net_in_speed": 1024000,
    "net_out_speed": 512000
  }
}
```

---

### 5.2 订阅服务探测结果

**端点**:`GET /api/v1/ws/service-probe` (WebSocket升级)

**订阅消息**:
```json
{
  "action": "subscribe",
  "service_ids": [1, 2]
}
```

**推送消息**(Server → 浏览器):
```json
{
  "type": "probe_result",
  "service_id": 1,
  "data": {
    "timestamp": "2025-10-07T10:05:00Z",
    "host_id": 1,
    "success": true,
    "latency": 0.352,
    "status_code": 200
  }
}
```

---

### 5.3 订阅告警事件

**端点**:`GET /api/v1/ws/alert-events` (WebSocket升级)

**订阅消息**:
```json
{
  "action": "subscribe",
  "levels": ["warning", "critical"]
}
```

**推送消息**(Server → 浏览器):

**新告警**:
```json
{
  "type": "alert_fired",
  "event": {
    "id": 1,
    "alert_rule_name": "CPU使用率告警",
    "target_name": "prod-server-01",
    "level": "warning",
    "title": "主机CPU使用率过高",
    "message": "主机prod-server-01 CPU使用率92.5%,超过阈值90%",
    "trigger_time": "2025-10-07T09:30:00Z"
  }
}
```

**告警恢复**:
```json
{
  "type": "alert_resolved",
  "event_id": 1,
  "recover_time": "2025-10-07T09:45:00Z"
}
```

---

## 认证与授权

同主机管理API,所有请求需要JWT认证。

**权限要求**:
- 创建/更新/删除服务探测规则:需要`service_monitors:write`权限
- 查看服务探测数据:需要`service_monitors:read`权限
- 创建/更新/删除告警规则:需要`alert_rules:write`权限
- 查看告警事件:需要`alert_events:read`权限
- 确认/静默告警:需要`alert_events:write`权限
