[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$copilot = Get-Command copilot -ErrorAction Stop
$scratch = Join-Path ([IO.Path]::GetTempPath()) ("copilot-native-skills-{0}" -f [guid]::NewGuid().ToString('N'))
$copilotHome = Join-Path $scratch '.copilot'
$previousCopilotHome = $env:COPILOT_HOME
$previousUserProfile = $env:USERPROFILE

function Ensure-Directory {
    param([string]$Path)
    New-Item -ItemType Directory -Path $Path -Force | Out-Null
}

function Write-Info { param([string]$Message) $null = $Message }
function Write-Success { param([string]$Message) $null = $Message }
function Write-Warn { param([string]$Message) Write-Warning $Message }

function Import-SetupFunction {
    param(
        [System.Management.Automation.Language.Ast]$Ast,
        [string]$Name
    )
    $definition = $Ast.FindAll({
        param($node)
        $node -is [System.Management.Automation.Language.FunctionDefinitionAst]
    }, $true) | Where-Object Name -EQ $Name | Select-Object -First 1
    if (-not $definition) {
        throw "Function $Name not found in setup-windows.ps1"
    }
    $scriptDefinition = $definition.Extent.Text -replace (
        '^function\s+' + [regex]::Escape($Name)
    ), "function script:$Name"
    . ([scriptblock]::Create($scriptDefinition))
}

try {
    $tokens = $null
    $parseErrors = $null
    $setupAst = [System.Management.Automation.Language.Parser]::ParseFile(
        (Join-Path $repoRoot 'setup-windows.ps1'),
        [ref]$tokens,
        [ref]$parseErrors
    )
    if ($parseErrors.Count -gt 0) {
        throw "setup-windows.ps1 failed to parse: $($parseErrors[0].Message)"
    }
    # Write-Utf8LfFile (and the console encoding block) live in utils.ps1, which
    # setup-windows.ps1 dot-sources before Deploy-SkillRecord runs.
    . (Join-Path $repoRoot 'scripts\utils.ps1')
    foreach ($functionName in @(
        'Test-SkillTargetsAgent',
        'Get-SkillField',
        'Convert-SkillRecord',
        'Deploy-SkillRecord'
    )) {
        Import-SetupFunction -Ast $setupAst -Name $functionName
    }

    $env:USERPROFILE = $scratch
    $env:COPILOT_HOME = $copilotHome
    # The catalog is injected into an EXISTING deployed instructions file (the
    # base file is copied verbatim earlier in setup). Seed it from the repo
    # source so the injector runs, then assert the LF/no-BOM contract the
    # doctor's drift check depends on (WIN-008/#1289).
    Ensure-Directory $copilotHome
    Copy-Item -LiteralPath (Join-Path $repoRoot 'ai\copilot\copilot-instructions.md') -Destination (Join-Path $copilotHome 'copilot-instructions.md') -Force
    Deploy-SkillRecord -DotfilesDir $repoRoot

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
    $claudeOnly = Join-Path $copilotHome 'skills\crystallize'
    if (-not (Test-Path -LiteralPath $handoff)) {
        throw "Windows deploy did not create $handoff"
    }
    if (-not (Test-Path -LiteralPath $auxiliary)) {
        throw "Windows deploy did not preserve $auxiliary"
    }
    if (Test-Path -LiteralPath $claudeOnly) {
        throw "Windows deploy ignored Copilot target filtering: $claudeOnly"
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
    $env:USERPROFILE = $previousUserProfile
    if (Test-Path -LiteralPath $scratch) {
        Remove-Item -LiteralPath $scratch -Recurse -Force
    }
}
