# Pester 5 guard for command-name collisions in powershell/profile.ps1 (BUG-034).
#
# PowerShell resolves commands Alias -> Function -> Cmdlet -> Application, so a
# profile function whose name matches a built-in alias is simply unreachable.
# Unlike the zsh sibling (#744) there is no parse error and no warning: the
# function is silently dead, which is why gp/gl/gcs/gbp went unnoticed.
#
# The zsh half of this guard is tests/shell-alias-collision.bats. This is the
# PowerShell half, and it is behavioural on purpose: it loads the real profile in
# a clean session and asks what each name actually resolves to.

BeforeAll {
    $script:ProfilePath = (Resolve-Path (Join-Path $PSScriptRoot '..\powershell\profile.ps1')).Path

    # Probe run in a child `pwsh -NoProfile`: snapshot the built-in alias table
    # first, then load the profile and report what every function it defines
    # resolves to. Function names come from the parser rather than a regex, so
    # definitions nested inside conditional blocks are found too.
    $probe = @'
param([Parameter(Mandatory)][string]$ProfilePath)

$ast = [System.Management.Automation.Language.Parser]::ParseFile($ProfilePath, [ref]$null, [ref]$null)
$names = $ast.FindAll(
    { $args[0] -is [System.Management.Automation.Language.FunctionDefinitionAst] }, $true) |
    ForEach-Object { $_.Name } | Sort-Object -Unique

$builtin = @{}
Get-Alias | ForEach-Object { $builtin[$_.Name] = $_.Definition }

. $ProfilePath *> $null

$names | ForEach-Object {
    $cmd = Get-Command -Name $_ -ErrorAction SilentlyContinue | Select-Object -First 1
    [pscustomobject]@{
        Name            = $_
        WasBuiltinAlias = $builtin.ContainsKey($_)
        ResolvesAs      = if ($cmd) { [string]$cmd.CommandType } else { 'None' }
        Definition      = if ($cmd -and $cmd.CommandType -eq 'Function') { [string]$cmd.Definition } else { '' }
    }
} | ConvertTo-Json -Depth 3
'@
    $probeFile = Join-Path ([System.IO.Path]::GetTempPath()) ('profile-probe-' + [System.IO.Path]::GetRandomFileName() + '.ps1')
    Set-Content -LiteralPath $probeFile -Value $probe -Encoding UTF8
    try {
        $json = & pwsh -NoProfile -File $probeFile -ProfilePath $script:ProfilePath
        $script:Resolution = @($json | ConvertFrom-Json)
    } finally {
        Remove-Item -LiteralPath $probeFile -Force -ErrorAction SilentlyContinue
    }

    function Get-Resolution {
        param([string]$Name)
        $script:Resolution | Where-Object { $_.Name -eq $Name } | Select-Object -First 1
    }
}

Describe 'powershell/profile.ps1 command-name collisions' {
    It 'the probe actually loaded the profile' {
        # Guards against a vacuously green suite: an empty resolution table would
        # make every assertion below pass without testing anything.
        $script:Resolution.Count | Should -BeGreaterThan 5
    }

    It 'no function defined in the profile is left shadowed by a built-in alias' {
        # The class-level rule. A name that ships as a built-in alias must be
        # cleared by the profile, or the function it names can never run.
        $shadowed = $script:Resolution |
            Where-Object { $_.WasBuiltinAlias -and $_.ResolvesAs -ne 'Function' }
        $shadowed.Name -join ', ' | Should -BeNullOrEmpty
    }

    It 'the profile clears exactly the built-in aliases it needs to' {
        # Sanity check in the other direction: the profile should only be
        # removing built-ins it actually redefines, not shadowing unrelated ones.
        $cleared = $script:Resolution | Where-Object { $_.WasBuiltinAlias }
        $cleared.Count | Should -BeGreaterThan 0
    }

    Context 'the four collisions reported in BUG-034' {
        It "'<Name>' resolves to the profile function, not the built-in alias" -ForEach @(
            @{ Name = 'gp';  Expect = 'git pull' }
            @{ Name = 'gl';  Expect = 'git log' }
            @{ Name = 'gcs'; Expect = 'cheat-sheets' }
            @{ Name = 'gbp'; Expect = 'boilerplates' }
        ) {
            $r = Get-Resolution -Name $Name
            $r | Should -Not -BeNullOrEmpty -Because "profile.ps1 must define $Name"
            $r.ResolvesAs | Should -BeExactly 'Function'
            $r.Definition | Should -BeLike "*$Expect*"
        }
    }
}
