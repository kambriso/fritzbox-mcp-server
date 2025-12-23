#!/bin/sh
# fritzbox-mcp-server installer
# Downloads and installs pre-built binaries from GitHub releases
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/kambriso/fritzbox-mcp-server/main/install.sh | sh
#
# Environment variables:
#   FRITZBOX_MCP_VERSION       - Version to install (default: latest)
#   FRITZBOX_MCP_INSTALL_DIR   - Installation directory (default: $HOME/.local/bin)
#
# Security:
#   - Downloads over HTTPS only
#   - Verifies SHA256 checksums before installation
#   - No root/sudo required

set -eu

# Colors for TTY output
RED=''
GREEN=''
YELLOW=''
BLUE=''
RESET=''

# Global variables
TEMP_DIR=''
GITHUB_REPO='kambriso/fritzbox-mcp-server'
INSTALL_DIR="${FRITZBOX_MCP_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${FRITZBOX_MCP_VERSION:-latest}"

# Initialize colors if stdout is a TTY
init_colors() {
    if [ -t 1 ]; then
        RED='\033[0;31m'
        GREEN='\033[0;32m'
        YELLOW='\033[1;33m'
        BLUE='\033[0;34m'
        RESET='\033[0m'
    fi
}

# Log message to stderr
log() {
    printf '%b\n' "$*" >&2
}

# Log error message in red
log_error() {
    log "${RED}Error: ${RESET}$*"
}

# Log success message in green
log_success() {
    log "${GREEN}$*${RESET}"
}

# Log info message in blue
log_info() {
    log "${BLUE}$*${RESET}"
}

# Log warning message in yellow
log_warning() {
    log "${YELLOW}Warning: ${RESET}$*"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Cleanup temporary files
cleanup() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Validate required commands exist
check_dependencies() {
    log_info "Checking dependencies..."

    # Check for download tool
    if ! command_exists curl && ! command_exists wget; then
        log_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    # Check for checksum tool
    if ! command_exists sha256sum && ! command_exists shasum; then
        log_error "Neither sha256sum nor shasum found. Please install coreutils or similar package."
        exit 1
    fi

    # Check for tar
    if ! command_exists tar; then
        log_error "tar not found. Please install tar."
        exit 1
    fi

    log_success "✓ All dependencies found"
}

# Detect operating system
detect_os() {
    os_name=$(uname -s)
    case "$os_name" in
        Linux)
            echo "linux"
            ;;
        Darwin)
            echo "darwin"
            ;;
        *)
            log_error "Unsupported operating system: $os_name"
            log_error "Supported: Linux, macOS (Darwin)"
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    arch_name=$(uname -m)
    case "$arch_name" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        i686|i386|x86)
            echo "386"
            ;;
        armv7l|armv7|arm)
            echo "arm"
            ;;
        *)
            log_error "Unsupported architecture: $arch_name"
            log_error "Supported: x86_64, aarch64, i686, armv7l"
            exit 1
            ;;
    esac
}

# Resolve version to install
resolve_version() {
    if [ "$VERSION" != "latest" ]; then
        echo "$VERSION"
        return
    fi

    log_info "Resolving latest version..."

    # Query GitHub API for latest release
    api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"

    if command_exists curl; then
        response=$(curl -fsSL "$api_url" 2>/dev/null || true)
    else
        response=$(wget -qO- "$api_url" 2>/dev/null || true)
    fi

    if [ -z "$response" ]; then
        log_error "Failed to query GitHub API for latest version"
        log_error "Please specify version explicitly with FRITZBOX_MCP_VERSION environment variable"
        log_error "Example: FRITZBOX_MCP_VERSION=v0.4.0 sh install.sh"
        exit 1
    fi

    # Extract tag_name from JSON response
    # This is a simple grep/sed approach that doesn't require jq
    version_tag=$(echo "$response" | grep '"tag_name"' | head -n 1 | sed -e 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

    if [ -z "$version_tag" ]; then
        log_error "Failed to parse version from GitHub API response"
        log_error "Please specify version explicitly with FRITZBOX_MCP_VERSION environment variable"
        exit 1
    fi

    echo "$version_tag"
}

# Download file with retry logic
download_with_retry() {
    url="$1"
    output="$2"
    max_attempts=3
    attempt=1

    while [ $attempt -le $max_attempts ]; do
        if [ $attempt -gt 1 ]; then
            # Calculate delay: 2, 4, 8 seconds (POSIX-compliant)
            case $attempt in
                2) delay=2 ;;
                3) delay=4 ;;
                *) delay=8 ;;
            esac
            log_warning "Retry attempt $attempt/$max_attempts in ${delay}s..."
            sleep $delay
        fi

        if command_exists curl; then
            if curl -fsSL -o "$output" "$url" 2>/dev/null; then
                return 0
            fi
        else
            if wget -q -O "$output" "$url" 2>/dev/null; then
                return 0
            fi
        fi

        attempt=$((attempt + 1))
    done

    return 1
}

