# Seeds a Windows guest with the hive auto-upgrade trigger that the old
# setup-windows.ps1 installed (deleted in ec5170ff, #2876), so the converge step
# hive-upgrade-retire has something to remove. A clean guest never gets it: the
# old setup registered the task only when `Get-Command hive` resolved (AI-028
# verification.md, AC9).
#
# The task matches the old registration: task name, powershell.exe action with
# the same arguments, a 15-minute repetition, and a principal for $env:USERNAME
# whose logon type is S4U when elevated, Interactive otherwise. Run it with
# `dotf vm windows exec --elevated --script` to get the S4U task an elevated
# setup left behind.
$ErrorActionPreference = 'Stop'

$elevated = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
$logonType = if ($elevated) { 'S4U' } else { 'Interactive' }

$script = Join-Path $env:USERPROFILE '.claude\scripts\hive-upgrade.ps1'
New-Item -ItemType Directory -Force -Path (Split-Path $script -Parent) | Out-Null
Set-Content -Path $script -Value '# hive-upgrade.ps1 stand-in seeded for the hive-upgrade-retire check'

$arg = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$script`""
$action = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $arg
$trigger = New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Minutes 15)
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType $logonType -RunLevel Limited
Register-ScheduledTask -TaskName 'DotfilesHiveUpgrade' -Action $action -Trigger $trigger `
    -Settings $settings -Principal $principal -Description 'seeded for the hive-upgrade-retire check' -Force | Out-Null

$t = Get-ScheduledTask -TaskName 'DotfilesHiveUpgrade'
"seeded DotfilesHiveUpgrade logon=$($t.Principal.LogonType) script=$(Test-Path $script)"
