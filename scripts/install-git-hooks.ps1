# install-git-hooks.ps1 - deploy the GUARD-001 global memory-sink dispatcher and
# wire it machine-wide via core.hooksPath. The Windows twin of
# scripts/install-git-hooks.sh (which always noted "a Windows setup twin is a
# tracked follow-up" -- this is it, #691).
#
# Bootstrap stays shell (ADR-020 C7): the deployed `dotf` release carries no
# source tree, so setup-windows.ps1 places the dispatcher and `dotf doctor`
# verifies the wiring thereafter. The dispatcher hooks are POSIX and run under
# Git-for-Windows `sh`, so no per-OS hook rewrite is needed -- only the
# deploy + wire.
#
# Sourced by setup-windows.ps1; also runnable standalone to (re)install:
#     .\scripts\install-git-hooks.ps1 [SourceDir] [DotfilesDir]
#
# Idempotent (clean mirror + converge) and safety-first: an unrelated pre-existing
# core.hooksPath is preserved, never clobbered (a global hooksPath has machine-wide
# blast radius). Functions take src/dest as args so Pester can drive them against
# fixtures under a throwaway GIT_CONFIG_GLOBAL with no real ~/.gitconfig mutation.
#
# ASCII-only by repo convention (pattern-powershell-ascii-only).

Set-StrictMode -Version Latest

# Repair-HookLineEndings <Path>: strip CR from every text file under <Path> so the
# deployed POSIX dispatchers keep LF shebangs. Copy-Item is byte-verbatim, so a
# CRLF-tainted working tree (a checkout predating the .gitattributes 'eol=lf' rule,
# under core.autocrlf=true) would otherwise propagate "#!/usr/bin/env bash\r" into
# the mirror -- bash then resolves interpreter "bash\r" and every hook dies with
# "No such file or directory" (BUG-068). Binary files are left untouched.
function Repair-HookLineEndings {
    [Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSUseSingularNouns', '',
        Justification = 'LineEndings names a byte-sequence concept; a singular form would misname it.')]
    param([Parameter(Mandatory)][string]$Path)

    foreach ($file in (Get-ChildItem -LiteralPath $Path -Recurse -File)) {
        $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
        if ($bytes -contains 0) { continue }      # binary: never normalize
        if ($bytes -notcontains 13) { continue }  # no CR: already LF
        $out = [System.Collections.Generic.List[byte]]::new($bytes.Length)
        for ($i = 0; $i -lt $bytes.Length; $i++) {
            if ($bytes[$i] -eq 13 -and ($i + 1) -lt $bytes.Length -and $bytes[$i + 1] -eq 10) { continue }
            $out.Add($bytes[$i])
        }
        [System.IO.File]::WriteAllBytes($file.FullName, $out.ToArray())
    }
}

