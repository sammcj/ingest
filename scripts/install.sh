#!/usr/bin/env bash

# This is a simple installer that gets the latest version of ingest from Github and installs it to /usr/local/bin

INSTALL_DIR="/usr/local/bin"
INSTALL_PATH="${INSTALL_PATH:-$INSTALL_DIR/ingest}"
ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# Ensure the user is not root
if [ "$EUID" -eq 0 ]; then
  echo "Please do not run as root"
  exit 1
fi

# Get the latest release from Github
VER=$(curl --silent -qI https://github.com/sammcj/ingest/releases/latest | awk -F '/' '/^location/ {print  substr($NF, 1, length($NF)-1)}')

echo "Downloading ingest ${VER} for ${OS}-${ARCH}..."

wget https://github.com/sammcj/ingest/releases/download/$VER/ingest-${OS}-${ARCH} -O ingest

# # Move the binary to the install directory
mv ingest "${INSTALL_PATH}"

# # Make the binary executable
chmod +x "${INSTALL_PATH}"

echo "ingest has been installed to ${INSTALL_PATH}"
