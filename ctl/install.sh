#!/usr/bin/env sh

# This script installs veidemannctl
#
# It tries to detect the architecture and operating system
# and downloads the appropriate binary.
#
# If you want to install a specific version or architecture
# you can set the environment variables VERSION and ARCH
# before running this script.
#
# For example:
# $ VERSION=1.0.0 ARCH=arm64 ./install.sh
#
# This script requires curl, sed and tr to be installed.

set -e

RELEASES="https://github.com/NationalLibraryOfNorway/veidemann/ctl/releases"

# Detect architecture
ARCH=${ARCH:-$(uname -m)}
case $ARCH in
"x86_64")
  ARCH="amd64"
  ;;
"aarch64"|"arm64")
    ARCH="arm64"
    ;;
"armv7l")
    ARCH="armv7"
    ;;
"armv6l"|"aprm")
    ARCH="armv6"
    ;;
"armv5l")
    ARCH="armv5"
    ;;
esac

# Detect operating system
KERNEL=$(uname -s | tr '[:upper:]' '[:lower:]')

# Detect version
VERSION=${VERSION:-$(curl -s -I "${RELEASES}/latest" | grep location | sed -E 's|.*tag/v?([0-9.]+.*)$|\1|' | tr -d '\r')}

echo "Installing veidemannctl v${VERSION}"

curl -Lo veidemannctl "${RELEASES}/download/v${VERSION}/veidemannctl_${VERSION}_${KERNEL}_${ARCH}"
sudo install veidemannctl /usr/local/bin/veidemannctl
rm veidemannctl

# Install command completion for bash and zsh
if [ -n "${BASH}" ]; then
  echo "Installing bash completion for veidemannctl"
  sudo sh -c "/usr/local/bin/veidemannctl completion bash > /etc/bash_completion.d/veidemannctl"
fi
if [ -n "${ZSH_NAME}" ]; then
  echo "Installing zsh completion for veidemannctl"
  sudo sh -c "/usr/local/bin/veidemannctl completion zsh > /usr/local/share/zsh/site-functions/_veidemannctl"
fi
