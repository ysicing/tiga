---
created: 2025-10-06T23:34:01+0800
last_updated: 2025-10-06T23:34:01+0800
version: 1.0
author: Claude Code PM System
---

# Tiga 项目简介

## 📝 项目概述

**项目名称**: Tiga
**项目类型**: 开源 DevOps Dashboard 平台
**当前版本**: v1.0.0 (开发中)
**许可证**: Apache License 2.0
**开发语言**: Go (后端) + TypeScript/React (前端)
**项目状态**: 快速迭代开发中

## 🎯 项目使命

构建一个**现代化、轻量级、易用的 DevOps Dashboard 平台**，帮助 DevOps 工程师、SRE 和系统管理员高效管理 Kubernetes 集群和各类基础设施资源。

## 💡 核心理念

### 1. 简单即强大
- 单个二进制文件部署
- Web 安装向导，5 分钟快速上手
- 直观的用户界面，降低学习成本

### 2. 开放与可扩展
- 开源免费（Apache 2.0）
- 插件化架构（规划中）
- 完善的 API 文档

### 3. 安全第一
- 完善的认证与授权（JWT + RBAC）
- 操作审计日志
- OAuth 集成支持

### 4. 性能优先
- 轻量级架构
- 高效的资源管理
- 实时数据更新

## 🏗️ 架构概览

### 技术架构
```
┌─────────────────────────────────────────┐
│         React 19 Frontend (Vite)        │
│  TailwindCSS + Radix UI + shadcn/ui     │
└─────────────────────────────────────────┘
                    ↕ HTTP/WebSocket
┌─────────────────────────────────────────┐
│          Go Backend (Gin)                │
│    Middleware → Service → Repository     │
└─────────────────────────────────────────┘
                    ↕
┌─────────────────────────────────────────┐
│  Database (SQLite/PostgreSQL/MySQL)      │
└─────────────────────────────────────────┘
                    ↕
┌─────────────────────────────────────────┐
│  External Systems (K8s, MinIO, Docker)   │
└─────────────────────────────────────────┘
```

### 功能模块
- **Kubernetes 管理**: 集群、资源、监控、日志、终端
- **实例管理**: VMs、数据库、MinIO、Docker、Web 服务器
- **用户管理**: 用户、角色、权限、OAuth
- **监控告警**: Prometheus 集成、告警规则、通知
- **审计日志**: 操作记录、资源历史、合规性

## 🎯 项目目标

### 短期目标（1-3 月）
1. ✅ 完成核心 Kubernetes 管理功能
2. ✅ 实现 Web 安装向导
3. 🚧 完善实例管理功能（MinIO、Docker、WebServer）
4. 🚧 增强监控和告警系统
5. 📝 完善用户文档和 API 文档

### 中期目标（3-6 月）
1. 📋 自定义仪表板功能
2. 📋 批量操作支持
3. 📋 Helm Chart 管理
4. 📋 GitOps 集成（ArgoCD/Flux）
5. 📋 多语言支持增强

### 长期目标（6-12 月）
1. 🔮 插件系统
2. 🔮 AI 辅助运维
3. 🔮 成本分析和优化
4. 🔮 多租户支持
5. 🔮 企业级功能（SSO、LDAP、高可用）

## 📊 项目范围

### 包含的功能
✅ Kubernetes 集群和资源管理
✅ 虚拟机和容器管理
✅ 对象存储管理（MinIO）
✅ 数据库实例管理
✅ 用户和权限管理
✅ 监控和告警
✅ 审计日志
✅ Web 终端和日志查看

### 不包含的功能
❌ 集群创建和销毁（使用现有集群）
❌ 资源的自动扩缩容（仅手动）
❌ 成本计费
❌ 备份和恢复（规划中）

## 🎯 成功标准

### 功能完整性
- [ ] 所有核心功能实现并稳定
- [ ] API 文档完整
- [ ] 用户文档齐全

### 性能指标
- [ ] 页面加载 < 2 秒
- [ ] API 响应 < 500ms
- [ ] 支持 1000+ Pods 的集群

### 用户体验
- [ ] 直观的 UI/UX
- [ ] 快速的操作响应
- [ ] 友好的错误提示

