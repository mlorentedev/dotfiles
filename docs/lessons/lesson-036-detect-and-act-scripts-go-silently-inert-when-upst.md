---
id: lesson-036-detect-and-act-scripts-go-silently-inert-when-upst
type: lesson
status: active
created: "2026-05-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 036: detect-and-act scripts go silently inert when upstream products change their surface

**Context:** dotfiles setup-windows.ps1 and setup-linux.sh detected GitHub Copilot CLI via `gh extension list | grep github/gh-copilot`. GitHub released a new standalone `copilot` CLI (winget GitHub.Copilot, agentic interface closer to Claude Code than to the legacy suggest/explain wrappers). On machines with the new v2 installed and operative, the script logged "extension not installed, skipping" and never deployed ~/.copilot/copilot-instructions.md. The AI-013 pointer-style refactor was functionally inert on those machines. Discovered 2026-05-18 only when the user typed `gh extension install github/gh-copilot` and got "matches built-in alias" error -- revealing both products existed.</context>
<problem>Detect-and-act scripts assume the upstream product's surface stays stable. When the vendor ships a new product with the same conceptual purpose but different binary/CLI/install path, the detection silently misses it. Two failure modes: (1) deploy never fires (config missing); (2) the script LOGS "success" because the detect branch was simply skipped (no fail, no warn -- log says "not installed, skipping"). The longer the gap between vendor change and detection update, the more machines drift silently.</problem>
<solution>Three layers. (1) Detect on the BINARY not on a package-manager extension list: `Get-Command copilot` / `command -v copilot`. Binaries are more stable as a surface than extension manager state. (2) Re-audit upstream products at least once per sprint -- vendor changes are frequent; quarterly audit is too slow. Add to sprint checklists: "any AI/dev-tool integration deployed in past 6 months -- verify upstream hasn't shipped a replacement product." (3) Annotate every detect-and-act block with an inline comment: `# Upstream: https://...  Last validated: YYYY-MM-DD`. Forces the re-audit habit when the comment goes stale.</solution>
**Tags:** `#setup-scripts` `#detect-and-act` `#upstream-drift` `#copilot` `#silent-failure`
