# 研究文档：构建版本信息注入与 Agent Docker 上报控制

**功能分支**：`008-commitid-commit-agent`
**日期**：2025-10-26

## 研究目标

本文档记录实施构建版本信息注入和 Agent Docker 上报控制功能所需的技术决策和最佳实践研究。

## 研究任务

### 1. Go 编译时版本注入机制

**决策**：使用 Go `-ldflags -X` 注入版本变量

**理由**：
- Go 标准做法，无需额外工具
- 编译时注入，零运行时开销
- 支持多个变量注入（Version、BuildTime、CommitID）
- 广泛应用于 Kubernetes、Docker 等开源项目

**实现模式**：
```go
// internal/version/version.go
package version

var (
    Version   = "dev"      // 构建时注入
    BuildTime = "unknown"  // 构建时注入
    CommitID  = "0000000"  // 构建时注入
)

func GetVersion() string {
    return Version
}

func GetBuildInfo() map[string]string {
    return map[string]string{
        "version":    Version,
        "build_time": BuildTime,
        "commit_id":  CommitID,
    }
}
```

**构建命令示例**：
```bash
go build -ldflags "\
  -X github.com/ysicing/tiga/internal/version.Version=${VERSION} \
  -X github.com/ysicing/tiga/internal/version.BuildTime=${BUILD_TIME} \
  -X github.com/ysicing/tiga/internal/version.CommitID=${COMMIT_ID}"
```

**考虑的替代方案**：
- ❌ 代码生成（go generate）：增加构建复杂度
- ❌ 配置文件：需要维护额外文件，易出错
- ❌ 嵌入文件（go:embed）：增加二进制体积

### 2. Taskfile 版本信息提取

**决策**：创建 `scripts/version.sh` 脚本，由 Taskfile 调用

**理由**：
- 分离关注点：Taskfile 负责流程，脚本负责逻辑
- 便于测试和独立调试
- 支持复杂的 git 命令组合
- 可复用于 Docker 构建

**版本提取逻辑**：
```bash
#!/bin/bash
# scripts/version.sh

# 1. 尝试获取最近的 git tag
TAG=$(git describe --tags --abbrev=0 2>/dev/null)

# 2. 获取 commit 短 hash
COMMIT=$(git rev-parse --short=7 HEAD 2>/dev/null || echo "0000000")

# 3. 获取构建时间（RFC3339 格式）
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# 4. 生成版本号
if [ -n "$TAG" ]; then
    VERSION="${TAG}-${COMMIT}"
else
    # 无 tag，使用日期
    DATE=$(date -u +"%Y%m%d")
    VERSION="${DATE}-${COMMIT}"
fi

# 5. 输出环境变量格式（供 Taskfile 使用）
echo "VERSION=${VERSION}"
echo "BUILD_TIME=${BUILD_TIME}"
echo "COMMIT_ID=${COMMIT}"
```

**Taskfile 集成**：
```yaml
tasks:
  build:
    vars:
      VERSION_INFO:
        sh: bash scripts/version.sh
    cmds:
      - |
        eval "$(bash scripts/version.sh)"
        go build -ldflags "\
          -X github.com/ysicing/tiga/internal/version.Version=${VERSION} \
          -X github.com/ysicing/tiga/internal/version.BuildTime=${BUILD_TIME} \
          -X github.com/ysicing/tiga/internal/version.CommitID=${COMMIT_ID}" \
          -o bin/tiga cmd/tiga/main.go
```

**考虑的替代方案**：
- ❌ 直接在 Taskfile 中写 git 命令：可读性差，难以维护
- ❌ 使用 Makefile：项目已使用 Taskfile，保持一致性
- ❌ 使用 goreleaser：过于重量级，不适合本功能

### 3. gRPC Proto 消息扩展

**决策**：在 `HostState` 消息中添加 `version_info` 字段

**理由**：
- `HostState` 是 Agent 定期上报的主要消息
- 使用嵌套消息 `VersionInfo`，便于未来扩展
- 向后兼容：新字段使用 `optional`，旧 Agent 不受影响

**Proto 定义**：
```protobuf
// proto/host_monitor.proto

message VersionInfo {
  string version = 1;      // v1.2.3-a1b2c3d 或 20251026-a1b2c3d
  string build_time = 2;   // RFC3339 格式
  string commit_id = 3;    // 7位短 hash
}

message HostState {
  // 现有字段...

  // 新增版本信息字段
  optional VersionInfo version_info = 20;  // 使用较大的字段号避免冲突
}
```

**Agent 端实现**：
```go
// cmd/tiga-agent/main.go
import "github.com/ysicing/tiga/internal/version"

func reportState() *pb.HostState {
    return &pb.HostState{
        // 现有字段...

        VersionInfo: &pb.VersionInfo{
            Version:   version.Version,
            BuildTime: version.BuildTime,
            CommitId:  version.CommitID,
        },
    }
}
```

**考虑的替代方案**：
- ❌ 创建新的 `ReportVersion` RPC：增加 RPC 方法数量
- ❌ 在 `HostInfo` 中添加：HostInfo 只在首次连接时发送
- ❌ 使用 metadata：不适合持久化存储

### 4. Agent 配置系统扩展

**决策**：在 `internal/config/config.go` 中添加 `Agent.DisableDockerReport` 字段

**理由**：
- 遵循现有配置结构（YAML + 环境变量）
- 使用负向命名（`DisableDockerReport`）明确默认行为（启用）
- 支持 YAML 配置和环境变量覆盖

