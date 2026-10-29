# 更新日志 (Changelog)

本文件记录 UPanel 各版本的变更内容。版本号遵循 `v主.次.修订` 格式，LTS 版本会在末尾标注 `-lts`。

---

## 未发布

### 🎨 可插拔主题

- 新增 `themes/` 文件式主题系统：每个主题是一个文件夹，包含可选的 `theme.json` 和必需的 `theme.css`，无需重新编译。
- 主题元数据支持名称、描述、版本、作者、亮/暗标记、`themeColor`、默认圆角与主题卡配色。
- 主题 CSS 通过 `html[data-theme="<id>"]` 覆盖 UPanel 的设计 token；支持主题内 `assets/` 静态资源。
- `/api/themes`、`/api/themes/:id/theme.css`、`/api/themes/:id/assets/*` 公开接口；服务端自动把主题 CSS 注入入口页，登录页也能加载。
- 设置 → 外观会列出 `themes/` 中发现的自定义主题；点击「重新加载」可识别新放入的主题文件夹。
- 新增 **Flarum** 主题：4px 圆角、36px 控件、system-ui 字体栈、灰蓝主色、填充式控件；不改变 UPanel 布局。
- 发布包和安装脚本会携带 `themes/`，安装时保留用户自己添加的主题。

---

## v0.1.8 (2026-10-25)

本版本集中优化界面结构、主题、底部栏、侧边栏配置与登录/表单交互。

### 🎨 UI 细节优化
- 页面标题、操作按钮、标签页和内容收进同一个 `page-shell` 面板；标签页改为卡片顶部的分段式导航，页面更整齐。
- 主内容与侧边栏的间距缩小一半，面板更贴近侧边栏。
- 新增「设置 → 外观 → 界面圆角」：0-16px 滑块与预设，默认 4px，统一影响卡片、按钮、输入框、标签页和弹窗。
- 新增「设置 → 外观 → 底部栏」：可开关底部栏、自定义左侧文字、显示/隐藏版本号与 GitHub 链接；底部栏固定在视口底部。
- 侧边栏底部新增「收起/展开侧边栏」与「菜单设置」按钮；菜单设置支持显示/隐藏和上下排序，配置保存在当前浏览器。
- 桌面端支持侧边栏折叠为 64px 图标栏，移动端按钮改为关闭抽屉。
- 新增右侧抽屉式表单：新建/编辑网站、SSL、创建数据库、安装应用、新建/重命名/编辑文件、上传文件、计划任务、修改密码；点击遮罩或 Esc 关闭，保留焦点恢复与滚动锁定。
- 新 Logo：渐变圆角方块内嵌 `UP` 字母，顶栏、favicon 同步更新。
- 登录页精简：移除登录页 Logo、底部版本/Docker 文案、登录安全提示、品牌下方分隔线、简介副标题；标题简化为居中的「登录」；用户名/密码改为输入框内提示，移除外部标签；输入框改为灰色填充区域，聚焦时用主题色区域区分，不再依赖边框。
- 新增内置主题：复古黄（暖黄护眼）、护眼纸（绿色纸张）、黑金、暗夜橙红；主题色同时适配 Logo、按钮和状态色。
- 新增 GitHub Actions release 工作流：推送 `v*` tag 自动构建前端/后端并发布 GitHub Release。

---

## v0.1.7 (2026-10-24)

本版本集中在「网站真正能跑通」、安全加固与后台 UI 重构。

### 🚀 网站部署修复（关键）
- 修复 **Docker bind mount 文件被无条件 `MkdirAll` 导致网关创建失败**：`nginx.conf` / `docker.sock` 等文件挂载不再被当成目录创建。
- 修复 **PHP `SCRIPT_FILENAME` 指向网关路径**：php-fpm 在独立容器中，现固定为 `/var/www/html$fastcgi_script_name`，并设置 `DOCUMENT_ROOT`。
- 修复 Docker 网络 `EndpointSettings.NetworkID` 使用网络名的问题，改为真实网络 ID。
- 网关容器站点目录 / 证书 / 配置改为只读挂载，日志目录单独可写；主配置原地写入以保留 single-file bind mount 的 inode。
- 网关写入配置前备份、`nginx -t` 失败自动回滚；重载失败不再用「重启容器」兜底，避免坏配置导致全部站点宕机。
- 修复强制 HTTPS 时 80 端口出现重复 server 块、HTTPS 丢失安全响应头、静态站点把 `index.php` 当默认索引导致源码泄露等问题。
- 删除站点但保留文件后，可以用同域名重建，且不会覆盖已有首页。
- 修复 PHP 版本切换时同名容器直接复用、版本不生效的问题，并支持失败回滚。
- 站点、数据库、应用创建失败时不再删除已存在的数据目录（保留数据重建场景）。
- 处理 PHP / Redis / MySQL / PostgreSQL / MinIO / WordPress 等容器用户对宿主机数据目录的写权限。
- 支持 SELinux 主机（bind mount `:z`）。

