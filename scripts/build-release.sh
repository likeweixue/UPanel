#!/bin/bash
# ============================================================================
#  UPanel 发布包构建脚本
#  产出: release/upanel-<version>-linux-amd64.tar.gz
#  用法: bash scripts/build-release.sh [版本号]
# ============================================================================

set -e

cd "$(dirname "$0")/.."
ROOT="$(pwd)"

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    if [ -f "$ROOT/VERSION" ]; then
        VERSION="$(cat "$ROOT/VERSION")"
    else
        VERSION="v0.1.7"
    fi
fi

GOOS_TARGET=linux
ARCH="${ARCH:-amd64}"
NODE_BIN="${NODE_BIN:-}"
NPM_BIN="${NPM_BIN:-}"

echo ""
echo "════════════════════════════════════════"
echo "  UPanel 发布包构建"
echo "  版本: $VERSION  平台: $GOOS_TARGET/$ARCH"
echo "════════════════════════════════════════"
echo ""

# ---------- 1. 构建前端 ----------
echo "[1/4] 构建前端 ..."
if [ -d "$ROOT/web" ]; then
    pushd "$ROOT/web" >/dev/null
    if [ ! -d node_modules ]; then
        echo "  安装前端依赖 ..."
        $NPM_BIN npm install --no-audit --no-fund || {
            echo "  npm install 失败，尝试国内镜像 ..."
            $NPM_BIN npm install --no-audit --no-fund --registry=https://registry.npmmirror.com
        }
    fi
    if [ "$SKIP_WEB_BUILD" = "1" ] && [ -f "$ROOT/web/dist/index.html" ]; then
        echo "  已存在构建产物，跳过 (SKIP_WEB_BUILD=1)"
    else
        $NPM_BIN npm run build
    fi
    popd >/dev/null

    if [ ! -f "$ROOT/web/dist/index.html" ]; then
        echo "  错误: 前端构建产物缺失"
        exit 1
    fi
    echo "  前端构建完成"
else
    echo "  警告: 未找到 web 目录，跳过前端构建"
fi

# ---------- 2. 编译后端 ----------
echo "[2/4] 编译后端 ..."
export CGO_ENABLED=0
export GOOS=$GOOS_TARGET
export GOARCH=$ARCH

mkdir -p "$ROOT/build"
go build -ldflags="-s -w -X main.Version=${VERSION}" -o "$ROOT/build/upanel" "$ROOT/cmd/upanel/main.go"
go build -ldflags="-s -w -X main.Version=${VERSION}" -o "$ROOT/build/up" "$ROOT/cmd/upanel/cli/main.go"
echo "  后端编译完成"

# ---------- 3. 组装目录 ----------
echo "[3/4] 组装发布目录 ..."
PKG_NAME="upanel-${VERSION}-linux-${ARCH}"
PKG_DIR="$ROOT/release/$PKG_NAME"
rm -rf "$PKG_DIR"
mkdir -p "$PKG_DIR/bin" "$PKG_DIR/web"

install -m 0755 "$ROOT/build/upanel" "$PKG_DIR/bin/upanel"
install -m 0755 "$ROOT/build/up"      "$PKG_DIR/bin/up"

if [ -d "$ROOT/web/dist" ]; then
    # 用 tar 管道拷贝，避免不同 cp 实现在隐藏文件 / 权限上的差异
    (cd "$ROOT/web/dist" && tar -cf - .) | (cd "$PKG_DIR/web" && tar -xf -)
fi

install -m 0755 "$ROOT/scripts/install.sh"    "$PKG_DIR/install.sh"
install -m 0755 "$ROOT/scripts/uninstall.sh"  "$PKG_DIR/uninstall.sh"
echo "$VERSION" > "$PKG_DIR/VERSION"

if [ -f "$ROOT/README.md" ]; then
    install -m 0644 "$ROOT/README.md" "$PKG_DIR/README.md"
fi
if [ -f "$ROOT/LICENSE" ]; then
    install -m 0644 "$ROOT/LICENSE" "$PKG_DIR/LICENSE"
fi

# ---------- 4. 打包 ----------
echo "[4/4] 打包 ..."
mkdir -p "$ROOT/release"
TARBALL="$ROOT/release/${PKG_NAME}.tar.gz"
rm -f "$TARBALL"
tar -czf "$TARBALL" -C "$ROOT/release" "$PKG_NAME"

rm -rf "$ROOT/build"

echo ""
echo "════════════════════════════════════════"
echo "  构建完成"
echo "  发布包: $TARBALL"
echo "  大小:   $(du -h "$TARBALL" | cut -f1)"
echo "════════════════════════════════════════"
echo ""
echo "安装方式:"
echo "  1. 上传到服务器任意目录，例如 /opt"
echo "  2. tar -xzf ${PKG_NAME}.tar.gz"
echo "  3. cd ${PKG_NAME} && bash install.sh"
echo ""
