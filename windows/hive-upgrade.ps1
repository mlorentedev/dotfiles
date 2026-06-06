<#
.SYNOPSIS
    Windows-safe hive-vault auto-upgrade (ADR-015 / hive#176).

.DESCRIPTION
    Windows cannot replace a running executable, and the Phase C daemon holds
    its own hive.exe, so a plain `uv tool upgrade hive-vault` fails (os error 32)
    and a `--reinstall` corrupts a locked venv (os error 5). This orchestrates a
    clean, atomic-or-deferred swap:

      0. Only act when a NEWER version is actually published. Otherwise return
         immediately and leave the daemon untouched -- at a 15-minute cadence
         the overwhelming majority of runs are this fast no-op, so the daemon is
         never restarted just to check for updates.
      1. If a hive process OTHER than the daemon holds the uv-tool install
         (an open `hive client` session), DEFER this cycle. The daemon keeps
         running the current version; the upgrade lands the next cycle the
         install is free. Never a partial / half-version venv.
      2. Otherwise stop the daemon (releases its locks), run a PLAIN
         `uv tool upgrade hive-vault` (never `--reinstall`), and start the
         daemon (its supervisor relaunches onto the new version).

    Runs from PowerShell -- the Task Scheduler host -- NOT from the hive install,
    so this orchestrator holds no lock on the files it is replacing.

    The residual OS limitation (a release that changes a loaded native .pyd while
    a session is open) is tracked upstream in uv (#8528, #11930, #11134), not
    reimplemented here; defer-if-locked sidesteps it.
#>

[CmdletBinding()]
param(
    [string] $DaemonTask = 'HiveVaultDaemon'
)

$ErrorActionPreference = 'Continue'
$uv = Join-Path $env:USERPROFILE '.local\bin\uv.exe'
$toolsGlob = '*\uv\tools\hive-vault\*'

if (-not (Test-Path $uv)) {
    Write-Output "hive-upgrade: uv not found at $uv; nothing to do."
    exit 0
}

# Step 0: only proceed when a newer version is actually published. Comparing
# installed vs latest BEFORE touching the daemon means a 15-minute cadence does
# not restart the daemon on every tick -- only on an actual release.
$installedMatch = (& $uv tool list 2>$null | Select-String '^hive-vault\s+v?([\d.]+)')
$installed = if ($installedMatch) { $installedMatch.Matches[0].Groups[1].Value } else { $null }
$latest = $null
try { $latest = (Invoke-RestMethod 'https://pypi.org/pypi/hive-vault/json' -TimeoutSec 15).info.version } catch { $latest = $null }
if (-not $installed -or -not $latest -or ([version]$installed -ge [version]$latest)) {
    exit 0
}

# Step 1: resolve the daemon PID (the listener on the published loopback port)
# so it is excluded from the "is the install held by a client session?" check.
$daemonPid = $null
$portFile = Join-Path $env:USERPROFILE '.local\share\hive\daemon.port'
if (Test-Path $portFile) {
    $port = (Get-Content $portFile -Raw -ErrorAction SilentlyContinue).Trim()
    if ($port) {
        $conn = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
        if ($conn) { $daemonPid = ($conn.OwningProcess | Select-Object -First 1) }
    }
}

# A hive/python process (other than the daemon) running from the uv-tool install
# is an open client session holding the files -> defer.
$holders = Get-Process -ErrorAction SilentlyContinue | Where-Object {
    $_.Id -ne $daemonPid -and (& { try { $_.Path -like $toolsGlob } catch { $false } })
}
if ($holders) {
    Write-Output "hive-upgrade: $installed -> $latest available but install held by $($holders.Count) client session(s); deferring."
    exit 0
}

# Step 2: free to upgrade. Stop the daemon, swap cleanly, restart.
Write-Output "hive-upgrade: upgrading $installed -> $latest."
Stop-ScheduledTask -TaskName $DaemonTask -ErrorAction SilentlyContinue
& $uv tool upgrade hive-vault
$rc = $LASTEXITCODE
Start-ScheduledTask -TaskName $DaemonTask -ErrorAction SilentlyContinue

if ($rc -ne 0) {
    Write-Output "hive-upgrade: uv tool upgrade exited $rc (daemon restarted regardless)."
}
exit $rc
