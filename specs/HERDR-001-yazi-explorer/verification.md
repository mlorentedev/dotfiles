---
tags: [spec, verification]
created: "2026-10-18"
---

# Verification - HERDR-001-yazi-explorer

## Evidence

Every criterion, with the command that produced the output. All of it was run in
this session, on herdr 0.9.3, macOS arm64.

- **AC1 config accepted** -> `HERDR_CONFIG_PATH=ai/herdr/config.toml herdr config check` → `config: ok`; the deployed copy, same command → `config: ok`. `bats tests/herdr-config.bats` → 13/13.
- **AC2 splits right and runs yazi in the pane's directory** -> `go test ./internal/cmd -run TestHerdrExplorerOpensOnTheRightWhenNoneIsOpen` → PASS (asserts `pane split --pane w1:p1 --direction right --cwd /repo/wt --ratio 0.30`, then `pane run w1:p2 yazi`). Live: from pane `w10:p8`, `dotf herdr explorer` created `w10:p9`, `herdr pane process-info --pane w10:p9` → `['yazi']`, and the command printed `opened the explorer in w10:p9 at /Users/manu/Projects/worktrees/dotfiles/feat-herdr-optimization`.
- **AC3 toggle** -> `go test ./internal/cmd -run 'TestHerdrExplorerClosesTheOneAlreadyRunning|TestHerdrExplorerIgnoresOtherTabsExplorer'` → PASS. Live: a second call printed `closed the explorer in w10:p9`, and `herdr pane list` went from `w10:p1, w10:p8, w10:p9` back to `w10:p1, w10:p8`.
- **AC4 refuses without yazi** -> `go test ./internal/cmd -run 'TestHerdrExplorerRefusesWithoutYazi|TestHerdrExplorerRefusesOutsideHerdr'` → PASS (messages name `dotf tools sync` and `HERDR_PANE_ID`).
- **AC5 pin provisioned** -> from the checkout: `dotf tools sync` (tree build) → `wrote /Users/manu/.config/mise/conf.d/dotfiles.toml`, `installed: yazi`, `41 tool(s) at their pin`; `grep yazi ~/.config/mise/conf.d/dotfiles.toml` → `yazi = "26.9.1"`; second run → `41 tool(s) at their pin; nothing to do`; `yazi --version` → `Version: 26.9.1`. `dotf deploy herdr`, `dotf deploy zshrc`, `dotf deploy bashrc`, each twice → `deployed` then `in sync`.
- **AC6 wrappers** -> `bats tests/pi-config.bats` → 26/26. Live, before the change: `herdr pane process-info --pane w10:p1` → foreground `dotf`; after the same command form with the hint in a spare pane: `herdr agent get` → `agent: "pi"`, `agent_status: "idle"`, `agent_session{source: herdr:pi}`, `terminal_title: "π - feat-herdr-optimization"`, `screen_detection_skipped: true`.

## Test status

- `cd cli && go vet ./... && go test ./...` → clean (no failures in any package).
- `bats tests/herdr-config.bats tests/pi-config.bats tests/versions-conf.bats` → 63 ok, 0 failures.
- `bats tests/bashrc-guards.bats tests/zshrc-guards.bats tests/shell-profile.bats tests/shell-wrapper-dedup.bats tests/opencode.bats tests/secrets-only-scope.bats tests/stub-real-pairing.bats tests/versions-no-hardcode.bats` → 101 ok, 0 failures.
- Manual smoke test: the docked explorer opened and toggled closed in the owner's live session; all scratch panes were closed, workspace `w10` is back to `w10:p1`.
- No regressions: yes. One collateral fix was needed: the `herdr config check`
  canary appended a second `[ui]` table, which is a duplicate-table error once
  `[ui]` exists — it now appends `[terminal]` and still names the unknown key.

## Not verified

- `opencode` detection live. It is wrapped identically and herdr classifies it
  the same way ("also reports state"), but only `pi` was exercised.
- Windows. herdr there is preview, and the PowerShell profile is deliberately
  untouched: it runs the real binary in-process, so no hint is needed and the
  change would be unverified either way.
- The docked binding in a live session once the release carrying #2854 is installed on the owner's machine, and the deployed config merged by `dotf converge`.

## Decisions made during implementation

- **The docked binding waits for a release.** A `[[keys.command]]` naming
  `dotf herdr explorer` deploys fine in CI (built from the tree) and fails on
  every new machine, whose `DOTF_VERSION` is `0.75.0` and has no such subcommand
  (lesson 373). A test enforced the order until the binding landed with 0.76.0.
