---
id: "lesson-463-mise-counts-an-install-directory-as-the-install-even-without-its-file"
type: lesson
status: active
title: "Mise counts an install directory as the install, even without its file"
created: "2026-10-16"
owner: manu
tags: [mise, http-backend, idempotence, resilience]
---

# Mise counts an install directory as the install, even without its file

**Context:** zsh-autosuggestions ships no binary and has no mise registry entry, so DX-007 (#2654) installs it through mise's `http:` backend and has the sync verify the plugin file itself, since `mise which` and `<tool> --version` only know executables.

**Problem:** Measured with mise 2026.10.3 in a scratch `MISE_DATA_DIR`: with the install directory present and the file deleted, `mise install` prints "all tools are installed" and changes nothing, so the sync's post-condition failed on every later run until someone removed the directory by hand. Two neighbouring facts were equally invisible until measured: `mise ls --json` and `--prunable` key an http tool as `http:<name>`, so matching it by the pin's short name makes the orphan prune remove a plugin it just installed (never `changed=0`), and `mise where <name>` without the `http:` prefix answers "not found in mise tool registry".

**Solution:** A file tool is checked by the file inside `mise where http:<name>`, and a pinned tool whose file is missing is reinstalled with `mise install --force http:<name>@<version>`, which unpacks the digest-checked archive again. The orphan listing maps `http:<name>` back to the pin through one function (`pinName`), and a Go test fails if the pinned plugin is reported as an orphan.

**Why:** An idempotent step has to treat "the directory exists" and "the thing works" as separate facts, and repair the second one itself; a post-condition that fails forever with no repair path is a manual operation in disguise. Name translation between a pin and the tool manager's own key is where a prune quietly fights an install.
