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
# Windows PowerShell decodes a native command's output with the console's
# code page (OEM 437 here), which turns dotf's UTF-8 into mojibake. Setting it
# on the console reaches the child powershell.exe that runs install.ps1 too.
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
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
    # A dotf source build the host copied in (--dotf) goes where step two runs
    # dotf from, and on PATH: install.ps1 keeps a dotf on PATH that reports
    # "dev", so the run converges with the build, not the pinned release.
    $dev = Join-Path $work 'dotf.exe'
    $bin = Join-Path $env:USERPROFILE '.local\bin'
    $installed = Join-Path $bin 'dotf.exe'
    if ($Step -eq 'first' -and (Test-Path $dev)) {
        New-Item -ItemType Directory -Force -Path $bin -ErrorAction Stop | Out-Null
        Copy-Item -Path $dev -Destination $installed -Force -ErrorAction Stop
        $env:PATH = "$bin;$env:PATH"
    }
    if ($Step -eq 'first') {
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repo 'install.ps1') 2>&1 |
            ForEach-Object { "$_" } | Out-File -FilePath $log -Encoding utf8
    } else {
        & $installed converge 2>&1 |
            ForEach-Object { "$_" } | Out-File -FilePath $log -Encoding utf8
    }
    $code = $LASTEXITCODE
    # Nothing else asserts that the build is what converged: had install.ps1
    # replaced it with a release, the run would judge the release.
    if ((Test-Path $dev) -and (Get-FileHash -Path $dev).Hash -ne (Get-FileHash -Path $installed).Hash) {
        "the dotf in $bin is not the --dotf build: the step did not run the build" |
            Out-File -FilePath $log -Encoding utf8 -Append
        $code = 1
    }
} catch {
    "$_" | Out-File -FilePath $log -Encoding utf8 -Append
} finally {
    $tmp = Join-Path $work "$Step.exit.tmp"
    Set-Content -Path $tmp -Value $code
    Move-Item -Path $tmp -Destination (Join-Path $work "$Step.exit") -Force
}
