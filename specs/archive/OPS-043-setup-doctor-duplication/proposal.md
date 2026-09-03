---
id: "OPS-043-setup-doctor-duplication"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-02"
issue: "mlorentedev/dotfiles#1337"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-043-setup-doctor-duplication

> **Naming**: file lives at `<repo>/specs/OPS-043-setup-doctor-duplication/proposal.md`. `OPS-043-setup-doctor-duplication` is `AREA-NNN-slug`.

## Why

<!-- from issue #1337: OPS-043: setup duplicates dotf doctor: check_deployed/check_dependencies and six pre-exported contract vars run eleven lines before doctor covers the same -->

`setup-linux.sh` ends with a shell verification layer — `check_deployed` ×3 and
`check_dependencies` over 14 tools — eleven lines before it hands off to
`dotf doctor`. ADR-020 makes `dotf doctor` the single post-setup diagnostic, and
`setup-windows.ps1` already converged there (CLI-018): it carries **no**
`check_deployed`/`check_dependencies` twin at all. Linux keeping a second
verification surface is the drift.

**But the issue's premise is false in five places, and measuring that is most of
this spec's value.** Deleting what #1337 names would remove coverage nothing
replaces:

1. **`check_deployed` is not duplicated.** It compares the deploy dir to `$HOME`
   (`~/.dotfiles/.zsh/aliases.zsh` vs `~/.zsh/aliases.zsh`, `cmp -s`).
   `checkDeployDrift` (`checks_deploy.go:915`) compares the **repo** to the
   **deploy dir**. Two different legs of `repo → ~/.dotfiles → $HOME`.
2. **No doctor section compares bytes on the second leg.** All 40 sections were
   enumerated. `checkSymlinks` (`checks_deploy.go:20`) and `checkProfileFiles`
   (`checks_profile.go:45`) test *existence*; a real file that has drifted from
   the repo reports `PASS` ("exists (not a symlink)"). So `check_deployed` is
   today the **only** content assertion on `deploy-dir → $HOME` — and since
   Windows never had it, that leg is **already unguarded on Windows**.
3. **`check_dependencies` never fails.** `scripts/utils.sh:359-372` only
   `log_warning`s. Of its 14 tools: **9** sit in doctor's `coreTools` (FAIL —
   strictly stronger, genuine duplication), **4** (helm, terraform, ansible, pip)
   sit in `optionalTools` (SKIP — advisory, same practical severity as the WARN
   it replaces), and **`docker-compose` is covered nowhere** in doctor, the
   contract, or the package lists.
4. **`.zsh/functions.sh` is in no doctor list.** `checkSymlinks` covers
   `aliases.zsh` and `functions.zsh` only.
5. **The six pre-exported vars are not duplication.** `setup-linux.sh:1657-1668`
   exports them precisely *because* the setup shell has not re-sourced the RC
   files, so doctor would otherwise report false warnings (BUG-021, and the
   Windows twin at `setup-windows.ps1:2185` carries the same fix). Removing them
   reintroduces the bug they closed.

So the honest shape is strangler-fig (ADR-020 §5), in this order: **port the
missing coverage into `dotf doctor` first, then delete the shell layer.** The
delete alone is a regression; the port alone leaves the duplication.

## What

After this PR:

- `dotf doctor` gains a **`Deploy-dir↔$HOME drift`** section that byte-compares
  every file setup deploys into `$HOME` against its `~/.dotfiles` source, driven
  by an explicit deploy-dir→`$HOME` path map (the two legs have **different**
  relative paths — `ssh/config` → `~/.ssh/config`, `tmux.conf` → `~/.tmux.conf` —
  so `isManagedDeployPath` cannot be reused). Drift → FAIL, naming both paths and
  the remedy. A **symlink** where a regular file is expected also FAILs, keeping
  `check_deployed`'s severity (ADR-012 moved deployment to copy, so a symlink
  there is a pre-ADR-012 leftover worth surfacing; `checkSymlinks` PASSes it and
  `cmp` follows it, so nothing else would catch it).
