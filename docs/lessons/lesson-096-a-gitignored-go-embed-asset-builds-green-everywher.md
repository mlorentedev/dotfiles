---
id: lesson-096-a-gitignored-go-embed-asset-builds-green-everywher
type: lesson
status: active
created: "2026-06-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 096: A gitignored `//go:embed` asset builds green everywhere and only fails at runtime in a fresh checkout

**Context**: `dotf init` (CLI-014) vendors its scaffold templates under `cli/internal/initrepo/templates/` via `//go:embed`. The CLAUDE.md pointer template was committed as `templates/CLAUDE.md`.

**Problem**: The repo's root `.gitignore` has a `CLAUDE.md` rule, which silently swallowed `templates/CLAUDE.md` — it was never tracked. `//go:embed` reads the **working tree**, not the git index, so it embedded fine on the dev machine and every local `go test`/`go build` passed. Embedding a whole directory never requires a specific file, so even a fresh CI checkout **built a green binary**. The gap surfaced only at *runtime* on the first CI integration run: `dotf init` panicked with `open templates/CLAUDE.md: file does not exist`, because the file the code expected was absent from the clean checkout. No build, unit, or lint step caught it — the failure window was build-green / run-red.

**Solution**: Commit the template under a name no `.gitignore` rule matches (`claude-md`) and map it to its output filename in `scaffold.go` (`{"claude-md", "CLAUDE.md"}`). Add `tests/cli-embed-templates.bats` (incident -> guard) asserting (1) every file under `templates/` is git-tracked via `git ls-files --error-unmatch`, and (2) the `claude-md` mapping exists while `templates/CLAUDE.md` does not — catching the whole bug-class locally before CI.

**Rule**: `//go:embed` (and any embed-from-working-tree mechanism) trusts the working tree, not the index — a gitignored embedded asset is invisible to the compiler's tracking and fails only at runtime in a clean checkout. Never let an embedded asset's filename collide with a `.gitignore` rule, and assert every embed asset is git-tracked. Generalizes: any "works on my machine, missing in CI" symptom is a tracking gap — probe it with `git ls-files`, not another rebuild.
