# Pester 5 tests for scripts/dotfiles-sync.ps1, the twin of the bats suite in
# tests/dotfiles-sync.bats (#1795). Every run points DOTFILES_DIR and
# DOTFILES_REPO_DIR at TestDrive: the script's defaults are the real
# installation and checkout, and a test that reached them synced real files.
# Each case runs in a CHILD pwsh, so the script's `exit` ends the child only.

Describe 'dotfiles-sync.ps1' {

    BeforeAll {
        $script:Sync = (Resolve-Path (Join-Path $PSScriptRoot '..' 'scripts' 'dotfiles-sync.ps1')).Path

        function Invoke-Sync {
            param([string]$Local, [string]$Repo, [string[]]$Arguments = @())
            $env:DOTFILES_DIR = $Local
            $env:DOTFILES_REPO_DIR = $Repo
            try {
                $out = & pwsh -NoProfile -NonInteractive -File $script:Sync @Arguments 2>&1 | Out-String
                [pscustomobject]@{ Code = $LASTEXITCODE; Output = $out }
            } finally {
                Remove-Item Env:DOTFILES_DIR, Env:DOTFILES_REPO_DIR -ErrorAction SilentlyContinue
            }
        }
    }

    It 'rejects the retired -SecretsOnly switch instead of running a sync' {
        $local = New-Item -ItemType Directory -Path (Join-Path $TestDrive 'reject-local')
        $repo = New-Item -ItemType Directory -Path (Join-Path $TestDrive 'reject-repo')
        $r = Invoke-Sync -Local $local -Repo $repo -Arguments @('-SecretsOnly')
        $r.Code | Should -Not -Be 0
        $r.Output | Should -Match 'SecretsOnly'
        $r.Output | Should -Not -Match 'Sync complete'
    }

    It 'fails on a missing installation directory' {
        $repo = New-Item -ItemType Directory -Path (Join-Path $TestDrive 'missing-repo')
        $r = Invoke-Sync -Local (Join-Path $TestDrive 'absent') -Repo $repo
        $r.Code | Should -Be 1
        $r.Output | Should -Match 'Local dotfiles not found'
    }

    It 'refuses to push while the repo has an unstaged change, as the bash twin does' {
        $repo = Join-Path $TestDrive 'dirty-repo'
        $local = Join-Path $TestDrive 'dirty-local'
        & git init -q $repo
        Set-Content -Path (Join-Path $repo 'tracked.txt') -Value 'v1'
        & git -C $repo add tracked.txt
        & git -C $repo -c user.name=t -c user.email=t@example.invalid commit -q -m init
        & git clone -q $repo $local
        Set-Content -Path (Join-Path $repo 'tracked.txt') -Value 'v2, not staged'

        $r = Invoke-Sync -Local $local -Repo $repo
        $r.Output | Should -Match 'Uncommitted changes in repo'
        $r.Output | Should -Not -Match 'Push complete|Pull complete'
    }

    It 'never copies sensitive/ in either direction' {
        $repo = Join-Path $TestDrive 'copy-repo'
        $local = Join-Path $TestDrive 'copy-local'
        & git init -q $repo
        & git -C $repo -c user.name=t -c user.email=t@example.invalid commit -q --allow-empty -m init
        & git clone -q $repo $local
        New-Item -ItemType Directory -Path (Join-Path $local 'sensitive'), (Join-Path $repo 'sensitive') | Out-Null
        Set-Content -Path (Join-Path $local 'sensitive' 'retired.secret.age') -Value 'mirror-only'
        Set-Content -Path (Join-Path $repo 'sensitive' 'repo-only.secret.age') -Value 'repo-only'

        $r = Invoke-Sync -Local $local -Repo $repo
        $r.Code | Should -Be 0

        Join-Path $repo 'sensitive' 'retired.secret.age' | Should -Not -Exist
        Join-Path $local 'sensitive' 'repo-only.secret.age' | Should -Not -Exist
    }
}
