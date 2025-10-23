# gRPC协议契约：Docker实例远程管理

**功能分支**：`007-docker-docker-agent`
**创建日期**：2025-10-22
**协议版本**：v1.0.0
**状态**：草稿

---

## 概述

Docker实例远程管理功能通过扩展现有Agent gRPC协议实现，新增 `DockerService` 服务。

**架构设计**：
- Agent作为gRPC Server，Server（Tiga后端）作为gRPC Client
- 复用现有Agent连接（002-nezha-webssh分支），无需新建连接
- 所有Docker操作通过Agent转发到目标主机执行

**Protobuf文件位置**：`pkg/grpc/proto/docker/docker.proto`

---

## 服务定义

```protobuf
syntax = "proto3";

package docker;

option go_package = "github.com/ysicing/tiga/pkg/grpc/proto/docker";

// DockerService Docker管理服务
service DockerService {
  // ==================== Docker实例信息 ====================

  // GetDockerInfo 获取Docker守护进程信息
  rpc GetDockerInfo(GetDockerInfoRequest) returns (DockerInfo);

  // ==================== 容器操作 ====================

  // ListContainers 获取容器列表（支持分页和过滤）
  rpc ListContainers(ListContainersRequest) returns (ListContainersResponse);

  // GetContainer 获取容器详情
  rpc GetContainer(GetContainerRequest) returns (Container);

  // StartContainer 启动容器
  rpc StartContainer(ContainerActionRequest) returns (ContainerActionResponse);

  // StopContainer 停止容器
  rpc StopContainer(ContainerActionRequest) returns (ContainerActionResponse);

  // RestartContainer 重启容器
  rpc RestartContainer(ContainerActionRequest) returns (ContainerActionResponse);

  // PauseContainer 暂停容器
  rpc PauseContainer(ContainerActionRequest) returns (ContainerActionResponse);

  // UnpauseContainer 恢复容器
  rpc UnpauseContainer(ContainerActionRequest) returns (ContainerActionResponse);

  // DeleteContainer 删除容器
  rpc DeleteContainer(DeleteContainerRequest) returns (ContainerActionResponse);

  // GetContainerStats 获取容器资源统计（流式，实时推送）
  rpc GetContainerStats(GetContainerStatsRequest) returns (stream ContainerStats);

  // GetContainerLogs 获取容器日志（流式）
  rpc GetContainerLogs(GetContainerLogsRequest) returns (stream LogEntry);

  // ExecContainer 在容器中执行命令（双向流，用于终端）
  // 注意：实际终端通过WebSocket实现，此RPC仅作为Agent端执行docker exec的内部接口
  rpc ExecContainer(stream ExecRequest) returns (stream ExecResponse);

  // ==================== 镜像操作 ====================

  // ListImages 获取镜像列表
  rpc ListImages(ListImagesRequest) returns (ListImagesResponse);

  // GetImage 获取镜像详情
  rpc GetImage(GetImageRequest) returns (Image);

  // DeleteImage 删除镜像
  rpc DeleteImage(DeleteImageRequest) returns (DeleteImageResponse);

  // PullImage 拉取镜像（流式，返回拉取进度）
  rpc PullImage(PullImageRequest) returns (stream PullImageProgress);

  // TagImage 给镜像打标签
  rpc TagImage(TagImageRequest) returns (TagImageResponse);
}
```

---

## 消息定义

### Docker实例信息

```protobuf
// GetDockerInfoRequest 获取Docker信息请求
message GetDockerInfoRequest {
  // 无参数，返回当前Agent主机上的Docker信息
}

// DockerInfo Docker守护进程信息
message DockerInfo {
  string version = 1;            // Docker版本，如 "24.0.7"
  string api_version = 2;        // API版本，如 "1.43"
  string storage_driver = 3;     // 存储驱动，如 "overlay2"
  int32 containers = 4;          // 容器总数
  int32 containers_running = 5;  // 运行中容器数
  int32 containers_paused = 6;   // 暂停的容器数
  int32 containers_stopped = 7;  // 停止的容器数
  int32 images = 8;              // 镜像总数
  string operating_system = 9;   // 操作系统，如 "Ubuntu 22.04"
  string architecture = 10;      // 架构，如 "x86_64"
  string kernel_version = 11;    // 内核版本
  int64 mem_total = 12;          // 总内存（字节）
  int32 n_cpu = 13;              // CPU核心数
  string server_version = 14;    // Docker Engine版本
}
```

