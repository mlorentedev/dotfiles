---
id: "IDEAS-005-curl-bootstrap"
type: spec
status: draft
created: "2026-05-25"
issue: "mlorentedev/dotfiles#1094"
tags: [spec, proposal, ideas-005, bootstrap, dotfiles-survey, tier-2]
template_version: "1.0"
---

# IDEAS-005: One-liner curl bootstrap

> **Naming**: file lives at `<repo>/specs/IDEAS-005-curl-bootstrap/proposal.md`.
> Origin: dotfiles-survey research, Tier-2 idea (#5) from fmontes (`curl | bash` install).

## Why

<!-- from research/dotfiles-survey.md §"Top 6 ideas a aplicar" #5: fmontes one-liner curl install, Tier-2 ROI -->

Factory-reset to coding currently requires three manual steps: install git, `git clone https://github.com/<user>/dotfiles ~/Projects/dotfiles`, `cd ~/Projects/dotfiles && ./setup-linux.sh`. On a fresh machine this is 5+ minutes of guided-by-memory commands. fmontes ships a true zero-state one-liner: `curl -fsSL https://fmontes.com/install.sh | bash`. The user's `setup-linux.sh` is already idempotent — the missing piece is a thin `install.sh` at repo root that handles "git might not exist yet, repo might not be cloned, repo might already be cloned and need pulling" before delegating to setup. Without this, the user remembers the steps but can't share the dotfiles with a new colleague / paste into a setup doc / use them as a one-shot on a throwaway VM.

## What

A new `install.sh` at the repo root, ~40 LOC. Self-contained logic:

```bash
#!/usr/bin/env bash
set -euo pipefail
DOTFILES_DIR="${DOTFILES_DIR:-$HOME/Projects/dotfiles}"
DOTFILES_REPO="${DOTFILES_REPO:-https://github.com/mlorentedev/dotfiles.git}"

# 1. Pre-flight: git must exist
command -v git >/dev/null 2>&1 || { echo "ERROR: git not installed"; exit 1; }

# 2. Clone or update
if [ -d "$DOTFILES_DIR/.git" ]; then
    echo "Updating existing $DOTFILES_DIR..."
    git -C "$DOTFILES_DIR" pull --ff-only
else
    echo "Cloning $DOTFILES_REPO → $DOTFILES_DIR..."
    git clone "$DOTFILES_REPO" "$DOTFILES_DIR"
fi

# 3. Delegate
cd "$DOTFILES_DIR"
exec ./setup-linux.sh "$@"
```

README updated with the one-liner:

```bash
curl -fsSL https://raw.githubusercontent.com/mlorentedev/dotfiles/main/install.sh | bash
```

Plus a "verify before piping" instruction (`curl ... | less` first).

## Out of scope

- **Windows PowerShell equivalent** (`iwr ... | iex`) — tracked as IDEAS-005b for the Windows VM session.
- **Custom domain hosting** — `raw.githubusercontent.com` is sufficient. A `mlorente.dev/install.sh` redirect is future cleanup, not this spec.
- **Checksum verification / signing** — would mitigate the `curl | bash` risk but adds complexity (where to host the checksum, how to update it). Documented limitation; future spec.
- **Replacing `setup-linux.sh` itself** — install.sh is a *delegator*, not a replacement. The actual setup logic stays in setup-linux.sh.
- **Bootstrapping into a non-default location** — supported via `DOTFILES_DIR=/custom/path bash <(curl ...)` but not the documented happy path.

## Risks / open questions

- **R1 (security)**: `curl | bash` is the de facto unsafe pattern. Mitigation: (a) keep `install.sh` minimal (~40 LOC, human-auditable in one screen), (b) README has a prominent "verify before piping" instruction with the `curl ... | less` command, (c) repo is public so the URL is inspectable. Accept the standard risk profile.
- **R2**: clone destination collision. If `~/Projects/dotfiles` exists but is NOT a git repo (e.g., user created it manually), `git -C ... pull` fails. Mitigation: explicit check for `$DOTFILES_DIR/.git` directory; if dir exists without `.git`, fail with clear "directory exists but is not a git repo — remove or set DOTFILES_DIR=..." message.
- **R3**: GitHub URL stability. The one-liner pins to `main` via `raw.githubusercontent.com`. If the repo is renamed / made private / branch renamed, the one-liner breaks silently. Acceptable risk — document the fallback (manual clone) in README.
- **R4 (BLOCKER)**: first-time prerequisites. `setup-linux.sh` assumes git, curl, sudo, and some baseline. `install.sh` must check at least `git` (since it needs to clone). The remaining prereqs are checked by `setup-linux.sh` itself.
- **R5 (open question, non-blocker)**: should `install.sh` ALSO be runnable inside the cloned repo (as a no-op-then-delegate)? Pro: single canonical entry point. Con: `setup-linux.sh` already works directly. **Recommendation: yes, `install.sh` is the canonical entry — README points users at it whether bootstrapping fresh or re-running. setup-linux.sh stays callable but is the "I know what I'm doing" path.**

## Acceptance criteria

- [ ] `install.sh` exists at repo root, executable (`chmod +x install.sh`).
- [ ] Idempotent: running on a machine where `$DOTFILES_DIR/.git` exists → `git pull --ff-only`, no clone, no errors.
- [ ] Honors `DOTFILES_DIR` env var: `DOTFILES_DIR=/tmp/df bash install.sh` clones to `/tmp/df`.
- [ ] Honors `DOTFILES_REPO` env var: useful for forks.
- [ ] Fails fast (exit 1) with clear message if `git` missing.
- [ ] Fails fast (exit 1) with clear message if `$DOTFILES_DIR` exists but is not a git repo.
- [ ] README includes the curl one-liner and a "verify before piping" instruction.
- [ ] Bats test: empty `$DOTFILES_DIR` → install.sh runs → dir is a git repo (mock setup-linux.sh via env var `DOTFILES_SKIP_SETUP=1` to avoid actually installing in test).
- [ ] Bats test: pre-existing `$DOTFILES_DIR/.git` → install.sh runs pull, no clone error.
- [ ] Shellcheck clean.

## Completeness review

Standard items considered:

- **Rate limit / cost guard** — N/A (GitHub raw has its own rate limits; not a concern for one-off bootstraps).
- **Idempotency** — covered by criterion 2.
- **Regression test** — bats covers happy path + key error cases.
- **Cert provisioning** — N/A (relies on system CA bundle for HTTPS).
- **Rollback** — `install.sh` is additive; revert removes the file. Existing `setup-linux.sh` invocation paths unchanged.

Adding (not in template, load-bearing here):

- **Security disclosure in README**: `curl | bash` warning prominent, with `curl ... | less` inspection command.
- **`DOTFILES_SKIP_SETUP=1` env var**: for bats integration tests to exercise clone-path without running full setup. Document this in `install.sh` comments.
- **Cross-OS gap note**: Linux/macOS only. Windows equivalent (PowerShell `iwr | iex`) is IDEAS-005b — explicitly mention in README.

## References

- Research source: `research/dotfiles-survey.md` § "Top 6 ideas a aplicar" #5 (Tier 2).
- Upstream: fmontes/dotfiles — `curl -fsSL https://fmontes.com/install.sh | bash`.
- Related: IDEAS-004 (collision prompt) — first-time `install.sh` invocation hits collisions on pre-existing `~/.zshrc` etc.; force-mode default must be safe.
- Project rules: `.claude/CLAUDE.md` — setup script changes touch the cross-platform layer (mirror in setup-windows.ps1 contract).

## LOC estimate

~40 LOC `install.sh` + ~50 LOC bats + ~30 LOC README diff = **~120 LOC total**. Above the 50-LOC threshold; full SDD discipline applies.
