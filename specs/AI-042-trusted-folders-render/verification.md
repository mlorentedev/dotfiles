---
tags: [spec, verification]
created: "2026-08-29"
---

# Verification - AI-042-trusted-folders-render

## Evidence

Run on the Windows work box, 2026-08-29, worktree `dotfiles-wt-trusted`, branch
`feat/trusted-folders-render` (stacked on #1369), `dotf` built from the branch, Copilot CLI 1.0.81,
agy 1.1.14.

- [x] **AC1** → `TestExpandPaths_RendersTheDeclaredForm` (native and slash from one source; untouched
  strings; valid JSON; on Windows the native output carries `\\Users\\`, escaped by the encoder),
  `TestExpandPaths_RejectsWhatItCannotRender` (unknown form, non-JSON source, unresolvable token
  named), `TestParseManifest_ValidatesPathsByName`. Mutation: `FromSlash` skipped → red; restored.
- [x] **AC2** → `TestDeploy_PathsComposeWithMergeAndReportInSync` (merge keeps `firstLaunchAt`, the
  rendered list replaces the managed key, `PlanConfig` in sync afterwards). Mutation: expansion
  moved after the merge → red; restored.
- [x] **AC3** → `tests/copilot-config.bats` (30/30): every JSON under `ai/*/` free of `/home/<user>`,
  `C:\Users\<user>`, `C:/Users/<user>`; both templates asserted; manifest v3 with
  `copilot-config … native` and `agy-settings … slash`; neither setup carries the copy (invocation
  line, not mention). `TestManifestVersion_FreezesTheFieldSet` extended to v3 with `paths`;
  `TestParseManifest_ShippedManifestIsValid` green. `shellcheck setup-linux.sh` 0 warnings;
  `setup-windows.ps1` parse 0 errors, non-ASCII 32 = origin/main, CRLF intact.
- [x] **AC4** → box, in order:
  1. `dotf deploy --dry-run` → `would deploy copilot-config`, `would deploy agy-settings`, the other
     four `in sync`.
  2. `dotf deploy` → both deployed. `~/.copilot/config.json`: `firstLaunchAt` kept, `trustedFolders` =
     `C:\Users\mlorente\Projects`, `…\Projects\*`, `…\Projects\Workspace`, `…\Projects\Workspace\*`
     (the CLI's own form; the two `/home/manu` entries a Windows box never needed are gone).
     `~/.gemini/antigravity-cli/settings.json`: `trustedWorkspaces` = `C:/Users/mlorente/Projects/*`,
     `C:/Users/mlorente/Projects/Workspace/*`.
  3. `dotf deploy` again → six × `in sync`.
  4. `dotf doctor` → `[Deployed agent configs (ai/deploy.json)] (1 checks, all ok)`.
  5. `copilot -p "Reply with exactly: OK"` → `OK`; `agy --version` → `1.1.14`.

## Test status

```text
go build ./... && go vet ./... && GOOS=linux go vet ./... && go test ./internal/deploy/ ./internal/cmd/ ./internal/doctor/   -> ok
golangci-lint run ./internal/deploy/   -> 0 issues
bats tests/copilot-config.bats tests/antigravity.bats; bats tests/setup-{linux,windows}.bats -f 'agy|antigravity|AI-042|deploy'   -> all ok
```

- No regressions in the existing suite: yes.

## Decisions made during implementation

- **The separator form is declared per entry, from evidence.** Copilot's trust check is
  `repoPathsEqual` in a native module and `copilot -p` does not evaluate trust at all (measured:
  identical behaviour with a forward-slash list and an empty list), so the only evidence for
  Copilot is the backslash form the CLI wrote itself when the user trusted folders on Windows →
  `native`. agy has read `C:/…` from our file on this box daily → `slash`. A guess in either
  direction would have been silent on the box it was wrong for.
- **JSON-aware expansion, not textual.** A native Windows path holds backslashes; the encoder
  escapes them. Key order follows the encoder (sorted), which is stable run to run, so "in sync"
  stays answerable.
- **agy's settings move onto the manifest** rather than gaining a third render path in two shell
  twins (ADR-020 C7). The first merge shipped the entry **without** `requires: agy` — the setups
  had copied the file unconditionally and `tests/antigravity.bats` asserted it exists — while
  AC3 said `requires: agy`. Review round 1 (`nan/deepseek-v4-flash`, FAIL) caught the
  disagreement (Major, REAL) and the consequence (Major, THEORETICAL): an ungated entry deploys
  `~/.gemini/antigravity-cli/settings.json` onto a box without agy, the #843/#1312 class the
  manifest's own `$comment` names. Reconciled toward the AC, not away from it: `requires: agy`
  added; safe because both setups install agy **before** `dotf deploy` (setup-windows.ps1 472 →
  1287, setup-linux.sh 370 → the deploy block), so a box that carries agy still gets the file on
  first run; the bats test skips where the binary is absent, the same gate the manifest applies.
- **`replace` stands for `agy-settings`** (round 1, next step 2 — "does agy write its own
  settings.json?"). Measured on the box: the deployed file's key set equals the source's, no key
  added by agy after daily use; agy reads, it does not write. Copilot does write its
  `config.json` (`firstLaunchAt`), which is why that entry is `merge`.
- **`TestDeploy_PathsComposeWithReplace`** added (round 1, Minor THEORETICAL): AC2's replace half
  had only manifest-shape bats and box evidence; it now has a named regression test beside the
  merge one.
- **Manifest version 3.** `paths` changes what an entry's content means; the frozen-schema test
  forced the bump, which is the point of that test.

## Promotion candidates

- [ ] Lesson: no — the spec and the manifest comment carry the measurement.
- [ ] ADR-worthy decision: no — extends CLI-039's manifest under ADR-020 C7 / ADR-025.
- [ ] Pattern: no.

## Archive checklist

- [ ] `dotf spec review AI-042-trusted-folders-render` PASS
- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/AI-042-trusted-folders-render/`
- [ ] Bitácora #1334 closed with the PR link
