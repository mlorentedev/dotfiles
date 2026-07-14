# vault-maintenance-weekly.ps1: Automated weekly vault maintenance
#
# Runs knowledge-crystallize.ps1 (all projects) + basic health checks.
# Logs results and sends toast notification (best-effort).
#
# Deployed to Task Scheduler by setup-windows.ps1: Sundays 10:00 AM
# Usage: .\scripts\vault-maintenance-weekly.ps1

$ErrorActionPreference = "Continue"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$LogDir = "$env:LOCALAPPDATA\vault-maintenance"
$LogFile = "$LogDir\latest.log"

New-Item -ItemType Directory -Path $LogDir -Force | Out-Null

$output = @()
$output += "=== Vault Maintenance: $(Get-Date) ==="
$output += ""

# Step 1: MEMORY.md maintenance across all projects.
# -All is a PowerShell [switch]; the POSIX-style "--all" bound positionally to
# $ProjectDir and left $All false, so this task never fanned out (#689 / C5).
$output += "--- knowledge-crystallize -All ---"
try {
    $crystOutput = & "$ScriptDir\knowledge-crystallize.ps1" -All 2>&1
    $output += $crystOutput
} catch {
    $output += "Error: $_"
}
$output += ""

$output += "=== Done: $(Get-Date) ==="

$output | Out-File -FilePath $LogFile -Encoding UTF8

# Count issues
$issues = ($output | Select-String -Pattern "WARNING|FAIL|ACTION|STALE" -CaseSensitive:$false).Count

# Toast notification (best-effort)
try {
    Add-Type -AssemblyName System.Windows.Forms -ErrorAction Stop
    $balloon = New-Object System.Windows.Forms.NotifyIcon
    $balloon.Icon = [System.Drawing.SystemIcons]::Information
    $balloon.BalloonTipTitle = "Vault Maintenance"
    if ($issues -gt 0) {
        $balloon.BalloonTipText = "$issues potential issues. Run /insights on active projects."
    } else {
        $balloon.BalloonTipText = "All clean. No action needed."
    }
    $balloon.Visible = $true
    $balloon.ShowBalloonTip(10000)
    Start-Sleep -Seconds 11
    $balloon.Dispose()
} catch {
    Write-Host "Vault Maintenance: $issues issues found. See $LogFile"
}

Write-Host "Log written to $LogFile"
