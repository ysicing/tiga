#!/bin/sh
# Post-removal script for tiga
# Cleanup after package removal (preserves user data)

set -e

echo "=========================================="
echo "Tiga has been removed from your system."
echo "=========================================="

# Reload systemd daemon
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
fi

echo ""
echo "Note: User data and configurations are preserved:"
echo "  - Configuration: /etc/tiga/"
echo "  - Data directory: /var/lib/tiga/"
echo "  - Logs: /var/log/tiga/"
echo ""
echo "To completely remove all data:"
echo "  sudo rm -rf /etc/tiga /var/lib/tiga /var/log/tiga"
echo ""
echo "To remove tiga user and group:"
echo "  sudo userdel tiga"
echo "  sudo groupdel tiga"
echo "=========================================="

exit 0
