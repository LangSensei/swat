# SWAT Uninstaller for Windows
# Usage: irm https://raw.githubusercontent.com/LangSensei/swat/main/uninstall.ps1 | iex

$ErrorActionPreference = "Stop"

$SwatHome = Join-Path $env:USERPROFILE ".swat"
$BinDir = Join-Path $env:USERPROFILE ".swat\bin"

function Info  { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Cyan }
function Ok    { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Green }
function Warn  { param($Msg) Write-Host "[swat] $Msg" -ForegroundColor Yellow }

$Purge = $args -contains "--purge"
$Yes = $args -contains "--yes"

Write-Host ""
Write-Host "  SWAT Uninstaller"
Write-Host "  ==================="
Write-Host ""

if (-not $Yes) {
    Warn "This will remove:"
    Write-Host "  - Binary:     $BinDir\swat.exe"
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
    if ((Test-Path $SwatHome) -and ((Get-ChildItem $SwatHome -Force).Count -eq 0)) {
        Remove-Item -Force $SwatHome
        Ok "Removed $SwatHome"
    }
} else {
    if (Test-Path (Join-Path $SwatHome "squads")) {
        Info "Runtime data kept at $SwatHome\squads\ (use --purge to remove)"
    }
}

# --- Remove from user PATH ---

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -like "*$BinDir*") {
    $newPath = ($userPath -split ';' | Where-Object { $_ -ne $BinDir }) -join ';'
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Ok "Removed $BinDir from user PATH"
}

Write-Host ""
Ok "SWAT uninstalled."
if (-not $Purge -and (Test-Path (Join-Path $SwatHome "squads"))) {
    Info "To fully remove all data: Remove-Item -Recurse -Force $SwatHome"
}
Write-Host ""
