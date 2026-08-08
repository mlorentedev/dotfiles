---
id: "HARNESS-054"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#817"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, cross-agent]
template_version: "1.0"
---

# HARNESS-054

## Why

<!-- from issue #817: HARNESS-054: agy and codex receive no doctrine — enforced and presence regions stop at four agents -->

A rule that reaches four agents out of six is not a rule, it is a coin flip decided by whichever client the work happens in. Claude, opencode, pi and copilot each carry the enforced policy rules and the agent-presence block; Antigravity received the largest skill payload of any harness (34 skills) and no rules at all, and codex had no row in the manifest whatsoever. Cross-agent parity is the entire point of the pipeline, so the two surfaces that receive nothing are the ones that decide whether the doctrine is real.

## What

`compile-harness.sh --deploy` delivers a **compact doctrine payload** — the enforced rules plus the agent-presence block, about 2 KB — to `~/.gemini/GEMINI.md` (agy) and `~/.codex/AGENTS.md` (codex), creating the file when absent and injecting into a marked region when it already exists. Both platforms document a size limit that a full 21851-character `AGENTS.md` cannot satisfy: Antigravity caps each rules file at 12000 characters, and codex stops adding instruction files once the global-plus-project chain reaches 32 KiB, which would silently crowd out the repository's own `AGENTS.md`. The payload is therefore sized to what each platform actually reads, using the same content and the same injection mechanism as every other surface.

## Out of scope

- Skills deployment to codex. No primary source documents its skill-discovery path, and inferring one is the guess this ticket exists to remove.
- Action-level determinism (hooks) for either agent — cross-agent hook emission is #561, copilot hooks are #803.
- Slimming `AGENTS.md` itself. The 32 KiB codex chain cap is an argument for it, but it is GOV-004 #673's scope.

## Risks / open questions

- **Marker tolerance (unverified).** Neither agent is installed on the development machine, so it is reasoned — not observed — that Antigravity's `GEMINI.md` loader tolerates HTML comment markers. It is a markdown file and the same markers survive in four other harnesses.
- **Shared file.** `~/.gemini/GEMINI.md` is also written by the Gemini CLI (google-gemini/gemini-cli#16058, closed as not planned). Mitigated by injecting a region and never overwriting the file; covered by a test.
- **Shadowing.** Codex reads `AGENTS.override.md` in preference to `AGENTS.md`, so a deploy could look successful while changing nothing the agent reads. Mitigated by a deploy-time warning.
- **Secondary sources disagree with the official docs** on whether Antigravity natively reads `AGENTS.md`; the official documentation mentions only `GEMINI.md` and workspace `.agents/rules`, so nothing here is built on the unconfirmed claim.

## Acceptance criteria

- [x] Every agent surface the manifest declares receives both the enforced rules and the presence block, or is recorded as out of scope with a reason.
- [x] A test fails when a declared agent surface is missing its region.
- [x] Injection preserves pre-existing user content and is idempotent across re-deploys.
- [x] A target that exceeds its platform's documented cap produces a warning naming both numbers.
- [x] A codex override file that shadows the deployed doctrine produces a warning.
- [x] The agy and codex decisions, with their byte numbers and sources, live next to the manifest rows rather than only in the pull request.

## References

- Bitácora board: mlorentedev/dotfiles#817 (see the `issue:` frontmatter field)
- Related patterns: `00_meta/patterns/pattern-cross-agent-skill-pipeline.md`, `00_meta/patterns/pattern-cross-agent-agent-pipeline.md` (ADR-027)
- [Antigravity — Rules & Workflows](https://antigravity.google/docs/rules-workflows) — global rules path, 12000-character per-file cap
- [Codex — Custom instructions with AGENTS.md](https://learn.chatgpt.com/docs/agent-configuration/agents-md.md) — `~/.codex` home, `AGENTS.override.md` precedence, `project_doc_max_bytes` 32 KiB
- [google-gemini/gemini-cli#16058](https://github.com/google-gemini/gemini-cli/issues/16058) — the shared-`GEMINI.md` collision
