# Hive Phase C daemon supervisor for Windows boxes where Task Scheduler is
# unavailable (non-admin / policy-locked: even a trivial schtasks /Create returns
# "Access is denied"). Launched hidden at logon by the Startup-folder .vbs that
# setup-windows.ps1 provisions. Relaunches `hive serve` while it exits non-zero;
# a clean exit 0 (singleton-decline / drift-triggered stop) ends the loop --
# mirroring the Scheduled Task supervisor semantics (ADR-015 / hive#252).
# The daemon reads HIVE_VAULT_PATH from the User-scope env (set by setup).
$ErrorActionPreference = 'SilentlyContinue'
$hive = (Get-Command hive -ErrorAction SilentlyContinue).Source
if (-not $hive) { exit 0 }
while ($true) {
    & $hive serve
    if ($LASTEXITCODE -eq 0) { break }
    Start-Sleep -Seconds 1
}