### 容器操作

```protobuf
// ListContainersRequest 容器列表请求
message ListContainersRequest {
  bool all = 1;           // true=所有容器，false=仅运行中
  int32 page = 2;         // 页码（从1开始，0=不分页）
  int32 page_size = 3;    // 每页条数（默认50，最大1000）
  string filter = 4;      // Docker原生filter，如 "name=nginx" "status=running"
  string sort_by = 5;     // 排序字段：created, name, status
  string sort_order = 6;  // 排序方式：asc, desc
}

// ListContainersResponse 容器列表响应
message ListContainersResponse {
  repeated Container containers = 1; // 容器列表
  int32 total = 2;                   // 总数
  int32 page = 3;                    // 当前页
  int32 page_size = 4;               // 每页条数
}

// Container 容器信息
message Container {
  string id = 1;                  // 容器ID（短格式，12字符）
  string name = 2;                // 容器名称
  string image = 3;               // 镜像名称
  string image_id = 4;            // 镜像ID
  string state = 5;               // 状态：created, running, paused, exited, dead
  string status = 6;              // 详细状态描述
  int64 created = 7;              // 创建时间（Unix时间戳）
  int64 started_at = 8;           // 启动时间
  int64 finished_at = 9;          // 停止时间
  repeated Port ports = 10;       // 端口映射
  repeated Mount mounts = 11;     // 挂载卷
  map<string, Network> networks = 12; // 网络配置
  repeated string env = 13;       // 环境变量
  map<string, string> labels = 14; // 标签
  repeated string command = 15;   // 启动命令
  int64 cpu_limit = 16;           // CPU限制
  int64 memory_limit = 17;        // 内存限制
  int32 restart_count = 18;       // 重启次数
  string restart_policy = 19;     // 重启策略
}

// Port 端口映射
message Port {
  string ip = 1;           // 绑定IP
  int32 private_port = 2;  // 容器内端口
  int32 public_port = 3;   // 主机端口
  string type = 4;         // tcp/udp
}

// Mount 挂载卷
message Mount {
  string type = 1;        // bind, volume, tmpfs
  string source = 2;      // 源路径/卷名
  string destination = 3; // 容器内路径
  string mode = 4;        // rw, ro
  bool rw = 5;            // 是否可写
}

// Network 网络配置
message Network {
  string network_id = 1;   // 网络ID
  string gateway = 2;      // 网关地址
  string ip_address = 3;   // 容器IP
  int32 ip_prefix_len = 4; // IP前缀长度
  string mac_address = 5;  // MAC地址
}

// GetContainerRequest 获取容器详情请求
message GetContainerRequest {
  string container_id = 1; // 容器ID（支持短ID和完整ID）
}

// ContainerActionRequest 容器操作请求（通用）
message ContainerActionRequest {
  string container_id = 1; // 容器ID
  int32 timeout = 2;       // 超时时间（秒），仅用于stop/restart
}

// DeleteContainerRequest 删除容器请求
message DeleteContainerRequest {
  string container_id = 1; // 容器ID
  bool force = 2;          // 强制删除（停止并删除）
  bool remove_volumes = 3; // 删除关联的匿名卷
}

// ContainerActionResponse 容器操作响应（通用）
message ContainerActionResponse {
  bool success = 1;        // 操作是否成功
  string message = 2;      // 响应消息
  string container_id = 3; // 容器ID
  int64 duration = 4;      // 操作耗时（毫秒）
}

// GetContainerStatsRequest 获取容器统计请求
message GetContainerStatsRequest {
  string container_id = 1; // 容器ID
  bool stream = 2;         // true=持续推送，false=单次查询
}

// ContainerStats 容器资源统计
message ContainerStats {
  string container_id = 1;        // 容器ID
  int64 timestamp = 2;            // 时间戳

  // CPU统计
  double cpu_usage_percent = 3;   // CPU使用率（百分比）
  uint64 cpu_usage_nano = 4;      // CPU使用量（纳秒）

  // 内存统计
  uint64 memory_usage = 5;        // 内存使用量（字节）
  uint64 memory_limit = 6;        // 内存限制（字节）
  double memory_usage_percent = 7; // 内存使用率（百分比）

  // 网络统计
  uint64 network_rx_bytes = 8;    // 网络接收字节数
  uint64 network_tx_bytes = 9;    // 网络发送字节数

  // 磁盘IO统计
  uint64 block_read_bytes = 10;   // 磁盘读取字节数
  uint64 block_write_bytes = 11;  // 磁盘写入字节数

  // PIDs
  uint64 pids_current = 12;       // 当前进程数
}

// GetContainerLogsRequest 获取容器日志请求
message GetContainerLogsRequest {
  string container_id = 1;  // 容器ID
  bool follow = 2;          // true=流式跟踪，false=历史日志
  int32 tail_lines = 3;     // 最后N行（历史日志模式，0=全部）
  int64 since_timestamp = 4; // Unix时间戳（可选）
  bool timestamps = 5;      // 是否包含时间戳
}

// LogEntry 日志条目
message LogEntry {
  string timestamp = 1; // 时间戳（RFC3339格式）
  string stream = 2;    // stdout/stderr
  string log = 3;       // 日志内容
}

// ExecRequest 容器命令执行请求（双向流）
message ExecRequest {
  oneof request {
    ExecStart start = 1;  // 启动执行
    ExecInput input = 2;  // 输入数据（终端输入）
    ExecResize resize = 3; // 调整终端大小
  }
}

// ExecStart 启动执行
message ExecStart {
  string container_id = 1;   // 容器ID
  repeated string cmd = 2;   // 命令，如 ["/bin/sh"]
  bool tty = 3;              // 是否分配TTY
  bool attach_stdin = 4;     // 是否附加stdin
  bool attach_stdout = 5;    // 是否附加stdout
  bool attach_stderr = 6;    // 是否附加stderr
  map<string, string> env = 7; // 环境变量
  string working_dir = 8;    // 工作目录
}

// ExecInput 输入数据
message ExecInput {
  bytes data = 1; // 输入数据（终端输入）
}

// ExecResize 调整终端大小
message ExecResize {
  int32 rows = 1; // 行数
  int32 cols = 2; // 列数
}

// ExecResponse 容器命令执行响应（双向流）
message ExecResponse {
  oneof response {
    ExecOutput output = 1; // 输出数据
    ExecExit exit = 2;     // 执行结束
  }
}

// ExecOutput 输出数据
message ExecOutput {
  bytes data = 1; // 输出数据（stdout/stderr）
}

// ExecExit 执行结束
message ExecExit {
  int32 exit_code = 1; // 退出码
}
```

