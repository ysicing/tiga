# 快速开始：构建版本信息注入与 Agent Docker 上报控制

**功能分支**：`008-commitid-commit-agent`
**日期**：2025-10-26

## 概述

本文档提供实施后的快速验证指南，包含所有用户故事的验收步骤。

## 前置条件

1. **开发环境**：
   - Go 1.24+
   - Node.js 18+ 和 pnpm
   - Git
   - Taskfile

2. **仓库状态**：
   - 当前分支：`008-commitid-commit-agent`
   - 所有实施任务已完成
   - 测试通过

3. **依赖服务**：
   - 无需额外服务（本功能无外部依赖）

## 验收场景

### 场景 1：构建版本信息注入

**目标**：验证构建时版本信息正确注入到二进制文件

**步骤**：

1. 确保在 git 仓库中且有 commit：
   ```bash
   cd /Users/ysicing/go/src/github.com/ysicing/tiga
   git log -1 --oneline
   # 预期输出：<commit-hash> <commit-message>
   ```

2. 构建服务端二进制：
   ```bash
   task backend
   # 或单独构建
   task build:server
   ```

3. 构建 Agent 二进制：
   ```bash
   task build:agent
   ```

4. 验证版本脚本输出：
   ```bash
   bash scripts/version.sh
   # 预期输出示例：
   # VERSION=v1.2.3-a1b2c3d  (如果有tag)
   # 或
   # VERSION=20251026-a1b2c3d (无tag)
   # BUILD_TIME=2025-10-26T10:30:00Z
   # COMMIT_ID=a1b2c3d
   ```

**验收标准**：
- ✅ 脚本成功生成版本号（格式正确）
- ✅ 构建过程无报错
- ✅ 二进制文件成功生成（bin/tiga、bin/tiga-agent）

---

### 场景 2：启动时显示版本信息

**目标**：验证应用启动时在日志中显示版本信息

**步骤**：

1. 启动服务端（不使用 --version 参数）：
   ```bash
   ./bin/tiga
   ```

2. 查看启动日志前几行：
   ```
   预期输出包含：
   INFO[0000] Starting Tiga Server
       version=v1.2.3-a1b2c3d
       build_time=2025-10-26T10:30:00Z
       commit_id=a1b2c3d
   ```

3. 停止服务端（Ctrl+C）

4. 启动 Agent（不使用 --version 参数）：
   ```bash
   ./bin/tiga-agent
   ```

5. 查看 Agent 启动日志：
   ```
   预期输出包含：
   INFO[0000] Starting Tiga Agent
       version=v1.2.3-a1b2c3d
       build_time=2025-10-26T10:30:00Z
       commit_id=a1b2c3d
   ```

**验收标准**：
- ✅ 服务端启动日志显示版本信息
- ✅ Agent 启动日志显示版本信息
- ✅ 版本信息包含 version、build_time、commit_id 三个字段
- ✅ 版本信息与构建时注入的值一致

---

### 场景 3：版本查询命令

**目标**：验证 --version 命令行选项显示版本信息并退出

**步骤**：

1. 执行服务端 --version：
   ```bash
   ./bin/tiga --version
   ```

   预期输出：
   ```
   Tiga Server
   Version:    v1.2.3-a1b2c3d
   Build Time: 2025-10-26T10:30:00Z
   Commit ID:  a1b2c3d
   ```

2. 验证退出码：
   ```bash
   echo $?
   # 预期：0
   ```

3. 执行 Agent --version：
   ```bash
   ./bin/tiga-agent --version
   ```

   预期输出：
   ```
   Tiga Agent
   Version:    v1.2.3-a1b2c3d
   Build Time: 2025-10-26T10:30:00Z
   Commit ID:  a1b2c3d
   ```

4. 测试 version 子命令（如支持）：
   ```bash
   ./bin/tiga version
   # 预期与 --version 相同
   ```

**验收标准**：
- ✅ --version 参数正确显示版本信息
- ✅ 显示版本后立即退出（不启动完整应用）
- ✅ 退出码为 0
- ✅ 输出格式清晰易读

---

### 场景 4：禁用 Docker 实例上报

**目标**：验证 Agent 配置禁用 Docker 上报后不上报 Docker 实例

**步骤**：

1. 创建测试配置文件 `config-test.yaml`：
   ```yaml
   agent:
     disable_docker_report: true
   ```

2. 使用配置文件启动 Agent：
   ```bash
   ./bin/tiga-agent --config config-test.yaml
   ```

3. 查看 Agent 日志：
   ```
   预期输出包含：
   INFO[xxxx] Docker instance reporting disabled
   ```

