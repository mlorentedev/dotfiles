# dotfiles-selfupdate.ps1 -- OPS-001 (Windows parity of scripts/dotfiles-selfupdate.sh)
#
# Opt-in self-deploy, run by the DotfilesSelfUpdate Scheduled Task. Pulls the
# dotfiles repo from its git remote and re-runs the idempotent setup -- but ONLY
# via a clean fast-forward, and ONLY when HEAD actually moved. Anything else
# (dirty worktree, diverged history, unreachable remote, no upstream) is logged
# and skipped with exit 0. A real setup failure is the only non-zero exit.
#
# ASCII-only (PSScriptAnalyzer CI fails on non-ASCII without a BOM).
#
# Env:
#   DOTFILES_REPO_DIR              repo to update  (default: $HOME\Projects\dotfiles)
#   DOTFILES_SELFUPDATE_SETUP_CMD  setup .ps1 path (default: <repo>\setup-windows.ps1)

$ErrorActionPreference = 'Stop'

$repo = if ($env:DOTFILES_REPO_DIR) { $env:DOTFILES_REPO_DIR } else { Join-Path $env:USERPROFILE 'Projects\dotfiles' }
$setupCmd = if ($env:DOTFILES_SELFUPDATE_SETUP_CMD) { $env:DOTFILES_SELFUPDATE_SETUP_CMD } else { Join-Path $repo 'setup-windows.ps1' }

function Write-Skip([string]$m) { Write-Host "[skip] $m" }
function Write-Step([string]$m) { Write-Host "-> $m" }

# guard: must be a git repo
& git -C $repo rev-parse --git-dir *> $null
if ($LASTEXITCODE -ne 0) { Write-Skip "Not a git repo: $repo -- nothing to self-update"; exit 0 }

# guard: never touch a dirty worktree
$dirty = & git -C $repo status --porcelain
if ($dirty) { Write-Skip "Dirty worktree in $repo -- skipping (commit or stash first)"; exit 0 }

# fetch; a network failure is transient -> skip and retry next slot
& git -C $repo fetch --quiet *> $null
if ($LASTEXITCODE -ne 0) { Write-Skip "git fetch failed (network?) -- skipping this run"; exit 0 }

# guard: an upstream must be configured for the current branch
$upstream = & git -C $repo rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>$null
if ($LASTEXITCODE -ne 0 -or -not $upstream) { Write-Skip "No upstream configured -- skipping"; exit 0 }

$localRev  = & git -C $repo rev-parse HEAD
$remoteRev = & git -C $repo rev-parse '@{u}'
$baseRev   = & git -C $repo merge-base HEAD '@{u}'

# already current: no deploy needed
if ($localRev -eq $remoteRev) { Write-Step "Already current ($upstream) -- no deploy needed"; exit 0 }

# diverged / non-fast-forward: never merge, rebase, or reset unattended
if ($baseRev -ne $localRev) { Write-Skip "Local branch has diverged from $upstream (non fast-forward) -- skipping"; exit 0 }

# clean fast-forward, then re-run the idempotent setup
Write-Step "Fast-forwarding to $upstream ..."
& git -C $repo merge --ff-only '@{u}' *> $null
if ($LASTEXITCODE -ne 0) { Write-Skip "Fast-forward failed unexpectedly -- skipping (worktree untouched)"; exit 0 }

Write-Step "Re-running setup: $setupCmd"
& powershell.exe -NoProfile -ExecutionPolicy Bypass -File $setupCmd
if ($LASTEXITCODE -ne 0) { Write-Host "[err] Setup failed (exit $LASTEXITCODE) -- see output above"; exit $LASTEXITCODE }
Write-Host "[ok] Self-update complete"
