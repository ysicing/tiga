# tiga - 现代化的 DevOps Dashboard

<div align="center">

<img src="./docs/assets/logo.svg" alt="tiga Logo" width="128" height="128">

_一个现代化、直观的 DevOps Dashboard_

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![React](https://img.shields.io/badge/React-19+-61DAFB?style=flat&logo=react)](https://reactjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5+-3178C6?style=flat&logo=typescript)](https://www.typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache-green.svg)](LICENSE)

[English](./README.md) | **中文**

</div>

tiga 是一个轻量级、现代化的 DevOps Dashboard。它提供实时指标、全面的资源管理、多集群支持和优美的用户体验。

> [!WARNING]
> 本项目正在快速迭代开发中，使用方式和 API 都有可能变化。

![Dashboard Overview](docs/screenshots/overview.png)
_全面的集群概览，包含实时指标和资源统计_

## ✨ 功能特性

### 🎯 **现代化的用户体验**

- 🌓 **多主题支持** - 暗色/亮色/彩色主题，并能自动适应系统偏好
- 🔍 **高级搜索** - 支持跨所有资源的全局搜索
- 🌐 **国际化支持** - 支持英文和中文语言
- 📱 **响应式设计** - 针对桌面、平板和移动设备优化

### 🏘️ **多集群管理**

- 🔄 **无缝集群切换** - 可在多个 Kubernetes 集群之间切换
- 📊 **分集群监控** - 每个集群可独立配置 Prometheus
- 🔐 **集群访问控制** - 集群访问管理的细粒度权限控制

### 🔍 **全面的资源管理**

- 📋 **全资源覆盖** - 支持 Pods, Deployments, Services, ConfigMaps, Secrets, PVs, PVCs, Nodes 等
- 📄 **实时 YAML 编辑** - 内置 Monaco 编辑器，支持语法高亮和校验
- 📊 **详细的资源视图** - 提供容器、卷、事件和状况等深入信息
- 🔗 **资源关系可视化** - 可视化相关资源之间的连接（例如，Deployment → Pods）
- ⚙️ **资源操作** - 直接从 UI 创建、更新、删除、扩缩容和重启资源
- 🔄 **自定义资源** - 完全支持 CRD (Custom Resource Definitions)
- 🏷️ **镜像标签快速选择器** - 基于 Docker 和容器镜像仓库 API，轻松选择和更改容器镜像标签

### 📈 **监控与可观测性**

- 📊 **实时指标** - 由 Prometheus 驱动的 CPU、内存、磁盘 I/O 和网络使用情况图表
- 📋 **集群概览** - 全面的集群健康状况和资源统计仪表板
- 📝 **实时日志** - 实时流式传输 Pod 日志，支持过滤和搜索
- 💻 **网页终端** - 直接在浏览器中进入 Pod/Node 执行命令
- 📈 **节点监控** - 详细的节点级别性能指标和利用率
- 📊 **Pod 监控** - 单个 Pod 资源使用情况和性能跟踪

### 🔐 **安全**

- 🛡️ **OAuth 集成** - 支持在 UI 管理 OAuth
- 🔒 **基于角色的访问控制** - 支持在 UI 管理用户的权限
- 👥 **用户管理** - 完整的用户管理和角色分配
- 🔐 **权限粒度** - 资源级别的精确访问控制权限

---

## 🚀 快速开始

### Docker

要使用 Docker 运行 tiga，您可以使用预构建的镜像：

```bash
docker run --rm -p 8080:8080 ghcr.io/ysicing/tiga:latest
```

### 在 Kubernetes 中部署

#### 使用 Helm (推荐)

1.  **添加 Helm 仓库**

    ```bash
    helm repo add tiga https://ysicing.github.io/tiga
    helm repo update
    ```

2.  **使用默认值安装**

    ```bash
    helm install tiga tiga/tiga -n kube-system
    ```

#### 使用 kubectl

1.  **应用部署清单**

    ```bash
    kubectl apply -f deploy/install.yaml
    # 或在线安装
    kubectl apply -f https://raw.githubusercontent.com/ysicing/tiga/refs/heads/main/deploy/install.yaml
    ```

2.  **通过端口转发访问**

    ```bash
    kubectl port-forward -n kube-system svc/tiga 8080:8080
    ```

### 从源码构建

#### 📋 准备工作

1.  **克隆仓库**

    ```bash
    git clone https://github.com/ysicing/tiga.git
    cd tiga
    ```

2.  **构建项目**

    ```bash
    make deps
    make build
    ```

    > 💡 **版本信息注入**: 构建过程会自动注入版本信息（版本号、构建时间、commit ID）到二进制文件中。
    > - 如果是 git 仓库，将使用 git 标签和 commit hash
    > - 如果没有 git 环境，将使用默认值（version: "dev", commit_id: "0000000"）

3.  **运行服务**

    ```bash
    make run
    ```

---

## 📌 版本信息

可通过以下方式查询版本信息：

- **命令行**: `./bin/tiga --version`
- **API 接口**: `GET /api/v1/version`
- **前端 UI**: 系统管理 → 全局配置 → 版本标签页

> 💡 构建过程会自动注入版本号、构建时间和 commit ID

---

## 🔍 问题排查

## 🤝 贡献

我们欢迎贡献，直接PR即可。

## 📄 许可证

本项目采用 Apache License 2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。
