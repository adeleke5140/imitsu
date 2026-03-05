#!/bin/sh
set -e

REPO="adeleke5140/imitsu"
BINARY="itui"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Darwin) OS="darwin" ;;
    Linux)  OS="linux" ;;
    *)
        echo "Error: unsupported OS: $OS"
        exit 1
        ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
        echo "Error: unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Determine latest release tag
TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)"
if [ -z "$TAG" ]; then
    echo "Error: could not determine latest release"
    exit 1
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}-${OS}-${ARCH}"

echo "Downloading ${BINARY} ${TAG} for ${OS}/${ARCH}..."
TMPFILE="$(mktemp)"
curl -fsSL "$URL" -o "$TMPFILE"
chmod +x "$TMPFILE"

# Install — use sudo if needed
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
else
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "$TMPFILE" "${INSTALL_DIR}/${BINARY}"
fi

echo "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Run 'itui' to start."
