# SWAT v2 Uninstaller for Windows
# Usage: irm https://raw.githubusercontent.com/LangSensei/swat-v2/main/uninstall.ps1 | iex

$ErrorActionPreference = "Stop"

$SwatHome = Join-Path $env:USERPROFILE ".swat"
$BinDir = Join-Path $env:USERPROFILE ".local\bin"

# --- Helpers ---

function Info  { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Cyan }
function Ok    { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Green }
function Warn  { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Yellow }
function Err   { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Red }

$Purge = $args -contains "--purge"
$Yes = $args -contains "--yes"

Write-Host ""
Write-Host "  SWAT v2 Uninstaller"
Write-Host "  ==================="
Write-Host ""

# --- Confirm ---

if (-not $Yes) {
    Warn "This will remove:"
    Write-Host "  - Binary:     $BinDir\swat.exe"
    Write-Host "  - Plugin:     $SwatHome\plugin\"
    Write-Host "  - Framework:  $SwatHome\blueprints\squads\_framework\, OPERATION.md"
    Write-Host ""
    Warn "User data (squads, skills, mcps, schedules, operations) will be KEPT unless you pass --purge"
    Write-Host ""
    $confirm = Read-Host "Continue? [y/N]"
    if ($confirm -notin @("y", "Y")) {
        Info "Aborted."
        exit 0
    }
}

# --- Remove binary ---

$binPath = Join-Path $BinDir "swat.exe"
if (Test-Path $binPath) {
    Remove-Item -Force $binPath
    Ok "Removed $binPath"
} else {
    Info "Binary not found at $binPath (skipped)"
}

# --- Remove plugin ---

$pluginDir = Join-Path $SwatHome "plugin"
if (Test-Path $pluginDir) {
    Remove-Item -Recurse -Force $pluginDir
    Ok "Removed $pluginDir"
}

# --- Remove blueprints ---

$bpDir = Join-Path $SwatHome "blueprints"
if (Test-Path $bpDir) {
    if ($Purge) {
        Remove-Item -Recurse -Force $bpDir
        Ok "Purged $bpDir"
    } else {
        $fwDir = Join-Path $bpDir "squads\_framework"
        if (Test-Path $fwDir) {
            Remove-Item -Recurse -Force $fwDir
            Ok "Removed blueprints\squads\_framework\"
        }
        $opMd = Join-Path $bpDir "OPERATION.md"
        if (Test-Path $opMd) {
            Remove-Item -Force $opMd
            Ok "Removed blueprints\OPERATION.md"
        }
        Info "Kept blueprints\squads\, blueprints\skills\, blueprints\mcps\ (use --purge to remove)"
    }
}

# --- Purge runtime data ---

if ($Purge) {
    foreach ($sub in @("squads", "schedules")) {
        $d = Join-Path $SwatHome $sub
        if (Test-Path $d) {
            Remove-Item -Recurse -Force $d
            Ok "Purged $d"
        }
    }
    # Remove .swat if empty
    if ((Test-Path $SwatHome) -and ((Get-ChildItem $SwatHome -Force).Count -eq 0)) {
        Remove-Item -Force $SwatHome
        Ok "Removed $SwatHome"
    }
} else {
    if (Test-Path (Join-Path $SwatHome "squads")) {
        Info "Runtime data kept at $SwatHome\squads\ (use --purge to remove)"
    }
}

# --- Remove skill ---

$candidates = @(
    (Join-Path $env:APPDATA "npm\node_modules\openclaw\skills\swat"),
    (Join-Path $env:USERPROFILE ".npm-global\lib\node_modules\openclaw\skills\swat")
)
foreach ($dir in $candidates) {
    if (Test-Path $dir) {
        Remove-Item -Recurse -Force $dir
        Ok "Removed skill from $dir"
        break
    }
}

# --- Remove from user PATH ---

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -like "*$BinDir*") {
    $newPath = ($userPath -split ';' | Where-Object { $_ -ne $BinDir }) -join ';'
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Ok "Removed $BinDir from user PATH"
}

# --- Remove OpenClaw plugin config ---

$ocConfig = Join-Path $env:USERPROFILE ".openclaw\openclaw.json"
if ((Test-Path $ocConfig) -and (Get-Command node -ErrorAction SilentlyContinue)) {
    try {
        node -e @"
const fs = require('fs');
const cfg = JSON.parse(fs.readFileSync('$($ocConfig -replace '\\', '/')', 'utf8'));
let changed = false;
if (cfg.plugins?.load?.paths) {
    const before = cfg.plugins.load.paths.length;
    cfg.plugins.load.paths = cfg.plugins.load.paths.filter(p => !p.includes('.swat/plugin') && !p.includes('.swat\\plugin'));
    if (cfg.plugins.load.paths.length < before) changed = true;
    if (cfg.plugins.load.paths.length === 0) delete cfg.plugins.load.paths;
}
if (cfg.plugins?.entries?.['swat-mcp-bridge']) {
    delete cfg.plugins.entries['swat-mcp-bridge'];
    changed = true;
}
if (changed) {
    fs.writeFileSync('$($ocConfig -replace '\\', '/')', JSON.stringify(cfg, null, 2) + '\n');
}
"@
        Ok "Cleaned OpenClaw config"
    } catch {
        Info "Could not auto-clean OpenClaw config"
    }
}

Write-Host ""
Ok "SWAT v2 uninstalled."
if (-not $Purge -and (Test-Path (Join-Path $SwatHome "squads"))) {
    Info "To fully remove all data: Remove-Item -Recurse -Force $SwatHome"
}
Write-Host ""
