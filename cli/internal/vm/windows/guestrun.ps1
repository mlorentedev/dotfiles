# One step of a Windows guest run, started by a scheduled task as the test
# user, so dotf converges that user's profile and not LocalSystem's, which is
# who the guest agent runs commands as. The host reads the outcome from files
# under $work: <step>.log, the output, and <step>.exit, the exit code, which
# appears last and whole.
#   first:  unpack the ref's archive and run install.ps1 from it (installs
#           the dotf release the ref pins, then dotf converge)
#   second: dotf converge again, which must change nothing
param(
    [Parameter(Mandatory)]
    [ValidateSet('first', 'second')]
    [string]$Step
)
$work = 'C:\Users\Public\dotf'
$repo = Join-Path $env:USERPROFILE 'Projects\dotfiles'
$log = Join-Path $work "$Step.log"
$code = 1
try {
    if ($Step -eq 'first') {
        Expand-Archive -Path (Join-Path $work 'src.zip') -DestinationPath (Split-Path $repo) -Force -ErrorAction Stop
    }
    Set-Location $repo -ErrorAction Stop
    $env:DOTFILES_REPO_DIR = $repo
    # Windows PowerShell turns a native command's stderr into error records;
    # "$_" keeps their text and drops the record decoration. The default
    # ErrorActionPreference (Continue) lets a warning on stderr pass, where
    # Stop would end the step on it.
    if ($Step -eq 'first') {
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repo 'install.ps1') 2>&1 |
            ForEach-Object { "$_" } | Out-File -FilePath $log -Encoding utf8
    } else {
        & (Join-Path $env:USERPROFILE '.local\bin\dotf.exe') converge 2>&1 |
            ForEach-Object { "$_" } | Out-File -FilePath $log -Encoding utf8
    }
    $code = $LASTEXITCODE
} catch {
    "$_" | Out-File -FilePath $log -Encoding utf8 -Append
} finally {
    $tmp = Join-Path $work "$Step.exit.tmp"
    Set-Content -Path $tmp -Value $code
    Move-Item -Path $tmp -Destination (Join-Path $work "$Step.exit") -Force
}
