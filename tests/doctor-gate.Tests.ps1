# Pester 5 tests for .github/scripts/doctor-gate.ps1 (TEST-003, #1298): the
# classification the gate makes over dotf doctor output and its known-failures
# list. Pure function tests; the script's runtime half (PATH refresh, running
# doctor) is exercised by the test-windows job itself.

Describe 'Test-DoctorGate' {

    BeforeAll {
        . (Join-Path $PSScriptRoot '..\.github\scripts\doctor-gate.ps1')
        $script:Doctor = @(
            '[Core tools in PATH]',
            '  [FAIL] wget not in PATH',
            '  [WARN] something advisory',
            '  [FAIL] SCRIPTS_DIR=C:\Users\x\.dotfiles\scripts (path does not exist)',
            'Results: 90 passed, 2 failed, 1 warned, 40 skipped'
        )
    }

    It 'passes when every FAIL is known and every entry matches' {
        $r = Test-DoctorGate -Lines $script:Doctor -Patterns @('wget not in PATH', 'SCRIPTS_DIR=.*path does not exist')
        $r.Failures.Count | Should -Be 2
        $r.Unexpected.Count | Should -Be 0
        $r.Stale.Count | Should -Be 0
    }

    It 'reports a FAIL no entry covers as unexpected' {
        $r = Test-DoctorGate -Lines $script:Doctor -Patterns @('wget not in PATH')
        $r.Unexpected | Should -HaveCount 1
        $r.Unexpected[0] | Should -Match 'SCRIPTS_DIR'
    }

    It 'reports an entry that matches nothing as stale, so the list cannot rot' {
        $r = Test-DoctorGate -Lines $script:Doctor -Patterns @('wget not in PATH', 'SCRIPTS_DIR=.*path does not exist', 'eza not in PATH')
        $r.Stale | Should -HaveCount 1
        $r.Stale[0] | Should -Be 'eza not in PATH'
    }

    It 'ignores WARN lines: only [FAIL] lines gate' {
        $r = Test-DoctorGate -Lines @('  [WARN] only advisory', 'Results: 1 passed, 0 failed, 1 warned, 0 skipped') -Patterns @()
        $r.Failures.Count | Should -Be 0
        $r.Unexpected.Count | Should -Be 0
    }

    It 'a clean doctor with an empty list is a pass' {
        $r = Test-DoctorGate -Lines @('Results: 5 passed, 0 failed, 0 warned, 0 skipped') -Patterns @()
        $r.Unexpected.Count + $r.Stale.Count | Should -Be 0
    }
}

Describe 'Read-KnownFailures' {

    BeforeAll {
        . (Join-Path $PSScriptRoot '..\.github\scripts\doctor-gate.ps1')
    }

    It 'skips comments and blank lines and trims entries' {
        $p = Join-Path $TestDrive 'known.txt'
        Set-Content -Path $p -Value @('# ticket #1: reason', '', '  wget not in PATH  ', '# another', 'SCRIPTS_DIR=.*exist')
        $entries = Read-KnownFailures -Path $p
        $entries | Should -Be @('wget not in PATH', 'SCRIPTS_DIR=.*exist')
    }

    It 'returns an empty list when the file is absent' {
        (Read-KnownFailures -Path (Join-Path $TestDrive 'nope.txt')).Count | Should -Be 0
    }

    It 'every committed entry is a valid regex and names its ticket in a preceding comment' {
        $known = Join-Path $PSScriptRoot '..\.github\scripts\doctor-gate-known-failures.txt'
        $lines = @(Get-Content -LiteralPath $known)
        $previousComment = $false
        foreach ($line in $lines) {
            $t = $line.Trim()
            if (-not $t) { $previousComment = $false; continue }
            if ($t.StartsWith('#')) { $previousComment = $previousComment -or ($t -match '#\d+'); continue }
            { [regex]::new($t) } | Should -Not -Throw
            $previousComment | Should -BeTrue -Because "entry '$t' must be preceded by a comment naming its owning ticket (#NNN)"
            $previousComment = $false
        }
    }
}