### 稳定性
- [ ] 系统可用性 > 99.9%
- [ ] 无严重 Bug
- [ ] 完善的错误处理

## 👥 项目团队（开源项目）

### 核心维护者
- **作者**: ysicing
- **贡献者**: 欢迎社区贡献

### 贡献方式
- 提交 Issue 报告 Bug
- 提交 Pull Request 贡献代码
- 完善文档
- 分享使用案例

## 🗓️ 项目里程碑

### Phase 1: 基础功能 (已完成 ✅)
- 核心 Kubernetes 资源管理
- 用户认证和授权
- 基础监控功能
- Web 安装向导

### Phase 2: 增强功能 (进行中 🚧)
- 多实例管理完善
- 告警系统增强
- 审计日志优化
- 用户体验改进

### Phase 3: 高级功能 (规划中 📋)
- 自定义仪表板
- 批量操作
- Helm Chart 管理
- GitOps 集成

### Phase 4: 企业级功能 (未来 🔮)
- 插件系统
- AI 辅助运维
- 多租户支持
- 高级安全功能

## 📈 项目进展跟踪

### 当前状态
- **功能完成度**: ~70%
- **测试覆盖率**: ~50%
- **文档完整性**: ~80%
- **性能优化**: ~60%

### 最近更新（2025-10）
- ✅ 完成 Web 安装向导
- ✅ 迁移日志系统到 logrus
- ✅ 实现多子系统架构
- 🚧 完善 MinIO/Docker/WebServer 管理页面

## 🔗 相关链接

- **GitHub 仓库**: https://github.com/ysicing/tiga
- **在线 Demo**: https://tiga-demo.zzde.me
- **文档**: https://tiga.zzde.me
- **Issue 追踪**: https://github.com/ysicing/tiga/issues

## 📋 依赖项

### 运行时依赖
- Kubernetes 集群（可选，用于 K8s 功能）
- 数据库（SQLite/PostgreSQL/MySQL）
- Web 浏览器（现代浏览器）

### 开发依赖
- Go 1.24+
- Node.js 18+ (pnpm)
- Task (任务运行器)
- Docker (可选，用于构建镜像)

## ⚠️ 已知限制

1. **浏览器支持**: 仅支持现代浏览器（Chrome、Firefox、Safari、Edge）
2. **集群规模**: 建议管理 < 5000 Pods 的集群
3. **并发用户**: 当前版本建议 < 100 并发用户
4. **数据库**: 生产环境建议使用 PostgreSQL 或 MySQL

## 🚀 快速开始

### Docker 运行
```bash
docker run -p 8080:8080 ghcr.io/ysicing/tiga:latest
```

### Kubernetes 部署
```bash
helm repo add tiga https://ysicing.github.io/tiga
helm install tiga tiga/tiga -n kube-system
```

### 从源码构建
```bash
git clone https://github.com/ysicing/tiga.git
cd tiga
task backend
./bin/tiga
```

## 📝 许可证

本项目采用 [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0) 许可证。

## 🙏 致谢

感谢以下开源项目：
- Kubernetes
- Gin
- GORM
- React
- Vite
- TailwindCSS
- Radix UI
- shadcn/ui
- Monaco Editor
- xterm.js
- Recharts
- 以及所有其他依赖项

## 🎯 为什么创建 Tiga？

### 现有方案的痛点
- **Kubernetes Dashboard**: 功能单一，UI 陈旧
- **Rancher**: 过于庞大，部署复杂
- **Lens**: 需要客户端安装，不支持多用户

### Tiga 的优势
- ✅ 轻量级：单个二进制文件
- ✅ 现代化：美观的 UI，优秀的 UX
- ✅ 全面：不仅是 K8s，还支持多种基础设施
- ✅ 易用：Web 安装向导，5 分钟部署
- ✅ 开源：Apache 2.0，完全免费

## 📞 联系方式

- **GitHub Issues**: https://github.com/ysicing/tiga/issues
- **Email**: (如有提供)
- **社区**: (Discord/Slack 链接，如有)

---

**最后更新**: 2025-10-06
**项目状态**: 活跃开发中
**欢迎贡献**: 是
