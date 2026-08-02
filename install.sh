#!/usr/bin/env bash
set -euo pipefail

REPO="FynxLabs/rwr"
BINARY_PATH="/usr/local/bin"
LICENSE_PATH="/usr/local/share/doc/rwr"
README_PATH="/usr/local/share/doc/rwr"

USER_AGENT="rwr-installer"

fail() {
    echo "$@" >&2
    exit 1
}

# By default the latest stable release is installed. GitHub's /releases/latest
# endpoint never returns prereleases, so the rolling `nightly` build (and any
# pinned tag) is reached through /releases/tags/<tag> instead — same response
# shape, same assets, same checksums.txt.
RELEASE_TAG=""
while [ $# -gt 0 ]; do
    case "$1" in
        --nightly)
            RELEASE_TAG="nightly"
            ;;
        --tag)
            [ $# -ge 2 ] || fail "--tag needs a value, e.g. --tag v0.5.1 or --tag nightly."
            RELEASE_TAG="$2"
            shift
            ;;
        --tag=*)
            RELEASE_TAG="${1#--tag=}"
            [ -n "$RELEASE_TAG" ] || fail "--tag needs a value, e.g. --tag v0.5.1 or --tag nightly."
            ;;
        -h|--help)
            echo "Usage: install.sh [--nightly | --tag <tag>]"
            echo "  --nightly     install the rolling prerelease built from master"
            echo "  --tag <tag>   install a specific release tag (e.g. v0.5.1)"
            exit 0
            ;;
        *)
            fail "Unknown option: $1. Supported options: --nightly, --tag <tag>."
            ;;
    esac
    shift
done

OS=$(uname -s)
case "$OS" in
    Linux*)     OS="Linux";;
    Darwin*)    OS="Darwin";;
    *)          fail "Unsupported operating system: $OS. RWR publishes Linux and Darwin builds; use install.ps1 on Windows.";;
esac

# These names have to match what goreleaser publishes (see .goreleaser.yaml):
# amd64 -> x86_64, arm+goarm=7 -> armv7, arm64 and riscv64 keep their names.
# There is deliberately no i386 entry: no 386 target is built, so accepting it
# only produced a misleading "could not find a download URL" later on.
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)       ARCH="x86_64";;
    arm64|aarch64)      ARCH="arm64";;
    armv7*|armv8l)      ARCH="armv7";;
    riscv64)            ARCH="riscv64";;
    *)                  fail "Unsupported architecture: $ARCH. RWR publishes x86_64, arm64, armv7 (Linux only) and riscv64 (Linux only) builds.";;
esac

# riscv64 and armv7 are only built for Linux.
if [ "$OS" = "Darwin" ] && { [ "$ARCH" = "riscv64" ] || [ "$ARCH" = "armv7" ]; }; then
    fail "Unsupported platform: $OS $ARCH. RWR publishes Darwin builds for x86_64 and arm64 only."
fi

ASSET="rwr_${OS}_${ARCH}.tar.gz"
CHECKSUMS="checksums.txt"

# Work out how to write to the install locations before downloading anything, so
# a missing sudo is reported up front rather than half way through an install.
first_existing_dir() {
    local dir="$1"
    while [ ! -d "$dir" ] && [ "$dir" != "/" ]; do
        dir=$(dirname "$dir")
    done
    printf '%s' "$dir"
}

SUDO=""
as_root() {
    if [ -n "$SUDO" ]; then
        sudo "$@"
    else
        "$@"
    fi
}

if [ "$(id -u)" -ne 0 ]; then
    for target in "$BINARY_PATH" "$LICENSE_PATH" "$README_PATH"; do
        if [ ! -w "$(first_existing_dir "$target")" ]; then
            SUDO="sudo"
            break
        fi
    done
    if [ -n "$SUDO" ] && ! command -v sudo >/dev/null 2>&1; then
        fail "Installing to $BINARY_PATH needs root, but sudo is not available. Re-run this script as root, or set a writable install prefix."
    fi
fi

# A private per-run directory: fixed /tmp paths can be pre-created by any local
# user as a symlink, and everything below either downloads into this directory
# or copies out of it with root privileges.
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/rwr-install.XXXXXXXXXX")
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

if [ -n "$RELEASE_TAG" ]; then
    RELEASE_API="https://api.github.com/repos/$REPO/releases/tags/$RELEASE_TAG"
    RELEASE_DESC="release $RELEASE_TAG"
    if [ "$RELEASE_TAG" = "nightly" ]; then
        echo "Installing the nightly prerelease: an unvetted build of whatever master last was."
    fi
else
    RELEASE_API="https://api.github.com/repos/$REPO/releases/latest"
    RELEASE_DESC="latest release"
fi

echo "Installing RWR for $OS $ARCH"

if ! latest_release=$(curl -fsSL -H "User-Agent: $USER_AGENT" "$RELEASE_API"); then
    fail "Failed to query the $RELEASE_DESC from the GitHub API. The tag may not exist, the API may be unreachable, or it is rate limiting this host (unauthenticated requests are limited per IP)."
fi

asset_urls=$(printf '%s' "$latest_release" |
    grep -o '"browser_download_url"[[:space:]]*:[[:space:]]*"[^"]*"' |
    sed 's/.*"\(https[^"]*\)"$/\1/' || true)

if [ -z "$asset_urls" ]; then
    fail "The GitHub API response contained no release assets. The release may still be publishing, or the response was not what was expected."