### 镜像操作

```protobuf
// ListImagesRequest 镜像列表请求
message ListImagesRequest {
  bool all = 1;       // true=所有镜像（包括中间层），false=仅顶层镜像
  string filter = 2;  // Docker原生filter，如 "dangling=true" "reference=nginx"
}

// ListImagesResponse 镜像列表响应
message ListImagesResponse {
  repeated Image images = 1; // 镜像列表
  int32 total = 2;           // 总数
}

// Image 镜像信息
message Image {
  string id = 1;                  // 镜像ID（短格式）
  repeated string repo_tags = 2;  // 标签列表
  repeated string repo_digests = 3; // 摘要列表
  int64 size = 4;                 // 镜像大小（字节）
  int64 virtual_size = 5;         // 虚拟大小（包含共享层）
  int64 created = 6;              // 创建时间（Unix时间戳）
  map<string, string> labels = 7; // 标签
  repeated string layers = 8;     // 镜像层SHA256列表

  // 详情字段（仅GetImage时返回）
  string comment = 9;
  string author = 10;
  string architecture = 11;
  string os = 12;
  ImageConfig config = 13;
  repeated ImageHistory history = 14;
}

// ImageConfig 镜像配置
message ImageConfig {
  repeated string env = 1;         // 环境变量
  repeated string cmd = 2;         // 默认命令
  repeated string entrypoint = 3;  // 入口点
  string working_dir = 4;          // 工作目录
  map<string, Empty> exposed_ports = 5; // 暴露端口
  map<string, Empty> volumes = 6;  // 数据卷
  map<string, string> labels = 7;  // 标签
}

// Empty 空消息（用于map<string, Empty>）
message Empty {}

// ImageHistory 镜像历史记录
message ImageHistory {
  int64 created = 1;     // 创建时间
  string created_by = 2; // 创建命令
  int64 size = 3;        // 该层大小
  string comment = 4;    // 注释
  bool empty_layer = 5;  // 是否为空层
}

// GetImageRequest 获取镜像详情请求
message GetImageRequest {
  string image_id = 1; // 镜像ID或标签
}

// DeleteImageRequest 删除镜像请求
message DeleteImageRequest {
  string image_id = 1; // 镜像ID或标签
  bool force = 2;      // 强制删除（即使被容器使用）
  bool no_prune = 3;   // 不删除未标记的父镜像
}

// DeleteImageResponse 删除镜像响应
message DeleteImageResponse {
  bool success = 1;          // 操作是否成功
  string message = 2;        // 响应消息
  repeated string deleted = 3; // 已删除的镜像ID列表
  repeated string untagged = 4; // 已取消标记的镜像标签列表
}

// PullImageRequest 拉取镜像请求
message PullImageRequest {
  string image_name = 1;      // 镜像名称，如 "nginx:latest" "registry.example.com/app:v1"
  RegistryAuth auth = 2;      // Registry认证（可选）
  string platform = 3;        // 平台，如 "linux/amd64"（可选）
}

// RegistryAuth Registry认证信息
message RegistryAuth {
  string username = 1; // 用户名
  string password = 2; // 密码（Server端解密后传递）
}

// PullImageProgress 拉取镜像进度（流式）
message PullImageProgress {
  string status = 1;   // 状态描述，如 "Pulling from library/nginx"
  string progress = 2; // 进度信息，如 "50% [==========>"
  string id = 3;       // 镜像层ID
  int64 current = 4;   // 当前字节数
  int64 total = 5;     // 总字节数
}

// TagImageRequest 给镜像打标签请求
message TagImageRequest {
  string source_image = 1; // 源镜像ID或标签
  string target_repo = 2;  // 目标仓库，如 "myregistry.com/myimage"
  string target_tag = 3;   // 目标标签，如 "v1.0"
}

// TagImageResponse 打标签响应
message TagImageResponse {
  bool success = 1;   // 操作是否成功
  string message = 2; // 响应消息
}
```

