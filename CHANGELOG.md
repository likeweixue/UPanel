# 更新日志 (Changelog)

本文件记录 UPanel 各版本的变更内容。版本号遵循 `v主.次.修订` 格式，LTS 版本会在末尾标注 `-lts`。

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
