---
tags: [spec, verification, claude-mem, epipe]
created: "2026-06-06"
---

# Verification - 2026-05-27-claude-mem-heal-consumer-epipe

## Evidence

- [x] **AC1 (root-cause guard, functional, cross-OS)** → `tests/claude-mem-heal.bats`
  test 18 "consumer pipe: head -n1 races but sed -n 1p drains under SIGPIPE-ignore".
  Deterministic repro: `trap '' PIPE` (Node's disposition) + slow writer →
  `head -n1` reports a write error, `sed -n 1p` is clean. Confirmed manually:
  variant C (slow + trap) printed `printf: write error: Broken pipe` 5/5;
  variant D (slow, default SIGPIPE) clean 3/3.
- [x] **AC2 (.sh heal)** → tests 19 (`break; }; done` fixture) + 20 (`head -n1`
  fixture): both yield `sed -n 1p`, no `head -n1`, BUG-018 directive preserved.
  Were RED before the fix (produced `head -n1`), GREEN after.
- [x] **AC3 (.ps1 parity)** → `tests/claude-mem-heal-ps1.bats` tests 22–23: same
  structural outcome from `Repair-HooksJson` via pwsh.
- [x] **AC4 (idempotency)** → tests 21 (.sh) + 24 (.ps1): re-heal of the fixed
  form is a silent no-op, file byte-unchanged.
- [x] **AC5 (live)** → healed a copy of the real deployed
  `~/.claude/.../13.3.0/hooks/hooks.json`: `head -n1` 7 → 0, `sed -n 1p` 0 → 7;
  second run silent; output still valid JSON (6 hook groups / 7 commands);
  word-level diff shows ONLY `head -n1` → `sed -n 1p` (7×), nothing else touched.

## Test status

- `bats tests/claude-mem-heal.bats tests/claude-mem-heal-ps1.bats` → 25/26 ok.
  The single failure (test 14, `ensure_marketplace_compat_symlink creates the
  legacy symlink`) is **pre-existing and environment-only**: Git Bash/MSYS does
  not create a real symlink without Developer Mode, so `[ -L ... ]` fails
  locally; it passes on the Linux CI runner and is unrelated to this change
  (it failed identically before any edit in this branch).
- `bash -n scripts/claude-mem-heal.sh` → OK (bats test 1). ShellCheck not
  installed locally; runs in CI.
- `Invoke-ScriptAnalyzer scripts/claude-mem-heal.ps1 -Severity Error,Warning`
  → CLEAN (0).
- No regressions in `heal_mcp_json` / `heal_zod` / symlink tests (unchanged).

## Decisions made during implementation

- **`sed -n 1p` over `head -n1`** (not `1q`): prints line 1 but reads to EOF, so
  it drains the writer and never closes the pipe under it. No quotes/backslashes
  /newlines → no JSON-string or sed-script escaping, unlike a `{ read; printf;
  cat >/dev/null; }` drain.
- **Option A (sed-patch) over Option B (canonical template emission).** Kept the
  resilient sed-patch: it only rewrites the pipe shape and preserves upstream's
  `node` invocation, so it survives upstream *semantic* drift. A static
  self-emitted hooks.json would be fragile to new hook stages / changed args.
- **Removed two dead assignments** (`has_017` / `has_018`) in `heal_hooks_json`
  while rewriting (were computed but never read).
- **Live ~/.claude left untouched.** The deployed heal copy still carries the
  old logic until the next `setup` re-deploys scripts; verification used a copy.
  AC5 evidence is on the real file shape, no live mutation mid-session.

## Promotion candidates

- [ ] Lesson for vault `90-lessons.md`? **no** — project-specific; the
  cross-OS-via-timing nuance is recorded here + in the proposal.
- [ ] ADR-worthy? **no** — within the existing claude-mem-heal pattern (BUG-016/017).
- [ ] New cross-project pattern? **no** — single-project healer.

## Follow-ups surfaced (tracked, not done here — atomic PR)

- `.mcp.json` heal still emits `head -n1`; same race is theoretically possible
  when ≥2 candidates carry `mcp-server.cjs`. Lower blast radius (one-shot at MCP
  load, not per tool call). Decide separately whether to apply the same drain.
- `init-spec.{sh,ps1}` vault-gate regex `\[[ x-]\]` rejects `[~]` (paused)
  entries, so a resumed paused task can't be scaffolded without `-ForceNoVault`.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/2026-05-27-claude-mem-heal-consumer-epipe/` → `specs/archive/...`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (none)
