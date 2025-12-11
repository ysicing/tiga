#!/bin/sh
# Pre-removal script for tiga
# Stops services before package removal

set -e

if command -v systemctl >/dev/null 2>&1; then
    # Stop services if they are running
    if systemctl is-active --quiet tiga; then
        echo "Stopping tiga service..."
        systemctl stop tiga || true
    fi

    if systemctl is-active --quiet tiga-agent; then
        echo "Stopping tiga-agent service..."
        systemctl stop tiga-agent || true
    fi

    # Disable services
    if systemctl is-enabled --quiet tiga; then
        echo "Disabling tiga service..."
        systemctl disable tiga || true
    fi

    if systemctl is-enabled --quiet tiga-agent; then
        echo "Disabling tiga-agent service..."
        systemctl disable tiga-agent || true
    fi
fi

exit 0
