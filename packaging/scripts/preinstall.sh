#!/bin/sh
# Pre-installation script for tiga
# Creates system user and group if they don't exist

set -e

# Create tiga user and group
if ! getent group tiga >/dev/null 2>&1; then
    echo "Creating tiga group..."
    groupadd -r tiga
fi

if ! getent passwd tiga >/dev/null 2>&1; then
    echo "Creating tiga user..."
    useradd -r -g tiga -d /var/lib/tiga -s /sbin/nologin \
        -c "Tiga DevOps Platform" tiga
fi

exit 0