### 🔐 安全加固
- 移除固定默认密码 `upanel123`：首次裸启动生成 16 位随机密码到 `data/initial-password.txt`（0600），安装脚本也不允许弱密码；升级时检测到旧版默认密码会强制重置。
- 移除固定 JWT 兜底密钥；JWT 密钥自动生成并持久化，长度不足 32 时重新生成。
- JWT 绑定密码哈希指纹与会话版本：修改密码 / 登出立即使旧令牌失效，且登出状态在重启后仍然有效。
- 登录失败限流（同 IP 8 次 / 10 分钟，锁定 15 分钟）+ 登录审计。
- 安全响应头（CSP、XFO、nosniff、Referrer-Policy、Permissions-Policy），移除通配 CORS。
- 安全入口完整生效：API 也挂到入口下，未带入口的路径 404，前端资源通过注入 `<base>` 正常加载；保留路径（api/assets）被拒绝。
- 网站域名、别名、PHP 版本、镜像、端口、站点类型、数据库名/版本、应用名/版本、数据目录、面板端口/监听地址全部校验。
- 数据库与应用端口默认只绑定 `127.0.0.1`；Redis 通过 `--requirepass` 启用认证；随机凭据。
- 文件上传大小限制、文件名清洗、下载需要 Bearer Token；阻止路径穿越与隐藏文件 / 备份文件暴露。
- 请求体大小限制、HTTP server 超时、`SetTrustedProxies(nil)`、优雅退出。

### 🎨 UI 重构：Flarum 2.0 admin SPA 风格
- 全新 flat / minimalist 视觉：固定 header + drawer + Widget 卡片流；
- 渐进披露导航：抽屉 8 个一级入口，全部 **36 个入口**收进一个可搜索下拉（`/` 或 `Ctrl/Cmd+K`）；
- 自研 BEM-ish 组件库：`Button--icon`、`Dropdown--select`、`DashboardWidget`、`StatusDot`、Modal、Tabs、Toast、Sparkline 等；
- 浅色 / 深色 / 跟随系统主题，标准 / 紧凑密度；
- a11y：`sr-only` skip link、`aria-*`、键盘导航、焦点陷阱与焦点恢复、`aria-live`、减少动画偏好；
- 全部页面重写：总览、网站、数据库、应用商店、容器、文件、监控、安全、计划任务、SSH、邮件、设置、登录；
- 桌面 / 平板 / 手机适配；Docker 未连接时网站、数据库、应用列表仍可只读查看。

### ✨ 新增功能
- 计划任务（cron）：创建 / 编辑 / 启用 / 立即执行 / 输出记录（文件 `data/cron.json`，0600）；
- SMTP 邮件配置与测试发送（`data/email.json`，0600，GET 不返回密码）；
- 登录审计、安全基线、SSH / 防火墙只读状态；
- 容器镜像 / 数据卷 / 网络列表、站点日志读取、文件下载、应用凭据查询；
- Docker daemon.json 合并保存（保留未知字段并备份），日志驱动 / 镜像加速 / 并发下载配置。

### 🧪 测试
- 新增网站配置、域名 / 别名注入防护、ForceSSL、PHP SCRIPT_FILENAME、静态站点隐藏文件、通配域名 Slug、cron 解析、Redis requirepass、文件 tail、时长格式化、安全入口等单元测试；
- `go build ./...`、`go vet ./...`、`go test ./...` 全部通过；前端 `vite build` 通过。

### ⚠️ 已知限制
- 暂未集成 Let's Encrypt 自动签发；
- SSH / 防火墙为只读检查；
- 单管理员模式，面板自身 HTTPS 需外部反向代理；
- 站点日志未自动轮转。

---

## v0.1.6-lts (2026-10-19)

