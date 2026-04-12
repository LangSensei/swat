#!/usr/bin/env bash
set -euo pipefail

# SWAT Uninstaller
# Usage: curl -fsSL https://raw.githubusercontent.com/LangSensei/swat/main/uninstall.sh | bash

SWAT_HOME="$HOME/.swat"
BIN_DIR="$HOME/.local/bin"

info()  { echo -e "\033[0;36m[swat]\033[0m $*"; }
ok()    { echo -e "\033[0;32m[swat]\033[0m $*"; }
warn()  { echo -e "\033[0;33m[swat]\033[0m $*"; }
err()   { echo -e "\033[0;31m[swat]\033[0m $*" >&2; }

echo ""
echo "  SWAT Uninstaller"
echo "  ==================="
echo ""

# --- Confirm -----------------------------------------------------------

if [[ "${1:-}" != "--yes" ]]; then
    warn "This will remove:"
    echo "  - Binary:     $BIN_DIR/swat"
    echo "  - Framework:  $SWAT_HOME/blueprints/squads/_framework/, OPERATION.md"
    echo ""
    warn "User data (squads, skills, mcps, schedules, operations) will be KEPT unless you pass --purge"
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


# --- Remove blueprints (framework only; squads/skills/mcps kept unless --purge) ---

if [[ -d "$SWAT_HOME/blueprints" ]]; then
    if $PURGE; then
        rm -rf "$SWAT_HOME/blueprints"
        ok "Purged $SWAT_HOME/blueprints/"
    else
        # Remove framework files (reinstalled by install.sh)
        rm -rf "$SWAT_HOME/blueprints/squads/_framework" 2>/dev/null && ok "Removed blueprints/squads/_framework/"
        rm -f "$SWAT_HOME/blueprints/OPERATION.md" 2>/dev/null && ok "Removed blueprints/OPERATION.md"
        # Keep user-installed squads, skills, mcps
        info "Kept blueprints/squads/, blueprints/skills/, blueprints/mcps/ (use --purge to remove)"
    fi
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


echo ""
ok "SWAT uninstalled."
if ! $PURGE && [[ -d "$SWAT_HOME/squads" ]]; then
    info "To fully remove all data: rm -rf $SWAT_HOME"
fi
echo ""
