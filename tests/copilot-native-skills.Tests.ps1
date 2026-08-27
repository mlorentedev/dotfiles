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

Describe 'copilot native skills (Deploy-SkillRecord end to end)' -Skip:(-not $script:hasCopilot) {

    It 'deploys handoff with LF, no BOM, and the catalog injected' {
        $smoke = Join-Path $PSScriptRoot 'copilot-native-skills-smoke.ps1'
        $out = & $smoke 2>&1 | Out-String
        $out | Should -Match 'deployed handoff \(LF, no BOM\)'
    }
}
