---
tags: [spec, tasks]
created: "2026-06-07"
---

# Tasks - REFACTOR-011-version-manifest

> Implementation **deferred** (spec-only session 2026-06-06). Concrete plan below with
> exact file:line from the investigation, so the next session executes without re-scoping.
> Decision approved: **remove RC fallbacks + add a guard** (single-source manifest).

## Setup

- [x] Branch created: `refactor-011-version-manifest` (from `origin/main`)
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Open question resolved: the opencode installer honors `--version`/`VERSION` (verified against `https://opencode.ai/install`)

## Implementation (TDD order — one commit each)

1. **Add the pin to the manifest** — `versions.conf`: add `OPENCODE_VERSION=1.16.2` (currently-installed version; bump later as desired).
2. **Assert it** — `tests/versions-conf.bats`: add `@test "versions.conf sets OPENCODE_VERSION"` mirroring the per-var tests (lines 30-52); the semver-all-values test (55-62) already covers format.
3. **Make setup read the manifest** — `setup-linux.sh` does **not** currently source `versions.conf` (only `safe_copy` at :38). Add `[ -f "$CURRENT_DIR/versions.conf" ] && . "$CURRENT_DIR/versions.conf"` early (after `CURRENT_DIR`/`DOTFILES_DIR` are set, ~line 30) so `$OPENCODE_VERSION`/`$GHOSTTY_VERSION` are reliable regardless of the parent shell.
4. **Pin opencode (Linux)** — `setup-linux.sh:596`: `curl -fsSL https://opencode.ai/install | bash -s -- --version "$OPENCODE_VERSION"`. Consider re-install-on-drift in the idempotence block (:593-602).
5. **Drop the ghostty fallback** — `setup-linux.sh:688`: `${GHOSTTY_VERSION:-1.3.0}` → `${GHOSTTY_VERSION}` (now that the manifest is sourced).
6. **Remove RC fallbacks** — `.zshrc:48-52` and `.bashrc:71-75`: `${JAVA_VERSION:-21.0.4}` → `${JAVA_VERSION}` for JAVA/MAVEN/PYTHON/MINIKUBE/GO (10 lines total). The `. versions.conf` source line stays (`.zshrc:45`, `.bashrc:68`).
7. **Guard test (incident→guard)** — new `tests/versions-no-hardcode.bats`:
   - fail if `.zshrc`/`.bashrc` match `\$\{[A-Z_]+_VERSION:-[0-9]` (a re-introduced hardcoded fallback);
   - assert every `${*_VERSION}` referenced in the RC tool-home exports is a key defined in `versions.conf`.
8. **healthcheck opencode assert (Linux)** — `scripts/healthcheck.sh` opencode block (~353-357): add a version-match mirroring the ghostty pattern (379-388): `OPENCODE_PINNED="$(grep -E '^OPENCODE_VERSION=' "$DOTFILES_DIR/versions.conf" | cut -d= -f2)"`, compare to `opencode --version`.
9. **Windows (⚠️ Windows-empirical — defer to a Windows session per the batch-windows rule):**
   - `setup-windows.ps1` installs opencode via **winget** (`SST.opencode`, generic loop :310-322). Pinning needs `winget install --version $OPENCODE_VERSION SST.opencode` (special-case out of the loop) **and** the script must parse `OPENCODE_VERSION` from `versions.conf`. Verify winget honors `--version` for this package + that the version is available.
   - `scripts/healthcheck.ps1` (~472): mirror the opencode version assert.

## Closing

- [ ] Every AC covered by a test; add sibling `features.json` (harness contract) at implementation time.
- [ ] `bats tests/*.bats` green; `shellcheck` clean on changed scripts.
- [ ] `verification.md` filled with evidence (commit hashes / test names).
- [ ] Windows items either landed (if at a Windows box) or split into a follow-up ticket.
- [ ] PR opened referencing this spec folder.

## Notes / discovered debt (capture, don't fix here)

- `init-spec.sh` is still vault-`11-tasks.md`-rooted; ADR-018 moved task tracking to the bitácora. Used `--force-no-vault` to scaffold. Separate ticket candidate.
- `setup-linux.sh` not sourcing `versions.conf` means `${GHOSTTY_VERSION:-1.3.0}` (:688) only worked via the invoking shell's env — same masked-failure class this refactor removes.
