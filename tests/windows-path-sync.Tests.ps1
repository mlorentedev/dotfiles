# Pester 5 tests for Sync-SessionPath in scripts/utils.ps1 (TEST-003/#1298,
# #1308): the one PATH refresh setup-windows.ps1 runs after its installers.
# Each case runs in a CHILD pwsh so the process PATH under test is chosen by
# the test and the test runner's own PATH is never mutated. Windows only: the
# Machine/User registry scopes and the ';' separator are Windows concepts.

$script:onWindows = $env:OS -eq 'Windows_NT'

Describe 'Sync-SessionPath' -Skip:(-not $script:onWindows) {

    BeforeAll {
        $script:Utils = (Resolve-Path (Join-Path $PSScriptRoot '..\scripts\utils.ps1')).Path
        # Fixture: a process-only entry, a registry entry repeated in three
        # spellings, and the process-only entry again.
        $script:Probe = @'
. '__UTILS__'
$env:PATH = 'C:\only-in-process;C:\Windows;c:\windows\;C:\ONLY-IN-PROCESS'
Sync-SessionPath
$first = $env:PATH
$entries = @($env:PATH -split ';' | Where-Object { $_ })
$keys = @($entries | ForEach-Object { $_.TrimEnd('\').ToLowerInvariant() })
$machineFirst = @(([Environment]::GetEnvironmentVariable('PATH', 'Machine') -split ';') | Where-Object { $_ } | Select-Object -First 1)
Sync-SessionPath
[pscustomobject]@{
    Entries      = $entries.Count
    Duplicates   = ($keys.Count - @($keys | Sort-Object -Unique).Count)
    ProcessKept  = ($keys -contains 'c:\only-in-process')
    RegistryFirst = ($machineFirst.Count -eq 0 -or $entries[0] -eq $machineFirst[0])
    Idempotent   = ($env:PATH -eq $first)
} | ConvertTo-Json -Compress
'@.Replace('__UTILS__', $script:Utils)
        $script:Result = (& pwsh -NoProfile -Command $script:Probe 2>&1 | Select-Object -Last 1) | ConvertFrom-Json
    }

    It 'keeps an entry that exists only in the process PATH' {
        $script:Result.ProcessKept | Should -BeTrue
    }

    It 'emits no duplicate entries, comparing case-insensitively and ignoring a trailing backslash' {
        $script:Result.Duplicates | Should -Be 0
    }

    It 'puts the registry (Machine) PATH first so a fresh install beats a stale process entry' {
        $script:Result.RegistryFirst | Should -BeTrue
    }

    It 'is idempotent: a second call leaves PATH byte-identical' {
        $script:Result.Idempotent | Should -BeTrue
    }

    It 'a plain concatenation would have grown here; the merge does not' {
        # Every registry entry appears once, plus the two process-only spellings
        # collapsed to one -- never registry-count times two.
        $registry = @((([Environment]::GetEnvironmentVariable('PATH', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('PATH', 'User')) -split ';') | Where-Object { $_ } | ForEach-Object { $_.TrimEnd('\').ToLowerInvariant() } | Sort-Object -Unique)
        $script:Result.Entries | Should -BeLessOrEqual ($registry.Count + 2)
    }
}
