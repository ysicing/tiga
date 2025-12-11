# Tiga 打包说明

本目录包含使用 nfpm 构建 Linux 软件包（deb、rpm、apk）的配置文件和脚本。

## 目录结构

```
packaging/
├── systemd/           # Systemd 服务文件
│   ├── tiga.service          # 主服务
│   └── tiga-agent.service    # Agent 服务
└── scripts/           # 安装/卸载脚本
    ├── preinstall.sh         # 安装前（创建用户）
    ├── postinstall.sh        # 安装后（提示信息）
    ├── preremove.sh          # 卸载前（停止服务）
    └── postremove.sh         # 卸载后（清理提示）
```

## 快速开始

### 1. 安装 nfpm

```bash
# 使用 go install
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

# 或使用包管理器
# macOS
brew install nfpm

# Linux
curl -sfL https://goreleaser.com/static/run | sh -s -- nfpm
```

### 2. 构建软件包

```bash
# 构建所有格式的包
task package:all

# 或单独构建
task package:deb    # Debian/Ubuntu (.deb)
task package:rpm    # RHEL/CentOS/Fedora (.rpm)
task package:apk    # Alpine Linux (.apk)
```

构建的软件包位于 `dist/` 目录。

## 软件包内容

### 文件布局

```
# 二进制文件
/usr/bin/tiga              # 主服务二进制
/usr/bin/tiga-agent        # Agent 二进制

# 配置文件（只存放配置，不存放数据）
/etc/tiga/config.yaml          # 主配置文件
/etc/tiga/config.yaml.example  # 配置示例
/etc/tiga/.env.example         # 环境变量示例

# 数据目录（存放所有应用数据）
/var/lib/tiga/                 # 数据根目录
/var/lib/tiga/tiga.db          # SQLite 数据库（默认）
/var/lib/tiga/recordings/      # 终端录制文件

# 日志目录
/var/log/tiga/                 # 日志文件（备用，默认使用 journal）

# Systemd 服务
/usr/lib/systemd/system/tiga.service        # 主服务
/usr/lib/systemd/system/tiga-agent.service  # Agent 服务
```

**重要**：严格遵循 Linux FHS 标准
- `/etc/tiga/` - 仅配置文件
- `/var/lib/tiga/` - 所有数据文件
- `/var/log/tiga/` - 日志文件

### 系统用户

软件包会自动创建：
- 用户：`tiga`
- 组：`tiga`
- Home：`/var/lib/tiga`
- Shell：`/sbin/nologin`（安全）

## 使用方法

### Debian/Ubuntu

```bash
# 安装
sudo dpkg -i dist/tiga_*.deb
sudo apt-get install -f  # 安装依赖

# 或使用 apt
sudo apt install ./dist/tiga_*.deb

# 卸载
sudo apt remove tiga
```

### RHEL/CentOS/Fedora

```bash
# 安装
sudo rpm -ivh dist/tiga-*.rpm

# 或使用 dnf/yum
sudo dnf install dist/tiga-*.rpm

# 卸载
sudo dnf remove tiga
```

### Alpine Linux

```bash
# 安装
sudo apk add --allow-untrusted dist/tiga_*.apk

# 卸载
sudo apk del tiga
```

## 安装后配置

### 🎉 首次设置 - 安装向导

Tiga 使用基于 Web 的安装向导进行首次设置（类似 WordPress）。

#### 1. 启动服务

```bash
sudo systemctl start tiga
```

#### 2. 访问安装向导

打开浏览器访问：**http://localhost:12306**

系统会自动重定向到安装向导页面：`/install`

#### 3. 完成安装向导

安装向导会引导你完成以下配置：

**步骤 1：数据库配置**
- 选择数据库类型：SQLite / PostgreSQL / MySQL
- SQLite（默认）：无需额外配置，自动使用 `/var/lib/tiga/tiga.db`
- PostgreSQL/MySQL：输入连接信息（主机、端口、用户名、密码）

**步骤 2：管理员账户**
- 设置管理员用户名（3-20 字符）
- 设置管理员密码（至少 8 字符）
- 输入管理员邮箱

