#!/usr/bin/env bash
set -euo pipefail

# SWAT v2 Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/master/install.sh | bash

REPO="https://github.com/LangSensei/swat-v2.git"
SWAT_HOME="$HOME/.swat"
INSTALL_DIR="$SWAT_HOME/core"
BIN_DIR="$HOME/.local/bin"

info()  { echo -e "\033[0;36m[swat]\033[0m $*"; }
ok()    { echo -e "\033[0;32m[swat]\033[0m $*"; }
err()   { echo -e "\033[0;31m[swat]\033[0m $*" >&2; }
die()   { err "$@"; exit 1; }

# --- Prerequisites ---

check_prereqs() {
    local missing=()

    command -v go   >/dev/null 2>&1 || missing+=("go")
    command -v node >/dev/null 2>&1 || missing+=("node")
    command -v npm  >/dev/null 2>&1 || missing+=("npm")
    command -v git  >/dev/null 2>&1 || missing+=("git")

    if [[ ${#missing[@]} -gt 0 ]]; then
        die "Missing prerequisites: ${missing[*]}"
    fi

    # Copilot CLI is optional at install time, required at runtime
    if ! command -v copilot >/dev/null 2>&1; then
        info "Warning: GitHub Copilot CLI not found. Install it before dispatching tasks."
        info "  npm install -g @githubnext/github-copilot-cli"
    fi
}

# --- Clone / Update Source ---

fetch_source() {
    if [[ -d "$INSTALL_DIR/.git" ]]; then
        info "Updating SWAT core..."
        git -C "$INSTALL_DIR" pull --quiet
    else
        info "Cloning SWAT core..."
        mkdir -p "$INSTALL_DIR"
        git clone --quiet "$REPO" "$INSTALL_DIR"
    fi
}

# --- Build Binary ---

build_binary() {
    info "Building SWAT binary..."
    cd "$INSTALL_DIR"
    go build -o swat .
    mkdir -p "$BIN_DIR"
    cp swat "$BIN_DIR/swat"
    chmod +x "$BIN_DIR/swat"
    ok "Binary installed to $BIN_DIR/swat"
}

# --- Install Plugin Dependencies ---

install_plugin() {
    info "Installing plugin dependencies..."
    cd "$INSTALL_DIR/plugin"
    npm install --quiet 2>/dev/null
    ok "Plugin ready"
}

# --- Set Up Blueprints ---

setup_blueprints() {
    local bp="$SWAT_HOME/blueprints"
    mkdir -p "$bp/squads/_framework" "$bp/skills" "$bp/mcps"

    # Core framework files (from swat-v2 repo)
    info "Installing framework blueprints..."
    cp "$INSTALL_DIR/blueprints/OPERATION.md" "$bp/"
    cp "$INSTALL_DIR/blueprints/_framework/"* "$bp/squads/_framework/"

    ok "Blueprints installed"
    info "Install squads/skills/mcps from the marketplace:"
    echo "  https://github.com/LangSensei/swat-marketplace"
}

# --- Set Up Runtime Directory ---

setup_runtime() {
    mkdir -p "$SWAT_HOME/squads"
}

# --- Register OpenClaw Plugin ---

register_plugin() {
    local oc_config="$HOME/.openclaw/openclaw.json"
    if [[ -f "$oc_config" ]]; then
        info "OpenClaw detected. Register the SWAT plugin by adding to your config:"
        echo ""
        echo "  \"plugins\": [\"$INSTALL_DIR/plugin\"]"
        echo ""
        info "Then restart OpenClaw: openclaw gateway restart"
    else
        info "OpenClaw not detected. Install OpenClaw and add the plugin path:"
        echo "  $INSTALL_DIR/plugin"
    fi
}

# --- PATH Check ---

check_path() {
    if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
        info "Add to your shell profile:"
        echo "  export PATH=\"$BIN_DIR:\$PATH\""
    fi
}

# --- Main ---

main() {
    echo ""
    info "Installing SWAT v2..."
    echo ""

    check_prereqs
    fetch_source
    build_binary
    install_plugin
    setup_blueprints
    setup_runtime
    register_plugin
    check_path

    echo ""
    ok "SWAT v2 installed successfully! 🚀"
    echo ""
}

main "$@"
