#!/bin/sh
# Post-installation script for tiga
# Reloads systemd and provides usage instructions

set -e

echo "=========================================="
echo "Tiga DevOps Platform installed successfully!"
echo "=========================================="
echo ""

# Reload systemd daemon
if command -v systemctl >/dev/null 2>&1; then
    echo "Reloading systemd daemon..."
    systemctl daemon-reload
fi

# Ensure data directories have correct ownership
echo "Setting up data directories..."
chown -R tiga:tiga /var/lib/tiga
chown -R tiga:tiga /var/log/tiga
chmod 755 /var/lib/tiga
chmod 755 /var/log/tiga

echo ""
echo "Installation paths:"
echo "  - Configuration: /etc/tiga/config.yaml"
echo "  - Data directory: /var/lib/tiga/"
echo "  - Logs: /var/log/tiga/"
echo "  - Binaries: /usr/bin/tiga, /usr/bin/tiga-agent"
echo ""
echo "=========================================="
echo "🎉 First-Time Setup Required!"
echo "=========================================="
echo ""
echo "Tiga uses a web-based installation wizard for first-time setup."
echo ""
echo "Next steps:"
echo ""
echo "  1. Start tiga server:"
echo "     sudo systemctl start tiga"
echo ""
echo "  2. Open browser and visit:"
echo "     http://localhost:12306"
echo ""
echo "  3. Complete the installation wizard:"
echo "     - Choose database type (SQLite/PostgreSQL/MySQL)"
echo "     - Create administrator account"
echo "     - Configure system settings"
echo ""
echo "  4. After wizard completion, enable on boot:"
echo "     sudo systemctl enable tiga"
echo ""
echo "  5. Check status:"
echo "     sudo systemctl status tiga"
echo ""
echo "  6. View logs:"
echo "     sudo journalctl -u tiga -f"
echo ""
echo "Optional - Start tiga-agent for host monitoring:"
echo "  sudo systemctl start tiga-agent"
echo "  sudo systemctl enable tiga-agent"
echo ""
echo "Note: Configuration file at /etc/tiga/config.yaml is minimal."
echo "      The installation wizard will complete it automatically."
echo ""
echo "Documentation: https://github.com/ysicing/tiga"
echo "=========================================="

exit 0
