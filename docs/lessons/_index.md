---
id: dotfiles-lessons-index
type: index
status: active
created: "2026-08-18"
owner: manu
tags: [lessons, index, dotfiles]
---

# Dotfiles — Lessons Learned Index

> Granular project-bound lessons, gotchas, and post-mortems for dotfiles.
> Individual lessons live alongside this index as `lesson-NNN-*.md` files.
>
> The entry count used to be stated here. It was a third copy of a fact the
> directory listing and this table already hold, it drifted from both (it read
> 212 against 234 files and 230 rows), and a stale count reads as authoritative.
> Count the files.

| Lesson | Date | Scope |
|---|---|---|
| [001 - Go's exec.LookPath is blind to extensionless scripts on Windows](lesson-001-go-s-exec-lookpath-is-blind-to-extensionless-scrip.md) | 2026-08-10 |  |
| [002 - echo -e breaks in zsh](lesson-002-echo-e-breaks-in-zsh.md) | 2025-12-15 |  |
| [003 - &>/dev/null is not POSIX](lesson-003-dev-null-is-not-posix.md) | 2025-12-15 |  |
| [004 - ((count++)) exits with code 1 when count is 0](lesson-004-count-exits-with-code-1-when-count-is-0.md) | 2025-12-15 |  |
| [005 - ${BASH_SOURCE[0]} is empty in zsh](lesson-005-bash-source-0-is-empty-in-zsh.md) | 2025-12-15 |  |
| [006 - Claude Code auto-memory path encoding](lesson-006-claude-code-auto-memory-path-encoding.md) | 2026-02-26 |  |
| [007 - set -u requires ${1:-} for optional positional parameters](lesson-007-set-u-requires-1-for-optional-positional-parameter.md) | 2026-02-26 |  |
| [008 - ${VAR:-fallback} pattern for sourced config files](lesson-008-var-fallback-pattern-for-sourced-config-files.md) | 2026-02-26 |  |
| [009 - Always edit the repo copy, never the deployed system copy](lesson-009-always-edit-the-repo-copy-never-the-deployed-syste.md) | 2026-02-27 |  |
| [010 - grep -c with 0 matches outputs "0" AND exits with code 1](lesson-010-grep-c-with-0-matches-outputs-0-and-exits-with-cod.md) | 2026-02-28 |  |
| [011 - Claude Code path encoding: Linux vs Windows differ](lesson-011-claude-code-path-encoding-linux-vs-windows-differ.md) | 2026-02-28 |  |
| [012 - bash set -e does not exit on [ with integer error when in && chain](lesson-012-bash-set-e-does-not-exit-on-with-integer-error-whe.md) | 2026-02-28 |  |
| [013 - Claude Code SessionStart hook for vault health context](lesson-013-claude-code-sessionstart-hook-for-vault-health-con.md) | 2026-02-27 |  |
| [014 - Aider requires Python 3.12 — audioop removed in 3.13](lesson-014-aider-requires-python-3-12-audioop-removed-in-3-13.md) | 2026-03-10 |  |
| [015 - Single-quoted shell strings prevent variable expansion in JSON](lesson-015-single-quoted-shell-strings-prevent-variable-expan.md) | 2026-03-12 |  |
| [016 - grep -c '.' counts 1 on empty input (newline matches dot)](lesson-016-grep-c-counts-1-on-empty-input-newline-matches-dot.md) | 2026-03-12 |  |
| [017 - Plaintext secrets must never touch disk — pipe to age directly](lesson-017-plaintext-secrets-must-never-touch-disk-pipe-to-ag.md) | 2026-03-12 |  |
| [018 - Uninitialized variable under set -u in conditional-only assignment](lesson-018-uninitialized-variable-under-set-u-in-conditional-.md) | 2026-03-12 |  |
| [019 - \s is not POSIX — use `[[:space:]]` in bash regex](lesson-019-s-is-not-posix-use-space-in-bash-regex.md) | 2026-03-12 |  |
| [020 - Stray bare word causes silent set -e abort](lesson-020-stray-bare-word-causes-silent-set-e-abort.md) | 2026-03-12 |  |
| [021 - ShellCheck treats "# shellcheck" comments as directives](lesson-021-shellcheck-treats-shellcheck-comments-as-directive.md) | 2026-03-16 |  |
| [022 - Secrets mapping and file inventory must be reconciled automatically](lesson-022-secrets-mapping-and-file-inventory-must-be-reconci.md) | 2026-03-25 |  |
| [023 - cp fails when source and destination resolve to the same file via symlink](lesson-023-cp-fails-when-source-and-destination-resolve-to-th.md) | 2026-03-18 |  |
| [024 - PSScriptAnalyzer fails on non-ASCII chars outside here-strings](lesson-024-psscriptanalyzer-fails-on-non-ascii-chars-outside-.md) | 2026-03-26 |  |
| [025 - File deployment requires delete-then-copy, not additive-only copy](lesson-025-file-deployment-requires-delete-then-copy-not-addi.md) | 2026-03-29 |  |
| [026 - Config deployment guards vs tool installation guards](lesson-026-config-deployment-guards-vs-tool-installation-guar.md) | 2026-03-25 |  |
| [027 - Self-heal third-party plugin breakage at SessionStart](lesson-027-self-heal-third-party-plugin-breakage-at-sessionst.md) | 2026-05-08 |  |
| [028 - tmux clipboard needs an external bridge — and that bridge is display-server-specific](lesson-028-tmux-clipboard-needs-an-external-bridge-and-that-b.md) | 2026-05-11 |  |
| [029 - Editing a dotfile in the repo does not take effect until `setup-linux.sh` runs](lesson-029-editing-a-dotfile-in-the-repo-does-not-take-effect.md) | 2026-05-11 |  |
| [030 - Env-vs-disk drift after secret mutation](lesson-030-env-vs-disk-drift-after-secret-mutation.md) | 2026-05-12 |  |
| [031 - git log --pretty=format: drops last commit silently in while-read pipelines](lesson-031-git-log-pretty-format-drops-last-commit-silently-i.md) | 2026-05-12 |  |
| [032 - MCP transport state and daemon state can disagree per-conversation](lesson-032-mcp-transport-state-and-daemon-state-can-disagree-.md) | 2026-05-15 |  |
| [033 - bash `IFS=$'\t' read` collapses consecutive tabs (whitespace IFS chars never preserve empty fields)](lesson-033-bash-ifs-t-read-collapses-consecutive-tabs-whitesp.md) | 2026-05-15 |  |
| [034 - PSScriptAnalyzer fails on non-ASCII in .ps1 without BOM](lesson-034-psscriptanalyzer-fails-on-non-ascii-in-ps1-without.md) | 2026-05-16 |  |
| [035 - Verify post-checks with hardcoded strings rot when the verified file is refactored](lesson-035-verify-post-checks-with-hardcoded-strings-rot-when.md) | 2026-05-18 |  |
| [036 - detect-and-act scripts go silently inert when upstream products change their surface](lesson-036-detect-and-act-scripts-go-silently-inert-when-upst.md) | 2026-05-18 |  |
| [037 - When an invariant changes, dead code emerges silently downstream](lesson-037-when-an-invariant-changes-dead-code-emerges-silent.md) | 2026-05-18 |  |
| [038 - Bulk-copy operations collide silently with per-file deploy logic](lesson-038-bulk-copy-operations-collide-silently-with-per-fil.md) | 2026-05-18 |  |
| [039 - Defensive monitors are not fixes — trigger fix and monitor are siblings, not substitutes](lesson-039-defensive-monitors-are-not-fixes-trigger-fix-and-m.md) | 2026-05-19 |  |
| [040 - Wide try/catch misclassifies the error and misleads the next reader](lesson-040-wide-try-catch-misclassifies-the-error-and-mislead.md) | 2026-05-19 |  |
| [041 - Incident → guard pattern (red-team thyself)](lesson-041-incident-guard-pattern-red-team-thyself.md) | 2026-05-19 |  |
| [042 - Filename glob *.lock does NOT match package-lock.json (basename matters)](lesson-042-filename-glob-lock-does-not-match-package-lock-jso.md) | 2026-05-19 |  |
| [043 - Numeric bats threshold drift is invisible — comment the bump inline](lesson-043-numeric-bats-threshold-drift-is-invisible-comment-.md) | 2026-05-19 |  |
| [044 - "chore: close spec lifecycle" pattern — for features that shipped piecemeal before archive](lesson-044-chore-close-spec-lifecycle-pattern-for-features-th.md) | 2026-05-19 |  |
| [045 - JSONC native // comments beat _commentKey JSON convention for documentation](lesson-045-jsonc-native-comments-beat-commentkey-json-convent.md) | 2026-05-19 |  |
| [046 - Audit ALL call sites of a vulnerable upstream API when guarding one](lesson-046-audit-all-call-sites-of-a-vulnerable-upstream-api-.md) | 2026-05-20 |  |
| [047 - Claude Code marketplace dir naming follows GitHub repo, NOT declared name field](lesson-047-claude-code-marketplace-dir-naming-follows-github-.md) | 2026-05-20 |  |
| [048 - vault_patch timeout != patch not applied](lesson-048-vault-patch-timeout-patch-not-applied.md) | 2026-05-20 |  |
| [049 - Fix ALL surfaces in the same PR when a bug class spans multiple call sites](lesson-049-fix-all-surfaces-in-the-same-pr-when-a-bug-class-s.md) | 2026-05-21 |  |
| [050 - The detection probe MUST use the race-free pattern, not the upstream broken one](lesson-050-the-detection-probe-must-use-the-race-free-pattern.md) | 2026-05-21 |  |
| [051 - Healthcheck must validate end-state, not proxy artifacts](lesson-051-healthcheck-must-validate-end-state-not-proxy-arti.md) | 2026-05-21 |  |
| [052 - PowerShell single-quoted strings + grep BRE: backslash counting](lesson-052-powershell-single-quoted-strings-grep-bre-backslas.md) | 2026-05-21 |  |
| [053 - Heal scripts versioned against the upstream bug class they paper over](lesson-053-heal-scripts-versioned-against-the-upstream-bug-cl.md) | 2026-05-21 |  |
| [054 - Safety-net fixes must be audited against the same bug-class they paper over](lesson-054-safety-net-fixes-must-be-audited-against-the-same-.md) | 2026-05-21 |  |
| [055 - Classify "extract boilerplate" audit findings as bootstrap (chicken-and-egg) vs logic before estimating LOC savings](lesson-055-classify-extract-boilerplate-audit-findings-as-boo.md) | 2026-05-21 |  |
| [056 - Setup-time mutations to repo-symlinked files create permanent drift false-positives](lesson-056-setup-time-mutations-to-repo-symlinked-files-creat.md) | 2026-05-21 |  |
| [057 - Byte-equivalence assertions require SCRIPT_DIR control, not just literal diff](lesson-057-byte-equivalence-assertions-require-script-dir-con.md) | 2026-05-21 |  |
| [058 - Batch-scaffold N specs in one PR from a research worktree, defer implementation](lesson-058-batch-scaffold-n-specs-in-one-pr-from-a-research-w.md) | 2026-05-25 |  |
| [059 - Incomplete migration: file rename leaves callers stale](lesson-059-incomplete-migration-file-rename-leaves-callers-st.md) | 2026-05-26 |  |
| [060 - Stop fighting agent filesystem expectations](lesson-060-stop-fighting-agent-filesystem-expectations.md) | 2026-05-26 |  |
| [061 - pkill -f self-kill: pattern matches the pkill command line itself](lesson-061-pkill-f-self-kill-pattern-matches-the-pkill-comman.md) | 2026-05-26 |  |
| [062 - MEMORY.md files break YAML parsing with `# currentDate` and `---` separators](lesson-062-memory-md-files-break-yaml-parsing-with-currentdat.md) | 2026-05-27 |  |
| [063 - PowerShell -replace with [\s\S]*? expands large strings instead of replacing](lesson-063-powershell-replace-with-s-s-expands-large-strings-.md) | 2026-05-26 |  |
| [064 - PowerShell -replace with [\s\S]*? expands large strings instead of replacing](lesson-064-powershell-replace-with-s-s-expands-large-strings-.md) | 2026-05-27 |  |
| [065 - PowerShell -replace with [\s\S]*? expands large strings instead of replacing](lesson-065-powershell-replace-with-s-s-expands-large-strings-.md) | 2026-05-27 |  |
| [066 - Split-Path -LiteralPath and -Parent are mutually exclusive parameter sets in PowerShell](lesson-066-split-path-literalpath-and-parent-are-mutually-exc.md) | 2026-05-27 |  |
| [067 - Verify a plan's file-map against the code before editing load-bearing scripts](lesson-067-verify-a-plan-s-file-map-against-the-code-before-e.md) | 2026-05-30 |  |
| [068 - A whole-file transform must inspect the data shape before assuming a uniform model](lesson-068-a-whole-file-transform-must-inspect-the-data-shape.md) | 2026-05-30 |  |
| [069 - Don't commit to a shared vault that has another session's staged work — use an isolated worktree](lesson-069-don-t-commit-to-a-shared-vault-that-has-another-se.md) | 2026-05-30 |  |
| [070 - A structural integrity guard surfaces latent issues you didn't know you had](lesson-070-a-structural-integrity-guard-surfaces-latent-issue.md) | 2026-05-31 |  |
| [071 - A SCRIPT_DIR root-resolution fix breaks CWD-pinned fixture tests — add an env-override seam, not a code branch](lesson-071-a-script-dir-root-resolution-fix-breaks-cwd-pinned.md) | 2026-05-31 |  |
| [072 - A skip-guarded test is green in CI but a real assertion locally — it can hide a genuine cross-OS parity gap](lesson-072-a-skip-guarded-test-is-green-in-ci-but-a-real-asse.md) | 2026-05-31 |  |
| [073 - Onboarding a junior/remote agent: verify its self-authored docs, enforce boundaries mechanically](lesson-073-onboarding-a-junior-remote-agent-verify-its-self-a.md) | 2026-05-31 |  |
| [074 - A cross-environment SSOT validator must split "content drift" (fail) from "runtime absent off-box" (warn)](lesson-074-a-cross-environment-ssot-validator-must-split-cont.md) | 2026-05-31 |  |
| [075 - A harness consumer spec scoped on a retired axis must be reconciled, not implemented (WORKMODE-001)](lesson-075-a-harness-consumer-spec-scoped-on-a-retired-axis-m.md) | 2026-06-02 |  |
| [076 - Archived-spec ≠ issue-complete; verify "high-value open" items against git before implementing](lesson-076-archived-spec-issue-complete-verify-high-value-ope.md) | 2026-06-01 |  |
| [077 - A held spec can be obsoleted by a later ADR — reconcile+close, don't implement-as-written](lesson-077-a-held-spec-can-be-obsoleted-by-a-later-adr-reconc.md) | 2026-06-01 |  |
| [078 - Broad sed over a backlog ticks substring mentions, not just the entry — anchor to the line-start id](lesson-078-broad-sed-over-a-backlog-ticks-substring-mentions-.md) | 2026-06-02 |  |
| [079 - A late push to a PR branch can miss the squash — verify the commit is on the PR head, and each deliverable is on main post-merge](lesson-079-a-late-push-to-a-pr-branch-can-miss-the-squash-ver.md) | 2026-06-03 |  |
| [080 - Re-running a failed Actions run replays the *original commit's* workflow file](lesson-080-re-running-a-failed-actions-run-replays-the-origin.md) | 2026-06-07 |  |
| [081 - `gh project --owner` → "unknown owner type" under a fine-grained PAT; a green CI is not a green workflow](lesson-081-gh-project-owner-unknown-owner-type-under-a-fine-g.md) | 2026-06-07 |  |
| [082 - setup-time `compile-harness.sh --refresh` leaves silent drift -- surface it, don't delete it](lesson-082-setup-time-compile-harness-sh-refresh-leaves-silen.md) | 2026-06-09 |  |
| [083 - Frontmatter must be strict-YAML clean — the most lenient parser in the fleet is not the contract](lesson-083-frontmatter-must-be-strict-yaml-clean-the-most-len.md) | 2026-06-10 |  |
| [084 - Deploying workflow files to N repos via the contents API: three gotchas the happy path hides](lesson-084-deploying-workflow-files-to-n-repos-via-the-conten.md) | 2026-06-10 |  |
| [085 - Tests aimed at a runner that doesn't exist yet are dead weight — and "if available" guards rot silently](lesson-085-tests-aimed-at-a-runner-that-doesn-t-exist-yet-are.md) | 2026-06-10 |  |
| [086 - goreleaser monorepo.tag_prefix is Pro-only — verify paywalled features empirically](lesson-086-goreleaser-monorepo-tag-prefix-is-pro-only-verify-.md) | 2026-06-12 |  |
| [087 - Non-streaming chat endpoints behind a gateway drop long generations — a client timeout cannot fix a server-side cut](lesson-087-non-streaming-chat-endpoints-behind-a-gateway-drop.md) | 2026-06-12 |  |
| [088 - `gh project item-list` truncates to `--limit` silently — check `totalCount` before asserting absence](lesson-088-gh-project-item-list-truncates-to-limit-silently-c.md) | 2026-06-13 |  |
| [089 - A bats teardown's last command classifies even *skipped* tests — never end it with a bare `[ cond ] && cmd`](lesson-089-a-bats-teardown-s-last-command-classifies-even-ski.md) | 2026-06-13 |  |
| [090 - Sourced-vs-executed guard: use `(return 0 2>/dev/null)`, not a `BASH_SOURCE`-vs-`$0` compare](lesson-090-sourced-vs-executed-guard-use-return-0-2-dev-null-.md) | 2026-06-13 |  |
| [091 - A migration's acceptance guard-grep is the completeness oracle, not the spec's hand-listed targets](lesson-091-a-migration-s-acceptance-guard-grep-is-the-complet.md) | 2026-06-13 |  |
| [092 - Editing a committed render without its source-of-truth is a half-migration that `--refresh` reverts](lesson-092-editing-a-committed-render-without-its-source-of-t.md) | 2026-06-13 |  |
| [093 - Deleting one OS twin while keeping its sibling forces asymmetric parity tests — rewrite them to the migration reality, don't fake symmetry](lesson-093-deleting-one-os-twin-while-keeping-its-sibling-for.md) | 2026-06-14 |  |
| [094 - A consolidated diagnostic that shells out to a generator is on-demand-cheap but per-event-expensive](lesson-094-a-consolidated-diagnostic-that-shells-out-to-a-gen.md) | 2026-06-14 |  |
| [095 - A non-runnable cobra parent with no subcommands is demoted to "Additional help topics"](lesson-095-a-non-runnable-cobra-parent-with-no-subcommands-is.md) | 2026-06-14 |  |
| [096 - A gitignored `//go:embed` asset builds green everywhere and only fails at runtime in a fresh checkout](lesson-096-a-gitignored-go-embed-asset-builds-green-everywher.md) | 2026-06-15 |  |
| [097 - Minting a new script to delete four others fights a reduce-the-surface goal — remove and ticket-restore, don't extract](lesson-097-minting-a-new-script-to-delete-four-others-fights-.md) | 2026-06-15 |  |
| [098 - A migration that *broadens* a system's scope leaves single-repo assumptions hardcoded in the tools built against the old shape](lesson-098-a-migration-that-broadens-a-system-s-scope-leaves-.md) | 2026-06-16 |  |
| [099 - Extracting a hook function into a sibling script can flip its exit status and silently kill the hook under `set -e`](lesson-099-extracting-a-hook-function-into-a-sibling-script-c.md) | 2026-06-16 |  |
| [100 - Extract the shared resolution logic, not the whole caller — keep agent-specific detail in the hook](lesson-100-extract-the-shared-resolution-logic-not-the-whole-.md) | 2026-06-16 |  |
| [101 - A byte-identical parity contract is the tripwire that exposes a template divergence masquerading as a rename](lesson-101-a-byte-identical-parity-contract-is-the-tripwire-t.md) | 2026-06-17 |  |
| [102 - Per-repo git hooks can't enforce a machine-wide invariant — core.hooksPath + a chaining dispatcher is the keystone (GUARD-001)](lesson-102-per-repo-git-hooks-can-t-enforce-a-machine-wide-in.md) | 2026-06-17 |  |
| [103 - A Windows CI tool added only to $GITHUB_PATH vanishes when setup-windows rebuilds PATH from the registry](lesson-103-a-windows-ci-tool-added-only-to-github-path-vanish.md) | 2026-06-17 |  |
| [104 - A WARN that doesn't move the exit code is invisible to CI — give the CI surface its own probe, don't shell out to the tool](lesson-104-a-warn-that-doesn-t-move-the-exit-code-is-invisibl.md) | 2026-06-18 |  |
| [105 - An npm-global CLI under nvm is invisible to GUI/ADE processes and to any shell on a different node version — install agent CLIs into ~/.local](lesson-105-an-npm-global-cli-under-nvm-is-invisible-to-gui-ad.md) | 2026-06-18 |  |
| [106 - An env-var seam is inert until something sets it — a hardcoded fallback that matches reality hides the broken seam](lesson-106-an-env-var-seam-is-inert-until-something-sets-it-a.md) | 2026-06-18 |  |
| [107 - "Wire all consumers" must enumerate the non-shell ones — services and daemons never source a shell profile](lesson-107-wire-all-consumers-must-enumerate-the-non-shell-on.md) | 2026-06-18 |  |
| [108 - Number an ADR off the latest origin/main, not your branch base — a stale base collides with ADRs shipped in parallel](lesson-108-number-an-adr-off-the-latest-origin-main-not-your-.md) | 2026-06-18 |  |
| [109 - `gh issue/pr create` use GraphQL — when that bucket is rate-limited, `gh api -X POST` (REST) still works](lesson-109-gh-issue-pr-create-use-graphql-when-that-bucket-is.md) | 2026-06-18 |  |
| [110 - A release binary goreleaser already builds is worthless until each OS's setup script actually downloads it](lesson-110-a-release-binary-goreleaser-already-builds-is-wort.md) | 2026-06-18 |  |
| [111 - Orca regenerates its Copilot hooks — re-apply the fix idempotently and guard the drift (DX-006)](lesson-111-orca-regenerates-its-copilot-hooks-re-apply-the-fi.md) | 2026-06-19 |  |
| [112 - Strangler-fig deletion: the parity gate must cover OS-specific side effects, and a "different-by-design" Go path can still be parity (CLI-020)](lesson-112-strangler-fig-deletion-the-parity-gate-must-cover-.md) | 2026-06-21 |  |
| [113 - Windows winget jq emits CRLF — breaks `< <(jq)` + read](lesson-113-windows-winget-jq-emits-crlf-breaks-jq-read.md) | 2026-06-20 |  |
| [114 - Catalog installer: release naming is per-repo data, not a convention (CLI-029)](lesson-114-catalog-installer-release-naming-is-per-repo-data-.md) | 2026-06-21 |  |
| [115 - release-please can close a multi-PR issue from a build-only sub-PR's `Refs` — keep the parent issue out of sub-PR footers](lesson-115-release-please-can-close-a-multi-pr-issue-from-a-b.md) | 2026-06-21 |  |
| [116 - `git branch --merged` answers "is the tip an ancestor?", not "is the content backed up" — verify before deleting](lesson-116-git-branch-merged-answers-is-the-tip-an-ancestor-n.md) | 2026-06-21 |  |
| [117 - Resolving Windows `$PROFILE` from Go must include the OneDrive-redirected Documents root](lesson-117-resolving-windows-profile-from-go-must-include-the.md) | 2026-06-21 |  |
| [118 - In bats, a `! grep -q` guard is exempt from errexit — it won't fail the test when the pattern is found](lesson-118-in-bats-a-grep-q-guard-is-exempt-from-errexit-it-w.md) | 2026-06-21 |  |
| [119 - A strict cross-OS `dotf doctor` is not a drop-in CI gate for a lenient platform-specific healthcheck](lesson-119-a-strict-cross-os-dotf-doctor-is-not-a-drop-in-ci-.md) | 2026-06-21 |  |
| [120 - A delete ripples past the direct caller — token guard-greps miss transitive refs, and "orphaned" fixtures can have hidden consumers](lesson-120-a-delete-ripples-past-the-direct-caller-token-guar.md) | 2026-06-21 |  |
| [121 - A thin per-OS shim is still a twin — converge to direct CLI invocation](lesson-121-a-thin-per-os-shim-is-still-a-twin-converge-to-dir.md) | 2026-06-23 |  |
| [122 - A ~120-LOC change is over the SDD bar even when it "obviously" mirrors an existing check](lesson-122-a-120-loc-change-is-over-the-sdd-bar-even-when-it-.md) | 2026-06-23 |  |
| [123 - A Go-vs-shell byte-equivalence gate is POSIX-only, and it retires at cutover](lesson-123-a-go-vs-shell-byte-equivalence-gate-is-posix-only-.md) | 2026-06-24 |  |
| [124 - CI golangci-lint enforces staticcheck QF* quickfixes a stale local version skips — heed the gopls hints](lesson-124-ci-golangci-lint-enforces-staticcheck-qf-quickfixe.md) | 2026-06-24 |  |
| [125 - Three Windows path gotchas behind a "broken" auto-memory junction (Go 1.26)](lesson-125-three-windows-path-gotchas-behind-a-broken-auto-me.md) | 2026-06-25 |  |
| [126 - PR title is the release contract under squash + release-please](lesson-126-pr-title-is-the-release-contract-under-squash-rele.md) | 2026-06-25 |  |
| [127 - agy bakes secrets into JSON; opencode/pi self-decrypt (they ignore ambient env)](lesson-127-agy-bakes-secrets-into-json-opencode-pi-self-decry.md) | 2026-06-25 |  |
| [128 - A new top-level dir backing a dotf runtime read must be deployed by setup](lesson-128-a-new-top-level-dir-backing-a-dotf-runtime-read-mu.md) | 2026-06-25 |  |
| [129 - CI gotchas: Set-Content CRLF on .sh, and repointing tests creates duplicate names](lesson-129-ci-gotchas-set-content-crlf-on-sh-and-repointing-t.md) | 2026-06-25 |  |
| [130 - Determinism "presence" is cheapest as instructions-file injection, not a provider hook](lesson-130-determinism-presence-is-cheapest-as-instructions-f.md) | 2026-06-25 |  |
| [131 - bats silently drops @test names with non-ASCII chars or duplicates — lint them](lesson-131-bats-silently-drops-test-names-with-non-ascii-char.md) | 2026-06-25 |  |
| [132 - Never read a locked secret store as "absent" — discriminate before create, or you spawn duplicates](lesson-132-never-read-a-locked-secret-store-as-absent-discrim.md) | 2026-06-26 |  |
| [133 - On Windows, `bash` from PATH is the System32 WSL launcher, not Git Bash — resolve the real interpreter before shelling out](lesson-133-on-windows-bash-from-path-is-the-system32-wsl-laun.md) | 2026-06-26 |  |
| [134 - `secrets sync ci` refreshed `updated_at` on a dead PAT — a successful write is not a live credential](lesson-134-secrets-sync-ci-refreshed-updated-at-on-a-dead-pat.md) | 2026-06-26 |  |
| [135 - Name-match at the consumer boundary, decouple at the storage boundary](lesson-135-name-match-at-the-consumer-boundary-decouple-at-th.md) | 2026-06-26 |  |
| [136 - A CLI that reads its config from the *deployed* copy, not the checkout, silently reverts its own writes](lesson-136-a-cli-that-reads-its-config-from-the-deployed-copy.md) | 2026-06-27 |  |
| [137 - "Same set as the script it replaces" is the wrong parity gate when the old tool was itself wrong](lesson-137-same-set-as-the-script-it-replaces-is-the-wrong-pa.md) | 2026-06-27 |  |
| [138 - A successful operation is not evidence of the property you depend on — assert the property, not the success](lesson-138-a-successful-operation-is-not-evidence-of-the-prop.md) | 2026-06-27 |  |
| [139 - A "latest/stable" download URL rots silently, and `curl` without `-f` turns a 404 into a corrupt artifact](lesson-139-a-latest-stable-download-url-rots-silently-and-cur.md) | 2026-06-27 |  |
| [140 - A uv tool's Windows launcher is a trampoline that orphans silently — and a running daemon blocks its own repair](lesson-140-a-uv-tool-s-windows-launcher-is-a-trampoline-that-.md) | 2026-06-29 |  |
| [141 - A CLI's `--help`/`Long` strings are untested literals — a dangling doc ref ships green](lesson-141-a-cli-s-help-long-strings-are-untested-literals-a-.md) | 2026-07-01 |  |
| [142 - `bash` on PATH via scoop is not GNU Bash — it silently mis-executes bashisms](lesson-142-bash-on-path-via-scoop-is-not-gnu-bash-it-silently.md) | 2026-07-07 |  |
| [143 - A guard test that names its own trigger string can match itself once tracked](lesson-143-a-guard-test-that-names-its-own-trigger-string-can.md) | 2026-07-08 |  |
| [144 - Keeping a secret off curl's argv: `-K -` (stdin config) is portable; process-substitution and `mktemp` are not](lesson-144-keeping-a-secret-off-curl-s-argv-k-stdin-config-is.md) | 2026-07-08 |  |
| [145 - A three-dot `origin/BASE...HEAD` diff needs the merge-base — `--depth=1` starves it, and a fail-closed gate makes that loud](lesson-145-a-three-dot-origin-base-head-diff-needs-the-merge-.md) | 2026-07-09 |  |
| [146 - A clean local `golangci-lint` does not certify CI — v1 default-excludes errcheck Close/Remove, v2 does not](lesson-146-a-clean-local-golangci-lint-does-not-certify-ci-v1.md) | 2026-07-10 |  |
| [147 - A characterization test can pin the bug you are removing — grep every test extension, not just the source](lesson-147-a-characterization-test-can-pin-the-bug-you-are-re.md) | 2026-07-14 |  |
| [148 - zsh expands aliases at parse time, and the resulting parse error still exits 0](lesson-148-zsh-expands-aliases-at-parse-time-and-the-resultin.md) | 2026-08-04 |  |
| [149 - Validating config files in isolation cannot catch a broken reference between them](lesson-149-validating-config-files-in-isolation-cannot-catch-.md) | 2026-08-04 |  |
| [150 - A config file the tool itself rewrites must be seeded, not synced](lesson-150-a-config-file-the-tool-itself-rewrites-must-be-see.md) | 2026-08-05 |  |
| [151 - A guard can be green because its assertion never ran](lesson-151-a-guard-can-be-green-because-its-assertion-never-r.md) | 2026-08-05 |  |
| [152 - An enforcement gate fails in two directions, and the cheap one is the refusal](lesson-152-an-enforcement-gate-fails-in-two-directions-and-th.md) | 2026-08-06 |  |
| [153 - A guard installed machine-wide can silently disable every other guard](lesson-153-a-guard-installed-machine-wide-can-silently-disabl.md) | 2026-08-06 |  |
| [154 - GraphQL's primary rate limit is billed to the account, not the token](lesson-154-graphql-s-primary-rate-limit-is-billed-to-the-acco.md) | 2026-08-06 |  |
| [155 - A `git revert` cancels a commit's diff but not its message, and GitHub auto-close reads both](lesson-155-a-git-revert-cancels-a-commit-s-diff-but-not-its-m.md) | 2026-08-07 |  |
| [156 - A PR that documents its own text-scanning gate can trip that gate with its own prose](lesson-156-a-pr-that-documents-its-own-text-scanning-gate-can.md) | 2026-08-07 |  |
| [157 - Whether the harness auto-installs a tool decides whether its config deploy is conditional](lesson-157-whether-the-harness-auto-installs-a-tool-decides-w.md) | 2026-08-07 |  |
| [158 - "Zero real invocations" needs the transcript, not the plugin listing — and a removed plugin can still be pinned by a hard-coded count](lesson-158-zero-real-invocations-needs-the-transcript-not-the.md) | 2026-08-06 |  |
| [159 - A guard that is quiet when idle and quiet when broken is not a guard](lesson-159-a-guard-that-is-quiet-when-idle-and-quiet-when-bro.md) | 2026-08-07 |  |
| [160 - A test suite must test the tree it ships in, not the tree it deployed to](lesson-160-a-test-suite-must-test-the-tree-it-ships-in-not-th.md) | 2026-08-07 |  |
| [161 - A linked worktree's checkout is not self-contained — its `.git` is a file, and tools that assume a directory all fail together](lesson-161-a-linked-worktree-s-checkout-is-not-self-contained.md) | 2026-08-07 |  |
| [162 - `gh run rerun` replays the original *event payload*, not just the original workflow file](lesson-162-gh-run-rerun-replays-the-original-event-payload-no.md) | 2026-08-07 |  |
| [163 - A deploy can only prune what it marked, and a compatibility fence set by agent identity points the wrong way](lesson-163-a-deploy-can-only-prune-what-it-marked-and-a-compa.md) | 2026-08-07 |  |
| [164 - The platform's documented cap decides the render, and a shared file is injected into, never written](lesson-164-the-platform-s-documented-cap-decides-the-render-a.md) | 2026-08-08 |  |
| [165 - The defensive half of a fix is the least-reviewed code in the PR, and its failures are silent by construction](lesson-165-the-defensive-half-of-a-fix-is-the-least-reviewed-.md) | 2026-08-08 |  |
| [166 - `git rev-parse` echoes an option it does not understand back at you, and exits 0](lesson-166-git-rev-parse-echoes-an-option-it-does-not-underst.md) | 2026-08-08 |  |
| [167 - A guard can be inverted: matching only the shape that is always a false positive, and blind to the shape that is always a true positive](lesson-167-a-guard-can-be-inverted-matching-only-the-shape-th.md) | 2026-08-08 |  |
| [168 - Two enforcement gates, each correct alone, can compose into a state no change can satisfy](lesson-168-two-enforcement-gates-each-correct-alone-can-compo.md) | 2026-08-08 |  |
| [169 - A structural assertion over a file also matches the comments that explain it](lesson-169-a-structural-assertion-over-a-file-also-matches-th.md) | 2026-08-08 |  |
| [170 - A health report over a tree another process is writing is a dirty read, and re-running it is how you find out](lesson-170-a-health-report-over-a-tree-another-process-is-wri.md) | 2026-08-08 |  |
| [171 - A comment asserting an upstream contract is not evidence of that contract](lesson-171-a-comment-asserting-an-upstream-contract-is-not-ev.md) | 2026-08-08 |  |
| [172 - `gh` splits its subcommands across two rate-limit pools, so a polling loop can exhaust the one you need](lesson-172-gh-splits-its-subcommands-across-two-rate-limit-po.md) | 2026-08-08 |  |
| [173 - A merged PR is not a deployed change, and the deploy takes whatever branch the checkout happens to be on](lesson-173-a-merged-pr-is-not-a-deployed-change-and-the-deplo.md) | 2026-08-08 |  |
| [174 - A workaround installed to route around a bug outlives the bug silently, because nothing re-examines it](lesson-174-a-workaround-installed-to-route-around-a-bug-outli.md) | 2026-08-08 |  |
| [175 - A test that does not isolate from the machine ends up measuring the machine](lesson-175-a-test-that-does-not-isolate-from-the-machine-ends.md) | 2026-08-08 |  |
| [176 - Reusing a helper inherits its error policy, not just its code](lesson-176-reusing-a-helper-inherits-its-error-policy-not-jus.md) | 2026-08-09 |  |
| [177 - A guard stops new violations; it does not clean the stock, and the rule then reads as if it did](lesson-177-a-guard-stops-new-violations-it-does-not-clean-the.md) | 2026-08-09 |  |
| [178 - The sandbox is not the territory: a dry run that can reach production, and a fixture that cannot know what production contains](lesson-178-the-sandbox-is-not-the-territory-a-dry-run-that-ca.md) | 2026-08-09 |  |
| [179 - GitHub Actions supplies the `-e`, so `set -uo pipefail` disables nothing](lesson-179-github-actions-supplies-the-e-so-set-uo-pipefail-d.md) | 2026-08-09 |  |
| [180 - A freshness check that includes the artifact it validates is stale by construction](lesson-180-a-freshness-check-that-includes-the-artifact-it-va.md) | 2026-08-09 |  |
| [181 - Pin a characterization oracle by content, not by commit — a SHA answers the wrong question](lesson-181-pin-a-characterization-oracle-by-content-not-by-co.md) | 2026-08-09 |  |
| [182 - A check that never ran and an escape that was never taken are the same defect: verified in isolation, never exercised in situ](lesson-182-a-check-that-never-ran-and-an-escape-that-was-neve.md) | 2026-08-09 |  |
| [183 - A real-dependency test can rest on an undeclared environment precondition, and then it tests one thing locally and another in CI](lesson-183-a-real-dependency-test-can-rest-on-an-undeclared-e.md) | 2026-08-09 |  |
| [184 - Two clients, one resource: the green path and the red path shared a credential, so the credential was never the answer](lesson-184-two-clients-one-resource-the-green-path-and-the-re.md) | 2026-08-09 |  |
| [185 - Mutation testing does not only catch tautologies — it finds the boundaries your fixtures never land on](lesson-185-mutation-testing-does-not-only-catch-tautologies-i.md) | 2026-08-09 |  |
| [186 - The dangerous shell incompatibility is the one that answers wrongly instead of failing](lesson-186-the-dangerous-shell-incompatibility-is-the-one-tha.md) | 2026-08-09 |  |
| [187 - Cobra's `Print` family writes to stderr, and a `SetOut` test cannot tell you otherwise](lesson-187-cobra-s-print-family-writes-to-stderr-and-a-setout.md) | 2026-08-10 |  |
| [188 - The reviewer that cannot be you: four rounds, four defects, three of them in the fix for the last one](lesson-188-the-reviewer-that-cannot-be-you-four-rounds-four-d.md) | 2026-08-11 |  |
| [189 - An apostrophe in a comment inside an open `awk '...'` block reopens bash's own parser](lesson-189-an-apostrophe-in-a-comment-inside-an-open-awk-bloc.md) | 2026-08-12 |  |
| [190 - A bash `case` pattern is a glob, not a regex — `g[a-z]*` doesn't mean what it looks like it means](lesson-190-a-bash-case-pattern-is-a-glob-not-a-regex-g-a-z-do.md) | 2026-08-12 |  |
| [191 - Continuing work on a branch after its PR squash-merged reopens the whole original diff](lesson-191-continuing-work-on-a-branch-after-its-pr-squash-me.md) | 2026-08-12 |  |
| [192 - A "multi-call binary" bug report can name the wrong mechanism — verify the dispatch, not just the symptom](lesson-192-a-multi-call-binary-bug-report-can-name-the-wrong-.md) | 2026-08-12 |  |
| [193 - "Weaker locally, CI catches it" is not safe when local and CI share the same script](lesson-193-weaker-locally-ci-catches-it-is-not-safe-when-loca.md) | 2026-08-12 |  |
| [194 - "Looks like a known bug class" is a hypothesis, not a finding — reproduce before you fix](lesson-194-looks-like-a-known-bug-class-is-a-hypothesis-not-a.md) | 2026-08-12 |  |
| [195 - `resolveRepoDir`'s cwd fallback silently defeats "unresolvable repo" test cases](lesson-195-resolverepodir-s-cwd-fallback-silently-defeats-unr.md) | 2026-08-12 |  |
| [196 - A dangling citation and a missing file are different bugs — check for the first before assuming the second](lesson-196-a-dangling-citation-and-a-missing-file-are-differe.md) | 2026-08-12 |  |
| [197 - A health check that reads local state proves the liveness of nothing](lesson-197-a-health-check-that-reads-local-state-proves-the-l.md) | 2026-08-13 |  |
| [198 - A PR's `head.sha` not matching your latest push can mean the PR is already merged, not that the API is lagging](lesson-198-a-pr-s-head-sha-not-matching-your-latest-push-can-.md) | 2026-08-13 |  |
| [199 - A default is not a pin: the model that reviewed your code may not be the one you think](lesson-199-a-default-is-not-a-pin-the-model-that-reviewed-you.md) | 2026-08-13 |  |
| [200 - A dormant declared field must be validated on the same schedule it's written, not the schedule it activates on](lesson-200-a-dormant-declared-field-must-be-validated-on-the-.md) | 2026-08-14 |  |
| [201 - An agent that cannot reach the repo still writes a confident review](lesson-201-an-agent-that-cannot-reach-the-repo-still-writes-a.md) | 2026-08-14 |  |
| [202 - Widening a shared return type is a change to every consumer, and Go's zero values hide the ones you missed](lesson-202-widening-a-shared-return-type-is-a-change-to-every.md) | 2026-08-14 |  |
| [203 - A check whose precondition the architecture forbids reports SKIP forever, and SKIP reads as nothing-to-check](lesson-203-a-check-whose-precondition-the-architecture-forbid.md) | 2026-08-15 |  |
| [204 - A check that cannot fail the way you cite it](lesson-204-a-check-that-cannot-fail-the-way-you-cite-it.md) | 2026-08-15 |  |
| [205 - Redact at the producer, because the consumer's filter is a guess about a format you have not seen](lesson-205-redact-at-the-producer-because-the-consumer-s-filt.md) | 2026-08-15 |  |
| [206 - `bw status` answers for the CLI's session, not for the daemon your code actually uses](lesson-206-bw-status-answers-for-the-cli-s-session-not-for-th.md) | 2026-08-15 |  |
| [207 - A gate that scans a directory will eventually scan its own evidence](lesson-207-a-gate-that-scans-a-directory-will-eventually-scan.md) | 2026-08-15 |  |
| [208 - The health probe was the illness: a liveness check that breaks the operation it authorises](lesson-208-the-health-probe-was-the-illness-a-liveness-check-.md) | 2026-08-15 |  |
| [209 - Every layer reported a health none of them had established](lesson-209-every-layer-reported-a-health-none-of-them-had-est.md) | 2026-08-16 |  |
| [210 - Under squash-merge, `git branch --merged` says no about every branch that landed](lesson-210-under-squash-merge-git-branch-merged-says-no-about.md) | 2026-08-16 |  |
| [211 - Worktree config discovery must prefer CWD walk-up over global repo env](lesson-211-worktree-config-discovery-must-prefer-cwd-walk-up-over-global-repo-env.md) | 2026-08-18 |  |
| [212 - An invalid instrument is indistinguishable from an absent guard](lesson-212-an-invalid-instrument-is-indistinguishable-from-an.md) | 2026-08-19 |  |
| [213 - A reviewer that reports success while publishing nothing, in two shapes](lesson-213-a-reviewer-that-reports-success-while-publishing.md) | 2026-08-20 |  |
| [214 - A declared status is not evidence, and a guard that exists is not a guard that covers](lesson-214-a-declared-status-is-not-evidence-probe-the-syst.md) | 2026-08-20 |  |
| [215 - A parser written for one runner reads the other runner's review as empty](lesson-215-a-parser-for-one-runner-reads-the-other-runners-re.md) | 2026-08-21 |  |
| [216 - A three-dot diff does not answer "what is main missing"](lesson-216-a-three-dot-diff-does-not-answer-what-main-is-missi.md) | 2026-08-21 |  |
| [217 - `go test -run` passes when the test name matches nothing](lesson-217-go-test-run-passes-when-the-test-name-matches-not.md) | 2026-08-21 |  |
| [218 - The Go build cache does not see the data file your test reads](lesson-218-the-go-build-cache-does-not-see-the-data-file-your.md) | 2026-08-21 |  |
| [219 - A stale CLI refuses with the same exit status as a legitimate refusal](lesson-219-a-stale-cli-refuses-with-the-same-exit-status-as-a.md) | 2026-08-21 |  |
| [220 - Four defects, one shape: a thing verified by a proxy that lives somewhere else](lesson-220-four-defects-one-shape-a-thing-verified-by-a-proxy.md) | 2026-08-22 |  |
| [221 - An allow-list merge makes every new template key a silent no-op](lesson-221-an-allow-list-merge-makes-every-new-template-key-a.md) | 2026-08-22 |  |
| [222 - A coupling's scope is measured, not inferred from where you saw it fail](lesson-222-a-couplings-scope-is-measured-not-inferred-from-whe.md) | 2026-08-23 |  |
| [223 - A test updated to keep passing stops being a guard](lesson-223-a-test-updated-to-keep-passing-stops-being-a-guar.md) | 2026-08-23 |  |
| [224 - A negated assertion is exempt from `set -e`, so it cannot fail a test](lesson-224-a-negated-assertion-is-exempt-from-set-e-so-it-cann.md) | 2026-08-23 |  |
| [225 - A stacked PR costs a reviewer, and re-conflicts on every squash](lesson-225-a-stacked-pr-costs-a-reviewer-and-re-conflicts-on-e.md) | 2026-08-23 |  |
| [226 - Naming a key under `properties` exempts it from `additionalProperties`, so adding a constraint there can loosen the schema](lesson-226-naming-a-key-under-properties-exempts-it-from-additio.md) | 2026-08-23 |  |
| [227 - A test suite inherits the developer's installed applications, and PATH is the door](lesson-227-a-test-suite-inherits-the-developers-installed-apps.md) | 2026-08-23 |  |
| [228 - A calendar date is not a timestamp, and UTC is the wrong zone for one](lesson-228-a-calendar-date-is-not-a-timestamp-utc-is-wrong-for.md) | 2026-08-23 |  |
| [229 - An empty secret is not an error, so a job with no credential fails as if the work failed](lesson-229-an-empty-secret-is-not-an-error-so-a-job-with-no-cr.md) | 2026-08-24 |  |
| [230 - A config that parses is not a config the consumer reads](lesson-230-a-config-that-parses-is-not-a-config-the-consumer-re.md) | 2026-08-24 |  |
| [231 - A hand-wired dev symlink outranks the managed install, and the host fails closed](lesson-231-a-hand-wired-dev-symlink-outranks-the-managed-instal.md) | 2026-08-26 |  |
| [232 - Detect the shape that is wrong, not the shape that is merely absent](lesson-232-detect-the-shape-that-is-wrong-not-the-shape-that.md) | 2026-08-26 |  |
| [233 - Piping a single-element array into ConvertTo-Json unwraps it](lesson-233-piping-a-single-element-array-into-convertto-json-un.md) | 2026-08-26 |  |
| [234 - Orchestrating Orca ADE declarative configuration and bidirectional settings capture](lesson-234-orchestrating-orca-ade-declarative-configuration-and-bi.md) | 2026-08-26 |  |
| [235 - Reproducing a bug from an environment that already works measures the environment, not the bug](lesson-235-reproducing-a-bug-from-an-environment-that-alread.md) | 2026-08-27 |  |
| [236 - A CI job that installs a tool and never puts it on PATH certifies nothing it was built to check](lesson-236-a-ci-job-that-installs-a-tool-and-never-puts-it-on-p.md) | 2026-08-27 |  |
| [237 - CREATE_NEW_PROCESS_GROUP is not detachment: a console child dies with its terminal](lesson-237-create-new-process-group-is-not-detachment-a-console.md) | 2026-08-27 |  |
| [238 - Two independent defects, each sufficient for a permanent red, hide behind one symptom](lesson-238-two-independent-defects-each-sufficient-for-a-perman.md) | 2026-08-27 |  |
| [239 - Re-measure a filed bug on the current toolchain before implementing its fix](lesson-239-re-measure-a-filed-bug-on-the-current-toolchain-before.md) | 2026-08-27 |  |
| [240 - A partial mirror the code believes absent: the check that could have flagged the gap was the one switched off](lesson-240-a-partial-mirror-the-code-believes-absent-the-check.md) | 2026-08-27 |  |
| [241 - Re-running a command just to print what it already told you inherits its exit status, silently, under `pipefail`](lesson-241-re-running-a-command-just-to-print-what-it-already.md) | 2026-08-27 |  |
| [242 - A process nobody can watch must leave its own trace, and the redirect has to survive the parent](lesson-242-a-process-nobody-can-watch-must-leave-its-own-trace.md) | 2026-08-27 |  |
| [243 - A guard that reads a cache reports the cache's age as the credential's health](lesson-243-a-guard-that-reads-a-cache-reports-the-cache-not-the-credential.md) | 2026-08-28 |  |
| [244 - A sweep is bounded by what the writer recorded, not by what the store holds; and the store's name rules decide the order](lesson-244-a-sweep-is-bounded-by-what-the-writer-recorded-not-by-what-the-store-holds.md) | 2026-08-29 |  |
| [245 - A fix applied to one of two renderers is an outage on the other OS, and nothing on the fixed side can see it](lesson-245-a-fix-to-one-of-two-renderers-is-an-outage-on-the-other-os.md) | 2026-08-29 |  |
| [246 - A guard that reads its own evidence from a file allowed not to exist switches itself off exactly when it is needed](lesson-246-a-guard-that-reads-its-own-evidence-from-an-optional-file.md) | 2026-08-30 |  |
