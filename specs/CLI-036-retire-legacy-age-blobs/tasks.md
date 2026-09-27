---
tags: [spec, tasks, templates]
created: "2026-09-25"
---

# Tasks - CLI-036-retire-legacy-age-blobs

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Gating issue open and self-assigned: #938
- [x] Worktree `../dotfiles-wt-age-blobs`, branch `fix/retire-legacy-age-blobs` from `origin/main` (`f9d5b25`)
- [x] `proposal.md` complete; the prune semantic comes from #802's recorded decision
- [x] Owner approves the plan before any code (2026-09-25: "continua pero no acumules deuda tecnica")

Heavy runs go through `systemd-run --user --scope -p MemoryMax=3G -p MemorySwapMax=0`.

## Implementation

- [x] [AC2] Rewrite `TestCheckSecrets_BwBackedEntriesAreNotAgeAsserted`'s orphan half: an unclaimed `chatgpt.api-key.secret.age` next to a bw entry must be a FAIL naming it, and the output must not contain `Tracked as #971`. Run `go test ./internal/doctor -run TestCheckSecrets`. Expected: FAIL (today it is one WARN).
- [x] [AC2] In `cli/internal/doctor/checks_deploy.go`, `reportUnreferencedBlobs` FAILs per blob regardless of `migrated`; drop the `migrated` counter. Expected: the test passes; `TestCheckSecrets` still passes.
- [x] [AC3] Write `TestCheckSecrets_FixPrunesUnclaimedMirrorBlob`: mirror `~/.dotfiles/sensitive/` holds `old.secret.age` and `old.secret.age.tmp.123`; a checkout (via `DOTFILES_REPO_DIR`) holds `secrets/registry.yaml` and `sensitive/dr/bitwarden-export.age` but neither blob. With `fix=true`: both files gone, two `[FIX ]` lines; a second run prints no `[FIX ]`. Expected: FAIL (`checkSecrets` takes no `fix`).
- [x] [AC3] Thread `fix` into `checkSecrets` (`doctor.go:102`) and implement the prune, following `checkHarnessMirrorOrphans`: resolve the checkout, refuse unless it differs from the mirror and holds both gate files, and remove only files absent from `<checkout>/sensitive/`. Expected: the test passes.
- [x] [AC4] Table test `TestCheckSecrets_FixRefuses` with four rows: blob still in the checkout; checkout without the escrow; mirror == checkout; `env-mapping.conf`, `README.md` and `dr/` present in the mirror. Each row: the files survive and the blob is still a FAIL. Mutation: drop the escrow gate; the second row fails. Record it in `verification.md`.
- [x] [AC1] Add `TestCommittedAgeBlobsAreClaimed` in `cli/internal/secrets` (repo-reading, like the existing registry drift tests): every `sensitive/*.secret.age` in the checkout is named by an age-backed entry. Expected: FAIL with 31 names.
- [x] [AC1] `git rm` the 31 unclaimed blobs, by explicit path. Expected: the test passes; `git ls-files 'sensitive/*.secret.age'` prints only `sensitive/id_ed25519.secret.age`.
- [x] [AC5] ADR-028 amendment (2026-09-26): the escrow is the floor; per-secret blobs exist only for `age-offline` entries; #971 superseded. Rewrite the `reportUnreferencedBlobs` comment to match.
- [x] [AC6] On msi: `go run ./cmd/dotf doctor --fix` from the worktree, then `ls ~/.dotfiles/sensitive/*.secret.age*` (expected: only `id_ed25519.secret.age`) and the owner runs `dotf secrets drift` (expected: 0 findings). Record both in `verification.md`.

## Closing

- [ ] `go build ./... && go vet ./... && go test ./...`, `golangci-lint run` (pinned), `GOOS=windows go vet ./...`
- [x] `features.json` verifications pass; `verification.md` filled
- [x] Lesson: a comment that grants a safety exemption ("these blobs ARE the floor") outlived the ADR it cited
- [ ] PR body closes #938, #971, #802; `## Review triage` recorded
