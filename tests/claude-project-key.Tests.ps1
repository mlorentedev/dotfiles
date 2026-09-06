# Pester 5 anti-drift guard for the Claude auto-memory project-key encoding
# (BUG-031 / #689). The PowerShell encoder MUST stay byte-for-byte equal to the
# Go layer (memlink.ClaudeProjectKey). This locks the pure PS encoder to the same
# known-good keys the Go TestClaudeProjectKey asserts, so the #689 drift (deleting
# ':' -> the wrong single-dash key Claude never reads) can never silently return.
#
# WIN-004 lesson: -Skip conditions are computed at DISCOVERY time (top-level),
# NOT inside BeforeAll.

# Known-good keys -- identical to cli/internal/memlink/memlink_test.go. Any change
# that makes the PS encoder diverge from these (== Go) fails the build.
$script:Cases = @(
    @{ Path = 'C:\Users\me\p';                                  Key = 'C--Users-me-p' }
    @{ Path = 'C:\Users\mlorente\Projects\Workspace\dotfiles';  Key = 'C--Users-mlorente-Projects-Workspace-dotfiles' }
    @{ Path = '/home/me/Projects/dotfiles';                     Key = '-home-me-Projects-dotfiles' }
    @{ Path = '/home/manu/Projects/svqtriana.github.io';        Key = '-home-manu-Projects-svqtriana-github-io' }
)

# Probe (discovery time): does the installed dotf support `mem project-key`? A
# released dotf predating this PR does not (#734), so the runtime cross-check is
# SKIPPED, not failed, in that environment.
$script:DotfKeySupported = $false
if (Get-Command dotf -ErrorAction SilentlyContinue) {
    $probe = (& dotf mem project-key 'C:\x' 2>$null)
    if ($LASTEXITCODE -eq 0 -and $probe) { $script:DotfKeySupported = $true }
}

BeforeAll {
    . (Join-Path $PSScriptRoot '..\scripts\utils.ps1')
}

Describe 'Get-ClaudeProjectKeyEncoded (pure encoder locked to Go)' {
    It "encodes '<Path>' as '<Key>' (== memlink.ClaudeProjectKey)" -ForEach $script:Cases {
        Get-ClaudeProjectKeyEncoded $Path | Should -BeExactly $Key
    }

    It "never deletes the drive ':' (the #689 regression)" {
        # The old bug produced 'C-Users-...' (single dash). Guard the exact class.
        Get-ClaudeProjectKeyEncoded 'C:\Users\me\p' | Should -Not -BeExactly 'C-Users-me-p'
    }
}

Describe 'Get-ClaudeProjectKey (dotf-first wrapper)' {
    It "resolves '<Path>' to '<Key>' via dotf or the fallback" -ForEach $script:Cases {
        Get-ClaudeProjectKey $Path | Should -BeExactly $Key
    }

    It "agrees with 'dotf mem project-key' when the subcommand exists" -Skip:(-not $script:DotfKeySupported) -ForEach $script:Cases {
        $fromCli = (& dotf mem project-key $Path | Select-Object -First 1).Trim()
        $fromCli | Should -BeExactly $Key
        $fromCli | Should -BeExactly (Get-ClaudeProjectKeyEncoded $Path)
    }
}
