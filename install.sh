#!/usr/bin/env bash
set -euo pipefail

# SWAT Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/LangSensei/swat/main/install.sh | bash

REPO="LangSensei/swat"
SWAT_HOME="$HOME/.swat"
BIN_DIR="$HOME/.swat/bin"

# --- Safety: refuse to run as root ---
if [[ "$(id -u)" -eq 0 ]]; then
    echo -e "\033[0;31m[swat]\033[0m Do not run this installer as root or with sudo."
    echo -e "\033[0;31m[swat]\033[0m Run as your normal user:  curl -fsSL ... | bash"
    exit 1
fi

info()  { echo -e "\033[0;36m[swat]\033[0m $*"; }
ok()    { echo -e "\033[0;32m[swat]\033[0m $*"; }
err()   { echo -e "\033[0;31m[swat]\033[0m $*" >&2; }
die()   { err "$@"; exit 1; }

# --- Detect Platform ---

detect_platform() {
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$os" in
        linux)  OS="linux" ;;
        darwin) OS="darwin" ;;
        *)      die "Unsupported OS: $os" ;;
    esac

    case "$arch" in
        x86_64|amd64)  ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)             die "Unsupported architecture: $arch" ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    info "Detected platform: $PLATFORM"
}

# --- Prerequisites ---

check_prereqs() {
    local missing=()
    command -v curl >/dev/null 2>&1 || command -v wget >/dev/null 2>&1 || missing+=("curl or wget")
    command -v tar  >/dev/null 2>&1 || missing+=("tar")
    command -v node >/dev/null 2>&1 || missing+=("node")
    command -v npm  >/dev/null 2>&1 || missing+=("npm")

    if [[ ${#missing[@]} -gt 0 ]]; then
        die "Missing prerequisites: ${missing[*]}"
    fi
}

# --- Download & Extract ---

download() {
    local url="$1" dest="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$dest" "$url"
    else
        wget -qO "$dest" "$url"
    fi
}

fetch_release() {
    # Get latest release tag
    local api_url="https://api.github.com/repos/$REPO/releases/latest"
    local tag

    if command -v curl >/dev/null 2>&1; then
        tag=$(curl -fsSL "$api_url" | grep '"tag_name"' | head -1 | sed 's/.*: "\(.*\)".*/\1/')
    else
        tag=$(wget -qO- "$api_url" | grep '"tag_name"' | head -1 | sed 's/.*: "\(.*\)".*/\1/')
    fi

    if [[ -z "$tag" ]]; then
        die "Failed to fetch latest release"
    fi

    TAG="$tag"
    info "Latest release: $tag"

    # Check if already installed at this version
    local current
    current=$(swat --version 2>/dev/null | awk '{print $2}' || true)
    if [[ -n "$current" ]] && [[ "$current" == "$tag" ]]; then
        ok "Already up to date ($tag)"
        exit 0
    fi

    local tarball="swat-${tag}-${PLATFORM}.tar.gz"
    local dl_url="https://github.com/$REPO/releases/download/${tag}/${tarball}"
    local tmp_dir
    tmp_dir=$(mktemp -d)

    info "Downloading $tarball..."
    download "$dl_url" "$tmp_dir/$tarball"

    info "Extracting..."
    tar -xzf "$tmp_dir/$tarball" -C "$tmp_dir"

    EXTRACT_DIR="$tmp_dir"
}

# --- Install ---

install_binary() {
    mkdir -p "$BIN_DIR"
    cp "$EXTRACT_DIR/swat" "$BIN_DIR/swat"
    chmod +x "$BIN_DIR/swat"
    ok "Binary installed to $BIN_DIR/swat"
}



install_blueprints() {
    local bp="$SWAT_HOME/blueprints"
    mkdir -p "$bp/squads/_framework" "$bp/skills" "$bp/mcps"

    cp "$EXTRACT_DIR/blueprints/OPERATION.md" "$bp/"
    cp "$EXTRACT_DIR/blueprints/squads/_framework/"* "$bp/squads/_framework/"

    ok "Framework blueprints installed"
    info "Install squads/skills/mcps from the marketplace:"
    echo "  https://github.com/LangSensei/swat-marketplace"
}

# --- Post-Install ---


post_install() {
    # Symlink to ~/.local/bin (commonly in PATH on Linux/macOS)
    local local_bin="$HOME/.local/bin"
    mkdir -p "$local_bin"
    ln -sf "$BIN_DIR/swat" "$local_bin/swat"
    ok "Symlinked $local_bin/swat → $BIN_DIR/swat"

    # Also add BIN_DIR to shell profiles as fallback
    if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
        local line="export PATH=\"$BIN_DIR:\$PATH\""
        for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
            if [[ -f "$rc" ]] && ! grep -qF "$BIN_DIR" "$rc" 2>/dev/null; then
                echo "" >> "$rc"
                echo "# Added by SWAT installer" >> "$rc"
                echo "$line" >> "$rc"
                ok "Added $BIN_DIR to PATH in $(basename "$rc")"
            fi
        done
        export PATH="$BIN_DIR:$PATH"
    fi
}

# --- Cleanup ---

cleanup() {
    rm -rf "$EXTRACT_DIR"
}

# --- Main ---

main() {
    echo ""
    info "Installing SWAT..."
    echo ""

    detect_platform
    check_prereqs
    fetch_release
    install_binary
    install_blueprints
    post_install
    cleanup

    echo ""
    ok "SWAT installed successfully! 🚀"

    echo ""
    info "Next steps:"
    echo "  1. Add SWAT MCP server to your agent config:"
    echo "     {\"mcpServers\":{\"swat\":{\"command\":\"swat\",\"args\":[]}}}"
    echo ""
    echo "  Options (add to args):"
    echo "     --runtime <name>   Agent runtime: copilot (default), gemini"
    echo "     --notify <target>  Notifications: desktop (default), openclaw"
    echo ""
    echo "  2. For OpenClaw notifications, configure ~/.swat/.env:"
    echo "     OPENCLAW_NOTIFY_TARGET=<your_chat_id>"
    echo "     OPENCLAW_NOTIFY_CHANNEL=<telegram|discord|signal>"
    echo ""
    echo "     How to find your target ID:"
    echo "       Telegram: use the allowFrom value from ~/.openclaw/openclaw.json"
    echo "       Discord: your DM channel ID (enable Developer Mode, right-click channel → Copy ID)"
    echo ""
    echo "     Gateway settings (port and token) are read from ~/.openclaw/openclaw.json"
    echo "     automatically. Override with OPENCLAW_GATEWAY_PORT / OPENCLAW_GATEWAY_TOKEN"
    echo "     in ~/.swat/.env if needed."
    echo ""
    echo "     Test with: swat_notify after configuration."
    echo ""
    echo "  3. For OpenClaw integration: https://github.com/LangSensei/swat-openclaw"
    echo ""
}

main "$@"
