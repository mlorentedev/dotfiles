# Reports whether the hive auto-upgrade trigger seed-hive-upgrade-task.ps1
# planted is still on the guest: the task and its script copy. The task list
# is read with terminating errors, so a scheduler that cannot answer fails the
# script rather than reading as task=False.
$ErrorActionPreference = 'Stop'
$task = Get-ScheduledTask | Where-Object TaskName -eq 'DotfilesHiveUpgrade'
$script = Join-Path $env:USERPROFILE '.claude\scripts\hive-upgrade.ps1'
"task=$([bool]$task) script=$(Test-Path $script)"
