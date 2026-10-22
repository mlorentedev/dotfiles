[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$copilot = Get-Command copilot -ErrorAction Stop
$scratch = Join-Path ([IO.Path]::GetTempPath()) ("copilot-native-skills-{0}" -f [guid]::NewGuid().ToString('N'))
$copilotHome = Join-Path $scratch '.copilot'
$previousCopilotHome = $env:COPILOT_HOME

function Ensure-Directory {
    param([string]$Path)
    New-Item -ItemType Directory -Path $Path -Force | Out-Null
}

function Write-Warn { param([string]$Message) Write-Warning $Message }

# The skills surface of `dotf harness deploy`, which setup-windows.ps1 runs.
# The dotf on PATH is the one setup installed (or CI built from this tree).
function Invoke-SkillDeploy {
    param([string]$RepoRoot, [string]$TargetHome)
    $out = & dotf harness skills --repo-root $RepoRoot --home $TargetHome 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "dotf harness skills failed:`n$($out -join "`n")"
    }
}

try {
    $env:COPILOT_HOME = $copilotHome
    # The catalog is injected into an EXISTING deployed instructions file (the
    # base file is copied verbatim earlier in setup). Seed it from the repo
    # source so the injector runs, then assert the LF/no-BOM contract the
    # doctor's drift check depends on (WIN-008/#1289).
    Ensure-Directory $copilotHome
    Copy-Item -LiteralPath (Join-Path $repoRoot 'ai\copilot\copilot-instructions.md') -Destination (Join-Path $copilotHome 'copilot-instructions.md') -Force
    Invoke-SkillDeploy -RepoRoot $repoRoot -TargetHome $scratch

    $instructions = Join-Path $copilotHome 'copilot-instructions.md'
    $bytes = [System.IO.File]::ReadAllBytes($instructions)
    if (-not ([System.Text.Encoding]::UTF8.GetString($bytes) -match 'skill catalog')) {
        throw "catalog was not injected into $instructions"
    }
    if ($bytes -contains 13) {
        throw "$instructions carries CR bytes; the deployed copy must stay LF like its repo source"
    }
    if ($bytes.Length -ge 3 -and $bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF) {
        throw "$instructions starts with a UTF-8 BOM"
    }
    $skillBytes = [System.IO.File]::ReadAllBytes((Join-Path $copilotHome 'skills\handoff\SKILL.md'))
    if ($skillBytes -contains 13) {
        throw "rendered SKILL.md carries CR bytes"
    }

    $handoff = Join-Path $copilotHome 'skills\handoff\SKILL.md'
    $auxiliary = Join-Path $copilotHome 'skills\systematic-debugging\root-cause-tracing.md'
    if (-not (Test-Path -LiteralPath $handoff)) {
        throw "Windows deploy did not create $handoff"
    }
    if (-not (Test-Path -LiteralPath $auxiliary)) {
        throw "Windows deploy did not preserve $auxiliary"
    }

    # Target filtering is proved on a skill this test owns. It used to name a real
    # skill (crystallize), which stopped being Claude-only when the vault dropped
    # its targets, and the assertion then failed for a reason that is not a bug.
    # The fixture is the handoff record with targets: [claude] added, deployed from
    # a scratch dotfiles dir: the claude target must receive it (the control that
    # the fixture deployed at all) and the copilot target must not.
    $fixtureDir = Join-Path $scratch 'fixture'
    $fixtureRecord = Join-Path $fixtureDir 'harness\skills\claude-only'
    Ensure-Directory $fixtureRecord
    Copy-Item -LiteralPath (Join-Path $repoRoot 'harness\manifest.json') -Destination (Join-Path $fixtureDir 'harness\manifest.json') -Force
    $fixtureMd = (Get-Content -LiteralPath (Join-Path $repoRoot 'harness\skills\handoff\SKILL.md') -Raw) -replace '(?m)^name: handoff\r?$', "name: claude-only`ntargets: [claude]"
    [System.IO.File]::WriteAllText((Join-Path $fixtureRecord 'SKILL.md'), $fixtureMd, (New-Object System.Text.UTF8Encoding($false)))
    # Its own home: a deploy prunes generated outputs whose record is missing, and
    # this fixture holds only one record, so deploying into the home the Copilot
    # discovery check reads below would delete handoff from it.
    $fixtureHome = Join-Path $scratch 'fixture-home'
    Invoke-SkillDeploy -RepoRoot $fixtureDir -TargetHome $fixtureHome
    $claudeOnlyForClaude = Join-Path $fixtureHome '.claude\skills\claude-only\SKILL.md'
    $claudeOnlyForCopilot = Join-Path $fixtureHome '.copilot\skills\claude-only'
    if (-not (Test-Path -LiteralPath $claudeOnlyForClaude)) {
        throw "Windows deploy did not render the Claude-only fixture for the claude target: $claudeOnlyForClaude"
    }
    if (Test-Path -LiteralPath $claudeOnlyForCopilot) {
        throw "Windows deploy ignored Copilot target filtering: $claudeOnlyForCopilot"
    }

    $output = & $copilot.Source skill list 2>&1
    if ($LASTEXITCODE -ne 0) {
        # The deploy half above is the deterministic guard. Discovery needs the
        # CLI itself to run, which on a runner without a Copilot login may not;
        # say so loudly rather than fail the deploy assertions with it.
        Write-Warn "copilot skill list failed (discovery not verified):`n$($output -join "`n")"
        Write-Output 'Windows setup deployed handoff (LF, no BOM); Copilot discovery not verified on this host.'
        return
    }
    if (($output -join "`n") -notmatch '(?m)^\s*handoff\s+-') {
        throw "Copilot did not discover handoff in $copilotHome`n$($output -join "`n")"
    }

    Write-Output 'Windows setup deployed handoff (LF, no BOM) and Copilot discovered it.'
}
finally {
    $env:COPILOT_HOME = $previousCopilotHome
    if (Test-Path -LiteralPath $scratch) {
        Remove-Item -LiteralPath $scratch -Recurse -Force
    }
}
