# One `dotf vm windows exec`, started by a scheduled task as the test user. The
# host reads request.json's program from $Dir, never from this command line
# (ADR-052), and reads the outcome back from files in it: stdout, stderr and
# exit, which appears last and whole. exit is the program's code, or `timeout`.
# The limit is enforced here and not only on the host: a host that stops
# polling leaves the task running, so this kills the program and its children.
param(
    [Parameter(Mandatory)]
    [string]$Dir
)
$req = Get-Content -Raw -Encoding UTF8 -Path (Join-Path $Dir 'request.json') | ConvertFrom-Json
$stdin = Join-Path $Dir 'stdin'
$stdout = Join-Path $Dir 'stdout'
$stderr = Join-Path $Dir 'stderr'
$exit = Join-Path $Dir 'exit'
$code = '127'
# Windows PowerShell's children inherit the console's code page, OEM 437 here.
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
try {
    # An empty stdin: the agent has none to give, and EOF beats a hang.
    [IO.File]::WriteAllText($stdin, '')
    $start = @{
        FilePath               = $req.file
        WorkingDirectory       = $env:USERPROFILE
        RedirectStandardInput  = $stdin
        RedirectStandardOutput = $stdout
        RedirectStandardError  = $stderr
        NoNewWindow            = $true
        PassThru               = $true
        ErrorAction            = 'Stop'
    }
    if ($req.args) { $start.ArgumentList = $req.args }
    $p = Start-Process @start
    # Windows PowerShell loses ExitCode unless the handle was read while running.
    $null = $p.Handle
    $done = $true
    if ($req.timeout_ms -gt 0) { $done = $p.WaitForExit([int]$req.timeout_ms) } else { $p.WaitForExit() }
    if ($done) {
        $code = "$($p.ExitCode)"
    } else {
        & taskkill.exe /T /F /PID $p.Id | Out-Null
        $code = 'timeout'
    }
} catch {
    [IO.File]::WriteAllText($stderr, "$_`r`n")
}
[IO.File]::WriteAllText("$exit.tmp", $code)
Move-Item -Path "$exit.tmp" -Destination $exit -Force
