# 技术研究:主机管理子系统-Nezha监控与WebSSH集成

**功能分支**:`002-nezha-webssh`
**研究日期**:2025-10-07
**相关规格**:[spec.md](./spec.md)

## 研究概述

本文档记录了主机监控与WebSSH功能的技术研究和决策过程。该功能需要实现Agent-Server架构的实时监控系统,参考了Nezha项目的成熟设计。

## 技术决策

### 决策1:通信协议选择

**选择**:gRPC双向流式通信

**理由**:
- **实时性需求**:主机监控数据需要实时推送,gRPC的双向流式RPC完美支持持久连接和实时数据传输
- **二进制协议**:Protobuf序列化比JSON更高效,减少网络带宽和CPU开销,适合高频监控数据传输
- **双向通信**:Server需要主动向Agent下发任务(如WebSSH、服务探测),gRPC原生支持双向流
- **类型安全**:Protobuf提供强类型定义,减少序列化错误
- **现有基础**:项目已使用Kubernetes client-go(基于gRPC),技术栈一致

**考虑的替代方案**:
- **WebSocket**:虽然也支持双向通信,但需要自定义协议和消息格式,缺乏类型安全;更适合浏览器场景
- **HTTP长轮询**:实时性差,Server资源消耗高,不适合高频监控数据
- **MQTT**:IoT场景更常用,Go生态支持不如gRPC,增加学习成本

### 决策2:Agent部署架构

**选择**:独立二进制Agent + systemd服务管理

**理由**:
- **轻量级**:Agent作为单一Go二进制文件,无外部依赖,部署简单
- **跨平台**:Go编译支持Linux/Windows/macOS多平台,覆盖主流操作系统
- **自动重连**:Agent内置心跳和重连机制,网络故障后自动恢复
- **资源占用低**:Go运行时内存占用小(~10-20MB),CPU开销低,适合在生产主机上运行
- **服务化管理**:通过systemd/Windows Service实现开机自启和进程守护

**考虑的替代方案**:
- **容器化Agent**:增加Docker依赖,不适合非容器化环境;监控宿主机需要特权模式
- **脚本Agent(Python/Shell)**:需要运行时环境,依赖管理复杂,性能不如编译型语言
- **Sidecar模式**:仅适用于Kubernetes环境,无法监控物理机和虚拟机

### 决策3:监控数据存储策略

**选择**:时序数据库(可选Prometheus)+关系数据库混合存储

**理由**:
- **关系数据库(SQLite/PostgreSQL)**:
  - 存储主机元数据(名称、UUID、分组、配置等)
  - 存储告警规则、服务探测规则等配置数据
  - 存储告警事件历史(便于关联查询)
  - 项目已有GORM抽象,复用现有技术栈

- **时序数据可选策略**:
  - **短期数据(7天)**:存储在关系数据库,满足快速查询和趋势展示
  - **长期存储(可选)**:接入Prometheus,利用其高效的时序存储和查询能力
  - **灵活性**:初期仅用关系数据库,后续可无缝对接Prometheus

**考虑的替代方案**:
- **仅Prometheus**:需要额外部署,增加系统复杂度;查询配置数据不便
- **仅关系数据库**:长期存储监控数据会导致表膨胀,查询性能下降;但可通过数据保留策略缓解
- **InfluxDB**:专业时序数据库,但Go生态不如Prometheus成熟,社区支持较少

### 决策4:WebSSH实现方案

**选择**:gRPC流式代理 + WebSocket浏览器连接

**理由**:
- **双层代理架构**:
  - 浏览器 ↔ (WebSocket) ↔ Server ↔ (gRPC Stream) ↔ Agent ↔ (SSH) ↔ 主机
  - Server作为中间代理,协调WebSocket和gRPC流,隔离浏览器和Agent
- **安全性**:
  - Agent在主机本地执行SSH,无需暴露SSH端口到公网
  - Server统一鉴权和审计,所有会话可追溯
  - SSH凭证仅在Agent端使用,不经过网络传输
- **实时性**:gRPC和WebSocket都是全双工通信,终端IO无延迟
- **会话管理**:
  - Server维护会话ID映射(WebSocket ↔ gRPC Stream)
  - 支持会话超时和主动关闭
  - 可记录会话操作日志用于审计

**考虑的替代方案**:
- **直接SSH代理**:需要Server到目标主机的网络可达,不适合NAT后的内网主机
- **VNC/RDP over WebSocket**:图形界面开销大,非必需;终端操作CLI更高效
- **Web-based IDE**:功能过重,不符合运维场景需求

### 决策5:服务探测执行策略

**选择**:Agent端分布式探测 + Server端调度协调

**理由**:
- **分布式探测**:
  - 探测任务由Agent执行,可从不同网络位置验证服务可用性
  - 避免单点探测导致的网络盲区(如内网服务、跨区域服务)
  - Agent本地探测延迟更准确,反映真实用户体验
