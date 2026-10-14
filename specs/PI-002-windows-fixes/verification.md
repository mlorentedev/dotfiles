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
   Recorded 2026-10-13 on msi (Linux, dotf 0.69.0), whose opencode catalog is
   cached (`~/.cache/opencode/models.json`, written 2026-09-24, carrying
   `openrouter/deepseek/deepseek-chat` limit `{'context': 163840, 'output': 16384}`):
   `dotf doctor --verbose` prints `[Model limit drift] [ OK ] 1 models match the
   provider catalog`. The macmini still SKIPs the section (no cached catalog).
   The Windows run of AC3 below remains owed.

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

## Windows runtime evidence (2026-10-13, EGW-LEN029)

Recorded on the corporate Windows 11 box (standard user, no elevation) with dotf
0.69.0 and checkout `132fa129`, after two `dotf converge` runs. The second run
reported `0 changed, 9 ok, 1 opaque, 3 skipped, 0 failed`.

**(a) No console window from `bw serve`.** `bw` resolves through the Scoop shim
(`C:\Users\mlorente\scoop\shims\bw.exe`). The owner ran `dotf secrets unlock` in
their own terminal and saw no console window. The process table agrees: both
the shim and the real `bw` child have no main window.

```text
pidfile=26400
proc=bw.exe parent=27984 cmd=bw serve --hostname 127.0.0.1 --port 8087
bw.exe pid=26400 parent=27984 hwnd=0      <- Scoop shim
bw.exe pid=28160 parent=26400 hwnd=0      <- bw serve
bw-serve.log: ==== dotf: started bw serve pid 26400 at 2026-10-14T02:08:31Z ====
```

`TestBWServeDetachAttr_ChildHasNoConsole` still cannot assert on this box from
an agent's tool call, because that test process has no console either:

```text
bwserve_windows_test.go:51: test process has no console (control probe: "console=none"); detachment cannot be observed here
--- SKIP: TestBWServeDetachAttr_ChildHasNoConsole (0.11s)
```

The owner's observation is therefore the evidence for (a).

**(b) Full `dotf doctor` on Windows (AC3).** The section this spec owns is clean:

```text
[Model limit drift]
  (1 checks, all ok)
Results: 162 passed, 1 failed, 6 warned, 28 skipped
```

The one FAIL is `hive service status gave no verdict`, which is #2258 (TEST-015,
the Windows hive install never starts the daemon) and is unrelated to this spec.
The six WARNs are local environment (mise not on PATH, no DR drill, two opencode
copies, Docker stopped, persona `enforce:` coverage, deployed dotf two `cli/`
commits behind HEAD).

Reaching the clean section took two fixes outside this spec, recorded on their
own tickets:

- The box's user-scope `HOME` had been set to `C:\SPB_Data` by a Cadence install,
  which split dotf across two homes (#2432). The variable was removed before this
  run.
- The opencode catalog cache was last written on 2026-06-09 and still carried
  the June limits (128000 / 16000). The first run therefore FAILed both deepseek
  limits, although models.dev and OpenRouter both reported 163840 / 16384 live.
  `opencode models openrouter --refresh` cleared it. Doctor not checking the
  cache's age is #2434.
