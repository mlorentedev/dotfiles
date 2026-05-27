---
id: "POLISH-004-cross-os-polish-bundle"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# POLISH-004-cross-os-polish-bundle

> **Naming**: file lives at `<repo>/specs/POLISH-004-cross-os-polish-bundle/proposal.md`. `POLISH-004-cross-os-polish-bundle` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: Two trivia ports from mathiasbynens — `.editorconfig` at repo root (cross-OS consistency) + `.inputrc` (case-insensitive readline completion, smart history search, `set show-all-if-ambiguous on`). Effort: S, ~30 LOC. Anti-scope: don't reformat existing files; defaults only, no custom keybindings. -->

Two small cross-OS polish items from mathiasbynens, classed "trivia" by the prior `research/dotfiles-survey.md` and skipped from the IDEAS-001..006 batch. They're worth shipping as a single small bundle now that broader audit is happening: `.editorconfig` (cross-IDE consistency, zero runtime overhead) and `.inputrc` (case-insensitive readline tab-complete + smart history-jump). Both are universally-applicable defaults that improve daily ergonomics with no downside.

## What

Two new files at repo root:

- `.editorconfig` — `root = true`, `[*]` defaults (UTF-8, LF except `.ps1`/`.bat`/`.cmd` CRLF, `trim_trailing_whitespace = true`, `insert_final_newline = true`), per-extension stanzas (`.md` indent 2-space + keep trailing whitespace for line breaks, `.py` indent 4-space, `.sh`/`.ps1` indent 4-space, `.bats` indent 2-space).
- `.inputrc` — `set completion-ignore-case on`, `set show-all-if-ambiguous on`, `set show-all-if-unmodified on`, `set colored-stats on`, history-search-backward bound to `\e[A` (up arrow), `\e[B` (down arrow). Deployed to `$HOME/.inputrc` by `setup-linux.sh`.

## Out of scope

- **Reformatting existing files** to match `.editorconfig` — separate ticket if any pre-existing file violates the rules; this PR's scope is the file itself + a validation pass.
- **Custom keybindings** beyond history-search — advanced users add via a future `.inputrc.local` if a use case emerges.
- **Windows PSReadLine equivalent** — its own config domain; separate POLISH-XXX.
- **`.editorconfig` enforcement in CI** — file ships now; CI integration (editorconfig-checker action) is a separate POLISH-XXX once we know nothing breaks.

## Risks / open questions

- **R1**: `.editorconfig` triggers IDE reformatting on save in editors that respect it (VS Code, JetBrains). Confirm no current file violates the chosen rules; if any do, fix in this PR or relax that rule. Audit via `editorconfig-checker -no-color .`.
- **R2**: `.inputrc` history-search binding may conflict with a user's existing `~/.inputrc` if they bring one. Guard with `$if Bash` ... `$endif` blocks where appropriate and document the override path.
- **R3**: `.bats` indent convention — verify the existing tests' style. If 2-space, `[*.bats]` indent_size = 2; if 4-space, set accordingly. Don't introduce reformat churn.

## Acceptance criteria

- [ ] `.editorconfig` exists at repo root.
- [ ] `editorconfig-checker -no-color .` exits 0 against the current tree (or PR documents any pre-existing violations and explicitly excludes them).
- [ ] `.inputrc` exists at repo root.
- [ ] `setup-linux.sh` deploys `.inputrc` to `$HOME/.inputrc`.
- [ ] New bats test asserts `.inputrc` deployment + content presence post-`setup-linux.sh`.
- [ ] README "Features" section briefly mentions both with one-line descriptions.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § "Session 2026-05-27 — fresh-eyes audit" → POLISH-004.
- Inspiration: mathiasbynens/dotfiles `.editorconfig`, `.inputrc`.
- Prior classification as "trivia": `research/dotfiles-survey.md` § (skipped, below the Top 6).
