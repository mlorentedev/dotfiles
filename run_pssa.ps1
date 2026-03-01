$ErrorActionPreference = 'Stop'
Install-Module PSScriptAnalyzer -Force -Scope CurrentUser -ErrorAction SilentlyContinue
$results = Invoke-ScriptAnalyzer -Path 'scripts/knowledge-crystallize.ps1' -Severity Error,Warning
if ($results) {
    $results | Format-Table -AutoSize
    exit 1
}
Write-Host 'PSScriptAnalyzer: OK'