# Verify SHA256 checksum
verify_checksum() {
    file_path="$1"
    checksums_file="$2"
    file_name=$(basename "$file_path")

    log_info "Verifying SHA256 checksum..."

    # Extract expected checksum for this file
    expected=$(grep "$file_name" "$checksums_file" | head -n 1 | awk '{print $1}')

    if [ -z "$expected" ]; then
        log_error "No checksum found for $file_name in SHA256SUMS"
        return 1
    fi

    # Validate checksum format (64 hex characters)
    if ! echo "$expected" | grep -qE '^[a-fA-F0-9]{64}$'; then
        log_error "Invalid checksum format: $expected"
        return 1
    fi

    # Compute actual checksum
    if command_exists sha256sum; then
        actual=$(sha256sum "$file_path" | awk '{print $1}')
    else
        actual=$(shasum -a 256 "$file_path" | awk '{print $1}')
    fi

    # Compare (case-insensitive)
    expected_lower=$(echo "$expected" | tr '[:upper:]' '[:lower:]')
    actual_lower=$(echo "$actual" | tr '[:upper:]' '[:lower:]')

    if [ "$expected_lower" != "$actual_lower" ]; then
        log_error "Checksum verification failed!"
        log_error "Expected: $expected"
        log_error "Actual:   $actual"
        return 1
    fi

    log_success "✓ Checksum verified"
    return 0
}

# Prompt user for confirmation (only in TTY)
prompt_confirm() {
    prompt_text="$1"

    if [ ! -t 0 ]; then
        # Non-interactive, auto-confirm
        return 0
    fi

    printf '%b' "${YELLOW}${prompt_text} [y/N] ${RESET}" >&2
    read -r response
    case "$response" in
        [yY]|[yY][eE][sS])
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Main installation function
main() {
    init_colors

    log_info "fritzbox-mcp-server installer"
    log_info "=============================="
    echo ""

    # Setup cleanup trap
    trap cleanup EXIT INT TERM

    # Create temp directory
    TEMP_DIR=$(mktemp -d)
    if [ ! -d "$TEMP_DIR" ]; then
        log_error "Failed to create temporary directory"
        exit 1
    fi

    # Check dependencies
    check_dependencies

    # Detect platform
    os=$(detect_os)
    arch=$(detect_arch)
    log_info "Detected platform: $os-$arch"

    # Resolve version
    version=$(resolve_version)
    log_info "Installing version: $version"

    # Construct binary name and URLs
    binary_name="fritz-mcp-${os}-${arch}.tar.xz"
    base_url="https://github.com/${GITHUB_REPO}/releases/download/${version}"
    binary_url="${base_url}/${binary_name}"
    checksums_url="${base_url}/SHA256SUMS"

    # Download checksums file
    log_info "Downloading SHA256SUMS..."
    checksums_file="${TEMP_DIR}/SHA256SUMS"
    if ! download_with_retry "$checksums_url" "$checksums_file"; then
        log_error "Failed to download SHA256SUMS from $checksums_url"
        log_error "Please check your internet connection and verify the version exists"
        exit 1
    fi

    # Validate checksums file is non-empty
    if [ ! -s "$checksums_file" ]; then
        log_error "Downloaded SHA256SUMS file is empty"
        exit 1
    fi

    # Download binary tarball
    log_info "Downloading $binary_name..."
    tarball_path="${TEMP_DIR}/${binary_name}"
    if ! download_with_retry "$binary_url" "$tarball_path"; then
        log_error "Failed to download binary from $binary_url"
        log_error "Please check your internet connection and verify the version exists"
        exit 1
    fi

    # Validate tarball is non-empty
    if [ ! -s "$tarball_path" ]; then
        log_error "Downloaded tarball is empty"
        exit 1
    fi

    # Verify checksum
    if ! verify_checksum "$tarball_path" "$checksums_file"; then
        log_error "Checksum verification failed - refusing to install"
        rm -f "$tarball_path"
        exit 1
    fi

    # Extract tarball
    log_info "Extracting binary..."
    if ! tar -xJf "$tarball_path" -C "$TEMP_DIR" 2>/dev/null; then
        log_error "Failed to extract tarball"
        exit 1
    fi

    # Verify extracted binary exists
    extracted_binary="${TEMP_DIR}/fritz-mcp"
    if [ ! -f "$extracted_binary" ]; then
        log_error "Extracted binary not found: fritz-mcp"
        exit 1
    fi

    # Create installation directory if needed
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating installation directory: $INSTALL_DIR"
        if ! mkdir -p "$INSTALL_DIR"; then
            log_error "Failed to create installation directory: $INSTALL_DIR"
            exit 1
        fi
    fi

    # Check for existing installation
    target_binary="${INSTALL_DIR}/fritzbox-mcp-server"
    if [ -f "$target_binary" ]; then
        log_warning "Existing installation found at: $target_binary"
        if ! prompt_confirm "Overwrite existing installation?"; then
            log_info "Installation cancelled by user"
            exit 0
        fi
    fi

    # Install binary
    log_info "Installing to: $target_binary"
    if ! cp "$extracted_binary" "$target_binary"; then
        log_error "Failed to copy binary to $target_binary"
        exit 1
    fi

    # Make executable
    if ! chmod +x "$target_binary"; then
        log_error "Failed to make binary executable"
        exit 1
    fi

    # Verify installed binary is executable
    if [ ! -x "$target_binary" ]; then
        log_error "Installed binary is not executable"
        exit 1
    fi

    # Success!
    echo ""
    log_success "=============================="
    log_success "✓ fritzbox-mcp-server installed successfully!"
    log_success "=============================="
    echo ""
    log_info "Installation location: $target_binary"
    log_info "Version: $version"
    echo ""
    log_info "Next steps:"
    log_info "1. Configure your AI agent (Claude Desktop, Cline, etc.) to use this MCP server"
    log_info "2. Create a .env file with your Fritz!Box credentials (see README for details)"
    log_info "3. Add the server to your MCP configuration file"
    echo ""
    log_info "Documentation: https://github.com/${GITHUB_REPO}"
    echo ""
}

# Run main function
main