fi

# Match the whole file name, not a substring, so rwr_Linux_arm64.tar.gz can
# never satisfy a request for rwr_Linux_arm.tar.gz.
url_for_asset() {
    printf '%s\n' "$asset_urls" |
        awk -v n="$1" 'substr($0, length($0) - length(n)) == "/" n { print; exit }'
}

download_url=$(url_for_asset "$ASSET")
if [ -z "$download_url" ]; then
    fail "The $RELEASE_DESC does not contain $ASSET, so there is no build for $OS $ARCH. Exiting."
fi

checksums_url=$(url_for_asset "$CHECKSUMS")
if [ -z "$checksums_url" ]; then
    fail "The $RELEASE_DESC does not publish $CHECKSUMS, so the download cannot be verified. Exiting."
fi

echo "Downloading $ASSET"
if ! curl -fsSL -H "User-Agent: $USER_AGENT" -o "$TMP_DIR/$ASSET" "$download_url"; then
    fail "Failed to download $ASSET from $download_url. Exiting."
fi

if ! curl -fsSL -H "User-Agent: $USER_AGENT" -o "$TMP_DIR/$CHECKSUMS" "$checksums_url"; then
    fail "Failed to download $CHECKSUMS from $checksums_url. Exiting."
fi

# Releases carry a keyless cosign signature over checksums.txt, bound to this
# repository's GitHub Actions identity. Verifying it upgrades the checksum
# comparison from integrity to authenticity. Opportunistic: cosign is not a
# requirement to install, but when it is present and the release publishes a
# signature, a bad signature is a hard stop.
sig_url=$(url_for_asset "$CHECKSUMS.sig")
cert_url=$(url_for_asset "$CHECKSUMS.pem")
if command -v cosign >/dev/null 2>&1 && [ -n "$sig_url" ] && [ -n "$cert_url" ]; then
    if ! curl -fsSL -H "User-Agent: $USER_AGENT" -o "$TMP_DIR/$CHECKSUMS.sig" "$sig_url" ||
       ! curl -fsSL -H "User-Agent: $USER_AGENT" -o "$TMP_DIR/$CHECKSUMS.pem" "$cert_url"; then
        fail "The release publishes a signature for $CHECKSUMS but it could not be downloaded. Refusing to install."
    fi
    if ! cosign verify-blob "$TMP_DIR/$CHECKSUMS" \
        --signature "$TMP_DIR/$CHECKSUMS.sig" \
        --certificate "$TMP_DIR/$CHECKSUMS.pem" \
        --certificate-identity-regexp "github.com/$REPO" \
        --certificate-oidc-issuer https://token.actions.githubusercontent.com >/dev/null 2>&1; then
        fail "cosign signature verification FAILED for $CHECKSUMS. The release may have been tampered with. Refusing to install."
    fi
    echo "Signature verified (cosign)"
elif [ -n "$sig_url" ]; then
    echo "Note: this release is signed; install cosign to verify signatures."
fi

# goreleaser writes "<sha256>  <file name>" lines. Compare $2 exactly so a
# prefix of another asset name cannot be picked up by mistake.
expected=$(awk -v n="$ASSET" '{ name = $2; sub(/^\*/, "", name); if (name == n) { print $1; exit } }' "$TMP_DIR/$CHECKSUMS")
if [ -z "$expected" ]; then
    fail "$CHECKSUMS has no entry for $ASSET, so the download cannot be verified. Refusing to install."
fi

if command -v sha256sum >/dev/null 2>&1; then
    SHA_CHECK=(sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
    SHA_CHECK=(shasum -a 256 -c -)
else
    fail "Neither sha256sum nor shasum is available, so the download cannot be verified. Refusing to install."
fi

if ! (cd "$TMP_DIR" && printf '%s  %s\n' "$expected" "$ASSET" | "${SHA_CHECK[@]}" >/dev/null 2>&1); then
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$TMP_DIR/$ASSET" | awk '{print $1}')
    else
        actual=$(shasum -a 256 "$TMP_DIR/$ASSET" | awk '{print $1}')
    fi
    echo "Checksum mismatch for $ASSET:" >&2
    echo "  expected $expected" >&2
    echo "  actual   $actual" >&2
    fail "Refusing to install."
fi
echo "Checksum verified"

EXTRACT_DIR="$TMP_DIR/extracted"
mkdir -p "$EXTRACT_DIR"
if ! tar -xzf "$TMP_DIR/$ASSET" -C "$EXTRACT_DIR"; then
    fail "Failed to extract $ASSET. Exiting."
fi

if [ ! -f "$EXTRACT_DIR/rwr" ]; then
    fail "Binary 'rwr' not found in the downloaded archive. Exiting."
fi

as_root mkdir -p "$BINARY_PATH" "$LICENSE_PATH" "$README_PATH"
as_root install -m 0755 "$EXTRACT_DIR/rwr" "$BINARY_PATH/rwr"

if [ -f "$EXTRACT_DIR/LICENSE" ]; then
    as_root install -m 0644 "$EXTRACT_DIR/LICENSE" "$LICENSE_PATH/LICENSE"
fi
for doc in README.adoc README.md README; do
    if [ -f "$EXTRACT_DIR/$doc" ]; then
        as_root install -m 0644 "$EXTRACT_DIR/$doc" "$README_PATH/$doc"
        break
    fi
done

echo "rwr has been installed to $BINARY_PATH for $OS $ARCH."
