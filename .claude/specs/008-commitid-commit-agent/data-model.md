# 数据模型：构建版本信息注入与 Agent Docker 上报控制

**功能分支**：`008-commitid-commit-agent`
**日期**：2025-10-26

## 概述

本功能不涉及数据库持久化的数据模型，主要定义以下数据结构：
1. 内存中的版本信息（Go 包变量）
2. gRPC Proto 消息定义
3. API 响应数据结构
4. 配置数据结构

## 实体定义

### 1. VersionInfo（内存结构）

**位置**：`internal/version/version.go`

**用途**：存储编译时注入的版本信息

**字段**：
| 字段名 | 类型 | 必需 | 描述 | 默认值 | 验证规则 |
|--------|------|------|------|--------|----------|
| Version | string | 是 | 版本号（tag+commit或日期+commit） | "dev" | 非空 |
| BuildTime | string | 是 | 构建时间（RFC3339格式） | "unknown" | RFC3339格式 |
| CommitID | string | 是 | Git commit短hash（7位） | "0000000" | 7位十六进制 |

**Go 类型定义**：
```go
package version

// 注意：这些变量通过 -ldflags 在编译时注入
var (
    // Version 版本号，格式：v1.2.3-a1b2c3d 或 20251026-a1b2c3d
    Version = "dev"

    // BuildTime 构建时间，RFC3339 格式
    BuildTime = "unknown"

    // CommitID Git commit 短 hash（7位）
    CommitID = "0000000"
)

// Info 版本信息结构（用于 API 响应）
type Info struct {
    Version   string `json:"version"`
    BuildTime string `json:"build_time"`
    CommitID  string `json:"commit_id"`
}

// GetInfo 返回版本信息
func GetInfo() Info {
    return Info{
        Version:   Version,
        BuildTime: BuildTime,
        CommitID:  CommitID,
    }
}
```

**状态转换**：无（编译时确定，运行时不可变）

**生命周期**：
- 创建：编译时通过 `-ldflags` 注入
- 读取：运行时只读访问
- 销毁：进程退出

### 2. VersionInfo（Proto 消息）

**位置**：`proto/host_monitor.proto`

**用途**：Agent 向服务端上报版本信息

**Proto 定义**：
```protobuf
syntax = "proto3";

package pb;

// VersionInfo Agent版本信息
message VersionInfo {
  // version 版本号（tag+commit或日期+commit）
  string version = 1;

  // build_time 构建时间（RFC3339格式）
  string build_time = 2;

  // commit_id Git commit短hash（7位）
  string commit_id = 3;
}

// HostState 主机状态上报（现有消息，添加新字段）
message HostState {
  // ... 现有字段 ...

  // version_info Agent版本信息（可选，向后兼容）
  optional VersionInfo version_info = 20;
}
```

**字段约束**：
- `version`：非空字符串，最大长度 50
- `build_time`：RFC3339 格式（如 "2025-10-26T10:30:00Z"）
- `commit_id`：7位十六进制字符串

**向后兼容性**：
- 使用 `optional` 修饰符，旧 Agent 不发送此字段时服务端不报错
- 服务端处理时需检查字段是否存在

### 3. AgentConfig（配置结构）

**位置**：`internal/config/config.go`

**用途**：Agent 配置选项

**字段扩展**：
| 字段名 | 类型 | 必需 | 描述 | 默认值 | 来源 |
|--------|------|------|------|--------|------|
| DisableDockerReport | bool | 否 | 禁用Docker实例上报 | false | YAML/ENV |

**Go 类型定义**：
```go
package config

// AgentConfig Agent配置（现有结构，添加新字段）
type AgentConfig struct {
    // ... 现有字段 ...

    // DisableDockerReport 禁用Docker实例上报（默认false，即默认上报）
    DisableDockerReport bool `yaml:"disable_docker_report" env:"AGENT_DISABLE_DOCKER_REPORT"`
}
```

**配置来源优先级**：
1. 环境变量 `AGENT_DISABLE_DOCKER_REPORT`
2. YAML 配置文件 `agent.disable_docker_report`
3. 默认值 `false`

**验证规则**：
- 布尔值，无需额外验证

## API 数据结构

### 版本信息响应

**端点**：`GET /api/v1/version`

**响应结构**：
```json
{
  "version": "v1.2.3-a1b2c3d",
  "build_time": "2025-10-26T10:30:00Z",
  "commit_id": "a1b2c3d"
}
```

**HTTP 状态码**：
- 200 OK：成功返回版本信息

**错误处理**：
- 无错误场景（版本信息始终可用）

## 关系图

