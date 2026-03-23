#!/usr/bin/env bash
set -euo pipefail

# SWAT v2 Uninstaller
# Usage: curl -fsSL https://raw.githubusercontent.com/LangSensei/swat-v2/main/uninstall.sh | bash

SWAT_HOME="$HOME/.swat"
BIN_DIR="$HOME/.local/bin"

info()  { echo -e "\033[0;36m[swat]\033[0m $*"; }
ok()    { echo -e "\033[0;32m[swat]\033[0m $*"; }
warn()  { echo -e "\033[0;33m[swat]\033[0m $*"; }
err()   { echo -e "\033[0;31m[swat]\033[0m $*" >&2; }

echo ""
echo "  SWAT v2 Uninstaller"
echo "  ==================="
echo ""

# --- Confirm -----------------------------------------------------------

if [[ "${1:-}" != "--yes" ]]; then
    warn "This will remove:"
    echo "  - Binary:     $BIN_DIR/swat"
    echo "  - Plugin:     $SWAT_HOME/plugin/"
    echo "  - Blueprints: $SWAT_HOME/blueprints/"
    echo ""
    warn "Runtime data ($SWAT_HOME/squads/, $SWAT_HOME/schedules/) will be KEPT unless you pass --purge"
    echo ""
    read -r -p "Continue? [y/N] " confirm
    if [[ "$confirm" != [yY] ]]; then
        info "Aborted."
        exit 0
    fi
fi

PURGE=false
for arg in "$@"; do
    [[ "$arg" == "--purge" ]] && PURGE=true
done

# --- Remove binary -----------------------------------------------------

if [[ -f "$BIN_DIR/swat" ]]; then
    rm -f "$BIN_DIR/swat"
    ok "Removed $BIN_DIR/swat"
else
    info "Binary not found at $BIN_DIR/swat (skipped)"
fi

# --- Remove plugin ------------------------------------------------------

if [[ -d "$SWAT_HOME/plugin" ]]; then
    rm -rf "$SWAT_HOME/plugin"
    ok "Removed $SWAT_HOME/plugin/"
fi

# --- Remove blueprints --------------------------------------------------

if [[ -d "$SWAT_HOME/blueprints" ]]; then
    rm -rf "$SWAT_HOME/blueprints"
    ok "Removed $SWAT_HOME/blueprints/"
fi

# --- Purge runtime data -------------------------------------------------

if $PURGE; then
    if [[ -d "$SWAT_HOME/squads" ]]; then
        rm -rf "$SWAT_HOME/squads"
        ok "Purged $SWAT_HOME/squads/"
    fi
    if [[ -d "$SWAT_HOME/schedules" ]]; then
        rm -rf "$SWAT_HOME/schedules"
        ok "Purged $SWAT_HOME/schedules/"
    fi
    # Remove entire .swat if empty
    rmdir "$SWAT_HOME" 2>/dev/null && ok "Removed $SWAT_HOME/" || true
else
    if [[ -d "$SWAT_HOME/squads" ]]; then
        info "Runtime data kept at $SWAT_HOME/squads/ (use --purge to remove)"
    fi
fi

# --- Remove skill --------------------------------------------------------

for dir in "$HOME/.npm-global/lib/node_modules/openclaw/skills" "/usr/local/lib/node_modules/openclaw/skills" "/usr/lib/node_modules/openclaw/skills"; do
    if [[ -d "$dir/swat" ]]; then
        rm -rf "$dir/swat"
        ok "Removed skill from $dir/swat/"
        break
    fi
done

# --- Remove PATH entry from shell profiles -------------------------------

for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
    if [[ -f "$rc" ]] && grep -q "# Added by SWAT installer" "$rc" 2>/dev/null; then
        # Remove the comment line and the export line right after it
        if sed --version &>/dev/null 2>&1; then
            sed -i '/# Added by SWAT installer/{N;d;}' "$rc"
        else
            sed -i '' '/# Added by SWAT installer/{N;d;}' "$rc"
        fi
        ok "Cleaned PATH from $(basename "$rc")"
    fi
done

# --- Remove OpenClaw plugin config --------------------------------------

OPENCLAW_CONFIG="$HOME/.openclaw/openclaw.json"
if [[ -f "$OPENCLAW_CONFIG" ]] && command -v node &>/dev/null; then
    node -e "
        const fs = require('fs');
        const cfg = JSON.parse(fs.readFileSync('$OPENCLAW_CONFIG', 'utf8'));
        let changed = false;

        // Remove from plugins.load.paths
        if (cfg.plugins?.load?.paths) {
            const before = cfg.plugins.load.paths.length;
            cfg.plugins.load.paths = cfg.plugins.load.paths.filter(p => !p.includes('.swat/plugin'));
            if (cfg.plugins.load.paths.length < before) changed = true;
            if (cfg.plugins.load.paths.length === 0) delete cfg.plugins.load.paths;
        }

        // Remove plugin entry
        if (cfg.plugins?.entries?.['swat-mcp-bridge']) {
            delete cfg.plugins.entries['swat-mcp-bridge'];
            changed = true;
        }

        if (changed) {
            fs.writeFileSync('$OPENCLAW_CONFIG', JSON.stringify(cfg, null, 2) + '\n');
            console.log('Removed SWAT plugin from OpenClaw config');
        }
    " 2>/dev/null && ok "Cleaned OpenClaw config" || true
fi

echo ""
ok "SWAT v2 uninstalled."
if ! $PURGE && [[ -d "$SWAT_HOME/squads" ]]; then
    info "To fully remove all data: rm -rf $SWAT_HOME"
fi
echo ""