- **Exempt from the content comparison** (existence only), measured on msi
  2026-09-02 rather than inherited: `.gitconfig` (**observed drifting** — every
  `git config --global` on the box rewrites it), plus `.zshrc`, `.bashrc` and
  `.profile`, whose exemption rests on a *named mechanism* — tool installers
  (opencode, bun, NVM, ggshield) append PATH/init lines episodically — not on
  drift observed today, since all three measured EQUAL. Each exemption carries
  its reason inline; an exemption with no live mechanism is a bug (see R5).
- `docker compose` gets a verdict where it has none today, **probed as the v2
  CLI plugin with the standalone v1 binary as fallback**, and reported as
  optional. The repo provisions compose nowhere (no entry in `versions.conf`,
  `env-contract.json`, or any installer block), so a FAIL would be wrong; and a
  bare `docker-compose`-on-PATH test would report SKIP on a modern box where
  `docker compose` works (measured on msi: v1 binary at `/usr/local/bin` *and*
  plugin v2.39.1 both present).
- `setup-linux.sh` drops the three `check_deployed` calls and the
  `check_dependencies` call; `dotf doctor` at the tail of the same script is the
  single verification surface, matching what `setup-windows.ps1` already does.
- A guard asserts the six REFACTOR-002 pre-exports are still there, so the next
  reader of #1337 cannot delete them on the same false premise (BUG-021).

## Out of scope

- **`checkDeployDrift`** — the repo→deploy-dir leg. Untouched; it is not the
  duplication and never was.
- **The FAIL/SKIP policy of helm, terraform, ansible, pip** — both sides are
  advisory today, so deleting the setup call loses no severity. BUG-052 made
  terraform a SKIP deliberately; re-litigating it belongs to its own ticket.
- **`docker`/`kubectl` FAIL-vs-SKIP on Windows** (BUG-052) — deliberate, and a
  per-machine tool-role scoping in `machine.json` is a separate open idea.
