# Reports whether the hive auto-upgrade trigger seed-hive-upgrade-task.ps1
# planted is still on the guest: the task and its script copy.
$task = Get-ScheduledTask -TaskName 'DotfilesHiveUpgrade' -ErrorAction SilentlyContinue
$script = Join-Path $env:USERPROFILE '.claude\scripts\hive-upgrade.ps1'
"task=$([bool]$task) script=$(Test-Path $script)"
