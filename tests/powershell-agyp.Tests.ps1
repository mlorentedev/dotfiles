# Pester 5 tests for the agyp function in powershell/profile.ps1 (PARITY-001,
# #764): the PowerShell twin of .zsh/functions.sh's agyp. The function is
# lifted out of the profile by its markers so nothing else in the profile runs,
# and agy is shimmed by a function of the same name that records its arguments.

Describe 'agyp' {

    BeforeAll {
        $profilePath = (Resolve-Path (Join-Path $PSScriptRoot '..\powershell\profile.ps1')).Path
        $text = Get-Content -LiteralPath $profilePath -Raw
        $m = [regex]::Match($text, '(?ms)^function agyp \{.*?^\}')
        if (-not $m.Success) { throw 'function agyp not found in profile.ps1' }
        . ([scriptblock]::Create($m.Value))

        $script:Gemini = Join-Path $TestDrive 'gemini'
        New-Item -ItemType Directory -Force -Path (Join-Path $script:Gemini 'prompts') | Out-Null
        Set-Content -LiteralPath (Join-Path (Join-Path $script:Gemini 'prompts') 'review.md') -Value 'Review this.' -NoNewline
        $env:GEMINI_DIR = $script:Gemini

        # The shim: whatever agyp launches lands here instead of the real binary.
        function global:agy { $script:AgyArgs = $args }
    }

    AfterAll {
        Remove-Item -Path Function:\global:agy -ErrorAction SilentlyContinue
        Remove-Item -Path Env:\GEMINI_DIR -ErrorAction SilentlyContinue
    }

    BeforeEach { $script:AgyArgs = $null }

    It 'launches agy -i with the prompt file, a blank line, and the extra words' {
        agyp review fix the tests
        $script:AgyArgs[0] | Should -Be '-i'
        $script:AgyArgs[1] | Should -Be "Review this.`n`nfix the tests"
    }

    It 'launches with the prompt alone when no extra words are given' {
        agyp review
        $script:AgyArgs[1] | Should -Be "Review this.`n`n"
    }

    It 'fails without launching agy when the prompt name is missing' {
        { agyp -ErrorAction Stop } | Should -Throw -ExpectedMessage '*usage: agyp*'
        $script:AgyArgs | Should -BeNullOrEmpty
    }

    It 'fails without launching agy when the prompt file does not exist' {
        { agyp nope -ErrorAction Stop } | Should -Throw -ExpectedMessage '*prompt not found*nope.md*'
        $script:AgyArgs | Should -BeNullOrEmpty
    }
}
