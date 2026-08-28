# Pester 5 wrapper for tests/copilot-native-skills-smoke.ps1: Deploy-SkillRecord
# end to end into a scratch COPILOT_HOME (handoff rendered, auxiliary file kept,
# claude-only skill filtered out, instructions file LF and BOM-free with the
# catalog injected) plus `copilot skill list` discovery where the CLI can run.
#
# The smoke script was invoked nowhere before this file existed: Pester
# discovers *.Tests.ps1 only, so the one test exercising the injector never ran
# in CI while the injector wrote CRLF for months (WIN-008/#1289, TEST-003/#1298).
# Skipped where copilot is not installed; the skip is computed at discovery
# time, not in BeforeAll (WIN-004 lesson).

$script:hasCopilot = [bool](Get-Command copilot -ErrorAction SilentlyContinue)
# On the runner setup installs copilot and the CI step declares it expects it
# (DOTFILES_CI_EXPECT_COPILOT=1): an absent binary there is a broken PATH or a
# broken install, not a machine without Copilot, and it must fail loudly. The
# skip stayed green on every CI run for months (TEST-006, #1320).
$script:expectCopilot = $env:DOTFILES_CI_EXPECT_COPILOT -eq '1'

Describe 'copilot presence on a runner that declared it' -Skip:(-not $script:expectCopilot) {
    It 'copilot resolves on PATH when DOTFILES_CI_EXPECT_COPILOT=1' {
        # Re-probed at run time: Pester 5 does not carry discovery-time script
        # variables into the Run phase.
        [bool](Get-Command copilot -ErrorAction SilentlyContinue) | Should -BeTrue -Because 'setup installed copilot and the Pester step refreshed PATH; a skip here would pass a broken injector as green'
    }
}

Describe 'copilot native skills (Deploy-SkillRecord end to end)' -Skip:(-not $script:hasCopilot) {

    It 'deploys handoff with LF, no BOM, and the catalog injected' {
        $smoke = Join-Path $PSScriptRoot 'copilot-native-skills-smoke.ps1'
        $out = & $smoke 2>&1 | Out-String
        $out | Should -Match 'deployed handoff \(LF, no BOM\)'
    }
}
