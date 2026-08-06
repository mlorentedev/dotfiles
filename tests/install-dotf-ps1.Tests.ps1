# Pester 5 behavioral guard for install-dotf.ps1's binary placement (BUG-037).
#
# Windows refuses to overwrite or delete a *running* image, so `Copy-Item -Force`
# straight onto dotf.exe failed every upgrade attempted while a long-lived
# `dotf secrets run -- <agent>` wrapper held the binary open. Set-DotfBinary
# parks the live image and renames the staged one in, which Windows does allow.
#
# The .sh twin is covered behaviorally on Linux (tests/install-dotf.bats, real
# ETXTBSY). install-dotf-ps1.bats only greps the source — that is why this class
# of defect escaped CI on the Windows side, so the swap is exercised for real here.

BeforeAll {
    # Dot-sourcing defines the functions without tripping the standalone
    # run-guard (`$MyInvocation.InvocationName -ne '.'`), so nothing installs.
    . (Join-Path $PSScriptRoot '..\scripts\install-dotf.ps1')

    # Open $Path the way the Windows loader holds a running executable: readable
    # and renamable (FileShare.Read + Delete), but never overwritable. Returns the
    # handle; the caller disposes it to "exit" the simulated process.
    function New-RunningImageLock {
        param([string]$Path)
        [System.IO.File]::Open(
            $Path,
            [System.IO.FileMode]::Open,
            [System.IO.FileAccess]::Read,
            ([System.IO.FileShare]::Read -bor [System.IO.FileShare]::Delete))
    }
}

Describe 'Set-DotfBinary' {
    BeforeEach {
        $script:Dest = Join-Path ([System.IO.Path]::GetTempPath()) ('dotf-swap-' + [System.IO.Path]::GetRandomFileName())
        New-Item -ItemType Directory -Force -Path $script:Dest | Out-Null
        $script:Target = Join-Path $script:Dest 'dotf.exe'
        $script:Source = Join-Path $script:Dest 'staging-source.bin'
        Set-Content -LiteralPath $script:Source -Value 'NEW' -NoNewline
    }

    AfterEach {
        Remove-Item -Recurse -Force -LiteralPath $script:Dest -ErrorAction SilentlyContinue
    }

    It 'installs the binary when nothing is there yet' {
        Set-DotfBinary -Source $script:Source -Target $script:Target
        Get-Content -LiteralPath $script:Target -Raw | Should -BeExactly 'NEW'
    }

    It 'replaces an idle binary' {
        Set-Content -LiteralPath $script:Target -Value 'OLD' -NoNewline
        Set-DotfBinary -Source $script:Source -Target $script:Target
        Get-Content -LiteralPath $script:Target -Raw | Should -BeExactly 'NEW'
    }

    It 'cannot be done by overwriting a running image (the BUG-037 mechanism)' {
        # Guards the fidelity of the simulation itself: if this ever stops
        # throwing, the test below proves nothing.
        Set-Content -LiteralPath $script:Target -Value 'OLD' -NoNewline
        $lock = New-RunningImageLock -Path $script:Target
        try {
            { Copy-Item -LiteralPath $script:Source -Destination $script:Target -Force -ErrorAction Stop } |
                Should -Throw
        } finally {
            $lock.Dispose()
        }
    }

    It 'replaces a running binary and leaves the live process on the old image' {
        Set-Content -LiteralPath $script:Target -Value 'OLD' -NoNewline
        $lock = New-RunningImageLock -Path $script:Target
        try {
            Set-DotfBinary -Source $script:Source -Target $script:Target
            Get-Content -LiteralPath $script:Target -Raw | Should -BeExactly 'NEW'
            # The running process keeps reading its own (parked) image.
            $lock.Length | Should -Be 3
        } finally {
            $lock.Dispose()
        }
    }

    It 'leaves no staging artifact behind after a clean swap' {
        Set-DotfBinary -Source $script:Source -Target $script:Target
        Test-Path -LiteralPath "$($script:Target).new" | Should -BeFalse
        Test-Path -LiteralPath "$($script:Target).old" | Should -BeFalse
    }

    It 'clears a park left locked by a previous upgrade' {
        # Second upgrade in a row: the first one could not delete its park.
        Set-Content -LiteralPath "$($script:Target).old" -Value 'STALE' -NoNewline
        Set-Content -LiteralPath $script:Target -Value 'OLD' -NoNewline
        Set-DotfBinary -Source $script:Source -Target $script:Target
        Get-Content -LiteralPath $script:Target -Raw | Should -BeExactly 'NEW'
        Test-Path -LiteralPath "$($script:Target).old" | Should -BeFalse
    }

    It 'restores the previous binary when the source cannot be staged' {
        Set-Content -LiteralPath $script:Target -Value 'OLD' -NoNewline
        { Set-DotfBinary -Source (Join-Path $script:Dest 'absent.bin') -Target $script:Target } |
            Should -Throw
        # An aborted install must never leave the user without a dotf.
        Get-Content -LiteralPath $script:Target -Raw | Should -BeExactly 'OLD'
    }
}
