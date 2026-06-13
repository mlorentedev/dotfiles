---
tags: [spec, verification, templates]
created: "2026-06-13"
---

# Verification - CLI-009-setup-install-dot

## Evidence

Each acceptance criterion maps to a bats test and/or an observed end-to-end run.

- [x] `DOT_VERSION` in `versions.conf`, no hard-coded version elsewhere → `versions.conf` + `grep` guard in features.json f1
- [x] `install_dot` maps host OS/arch and installs to `~/.local/bin` → `tests/install-dot.bats` "happy path" + **real-release smoke** (`./scripts/install-dot.sh 0.1.0 <dest>` → `dot version` = `0.1.0`)
- [x] Unmapped arch → clear failure, no install → `tests/install-dot.bats` "arch mapping" (`_dot_arch i686` non-zero)
- [x] Checksum mismatch → install aborts, no binary left → `tests/install-dot.bats` "checksum mismatch" + "missing checksum entry"
- [x] Idempotent / converges on drift → `tests/install-dot.bats` "idempotent: no-op when pinned version on PATH"
- [x] `healthcheck.sh` reports `dot` + flags drift → section 6 block; `tests/healthcheck.bats` (shellcheck-clean) green
- [x] `cli.yml` `release` gated on `test` + `lint` → `needs: [test, lint]` (features.json f7 yaml assertion PASS)
- [x] shellcheck clean on changed scripts; full bats suite green → see Test status

## Test status

- `~/.local/bin/shellcheck scripts/install-dot.sh scripts/healthcheck.sh` → clean. `setup-linux.sh` shows only **pre-existing** `info`/`style` notes (SC1091/SC2015 in unrelated blocks); my additions add none (the `. install-dot.sh` source carries a `# shellcheck source=/dev/null` directive). CI's lint passes on main with those notes.
- `~/.local/bin/bats tests/install-dot.bats` → 5/5 (arch mapping, happy path, checksum mismatch, missing entry, idempotence). `versions-conf.bats` accepts the new `DOT_VERSION` line.
- Full `tests/*.bats`: 1144 ok / 3 not ok. The 3 are pre-existing **`shell-profile.bats`** timing-flaky tests (4/5/6) that fail identically on pristine `main` (verified) and are CI-green — untouched by this change (zero diff to `shell-profile.{sh,bats}`). **No new failures introduced.**
- **End-to-end against the real CI-published release:** `install_dot 0.1.0` downloaded `dot_0.1.0_linux_amd64.tar.gz` from the GitHub release, verified its sha256 against `checksums.txt`, installed to `~/.local/bin`, and the binary reported `dot version 0.1.0` with `spec`/`review` subcommands present.
- Release pipeline proven: pushing `v0.1.0` ran `goreleaser release` in Actions (run 27461122270, success), publishing 6 platform artifacts + `checksums.txt`.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Sourced-detection guard rewritten.** The first `install-dot.sh` used `[ "${BASH_SOURCE[0]:-$0}" = "$0" ]` to gate auto-run. Empirically that fired on *source* in some harness contexts (would auto-run `install_dot` with no args at source time — noise/side-effect). Replaced with `if ! (return 0 2>/dev/null)`, which is the robust "am I executed vs sourced" idiom and tested correct across `bash -c` source, script-source (setup-like), and direct execution.
- **`install_dot` is parameterized (`version dest base_url`).** Defaults come from `DOT_VERSION` / `~/.local/bin` / the GitHub release, but the injectable `base_url` lets bats drive the whole download→verify→extract path against a `file://` fixture — no network, fully deterministic.
- **Checksum is a hard gate.** A mismatch or a missing checksums entry aborts with no binary placed in dest (security: never install an unverified/poisoned artifact). Both paths are tested.
- **Release-job hardening rode along.** The `cli.yml` `release` job had no `needs:` — it could publish from red code. Gated it on `test`+`lint` here, since setup now depends on releases being trustworthy (incident → guard).
- **Pre-existing `setup-linux.sh` shellcheck info/style notes left untouched** — out of scope (a REFACTOR-class cleanup), not introduced by this change.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-009-setup-install-dot/` -> `specs/archive/CLI-009-setup-install-dot/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
