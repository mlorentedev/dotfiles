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

function Write-Info { param([string]$Message) }
function Write-Success { param([string]$Message) }
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
    Deploy-SkillRecord -DotfilesDir $repoRoot

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
        throw "copilot skill list failed:`n$($output -join "`n")"
    }
    if (($output -join "`n") -notmatch '(?m)^\s*handoff\s+-') {
        throw "Copilot did not discover handoff in $skillTarget`n$($output -join "`n")"
    }

    Write-Output 'Windows setup deployed handoff and Copilot discovered it.'
}
finally {
    $env:COPILOT_HOME = $previousCopilotHome
    $env:USERPROFILE = $previousUserProfile
    if (Test-Path -LiteralPath $scratch) {
        Remove-Item -LiteralPath $scratch -Recurse -Force
    }
}
