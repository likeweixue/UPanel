#!/bin/bash
# ============================================================================
#  UPanel 离线安装脚本
#  用法: 解压发布包后执行  bash install.sh
# ============================================================================

set -e

# ---------- 颜色 ----------
if [ -t 1 ]; then
    GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'
    CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
else
    GREEN=''; YELLOW=''; RED=''; CYAN=''; BOLD=''; NC=''
fi

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }
title() { echo -e "${CYAN}$1${NC}"; }

# ---------- 基础信息 ----------
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PANEL_NAME="upanel"
SERVICE_NAME="upanel"

VERSION="unknown"
if [ -f "$SELF_DIR/VERSION" ]; then
    VERSION="$(cat "$SELF_DIR/VERSION")"
fi

print_banner() {
    echo ""
    title "╔══════════════════════════════════════════════════════════╗"
    title "║                                                          ║"
    title "║   ██╗   ██╗██████╗  █████╗ ███╗   ██╗███████╗██╗        ║"
    title "║   ██║   ██║██╔══██╗██╔══██╗████╗  ██║██╔════╝██║        ║"
    title "║   ██║   ██║██████╔╝███████║██╔██╗ ██║█████╗  ██║        ║"
    title "║   ██║   ██║██╔═══╝ ██╔══██║██║╚██╗██║██╔══╝  ██║        ║"
    title "║   ╚██████╔╝██║     ██║  ██║██║ ╚████║███████╗███████╗   ║"
    title "║    ╚═════╝ ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝   ║"
    title "║                                                          ║"
    title "║          轻量级 Docker 容器管理面板 · 离线安装           ║"
    title "╚══════════════════════════════════════════════════════════╝"
    echo ""
    echo -e "  版本: ${BOLD}${VERSION}${NC}"
    echo ""
}

# ---------- 环境检测 ----------
check_os() {
    if [ "$(uname -s)" != "Linux" ]; then
        error "UPanel 服务端仅支持 Linux，当前系统: $(uname -s)"
        exit 1
    fi

    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)   ARCH_LABEL="amd64" ;;
        aarch64|arm64)  ARCH_LABEL="arm64" ;;
        *)
            warn "未验证的架构: $ARCH，继续尝试安装"
            ARCH_LABEL="$ARCH"
            ;;
    esac
    info "系统架构: $ARCH_LABEL"

    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_NAME="$PRETTY_NAME"
    else
        OS_NAME="Linux"
    fi
    info "操作系统: $OS_NAME"
}

check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        error "请使用 root 用户执行安装（或 sudo bash install.sh）"
        exit 1
    fi
}

check_package() {
    if [ ! -d "$SELF_DIR/bin" ] || [ ! -f "$SELF_DIR/bin/upanel" ]; then
        error "发布包不完整：缺少 bin/upanel"
        error "请确认已完整解压，且当前目录为解压后的目录"
        exit 1
    fi
    if [ ! -d "$SELF_DIR/web" ]; then
        warn "未找到 web 目录，面板界面可能无法访问"
    fi
}

# ---------- 交互输入 ----------
read_with_default() {
    local prompt="$1" default="$2" var_name="$3" input
    read -r -p "$(echo -e "${CYAN}${prompt}${NC} [${default}]: ")" input
    if [ -z "$input" ]; then input="$default"; fi
    eval "$var_name=\"\$input\""
}