- **The six pre-exported vars** — kept, and guarded. Not a deletion candidate.
- **`setup-windows.ps1` deletions** — nothing to delete there.
- **Windows deploy targets.** `homeDeployMap` is derived from `setup-linux.sh`'s
  `deploy_file` calls, and every entry in it is POSIX-only, so the new section
  **SKIPs entirely on Windows**. The second-leg hole named in Why#2 therefore
  **stays open on Windows** after this PR. Closing it needs a second map derived
  from `setup-windows.ps1`'s own deploy calls plus its own join guard — filed as
  **OPS-046 (#1447)**, blocked on this one, rather than doubling this PR's size
  against the 300-LOC cap. The Windows exemption set also cannot be measured from
  Linux, the same reason HIVE-118 stayed out of OPS-040.

## Risks / open questions

- **R1 — the path map is a new join, and nothing spans it. RESOLVED in design,
  MUST hold in implementation.** `deploy_file` in `setup-linux.sh` is the source
  of truth for what lands in `$HOME` (11 call sites, lines 83-1646). If the
  doctor map hardcodes its own copy, the two drift silently the first time a file
  is added to setup. The map must therefore ship **with a drift guard on the join
  itself** — a test that fails when a `deploy_file "$DOTFILES_DIR/..." "$HOME/..."`
  call has no corresponding map entry. Same idiom as
  `tests/guard-doctrine-target-not-deleted.bats`.
- **R2 — the exemption list is load-bearing, and deriving it from
  `check_deployed`'s comments would have got it wrong.** Those comments only ever
  knew about the 3 files that function checked; the map covers 11. Probing all
  11 on msi (2026-09-02) found `.gitconfig` drifting — a file `check_deployed`
  never watched and whose exemption no comment records. A content check that
  omits it turns doctor red on every box with a `git config --global`.
- **R5 — one of the inherited exemption reasons is already stale.** The NOTE at
  `setup-linux.sh:1586-1589` justifies exempting `.zshrc` partly by "setup may
  legitimately modify it (e.g. stripping stale gh-copilot eval lines via
  `sed -i`)" — but **no `sed -i` remains in the script**; the only match in the
  whole file is inside that comment. The OPS-040 purge removed the mechanism and
  left the justification. The exemption still stands on its other half (installer
  appends, which `.bashrc`'s NOTE names concretely), but the dead clause must be
  corrected in this PR rather than copied into the new map — this is lesson 256
  recurring inside the spec that cites it.
- **R3 — ordering within the PR.** The setup deletion is only safe once the
  doctor check is green. Both land in one PR, so this is a review property, not a
  runtime one; the reviewer should verify the port precedes the delete in the
  commit sequence, and that AC3's parity table is measured, not asserted.
- **R4 — `.gitconfig` deploy is conditional** (`setup-linux.sh:96`). It may be
  absent on a box that declined it, so its map entry must tolerate absence as
  SKIP rather than FAIL, matching `checkDeployDrift`'s both-sides-present rule.

## Acceptance criteria

- [ ] **AC1** — `dotf doctor` FAILs when a file deployed to `$HOME` differs
      byte-for-byte from its `~/.dotfiles` source, and PASSes when equal, for at
      minimum `.zsh/aliases.zsh`, `.zsh/functions.zsh`, `.zsh/functions.sh`.
      Today all three report PASS under drift.
- [ ] **AC2** — the exempt set (`.gitconfig`, `.zshrc`, `.bashrc`, `.profile`) is
      checked for existence only and never FAILs on content drift; every other
      map entry is content-checked. The `.zshrc` NOTE's stale `sed -i` clause is
      corrected in the same PR (R5).
- [ ] **AC3** — every tool and file covered by the deleted `check_deployed` ×3 and
      `check_dependencies` calls has an **equal-or-stronger** verdict in
      `dotf doctor`, demonstrated by a table-driven test enumerating all 14 tools
      and 3 files with their before/after severity. Three rows are decided here
      rather than left to the implementer: **`docker compose`** → optional,
      plugin-probed (the repo provisions it nowhere, so FAIL is wrong);
      **helm/terraform/ansible/pip** → doctor's SKIP is severity-equal to the
      shell's WARN, both advisory; **a symlink at `$HOME`** → FAIL, preserving
      `check_deployed`'s rejection that nothing else in doctor catches.
- [ ] **AC4** — a test fails if any of the six REFACTOR-002 pre-exports
      (`SCRIPTS_DIR`, `GEMINI_HOME`, `AGY_HOME`, `COPILOT_HOME`, `OPENCODE_HOME`,
      `DOTFILES_REPO_DIR`) is removed from `setup-linux.sh` (BUG-021 guard).
- [ ] **AC5** — a Go test fails if a `deploy_file … "$HOME/…"` call in
      `setup-linux.sh` has no entry in `homeDeployMap` (R1 join guard). It lives
      in Go, not bats, because the map is Go: one parser, and it breaks in
      `go test` where the map changes. Reading a repo file from a Go test is an
      established path here (`env_test.go:77`, `prtriage_test.go:134`,
      `repoRootForTest` in `cmd/`).
- [ ] **AC6** — `setup-linux.sh` no longer calls `check_deployed` or
      `check_dependencies`; `GOOS=windows go vet ./...` and the full Go + bats
      suites pass.

## References

- Bitácora board: `mlorentedev/dotfiles#1337`
- `docs/adr/adr-020-tooling-cli-go-convergence.md` — §5 strangler-fig on contact;
  ADR-021/CLI-012 retired the `doctor.sh`/`healthcheck.sh` twins into `dotf doctor`.
- BUG-021 (the pre-export fix), BUG-052 (terraform SKIP, Windows docker/kubectl).
- `docs/lessons.md` 256 — probe the target, never read the block's own comment.
  This spec is that lesson recurring: #1337 asserts duplication from **names**
  (`check_deployed` ≈ `checkDeployDrift`) without comparing direction or severity.
- Sibling precedent: `specs/archive/OPS-040-dead-migration-purge/` — same class of
  ticket, same class of false premise, same method (classify, then probe).