# Deploy-GitHooks: clean-mirror the dispatcher tree into the deploy mirror. A
# clean mirror (remove-then-copy, not a bare copy) so a hook removed upstream
# never lingers and keeps firing -- a stale security hook is worse than none.
# Returns $true on success, $false on a guard trip or error.
function Deploy-GitHooks {
    [Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSUseSingularNouns', '',
        Justification = 'GitHooks is inherently plural; mirrors install-git-hooks.sh and the git-hooks/ dispatcher dir')]
    param(
        [Parameter(Mandatory)][string]$Source,
        [Parameter(Mandatory)][string]$Destination
    )

    if (-not (Test-Path -LiteralPath $Source -PathType Container)) {
        Write-Host "[FAIL] install-git-hooks: source dispatcher dir not found: $Source" -ForegroundColor Red
        return $false
    }
    if (-not (Test-Path -LiteralPath (Join-Path $Source 'pre-commit') -PathType Leaf)) {
        Write-Host "[FAIL] install-git-hooks: $Source has no pre-commit dispatcher -- refusing to deploy" -ForegroundColor Red
        return $false
    }

    # Guard the destructive mirror: the dest MUST be a non-root *\git-hooks path.
    # Defends against an empty/misconfigured DOTFILES_DIR turning Remove-Item loose.
    $normDest = $Destination.TrimEnd('\', '/')
    if ($normDest -notmatch '[\\/]git-hooks$') {
        Write-Host "[FAIL] install-git-hooks: refusing to mirror to unexpected path: '$Destination'" -ForegroundColor Red
        return $false
    }
    if ($normDest -eq (Join-Path $HOME 'git-hooks').TrimEnd('\', '/') -or
        $normDest -match '^[A-Za-z]:[\\/]git-hooks$' -or $normDest -match '^[\\/]git-hooks$') {
        Write-Host "[FAIL] install-git-hooks: refusing unsafe dest: '$Destination'" -ForegroundColor Red
        return $false
    }

    # Self-mirror guard (#695): if src and dest resolve to the same directory, the
    # clean-mirror below (remove dest THEN copy) would delete the dispatcher and
    # copy nothing back. Nothing to mirror onto itself; report already-in-place.
    $srcFull = (Resolve-Path -LiteralPath $Source).Path.TrimEnd('\', '/')
    $destFull = if (Test-Path -LiteralPath $Destination) {
        (Resolve-Path -LiteralPath $Destination).Path.TrimEnd('\', '/')
    } else { $normDest }
    if ($srcFull -eq $destFull) {
        Write-Host "[INFO] install-git-hooks: src and dest are the same directory ($Destination) -- skipping mirror (already in place)" -ForegroundColor Blue
        Write-Host "[SUCCESS] GUARD dispatcher already in place at $Destination" -ForegroundColor Green
        return $true
    }

    if (Test-Path -LiteralPath $Destination) { Remove-Item -LiteralPath $Destination -Recurse -Force }
    New-Item -ItemType Directory -Path $Destination -Force | Out-Null
    Copy-Item -Path (Join-Path $Source '*') -Destination $Destination -Recurse -Force
    Repair-HookLineEndings -Path $Destination   # BUG-068: LF shebangs regardless of source EOL
    Write-Host "[SUCCESS] GUARD dispatcher deployed to $Destination" -ForegroundColor Green
    return $true
}

# Test-GuardDispatcher: does $Dir hold a GUARD-001 dispatcher? Structural test --
# the pre-commit entrypoint AND the memory-sink guard it delegates to (BUG-040).
# Not a marker grep: pre-commit execs lib/memory-sink-guard.sh, so without that
# script the guard cannot run whatever the files claim.
function Test-GuardDispatcher {
    param([string]$Dir)
    if (-not $Dir) { return $false }
    return (Test-Path -LiteralPath (Join-Path $Dir 'pre-commit')) -and
           (Test-Path -LiteralPath (Join-Path $Dir 'lib\memory-sink-guard.sh'))
}

# Test-SameHooksPath: compare two hooksPath values as paths, not as bytes. git
# reports core.hooksPath with forward slashes even on Windows, so a byte compare
# against a backslash Join-Path target is a false negative; trailing separators
# are the same class of noise. PowerShell -eq on strings is already
# case-insensitive, which matches Windows path semantics.
function Test-SameHooksPath {
    param([string]$A, [string]$B)
    $norm = { param($p) if (-not $p) { '' } else { ($p -replace '/', '\').TrimEnd('\') } }
    return ((& $norm $A) -eq (& $norm $B))
}

# Set-GlobalHooksPath: point git's global core.hooksPath at the deployed
# dispatcher -- but ONLY when unset. An unrelated value is preserved (machine-wide
# blast radius) and surfaced as a warning; an already-correct value is a no-op.
# Mirrors the `dotf doctor` checkGuardHooks contract.
#
# "Correct" means the guard RUNS, not that the path matches the mirror (BUG-040):
# developing the hooks from a repo checkout points hooksPath at an equivalent
# dispatcher, which is active and used to be reported INACTIVE on every run.
function Set-GlobalHooksPath {
    param([Parameter(Mandatory)][string]$Target)

    $current = (git config --global --get core.hooksPath 2>$null)
    if ($current) { $current = "$current".Trim() }

    if (-not $current) {
        git config --global core.hooksPath $Target
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[SUCCESS] core.hooksPath wired to the GUARD dispatcher ($Target)" -ForegroundColor Green
            return $true
        }
        Write-Host "[FAIL] install-git-hooks: failed to set core.hooksPath" -ForegroundColor Red
        return $false
    }
    if (Test-SameHooksPath -A $current -B $Target) {
        Write-Host "[INFO] core.hooksPath already wired to the GUARD dispatcher" -ForegroundColor Blue
        return $true
    }
    if (Test-GuardDispatcher -Dir $current) {
        # Active via a different tree. Name it so the divergence from the deploy
        # mirror stays visible rather than silently blessed.
        Write-Host "[SUCCESS] GUARD dispatcher active via $current (not the deploy mirror $Target)" -ForegroundColor Green
        return $true
    }
    Write-Host "[WARNING] core.hooksPath is '$current' (not the GUARD dispatcher) -- preserving it; the memory-sink guard is INACTIVE. Point it at $Target, or chain the dispatcher from your hooks, manually." -ForegroundColor Yellow
    return $true
}

# Install-GitHooks: deploy + wire. Defaults resolve the repo's git-hooks/ (this
# script's sibling parent) and the ~/.dotfiles deploy mirror.
function Install-GitHooks {
    [Diagnostics.CodeAnalysis.SuppressMessageAttribute('PSUseSingularNouns', '',
        Justification = 'GitHooks is inherently plural; mirrors install-git-hooks.sh and the git-hooks/ dispatcher dir')]
    param(
        [string]$Source = (Join-Path $PSScriptRoot '..\git-hooks'),
        [string]$DotfilesDir = $(if ($env:DOTFILES_DIR) { $env:DOTFILES_DIR } else { Join-Path $HOME '.dotfiles' })
    )

    if (-not $DotfilesDir -or $DotfilesDir.TrimEnd('\', '/') -eq $HOME.TrimEnd('\', '/') -or
        $DotfilesDir -match '^[A-Za-z]:[\\/]?$') {
        Write-Host "[FAIL] install-git-hooks: unsafe DOTFILES_DIR: '$DotfilesDir'" -ForegroundColor Red
        return $false
    }

    $dest = Join-Path $DotfilesDir 'git-hooks'
    if (-not (Deploy-GitHooks -Source $Source -Destination $dest)) { return $false }
    if (-not (Set-GlobalHooksPath -Target $dest)) { return $false }
    return $true
}

# Run standalone only when executed directly (not when dot-sourced by setup).
if ($MyInvocation.InvocationName -ne '.') {
    $null = Install-GitHooks @args
}
