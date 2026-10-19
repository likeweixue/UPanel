# UPanel

轻量级 Docker 容器管理面板 —— 网站部署、数据库、应用商店、文件管理，全部通过 Docker 容器完成。

## 特性

- **网站部署**：统一网关（Nginx）占用 80/443，按域名自动反代到各站点
  - 静态站点：网关直接托管文件，零额外容器
  - PHP 站点：php-fpm 容器 + fastcgi 转发
  - 反代站点：任意镜像容器 + `proxy_pass`，适合 Node / Go / Java 应用
  - 支持别名域名、HTTPS 证书部署、强制跳转 HTTPS
- **数据库**：MySQL / PostgreSQL / Redis 一键创建，端口自动分配，连接信息可查
- **应用商店**：Nginx、MySQL、Redis、WordPress、Portainer、MinIO 等
- **容器管理**：启动 / 停止 / 重启 / 删除 / 日志
- **文件管理**：在线浏览、编辑、上传、重命名、删除
- **面板设置**：网关端口、镜像加速、安全入口、修改密码

## 离线安装（推荐）

不需要云服务器托管安装脚本，把发布包上传到服务器任意位置即可。

```bash
# 1. 上传发布包到服务器，例如 /opt
scp upanel-v0.1.6-lts-linux-amd64.tar.gz root@你的服务器:/opt/

# 2. 登录服务器，解压
ssh root@你的服务器
cd /opt
tar -xzf upanel-v0.1.6-lts-linux-amd64.tar.gz
cd upanel-v0.1.6-lts-linux-amd64

# 3. 执行安装脚本，按提示完成
bash install.sh
```

安装脚本会依次询问：

| 项目 | 默认值 | 说明 |
| --- | --- | --- |
| 安装目录 | `/opt/upanel` | 面板程序与数据存放位置 |
| 面板端口 | `8080` | Web 管理界面端口 |
| 管理员用户名 | `admin` | 登录账号 |
| 管理员密码 | 随机生成 | 留空自动生成，务必保存 |
| 安全入口 | 空 | 设置后需通过 `/入口路径` 访问 |
| 网关 HTTP 端口 | `80` | 网站对外服务端口 |
| 网关 HTTPS 端口 | `443` | 网站 HTTPS 端口 |

安装完成后会输出访问地址、账号密码，并自动注册 systemd 服务。

### 系统要求

- Linux（x86_64 / arm64），需要 root 权限
- Docker 已安装且正在运行（安装脚本可帮你安装）
- 80 / 443 端口未被占用（如已装 nginx / apache 请先停止或更换网关端口）

## 常用命令

安装后会自动创建 `up` 命令：

```bash
up 6               # 查看面板地址、用户名、端口
up 9 password      # 修改面板密码
up 9 port 9090     # 修改面板端口
up 3               # 重启面板
up 8               # 查看版本
up gateway         # 查看网站网关状态
up logs            # 查看面板日志
up 5               # 卸载面板

systemctl status upanel    # 查看服务状态
```

## 部署网站的完整流程

1. 域名解析：把域名的 A 记录指向服务器公网 IP
2. 防火墙：`80`、`443`、面板端口需在安全组 / 防火墙中放行
3. 面板创建站点：「网站」→「创建网站」，选择类型并填写域名
4. 上传代码：站点根目录显示在列表中，形如
   `/opt/upanel/data/websites/<域名>/html`
   可通过面板「文件管理」直接上传，或用 scp / rsync
5. 访问：点击列表中的「访问」按钮，或浏览器打开 `http://<域名>`

> 提示：若网关状态显示异常，点击页面上方的「修复网关」，面板会重新拉取镜像并启动网关容器。

## 目录结构

```
/opt/upanel/
├── bin/
│   ├── upanel        面板主程序
│   └── up            命令行工具
├── web/              前端静态资源
├── install.sh
├── uninstall.sh
└── data/
    ├── config.json   面板配置（含密码哈希）
    ├── settings.json 偏好设置
    ├── websites/     站点目录
    │   └── <域名>/
    │       ├── html/      网站根目录
    │       ├── logs/
    │       └── site.json  站点元数据
    ├── databases/
    ├── apps/
    └── gateway/
        ├── conf.d/   各站点的 nginx 配置
        ├── certs/    SSL 证书
        └── nginx.conf
```

## 从源码构建发布包

```bash
# 需要 Go 1.21+ 与 Node.js 18+
bash scripts/build-release.sh v0.1.6-lts
```

产物位于 `release/upanel-<版本>-linux-amd64.tar.gz`。

仅编译后端（开发调试）：

```bash
make build          # 编译到 bin/
make build-all      # 编译多平台到 release/
```

## 卸载

```bash
bash /opt/upanel/uninstall.sh
# 或
up 5
```

卸载时会询问是否保留 `data/` 目录（网站文件与数据库数据）。

## 常见问题

**Q: 创建网站后访问显示 502？**
A: 检查站点容器是否运行（PHP / 反代类型），或网关是否正常。可在「网站」页点击「修复网关」。

**Q: 网关启动失败，提示端口被占用？**
A: 服务器上已有 nginx / apache 占用 80。停止它们（`systemctl stop nginx`），或在「面板设置」中把网关端口改为其它值（如 8088），再配合外部 Nginx 反代。

**Q: 拉取镜像很慢或失败？**
A: 在「面板设置」中配置 Docker 镜像加速地址，或直接修改 `/etc/docker/daemon.json` 后重启 Docker。

**Q: 忘记面板密码？**
A: 在服务器执行 `up 9 password 新密码`，无需重启立即生效。