read_password() {
    local prompt="$1" var_name="$2" pw1 pw2
    while true; do
        read -r -s -p "$(echo -e "${CYAN}${prompt}${NC}: ")" pw1
        echo ""
        if [ -z "$pw1" ]; then
            pw1="$(head -c 9 /dev/urandom | base64 | tr -d '/+=' | head -c 12)"
            info "已生成随机密码: $pw1"
            eval "$var_name=\"\$pw1\""
            return
        fi
        if [ ${#pw1} -lt 6 ]; then
            warn "密码长度不能少于 6 位"
            continue
        fi
        read -r -s -p "$(echo -e "${CYAN}请再次输入密码${NC}: ")" pw2
        echo ""
        if [ "$pw1" != "$pw2" ]; then
            warn "两次输入不一致，请重试"
            continue
        fi
        eval "$var_name=\"\$pw1\""
        return
    done
}

port_in_use() {
    local port="$1"
    if command -v ss >/dev/null 2>&1; then
        ss -ltn 2>/dev/null | grep -q ":$port " && return 0
    fi
    if command -v netstat >/dev/null 2>&1; then
        netstat -ltn 2>/dev/null | grep -q ":$port " && return 0
    fi
    if command -v lsof >/dev/null 2>&1; then
        lsof -i ":$port" >/dev/null 2>&1 && return 0
    fi
    return 1
}

check_port() {
    local port="$1" label="$2"
    if port_in_use "$port"; then
        warn "$label 端口 $port 已被占用"
        return 1
    fi
    return 0
}

# ---------- Docker ----------
docker_status() {
    if ! command -v docker >/dev/null 2>&1; then
        echo "missing"
        return
    fi
    if docker info >/dev/null 2>&1; then
        echo "running"
    else
        echo "stopped"
    fi
}

install_docker() {
    info "尝试安装 Docker ..."
    local script
    script="$(mktemp)"
    if ! curl -fsSL --connect-timeout 15 -o "$script" https://get.docker.com; then
        warn "官方脚本下载失败，改用国内镜像"
        if ! curl -fsSL --connect-timeout 15 -o "$script" https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo; then
            error "Docker 安装脚本下载失败，请手动安装 Docker 后重新运行本脚本"
            rm -f "$script"
            return 1
        fi
    fi
    sh "$script" || {
        error "Docker 安装失败，请手动安装后重新运行本脚本"
        rm -f "$script"
        return 1
    }
    rm -f "$script"

    if command -v systemctl >/dev/null 2>&1; then
        systemctl enable docker >/dev/null 2>&1 || true
        systemctl start docker >/dev/null 2>&1 || true
    fi
    info "Docker 安装完成"
}

setup_docker() {
    local st
    st="$(docker_status)"
    case "$st" in
        running)
            info "Docker 已安装且正在运行 ✓"
            ;;
        stopped)
            warn "Docker 已安装但未运行，尝试启动 ..."
            systemctl start docker >/dev/null 2>&1 || true
            sleep 2
            if [ "$(docker_status)" = "running" ]; then
                info "Docker 已启动 ✓"
            else
                error "Docker 启动失败，请检查 docker 服务"
            fi
            ;;
        missing)
            warn "未检测到 Docker"
            read -r -p "$(echo -e "${CYAN}是否现在安装 Docker? (y/N): ${NC}")" yn
            case "$yn" in
                [yY]*) install_docker || true ;;
                *)
                    warn "跳过 Docker 安装。注意：未安装 Docker 将无法使用网站/数据库/应用功能"
                    ;;
            esac
            ;;
    esac
}

# ---------- 收集安装参数 ----------
collect_options() {
    echo ""
    title "──────────────── 安装配置 ────────────────"
    echo ""

    read_with_default "安装目录" "/opt/upanel" INSTALL_DIR
    # 去掉结尾斜杠
    INSTALL_DIR="${INSTALL_DIR%/}"
    [ -z "$INSTALL_DIR" ] && INSTALL_DIR="/opt/upanel"

    read_with_default "面板端口" "8080" PANEL_PORT
    while ! [[ "$PANEL_PORT" =~ ^[0-9]+$ ]] || [ "$PANEL_PORT" -lt 1 ] || [ "$PANEL_PORT" -gt 65535 ]; do
        warn "端口必须为 1-65535 之间的数字"
        read_with_default "面板端口" "8080" PANEL_PORT
    done
    check_port "$PANEL_PORT" "面板" || true

    read_with_default "管理员用户名" "admin" PANEL_USER
    [ -z "$PANEL_USER" ] && PANEL_USER="admin"

    read_password "管理员密码（留空随机生成）" PANEL_PASS

    read -r -p "$(echo -e "${CYAN}安全入口（留空表示不启用，如设为 abc 则通过 /abc 访问）${NC}: ")" PANEL_ENTRY
    PANEL_ENTRY="$(echo "$PANEL_ENTRY" | tr -d ' ' | sed 's#^/##; s#/$##')"

    echo ""
    title "──────────────── 网站网关 ────────────────"
    echo ""
    info "网站通过统一网关容器对外提供 80/443 服务"
    echo ""

    read_with_default "网关 HTTP 端口" "80" GW_HTTP
    read_with_default "网关 HTTPS 端口" "443" GW_HTTPS

    if port_in_use "$GW_HTTP"; then
        warn "端口 $GW_HTTP 已被占用，网关可能无法启动"
        warn "常见原因：系统已安装 nginx/apache。可更换端口或停止占用服务"
        read -r -p "$(echo -e "${CYAN}仍要使用 $GW_HTTP ? (y/N): ${NC}")" yn
        case "$yn" in
            [yY]*) ;;
            *) read_with_default "改用端口" "8088" GW_HTTP ;;
        esac
    fi

    echo ""
    title "──────────────── 配置确认 ────────────────"
    echo ""
    echo -e "  安装目录:   ${BOLD}$INSTALL_DIR${NC}"
    echo -e "  面板端口:   ${BOLD}$PANEL_PORT${NC}"
    echo -e "  用户名:     ${BOLD}$PANEL_USER${NC}"
    echo -e "  密码:       ${BOLD}${PANEL_PASS}${NC}"
    echo -e "  安全入口:   ${BOLD}${PANEL_ENTRY:-（未启用）}${NC}"
    echo -e "  网关端口:   ${BOLD}$GW_HTTP / $GW_HTTPS${NC}"
    echo ""
    read -r -p "$(echo -e "${CYAN}确认开始安装? (Y/n): ${NC}")" yn
    case "$yn" in
        [nN]*) info "已取消安装"; exit 0 ;;
    esac
}

