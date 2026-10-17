---
id: "lesson-466-a-second-reader-of-a-generated-file-breaks-on-the-first-new-value-shape"
type: lesson
status: active
title: "A second reader of a generated file breaks on the first new value shape"
created: "2026-10-17"
---

# A second reader of a generated file breaks on the first new value shape


**Context:** `dotf tools sync` renders the mise pins into `conf.d/dotfiles.toml`. `tests/verify-setup.bats` reads that file again with awk, to check that dotf rendered the pin set versions.conf declares. DX-005 (#2654) added file tools, which render as an inline table: `"http:zsh-autosuggestions" = { version = "0.7.1", url = …, checksum = … }`. It held the versions.conf mark back until a release knew the table.

**Problem:** The follow-up that set the mark (#2667) failed `integration` on its first push. The awk took the third field as the version and read `{`. It then ran `mise where` on the bare name, which mise does not resolve without the `http:` backend. Unit tests, the local bats run (551 ok) and a live `dotf tools sync` on the Mac were all green, because `verify-setup.bats` skips outside the integration container. And no rendered file had carried the new shape before: the PR that taught dotf to write it deliberately did not mark it.

**Solution:** The reader now takes the version from the inline table's `version` key, and keeps the backend in the name it passes to `mise where`. Replay a container-only test's logic on the host when its precondition is "inside the container" but its subject (the rendered file, mise) exists locally too. Here that took one heredoc and found both defects before CI ran again.

**Why:** A second reader of a generated file encodes the shapes the writer produced when the reader was written. The writer's own tests cover the new shape, while the reader is only exercised by the job that runs on real output, and only once some real output has the shape. When a change splits "teach the writer" from "start emitting", the reader's first test is the second PR, so read the second readers before opening it: `git grep -l` the generated file's name under `tests/`.