**配置定义**：
```go
// internal/config/config.go

type AgentConfig struct {
    // 现有字段...

    // DisableDockerReport 禁用 Docker 实例上报（默认 false，即默认上报）
    DisableDockerReport bool `yaml:"disable_docker_report" env:"AGENT_DISABLE_DOCKER_REPORT"`
}
```

**YAML 配置示例**：
```yaml
# config.yaml
agent:
  disable_docker_report: false  # 默认值，可省略
```

**环境变量示例**：
```bash
export AGENT_DISABLE_DOCKER_REPORT=true
```

**使用方式**：
```go
// cmd/tiga-agent/docker_handler.go

func shouldReportDocker() bool {
    return !cfg.Agent.DisableDockerReport
}

func reportDockerInstances() {
    if !shouldReportDocker() {
        log.Info("Docker instance reporting disabled")
        return
    }

    // 执行上报逻辑...
}
```

**考虑的替代方案**：
- ❌ 使用正向命名 `EnableDockerReport`：需要默认值 true，不直观
- ❌ 使用命令行参数：配置文件更适合持久化配置
- ❌ 使用特性开关服务：过度设计

### 5. 服务端版本 API 设计

**决策**：创建 GET `/api/v1/version` 端点返回版本信息

**理由**：
- RESTful 风格，遵循现有 API 约定
- 无需认证，方便监控系统调用
- 返回 JSON 格式，便于前端解析

**API 契约**：
```http
GET /api/v1/version HTTP/1.1
Host: localhost:12306

HTTP/1.1 200 OK
Content-Type: application/json

{
  "version": "v1.2.3-a1b2c3d",
  "build_time": "2025-10-26T10:30:00Z",
  "commit_id": "a1b2c3d"
}
```

**Swagger 注解**：
```go
// @Summary      获取服务端版本信息
// @Description  返回服务端的版本号、构建时间和 commit ID
// @Tags         system
// @Accept       json
// @Produce      json
// @Success      200  {object}  VersionResponse
// @Router       /api/v1/version [get]
func GetVersion(c *gin.Context) {
    c.JSON(200, version.GetBuildInfo())
}
```

**考虑的替代方案**：
- ❌ 使用 `/health` 端点返回版本：语义不清晰
- ❌ 使用 WebSocket：过度设计
- ❌ 集成到 `/api/v1/system/info`：增加响应体积

### 6. 命令行版本显示

**决策**：支持 `--version` 和 `version` 子命令

**理由**：
- 遵循 Unix 工具约定（如 `git --version`）
- 启动日志自动显示版本信息
- 便于运维人员快速确认版本

**实现方式**：
```go
// cmd/tiga/main.go

func main() {
    // 处理 --version 标志
    if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "version") {
        fmt.Printf("Tiga Server\n")
        fmt.Printf("Version:    %s\n", version.Version)
        fmt.Printf("Build Time: %s\n", version.BuildTime)
        fmt.Printf("Commit ID:  %s\n", version.CommitID)
        os.Exit(0)
    }

    // 启动日志显示版本
    log.WithFields(log.Fields{
        "version":    version.Version,
        "build_time": version.BuildTime,
        "commit_id":  version.CommitID,
    }).Info("Starting Tiga Server")

    // 正常启动流程...
}
```

**输出示例**：
```
$ ./bin/tiga --version
Tiga Server
Version:    v1.2.3-a1b2c3d
Build Time: 2025-10-26T10:30:00Z
Commit ID:  a1b2c3d

$ ./bin/tiga
INFO[0000] Starting Tiga Server build_time=2025-10-26T10:30:00Z commit_id=a1b2c3d version=v1.2.3-a1b2c3d
```

## 性能考虑

### 版本信息提取性能

**测试场景**：10 次连续构建测试

**测量方法**：
```bash
time bash scripts/version.sh
```

**预期结果**：
- git describe: <50ms
- git rev-parse: <30ms
- date: <10ms
- 总计: <100ms（<5秒目标的 2%）

**优化策略**：
- 使用 git 命令缓存（Taskfile 的 `vars`）
- 避免重复执行脚本

### 版本 API 性能

**性能目标**：<10ms p99 延迟

**实现特点**：
- 内存中读取变量，无 I/O 操作
- 无数据库查询
- JSON 序列化开销小（<50 bytes）

**预期性能**：
- 延迟：<1ms p99
- 吞吐量：>10k req/s（单核）

### 二进制文件体积影响

**版本信息大小**：
- Version: ~20 bytes
- BuildTime: 24 bytes
- CommitID: 7 bytes
- 总计: <100 bytes（远低于 1KB 约束）

## 风险与缓解

### 风险 1：无 git 环境构建失败

**缓解措施**：
- 脚本使用 `|| echo "default"` 提供默认值
- Version="dev"、CommitID="0000000"
- 构建不会因缺少 git 而失败

### 风险 2：Agent 版本字段不兼容

**缓解措施**：
- Proto 字段使用 `optional`
- 服务端兼容缺少 version_info 的旧 Agent
- 数据库字段允许 NULL

### 风险 3：时区差异导致 BuildTime 不一致

**缓解措施**：
- 统一使用 UTC 时区（`date -u`）
- RFC3339 格式包含时区信息
- 显示时转换为用户本地时区

## 总结

所有技术选择均基于以下原则：
- ✅ **简单性**：使用 Go 和 git 标准工具，无额外依赖
- ✅ **兼容性**：向后兼容，支持旧版本 Agent
- ✅ **性能**：零运行时开销，构建时间增加 <5 秒
- ✅ **可测试性**：所有组件可独立测试

未发现需要进一步澄清的技术问题，可以进入阶段 1 设计。
