---
id: "POLISH-005-linux-idempotence-ci"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# POLISH-005-linux-idempotence-ci

> **Naming**: file lives at `<repo>/specs/POLISH-005-linux-idempotence-ci/proposal.md`. `POLISH-005-linux-idempotence-ci` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: (formalises Phase 2.6): formal idempotence CI job (Linux). Windows empirically validated in #105 (2 runs identical); Linux would need a `run-twice-and-diff` GHA job comparing `~/.dotfiles/` + `~/.claude/` + `~/.gemini/` + `~/.config/opencode/` between runs. -->

`setup-linux.sh` claims idempotency (re-running on a populated system should be a no-op). Today this is verified only by manual testing. Windows side was empirically validated in PR #105 (two consecutive runs produced byte-identical deployed state). Linux equivalent has no automated guard: an inadvertent commit that breaks idempotency (e.g., a script that appends instead of replaces) would silently ship until a user noticed. Formalising the check as CI converts a documented promise into an enforced invariant.

## What

New CI job `idempotence-linux` on `ubuntu-latest`:

1. Run `setup-linux.sh` once.
2. Snapshot `~/.dotfiles/`, `~/.claude/`, `~/.gemini/`, `~/.config/opencode/`, `~/.zshrc`, `~/.bashrc`, `~/.gitconfig`.
3. Run `setup-linux.sh` again.
4. Snapshot the same paths.
5. Diff snapshot-1 vs snapshot-2 — must be empty (mod timestamps/permissions noise).

Job becomes required-to-merge alongside the existing test job.

## Out of scope

- **Windows mirror** — WIN-004 covers Windows execution. Windows idempotence verification piggybacks once WIN-004 lands or is its own follow-up.
- **macOS idempotence** — pending DOCS-001 decision.
- **Per-script idempotence assertions in bats** — separate finer-grained tests; this is the integration-level guard.
- **Performance regression checks** (faster on re-run, etc.) — not the goal of this ticket.

## Risks / open questions

- **R1**: Noise filtering. Timestamps and inode numbers differ across runs. Use `diff -r --exclude='*.log' --exclude='.git'` plus a normalisation step (e.g., zero out mtimes).
- **R2**: Snapshot tool. Use `tar --sort=name --mtime='UTC 2020-01-01'` to get reproducible archives for hash comparison; or just `find ... -exec sha256sum {} \;` sorted.
- **R3**: CI runtime cost. Two consecutive setup runs ≈ doubles the test job duration. Run in parallel with `test` job (no dependency).
- **R4**: External resource flakes (apt download retry, curl from external sources). Either pin and mock, OR accept some non-determinism by hashing only the deployed *config* paths, not third-party binaries.

## Acceptance criteria

- [ ] `.github/workflows/ci.yml` includes `idempotence-linux` job on `ubuntu-latest`.
- [ ] Job runs `setup-linux.sh` twice with snapshot+diff between.
- [ ] Diff is empty modulo defined noise filters; non-empty diff exits non-zero.
- [ ] Job is added to required checks for `main`.
- [ ] CI wall-time increase ≤ 5 minutes (measure before merge).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → POLISH-005 (formalises Phase 2.6).
- Windows precedent: PR #105 (manual idempotence validation).
- Companion: WIN-004 (Windows CI runner) would extend this to Windows.
