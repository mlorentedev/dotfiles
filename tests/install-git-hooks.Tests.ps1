# Pester 5 tests for scripts/install-git-hooks.ps1 (BUG-032 / #691): the Windows
# twin of install-git-hooks.sh that deploys the GUARD-001 dispatcher and wires
# core.hooksPath. Fixtures live under $TestDrive (auto-cleaned); git config is
# isolated via a throwaway GIT_CONFIG_GLOBAL so the real ~/.gitconfig is never
# mutated.

BeforeAll {
    . (Join-Path $PSScriptRoot '..\scripts\install-git-hooks.ps1')
}

Describe 'Deploy-GitHooks (clean mirror + safety guards)' {
    BeforeEach {
        $script:src = Join-Path $TestDrive 'repo\git-hooks'
        New-Item -ItemType Directory -Path $script:src -Force | Out-Null
        Set-Content -Path (Join-Path $script:src 'pre-commit') -Value 'dispatcher'
        New-Item -ItemType Directory -Path (Join-Path $script:src 'lib') -Force | Out-Null
        Set-Content -Path (Join-Path $script:src 'lib\memory-sink-guard.sh') -Value 'guard'
        $script:dest = Join-Path $TestDrive 'deploy\git-hooks'
    }

    It 'clean-mirrors the dispatcher tree to a *\git-hooks dest' {
        Deploy-GitHooks -Source $script:src -Destination $script:dest | Should -BeTrue
        Test-Path (Join-Path $script:dest 'pre-commit') | Should -BeTrue
        Test-Path (Join-Path $script:dest 'lib\memory-sink-guard.sh') | Should -BeTrue
    }

    It 'prunes a hook removed upstream on re-deploy (clean mirror, not additive)' {
        Deploy-GitHooks -Source $script:src -Destination $script:dest | Should -BeTrue
        Set-Content -Path (Join-Path $script:dest 'stale-hook') -Value 'old'
        Deploy-GitHooks -Source $script:src -Destination $script:dest | Should -BeTrue
        Test-Path (Join-Path $script:dest 'stale-hook') | Should -BeFalse
    }

    It 'refuses a dest that is not a *\git-hooks path' {
        Deploy-GitHooks -Source $script:src -Destination (Join-Path $TestDrive 'deploy\hooks') | Should -BeFalse
    }

    It 'refuses a drive-root git-hooks dest' {
        Deploy-GitHooks -Source $script:src -Destination 'C:\git-hooks' | Should -BeFalse
    }

    It 'refuses a source without a pre-commit dispatcher' {
        $empty = Join-Path $TestDrive 'empty\git-hooks'
        New-Item -ItemType Directory -Path $empty -Force | Out-Null
        Deploy-GitHooks -Source $empty -Destination $script:dest | Should -BeFalse
    }

    # BUG-068: Copy-Item is byte-verbatim, so a CRLF-tainted working tree would
    # propagate a CRLF shebang into the mirror and every hook would die "No such
    # file or directory". Deploy must normalize the deployed dispatchers to LF.
    It 'normalizes CRLF hook shebangs to LF on deploy (BUG-068)' {
        [System.IO.File]::WriteAllText(
            (Join-Path $script:src 'pre-commit'), "#!/usr/bin/env bash`r`nexit 0`r`n")
        Deploy-GitHooks -Source $script:src -Destination $script:dest | Should -BeTrue

        $bytes = [System.IO.File]::ReadAllBytes((Join-Path $script:dest 'pre-commit'))
        ($bytes -contains 13) | Should -BeFalse
        [System.Text.Encoding]::ASCII.GetString($bytes[0..18]) | Should -BeExactly '#!/usr/bin/env bash'
        $bytes[19] | Should -Be 10
    }
}

Describe 'Set-GlobalHooksPath (wire when unset, preserve otherwise)' {
    BeforeEach {
        $script:cfg = Join-Path $TestDrive 'throwaway-gitconfig'
        Set-Content -Path $script:cfg -Value ''
        $script:prevGlobal = $env:GIT_CONFIG_GLOBAL
        $env:GIT_CONFIG_GLOBAL = $script:cfg
    }
    AfterEach {
        $env:GIT_CONFIG_GLOBAL = $script:prevGlobal
    }

    It 'wires an unset core.hooksPath to the target' {
        $target = 'C:\Users\me\.dotfiles\git-hooks'
        Set-GlobalHooksPath -Target $target | Should -BeTrue
        (git config --global --get core.hooksPath).Trim() | Should -BeExactly $target
    }

    It 'is a no-op when already wired to the target' {
        $target = 'C:\d\git-hooks'
        git config --global core.hooksPath $target
        Set-GlobalHooksPath -Target $target | Should -BeTrue
        (git config --global --get core.hooksPath).Trim() | Should -BeExactly $target
    }

    It 'preserves an unrelated pre-existing core.hooksPath' {
        git config --global core.hooksPath 'C:\other\hooks'
        Set-GlobalHooksPath -Target 'C:\d\git-hooks' | Should -BeTrue
        (git config --global --get core.hooksPath).Trim() | Should -BeExactly 'C:\other\hooks'
    }
}
