---
id: "lesson-429-an-installer-that-picks-the-newest-interpreter-fails-with-the-runner-image"
type: lesson
status: active
title: "An installer that picks the newest interpreter fails with the runner image"
created: "2026-10-14"
owner: manu
tags: [python, uv, ci, macos, pinning]
---

# An installer that picks the newest interpreter fails with the runner image

**Context:** From the first week of October, from-zero (macos-latest) failed now and then on the `tools` step with `catalog: ansible: uv exited 0 but ansible does not run on PATH (is uv's global bin dir on PATH?)`. Reruns passed. It was filed as BUG-138 (#2481), marked intermittent, and #2482 made the job print `ansible --version` and `ansible-lint --version` when the install fails.

**Problem:** The probe output showed the venv at `~/.local/share/uv/tools/ansible-core/lib/python3.15/`. ansible-core 2.21.5 supports Python up to 3.14 and crashes on import under 3.15.0. `versions.conf` pins `PYTHON_VERSION=3.13.16`, which mise installs, but `uv tool install` was never told about it, so it built each venv on the newest interpreter it could find. Runner images that carried 3.15 failed and the others passed, so the failure followed the image rollout and looked random. Both error messages pointed the wrong way. The binary was on PATH; it crashed. ansible-lint's "below the pin" line had read a version out of ansible-core's warning on stderr.

**Solution:** `uv tool install --python 3.13`. The series is read from `PYTHON_VERSION` (`tools.PythonSeries`) and set by the one constructor that `dotf tools install` and converge share. On the Mac, uv resolves `3.13` to mise's own interpreter. When the tool is on PATH but reports no version, the error now says that and names the copy, instead of asking whether the bin dir is on PATH.

**Why:** A pinned runtime protects only the commands that are told to use it. An installer left to choose will take the newest interpreter on the machine, and the newest changes whenever the image changes, outside the repo's history. When a failure is "intermittent" on a hosted runner, ask which input the repo does not pin before you retry it. And make the job print the failing command's own output: a summary message that guesses the cause cost a week here.
