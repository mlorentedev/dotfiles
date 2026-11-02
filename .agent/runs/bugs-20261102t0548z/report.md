# Run bugs-20261102t0548z ledger

Armed 2026-11-02T05:50Z, expires 17:50Z. Wake-lock: caffeinate -ims (AC power required).

| Item | State | PR | Evidence |
|---|---|---|---|
| #3048 #3049 #3033 | skipped: claimed by run night-20261102T0555Z | - | coordination messages 05:5xZ |
| #2957 | skipped: lives in run plat005's paths (converge) | - | - |
| #3050 | skipped: claimed by feat-worktree-one-root | - | - |
| #3007 | skipped: needs the owner's A/B design choice (CI/branch-protection or merge-time numbering) | - | issue body |
| #2851 | skipped: depends on CLI-026 slice 3 (another session) and deletes files in user dirs | - | issue body |
| #2784 | merged | #3057 `a57d175` | `go test ./internal/cmd -run TestForge` ok; mutation fails the new test; lint linux+windows 0 issues; CI green |
