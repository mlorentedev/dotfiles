<#
.SYNOPSIS
    Sensible Windows engineering defaults via HKCU registry tweaks (WIN-005).

.DESCRIPTION
    The mathiasbynens .macos analog for Windows: idempotently sets ~15
    universally-engineered defaults (Explorer visibility, taskbar noise,
    advertising/telemetry prompts, Bing-in-Start, dark mode) with plain
    Set-ItemProperty calls.

    Invariants:
      - HKCU only. No admin required, ever. The -Root parameter is validated
        so even the test seam cannot escape HKCU.
      - Idempotent. Values are read before writing; a second run applies 0
        changes and says so.
      - Never restarts explorer.exe. Some keys only take effect after Explorer
        restarts or you sign out -- the script tells you, you decide.

    Opt-in from setup: setup-windows.ps1 -WithDefaults (OFF by default).

.PARAMETER Root
    Registry root to write under. Defaults to HKCU:. Exists as a test seam so
    the Pester suite can target a throwaway sandbox key; must stay inside HKCU.

.EXAMPLE
    .\windows-defaults.ps1
    Apply the defaults to the current user.

.EXAMPLE
    .\windows-defaults.ps1 -Root 'HKCU:\Software\my-sandbox'
    Write the same tree under a sandbox key (used by the tests).

.NOTES
    - Registry paths live in the $Defaults table below (R2: named constants,
      one place to patch when a Windows update moves a key).
    - Win10 vs Win11 keys are branched on OS build 22000 (R3) via
      [System.Environment], so detection needs no machine-hive registry read.
#>

[CmdletBinding()]
param(
    [string]$Root = 'HKCU:'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# HKCU-only invariant, enforced at runtime too (the bats suite enforces it
# structurally). Anything else would imply admin and is out of scope.
if ($Root -notmatch '^HKCU:') {
    throw "windows-defaults.ps1 writes HKCU only; refusing root '$Root' (WIN-005 anti-scope)"
}

# Win11 starts at build 22000; taskbar keys differ across that line (R3).
$build = [System.Environment]::OSVersion.Version.Build
$isWin11 = $build -ge 22000

# --- R2: every key/value in one named table -------------------------------
# Path is relative to $Root. Applies: 'all' | 'win10' | 'win11'.
$Defaults = @(
    # Explorer
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced';     Name = 'HideFileExt';      Value = 0; Note = 'Explorer: show file extensions' }
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced';     Name = 'Hidden';           Value = 1; Note = 'Explorer: show hidden files' }
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced';     Name = 'LaunchTo';         Value = 1; Note = 'Explorer: open to This PC (not Quick Access)' }
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\Explorer\CabinetState'; Name = 'FullPath';         Value = 1; Note = 'Explorer: full path in title bar' }
    # Taskbar
    @{ Applies = 'win11'; Path = 'Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced';     Name = 'TaskbarAl';        Value = 0; Note = 'Taskbar: left-align (Win11)' }
    @{ Applies = 'win11'; Path = 'Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced';     Name = 'TaskbarDa';        Value = 0; Note = 'Taskbar: disable widgets (Win11)' }
    @{ Applies = 'win10'; Path = 'Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced';     Name = 'TaskbarSmallIcons'; Value = 1; Note = 'Taskbar: smaller icons (Win10)' }
    @{ Applies = 'win10'; Path = 'Software\Microsoft\Windows\CurrentVersion\Feeds';                 Name = 'ShellFeedsTaskbarViewMode'; Value = 2; Note = 'Taskbar: disable news and interests (Win10)' }
    # Privacy
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\AdvertisingInfo';       Name = 'Enabled';          Value = 0; Note = 'Privacy: disable advertising ID' }
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\Privacy';               Name = 'TailoredExperiencesWithDiagnosticDataEnabled'; Value = 0; Note = 'Privacy: disable tailored-experiences prompts' }
    # Search
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\Search';                Name = 'BingSearchEnabled'; Value = 0; Note = 'Search: disable Bing in Start menu' }
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\Search';                Name = 'CortanaConsent';   Value = 0; Note = 'Search: disable Cortana web search' }
    # Theme
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\Themes\Personalize';    Name = 'AppsUseLightTheme'; Value = 0; Note = 'Theme: dark mode for apps' }
    @{ Applies = 'all';   Path = 'Software\Microsoft\Windows\CurrentVersion\Themes\Personalize';    Name = 'SystemUsesLightTheme'; Value = 0; Note = 'Theme: dark mode for system' }
)

$applied = 0
$alreadySet = 0
$osLabel = if ($isWin11) { 'win11' } else { 'win10' }

Write-Host "Windows engineering defaults (HKCU, build $build -> $osLabel profile)" -ForegroundColor Cyan

foreach ($entry in $Defaults) {
    if ($entry.Applies -ne 'all' -and $entry.Applies -ne $osLabel) {
        continue
    }

    $key = Join-Path $Root $entry.Path
    if (-not (Test-Path $key)) {
        New-Item -Path $key -Force | Out-Null
    }

    # R4: read before write -- idempotency with honest logs.
    $item = Get-ItemProperty -Path $key -Name $entry.Name -ErrorAction SilentlyContinue
    $current = if ($item) { $item.($entry.Name) } else { $null }

    if ($current -eq $entry.Value) {
        Write-Host "[OK]  $($entry.Note) (already set)" -ForegroundColor Green
        $alreadySet++
    } else {
        Set-ItemProperty -Path $key -Name $entry.Name -Value $entry.Value -Type DWord
        Write-Host "[SET] $($entry.Note)" -ForegroundColor Yellow
        $applied++
    }
}

Write-Host ""
Write-Host "Applied $applied, already set $alreadySet" -ForegroundColor Cyan
if ($applied -gt 0) {
    # R1: some keys only take effect after Explorer reloads. Inform, never act.
    Write-Host "Some changes need an Explorer restart to show: sign out/in, or restart explorer.exe from Task Manager yourself." -ForegroundColor Yellow
}
