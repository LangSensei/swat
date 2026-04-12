# SWAT v2 Installer for Windows
# Usage: irm https://raw.githubusercontent.com/LangSensei/swat-v2/main/install.ps1 | iex
#        powershell -File install.ps1 --openclaw   # also install OpenClaw plugin/skill

$ErrorActionPreference = "Stop"

$Repo = "LangSensei/swat-v2"
$SwatHome = Join-Path $env:USERPROFILE ".swat"
$BinDir = Join-Path $env:USERPROFILE ".local\bin"
$OpenClaw = $args -contains "--openclaw"

# --- Helpers ---

function Info  { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Cyan }
function Ok    { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Green }
function Err   { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Red }
function Warn  { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Yellow }

# --- Safety: refuse to run as SYSTEM ---
if ($env:USERNAME -eq "SYSTEM") {
    Err "Do not run this installer as SYSTEM."
    exit 1
}

# --- Detect Platform ---

function Detect-Platform {
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
    $script:Platform = "windows-$arch"
    Info "Detected platform: $script:Platform"
}

# --- Prerequisites ---

function Check-Prereqs {
    $missing = @()
    if (-not (Get-Command node -ErrorAction SilentlyContinue)) { $missing += "node" }
    if (-not (Get-Command npm -ErrorAction SilentlyContinue))  { $missing += "npm" }

    if ($missing.Count -gt 0) {
        Err "Missing prerequisites: $($missing -join ', ')"
        exit 1
    }

    if (-not (Get-Command copilot -ErrorAction SilentlyContinue)) {
        Warn "GitHub Copilot CLI not found. Required for running squads."
        Info "  npm install -g @github/copilot"
    }

    if (Get-Command gh -ErrorAction SilentlyContinue) {
        $authOk = gh auth status 2>&1
        if ($LASTEXITCODE -ne 0) {
            Warn "GitHub CLI not authenticated. Run before using SWAT:"
            Info "  gh auth login"
        }
    } else {
        Warn "GitHub CLI (gh) not found. Required for Copilot CLI auth."
        Info "  Install: https://cli.github.com"
    }
}

# --- Download & Extract ---

function Fetch-Release {
    $apiUrl = "https://api.github.com/repos/$Repo/releases/latest"
    $release = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
    $tag = $release.tag_name

    if (-not $tag) {
        Err "Failed to fetch latest release"
        exit 1
    }

    $script:Tag = $tag
    Info "Latest release: $tag"

    # Check if already installed at this version
    try {
        $current = & (Join-Path $BinDir "swat.exe") --version 2>$null
        if ($current -match $tag) {
            Ok "Already up to date ($tag)"
            exit 0
        }
    } catch {}

    $zipName = "swat-${tag}-${script:Platform}.zip"
    $dlUrl = "https://github.com/$Repo/releases/download/${tag}/${zipName}"
    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "swat-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    Info "Downloading $zipName..."
    Invoke-WebRequest -Uri $dlUrl -OutFile (Join-Path $tmpDir $zipName) -UseBasicParsing

    Info "Extracting..."
    Expand-Archive -Path (Join-Path $tmpDir $zipName) -DestinationPath $tmpDir -Force

    $script:ExtractDir = $tmpDir
}

# --- Install ---

function Install-Binary {
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    Copy-Item (Join-Path $script:ExtractDir "swat.exe") (Join-Path $BinDir "swat.exe") -Force
    Ok "Binary installed to $BinDir\swat.exe"
}

function Install-Plugin {
    $dest = Join-Path $SwatHome "plugin"
    if (Test-Path $dest) { Remove-Item -Recurse -Force $dest }
    New-Item -ItemType Directory -Path $dest -Force | Out-Null
    Copy-Item (Join-Path $script:ExtractDir "plugin\*") $dest -Recurse -Force

    Info "Installing plugin dependencies..."
    Push-Location $dest
    npm install --quiet 2>$null
    Pop-Location
    Ok "Plugin installed to $dest"
}

function Install-Skill {
    $ocSkills = $null
    $candidates = @(
        (Join-Path $env:APPDATA "npm\node_modules\openclaw\skills"),
        (Join-Path $env:USERPROFILE ".npm-global\lib\node_modules\openclaw\skills")
    )
    foreach ($dir in $candidates) {
        if (Test-Path $dir) { $ocSkills = $dir; break }
    }

    if (-not $ocSkills) {
        Info "OpenClaw skills directory not found. Manually copy skill\SKILL.md to your OpenClaw skills dir."
        return
    }

    $skillDir = Join-Path $ocSkills "swat"
    New-Item -ItemType Directory -Path $skillDir -Force | Out-Null
    Copy-Item (Join-Path $script:ExtractDir "skill\SKILL.md") $skillDir -Force
    Ok "Skill installed to $skillDir"
}

