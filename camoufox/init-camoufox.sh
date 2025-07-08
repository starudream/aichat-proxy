#!/usr/bin/env bash

GITHUB_BASE_URL="https://github.com"

# https://github.com/daijro/camoufox/releases/latest
CAMOUFOX_VERSION="135.0.1"
CAMOUFOX_RELEASE="beta.24"
CAMOUFOX_OS_ARCH=""
if [ "$(uname -m)" == "x86_64" ]; then
  CAMOUFOX_OS_ARCH="lin.x86_64"
elif [ "$(uname -m)" == "aarch64" ] || [ "$(uname -m)" == "arm64" ]; then
  CAMOUFOX_OS_ARCH="lin.arm64"
fi
CAMOUFOX_FULL_VERSION="${CAMOUFOX_VERSION}-${CAMOUFOX_RELEASE}"

echo "download camoufox-${CAMOUFOX_FULL_VERSION}-${CAMOUFOX_OS_ARCH}.zip from ${GITHUB_BASE_URL}"
wget -q --show-progress --progress dot:giga -O camoufox.zip ${GITHUB_BASE_URL}/daijro/camoufox/releases/download/v${CAMOUFOX_FULL_VERSION}/camoufox-${CAMOUFOX_FULL_VERSION}-${CAMOUFOX_OS_ARCH}.zip

echo "create version.json"
echo "{\"version\":\"${CAMOUFOX_VERSION}\",\"release\":\"${CAMOUFOX_RELEASE}\"}" > version.json

echo "done"
