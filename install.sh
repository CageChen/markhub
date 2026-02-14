#!/bin/sh
# MarkHub installer — downloads the latest release binary for your platform.
# Usage: curl -sSfL https://raw.githubusercontent.com/CageChen/markhub/master/install.sh | sh
set -e

REPO="CageChen/markhub"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY="markhub"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)  OS="linux";;
    Darwin*) OS="darwin";;
    MINGW*|MSYS*|CYGWIN*) OS="windows";;
    *) echo "Unsupported OS: $OS"; exit 1;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) ARCH="amd64";;
    aarch64|arm64) ARCH="arm64";;
    *) echo "Unsupported architecture: $ARCH"; exit 1;;
esac

# Get latest version
VERSION="$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')"
if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version"
    exit 1
fi
VERSION_NUM="${VERSION#v}"

echo "Installing ${BINARY} ${VERSION} (${OS}/${ARCH})..."

# Build download URL
if [ "$OS" = "windows" ]; then
    ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.zip"
else
    ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
fi
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

# Download and extract
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${URL}..."
curl -sSfL "$URL" -o "${TMPDIR}/${ARCHIVE}"

if [ "$OS" = "windows" ]; then
    unzip -q "${TMPDIR}/${ARCHIVE}" -d "$TMPDIR"
else
    tar xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"
fi

# Install
mkdir -p "$INSTALL_DIR" 2>/dev/null || true
if [ -w "$INSTALL_DIR" ]; then
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    echo "Need sudo to install to ${INSTALL_DIR}"
    sudo mkdir -p "$INSTALL_DIR"
    sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"
echo "Installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
echo "Run 'markhub --help' to get started."