function Install-Blueprints {
    $bp = Join-Path $SwatHome "blueprints"
    $fwDir = Join-Path $bp "squads\_framework"
    foreach ($d in @($fwDir, (Join-Path $bp "skills"), (Join-Path $bp "mcps"))) {
        New-Item -ItemType Directory -Path $d -Force | Out-Null
    }

    Copy-Item (Join-Path $script:ExtractDir "blueprints\OPERATION.md") $bp -Force
    Copy-Item (Join-Path $script:ExtractDir "blueprints\squads\_framework\*") $fwDir -Force

    Ok "Framework blueprints installed"
    Info "Install squads/skills/mcps from the marketplace:"
    Write-Host "  https://github.com/LangSensei/swat-marketplace"
}

function Setup-Runtime {
    New-Item -ItemType Directory -Path (Join-Path $SwatHome "squads") -Force | Out-Null
}

# --- Post-Install ---

function Register-Plugin {
    $ocConfig = Join-Path $env:USERPROFILE ".openclaw\openclaw.json"
    $pluginPath = (Join-Path $SwatHome "plugin") -replace '\\', '/'

    if (-not (Test-Path $ocConfig)) {
        Info "OpenClaw config not found at $ocConfig"
        Info "After installing OpenClaw, add the SWAT plugin to your config:"
        Write-Host ""
        Write-Host "  plugins.load.paths: [`"$pluginPath`"]"
        Write-Host "  plugins.entries.swat-mcp-bridge.enabled: true"
        Write-Host ""
        Info "Then restart OpenClaw: openclaw gateway restart"
        return
    }

    $binPath = (Join-Path $BinDir "swat.exe") -replace '\\', '/'

    try {
        node -e @"
const fs = require('fs');
const cfg = JSON.parse(fs.readFileSync('$($ocConfig -replace '\\', '/')', 'utf8'));
cfg.plugins = cfg.plugins || {};
cfg.plugins.load = cfg.plugins.load || {};
cfg.plugins.load.paths = cfg.plugins.load.paths || [];
cfg.plugins.entries = cfg.plugins.entries || {};
if (!cfg.plugins.load.paths.includes('$pluginPath')) {
    cfg.plugins.load.paths.push('$pluginPath');
}
cfg.plugins.entries['swat-mcp-bridge'] = {
    enabled: true,
    config: { binaryPath: '$binPath' }
};
fs.writeFileSync('$($ocConfig -replace '\\', '/')', JSON.stringify(cfg, null, 2) + '\n');
"@
        Ok "Plugin registered in OpenClaw config"
        Info "Restart OpenClaw to activate: openclaw gateway restart"
    } catch {
        Err "Failed to auto-register plugin. Manually add to $ocConfig"
    }
}

function Post-Install {
    # PATH check — add to user PATH if missing
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$BinDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$BinDir;$userPath", "User")
        $env:Path = "$BinDir;$env:Path"
        Ok "Added $BinDir to user PATH"
    }

    if ($OpenClaw) {
        Register-Plugin
    }
}

# --- Cleanup ---

function Cleanup {
    if ($script:ExtractDir -and (Test-Path $script:ExtractDir)) {
        Remove-Item -Recurse -Force $script:ExtractDir -ErrorAction SilentlyContinue
    }
}

# --- Main ---

Write-Host ""
Info "Installing SWAT v2..."
Write-Host ""

Detect-Platform
Check-Prereqs
Fetch-Release
Install-Binary
if ($OpenClaw) {
    Install-Plugin
    Install-Skill
}
Install-Blueprints
Setup-Runtime
Post-Install
Cleanup

Write-Host ""
Ok "SWAT v2 installed successfully! 🚀"
Write-Host ""
Info "Next steps:"
if ($OpenClaw) {
    Write-Host "  1. Restart OpenClaw:  openclaw gateway restart"
    Write-Host "  2. Install a squad:   Tell your agent: `"browse SWAT marketplace and install a squad`""
} else {
    Write-Host "  1. Add to Copilot CLI .mcp.json:"
    Write-Host "     {`"mcpServers`":{`"swat`":{`"command`":`"$($BinDir -replace '\\','/')/swat.exe`",`"args`":[`"mcp`"]}}}"
    Write-Host "  2. Install a squad:   swat install <squad-name>"
}
Write-Host ""
