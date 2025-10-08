# Code Audit TODO

## ✅ 已修复 (Fixed)

- [x] ~~internal/services/managers/manager.go:104 `GetConnectionString` 端口转换问题~~ - 已修复：使用 `fmt.Sprintf("%s:%d", host, int(port))`
- [x] ~~internal/services/managers/mysql_manager.go:54-56 `float64`→`int` panic 问题~~ - 已修复：添加了安全的类型断言
- [x] ~~internal/services/managers/postgres_manager.go:55-57 `float64`→`int` panic 问题~~ - 已修复：添加了安全的类型断言
- [x] ~~internal/services/auth/oauth.go:266 `parseJSON` 空实现~~ - 已修复：使用 `json.NewDecoder(r).Decode(v)`
- [x] ~~internal/services/auth/oauth.go:277 `randomString` 低熵 state 生成~~ - 已修复：使用 `crypto/rand` 生成安全随机数
- [x] ~~internal/api/handlers/service_monitor_handler.go:60/74/105/117/130 忽略 `uuid.Parse` 错误~~ - 已修复：添加了错误检查并返回 400
- [x] ~~internal/api/handlers/service_monitor_handler.go 缺少列表接口~~ - 已修复：添加了 `ListMonitors` 方法
- [x] ~~internal/api/routes.go:357 未注册 `GET /api/v1/vms/service-monitors` 路由~~ - 已修复：添加了列表路由
- [x] ~~internal/repository/service_repository.go:16 `HostID` 仍是 `uint`~~ - 已修复：改为 `*uuid.UUID` 并更新过滤逻辑
- [x] ~~ui/src/pages/hosts/alert-events-page.tsx:61/80/100 使用 `Bearer localStorage token`~~ - 已修复：改用 `credentials: 'include'`
- [x] ~~ui/src/pages/hosts/alert-events-page.tsx:61 请求 `/vms/alert-events`~~ - 已修复：改为 `/alerts/events`
- [x] ~~ui/src/lib/api-client.ts:388-393 硬编码 `/vms/alert-events`~~ - 已修复：改为 `/alerts/events`
- [x] ~~internal/api/handlers/webssh_handler.go:79 写死 `userID`~~ - 已修复：通过 `middleware.GetUserID` 注入真实用户并在 WebSocket 生命周期中同步更新/清理会话
- [x] ~~ui/src/pages/hosts/host-ssh-page.tsx 调用旧的 `/api/v1/vms/hosts/:id/ssh/connect` 并使用 Bearer Token~~ - 已修复：改为调用 `devopsAPI.vms.webssh.createSession` + 新版 WebSocket 协议并统一使用 Cookie 鉴权
- [x] ~~internal/api/handlers/service_monitor_handler.go:30 期待 `host_id`，前端发送 `host_node_id` 不匹配~~ - 已修复：改为 `host_node_id`
- [x] ~~internal/api/handlers/service_monitor_handler.go:84-87 更新接口仅保存 `interval/enabled`~~ - 已修复：支持更新所有字段（name, type, target, interval, timeout, host_node_id, enabled, notify_on_failure）
- [x] ~~ui/src/pages/hosts/service-monitor-page.tsx:75/103/126 使用手写 fetch 且缺少 `credentials: 'include'`~~ - 已修复：所有请求添加 `credentials: 'include'`
- [x] ~~ui/src/pages/hosts/service-monitor-page.tsx 缺少节点选择UI~~ - 已修复：添加了主机节点下拉选择器
- [x] ~~ui/src/pages/hosts/service-monitor-page.tsx 缺少监控数据展示~~ - 已修复：添加了探测结果、可用性和最后探测时间展示
- [x] ~~internal/services/monitor/probe_scheduler.go:74 仍以 `%d` 打印 UUID~~ - 已修复：日志改用 `monitor.ID.String()` 输出，避免 `%!d(uuid.UUID=...)`
- [x] ~~ui/src/components/hosts/webssh-terminal.tsx 与新版 WebSSH 协议不匹配~~ - 已修复：前端改为发送/接收 JSON 消息并对终端数据做 Base64 编解码
- [x] ~~internal/repository/instance_repo.go:151 `? = ANY(tags)` 过滤仅适用于 PostgreSQL~~ - 已修复：根据数据库方言分别使用 JSON 语法或回退 `LIKE`，实现跨数据库标签过滤
- [x] ~~ui/src/lib/api-client.ts:374~382 `devopsAPI.vms.alertRules.*` 仍指向 `/vms/alert-rules`~~ - 已修复：统一改为 `/alerts/rules` 与后端新路由对齐

