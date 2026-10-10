# Verification

- Verified `go build -o ../dotf.exe ./cmd/dotf` inside `cli/` completes successfully.
- Code inspection aligns with `syscall.SysProcAttr` definitions on Windows.

## Independent review follow-up (2026-10-03)

The first review (`review.md`, FAIL, agy/gemini-3.1-pro-high, committed in
`008fd531`) found three REAL gaps. Status of each:

1. **Doctor still warned on the deepseek limits (Blocker).** Fixed: the values are
   now the provider catalog's 163840 / 16384. On Linux (msi), against this
   checkout:

   ```text
   $ DOTFILES_REPO_DIR=<worktree> go run ./cmd/dotf doctor --verbose
   [Model limit drift]
     [ OK ] 1 models match the provider catalog
   ```

   The section is OS-independent (it reads `ai/pi/models.json` and opencode's
   models.dev cache). The same run's two other FAILs are local-environment drift
   unrelated to this spec (installed `dotf` 0.62.0 against the 0.63.0 pin, and an
   `orca.json` hook timeout).

   **The upstream catalog flaps, so this value has moved since.** #2011
   (`946afa50`, 2026-10-05) set 128000 / 16000 because models.dev reported that
   then. On 2026-10-10 both sources report 163840 / 16384 again:

   ```text
   models.dev  openrouter/deepseek/deepseek-chat  limit {'context': 163840, 'output': 16384}
   openrouter  /api/v1/models deepseek/deepseek-chat  163840 16384
   ```

   so `ai/pi/models.json` is back to those values. A pinned literal against a
   catalog that changes every few days drifts by design; sourcing pi's limits
   from the catalog is the model-config SSOT work of #902, not this spec. The
   doctor check stays the guard. Owed with the Windows items below: a
   `dotf doctor` run on a machine that has opencode's catalog cached (this Mac
   SKIPs the section) showing `[ OK ]` for these values.

2. **The proposal named `CREATE_NO_WINDOW`, the code does not use it (Major).**
   Fixed: `proposal.md` What #1 now describes `HideWindow` with
   `DETACHED_PROCESS`, and why (lesson-333).

3. **No Windows runtime evidence (Major).** Partly available, partly owed:
   - The Windows CI leg ran the secrets package after the change
     (`cli.yml` run 37043829519, `test (windows-latest)`, head `8f2611ad`):
     `ok github.com/mlorentedev/dotfiles/cli/internal/secrets 0.609s`. The run is
     not verbose, and `TestBWServeDetachAttr_ChildHasNoConsole` skips itself when
     the test process has no console, so this line does not prove the test
     asserted anything on the runner.
   - **Owed, Windows box only:** (a) `dotf secrets unlock` through the Scoop shim
     shows no console window; (b) a full `dotf doctor` run on Windows is clean
     (AC3). Both are deferred to the next Windows session (tracked on #1968). The
     spec stays active until they are recorded here and a fresh review passes.
