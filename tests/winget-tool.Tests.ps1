# Pester 5 tests for Install-WingetTool in scripts/utils.ps1 (#2157): the winget
# install setup-windows.ps1 runs for each developer tool. Each case runs in a
# CHILD pwsh where `winget` is a function standing in for winget.exe (a function
# wins over an application of the same name), so no package is ever installed
# and the runner's PATH is never mutated. Windows only, like Sync-SessionPath,
# which the function calls between attempts.

$script:onWindows = $env:OS -eq 'Windows_NT'

Describe 'Install-WingetTool' -Skip:(-not $script:onWindows) {

    BeforeAll {
        $script:Utils = (Resolve-Path (Join-Path $PSScriptRoot '..\scripts\utils.ps1')).Path
        # __WINGET__ is the body of the stand-in; it may read $script:calls (the
        # attempt number) and define the tool's command to model a success.
        $script:Run = {
            param([string]$WingetBody)
            $probe = @'
. '__UTILS__'
$script:calls = 0
function winget { $script:calls++; __WINGET__ }
$r = Install-WingetTool -Cmd 'dotfiles-test-tool-2157' -Id 'Example.Tool'
[pscustomobject]@{
    Installed = $r.Installed; Attempts = $r.Attempts; ExitCode = $r.ExitCode
    Detail = $r.Detail; Calls = $script:calls
} | ConvertTo-Json -Compress
'@.Replace('__UTILS__', $script:Utils).Replace('__WINGET__', $WingetBody)
            (& pwsh -NoProfile -Command $probe 2>&1 | Select-Object -Last 1) | ConvertFrom-Json
        }
    }

    It 'reports a failed install as not installed, with the exit code and winget''s last words' {
        $r = & $script:Run '$global:LASTEXITCODE = 1; "Searching sources..."; "No package found matching input criteria."'
        $r.Installed | Should -BeFalse
        $r.ExitCode | Should -Be 1
        $r.Detail | Should -Match 'No package found matching input criteria'
    }

    It 'retries once before giving up' {
        $r = & $script:Run '$global:LASTEXITCODE = 1; "transient"'
        $r.Calls | Should -Be 2
        $r.Attempts | Should -Be 2
    }

    It 'judges success by the command resolving, not by winget''s exit code' {
        # winget says 0 but the command never appears: still not installed.
        $r = & $script:Run '$global:LASTEXITCODE = 0; "Successfully installed"'
        $r.Installed | Should -BeFalse
    }

    It 'reports an already-installed package as installed when its command resolves' {
        # #2157 asks for winget's already-installed exit code to count as
        # success. Resolution decides, so it does with no code-specific branch:
        # 0x8A150061 (APPINSTALLER_CLI_ERROR_PACKAGE_ALREADY_INSTALLED) here.
        $r = & $script:Run 'function global:dotfiles-test-tool-2157 { } ; $global:LASTEXITCODE = -1978335135; "Found an existing package already installed."'
        $r.Installed | Should -BeTrue
        $r.Calls | Should -Be 1
    }

    It 'succeeds on the retry when the first attempt failed transiently' {
        $body = 'if ($script:calls -ge 2) { function global:dotfiles-test-tool-2157 { } ; $global:LASTEXITCODE = 0 } else { $global:LASTEXITCODE = 1 }'
        $r = & $script:Run $body
        $r.Installed | Should -BeTrue
        $r.Attempts | Should -Be 2
    }

    It 'stops after the first attempt when it works' {
        $r = & $script:Run 'function global:dotfiles-test-tool-2157 { } ; $global:LASTEXITCODE = 0'
        $r.Installed | Should -BeTrue
        $r.Calls | Should -Be 1
    }
}