# ---------- 安装 ----------
json_escape() {
    printf '%s' "$1" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read())[1:-1])' 2>/dev/null \
        || printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

deploy_files() {
    info "部署文件到 $INSTALL_DIR ..."

    mkdir -p "$INSTALL_DIR"

    # 二进制
    mkdir -p "$INSTALL_DIR/bin"
    install -m 0755 "$SELF_DIR/bin/upanel" "$INSTALL_DIR/bin/upanel"
    if [ -f "$SELF_DIR/bin/up" ]; then
        install -m 0755 "$SELF_DIR/bin/up" "$INSTALL_DIR/bin/up"
    fi

    # 前端
    if [ -d "$SELF_DIR/web" ]; then
        rm -rf "$INSTALL_DIR/web"
        cp -r "$SELF_DIR/web" "$INSTALL_DIR/web"
    fi

    # 脚本与文档
    for f in uninstall.sh README.md VERSION LICENSE; do
        [ -f "$SELF_DIR/$f" ] && install -m 0644 "$SELF_DIR/$f" "$INSTALL_DIR/$f"
    done
    if [ -f "$SELF_DIR/install.sh" ]; then
        install -m 0755 "$SELF_DIR/install.sh" "$INSTALL_DIR/install.sh"
    fi

    mkdir -p "$INSTALL_DIR/data"
    chmod 0755 "$INSTALL_DIR/data"
}

write_config() {
    info "写入配置文件 ..."
    mkdir -p "$INSTALL_DIR/data"

    local esc_user esc_entry
    esc_user="$(json_escape "$PANEL_USER")"
    esc_entry="$(json_escape "$PANEL_ENTRY")"

    cat > "$INSTALL_DIR/data/config.json" <<EOF
{
  "port": "$PANEL_PORT",
  "bind_ip": "0.0.0.0",
  "mode": "release",
  "base_dir": "$INSTALL_DIR",
  "username": "$esc_user",
  "jwt_secret": "$(head -c 24 /dev/urandom | base64 | tr -d '/+=')",
  "password_hash": "",
  "security_entry": "$esc_entry",
  "data_path": "data",
  "website_path": "data/websites",
  "database_path": "data/databases",
  "app_path": "data/apps",
  "static_path": "web",
  "gateway_image": "nginx:alpine",
  "gateway_http_port": "$GW_HTTP",
  "gateway_https_port": "$GW_HTTPS",
  "php_image": "php:8.2-fpm",
  "registry_mirrors": []
}
EOF
    chmod 0600 "$INSTALL_DIR/data/config.json"

    # 用 up 命令写入 bcrypt 密码哈希
    if [ -x "$INSTALL_DIR/bin/up" ]; then
        UPANEL_BASE_DIR="$INSTALL_DIR" "$INSTALL_DIR/bin/up" 9 password "$PANEL_PASS" >/dev/null 2>&1 \
            || warn "密码初始化失败，可在安装后执行: $INSTALL_DIR/bin/up 9 password 新密码"
    fi
}

install_service() {
    if ! command -v systemctl >/dev/null 2>&1; then
        warn "未检测到 systemd，跳过服务注册"
        return 1
    fi

    info "注册 systemd 服务 ..."
    cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=UPanel - Lightweight Docker Management Panel
Documentation=https://github.com/upanel/upanel
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/bin/upanel
Restart=always
RestartSec=5
LimitNOFILE=65535
Environment="UPANEL_BASE_DIR=$INSTALL_DIR"
Environment="GIN_MODE=release"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME" >/dev/null 2>&1 || true
    systemctl restart "$SERVICE_NAME"
    return 0
}

