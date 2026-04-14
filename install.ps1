# SWAT Installer for Windows
# Usage: irm https://raw.githubusercontent.com/LangSensei/swat/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "LangSensei/swat"
$SwatHome = Join-Path $env:USERPROFILE ".swat"
$BinDir = Join-Path $env:USERPROFILE ".swat\bin"

# --- Helpers ---

function Info  { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Cyan }
function Ok    { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Green }
function Err   { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Red }
function Warn  { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Yellow }

# --- Safety ---
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

    # Create default .env if it doesn't exist (existing files are never modified)
    $envFile = Join-Path $SwatHome ".env"
    if (-not (Test-Path $envFile)) {
        @"
# SWAT configuration
# Runtime: copilot | gemini
RUNTIME=copilot
"@ | Set-Content $envFile -Encoding UTF8
        Ok "Created $envFile"
    }
}

# --- Post-Install ---

function Post-Install {
    # PATH check — add to user PATH if missing
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$BinDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$BinDir;$userPath", "User")
        $env:Path = "$BinDir;$env:Path"
        Ok "Added $BinDir to user PATH"
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
Info "Installing SWAT..."
Write-Host ""

Detect-Platform
Check-Prereqs
Fetch-Release
Install-Binary
Install-Blueprints
Setup-Runtime
Post-Install
Cleanup

Write-Host ""
Ok "SWAT installed successfully! 🚀"
Write-Host ""
Info "Next steps:"
Write-Host "  1. Add SWAT MCP server to your agent config:"
Write-Host "     {`"mcpServers`":{`"swat`":{`"command`":`"swat`",`"args`":[]}}}"
Write-Host "  2. Change runtime: edit ~/.swat/.env (default: copilot)"
Write-Host "  3. For OpenClaw integration: https://github.com/LangSensei/swat-openclaw"
Write-Host ""
