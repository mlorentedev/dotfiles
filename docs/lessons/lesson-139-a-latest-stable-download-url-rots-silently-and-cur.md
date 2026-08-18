---
id: lesson-139-a-latest-stable-download-url-rots-silently-and-cur
type: lesson
status: active
created: "2026-06-27"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 139: A "latest/stable" download URL rots silently, and `curl` without `-f` turns a 404 into a corrupt artifact

**Context**: `setup-linux.sh` installed shellcheck from `…/releases/latest/download/shellcheck-stable.linux.x86_64.tar.xz`. A from-scratch container shakeout (running setup in a clean Ubuntu image) found shellcheck **never installs on a fresh machine** — a bug invisible on any box that already had it.

**Problem**: Two chained silent failures. (1) ShellCheck **retired the `-stable` asset alias** after v0.10, so `latest/download/shellcheck-stable…` 302s to the new tag then **404s** — an unversioned URL that rotted the moment upstream renamed its assets. (2) `curl -Lo … 2>/dev/null` with **no `-f`** wrote the 404 body (`Not Found`, 9 bytes) *as* `shellcheck.tar.xz`; the failure only surfaced later at `tar/xz` as a misleading "File format not recognized". A developer's machine had shellcheck only because it was installed back when `-stable` existed; fresh setups silently lost it. CI never caught it either: the integration container lacked `xz-utils`, so the path died even earlier with yet another error, masking the real one.

**Solution**: Pin `SHELLCHECK_VERSION` in `versions.conf` (the version SSOT) and build both the URL and the tarball's internal dir (`shellcheck-v<ver>/`) from it; switch to `curl -fsSL` so an HTTP error fails the step instead of poisoning the extract. Make the test representative: add `xz-utils` to `Dockerfile.integration` (without it `tar xJf` could never run, so CI never exercised the install) and assert `~/.local/bin/shellcheck` exists in `verify-setup.bats`.

**Rule**: A `latest`/`stable` download URL is an **unversioned dependency that rots without warning** — pin the version (in the version SSOT) and derive the URL from it. Any `curl` fetching a file MUST use `-f`: without it, a 404 or redirect-to-error-page is written as the file and the corruption detonates far from its cause. And a test environment that omits a tool the real install path needs (here `xz`) doesn't test that path — it hides it behind a different error; make the test environment representative or the path is effectively untested.
