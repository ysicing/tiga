# Tiga 项目代码审计 - 快速行动清单

**基于 AUDIT_REPORT.md 的立即行动指南**

---

## 🚨 必须立即修复（阻塞生产发布）

### 1. 加密密钥硬编码 [0.5 天]
```bash
# 文件: pkg/common/crypto.go
# 当前: const defaultEncryptionKey = "tiga-secret-key-32-bytes-long!"

# 修复步骤:
1. 生成新密钥: openssl rand -hex 32
2. 添加到 config.yaml:
   security:
     encryption_key_env: "TIGA_ENCRYPTION_KEY"
3. 更新代码读取环境变量
4. 重新加密所有现有数据

# 验证:
gosec -include=G101 ./pkg/common/crypto.go
```

### 2. JWT 密钥硬编码 [1 天]
```bash
# 文件: pkg/common/jwt.go
# 当前: const defaultJWTSecret = "tiga-jwt-secret-key"

# 修复步骤:
1. 生成 RSA 密钥对:
   openssl genrsa -out jwt-private.pem 2048
   openssl rsa -in jwt-private.pem -pubout -out jwt-public.pem
2. 迁移到 RS256 算法
3. 实施 token 黑名单机制

# 验证:
curl -X POST http://localhost:12306/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}'
```

### 3. 数据库连接泄漏 [0.5 天]
```bash
# 文件: internal/services/database/manager.go:156-180

# 修复步骤:
1. 搜索所有 GetConnection 调用
2. 添加 defer Close() 或错误路径清理
3. 添加连接池监控

# 验证:
# 压力测试检查连接数
ab -n 10000 -c 100 http://localhost:12306/api/v1/database/instances
# 监控 max_connections
```

### 4. 并发安全问题 [1 天]
```bash
# 文件: internal/services/database/manager.go:30-50

# 修复步骤:
1. 添加 sync.RWMutex 保护 map
2. 实施双重检查锁定模式
3. 运行并发测试

# 验证:
go test -race ./internal/services/database/...
```

**Phase 1 总工时**: 3 天
**完成标准**: gosec 无 Critical 问题，并发测试通过

---

## ⚡ 高优先级修复（1-2 周内完成）

### 5. SQL 安全过滤增强 [2 天]
```bash
# 文件: internal/services/database/security_filter.go

# 检查当前问题:
grep -n "dangerousKeywords" internal/services/database/security_filter.go

# 修复要点:
- 添加 REPLACE, HANDLER, PREPARE, EXECUTE 到黑名单
- 使用 sqlparser 库而非字符串匹配
- 实施白名单策略
- 严格验证 WHERE 子句（拒绝 WHERE 1=1）

# 测试用例:
go test -v ./tests/unit/security_filter_test.go
```

### 6. Redis 命令过滤增强 [1 天]
```bash
# 文件: internal/services/database/security_filter.go

# 添加黑名单:
EVAL, EVALSHA, SCRIPT, MODULE, MIGRATE, CLIENT, DEBUG,
SLAVEOF, REPLICAOF, RESTORE

# 实施 Redis ACL (Redis 6.0+):
ACL SETUSER readonly on >password ~* +@read -@write -@dangerous

# 测试:
redis-cli --eval malicious.lua
```

### 7. bcrypt 密码哈希 [1.5 天]
```bash
# 文件: internal/models/user.go, internal/services/auth/

# 检查当前实现:
grep -r "password\|Password" internal/models/user.go

# 实施 bcrypt:
import "golang.org/x/crypto/bcrypt"

func (u *User) SetPassword(password string) error {
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)
    u.PasswordHash = string(hash)
}

# 迁移现有用户:
# 1. 添加 password_hash 字段
# 2. 保留旧 password 字段
# 3. 用户登录时迁移
# 4. 完成后删除 password 字段

# 验证:
go test -v ./internal/services/auth/*_test.go
```

