---
tags: [spec, tasks, templates]
created: "2026-09-02"
---

# Tasks - OPS-043-setup-doctor-duplication

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers:**
> - `[P]` — no dependency on another unchecked task; safe to run in parallel.
> - `[AC<n>]` — helps satisfy acceptance criterion #`<n>` from `proposal.md`.
>
> **Ordering is load-bearing here (R3).** Every task under "Port" must be green
> before the single task under "Delete" runs. The reverse order opens the
> coverage hole this spec exists to avoid — on Windows it is already open.

## Setup

- [x] Branch created from main: `chore/ops-043-setup-doctor-duplication`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (R1 resolved by design: explicit map + join guard; R2 exemption list explicit; R3 ordering enforced below; R4 absence → SKIP)

## Implementation

### Port (must all be green before the Delete section)

- [ ] [P] [AC1] Failing test: `checkHomeDeployDrift` FAILs when `~/.zsh/functions.sh` differs from `~/.dotfiles/.zsh/functions.sh`, PASSes when equal. Table-driven over the three `.zsh` files, through the `System` seam.
- [ ] [AC1] Add `homeDeployMap` — the explicit deploy-dir→`$HOME` path map — and `checkHomeDeployDrift` in `cli/internal/doctor/checks_deploy.go`. New section `Deploy-dir↔$HOME drift`. Both-sides-present rule mirrors `checkDeployDrift` (R4: absent either side → SKIP, never FAIL).
- [ ] [AC1] Failing test: a symlink at `$HOME` FAILs even when it resolves to an identical file (`cmp` follows it; `checkSymlinks` PASSes it). Then implement — this is `check_deployed`'s severity that nothing else in doctor carries.
- [ ] [AC2] Failing test: the four exempt entries (`.gitconfig`, `.zshrc`, `.bashrc`, `.profile`) report PASS with drifted content; every other entry FAILs. Then encode the exemption in `homeDeployMap` as an explicit `contentChecked: false`, each with its own measured reason (R2 — `.gitconfig` observed drifting on msi; the three RC files on the installer-append mechanism).
- [ ] [AC2] Correct the stale clause in the `setup-linux.sh:1586-1589` NOTE: no `sed -i` remains in the script (R5). One-line comment fix, in this PR, not a follow-up.
- [ ] [P] [AC3] Failing test: `docker compose` reports a verdict, probed as the v2 CLI plugin (`docker compose version`) with the standalone v1 binary as fallback, reported optional. Then implement. Do **not** add the bare string to `coreTools` — the repo provisions compose nowhere, so a FAIL is wrong; and a PATH test alone SKIPs on a box where the plugin works.
- [ ] [AC3] Table-driven parity test enumerating all 14 `check_dependencies` tools and the 3 `check_deployed` files, asserting each has an equal-or-stronger doctor verdict than the shell call it replaces, plus the symlink row. This test is the evidence for the delete, not a description of it.
- [ ] [P] [AC5] Guard test (Go, not bats): every `deploy_file … "$HOME/…"` call site in `setup-linux.sh` has a `homeDeployMap` entry. Read the script from the repo root the way `env_test.go:77` and `prtriage_test.go:134` already do.
- [ ] [P] [AC4] Guard test (bats): the six REFACTOR-002 pre-exports are present in `setup-linux.sh`. Comment names BUG-021 and #1337 so the next reader sees why they are not deletable.

### Delete (only after every Port task is ticked)

- [ ] [AC6] Remove the three `check_deployed` calls (`setup-linux.sh:1583-1585`) and the `check_dependencies` call (`:1597`). Leave the two NOTE comments' rationale where it now belongs — in `homeDeployMap`. Do **not** touch lines 1657-1668 (the pre-exports) or `checkDeployDrift`.

### Cleanup

- [ ] Are `check_deployed` / `check_dependencies` still called anywhere? Grep repo-wide (`scripts/`, `tests/`, both setup scripts) before deleting the function bodies from `scripts/utils.sh` — the lesson from OPS-040 is that things look inert until the grep widens. If unreferenced, delete the bodies and their tests too; if referenced, leave them and say where.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] `cd cli && go build ./... && go vet ./... && go test ./...` green
- [ ] `GOOS=windows go vet ./...` green (the Windows leg compiles the same tree)
- [ ] `golangci-lint run` with the pinned version from `versions.conf` (BUG-071)
- [ ] `~/.local/bin/shellcheck setup-linux.sh` + `~/.local/bin/bats tests/*.bats` green
- [ ] Setup LOC re-measured and recorded in `verification.md` (baseline 3991)
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

See sibling `features.json`. Pass-state gating: the agent cannot write
`"state": "passing"` — only the harness, after running `verification` and
capturing exit 0, may set that terminal state.
