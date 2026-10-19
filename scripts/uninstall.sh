#!/bin/bash
# ============================================================================
#  UPanel 卸载脚本
#  用法: sudo bash uninstall.sh
# ============================================================================

set -e

if [ -t 1 ]; then
    GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; CYAN='\033[0;36m'; NC='\033[0m'
else
    GREEN=''; YELLOW=''; RED=''; CYAN=''; NC=''
fi

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

SERVICE_NAME="upanel"
INSTALL_DIR="/opt/upanel"

# 尽量从运行中的服务或可执行文件推断安装目录
if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
    dir="$(grep -m1 '^WorkingDirectory=' "/etc/systemd/system/${SERVICE_NAME}.service" | cut -d= -f2-)"
    [ -n "$dir" ] && INSTALL_DIR="$dir"
elif [ -n "$UPANEL_BASE_DIR" ]; then
    INSTALL_DIR="$UPANEL_BASE_DIR"
fi

if [ "$(id -u)" -ne 0 ]; then
    error "请使用 root 用户执行卸载"
    exit 1
fi

echo ""
echo -e "${CYAN}════════════════════════════════════════${NC}"
echo -e "${CYAN}         UPanel 卸载程序${NC}"
echo -e "${CYAN}════════════════════════════════════════${NC}"
echo ""
echo "  安装目录: $INSTALL_DIR"
echo ""

read -r -p "$(echo -e "${YELLOW}是否保留网站/数据库等数据？(y/N): ${NC}")" keep
read -r -p "$(echo -e "${RED}确认卸载 UPanel？(y/N): ${NC}")" sure
case "$sure" in
    [yY]*) ;;
    *) info "已取消"; exit 0 ;;
esac

# 1. 停止服务
if command -v systemctl >/dev/null 2>&1; then
    info "停止并禁用服务 ..."
    systemctl stop "$SERVICE_NAME" >/dev/null 2>&1 || true
    systemctl disable "$SERVICE_NAME" >/dev/null 2>&1 || true
fi

# 2. 清理 UPanel 管理的容器
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    info "清理 UPanel 管理的容器 ..."
    containers="$(docker ps -a --filter "label=managed-by=upanel" --format '{{.Names}}' 2>/dev/null || true)"
    for c in $containers; do
        [ -z "$c" ] && continue
        info "  移除容器: $c"
        docker rm -f "$c" >/dev/null 2>&1 || true
    done
    docker network rm upanel_network >/dev/null 2>&1 || true
fi

# 3. 删除文件
if [ -n "$INSTALL_DIR" ] && [ -d "$INSTALL_DIR" ]; then
    case "$keep" in
        [yY]*)
            info "保留数据目录: $INSTALL_DIR/data"
            rm -rf "$INSTALL_DIR/bin" "$INSTALL_DIR/web"
            rm -f "$INSTALL_DIR/install.sh" "$INSTALL_DIR/uninstall.sh"
            ;;
        *)
            info "删除安装目录: $INSTALL_DIR"
            rm -rf "$INSTALL_DIR"
            ;;
    esac
fi

rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
rm -f /usr/local/bin/up /usr/local/bin/upanel

if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || true
fi

echo ""
info "UPanel 已卸载完成"
echo ""