**步骤 3：系统设置**
- 应用名称（默认：Tiga Dashboard）
- 域名（默认：localhost）
- HTTP 端口（默认：12306）
- 语言（中文/英文）

**步骤 4：完成**
- 安装向导会自动：
  - 初始化数据库表结构
  - 创建管理员账户
  - 生成 JWT 密钥和加密密钥
  - 更新配置文件并设置 `install_lock: true`

#### 4. 登录系统

安装完成后，使用管理员账户登录：
- 地址：http://localhost:12306
- 用户名：（安装时设置的）
- 密码：（安装时设置的）

#### 5. 启用开机自启

```bash
sudo systemctl enable tiga
```

## 升级

```bash
# Debian/Ubuntu
sudo apt install ./dist/tiga_*.deb

# RHEL/CentOS/Fedora
sudo dnf install dist/tiga-*.rpm

# Alpine
sudo apk add --allow-untrusted dist/tiga_*.apk
```

升级会：
- 自动停止旧服务
- 保留配置文件
- 保留数据目录
- 重新启动服务

## 卸载

### 卸载但保留数据

```bash
# Debian/Ubuntu
sudo apt remove tiga

# RHEL/CentOS/Fedora
sudo dnf remove tiga

# Alpine
sudo apk del tiga
```

### 完全卸载（包括数据）

```bash
# 1. 卸载软件包
sudo apt remove tiga  # 或 dnf/apk

# 2. 删除数据和配置
sudo rm -rf /etc/tiga /var/lib/tiga /var/log/tiga

# 3. 删除用户和组
sudo userdel tiga
sudo groupdel tiga
```

## 配置自定义

### 修改 nfpm.yaml

如果需要自定义软件包：

1. 编辑 `nfpm.yaml` 修改：
   - 软件包元数据（名称、描述、版本）
   - 文件映射（添加或删除文件）
   - 依赖关系
   - 脚本行为

2. 重新构建：
   ```bash
   task package:all
   ```

### 修改 Systemd 服务

编辑 `packaging/systemd/*.service` 可以自定义：
- 运行用户/组
- 环境变量
- 资源限制
- 安全加固选项
- 重启策略

### 修改安装脚本

编辑 `packaging/scripts/*.sh` 可以自定义：
- 用户创建逻辑
- 安装后提示信息
- 清理行为
- 迁移逻辑

## 故障排查

### 查看服务状态

```bash
sudo systemctl status tiga
sudo systemctl status tiga-agent
```

### 查看日志

```bash
# 实时日志
sudo journalctl -u tiga -f
sudo journalctl -u tiga-agent -f

# 最近 100 行
sudo journalctl -u tiga -n 100
```

### 手动运行

```bash
# 以 tiga 用户运行（调试）
sudo -u tiga /usr/bin/tiga
```

### 权限问题

```bash
# 确保目录所有权正确
sudo chown -R tiga:tiga /var/lib/tiga /var/log/tiga
sudo chmod 755 /var/lib/tiga /var/log/tiga
```

### 配置文件问题

```bash
# 验证配置文件语法
/usr/bin/tiga --config /etc/tiga/config.yaml --validate
```

## 依赖说明

### 必需依赖
- systemd（服务管理）
- adduser / shadow-utils（用户管理）

### 推荐依赖
- postgresql（数据库）
- docker.io（容器管理功能）

### 运行时依赖
- libc（通常已安装）
- Linux 内核 3.10+

## 开发者说明

### 测试打包

```bash
# 1. 构建软件包
task package:deb

# 2. 在 Docker 中测试安装
docker run -it --rm -v $(pwd)/dist:/packages ubuntu:22.04 bash
# 容器内执行：
apt update && apt install -y /packages/tiga_*.deb
systemctl status tiga
```

### CI/CD 集成

```yaml
# GitHub Actions 示例
- name: Build packages
  run: |
    task package:all

- name: Upload artifacts
  uses: actions/upload-artifact@v3
  with:
    name: packages
    path: dist/*.{deb,rpm,apk}
```

## 更多信息

- nfpm 文档: https://nfpm.goreleaser.com/
- Systemd 文档: https://www.freedesktop.org/software/systemd/man/
- 项目主页: https://github.com/ysicing/tiga
