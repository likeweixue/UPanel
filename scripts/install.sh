#!/bin/bash

# UPanel 一键安装脚本
# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 默认配置
INSTALL_DIR="/opt/upanel"
PANEL_PORT=8080
PANEL_USER="admin"
PANEL_PASS="admin123"
GIT_REPO="https://github.com/likeweixue/UPanel.git"

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }

print_banner() {
    echo -e "${GREEN}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║                                                          ║"
    echo "║   ██╗   ██╗██████╗  █████╗ ███╗   ██╗███████╗██╗        ║"
    echo "║   ██║   ██║██╔══██╗██╔══██╗████╗  ██║██╔════╝██║        ║"
    echo "║   ██║   ██║██████╔╝███████║██╔██╗ ██║█████╗  ██║        ║"
    echo "║   ██║   ██║██╔═══╝ ██╔══██║██║╚██╗██║██╔══╝  ██║        ║"
    echo "║   ╚██████╔╝██║     ██║  ██║██║ ╚████║███████╗███████╗   ║"
    echo "║    ╚═════╝ ╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝   ║"
    echo "║                                                          ║"
    echo "║              轻量级容器管理面板 - 一键安装               ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_system() {
    print_info "检查系统环境..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        print_warning "检测到 macOS 系统"
        INSTALL_DIR="$HOME/upanel"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        print_success "系统: Linux"
    else
        print_error "不支持的操作系统"
        exit 1
    fi
    print_success "系统检查完成"
}

ask_config() {
    print_info "请配置面板参数（直接回车使用默认值）"
    echo ""
    read -p "安装目录 [$INSTALL_DIR]: " input
    INSTALL_DIR=${input:-$INSTALL_DIR}
    read -p "面板端口 [$PANEL_PORT]: " input
    PANEL_PORT=${input:-$PANEL_PORT}
    read -p "管理员用户名 [$PANEL_USER]: " input
    PANEL_USER=${input:-$PANEL_USER}
    read -sp "管理员密码 [$PANEL_PASS]: " input
    echo ""
    PANEL_PASS=${input:-$PANEL_PASS}
    echo ""
    print_info "配置确认："
    echo "  安装目录: $INSTALL_DIR"
    echo "  面板端口: $PANEL_PORT"
    echo "  管理员: $PANEL_USER"
    echo ""
    read -p "确认安装？(y/n): " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        print_info "安装已取消"
        exit 0
    fi
}

download_code() {
    print_info "下载 UPanel 代码..."
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
    if [ -d ".git" ]; then
        git pull
    else
        git clone "$GIT_REPO" .
    fi
    print_success "代码下载完成"
}

build_backend() {
    print_info "编译后端..."
    cd "$INSTALL_DIR"
    go mod download
    go mod tidy
    go build -o upanel cmd/upanel/main.go
    print_success "后端编译完成"
}

build_frontend() {
    print_info "构建前端..."
    cd "$INSTALL_DIR/web"
    npm install
    npm run build
    print_success "前端构建完成"
}

start_panel() {
    print_info "启动 UPanel..."
    cd "$INSTALL_DIR"
    nohup ./upanel > upanel.log 2>&1 &
    sleep 2
    print_success "UPanel 启动成功！"
}

print_summary() {
    echo ""
    echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}                    UPanel 安装完成！                        ${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  访问地址: ${BLUE}http://localhost:$PANEL_PORT${NC}"
    echo -e "  用户名:   ${BLUE}$PANEL_USER${NC}"
    echo -e "  密码:     ${BLUE}$PANEL_PASS${NC}"
    echo ""
    echo -e "  日志文件: ${YELLOW}$INSTALL_DIR/upanel.log${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════════════════${NC}"
}

main() {
    print_banner
    check_system
    ask_config
    download_code
    build_backend
    build_frontend
    start_panel
    print_summary
}

main "$@"