## ⚠️ 需要进一步处理 (Needs Further Action)

- [ ] internal/api/routes.go:112 `NewAuthHandler(..., nil)` 仍然不给 OAuth manager - **需要实现完整的 OAuthManager 并注入**
- [ ] internal/api/handlers/websocket_handler.go:25 WebSocket 升级全量放行 `CheckOrigin` - **需要限制允许的来源或通过配置控制**
- [ ] internal/api/handlers/webssh_handler.go:27 WebSSH 升级全量放行 `CheckOrigin` - **需要校验来源或携带 CSRF 保护**
- [ ] internal/api/handlers/websocket_handler.go:149 `ServiceProbe` 端点仍返回占位 JSON，未真正升级为 WebSocket 推送实时探测数据
- [ ] internal/api/handlers/websocket_handler.go:201 `AlertEvents` 同样返回占位 JSON，缺少面向前端的实时推送实现
- [ ] internal/api/handlers/auth_handler.go:240 `OAuthLogin` 仅返回占位信息，未持久化用户也未签发 JWT/Session - **需要完成 OAuth 登录流程闭环**

## 📝 修复说明

### 端口转换修复
**文件**: `internal/services/managers/manager.go`
**问题**: `string(rune(int(port)))` 将端口号转为单字符
**修复**: 使用 `fmt.Sprintf("%s:%d", host, int(port))`

### 类型断言修复
**文件**: `mysql_manager.go`, `postgres_manager.go`
**问题**: 直接 `.(int)` 转换会 panic
**修复**: 先尝试 `.(float64)` 再转 `int`，再尝试 `.(int)`，都失败则使用默认值

### OAuth 安全修复
**文件**: `internal/services/auth/oauth.go`
**问题**:
1. `parseJSON` 空实现导致用户信息解析失败
2. `randomString` 使用时间戳取模生成低熵随机数
**修复**:
1. 使用 `json.NewDecoder(r).Decode(v)` 正确解析 JSON
2. 使用 `crypto/rand` 生成 32 字节安全随机数

### UUID 解析错误处理
**文件**: `service_monitor_handler.go`
**问题**: 忽略 `uuid.Parse` 错误，非法 ID 会默默变成 `uuid.Nil`
**修复**: 检查错误并返回 HTTP 400 Bad Request

### API 路径修复
**问题**: Alert events 使用错误的路径 `/vms/alert-events`
**修复**: 统一改为 `/alerts/events` 匹配后端路由

### 认证方式修复
**问题**: 前端使用 `Bearer localStorage.getItem('token')`
**修复**: 改为 `credentials: 'include'` 使用 HTTP-only cookie

### 服务监控修复（新增）
**文件**: `internal/api/handlers/service_monitor_handler.go`, `ui/src/pages/hosts/service-monitor-page.tsx`
**问题**:
1. 前后端字段不一致：后端期待 `host_id`，前端发送 `host_node_id`
2. 更新接口仅保存 `interval` 和 `enabled`，忽略其他字段
3. 前端缺少节点选择UI
4. 前端缺少监控数据展示（探测结果、可用性统计）
5. 前端使用 Bearer token 而非 Cookie 认证

**修复**:
1. 后端 CreateMonitor 改为接收 `host_node_id` 字段
2. 后端 UpdateMonitor 支持更新所有字段：name, type, target, interval, timeout, host_node_id, enabled, notify_on_failure（使用指针类型支持部分更新）
3. 前端添加主机节点下拉选择器，支持选择探测节点或留空从服务端探测
4. 前端获取每个监控的可用性统计，展示探测结果、可用性百分比、最后探测时间
5. 所有请求添加 `credentials: 'include'` 使用 Cookie 认证

## 📋 待办事项优先级

**High Priority**:
1. 实现 OAuthManager 并注入到 AuthHandler

**Medium Priority**:
1. 前端统一改用 `devopsAPI` + Cookie 鉴权（告警规则等）

**Low Priority**:
1. 调整日志与接口细节（如调度器 UUID 打印）并补充测试覆盖率