- **Server调度**:
  - Server维护探测规则和调度策略
  - 通过gRPC下发探测任务到指定Agent
  - 聚合探测结果,计算可用性统计
  - 触发告警和通知
- **Cron调度**:
  - 使用robfig/cron库实现定时任务
  - 支持灵活的探测频率(秒级/分钟级)
  - 探测任务并发执行,互不阻塞

**考虑的替代方案**:
- **Server端中心化探测**:无法探测内网服务,网络路径单一,不够灵活
- **仅Agent自主探测**:缺乏统一管理,规则配置分散,难以维护
- **第三方服务(如Pingdom)**:外部依赖,数据隐私问题,成本高

### 决策6:告警规则引擎

**选择**:表达式引擎(expr) + 规则持久化

**理由**:
- **灵活的规则定义**:
  - 使用类SQL的表达式定义告警条件(如`cpu > 90 AND duration > 300`)
  - 支持逻辑运算(AND/OR/NOT)和比较运算(>/</==/!=)
  - 支持聚合函数(avg/max/min)和时间窗口
- **规则引擎**:
  - 使用antonmedv/expr库解析和执行表达式
  - 类型安全,性能高,支持复杂逻辑
  - 可扩展自定义函数(如地理位置、时间范围判断)
- **规则存储**:
  - 规则定义存储在数据库(models.MonitorAlertRule)
  - 支持热更新,无需重启服务
  - 关联主机、服务、通知组

**考虑的替代方案**:
- **硬编码规则**:不灵活,每次变更需重新部署
- **Lua/JavaScript脚本**:引入脚本引擎复杂度高,安全风险大
- **Prometheus AlertManager**:功能强大但需额外部署,对简单场景过重

### 决策7:前端实时数据展示

**选择**:WebSocket + React状态管理(Zustand)

**理由**:
- **WebSocket推送**:
  - Server通过WebSocket主动推送监控数据到浏览器
  - 避免前端轮询,减少请求数量和延迟
  - 支持多用户同时订阅,Server端Pub/Sub模式
- **React状态管理**:
  - 使用Zustand(轻量级状态库)管理实时数据
  - 相比Redux更简单,无样板代码
  - 支持选择性订阅,优化渲染性能
- **图表库**:
  - 使用Recharts渲染监控趋势图表
  - 支持实时数据流更新,动画流畅
  - 与TailwindCSS样式统一

**考虑的替代方案**:
- **HTTP轮询**:实时性差,Server压力大,浪费带宽
- **Server-Sent Events(SSE)**:单向推送,无法从浏览器发送命令(如暂停订阅)
- **GraphQL Subscription**:需要额外的GraphQL层,复杂度高

## 技术栈总结

### 后端新增技术
- **gRPC**:Agent-Server通信(github.com/grpc/grpc-go)
- **Protobuf**:消息序列化(google.golang.org/protobuf)
- **WebSocket**:浏览器实时通信(github.com/gorilla/websocket,已有)
- **Cron调度**:服务探测定时任务(github.com/robfig/cron)
- **表达式引擎**:告警规则(github.com/antonmedv/expr)

### Agent技术栈
- **Go 1.24+**:编译为单一二进制
- **gRPC Client**:连接Server
- **gopsutil**:系统监控数据采集(CPU/内存/磁盘/网络)
- **SSH库**:WebSSH本地执行(golang.org/x/crypto/ssh)
- **HTTP/TCP/ICMP客户端**:服务探测

### 前端新增技术
- **Zustand**:轻量级状态管理(用于实时数据)
- **Recharts**:监控图表库(react-recharts)
- **xterm.js**:WebSSH终端UI(已有,用于K8s终端)

## 性能考虑

### 可扩展性
- **水平扩展**:Server无状态设计(会话存储在内存或Redis),可部署多实例
- **Agent连接池**:Server维护Agent连接池,复用gRPC连接
- **数据聚合**:监控数据按主机和时间窗口聚合,减少存储和查询压力

### 性能目标
- **Agent资源占用**:内存<30MB,CPU<5%(空闲时<1%)
- **数据传输**:监控数据每30秒上报一次,每条消息<1KB
- **WebSSH延迟**:<50ms(局域网),<200ms(广域网)
- **服务探测**:支持1000+并发探测任务
- **Dashboard响应**:页面加载<2s,数据刷新<500ms

### 数据保留策略
- **实时数据**:最近24小时,秒级精度
- **短期数据**:7天,分钟级聚合
- **长期数据**:30天,小时级聚合(可选Prometheus存储90天+)
- **告警历史**:永久保留(或根据配置清理)

## 安全考虑

### Agent认证
- **密钥验证**:Agent启动时使用预配置密钥(UUID)连接Server
- **TLS加密**:gRPC连接强制TLS,防止中间人攻击
- **密钥轮换**:支持密钥过期和定期更新机制

