# First logon of the Windows test guest's build (ADR-053): install the ARM64
# virtio network and serial drivers and the QEMU guest agent from the
# virtio-win media, then shut down, which ends the build. On a failure the
# guest stays up and the build times out with a screenshot; the transcript is
# at C:\Windows\Temp\dotf-firstlogon.log.
$ErrorActionPreference = 'Stop'
Start-Transcript -Path 'C:\Windows\Temp\dotf-firstlogon.log'

$virtio = Get-PSDrive -PSProvider FileSystem |
    Where-Object { Test-Path (Join-Path $_.Root 'NetKVM\w11\ARM64') } |
    Select-Object -First 1
if (-not $virtio) { throw 'virtio-win media not found on any drive' }
$root = $virtio.Root

foreach ($driver in 'NetKVM', 'vioserial') {
    $inf = Join-Path $root "$driver\w11\ARM64\*.inf"
    & pnputil.exe /add-driver $inf /install
    if ($LASTEXITCODE -notin 0, 3010) { throw "pnputil $driver exited $LASTEXITCODE" }
}

# virtio-win ships the agent for x64 only; it runs under Arm64 emulation.
$msi = Join-Path $root 'guest-agent\qemu-ga-x86_64.msi'
$p = Start-Process -FilePath msiexec.exe -ArgumentList '/i', "`"$msi`"", '/qn', '/norestart' -Wait -PassThru
if ($p.ExitCode -notin 0, 3010) { throw "qemu-ga install exited $($p.ExitCode)" }
Set-Service -Name 'QEMU-GA' -StartupType Automatic

Stop-Transcript
# /s without /hybrid: a full shutdown, so clean-base holds no hibernated kernel.
& shutdown.exe /s /f /t 0
