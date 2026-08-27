# Pester 5 tests for the console-encoding block and the LF writer in
# scripts/utils.ps1 (WIN-009/#1290, WIN-008/#1289).
#
# The encoding cases run in a CHILD pwsh so the code page under test is chosen
# by the test, never inherited from the runner (a runner already on UTF-8 would
# pass vacuously). Windows only: code pages are a Windows console concept.

$script:onWindows = $env:OS -eq 'Windows_NT'

Describe 'scripts/utils.ps1 console encoding' -Skip:(-not $script:onWindows) {

    BeforeAll {
        $script:Utils = (Resolve-Path (Join-Path $PSScriptRoot '..\scripts\utils.ps1')).Path
        $script:Profile = (Resolve-Path (Join-Path $PSScriptRoot '..\powershell\profile.ps1')).Path
    }

    It 'flips a child pwsh that starts on code page 437 to UTF-8 when dot-sourced' {
        $probe = "[Console]::OutputEncoding = [Text.Encoding]::GetEncoding(437); . '$script:Utils'; [Console]::OutputEncoding.CodePage; `$OutputEncoding.CodePage"
        $out = @(& pwsh -NoProfile -Command $probe 2>&1 | Select-Object -Last 2)
        $out | Should -Be @('65001', '65001')
    }

    It 'lets a captured native em dash survive that used to arrive as OEM glyphs' {
        # The native child prints U+2014 as UTF-8; the parent captures it once on
        # code page 437 (mojibake) and once after sourcing utils.ps1 (intact).
        $inner = '[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false); [Console]::Out.Write([string][char]0x2014)'
        $probe = @"
[Console]::OutputEncoding = [Text.Encoding]::GetEncoding(437)
`$before = (& pwsh -NoProfile -Command '$inner' | Out-String).Trim()
. '$script:Utils'
`$after = (& pwsh -NoProfile -Command '$inner' | Out-String).Trim()
"`$(`$before -eq [string][char]0x2014)|`$(`$after -eq [string][char]0x2014)"
"@
        $out = (& pwsh -NoProfile -Command $probe 2>&1 | Select-Object -Last 1)
        $out | Should -Be 'False|True'
    }

    It 'is mirrored by the deployed profile' {
        Get-Content -LiteralPath $script:Profile -Raw | Should -Match '\[Console\]::OutputEncoding\s*='
    }
}

Describe 'Write-Utf8LfFile' -Skip:(-not $script:onWindows) {

    BeforeAll {
        . (Resolve-Path (Join-Path $PSScriptRoot '..\scripts\utils.ps1')).Path
    }

    It 'writes LF line endings and no BOM from CRLF input' {
        $p = Join-Path $TestDrive 'a.md'
        Write-Utf8LfFile -Path $p -Content "one`r`ntwo`r`n"
        $bytes = [IO.File]::ReadAllBytes($p)
        ($bytes -contains 13) | Should -BeFalse
        $bytes[0] | Should -Not -Be 0xEF
        [Text.Encoding]::UTF8.GetString($bytes) | Should -Be "one`ntwo`n"
    }

    It 'ends the file with exactly one LF' {
        $p = Join-Path $TestDrive 'b.md'
        Write-Utf8LfFile -Path $p -Content 'x'
        [IO.File]::ReadAllText($p) | Should -Be "x`n"
        Write-Utf8LfFile -Path $p -Content "x`n"
        [IO.File]::ReadAllText($p) | Should -Be "x`n"
    }

    It 'creates the parent directory' {
        $p = Join-Path $TestDrive 'nested\deeper\c.md'
        Write-Utf8LfFile -Path $p -Content 'y'
        Test-Path -LiteralPath $p | Should -BeTrue
    }

    It 'is what setup uses for every rendered .md it writes' {
        $setup = Get-Content -LiteralPath (Join-Path $PSScriptRoot '..\setup-windows.ps1') -Raw
        ([regex]::Matches($setup, 'Write-Utf8LfFile -Path')).Count | Should -BeGreaterOrEqual 3
        $setup | Should -Not -Match 'Set-Content -LiteralPath \$catFile'
    }
}