```
┌─────────────────────────────────────────────────────────────┐
│                     编译时注入                              │
│  Taskfile + scripts/version.sh                             │
│       ↓                                                     │
│  Go -ldflags -X                                            │
│       ↓                                                     │
│  internal/version.{Version, BuildTime, CommitID}           │
└─────────────────────────────────────────────────────────────┘
                        │
                        │ 读取
                        ↓
        ┌───────────────┴────────────────┐
        │                                │
        ↓                                ↓
┌──────────────────┐            ┌──────────────────┐
│  Server启动      │            │  Agent启动       │
│  cmd/tiga/       │            │  cmd/tiga-agent/ │
└──────────────────┘            └──────────────────┘
        │                                │
        │ 提供                           │ 上报
        ↓                                ↓
┌──────────────────┐            ┌──────────────────┐
│  GET /api/v1/    │            │  gRPC            │
│  version         │            │  VersionInfo     │
└──────────────────┘            └──────────────────┘
        │                                │
        │ 返回                           │ 接收
        ↓                                ↓
┌──────────────────┐            ┌──────────────────┐
│  前端页面显示    │            │  服务端记录      │
└──────────────────┘            └──────────────────┘
```

## 配置流程

```
┌─────────────────┐
│  config.yaml    │
│  agent:         │
│    disable_     │
│    docker_      │
│    report: true │
└─────────────────┘
        │ 或
        ↓
┌─────────────────┐
│  ENV变量        │
│  AGENT_DISABLE_ │
│  DOCKER_REPORT  │
│  =true          │
└─────────────────┘
        │
        ↓ 加载
┌─────────────────┐
│  AgentConfig    │
│  .Disable       │
│  DockerReport   │
└─────────────────┘
        │
        ↓ 检查
┌─────────────────┐
│  if !cfg.Agent  │
│  .DisableDocker │
│  Report {       │
│    report()     │
│  }              │
└─────────────────┘
```

## 数据流

### 版本信息流

```
1. 构建时
   git describe → VERSION
   git rev-parse → COMMIT_ID
   date -u → BUILD_TIME
   ↓
   -ldflags 注入到二进制

2. 运行时（Server）
   二进制启动 → 读取 version.* 变量
   ↓
   启动日志打印版本
   ↓
   /api/v1/version 返回版本
   ↓
   前端页面显示

3. 运行时（Agent）
   二进制启动 → 读取 version.* 变量
   ↓
   启动日志打印版本
   ↓
   gRPC 上报 VersionInfo
   ↓
   服务端记录（日志/存储）
```

### Docker 上报控制流

```
1. 配置加载
   config.yaml / ENV
   ↓
   AgentConfig.DisableDockerReport

2. Agent运行时
   shouldReportDocker() 检查配置
   ↓
   if false: 跳过 Docker 实例上报
   if true: 执行正常上报逻辑
```

## 数据大小估算

### 内存占用

- `Version` 字符串：~20 bytes（如 "v1.2.3-a1b2c3d"）
- `BuildTime` 字符串：24 bytes（如 "2025-10-26T10:30:00Z"）
- `CommitID` 字符串：7 bytes（如 "a1b2c3d"）
- **总计**：~60 bytes（内存中每个进程）

### 网络传输

- gRPC VersionInfo 消息：~80 bytes（包含 Proto 元数据）
- HTTP API 响应：~150 bytes（包含 JSON 格式和 HTTP headers）

### 二进制文件体积

- 注入的字符串常量：~60 bytes
- **总计**：<100 bytes（远低于 1KB 约束）

## 验证规则

### Version 验证

```go
func isValidVersion(v string) bool {
    // 格式1: v1.2.3-a1b2c3d (tag + commit)
    // 格式2: 20251026-a1b2c3d (date + commit)
    // 格式3: dev (默认值)
    // 格式4: snapshot (构建失败时)

    if v == "dev" || v == "snapshot" {
        return true
    }

    // 正则: ^(v?\d+\.\d+\.\d+|\d{8})-[0-9a-f]{7}$
    matched, _ := regexp.MatchString(`^(v?\d+\.\d+\.\d+|\d{8})-[0-9a-f]{7}$`, v)
    return matched
}
```

### BuildTime 验证

```go
func isValidBuildTime(bt string) bool {
    _, err := time.Parse(time.RFC3339, bt)
    return err == nil || bt == "unknown"
}
```

### CommitID 验证

```go
func isValidCommitID(cid string) bool {
    // 7位十六进制或默认值
    if cid == "0000000" {
        return true
    }
    matched, _ := regexp.MatchString(`^[0-9a-f]{7}$`, cid)
    return matched
}
```

## 索引设计

**不适用**：本功能不涉及数据库表和索引

## 迁移策略

**不适用**：本功能不涉及数据库迁移

## 总结

本功能的数据模型极其简单：
- ✅ **无数据库依赖**：版本信息存储在二进制文件和内存中
- ✅ **向后兼容**：Proto 使用 `optional` 字段，配置使用默认值
- ✅ **最小体积**：总计 <100 bytes，符合性能约束
- ✅ **易于测试**：所有结构体都是纯数据，无副作用