- **Toggle ownership by pane label, not by foreground process.** The first
  design looked for the pane running yazi; review showed that closes a yazi the
  owner opened themselves (#2815). The explorer is now the pane labelled `files`,
  and a press from inside it closes it (#2854, lesson 503).
- **`dotf herdr files` was designed and dropped.** The popup runs `yazi` directly
  (no dotf coupling, so it ships now); a second subcommand to add a friendlier
  missing-binary message was not worth a public CLI surface.
- **Third-party plugins rejected** (`speardragon/herdr-yazi`, `smarzban/herdr-file-viewer`,
  `alexarthurs/herdr-sidebar`): none is expressible as a pinned, reproducible
  artifact in this repo — `dotf deploy` is a file table with no "run this"
  field, and `herdr plugin install` resolves HEAD without `--ref`. The Go
  subcommand is OS- and agent-agnostic and stays under the repo's own CI.
- **`agent_panel_sort = "priority"`** applied as its own one-key change, with a
  static test, because it is unrelated to the explorer and reversible alone.

## Round 1 dispositions (review-round1.md, reviewed_sha fbfedfc3)

The review ran before #2815 merged. Code findings were applied there and are
named below by test; the later ones are dispositioned here.

1. *Toggle picks the first yazi pane* -> **applied** in `11614b9b`: ownership is
   the `files` label, the invoking pane is never a candidate, and another yazi is
   named in an error, not closed. `TestToggleExplorerRefusesToCloseAStrangersYazi`,
   `TestToggleExplorerNeverTouchesTheInvokingPane`.
2. *`pane run` failure leaves an empty pane; instant-exit yazi stacks explorers*
   -> **applied** in `11614b9b`: the split is closed on a `rename` or `run`
   failure, and a labelled pane whose yazi has exited is closed by the next press.
   `TestToggleExplorerTakesBackAPaneItCannotFill`,
   `TestToggleExplorerClosesALabelledPaneWithYaziGone`. `--ratio` is validated:
   `TestValidateExplorerRatio`.
3. *Provisioning assumes a checkout* -> **applied, and already covered**: the
   refusal names `dotf tools sync` and says a stale deployed `versions.conf`
   needs setup to refresh it, and `dotf converge` mirrors `versions.conf` into
   the deploy dir (`recordsMirror`, `cli/internal/converge/records.go`). Lesson
   494 records the trap. **Declined**: a doctor check for a pin missing from the
   file `sync` reads; the supported path (`converge`) refreshes that file on
   every run, so the stale window closes without one.
4. *`features.json` verified less than its criteria* -> **applied**: f2, f3 and f4
   named tests that had moved to `internal/herdr`, so `go test -run` printed
   "no tests to run" and passed; they now name the real tests. f1's command
   (`bats tests/herdr-config.bats`) covers both bindings. f5 already checks the rendered config (`tools sync
   --dry-run`).
5. *AC6 sourced outside the scope; stale comment* -> **applied**: `tasks.md` says
   AC6 landed in `216fa210`, and the config comment now describes both keys.

## Round 2 dispositions (review.md, reviewed_sha 26c22bef, FAIL)

All code findings landed in #2854 (`4c92cfc4`), ahead of this change, because the
binding runs the pinned `dotf` and so waits for a release that carries them.

1. *Blocker: the key pressed from inside the explorer opens a second one* ->
   **applied**: a labelled invoking pane is `ours`, an unlabelled one is neither
   `ours` nor a stranger. `TestToggleExplorerClosesItselfWhenPressedFromInside`,
   `TestToggleExplorerNeverTouchesAnUnlabelledInvokingPane`; lesson 503. The live
   re-check on a released binary is the owner's keyboard check on #2814.
2. *Major: AC2/AC3/AC4 evidence names tests that no longer exist* -> **applied**:
   the commands above run `./internal/herdr -run TestToggleExplorer…`.
3. *Major: `harness/skills/herdr/SKILL.md` says to toggle by foreground process* ->
   **applied**: it now says toggle by label, in the vault source and the rendered
   record.
4. *Minor: the deployed config lacks the binding* -> **open, by order**: the key
   is inert on a machine until the release is installed and `dotf converge` (or
   `dotf deploy herdr`) merges it. Listed under "Not verified".
5. *Minor: the config-wide guard was deleted* -> **declined**: no source maps a
   `dotf` subcommand to the release that added it, so a config-wide scan has
   nothing to compare against; the per-binding test bites (the review mutated
   `DOTF_VERSION` to 0.75.0 and it failed), and lesson 373 names the rule for the
   next binding.
6. *Minor: with two explorers the toggle closes `ours[0]`* -> **declined**: the
   Blocker was the only way a keypress created a second explorer, and it is fixed.
7. *Minor: `ToggleExplorer` over 40 lines* -> **applied**: `openExplorer` was
   extracted; the function is 27 lines.
8. *Minor: a pane the user renames `files` is taken as ours* -> **declined**:
   surface only, the label is the documented token.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: three —
      `docs/lessons/lesson-493-a-wrapper-hides-the-foreground-process-so-name-the-agent-for-the-inspector.md`,
      `docs/lessons/lesson-494-tools-sync-reads-the-deploy-dirs-versions-conf-outside-the-checkout.md`,
      `docs/lessons/lesson-495-a-heredoc-inside-a-command-substitution-still-runs-its-backticks.md`
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the design follows ADR-020
      (new tooling in Go) and lesson 373 (release ordering); no new decision was taken.
- [x] New pattern candidate for `00_meta/patterns/`? no: one project, one tool.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HERDR-001-yazi-explorer/` -> `specs/archive/HERDR-001-yazi-explorer/`
