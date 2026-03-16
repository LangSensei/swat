#!/usr/bin/env bash
set -euo pipefail

# SWAT v2 Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/master/install.sh | bash

REPO="LangSensei/swat-v2"
SWAT_HOME="$HOME/.swat"
BIN_DIR="$HOME/.local/bin"

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

    if ! command -v copilot >/dev/null 2>&1; then
        info "Warning: GitHub Copilot CLI not found. Required for running squads."
        info "  npm install -g @github/copilot"
    fi

    # Check gh CLI and auth status
    if command -v gh >/dev/null 2>&1; then
        if ! gh auth status &>/dev/null; then
            info "⚠️  GitHub CLI not authenticated. Run before using SWAT:"
            info "    gh auth login"
        fi
    else
        info "⚠️  GitHub CLI (gh) not found. Required for Copilot CLI auth."
        info "    Install: https://cli.github.com"
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

install_plugin() {
    local dest="$SWAT_HOME/plugin"
    rm -rf "$dest"
    mkdir -p "$dest"
    cp -r "$EXTRACT_DIR/plugin/"* "$dest/"

    info "Installing plugin dependencies..."
    cd "$dest" && npm install --quiet 2>/dev/null
    ok "Plugin installed to $dest"
}

install_skill() {
    local oc_skills
    # Find OpenClaw skills directory
    for dir in "$HOME/.npm-global/lib/node_modules/openclaw/skills" "/usr/local/lib/node_modules/openclaw/skills" "/usr/lib/node_modules/openclaw/skills"; do
        if [[ -d "$dir" ]]; then
            oc_skills="$dir"
            break
        fi
    done

    if [[ -z "$oc_skills" ]]; then
        info "OpenClaw skills directory not found. Manually copy skill/SKILL.md to your OpenClaw skills dir."
        return
    fi

    mkdir -p "$oc_skills/swat"
    cp "$EXTRACT_DIR/skill/SKILL.md" "$oc_skills/swat/"
    ok "Skill installed to $oc_skills/swat/"
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

setup_runtime() {
    mkdir -p "$SWAT_HOME/squads"
}

# --- Post-Install ---

register_plugin() {
    local oc_config="$HOME/.openclaw/openclaw.json"
    local plugin_path="$SWAT_HOME/plugin"

    if [[ ! -f "$oc_config" ]]; then
        info "OpenClaw config not found at $oc_config"
        info "After installing OpenClaw, add the SWAT plugin to your config:"
        echo ""
        echo "  plugins.load.paths: [\"$plugin_path\"]"
        echo "  plugins.entries.swat-mcp-bridge.enabled: true"
        echo ""
        info "Then restart OpenClaw: openclaw gateway restart"
        return
    fi

    # Check if already registered AND binaryPath is configured for this plugin
    if grep -q "$plugin_path" "$oc_config" 2>/dev/null && grep -A5 "swat-mcp-bridge" "$oc_config" 2>/dev/null | grep -q "binaryPath"; then
        ok "Plugin already registered in OpenClaw config"
        return
    fi

    # Use node to patch JSON config
    if node -e "
        const fs = require('fs');
        const cfg = JSON.parse(fs.readFileSync('$oc_config', 'utf8'));

        // Ensure plugins structure
        cfg.plugins = cfg.plugins || {};
        cfg.plugins.load = cfg.plugins.load || {};
        cfg.plugins.load.paths = cfg.plugins.load.paths || [];
        cfg.plugins.entries = cfg.plugins.entries || {};

        // Add plugin path if not present
        if (!cfg.plugins.load.paths.includes('$plugin_path')) {
            cfg.plugins.load.paths.push('$plugin_path');
        }

        // Enable plugin with binary path
        cfg.plugins.entries['swat-mcp-bridge'] = {
            enabled: true,
            config: { binaryPath: '$BIN_DIR/swat' }
        };

        fs.writeFileSync('$oc_config', JSON.stringify(cfg, null, 2) + '\n');
    " 2>/dev/null; then
        ok "Plugin registered in OpenClaw config"
        info "Restart OpenClaw to activate: openclaw gateway restart"
    else
        err "Failed to auto-register plugin. Manually add to $oc_config:"
        echo "  plugins.load.paths: [\"$plugin_path\"]"
        echo "  plugins.entries.swat-mcp-bridge.enabled: true"
    fi
}

post_install() {
    # PATH check — auto-add if missing
    if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
        local line="export PATH=\"$BIN_DIR:\$PATH\""
        local added=false
        for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
            if [[ -f "$rc" ]] && ! grep -qF "$BIN_DIR" "$rc" 2>/dev/null; then
                echo "" >> "$rc"
                echo "# Added by SWAT installer" >> "$rc"
                echo "$line" >> "$rc"
                added=true
                ok "Added $BIN_DIR to PATH in $(basename "$rc")"
            fi
        done
        if [[ "$added" == false ]]; then
            info "Add to your shell profile:"
            echo "  $line"
            echo ""
        fi
        export PATH="$BIN_DIR:$PATH"
    fi

    register_plugin
}

# --- Cleanup ---

cleanup() {
    rm -rf "$EXTRACT_DIR"
}

# --- Main ---

main() {
    echo ""
    info "Installing SWAT v2..."
    echo ""

    detect_platform
    check_prereqs
    fetch_release
    install_binary
    install_plugin
    install_skill
    install_blueprints
    setup_runtime
    post_install
    cleanup

    echo ""
    ok "SWAT v2 installed successfully! 🚀"

    echo ""
    info "Next steps:"
    echo "  1. Restart OpenClaw:  openclaw gateway restart"
    echo "  2. Install a squad:   Tell your agent: \"browse SWAT marketplace and install a squad\""
    echo "     Or use the tool directly: swat_browse → swat_install"
    echo ""
}

main "$@"
