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
- `bats tests/herdr-config.bats tests/pi-config.bats tests/versions-conf.bats` → 62 ok, 0 failures.
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
- The docked binding, which does not exist yet by design (see below).

## Decisions made during implementation

- **The docked binding waits for a release.** A `[[keys.command]]` naming
  `dotf herdr explorer` deploys fine in CI (built from the tree) and fails on
  every new machine, whose `DOTF_VERSION` is `0.75.0` and has no such subcommand
  (lesson 373). A test enforces the order until the follow-up lands.
- **Toggle detection by foreground process, not pane label.** `herdr pane rename`
  sets a label a later rename clears, and `pane list` only carries it when set;
  "the pane running yazi" is true for exactly as long as it is true.
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
