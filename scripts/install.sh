#!/bin/bash
# kspec installation script
# Usage: curl -sSL https://raw.githubusercontent.com/cloudcwfranck/kspec/main/scripts/install.sh | bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="cloudcwfranck/kspec"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="Linux";;
        Darwin*)    os="Darwin";;
        MINGW*|MSYS*|CYGWIN*) os="Windows";;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64";;
        aarch64|arm64)  arch="arm64";;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Get latest version from GitHub
get_latest_version() {
    if [ "$VERSION" = "latest" ]; then
        log_info "Fetching latest version..."
        local latest
        latest=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$latest" ]; then
            log_warning "Could not fetch latest version, defaulting to v0.1.0"
            echo "v0.1.0"
        else
            echo "$latest"
        fi
    else
        echo "$VERSION"
    fi
}

# Download and install kspec
install_kspec() {
    local platform version download_url tmpdir

    echo ""
    echo "┌─────────────────────────────────────────┐"
    echo "│ 📦 kspec Installer                      │"
    echo "└─────────────────────────────────────────┘"
    echo ""

    platform=$(detect_platform)
    log_info "Platform detected: $platform"

    version=$(get_latest_version)
    log_info "Version: $version"

    # Construct download URL
    local version_num="${version#v}"
    local archive_name="kspec_${version_num}_${platform}"

    if [ "${platform%_*}" = "Windows" ]; then
        archive_name="${archive_name}.zip"
    else
        archive_name="${archive_name}.tar.gz"
    fi

    download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"
    log_info "Download URL: $download_url"

    # Create temporary directory
    tmpdir=$(mktemp -d)
    trap "rm -rf '$tmpdir'" EXIT

    # Download
    log_info "Downloading kspec..."
    if ! curl -L --fail --progress-bar "$download_url" -o "$tmpdir/kspec-archive"; then
        log_error "Failed to download kspec from $download_url"
        log_error "Please check if the version exists: https://github.com/${REPO}/releases"
        exit 1
    fi
    log_success "Downloaded successfully"

    # Extract
    log_info "Extracting archive..."
    cd "$tmpdir"
    if [ "${platform%_*}" = "Windows" ]; then
        unzip -q kspec-archive
    else
        tar -xzf kspec-archive
    fi
    log_success "Extracted successfully"

    # Find the binary
    local binary_name="kspec"
    if [ "${platform%_*}" = "Windows" ]; then
        binary_name="kspec.exe"
    fi

    if [ ! -f "$binary_name" ]; then
        log_error "Binary not found in archive"
        exit 1
    fi

    # Install
    log_info "Installing to $INSTALL_DIR..."

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$binary_name" "$INSTALL_DIR/kspec"
        chmod +x "$INSTALL_DIR/kspec"
    else
        if command -v sudo >/dev/null 2>&1; then
            sudo mv "$binary_name" "$INSTALL_DIR/kspec"
            sudo chmod +x "$INSTALL_DIR/kspec"
        else
            log_error "Cannot write to $INSTALL_DIR and sudo is not available"
            log_error "Please run with sudo or set INSTALL_DIR to a writable location"
            exit 1
        fi
    fi

    log_success "kspec installed to $INSTALL_DIR/kspec"

    # Verify installation
    if command -v kspec >/dev/null 2>&1; then
        echo ""
        kspec version
    else
        log_warning "$INSTALL_DIR is not in your PATH"
        log_info "Add it to your PATH by adding this line to your shell profile:"
        echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
    fi

    echo ""
    log_success "Installation complete!"
    echo ""
    echo "🎉 Get started with:"
    echo "    kspec init              # Interactive setup wizard"
    echo "    kspec scan              # Scan your cluster"
    echo "    kspec --help            # See all commands"
    echo ""
}

# Run installation
install_kspec
