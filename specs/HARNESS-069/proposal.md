---
id: "HARNESS-069"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-12"
issue: "mlorentedev/dotfiles#917"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-069

> **Naming**: file lives at `<repo>/specs/HARNESS-069/proposal.md`. `HARNESS-069` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

`compile-harness.sh --deploy` already stamps every rendered $HOME copy with
`generated: true` / `generated_from` / `generated_sha` provenance, because a
deployed skill/agent file must not look hand-authored. But `--refresh`'s
committed records — `harness/skills/<name>/SKILL.md`, `harness/agents/<name>/AGENT.md`
— are a straight `cp -rf` of the vault source, byte-identical, starting
straight at `id:`. That is the copy a human actually opens when working *in
the repo*, and it carries no sign it is generated at all. A 2026-08-09 session
read the Regla-del-3 gate off `00_meta/patterns/` (audited in `00_meta/patterns/`
itself) without ever noticing the record was a derivative, because nothing in
the file said so.

## What

`--refresh` stamps each committed skill/agent record's own frontmatter with
`generated: true`, `generated_from: <vault-relative source path>` and
`generated_sha: <sha_of the vault source at refresh time>` — the same
mechanism `render_skill`/`render_agent` already use for deploy output, applied
in place via a new `inject_record_provenance` helper instead of rendered to
stdout. `render_skill`'s deploy-time injection is updated to strip any
pre-existing `generated`/`generated_from`/`generated_sha` lines from its
source before injecting its own, so a record's own provenance (its relation to
the vault) and a deployed copy's provenance (its relation to the record) never
stack into two conflicting sets in the same file. `render_agent` already only
passes through `name`/`description` in frontmatter, so it drops the record's
provenance naturally — verified, not assumed.

## Out of scope

- Changing `--check`'s validation: it already validates records by schema +
  clean render, never by byte-identity to the vault, so this needs no change
  there — confirmed by reading `do_check` before writing any code.
- `setup-windows.ps1`'s parallel deploy logic (CLI-026 territory; #833/#828
  cover the broader Windows-parity gap, not this spec).
- A doctor/CLI check that reads these fields — none exists today (`generated_sha`
  is grep-only elsewhere in this repo, not machine-consumed by `cli/`), and
  none is proposed here.

## Risks / open questions

- **Resolved before implementation**: does `--check` compare committed records
  byte-for-byte against the vault? No — `do_check`'s skill/agent branches call
  `validate_skill_frontmatter` (schema required-keys only, no
  `additionalProperties` enforcement) and `render_skill`/`render_agent`
  (renders cleanly), never a diff against `$VAULT_PATH`. Confirmed by reading
  `do_check` directly; the record no longer being byte-identical to the vault
  is therefore not a risk.
- **Resolved before implementation**: would injecting the record's own
  `generated_*` fields duplicate at deploy time, since `render_skill` already
  injects its own? Yes, for the `skill` and `command` render kinds — fixed by
  adding a strip rule to `render_skill`'s awk. `render_agent` was already safe
  (its passthrough is an allowlist of `name`/`description` only). Verified
  with a real `--deploy` to a scratch `$HOME`: exactly one set of the three
  fields in the deployed output, not two.

## Acceptance criteria

- [x] `compile-harness.sh --refresh` writes `generated: true` /
      `generated_from` / `generated_sha` into every committed
      `harness/skills/<name>/SKILL.md` and `harness/agents/<name>/AGENT.md`,
      pointing at the vault source it was refreshed from.
- [x] `compile-harness.sh --deploy` output still carries exactly one set of
      `generated_*` fields (describing the $HOME copy's relation to the
      record), not two stacked sets.
- [x] `compile-harness.sh --check` still passes clean, offline, with no
      change to its own logic.
- [x] Existing bats coverage updated to assert the new record content instead
      of asserting its absence; new assertions pin the no-duplication
      invariant at deploy time for both skills and agents.

## References

- Bitácora board: #917 (this spec's issue), filed as a follow-up to #916/DOCS-013
- Prior art: `render_skill`/`render_agent`'s existing deploy-time provenance
  mechanism, `scripts/compile-harness.sh:192-210` (before this change)
- Related: #833 (HARNESS-059, the never-hand-edit rule this record's own
  provenance line helps make self-evident) — not implemented here, re-scoped
  onto this spec per the 2026-08-11 handoff