本次版本是对「网站部署无法正常工作」这一核心问题的彻底重构，同时补齐了离线分发能力。

### 🚀 核心修复：网站部署终于能跑通了

此前版本存在 4 个致命问题，导致网站根本无法正常部署：

| 问题 | 旧表现 | 修复方式 |
| --- | --- | --- |
| 硬编码 macOS 路径 | 数据目录写死为 `/Users/.../Upanel/www`，在 Linux 服务器上直接失效 | 所有路径改为由配置 `base_dir` 推导，支持 `UPANEL_BASE_DIR` 环境变量 |
| 每站点随机端口 | compose 中 `ports: - "80"` 让 Docker 随机分配宿主机端口，多站点冲突且无法用域名访问 | 改为**统一网关**架构（见下） |
| 依赖 `docker compose` | 老版本 Docker 仅有 `docker-compose`，调用直接失败 | 改用 Docker SDK 直连，摆脱 compose 依赖 |
| 包级变量覆盖配置 | `middleware` 包初始化早于 `main()` 加载配置，把安装时设的端口重置为 8080 | 改为 `sync.Once` 懒加载，杜绝配置被覆盖 |

### 🌐 新架构：统一网关反向代理

```
域名:80/443 → upanel-gateway (nginx) → 按域名转发
                 ├─ static: 网关直接托管文件，零额外容器
                 ├─ php:    → php-fpm 容器 (fastcgi)
                 └─ proxy:  → 任意镜像容器 (proxy_pass)
```

- 站点容器只加入 `upanel_network`，**不再映射宿主机端口**，彻底消灭端口冲突。
- 关键细节：网关主配置写入 `resolver 127.0.0.11 valid=30s;`，后端地址用变量形式，避免后端容器未启动时 nginx 因 `host not found in upstream` 加载失败。
- 新增 `EnsureGateway` 能力：首次部署或网关异常时自动拉起/修复网关容器。

### 📦 离线安装包（无需云服务器/在线脚本）

针对没有预算购买云服务器托管在线安装脚本的场景，改为**离线压缩包**分发：

```bash
# 手动上传到服务器任意位置，例如 /opt
scp upanel-v0.1.6-lts-linux-amd64.tar.gz root@服务器:/opt/
cd /opt && tar -xzf upanel-v0.1.6-lts-linux-amd64.tar.gz
cd upanel-v0.1.6-lts-linux-amd64 && bash install.sh
```

- 新增 `scripts/install.sh`：交互式询问安装目录（默认 `/opt/upanel`）、面板端口、用户名、密码（留空随机生成）、安全入口、网关端口（80/443），随后注册 systemd 服务并输出访问地址；检测到 Docker 未安装会提示协助安装。
- 新增 `scripts/uninstall.sh`：安全卸载（停止服务、可选清理容器与数据）。
- 新增 `scripts/build-release.sh`：一键产出 `upanel-v0.1.6-lts-linux-amd64.tar.gz`，内含编译好的二进制、前端 `web/` 与安装脚本。

### 🔐 认证与配置持久化

- 登录凭据（bcrypt 哈希）真正落盘到 `data/config.json`，重启后依然有效。
- `up` 命令行工具（`9` 子命令）可真正读写配置：设置/重置密码、查看状态、启动/停止面板。
- 设置项对外暴露并持久化（面板端口、网关端口、安全入口等）。

### 🧩 数据库 / 应用服务

- 数据库服务修复：固定端口映射、凭据持久化回读、新增详情与日志接口。
- 应用安装服务支持端口分配与状态查询。
- 当 Docker 不可用时，相关接口返回**明确错误提示**，不再出现难以理解的 `API not found` 404。

### 🧪 测试与质量

- 新增 `internal/service/website_test.go`：覆盖域名合法性校验、三种站点类型（static/php/proxy）的 nginx 配置渲染、SSL 配置、网关 `resolver` 配置。
- `go build` + `go vet` 全量通过。

### ⚠️ 已知限制（后续版本计划）

- 前端 `Databases.vue` / `Containers.vue` 仍为占位/mock，后端 API 已就绪，待接入真实数据。
- 暂未实现 Let's Encrypt 自动签发，当前 SSL 仅支持手动粘贴证书。

---

## v0.1.5 (历史版本)

达芬奇主题优化 + 主题系统精简。（详细提交见 Git 历史）
