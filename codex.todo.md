# Code Audit TODO

- [ ] internal/api/routes.go:103 `NewAuthHandler(..., nil)` 仍然不给 OAuth manager，`/api/auth/providers` 只能返回 `password`，后续 OAuth 登录依旧不可用。
- [ ] internal/repository/instance_repo.go:151 `? = ANY(tags)` 过滤依旧假设数组字段，SQLite/MySQL 会报错，Postgres 永远匹配不到。
- [ ] internal/services/managers/manager.go:101 `GetConnectionString` 仍把端口转换成单字符，拼出的 `<host>:\x05` 之类字符串无法用于连接。
- [ ] internal/services/managers/mysql_manager.go:52 `GetConfigValue` 返回 `float64`，这里强转 `.(int)` 一旦配置中写了池参数立即 panic。
- [ ] internal/services/managers/postgres_manager.go:55 同样的 `float64`→`int` panic 问题依旧存在。
- [ ] internal/services/auth/oauth.go:266 `parseJSON` 仍是空实现，OAuth 用户信息解析始终返回空对象。
- [ ] internal/services/auth/oauth.go:277 `randomString` 继续用 `time.Now().UnixNano()%len` 生成低熵 state，容易被预测。
- [ ] internal/api/handlers/host_group_handler.go:71 依旧用 `ParseUint`/`[]uint` 处理 ID，切到 UUID 后删除分组或批量加主机会直接失败。
- [ ] internal/api/handlers/host_handler.go:140 以及 internal/repository/host_repository.go:16/95 `group_id` 仍按整数处理并拼 `string(rune(...))`，UUID 时代分组筛选彻底失效。
- [ ] internal/api/handlers/service_monitor_handler.go:60/79/99/116/128 忽略 `uuid.Parse` 错误，非法 ID 会默默落成 `uuid.Nil` 并作用在错误的记录上。
- [ ] internal/api/handlers/service_monitor_handler.go:34 起缺少列表接口且 `host_id` 字段与前端 `host_node_id` 不匹配，更新接口也只保存 `interval/enabled`，导致服务监控页面无法正常增改查。
- [ ] internal/api/routes.go:333 未注册 `GET /api/v1/vms/service-monitors` 路由，前端初始化列表直接 404。
- [ ] internal/repository/service_repository.go:13 `ServiceFilter.HostID` 仍是 `uint` 并与 `group_ids` 一样用大于零判断，UUID 下 host 过滤统统失效。
- [ ] internal/api/handlers/webssh_handler.go:74 为了兼容 UUID 暂时写死 `userID := uuid.MustParse("0000...01")`，还在 `CreateSession` 失败时继续访问 `wsSession.SessionID`，真实用户 ID 永远丢失且容易 panic。
- [ ] ui/src/pages/hosts/service-monitor-page.tsx:60 / alert-events-page.tsx:63 等仍然用 `Bearer ${localStorage.getItem('token')}` 且缺少 `credentials: 'include'`，新登录改用 Cookie 后所有增删改请求都会 401。
- [ ] ui/src/pages/hosts/host-ssh-page.tsx:45 依旧连向 `/api/v1/vms/hosts/:id/ssh/connect` 并发 Token Header，新后端改成 WebSSH 会话，这个页面现在始终 404/401。
- [ ] internal/models/host_node.go:7 同步把 host/agent/monitor 表主键改成 UUID，但没有任何迁移脚本，旧库仍是 int，自建环境会直接列类型不匹配。