4. 在服务端查看 Agent 上报的数据，确认：
   - ✅ Agent 成功连接到服务端
   - ✅ 上报主机状态信息（CPU、内存等）
   - ✅ **不上报** Docker 实例信息

**验收标准**：
- ✅ Agent 启动成功
- ✅ 日志显示 "Docker instance reporting disabled"
- ✅ Agent 正常上报其他信息（主机状态）
- ✅ Agent 不上报 Docker 实例列表

---

### 场景 5：默认启用 Docker 实例上报

**目标**：验证 Agent 配置未设置或缺失时默认上报 Docker 实例

**步骤**：

1. 创建不包含 disable_docker_report 的配置文件：
   ```yaml
   # config-default.yaml
   agent:
     # disable_docker_report 字段缺失
   ```

2. 启动 Agent：
   ```bash
   ./bin/tiga-agent --config config-default.yaml
   ```

3. 查看日志，确认：
   - 无 "Docker instance reporting disabled" 日志
   - 有 Docker 实例上报相关日志（如 "Reporting 3 Docker instances"）

4. 在服务端查看上报数据：
   - ✅ 收到 Docker 实例列表
   - ✅ Docker 实例信息完整

**验收标准**：
- ✅ 配置缺失时 Agent 启动成功
- ✅ 默认上报 Docker 实例信息
- ✅ 服务端正确接收 Docker 实例数据

---

### 场景 6：服务端 API 查询版本信息

**目标**：验证前端页面通过 API 查询服务端版本信息

**步骤**：

1. 启动服务端：
   ```bash
   ./bin/tiga
   ```

2. 使用 cURL 测试 API：
   ```bash
   curl -X GET http://localhost:12306/api/v1/version
   ```

   预期输出：
   ```json
   {
     "version": "v1.2.3-a1b2c3d",
     "build_time": "2025-10-26T10:30:00Z",
     "commit_id": "a1b2c3d"
   }
   ```

3. 验证响应状态码和头部：
   ```bash
   curl -v http://localhost:12306/api/v1/version 2>&1 | grep "< HTTP"
   # 预期：< HTTP/1.1 200 OK

   curl -I http://localhost:12306/api/v1/version 2>&1 | grep "Content-Type"
   # 预期：Content-Type: application/json; charset=utf-8
   ```

4. 访问前端页面（如果已实现 UI）：
   ```bash
   # 启动前端开发服务器（如需要）
   cd ui && pnpm dev
   ```

5. 在浏览器访问设置页面（如 http://localhost:5174/settings）

6. 验证页面显示版本信息

**验收标准**：
- ✅ API 返回 200 OK
- ✅ 响应格式为 JSON
- ✅ 包含 version、build_time、commit_id 字段
- ✅ 字段值与构建时注入的值一致
- ✅ 前端页面正确显示版本信息（如已实现）

---

### 场景 7：Agent 上报版本信息

**目标**：验证 Agent 启动时将自身版本信息上报到服务端

**步骤**：

1. 启动服务端并查看日志：
   ```bash
   ./bin/tiga
   ```

2. 启动 Agent：
   ```bash
   ./bin/tiga-agent
   ```

3. 在服务端日志中查找 Agent 版本信息：
   ```
   预期输出包含：
   INFO[xxxx] Agent connected
       agent_version=v1.2.3-a1b2c3d
       agent_build_time=2025-10-26T10:30:00Z
       agent_commit_id=a1b2c3d
   ```

4. 通过服务端 API 查询 Agent 列表（如已实现）：
   ```bash
   curl http://localhost:12306/api/v1/agents
   ```

   验证响应中包含 Agent 版本信息：
   ```json
   {
     "agents": [
       {
         "id": "agent-1",
         "version_info": {
           "version": "v1.2.3-a1b2c3d",
           "build_time": "2025-10-26T10:30:00Z",
           "commit_id": "a1b2c3d"
         }
       }
     ]
   }
   ```

**验收标准**：
- ✅ Agent 成功连接到服务端
- ✅ 服务端日志记录 Agent 版本信息
- ✅ Agent 列表 API 返回版本信息（如已实现）
- ✅ 版本信息与 Agent 二进制版本一致

---

## 边缘情况测试

### 测试 1：无 git 环境构建

**步骤**：

1. 创建测试目录（非 git 仓库）：
   ```bash
   mkdir -p /tmp/tiga-test
   cp -r /Users/ysicing/go/src/github.com/ysicing/tiga/* /tmp/tiga-test/
   cd /tmp/tiga-test
   rm -rf .git
   ```

2. 执行构建：
   ```bash
   task backend
   ```

