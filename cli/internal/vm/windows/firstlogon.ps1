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
# Its MSI also registers a VSS provider in COM+, which fails on Arm64
# (error 1722) and rolls the install back, and `qemu-ga -s install` does the
# same registration first. So the MSI is only extracted (an administrative
# install runs no custom action) and the service is created as the MSI would
# create it. Without the provider the agent answers every command but
# filesystem freeze, which nothing here uses.
$msi = Join-Path $root 'guest-agent\qemu-ga-x86_64.msi'
$dir = Join-Path $env:ProgramFiles 'qemu-ga'
$p = Start-Process -FilePath msiexec.exe -ArgumentList '/a', "`"$msi`"", '/qn', "TARGETDIR=`"$dir`"" -Wait -PassThru
if ($p.ExitCode -ne 0) { throw "qemu-ga extraction exited $($p.ExitCode)" }
$exe = (Get-ChildItem -Path $dir -Recurse -Filter 'qemu-ga.exe' | Select-Object -First 1).FullName
if (-not $exe) { throw "qemu-ga.exe not found under $dir" }
New-Service -Name 'QEMU-GA' -DisplayName 'QEMU Guest Agent' -StartupType Automatic -BinaryPathName "`"$exe`" -d --retry-path" | Out-Null

# A run's converge stands in for an owner who answers Yes to every UAC
# prompt: an installer that elevates (Docker Desktop, the AWS CLI's MSI)
# otherwise waits on a prompt nobody sees and fails. The admin account
# elevates without asking; the guest holds no credential to protect.
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name 'ConsentPromptBehaviorAdmin' -Value 0

Stop-Transcript
# /s without /hybrid: a full shutdown, so clean-base holds no hibernated kernel.
& shutdown.exe /s /f /t 0