### 8. Context 传播 [3 天]
```bash
# 当前问题: 89 处使用 context.Background()

# 批量查找:
grep -rn "context.Background()" internal/services/

# 修复模式:
# Before:
func (s *Service) DoWork() error {
    ctx := context.Background()
}

# After:
func (s *Service) DoWork(ctx context.Context) error {
    // 使用传入的 context
}

# 自动化重构脚本:
./scripts/refactor-context.sh
```

### 9. 数据库连接生命周期 [0.5 天]
```bash
# 文件: pkg/dbdriver/mysql.go, postgres.go, redis.go

# 一键修复:
find pkg/dbdriver -name "*.go" -exec sed -i '' \
  '/SetMaxIdleConns/a\
db.SetConnMaxLifetime(5 * time.Minute)\
db.SetConnMaxIdleTime(90 * time.Second)' {} \;

# 验证配置:
grep -A2 "SetMaxIdleConns" pkg/dbdriver/*.go
```

### 10. 审计日志异步化 [1 天]
```bash
# 文件: internal/services/database/audit_logger.go

# 实施异步队列:
type AuditLogger struct {
    logChan chan *models.DatabaseAuditLog
}

func (a *AuditLogger) LogQuery(log *models.DatabaseAuditLog) error {
    select {
    case a.logChan <- log:
        return nil
    default:
        return errors.New("queue full")
    }
}

# 性能测试:
go test -bench=BenchmarkAuditLogger ./internal/services/database/
```

**Phase 2 总工时**: 10 天

---

## 📊 快速健康检查命令

### 运行所有检查
```bash
#!/bin/bash
# scripts/health-check.sh

echo "=== 安全扫描 ==="
gosec -fmt=json -out=gosec-report.json ./...
govulncheck ./...

echo "=== 代码质量 ==="
golangci-lint run --out-format=json > lint-report.json

echo "=== 测试覆盖率 ==="
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

echo "=== 依赖漏洞 ==="
go list -json -m all | nancy sleuth

echo "=== 性能基准 ==="
go test -bench=. -benchmem ./internal/services/database/ > bench.txt

echo "=== 报告生成完成 ==="
echo "gosec: gosec-report.json"
echo "lint: lint-report.json"
echo "coverage: coverage.html"
echo "benchmark: bench.txt"
```

### CI/CD 门禁配置
```yaml
# .github/workflows/quality-gate.yml
name: Quality Gate

on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run gosec
        run: |
          go install github.com/securego/gosec/v2/cmd/gosec@latest
          gosec -fmt=sarif -out=results.sarif ./...
          # 失败条件: 有 Critical 或 High 问题
          gosec -severity=high -confidence=high ./...

      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
          args: --timeout=5m

      - name: Check test coverage
        run: |
          go test -coverprofile=coverage.out ./...
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 60" | bc -l) )); then
            echo "Coverage $COVERAGE% is below 60%"
            exit 1
          fi
```

---

## 🎯 每日检查清单

### 开发者自检
```bash
# 提交前运行
pre-commit() {
    task gofmt        # 格式化
    task lint         # 静态分析
    task test         # 单元测试
    gosec ./...       # 安全扫描
    go test -race ./... # 竞态检测
}
```

### 代码审查清单
- [ ] 所有数据库连接有 defer Close()
- [ ] 服务方法使用 context.Context 参数
- [ ] 错误正确包装（fmt.Errorf("%w", err)）
- [ ] 敏感数据不在日志中
- [ ] SQL 查询使用参数化
- [ ] 并发访问有锁保护
- [ ] 长时间操作有超时控制
- [ ] 新代码有单元测试

---

## 📞 紧急联系

**遇到问题？**
1. 查看完整报告: `AUDIT_REPORT.md`
2. 运行健康检查: `./scripts/health-check.sh`
3. 查看日志: `tail -f logs/tiga.log`

**需要帮助？**
- 安全问题: security@yourdomain.com
- 技术支持: tech-support@yourdomain.com
- 文档: https://github.com/ysicing/tiga/wiki

---

**创建时间**: 2025-10-16
**最后更新**: 2025-10-16
**下次检查**: 2025-11-16（每月检查）
