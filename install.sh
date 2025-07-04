#!/bin/bash

# Define default paths
BINARY_PATH="/usr/local/bin"
LICENSE_PATH="/usr/local/share/doc/rwr"
README_PATH="/usr/local/share/doc/rwr"

# GitHub repository owner and name
REPO="FynxLabs/rwr"

# Detect operating system
OS=$(uname -s)
case "$OS" in
    Linux*)     OS="Linux";;
    Darwin*)    OS="Darwin";;
    *)          echo "Unsupported operating system: $OS"; exit 1;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64*)    ARCH="x86_64";;
    i386*)      ARCH="i386";;
    arm64*)     ARCH="arm64";;
    armv7*)     ARCH="armv7";;
    aarch64*)   ARCH="arm64";;
    *)          echo "Unsupported architecture: $ARCH"; exit 1;;
esac

# Get the latest release data from the GitHub API
latest_release=$(curl --silent "https://api.github.com/repos/$REPO/releases/latest")

# Extract the download URL for the desired asset using pure Bash/sed (compatible with macOS)
download_url=$(echo "$latest_release" | sed -n 's/.*"browser_download_url": "\([^"]*rwr_'"${OS}"'_'"${ARCH}"'\.tar\.gz\)".*/\1/p' | head -1)

if [ -z "$download_url" ]; then
    echo "Could not find a download URL for $OS $ARCH. Exiting."
    exit 1
fi

# Download the file
echo "Downloading RWR from $download_url"
if ! curl -L -o /tmp/rwr.tar.gz "$download_url"; then
    echo "Failed to download RWR. Exiting."
    exit 1
fi

# Extract the tar file to a temporary directory
mkdir -p /tmp/rwr_extracted
if ! tar -xzf /tmp/rwr.tar.gz -C /tmp/rwr_extracted; then
    echo "Failed to extract RWR archive. Exiting."
    rm -f /tmp/rwr.tar.gz
    exit 1
fi

# Check if binary exists and move it to the default binary path
if [ ! -f /tmp/rwr_extracted/rwr ]; then
    echo "Binary 'rwr' not found in downloaded archive. Exiting."
    rm -rf /tmp/rwr.tar.gz /tmp/rwr_extracted
    exit 1
fi

sudo mv /tmp/rwr_extracted/rwr "$BINARY_PATH"

# Ensure the binary is executable
sudo chmod +x "$BINARY_PATH/rwr"

# Move the LICENSE and README to the default documentation path
sudo mkdir -p "$LICENSE_PATH"
sudo mkdir -p "$README_PATH"
if [ -f /tmp/rwr_extracted/LICENSE ]; then
    sudo mv /tmp/rwr_extracted/LICENSE "$LICENSE_PATH"
fi
if [ -f /tmp/rwr_extracted/README.adoc ]; then
    sudo mv /tmp/rwr_extracted/README.adoc "$README_PATH"
elif [ -f /tmp/rwr_extracted/README ]; then
    sudo mv /tmp/rwr_extracted/README "$README_PATH"
fi

# Clean up temporary files
rm -rf /tmp/rwr.tar.gz /tmp/rwr_extracted

echo "rwr has been installed successfully for $OS $ARCH."