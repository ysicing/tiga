#!/bin/bash
# Version extraction script for Tiga build-time injection
# Outputs VERSION, BUILD_TIME, and COMMIT_ID in environment variable format

set -e

# 1. Try to get the most recent git tag
TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

# 2. Get commit short hash (7 characters)
COMMIT=$(git rev-parse --short=7 HEAD 2>/dev/null || echo "0000000")

# 3. Get build time in RFC3339 format (UTC)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# 4. Generate version number based on git state
if [ -z "$(git rev-parse --git-dir 2>/dev/null)" ]; then
    # No git repository detected - use default values
    VERSION="dev"
    COMMIT="0000000"
elif [ -n "$TAG" ]; then
    # Tag found - use tag + commit format
    VERSION="${TAG}-${COMMIT}"
else
    # No tag - use date + commit format
    DATE=$(date -u +"%Y%m%d")
    VERSION="${DATE}-${COMMIT}"
fi

# 5. Output in environment variable format
echo "VERSION=${VERSION}"
echo "BUILD_TIME=${BUILD_TIME}"
echo "COMMIT_ID=${COMMIT}"
