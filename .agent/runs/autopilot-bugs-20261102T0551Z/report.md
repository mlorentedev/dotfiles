# Run report: autopilot-bugs-20261102T0551Z

- Armed 2026-11-02T05:51:11Z, expires 2026-11-02T17:51:11Z.
- The loop started at 16:08Z, about 10h after arming. It ended at 16:20Z with reason `queue-empty`.
- Envelope: committed copy matched the working tree at start.
- Wake-lock: `caffeinate -i -s -t 43200`.

| Item | Outcome |
|---|---|
| #3050 | Skipped. Another session closed it with the opposite direction (re-vendor into the CLI, #3063 / `0ed20bbe`). Reverting the vault now would re-break the drift test. |
| #3031 | Closed as could not reproduce: 10/10 full `go test ./...` runs on `de1a4934`, `cli/internal/cmd` ok in all, no ENOENT. Evidence in the issue comment. |
| #3033 | Already closed by another session. |
| #3007 | Not eligible: blocked on #2249 (merge queue). |
| #2896 | Out of scope: what is left is a vault skill edit compiled into `harness/**`, which this envelope freezes. Another session holds it. |
| #3051 | Already closed by another session. |

PRs opened: 0. Merges: 0. Release #3025 untouched.

Found while working: `TestEmbeddedTemplatesMatchVault` still fails on main for `work-sdk-memory.md` (`created: "{{date}}"` from vault `e8a90a65`). Already ticketed as #3062.