---

## 错误处理

**gRPC状态码映射**：

| 场景 | gRPC状态码 | 描述 |
|------|-----------|------|
| 容器不存在 | NOT_FOUND | Container not found: abc123 |
| 容器已停止，无法停止 | FAILED_PRECONDITION | Container is already stopped |
| Docker守护进程连接失败 | UNAVAILABLE | Cannot connect to Docker daemon |
| 操作超时 | DEADLINE_EXCEEDED | Operation timeout after 30s |
| 无效参数 | INVALID_ARGUMENT | Invalid container ID format |
| 权限不足 | PERMISSION_DENIED | Permission denied to access Docker socket |
| 内部错误 | INTERNAL | Docker API error: ... |

**错误响应示例**：
```go
// Agent端错误处理
if err := dockerClient.ContainerStop(ctx, containerID, timeout); err != nil {
    if client.IsErrNotFound(err) {
        return nil, status.Errorf(codes.NotFound, "Container not found: %s", containerID)
    }
    return nil, status.Errorf(codes.Internal, "Failed to stop container: %v", err)
}
```

---

## 超时和重试策略

**超时配置**（研究任务2决策）：
- 连接超时：5秒
- 读取超时：30秒（普通RPC）
- 流式超时：无限制（日志流、终端、stats流）

