#!/bin/bash
# ============================================================================
#  UPanel 在线一键安装（可选）
#  如果你把发布包托管到了某个可访问的地址，可以用这个脚本像 1Panel 那样安装。
#  用法:
#    bash -c "$(curl -sSL https://你的域名/upanel/quick_start.sh)"
#  可通过环境变量覆盖下载地址:
#    UPANEL_PKG_URL=https://xxx/upanel-v0.1.7-linux-amd64.tar.gz
# ============================================================================

set -e

PKG_URL="${UPANEL_PKG_URL:-https://github.com/likeweixue/UPanel/releases/latest/download/upanel-v0.1.7-linux-amd64.tar.gz}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "下载 UPanel 安装包 ..."
curl -fsSL -o "$TMP_DIR/upanel.tar.gz" "$PKG_URL"

echo "解压 ..."
tar -xzf "$TMP_DIR/upanel.tar.gz" -C "$TMP_DIR"

PKG_DIR="$(find "$TMP_DIR" -maxdepth 1 -type d -name 'upanel-*' | head -n1)"
if [ -z "$PKG_DIR" ] || [ ! -f "$PKG_DIR/install.sh" ]; then
    echo "错误: 安装包内容异常"
    exit 1
fi

echo "启动安装向导 ..."
exec bash "$PKG_DIR/install.sh"
