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

# Extract the download URL for the desired asset using pure Bash
if [ "$OS" == "Linux" ]; then
    download_url=$(echo "$latest_release" | grep -oP '"browser_download_url": "\K(.*?)(?=")' | grep "rwr_${OS}_${ARCH}.tar.gz")
else
    download_url=$(echo "$latest_release" | grep -oP '"browser_download_url": "\K(.*?)(?=")' | grep "rwr_${OS}_${ARCH}.tar.gz")
fi

if [ -z "$download_url" ]; then
    echo "Could not find a download URL for $OS $ARCH. Exiting."
    exit 1
fi

# Download the file
curl -L -o /tmp/rwr.tar.gz "$download_url"

# Extract the tar file to a temporary directory
mkdir -p /tmp/rwr_extracted
tar -xzf /tmp/rwr.tar.gz -C /tmp/rwr_extracted

# Move the binary to the default binary path
sudo mv /tmp/rwr_extracted/rwr "$BINARY_PATH"

# Ensure the binary is executable
sudo chmod +x "$BINARY_PATH/rwr"

# Move the LICENSE and README to the default documentation path
sudo mkdir -p "$LICENSE_PATH"
sudo mkdir -p "$README_PATH"
sudo mv /tmp/rwr_extracted/LICENSE "$LICENSE_PATH"
sudo mv /tmp/rwr_extracted/README "$README_PATH"

# Clean up temporary files
rm -rf /tmp/rwr.tar.gz /tmp/rwr_extracted

echo "rwr has been installed successfully for $OS $ARCH."