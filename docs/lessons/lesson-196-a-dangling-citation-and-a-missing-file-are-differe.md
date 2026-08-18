---
id: lesson-196-a-dangling-citation-and-a-missing-file-are-differe
type: lesson
status: active
created: "2026-08-12"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 196: A dangling citation and a missing file are different bugs — check for the first before assuming the second

**Context**: the docs truth-pass batch (#677/#681/#682/#683/#684), plus the standalone #738 ("fresh-machine bug issues cite an audit doc absent from the repo"). Nine-plus GitHub issues cited findings from `docs/audits/docs-audit-2026-07-07.md`, `docs/audits/codebase-audit-2026-07-06.md`, and `docs/audits/process-audit-2026-07-07.md`. None of those three paths existed in the working tree, and #738's own investigation (`git log --all -- 'docs/audits/*'`) had already shown none was ever committed — its proposed fix was either commit the audit (if it existed locally) or edit the citations to point wherever the findings actually live.

**Problem**: a literal filesystem search for those exact filenames *did* return three hits — but in a sibling repo (`kubelab`), not this one. Their content was about kubelab's own Docker/K3s/Proxmox migration, using the same D/C/P finding-ID scheme and the same audit methodology, dated within days of the dotfiles ones. Coincidence, not the source: evidently the same audit-generation process had been run across multiple repos around the same time, and the date-based filenames collided. Taking that hit at face value would have wrongly concluded "the audit is genuinely lost, resolve via option B" and started ticketing orphan findings that weren't actually orphans.

**Solution**: a full-text grep for one of the audit's own distinctive phrases (a finding's exact wording, quoted verbatim in an issue body) across `docs/adr/*.md` turned up `audit-008-codebase-comprehensive.md`, `audit-009-documentation.md`, and `audit-010-process-workflows.md` — all three audits, fully committed, dated exactly right, every cited finding ID present verbatim. They'd been committed under this repo's own `audit-NNN-topic.md` naming convention; the issues' `Source:` lines just cited a different, never-used naming scheme (`docs/audits/<type>-audit-<date>.md`) for the same documents. A `gh issue list --search "docs/audits"` swept for every other issue with the same dangling-path defect (24 total, not just the 4 named in #738) rather than fixing only the ones explicitly listed.

**Rule**: when a cited path doesn't resolve, don't stop at "confirmed missing" — grep the repo for the cited content (a distinctive phrase, a finding ID, a table row) before concluding it was never committed. A file matching the cited *name* in a different repo is not evidence either way; verify its content actually matches what's being cited, not just its filename. And once a citation-convention bug is found in one issue, search for every other issue with the same defect before fixing only the one that was pointed at — the same generation process that produced one dangling citation likely produced several.

**Tags**: `github`, `audit`, `documentation`, `verification`
