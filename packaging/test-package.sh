#!/bin/bash
# 快速测试脚本：在 Docker 容器中测试软件包安装

set -e

PACKAGE_TYPE="${1:-deb}"
DISTRO_IMAGE=""

case "$PACKAGE_TYPE" in
    deb)
        DISTRO_IMAGE="ubuntu:22.04"
        INSTALL_CMD="apt update && apt install -y /packages/*.deb"
        ;;
    rpm)
        DISTRO_IMAGE="rockylinux:9"
        INSTALL_CMD="dnf install -y /packages/*.rpm"
        ;;
    apk)
        DISTRO_IMAGE="alpine:3.19"
        INSTALL_CMD="apk add --allow-untrusted /packages/*.apk"
        ;;
    *)
        echo "Usage: $0 [deb|rpm|apk]"
        exit 1
        ;;
esac

echo "=========================================="
echo "Testing $PACKAGE_TYPE package in $DISTRO_IMAGE"
echo "=========================================="

# 构建软件包
echo "Building $PACKAGE_TYPE package..."
task "package:$PACKAGE_TYPE"

# 启动容器并测试安装
echo "Testing installation in Docker..."
docker run -it --rm \
    -v "$(pwd)/dist:/packages" \
    "$DISTRO_IMAGE" \
    /bin/sh -c "
        set -e
        echo '==> Installing package...'
        $INSTALL_CMD

        echo '==> Verifying installation...'
        command -v tiga || { echo 'ERROR: tiga not found'; exit 1; }
        command -v tiga-agent || { echo 'ERROR: tiga-agent not found'; exit 1; }
        test -f /etc/tiga/config.yaml || { echo 'ERROR: config not found'; exit 1; }
        test -d /var/lib/tiga || { echo 'ERROR: data dir not found'; exit 1; }
        id tiga >/dev/null 2>&1 || { echo 'ERROR: tiga user not found'; exit 1; }

        echo '==> Package installed successfully!'
        echo '==> Checking version...'
        tiga --version || echo 'Version command not available'
    "

echo ""
echo "=========================================="
echo "✓ $PACKAGE_TYPE package test passed!"
echo "=========================================="