### WebSSH安全
- **访问控制**:基于用户角色和主机权限控制WebSSH访问
- **会话审计**:记录所有终端操作(命令、时间、用户)
- **会话超时**:空闲5分钟自动断开,防止遗留会话
- **SSH密钥管理**:密钥仅存储在Agent端,Server不持有

### 数据隐私
- **敏感数据加密**:Agent连接密钥使用AES-256加密存储
- **日志脱敏**:避免记录SSH密码、密钥等敏感信息
- **RBAC隔离**:不同用户只能看到有权限的主机数据

## 集成现有系统

### 告警系统集成
- **复用现有通知渠道**:邮件、Webhook(internal/services/notification/)
- **告警规则存储**:复用models.Alert和AlertEvent结构
- **告警处理器**:扩展services.alert.AlertProcessor支持主机告警

### 调度器集成
- **复用Scheduler**:services.scheduler.Scheduler管理服务探测Cron任务
- **任务注册**:探测规则创建时动态注册Cron Job
- **任务清理**:规则删除时注销对应任务

### RBAC集成
- **权限控制**:复用pkg/rbac/系统
- **资源级权限**:主机作为资源对象,支持查看/操作/WebSSH权限
- **角色定义**:扩展现有角色支持主机管理权限

## 开发优先级

### Phase 1:核心监控(优先级:高)
1. Agent基础框架(gRPC客户端、心跳、数据采集)
2. Server端Agent连接管理和数据接收
3. 主机CRUD API和前端页面
4. 实时监控数据展示(WebSocket推送)
5. 历史数据存储和趋势图表

### Phase 2:服务探测(优先级:高)
1. 服务探测规则CRUD API
2. Agent端探测任务执行(HTTP/TCP/ICMP)
3. Server端任务调度和结果聚合
4. 探测结果展示和可用性统计
5. 探测告警集成

### Phase 3:WebSSH(优先级:中)
1. Agent端SSH执行器
2. Server端WebSocket-gRPC代理
3. 前端xterm.js终端集成
4. 会话管理和超时控制
5. 操作审计日志

### Phase 4:高级功能(优先级:低)
1. 主机分组管理
2. 自定义告警规则(表达式引擎)
3. Prometheus集成(可选)
4. Agent自动升级
5. 多地域部署支持

## 风险与缓解

### 风险1:大规模Agent连接
- **风险**:数百个Agent同时连接,Server内存和连接数压力
- **缓解**:
  - 连接池管理,设置最大连接数限制
  - 监控数据批量上报,减少消息频率
  - 水平扩展Server,负载均衡

### 风险2:网络不稳定
- **风险**:Agent与Server之间网络波动导致频繁断线
- **缓解**:
  - Agent指数退避重连策略
  - Server端连接超时容忍(如60秒无心跳才标记离线)
  - 监控数据缓冲,网络恢复后批量同步

### 风险3:WebSSH会话泄露
- **风险**:未授权用户访问WebSSH或会话劫持
- **缓解**:
  - 严格的RBAC权限检查
  - WebSocket连接JWT验证
  - 会话ID加密和有效期限制
  - 审计日志追溯

### 风险4:Agent版本碎片化
- **风险**:不同版本Agent与Server协议不兼容
- **缓解**:
  - Agent上报版本信息,Server检测兼容性
  - Protobuf向后兼容设计(optional字段)
  - 提供Agent自动升级机制
  - 文档明确版本兼容性

## 测试策略

### 单元测试
- Agent数据采集准确性(CPU/内存/磁盘计算)
- 服务探测逻辑(HTTP/TCP/ICMP)
- 告警规则表达式解析
- WebSSH命令执行和IO处理

### 集成测试
- Agent-Server gRPC通信
- WebSocket实时数据推送
- 服务探测任务调度和结果聚合
- 告警触发和通知发送

### 端到端测试
- 用户添加主机并部署Agent
- 实时监控数据展示
- 服务探测配置和告警
- WebSSH终端操作

### 性能测试
- 100+Agent并发连接
- 1000+服务探测并发执行
- WebSSH并发会话(10+)
- 监控数据查询性能(历史7天)

## 参考资源

- Nezha项目:https://github.com/nezhahq/nezha(架构参考)
- gRPC Go文档:https://grpc.io/docs/languages/go/
- gopsutil库:https://github.com/shirou/gopsutil(系统监控)
- Gorilla WebSocket:https://github.com/gorilla/websocket
- Robfig Cron:https://github.com/robfig/cron
- Expr引擎:https://github.com/antonmedv/expr
- xterm.js:https://xtermjs.org/(终端UI)

## 结论

基于Nezha项目的成熟设计,我们选择了gRPC双向流式通信、独立Agent部署、混合数据存储等技术方案。该架构具备高实时性、可扩展性和安全性,能够满足企业级主机监控需求。

通过分阶段实施,优先完成核心监控和服务探测功能,后续扩展WebSSH和高级特性,可有效控制开发风险和交付周期。

所有技术选型与现有Tiga项目技术栈保持一致,复用了GORM、Gin、React等基础设施,降低学习成本和维护复杂度。
