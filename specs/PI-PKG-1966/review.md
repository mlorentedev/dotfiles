---
spec: "PI-PKG-1966"
verdict: "PASS WITH GAPS"
reviewed_sha: "8bb09c2f63f5b83212ceb7904a19492b8f2c8633"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-10-03"
---

## Adversarial review

**Scope**: PI-PKG-1966
**Sources**: `specs/PI-PKG-1966/{proposal,tasks,verification}.md` + `git diff b73267a0fb530aaf299ebdc008bb4fd82d464d34...HEAD`

### Spec and task alignment
- `pi-mcp-client` is removed from active packages and placed in `retire`.
- `pi-web-access` and `pi-subagents` are bumped to versions that include the `typebox` peer dependency.
- `pi -v` output is clean of extension warnings.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor    | REAL    | config | The proposal states `pi-mcp-client` was moved to `retire` "so it is cleanly uninstalled". This is a conceptual mismatch: uninstallation happens automatically because it is removed from the `packages` list. The `retire` array only archives state directories in `~/.pi/agent/`, not the npm installation. Adding it to `retire` results in `0 retired`. | `dotf pi packages apply` output logs `0 retired`. | `cli/internal/pi/packages_test.go` | code (remove from `retire` in `packages.json`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | Criteria met; npm packages successfully upgraded and correctly declare `typebox` as a peer dependency. |
| Verification       | B | Evidence covers criteria but lacks `dotf pi packages apply` output which would prove the clean uninstall mentioned in the spec. |
| Scope              | C | Significant unrelated changes mixed in the diff (includes `PI-002-windows-fixes`, `FIX-WIN-TUI-001`, and harness fixes). |
| Reliability        | A | Gracefully handles the invalid `retire` path by doing nothing; idempotent configuration. |
| Maintainability    | B | Minor clutter with the unnecessary `retire` entry for `pi-mcp-client`. |
| Handoff-readiness  | A | Spec updates included and tasks complete. |

### Verdict
PASS WITH GAPS

### Recommended next steps
- [code] Remove the `pi-mcp-client` entry from the `retire` array in `ai/pi/packages.json` to avoid confusion about how extension uninstalls work.
- The `dotf spec archive` is advisable in the current state since the gap is Minor and does not affect functionality.