install_cli() {
    if [ -x "$INSTALL_DIR/bin/up" ]; then
        ln -sf "$INSTALL_DIR/bin/up" /usr/local/bin/up 2>/dev/null || true
    fi
    if [ -x "$INSTALL_DIR/bin/upanel" ]; then
        ln -sf "$INSTALL_DIR/bin/upanel" /usr/local/bin/upanel 2>/dev/null || true
    fi
}

open_firewall() {
    local port="$1"
    if command -v firewall-cmd >/dev/null 2>&1; then
        if systemctl is-active --quiet firewalld 2>/dev/null; then
            firewall-cmd --permanent --add-port="${port}/tcp" >/dev/null 2>&1 && \
                firewall-cmd --reload >/dev/null 2>&1 && \
                info "已放行防火墙端口 $port/tcp (firewalld)"
        fi
    elif command -v ufw >/dev/null 2>&1; then
        if ufw status 2>/dev/null | grep -qi "active"; then
            ufw allow "${port}/tcp" >/dev/null 2>&1 && info "已放行防火墙端口 $port/tcp (ufw)"
        fi
    fi
}

wait_for_panel() {
    local port="$1" i
    for i in $(seq 1 30); do
        if curl -fsS "http://127.0.0.1:${port}/api/panel/info" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    return 1
}

get_ip() {
    local ip
    ip="$(ip -4 addr show scope global 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -n1)"
    if [ -z "$ip" ]; then
        ip="$(curl -fsS --connect-timeout 5 ifconfig.me 2>/dev/null || true)"
    fi
    echo "${ip:-服务器IP}"
}

print_summary() {
    local entry_path=""
    [ -n "$PANEL_ENTRY" ] && entry_path="/$PANEL_ENTRY"
    local ip
    ip="$(get_ip)"

    echo ""
    title "╔══════════════════════════════════════════════════════════╗"
    title "║                    🎉  安装完成                           ║"
    title "╚══════════════════════════════════════════════════════════╝"
    echo ""
    echo -e "  ${BOLD}面板地址:${NC}   http://${ip}:${PANEL_PORT}${entry_path}/"
    echo -e "  ${BOLD}用户名:${NC}     $PANEL_USER"
    echo -e "  ${BOLD}密码:${NC}       $PANEL_PASS"
    echo -e "  ${BOLD}安装目录:${NC}   $INSTALL_DIR"
    echo ""
    echo -e "  ${BOLD}常用命令:${NC}"
    echo -e "    up 6          查看面板地址与账号"
    echo -e "    up 9 password 修改面板密码"
    echo -e "    up 3          重启面板"
    echo -e "    systemctl status $SERVICE_NAME"
    echo ""
    echo -e "  ${BOLD}网站部署提示:${NC}"
    echo -e "    1. 面板「网站」页创建站点，域名需已解析到本机"
    echo -e "    2. 确保网关端口 ${GW_HTTP}/${GW_HTTPS} 已在防火墙/安全组放行"
    echo -e "    3. 站点文件位于 $INSTALL_DIR/data/websites/<域名>/html"
    echo ""
    warn "请妥善保存上面显示的密码"
    echo ""
}

# ---------- 主流程 ----------
main() {
    print_banner
    check_os
    check_root
    check_package
    setup_docker
    collect_options

    echo ""
    title "──────────────── 开始安装 ────────────────"

    # 已存在则先停止旧服务
    if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        info "停止已运行的面板服务 ..."
        systemctl stop "$SERVICE_NAME" >/dev/null 2>&1 || true
    fi

    deploy_files
    write_config

    if install_service; then
        open_firewall "$PANEL_PORT"
        open_firewall "$GW_HTTP"
        echo ""
        info "等待面板启动 ..."
        if wait_for_panel "$PANEL_PORT"; then
            info "面板已就绪 ✓"
        else
            warn "面板未能自动响应，请检查: journalctl -u $SERVICE_NAME -n 50"
        fi
    else
        warn "未注册 systemd 服务，可手动启动："
        warn "  UPANEL_BASE_DIR=$INSTALL_DIR nohup $INSTALL_DIR/bin/upanel > $INSTALL_DIR/data/upanel.log 2>&1 &"
    fi

    install_cli
    print_summary
}

main "$@"