**Server端超时控制**：
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := dockerClient.ListContainers(ctx, &pb.ListContainersRequest{})
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        return nil, fmt.Errorf("Agent超时（30秒无响应）")
    }
    return nil, err
}
```

**重试策略**：
- 不重试（默认）：容器操作具有副作用，避免重复执行
- 可重试场景：仅读取操作（GetDockerInfo、ListContainers、GetContainer）
- 重试次数：最多2次，指数退避（1秒、2秒）

---

## 性能和限制

**分页限制**：
- 默认每页50条
- 最大每页1000条
- 超过限制返回 INVALID_ARGUMENT 错误

**流式传输限制**：
- 日志流：单次推送最多10000行
- Stats流：每1秒推送一次
- 镜像拉取进度：每完成一层推送一次

**并发限制**：
- 单Agent同时支持10个RPC调用
- 超出限制返回 RESOURCE_EXHAUSTED 错误

---

## 兼容性

**Docker API版本**：
- 最低支持：API v1.41（Docker 20.10+）
- 推荐版本：API v1.43+（Docker 24.0+）
- Agent启动时检测Docker版本，不兼容时拒绝启动

**API版本协商**：
```go
// Agent初始化时
cli, err := client.NewClientWithOpts(
    client.FromEnv,
    client.WithAPIVersionNegotiation(), // 自动协商
)
```

---

## 安全性

**认证和授权**：
- Agent与Server之间通过TLS双向认证
- Agent检查Server证书，防止中间人攻击
- Docker操作权限由Docker守护进程控制（Agent需有Docker socket访问权限）

**数据传输安全**：
- 所有gRPC通信通过TLS加密
- Registry密码由Server加密存储，解密后通过TLS传输到Agent

**防护措施**：
- 容器命令执行限制：仅允许交互式Shell（/bin/sh、/bin/bash）
- 环境变量过滤：禁止设置敏感环境变量（如PATH覆盖）
- 日志大小限制：单次日志查询最多10000行，防止内存溢出

---

## 测试覆盖

**契约测试**（tests/contract/docker/agent_grpc_test.go）：
```go
func TestDockerServiceContract(t *testing.T) {
    // 启动Mock Agent gRPC Server
    server := startMockAgentServer(t)
    defer server.Stop()

    // 创建gRPC客户端
    conn := dialAgent(t, server.Address())
    client := pb.NewDockerServiceClient(conn)

    // 测试GetDockerInfo契约
    t.Run("GetDockerInfo", func(t *testing.T) {
        resp, err := client.GetDockerInfo(context.Background(), &pb.GetDockerInfoRequest{})
        require.NoError(t, err)
        assert.NotEmpty(t, resp.Version)
        assert.NotEmpty(t, resp.ApiVersion)
    })

    // 测试ListContainers契约
    t.Run("ListContainers", func(t *testing.T) {
        resp, err := client.ListContainers(context.Background(), &pb.ListContainersRequest{
            All: true,
            PageSize: 50,
        })
        require.NoError(t, err)
        assert.GreaterOrEqual(t, resp.Total, int32(0))
    })

    // ... 其他RPC测试
}
```

**集成测试**（使用testcontainers启动真实Docker环境）：
- 测试Agent连接真实Docker守护进程
- 测试容器生命周期操作
- 测试日志流和stats流
- 测试镜像拉取和删除

---

## 实施指南

**Agent端实现步骤**：
1. 创建 `pkg/grpc/proto/docker/docker.proto`
2. 运行 `protoc` 生成Go代码
3. 实现 `DockerServiceServer` 接口
4. 集成到现有Agent gRPC Server
5. 添加Docker客户端初始化和健康检查

**Server端实现步骤**：
1. 生成gRPC客户端代码
2. 实现 `AgentForwarder` 服务（调用Agent gRPC）
3. 添加连接池和超时控制
4. 实现缓存层（5分钟TTL）
5. 集成到业务服务层

**protoc编译命令**：
```bash
protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  pkg/grpc/proto/docker/docker.proto
```

---

**协议版本**：v1.0.0
**创建时间**：2025-10-22
**状态**：草稿，待契约测试验证
