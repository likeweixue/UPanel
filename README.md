# UPanel

轻量级 Linux 服务器运维面板：网站部署、数据库、应用商店、容器、文件与安全审计，
全部围绕 Docker 与统一 Nginx 网关展开。

当前版本：**v0.1.7**

---

## 特性

### 网站部署（核心）
- **统一网关**：单个 `upanel-gateway` 容器独占宿主机 80 / 443，按域名转发，站点容器不再映射宿主机端口。
- **三种站点类型**：
  - `static`：网关直接托管文件，零额外容器；
  - `php`：php-fpm 容器 + fastcgi，PHP 文件路径已修正为容器内 `/var/www/html`；
  - `proxy`：任意镜像容器 + `proxy_pass`，适合 Node / Go / Java 应用。
- 域名 / 别名校验、冲突检测、通配域名（`*.example.com`）；
- 手动 SSL 证书部署（校验私钥匹配与到期时间），支持强制 HTTPS；
- 网关配置写入后自动执行 `nginx -t`，失败自动回滚，不会因为一个坏配置把全部站点拖垮；
- 站点访问 / 错误日志可在面板内查看；
- 删除站点时可选择保留网站文件，之后可重建同域名站点且不会覆盖已有首页。

### 数据库
- MySQL / PostgreSQL / Redis 一键创建；
- 端口自动分配，**默认只绑定 `127.0.0.1`**；
- Redis 显式启用 `requirepass`；
- 凭据随机生成并保存（`0600`），连接信息、DSN、日志可查。

### 应用商店
- Nginx / OpenResty / MySQL / PostgreSQL / MariaDB / Redis / phpMyAdmin / WordPress / Portainer / MinIO；
- 数据库与工具容器默认只监听本机；数据目录自动处理容器用户权限；
- 凭据仅在安装完成和「凭据」入口中展示。

### 主机与安全
- 文件管理（浏览、编辑、上传、下载、重命名、删除）；
- 资源监控（CPU / 内存 / 磁盘 / 负载 / 网络）；
- 计划任务（cron 表达式、立即执行、输出记录）；
- SSH 状态与 sshd 配置只读检查；
- 防火墙状态只读检查（ufw / firewalld / nftables）；
- 登录审计、安全基线、SMTP 邮件测试；
- 安全入口、面板端口、网关端口、Docker 配置。

### 前端界面（Flarum 2.0 admin SPA 风格）
- Flat / minimalist 视觉：固定 header + 可折叠 drawer + Widget 卡片流；
- **渐进披露导航**：抽屉只放 8 个一级入口，其余全部收进一个**可搜索下拉**（36 个入口，支持 `/` 或 `Ctrl/Cmd + K` 唤起）；
- 自研 BEM-ish 组件：`Button--icon`、`Dropdown--select`、`DashboardWidget`、`StatusDot`、Modal、Tabs、Toast 等；
- a11y：`sr-only` 跳转链接、`aria-*`、键盘导航、焦点陷阱、`aria-live` 提示、减少动画偏好；
- 浅色 / 深色 / 跟随系统主题，标准 / 紧凑密度，移动端适配。

---

## 离线安装（推荐）

发布包不依赖在线安装脚本，上传到服务器任意位置即可。

```bash
# 1. 上传（例如 /opt）
scp upanel-v0.1.7-linux-amd64.tar.gz root@你的服务器:/opt/

# 2. 解压
ssh root@你的服务器
cd /opt
tar -xzf upanel-v0.1.7-linux-amd64.tar.gz
cd upanel-v0.1.7-linux-amd64

# 3. 交互式安装
bash install.sh
```

安装脚本会询问：

| 项目 | 默认值 | 说明 |
| --- | --- | --- |
| 安装目录 | `/opt/upanel` | 面板程序与数据目录 |
| 面板端口 | `8080` | Web 管理界面端口 |
| 管理员用户名 | `admin` | 登录账号 |
| 管理员密码 | 随机生成 | 至少 8 位，需同时包含字母和数字 |
| 安全入口 | 空 | 设置后需通过 `/<入口>` 访问 |
| 网关 HTTP 端口 | `80` | 网站对外 HTTP |
| 网关 HTTPS 端口 | `443` | 网站 HTTPS |

完成后注册 systemd 服务并输出访问地址、账号密码。

### 系统要求
- Linux x86_64 / arm64，root 权限；
- Docker 已安装并运行（安装脚本可协助安装）；
- 80 / 443 未被系统 nginx / apache 占用（或在安装时改端口）；
- 建议使用 Ubuntu 22.04+ / Debian 12+ / Rocky Linux 9+。

### 面板自身使用 HTTPS
UPanel 默认监听 HTTP，建议在前面放一层外部 Nginx / Caddy 并配置 HTTPS，
或只允许通过内网 / SSH 隧道访问面板端口。

---

## 常用命令

```bash
up 6                  # 面板地址、用户名、端口
up 0                  # 服务状态
up 3                  # 重启面板
up 9 password 新密码   # 修改密码（无需重启）
up 9 port 9090        # 修改端口
up 9 entry mypanel    # 设置安全入口
up 9 entry -          # 取消安全入口
up gateway            # 网关状态
up logs               # 面板日志
up 5                  # 卸载
```

如果首次以裸二进制方式启动（没有通过安装脚本设置密码），面板会生成随机密码并写入：

```
<安装目录>/data/initial-password.txt   # 0600
```

