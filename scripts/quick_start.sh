#!/bin/bash

# --- 配置变量 ---
WORK_DIR="/opt"
PANEL_DIR="${WORK_DIR}/upanel"
SERVICE_NAME="upanel"
RANDOM_PASS=$(tr -dc A-Za-z0-9 </dev/urandom | head -c 12 ; echo '')
RANDOM_PORT=$((RANDOM % 40000 + 10000))
RANDOM_ENTRY=$(tr -dc a-z0-9 </dev/urandom | head -c 8 ; echo '')

# --- 颜色输出 ---
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
print_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# --- 检测操作系统和架构 ---
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        VER=$VERSION_ID
    else
        print_error "无法检测操作系统类型。"
        exit 1
    fi
    ARCH=$(uname -m)
    print_info "检测到系统: ${OS} ${VER}, 架构: ${ARCH}"
}

# --- 安装 Docker (如果需要) ---
install_docker() {
    if command -v docker &> /dev/null; then
        print_info "Docker 已安装，跳过。"
        return
    fi
    print_warn "未检测到 Docker，是否安装？[y/N]"
    read -r install_docker_choice
    if [[ "$install_docker_choice" =~ ^[Yy]$ ]]; then
        print_info "正在安装 Docker..."
        curl -fsSL https://get.docker.com | bash
        systemctl enable --now docker
        print_info "Docker 安装完成。"
    else
        print_error "UPanel 需要 Docker 环境，安装已终止。"
        exit 1
    fi
}

# --- 下载并解压 UPanel ---
download_upanel() {
    local release_url="https://github.com/likeweixue/UPanel/releases/latest/download/upanel-1.0.0-linux-amd64.tar.gz"
    local tmp_file="/tmp/upanel-latest.tar.gz"
    
    print_info "正在下载最新版 UPanel..."
    if ! curl -L -o "$tmp_file" "$release_url"; then
        print_error "下载失败，请检查网络连接。"
        exit 1
    fi
    
    print_info "解压到 ${PANEL_DIR}..."
    mkdir -p "$PANEL_DIR"
    tar -xzf "$tmp_file" -C "$WORK_DIR"
    rm -f "$tmp_file"
    
    if [[ ! -f "${PANEL_DIR}/bin/upanel" ]]; then
        print_error "解压后未找到可执行文件，安装失败。"
        exit 1
    fi
}

# --- 配置 systemd 服务 ---
configure_service() {
    print_info "配置 systemd 服务..."
    cat > "/etc/systemd/system/${SERVICE_NAME}.service" << EOF
[Unit]
Description=UPanel Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${PANEL_DIR}
ExecStart=${PANEL_DIR}/bin/upanel
Restart=always
RestartSec=10
Environment="PANEL_PORT=${RANDOM_PORT}"
Environment="PANEL_ENTRY=${RANDOM_ENTRY}"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "${SERVICE_NAME}"
    print_info "服务配置完成。"
}

# --- 启动面板 ---
start_upanel() {
    print_info "启动 UPanel..."
    systemctl start "${SERVICE_NAME}"
    sleep 3
    
    if systemctl is-active --quiet "${SERVICE_NAME}"; then
        print_info "UPanel 服务已成功启动。"
    else
        print_error "UPanel 服务启动失败，请检查日志：journalctl -u ${SERVICE_NAME} -f"
        exit 1
    fi
}

# --- 输出安装信息 ---
print_summary() {
    local server_ip=$(curl -s ifconfig.me)
    echo ""
    echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}                    UPanel 安装完成！                        ${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  🌐 访问地址: ${GREEN}http://${server_ip}:${RANDOM_PORT}/${RANDOM_ENTRY}${NC}"
    echo -e "  👤 用户名:   ${GREEN}admin${NC}"
    echo -e "  🔑 密码:     ${GREEN}${RANDOM_PASS}${NC}"
    echo ""
    echo -e "  ${YELLOW}请务必保存好您的密码！后续无法再次查看。${NC}"
    echo ""
    echo -e "  📌 管理命令:"
    echo -e "     启动: systemctl start ${SERVICE_NAME}"
    echo -e "     停止: systemctl stop ${SERVICE_NAME}"
    echo -e "     状态: systemctl status ${SERVICE_NAME}"
    echo -e "     日志: journalctl -u ${SERVICE_NAME} -f"
    echo ""
    echo -e "  ${YELLOW}提示: 请在服务器安全组中放行端口 ${RANDOM_PORT}${NC}"
    echo ""
}

# --- 主流程 ---
main() {
    detect_os
    install_docker
    download_upanel
    configure_service
    start_upanel
    print_summary
}

main "$@"