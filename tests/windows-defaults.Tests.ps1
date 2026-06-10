# Pester 5 behavioral tests for scripts/windows-defaults.ps1 (WIN-005).
#
# Runs only on Windows (registry). All writes are redirected under a throwaway
# sandbox key via the script's -Root test seam, so neither a dev box nor the
# CI runner has its real HKCU defaults mutated. WIN-004 lesson applied: the
# -Skip condition is computed at discovery time (top-level), NOT in BeforeAll.

$script:onWindows = $env:OS -eq 'Windows_NT'

Describe 'windows-defaults.ps1 (sandboxed registry behavior)' -Skip:(-not $script:onWindows) {

    BeforeAll {
        $script:Target = Join-Path $PSScriptRoot '..\scripts\windows-defaults.ps1'
        $script:SandboxRoot = "HKCU:\Software\dotfiles-win005-test-$PID"
    }

    AfterAll {
        if ($script:SandboxRoot -and (Test-Path $script:SandboxRoot)) {
            Remove-Item $script:SandboxRoot -Recurse -Force
        }
    }

    It 'applies every default on a clean root and reports the count' {
        $out = & $script:Target -Root $script:SandboxRoot *>&1 | Out-String
        $out | Should -Match '\[SET\]'
        $out | Should -Not -Match 'Applied 0'
    }

    It 'wrote show-file-extensions into the sandbox, not the real HKCU' {
        $key = "$script:SandboxRoot\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced"
        (Get-ItemProperty -Path $key -Name HideFileExt).HideFileExt | Should -Be 0
    }

    It 'is idempotent: second run applies nothing' {
        $out = & $script:Target -Root $script:SandboxRoot *>&1 | Out-String
        $out | Should -Match 'Applied 0'
        $out | Should -Not -Match '\[SET\]'
    }

    It 'rejects a root outside HKCU (runtime invariant)' {
        { & $script:Target -Root 'HKLM:\Software\should-never-happen' } | Should -Throw
    }
}