请登录后立即修改密码，并删除该文件。

---

## 部署网站的完整流程

1. **域名解析**：A 记录指向服务器公网 IP（通配域名使用 `*.example.com`）。
2. **放行端口**：云安全组 / 防火墙放行 `80`、`443`。
3. **创建站点**：面板「网站 → 创建网站」，选择静态 / PHP / 反代。
4. **上传文件**：站点根目录形如
   `/opt/upanel/data/websites/<域名>/html`，可通过面板文件管理上传，或用 scp / rsync。
5. **访问**：浏览器打开 `http://<域名>`；部署证书后使用 `https://`。

### 常见问题

**访问域名显示 502 / 502 Bad Gateway？**
- PHP / 反代站点容器是否运行；
- 反代站点的「容器内端口」是否与应用实际监听端口一致；
- 查看「网站 → 站点日志 → 错误日志」。

**PHP 站点提示 404 / Primary script unknown？**
- UPanel 已把 `SCRIPT_FILENAME` 固定为 PHP 容器内挂载路径 `/var/www/html`；
- 确认站点根目录存在 `index.php`；
- 如果手动改过 nginx 配置，执行「网关状态 → 修复 / 重载」。

**网关启动失败，端口被占用？**
- 停止宿主机上的 nginx / apache，或在「面板设置 → 网关」中更换端口；
- 云服务器还需同步修改安全组。

**Docker 未连接？**
- `systemctl status docker`；
- 如果面板启动时 Docker 尚未运行，启动 Docker 后执行 `up 3` 重启面板。

**忘记密码？**
```bash
up 9 password 新密码
```

---

## 安全设计

- 不再使用固定默认密码；首次启动生成 16 位随机密码（`data/initial-password.txt`，0600）。
- 升级时如果检测到旧版默认密码（`upanel123` / `admin123`），会强制重置为随机密码并写入上述文件。
- 密码使用 bcrypt 哈希；JWT 使用 HS256 + 随机持久化密钥；
- JWT 中绑定密码哈希指纹与会话版本：
  - 修改密码后旧令牌立即失效；
  - 登出会递增持久化的 `session_version`，重启后旧令牌仍然无效；
- 登录失败限流：同一 IP 10 分钟内 8 次失败后锁定 15 分钟，并写入登录审计；
- 面板与 API 安全响应头：CSP、`X-Frame-Options: DENY`、`X-Content-Type-Options`、`Referrer-Policy`、`Permissions-Policy`；
- 默认无 CORS；请求体大小限制；
- 域名、别名、PHP 版本、镜像名、端口、安全入口、数据目录均有白名单校验；
- 网关配置先 `nginx -t` 再 reload，失败自动回滚；删除最后一个站点后保留 default server；
- 网关容器对站点目录、证书、配置使用只读挂载；日志目录单独可写；
- 数据库与应用容器默认绑定 `127.0.0.1`；Redis 强制密码；
- 文件、数据库、应用、网站元数据与配置均以 `0600` 保存；
- 支持 SELinux 主机（bind mount 自动加 `:z`）。

> 面板以 root 运行，具备服务器最高权限。请务必使用强密码、启用安全入口、限制面板端口来源 IP，
> 并优先通过 HTTPS 反向代理访问。

---

## 目录结构

```
/opt/upanel/
├── bin/
│   ├── upanel               面板主程序
│   └── up                   命令行工具
├── web/                     前端静态资源
├── install.sh
├── uninstall.sh
├── VERSION
└── data/
    ├── config.json          面板配置（0600，含密码哈希）
    ├── settings.json        界面与偏好（0600）
    ├── initial-password.txt 仅裸二进制首次启动时生成
    ├── logs/
    │   ├── auth.log         登录审计
    │   └── ...
    ├── websites/
    │   └── <域名>/
    │       ├── html/        网站根目录
    │       ├── logs/
    │       └── site.json    站点元数据（0600）
    ├── databases/<名称>/
    ├── apps/<名称>/
    ├── cron.json            计划任务（0600）
    ├── email.json           SMTP 配置（0600）
    └── gateway/
        ├── conf.d/          各站点 nginx 配置
        ├── certs/           SSL 证书
        ├── logs/            站点访问 / 错误日志
        ├── html/            默认欢迎页
        └── nginx.conf
```

---

## 从源码构建

需要 Go 1.25+、Node.js 18+。

```bash
# 后端检查与测试
make test

# 前端开发（代理到 localhost:8080）
cd web
npm install
npm run dev

# 前端生产构建
npm run build

# 一键生成 Linux amd64 发布包
bash scripts/build-release.sh v0.1.7
# 产物：release/upanel-v0.1.7-linux-amd64.tar.gz
```

也可以只编译后端：

```bash
make build        # 输出到 bin/
```

> 前端使用相对 base + 服务端注入 `<base href="...">`，因此启用安全入口后资源仍能正确加载。

---

## 已知限制

- SSL 证书目前需要手动粘贴（fullchain + privkey），暂未集成 Let's Encrypt 自动签发；
- SSH 与防火墙页面为只读检查，不提供终端 / 规则修改；
- 单管理员模式；
- 面板自身 HTTPS 需要外部反向代理；
- 站点日志未做自动轮转，长期运行请配合 Docker 日志驱动或系统 logrotate。

---

## License

MIT（如仓库未附带 LICENSE 文件，以项目实际声明为准）。
