#!/bin/bash
# 验证 Tiga 安装后的目录结构是否符合 FHS 标准

set -e

echo "=========================================="
echo "Tiga Directory Structure Validation"
echo "=========================================="
echo ""

ERRORS=0

# 检查配置目录
echo "✓ Checking /etc/tiga/ (configuration only)..."
if [ ! -d /etc/tiga ]; then
    echo "  ✗ ERROR: /etc/tiga does not exist"
    ERRORS=$((ERRORS + 1))
else
    # 检查是否有数据文件（不应该有）
    if [ -f /etc/tiga/tiga.db ]; then
        echo "  ✗ ERROR: Database file found in /etc/tiga/ (should be in /var/lib/tiga/)"
        ERRORS=$((ERRORS + 1))
    fi

    if [ -d /etc/tiga/recordings ]; then
        echo "  ✗ ERROR: Recordings directory found in /etc/tiga/ (should be in /var/lib/tiga/)"
        ERRORS=$((ERRORS + 1))
    fi

    # 检查配置文件是否存在
    if [ -f /etc/tiga/config.yaml ]; then
        echo "  ✓ /etc/tiga/config.yaml exists"

        # 检查数据库路径配置
        if grep -q "database:" /etc/tiga/config.yaml; then
            DB_PATH=$(grep -A 5 "^database:" /etc/tiga/config.yaml | grep "  database:" | sed 's/.*database: *//' | tr -d '"' || echo "")
            if [ -n "$DB_PATH" ]; then
                case "$DB_PATH" in
                    /var/lib/tiga/*)
                        echo "  ✓ Database path: $DB_PATH (correct, in /var/lib/tiga/)"
                        ;;
                    /*)
                        echo "  ⚠ WARNING: Database path: $DB_PATH (absolute path, but not in /var/lib/tiga/)"
                        ;;
                    *)
                        echo "  ✗ ERROR: Database path: $DB_PATH (relative path, should use /var/lib/tiga/)"
                        ERRORS=$((ERRORS + 1))
                        ;;
                esac
            fi
        fi
    else
        echo "  ✗ WARNING: /etc/tiga/config.yaml not found"
    fi
fi

echo ""

# 检查数据目录
echo "✓ Checking /var/lib/tiga/ (data storage)..."
if [ ! -d /var/lib/tiga ]; then
    echo "  ✗ ERROR: /var/lib/tiga does not exist"
    ERRORS=$((ERRORS + 1))
else
    # 检查所有权
    OWNER=$(stat -c "%U:%G" /var/lib/tiga 2>/dev/null || stat -f "%Su:%Sg" /var/lib/tiga 2>/dev/null)
    if [ "$OWNER" = "tiga:tiga" ]; then
        echo "  ✓ Ownership correct: tiga:tiga"
    else
        echo "  ✗ ERROR: Wrong ownership: $OWNER (should be tiga:tiga)"
        ERRORS=$((ERRORS + 1))
    fi

    # 检查权限
    PERMS=$(stat -c "%a" /var/lib/tiga 2>/dev/null || stat -f "%Lp" /var/lib/tiga 2>/dev/null)
    if [ "$PERMS" = "755" ]; then
        echo "  ✓ Permissions correct: 755"
    else
        echo "  ⚠ WARNING: Permissions: $PERMS (recommended: 755)"
    fi

    # 检查 tiga 用户是否可写
    if sudo -u tiga test -w /var/lib/tiga; then
        echo "  ✓ tiga user can write to /var/lib/tiga"
    else
        echo "  ✗ ERROR: tiga user cannot write to /var/lib/tiga"
        ERRORS=$((ERRORS + 1))
    fi
fi

echo ""

# 检查日志目录
echo "✓ Checking /var/log/tiga/ (logs)..."
if [ ! -d /var/log/tiga ]; then
    echo "  ⚠ WARNING: /var/log/tiga does not exist (optional)"
else
    OWNER=$(stat -c "%U:%G" /var/log/tiga 2>/dev/null || stat -f "%Su:%Sg" /var/log/tiga 2>/dev/null)
    if [ "$OWNER" = "tiga:tiga" ]; then
        echo "  ✓ Ownership correct: tiga:tiga"
    else
        echo "  ✗ ERROR: Wrong ownership: $OWNER (should be tiga:tiga)"
        ERRORS=$((ERRORS + 1))
    fi
fi

echo ""

# 检查二进制文件
echo "✓ Checking binaries..."
if [ -f /usr/bin/tiga ]; then
    echo "  ✓ /usr/bin/tiga exists"
    /usr/bin/tiga --version 2>&1 | head -3 || echo "  ⚠ Cannot get version"
else
    echo "  ✗ ERROR: /usr/bin/tiga not found"
    ERRORS=$((ERRORS + 1))
fi

if [ -f /usr/bin/tiga-agent ]; then
    echo "  ✓ /usr/bin/tiga-agent exists"
else
    echo "  ⚠ WARNING: /usr/bin/tiga-agent not found (optional)"
fi

echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "✓ Validation passed! Installation structure is correct."
    echo ""
    echo "Next step: Start the service and access the installation wizard"
    echo "  sudo systemctl start tiga"
    echo "  Open browser: http://localhost:12306"
    exit 0
else
    echo "✗ Validation failed with $ERRORS error(s)."
    echo ""
    echo "To fix permissions:"
    echo "  sudo chown -R tiga:tiga /var/lib/tiga /var/log/tiga"
    echo "  sudo chmod 755 /var/lib/tiga /var/log/tiga"
    echo ""
    exit 1
fi
