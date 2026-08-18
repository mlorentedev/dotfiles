---
id: lesson-093-deleting-one-os-twin-while-keeping-its-sibling-for
type: lesson
status: active
created: "2026-06-14"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 093: Deleting one OS twin while keeping its sibling forces asymmetric parity tests — rewrite them to the migration reality, don't fake symmetry

**Context**: CLI-012 ported the Linux diagnostics twins (`healthcheck.sh`, `doctor.sh`) to a cross-compiled `dotf doctor` and deleted them, but kept the `.ps1` siblings because `dotf` is not yet installed on Windows (no Windows `install-dotf`).

**Problem**: A pile of cross-OS bats encoded `.sh`↔`.ps1` symmetry ("parity: both healthchecks include BUG-015", "parity: both doctors check min_version"). Deleting only the `.sh` breaks the `.sh` half of every one, and the pure `.sh`-structural greps (`healthcheck.sh has 12 sections`, the BUG-023 probe shape) become dangling. Keeping them green by quietly dropping the `.sh` line leaves a test still *named* "parity: both…" that now checks only one OS — a lie in the suite.

**Solution**: Rewrite each parity test to the actual migration state — the Linux side asserts the `dotf doctor` wiring (or its intent moves to `go test`), the Windows side keeps its `.ps1` assertion, and the test name says which ("healthcheck.ps1 includes BUG-015…", "(Windows port pending)"). Pure `.sh`-structural tests are deleted outright; their behavioural intent lives in the Go table tests.

**Rule**: When a strangler-fig port deletes one OS's twin but can't yet delete the other, the cross-OS parity tests are no longer true — don't paper over them by dropping a grep line under an unchanged "both…" name. Rewrite them to the asymmetric reality so a reader sees the migration window, and migrate the deleted side's intent to the new test home. One-OS-per-PR is the cleaner unit, and the tests should announce which OS is done.