3. 验证版本信息：
   ```bash
   ./bin/tiga --version
   ```

   预期输出：
   ```
   Tiga Server
   Version:    dev
   Build Time: 2025-10-26T10:30:00Z
   Commit ID:  0000000
   ```

**验收标准**：
- ✅ 构建成功（不因缺少 git 而失败）
- ✅ Version 为 "dev"
- ✅ CommitID 为 "0000000"
- ✅ BuildTime 为实际构建时间

### 测试 2：环境变量覆盖配置

**步骤**：

1. 创建配置文件（启用 Docker 上报）：
   ```yaml
   agent:
     disable_docker_report: false
   ```

2. 使用环境变量覆盖：
   ```bash
   AGENT_DISABLE_DOCKER_REPORT=true ./bin/tiga-agent --config config.yaml
   ```

3. 验证日志显示 "Docker instance reporting disabled"

**验收标准**：
- ✅ 环境变量优先级高于 YAML 配置
- ✅ Docker 上报被禁用

---

## 性能验证

### 测试 1：版本 API 延迟

**步骤**：

1. 使用 ApacheBench 测试：
   ```bash
   ab -n 1000 -c 10 http://localhost:12306/api/v1/version
   ```

2. 查看性能指标：
   ```
   预期结果：
   - Requests per second: >1000 req/s
   - Time per request (mean): <10ms
   - 99% percentile: <20ms
   ```

**验收标准**：
- ✅ p99 延迟 <20ms（目标 <10ms）
- ✅ 吞吐量 >1000 req/s
- ✅ 无错误响应

### 测试 2：构建脚本性能

**步骤**：

1. 测量版本脚本执行时间：
   ```bash
   time bash scripts/version.sh
   ```

   预期输出：
   ```
   real    0m0.100s  (应 <5秒)
   user    0m0.050s
   sys     0m0.030s
   ```

**验收标准**：
- ✅ 脚本执行时间 <500ms
- ✅ 构建时间增加 <5秒

---

## 集成测试

### 测试 1：完整构建和部署流程

**步骤**：

1. 清理旧构建产物：
   ```bash
   task clean
   ```

2. 完整构建：
   ```bash
   task backend
   ```

3. 验证二进制文件：
   ```bash
   ./bin/tiga --version
   ./bin/tiga-agent --version
   ```

4. 运行测试套件：
   ```bash
   task test
   ```

5. 运行契约测试：
   ```bash
   go test ./tests/contract/version_api_test.go -v
   ```

**验收标准**：
- ✅ 所有测试通过
- ✅ 版本信息正确注入
- ✅ API 契约测试通过

---

## 故障排查

### 问题 1：版本号显示为 "dev"

**可能原因**：
- 不在 git 仓库中构建
- git 命令未安装
- detached HEAD 状态

**解决方法**：
```bash
# 检查 git 状态
git status

# 确保有 commit
git log -1

# 重新构建
task clean && task backend
```

### 问题 2：Agent Docker 上报未禁用

**可能原因**：
- 配置文件路径错误
- 配置项拼写错误
- 环境变量未生效

**解决方法**：
```bash
# 检查配置加载
./bin/tiga-agent --config config.yaml 2>&1 | grep -i "disable_docker_report"

# 验证环境变量
env | grep AGENT_DISABLE_DOCKER_REPORT

# 启用调试日志
LOG_LEVEL=debug ./bin/tiga-agent --config config.yaml
```

### 问题 3：版本 API 404 错误

**可能原因**：
- 路由未注册
- API 前缀错误

**解决方法**：
```bash
# 检查所有路由
curl http://localhost:12306/api/v1/

# 查看服务端日志
./bin/tiga 2>&1 | grep -i "version"
```

---

## 回归测试清单

执行以下命令验证所有场景：

```bash
# 1. 构建
task clean && task backend

# 2. 版本查询
./bin/tiga --version
./bin/tiga-agent --version

# 3. 启动测试
./bin/tiga &
sleep 5
curl http://localhost:12306/api/v1/version
kill %1

# 4. 测试套件
task test
go test ./tests/contract/version_api_test.go -v

# 5. 性能测试
ab -n 100 -c 10 http://localhost:12306/api/v1/version

# 6. 配置测试
echo "agent:\n  disable_docker_report: true" > config-test.yaml
./bin/tiga-agent --config config-test.yaml
rm config-test.yaml
```

**预期结果**：所有步骤无报错，输出符合预期

---

## 清理

```bash
# 停止所有进程
killall tiga tiga-agent

# 清理测试文件
rm -f config-test.yaml config-default.yaml

# 清理构建产物（可选）
task clean
```

---

## 总结

完成上述所有验收场景后，功能即可视为验收通过。如有任何失败场景，请参考故障排查部分或查看实施日志。
