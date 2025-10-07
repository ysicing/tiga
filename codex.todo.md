# Code Audit TODO

- [ ] internal/api/middleware/router.go:42 `/api/v1/auth/login`/`refresh` are still stub handlers that return 200, so the real auth endpoints remain unreachable at the documented paths.
- [ ] internal/api/handlers/auth_handler.go:372 `GetCurrentUser`仍旧读取 `c.Get("user")`，而 OAuth 中间件只存储 `UserIDKey` 等 typed key，导致接口恒 401。
- [ ] internal/api/routes.go:52 Login 改走 `/api/auth/login/password`，但 `/api/v1/auth/logout`/`refresh` 等真正路由从未注册，调用的还是旧占位 Handler。
- [ ] internal/api/routes.go:56 `NewAuthHandler(..., nil)` 仍然不给 OAuth manager，`/api/auth/providers` 只能返回 `password`，后续 OAuth 登录依旧不可用。
- [x] ~~pkg/common/common.go:47 `LoadEnvs` 依旧没有被调用，运行时不会加载自定义 JWT/数据库/加密配置。~~ **已修复**: LoadEnvs 已移除，改用配置文件，JWT Secret 和加密密钥在安装时自动生成并保存到配置文件
- [x] ~~pkg/common/common.go:33 `ENABLE_ANALYTICS=true` 时却把 `EnableAnalytics` 设为 false，功能开关永远打不开。~~ **已实现**: Analytics 功能已完整实现，配置存储在数据库中，提供 API 让前端动态加载分析脚本，用户可在安装时选择启用并后续通过系统设置开关
- [ ] internal/repository/instance_repo.go:209 继续使用 `? = ANY(tags)` 过滤，`tags` 实际是 JSON/text，SQLite/MySQL 直接报错，Postgres 永远匹配不到。
- [ ] internal/services/managers/manager.go:90 `GetConnectionString` 仍然把端口转换成单个字符，拼出的 `<host>:é` 之类字符串无法用于连接。
- [ ] internal/services/managers/mysql_manager.go:53 `GetConfigValue` 返回 `float64`，这里强转 `.(int)` 一旦配置中写了池参数立即 panic。
- [ ] internal/services/managers/postgres_manager.go:53 同样的 `float64`→`int` panic 问题依旧存在。
- [x] ~~internal/app/app.go:131 认证中间件仍旧使用 `pkgauth.NewOAuthManager()` 里的 `common.JwtSecret`，与新登录服务签发的 `jwtSecret` 不一致，登录取得的 token 无法通过校验。~~ **已修复**: OAuth Manager 和所有认证组件现在都使用配置文件中的 JWT Secret
- [ ] internal/services/auth/oauth.go:250 `parseJSON` 仍是空实现，OAuth 用户信息解析始终返回空对象。
- [ ] internal/services/auth/oauth.go:289 `randomString` 继续用 `time.Now().UnixNano()%len` 生成低熵 state，容易被预测。
- [ ] internal/api/middleware/auth.go:42 将 `claims.UserID` 以字符串写入 context，再按 `uuid.UUID` 读取必然失败，所有需要用户 ID 的逻辑都失效。
- [ ] internal/api/handlers/auth_handler.go:146 `GetProfile` 依赖 `middleware.GetUserID`，因此当前实现永远 401。
- [ ] internal/services/auth/login.go:248/274 更新密码时仍写 `password_hash` 字段，新模型已改成 `password`，修改/重置密码都会报错。
- [ ] pkg/model/compat.go:273 `ResetPasswordByID` 同样更新 `password_hash`，兼容层无法重置新用户密码。
- [ ] internal/install/services/validation_service.go:144 安装流程创建管理员仍使用旧版 `models.User`（uint ID、`IsAdmin` 字段），与最新表结构不匹配，初始化必然失败。
