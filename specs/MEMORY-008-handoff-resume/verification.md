---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - MEMORY-008-handoff-resume

## Evidence

- [x] AC1 -> slice 3: `TestResumeIsAPureRenderOfTheThreadFile`
- [x] AC2 -> slices 1 and 3: `TestResumeRendersVerifyAndAwaiting`, `TestResumeListsCandidatesWhenTheThreadIsAmbiguous`
- [x] AC3 -> slice 4: `TestSessionStartResumeBySource`
- [x] AC4 -> slice 2: `TestAMemoryShapedLikeDotfilesStaysUnderBudgetWithTheNewFields`, `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget`
- [x] AC5 -> slice 5: the `handoff` and `catchup` records
- [x] AC6 -> slice 5: the compiled doctrine, `--check` and the 8,000-character budget test

Each slice PR fills its own section below and nothing else.

## Slice 0 evidence (prerequisite bugs, each on its own issue)

- #1620, `session-end` replaced an authored journal: PR #1701. `TestSessionEndLeavesAnAuthoredJournalByteIdentical` fails on main and passes with the fix.
- #1606, `handoff-write` keyed a thread from another repository: PR #1702. `TestHandoffThreadRefusesAKeyFromAnotherRepository` and `TestMemHandoffWriteFromAnotherRepositoryTouchesNoThread` fail on main (the first cannot build there, the second finds the other session's thread replaced) and pass with the fix.
- #1651, the legacy block stays in front: PR #1703. `TestWriteThreadMovesTheUnthreadedBlockIntoALegacyThread`, `TestWriteThreadMigratesEvenWhenItsOwnThreadIsUnchanged`, `TestWriteThreadNamesAnUndatedLegacyBlockAndNeverCollides` and `TestMemHandoffWriteDryRunShowsTheLegacyMigration`.

## Slice 1 evidence

- `ParseThread` and `ThreadLabels` in `cli/internal/mem/thread.go`.
  - `TestParseThreadReadsEveryCanonicalLabel`: all eight labels are read.
  - `TestParseThreadReadsTheImprovisedLabelsAsTheirFields`: `Verify at start` is read as `Verify`, and `Judgment calls left open` as `Awaiting Manu`.
  - `TestParseThreadKeepsAnUnknownLabel`: an unknown label is kept. A qualified canonical label counts as unknown, because its qualifier is content.
  - `TestParseThreadDropsNoLine`.
  - `TestParseThreadReadsEitherFormOfTheDateAndJournal`.
- `handoff-write` names a missing or empty `Next action`, and every unknown label, on stderr. It still writes the body. Tests: `TestThreadWarningsNameOnlyWhatIsMissingOrUnknown`, `TestMemHandoffWriteNamesWhatTheBodyLacksAndStillWrites`.
- **Mutation checks**, each reverted:

  | Mutation | Test that went red |
  |---|---|
  | aliases ignored | the improvised-labels test |
  | continuation lines dropped | two tests |
  | an empty `Next action` accepted | the warnings test |

- **Found along the way: `handoff-write` narrowed the mode of the file it replaced.** `os.CreateTemp` creates the temp file 0600, and the rename carried that mode onto MEMORY.md. On 2026-09-25, 9 of the vault's 21 MEMORY.md files were 0600, the ones this command had written; the rest were 0664.
  - Fix: the temp file takes the replaced file's mode before the rename, as `harness bind` does for settings files. Test: `TestMemHandoffWriteKeepsTheFileMode`, which failed at 600 before the fix.
  - The write now lives in `replaceMemoryFile`. That brings `newMemHandoffWriteCmd` from 17 (after #1711 and this slice's warnings) down to 14.
- Rebased onto #1711. Its `TestMemHandoffWriteWithoutAgentSaysWhatItAlwaysSaid` expected an empty stderr with a body that has no `Next action`, which this slice now warns about. The test now passes a body with one, so it still checks only what `--agent` could add.
- **Probe over every thread in the vault's `10_projects/*/memory/MEMORY.md`** (a throwaway test, not committed):
  - 34 threads parsed, with 0 lines lost.
  - 1 thread has no `Next action`.
  - 9 fields carry one of 7 labels outside the set. The most frequent are `Trap` and `Decisions (Manu, 2026-09-24)`, twice each. These are what the new warning will name.

## Slice 2 evidence

- The store is in `cli/internal/mem/thread_store.go`, and the design is in ADR-056.
  - `StoreThreadAs` writes the body to `memory/threads/<key>.md` and puts the derived entry (Updated, the first line of the Next action, the Awaiting Manu count, Journal, pointer) under the heading in `MEMORY.md`. Tests: `TestStoreThreadWritesTheBodyToItsFileAndADerivedEntry`, `TestTheDerivedEntryCutsALongNextAction` (entry at most 350 B), `TestTheDerivedEntryCapsEveryQuotedField` (every quoted field at its cap: at most 450 B).
  - Ownership and forks are shared with `WriteThreadAs` through `threadOwner`. Test: `TestStoreThreadForksAroundAnotherAgentsEntry`.
  - Migration on the next write: `TestStoreThreadMovesAnInlineBlockAndLeavesTheOthersByteIdentical`.
  - Key to file: `TestThreadFileNames`. The one live key with a `/` (ts-bridge `fix/ux355-init-headscale-control-plane`, 74 thread headings checked across the vault) maps to `fix-ux355-init-headscale-control-plane.md`. Collisions: `TestCheckThreadFileRefusesAFileAnotherKeyHolds`. A pointer in a body is refused: `TestStoreThreadRefusesAPointerInTheBody`.
- `handoff-write` writes the thread file first, then `MEMORY.md`, under the existing lock. A new file takes `MEMORY.md`'s mode. Tests: `TestMemHandoffWriteStoresTheThreadInItsOwnFile`, `TestMemHandoffWriteDryRunStoresNothing`.
- **Budget (#1477):** `CheckMemoryBudget` refuses only a write that grows the file past 25,000 B, and names the three least recently updated dated threads, each with how to archive it (stored or inline); undated threads are listed apart and never suggested. Tests: `TestCheckMemoryBudget`, `TestTheBudgetRefusalSuggestsOnlyDatedThreadsAndHowToArchiveEach`, `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget`.
- **Found along the way: `session-end` reads the thread too.** Its fallback journal archives the block from `MEMORY.md`, so after the store it would have archived the entry. It now reads through `threadText`. Test: `TestSessionEndArchivesTheStoredThreadNotItsEntry`. Lesson 507.
- **Probe on a copy of the real dotfiles `MEMORY.md`** (2026-10-20, a throwaway test, not committed). Each of its 8 threads was written again with a `Verify` line and two `Awaiting Manu` items:

  | | Bytes |
  |---|---|
  | as it is | 23,423 |
  | inline, with the fields | 24,488 |
  | stored, with the fields | 14,516 |
  | budget | 25,000 |

- **Mutation checks**, each reverted:

  | Mutation | Tests that went red |
  |---|---|
  | budget check off | `TestCheckMemoryBudget`, `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget` |
  | over-budget writes that shrink also refused | `TestCheckMemoryBudget` |
  | collision check off | `TestCheckThreadFileRefusesAFileAnotherKeyHolds` |
  | Next action not cut | `TestTheDerivedEntryCutsALongNextAction` |
  | resolver ignores the pointer | `TestThreadTextFollowsThePointer`, `TestSessionEndArchivesTheStoredThreadNotItsEntry` |

## Slice 3 evidence

- `dotf mem resume [--thread K] [--agent A] [--format prompt|context] [--memory PATH]` in `cli/internal/cmd/mem_resume.go`. The render is in `cli/internal/mem/resume.go`.
  - `RenderResume` is pure: `TestResumeIsAPureRenderOfTheThreadFile` renders a fixture twice per format and compares both renders with `testdata/resume/{context,prompt}.golden`. An unknown format is refused.
  - Next action, Verify and Awaiting Manu render first. Verify commands are listed one per line, and one outside the read-only allowlist is marked unverified (`TestResumeRendersVerifyAndAwaiting`). The allowlist matches on a word boundary, and any chaining, pipe, redirection or substitution fails it (`TestReadOnlyCommand`).
  - `SelectThread` takes the agent's fork of the key, then the key unless another agent's stamp is on it. Otherwise it picks none and returns every thread key (`TestResumeListsCandidatesWhenTheThreadIsAmbiguous`). The command lists those keys, each with its `--thread` flag (`TestMemResumeListsCandidates`).
  - `ThreadText` reads the stored file when the entry points at one, and the inline block otherwise. It also returns the source path (`TestThreadTextForKey`). Round trip through the command: `TestMemResumeRendersWhatHandoffWriteStored`.
- `--memory` defaults to the checkout project's vault `MEMORY.md`, through `mem.ProjectMemory`. That helper was lifted out of `session-end`, which resolved the same path inline, so the layout is derived in one place.
- The read-only allowlist leaves out generic file readers (`cat`, `grep`, `head` and the like), because they take any path and `cat <secrets file>` would render as safe. It also leaves out subcommands that have a writing form (`git branch`, `git fetch`, `go build`). A line break counts as shell control, and an argument that writes a file (`--output`, `-o`) fails the check (review of #2880).
- **Live run** on a copy of the real dotfiles `MEMORY.md` (inline blocks, 2026-10-21). `--thread feat-plat003-close@macmini --agent claude` rendered that thread's Next action. `--thread nope` listed all 8 threads.
- **Mutation checks**, each reverted:

  | Mutation | Test that went red |
  |---|---|
  | chaining allowed | `TestReadOnlyCommand` |
  | prefix without a word boundary | `TestReadOnlyCommand` |
  | another agent's thread chosen | `TestResumeListsCandidatesWhenTheThreadIsAmbiguous` |
  | this agent's fork ignored | `TestResumeListsCandidatesWhenTheThreadIsAmbiguous` |
  | Verify not rendered first | `TestResumeIsAPureRenderOfTheThreadFile` |

  The first attempt at the "another agent's thread" mutation did not compile. The harness printed no failing test, which read as a surviving mutant (lesson 508).

## Decisions made during implementation

- Slice 2: the AC4 fixture is synthetic, shaped like the real file. Copying the vault's `MEMORY.md` into a public repository would publish it. The real file was probed instead, and the probe's figures are recorded above.
- Slice 2: a write that keeps or shrinks a file already over the budget is allowed, so that file can still be brought back under (ADR-056 §7).

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-513-a-read-only-allowlist-that-names-a-code-runner-is-not-read-only.md (also lesson 507 and lesson 508)
- [x] ADR-worthy decision for the repo's `docs/adr/`? (the per-thread store changes the layout of every project's `memory/`) yes: docs/adr/adr-056-a-handoff-thread-is-stored-in-its-own-file.md
- [x] New pattern candidate for `00_meta/patterns/`? no: the store and the resume render are this repo's `dotf mem` contract, recorded in ADR-056; nothing in them generalises beyond it yet

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/MEMORY-008-handoff-resume/` -> `specs/archive/MEMORY-008-handoff-resume/`
- [ ] Bitácora board ticket moved to Done / closed with the closing PR (ADR-018)
- [ ] Promotions above executed

## Slice 4 evidence

- `ClaudeContext` takes the hook's `source` (`ClaudeContextInput.Source`, parsed by `hookFromPayload` in `cli/internal/cmd/mem.go`). It puts `sessionResume` (`cli/internal/mem/session_start_resume.go`) ahead of the rest of the brief.
  - `startup`, `clear` and `compact` get the full `dotf mem resume --format context` block, because those sessions lost the conversation. `resume` and `fork` kept it, so they get one line naming the thread and the command. An unknown or missing source gets the line too.
  - The thread is chosen by `SelectThread` for the agent `claude`, so another agent's thread or an absent memory adds nothing (`TestSessionStartResumeInjectsNothingThatIsNotThisAgents`).
  - One golden per source: `testdata/session_start_resume/<source>.golden` (`TestSessionStartResumeBySource`). The same test checks that a compacted `ClaudeContext` begins with the block.
  - The `handoff_resume` injector flag turns the block off (`TestSessionStartResumeCanBeTurnedOff`). Like every injector, an absent flag means on.
- **Mutation checks**, each reverted. All 5 were caught by a failing test, with no build failures:

  | Mutation | Test that went red |
  |---|---|
  | compact gets one line | `TestSessionStartResumeBySource` |
  | resume and fork get the full block | `TestSessionStartResumeBySource` |
  | agent ignored | `TestSessionStartResumeInjectsNothingThatIsNotThisAgents` |
  | block appended, not first | `TestSessionStartResumeBySource` |
  | flag ignored | `TestSessionStartResumeCanBeTurnedOff` |

- **Decisions:**
  - `startup` gets the full block. tasks.md lists it with `clear` and `compact`; AC3 names only the last two. A fresh session has no conversation either.
  - `session_id` is not parsed. Nothing reads it, and a field with no reader is dead weight.
  - `session-start-config.json` does not declare `handoff_resume` yet. That file and its bats (which counts 11 injectors) are outside this run's write scope. The injector is on by default, so behaviour does not depend on it. Follow-up: #2915.

### Refusal fix (#1477 follow-up): mutations

The first live refusal on the real file named `legacy-undated` first and told the reader to move a thread file that inline blocks do not have. Each mutation below was applied to `thread_store.go`, `go vet` passed, and `go test ./internal/mem -run Budget` went red:

| Mutation | Result |
|---|---|
| Undated threads are suggested again | killed |
| `Journal` date fallback removed | killed |
| Inline threads worded as stored | killed |
| Dated threads left unsorted | killed |
| The no-dated-thread message removed | killed |

### Slice 5: skills and doctrine (AC5, AC6)

- **AC5:** the vault `catchup` skill's step 1 runs `dotf mem resume --agent <agent>` and its `Verify` commands, and lists and asks when the thread is ambiguous; the `handoff` skill's summary points the next session at `dotf mem resume` and says never to hand-type a resume prompt. Vault commit `75d90a9a`; the repo records are its `--refresh` output. f5's command exits 0.
- **AC6:** the fallback is one bullet in the `pr-stewardship` record, one of the nine records `manifest.json`'s `doctrine.inject` puts in the compact doctrine (its vault source is `pattern-change-lifecycle.md#pr-stewardship`), so the sentence is in the compiled doctrine; the 117-character growth of the agy surface below is that bullet. It sits beside the other session-start obligation (`dotf pr triage-queue`): "at session start, where no hook injected a resume block, run it and follow its Next action". It is conditional so it does not contradict the SessionStart injection Claude already gets. It goes in an existing record rather than a new enforced id: a new id costs a manifest row, an inject row and coverage entries for one sentence. f6's command exits 0.
- **Budget:** the capped agy surface measured 7,574 characters on main before this slice (the proposal's 7,798 was stale) and 7,691 after it, against the 8,000 budget.
- `--check` exit 0 is evidence that the enforced regions match their records, not that the skill records are fresh: it passes on a tree whose skill records lag the vault (#1502). The freshness evidence for this slice's records is that they are byte-for-byte the `--refresh` output from `75d90a9a`, and f5 greps them.
- `--refresh` also rewrote the `adversarial-review` and `crystallize` records from vault edits other sessions made; those are left out of this slice (#1502).

### Review round 1 (FAIL) and its fixes

The first independent review (`review-round-1.md`, at `302a427b`, nan/qwen3.8-flash) returned FAIL on two REAL Majors inside AC2. Every finding, with its fix:

| Finding | Fix | Test |
|---|---|---|
| F-A: the read-only list rendered code runners' escape hatches as safe (`go test -exec=`, `--outputdir=`, `bats /tmp/x.bats`) | the four code runners pass only allowlisted flags, and positional arguments stay inside the checkout (lesson 513) | `TestReadOnlyCommand`: the three reproduced inputs plus `-toolexec`, `-vettool`, `-cpuprofile`, `--formatter`, `~`, `..`; controls `./...`, `-run '^X$'`, `--filter 'a b'` |
| F-B: `SelectThread` read only the heading stamp, so a thread owned by its journal was handed to another agent | `SelectThread` reads ownership with `threadWriter`, as `threadOwner` does | `TestResumeListsCandidatesWhenTheThreadIsAmbiguous`: `feat-j` (journal names pi) |
| F-C: with no `--agent`, an owned thread was handed to any caller | a caller that names no agent gets only a thread nobody owns; the hook always names `claude`, catchup passes `--agent`, and the doctrine fallback now reads `dotf mem resume --agent <agent>` (vault `552675d8`; agy surface 7,762 characters against 8,000) | same test: `feat-a` with no agent lists candidates |
| F-D: the printed source was rebuilt as `Dir(memory)/MEMORY.md` | `ThreadText` takes the memory path and returns it for an inline block | `TestThreadTextForKey` reads through `memory2.md` |
| F-E: the `handoff_resume` declaration was a verbal follow-up | filed as #2915 | n/a |
| F-F: AC5 and AC6 unticked in the Evidence list | ticked | n/a |
| F-G: a one-item list collapsed onto its label line | `writeField` treats a value starting `- ` as a list | `TestResumeRendersAOneItemListAsABlock` |
| Speculative: the launcher's base gives a 750-commit range | not this change's contract; the review read the nine slice commits | n/a |
| Question: does the `matcher: "*"` SessionStart group fire on `clear`/`compact` | Claude Code's SessionStart matcher filters on `source`, and `*` matches every source; a live `source=compact` payload is still to be observed | n/a |

Mutations, each applied with `go vet` passing, all turn `go test ./internal/mem` red:

| Mutation | Result |
|---|---|
| Ownership read from the stamp only | killed |
| A caller with no agent gets an owned thread | killed |
| Code runner arguments unchecked | killed |
| A `..` segment allowed | killed |
| An absolute path allowed | killed |
| A one-item list glued to its label | killed |
| Source rebuilt as `Dir(memory)/MEMORY.md` | killed |

### Review round 2 (FAIL) and its fixes

The second review (`review-round-2.md`, at `2e8777c6`, nan/deepseek-v4-flash) returned FAIL on one REAL Major. Every finding, with its disposition:

| Finding | Disposition | Test |
|---|---|---|
| Major: `verifyLines` dropped a `Verify` field's prose whenever it also held a command, against `ParseThread`'s "no line is dropped" | **Applied.** A field with words beside its commands renders them first as `As written: <field>`, then the command list; a field of commands alone renders no prose line. The goldens gained that one line, because the fixture's field joins its commands with "and" | `TestResumeKeepsVerifyProseBesideTheCommands` (both formats; red before the fix) |
| Minor: `handoff_resume` is not declared in `session-start-config.json` | **Ticketed** as #2915, with the reviewer's note that the bats count should follow the code | n/a |
| Minor: empty Slice 4 and Slice 5 evidence headings before the filled ones | **Applied**: the empty pair is removed | n/a |
| Question: the `*` matcher firing on `compact`/`clear` is unobserved live | **Kept open**: recorded here; the next real compaction in a session on a release carrying slice 4 is the observation, and a hook that does not fire gets a ticket | n/a |
| Question: `source` matched verbatim | **Applied**: `normaliseSource` trims and lowercases it | `TestSessionStartResumeNormalisesTheSource` |
| Speculative: the injected resume is uncapped | **Declined**: thread files are operator-written and the measured ones are small; MEMORY.md's budget is the bounded surface. Revisit if a resume block is ever observed crowding a compacted context | n/a |
| Minor: `proposal.md` boxes unticked, "about 350 B" vs the 450 B worst case | **Carried** to the archive PR; the contract set is not edited between rounds | n/a |

Mutations: dropping the `As written` line and dropping the lowercase both turn `go test ./internal/mem` red.

### Review round 3 (FAIL) and its fixes

The third review (`review-round-3.md`, at `4f19d07d`, nan/qwen3.8-flash) returned FAIL on three REAL Majors, two of them against the round-1 allowlist. Three rounds kept finding holes in the "verified read-only" claim because each fix defended it at a narrower point than the shell acts on. This round moves the claim to where it is decidable instead (lesson 513, amended).

| Finding | Disposition | Test |
|---|---|---|
| F-1: `go test ${IFS}-exec=/tmp/evil` rendered as verified | **Applied.** A command is read-only only if it is a listed prefix followed by plain words: characters `[A-Za-z0-9_./ -]`, no word starting with `-`, no absolute path or `..` segment. Such a string holds no shell metacharacter, so the shell splits it where `strings.Fields` does. `codeRunners`, `writesFile` and `shellControl` are gone, subsumed | `TestReadOnlyCommand`: `${IFS}-exec=`, quoted `'-exec='`, `HEAD:secrets/x`, a tab |
| F-2: `dotf doctor --fix` and `gh pr view 1 --web` rendered as read-only | **Applied** by the same rule: every flag renders unverified. Accepted cost: `go test ./... -run X`, `gh pr view 1 --json state` and `git diff main -- x.go` render unverified too; the permission gate still runs them. Getting flags back means a real shell parser as a dependency and an ADR | `TestReadOnlyCommand`: both inputs, and the formerly-true flag rows flipped |
| F-3: `As written:` kept only the field's first line, and repeated a marked command unmarked | **Applied.** The field renders once, as written, every line kept, with `(unverified)` placed right after each command that fails the check; the explanation appears once in the heading line, only when something is marked | `TestResumeKeepsVerifyProseBesideTheCommands`: one line, two lines, bullets with a prose-only bullet; the note and the markers must agree |
| Minor: `handoff_resume` undeclared in `session-start-config.json` | **Ticketed**, #2915 (open) | n/a |
| Question: live `source=compact` payload unobserved | **Kept open**, as in round 2 | n/a |
| Minor: contract-set boxes unticked, "about 350 B" | **Carried** to the archive PR | n/a |
| Speculative: uncapped resume | **Declined**, as in round 2 | n/a |

Mutations, each applied with `go vet` passing, all turn `go test ./internal/mem` red: the charset check removed, flags allowed, the field cut to its first line, the marker removed, a `..` segment allowed.

### Review round 4 (FAIL): the run's review budget is spent

The fourth review (`review-round-4.md`, at `dee3fe4f`, nan/deepseek-v4-flash) returned FAIL on two REAL Majors. F-1 is the same class rounds 1 and 3 failed on: only a backticked span is checked, so a command written as prose renders with no marker and no note. Rather than a fourth narrower patch, the shape of the "verified" promise goes to the owner on #1689 (keep it and make it total, or retire it).

| Finding | Disposition | Test |
|---|---|---|
| F-1: a prose (unbackticked) Verify command renders unmarked | **Owner decision** on #1689: mark every field with no backticked span, or drop the read-only promise and render one fixed note | UNTESTED until the shape is chosen |
| F-2: the candidate lines interpolate the thread key and `--agent` raw | **Applied.** Both are one shell word: as written when plain (`[A-Za-z0-9_./@+-]`), single-quoted otherwise | `TestMemResumeQuotesTheCandidateLine`; quoting removed from either the key or the agent turns it red |
| F-3: `proposal.md` AC boxes and the `tasks.md` Closing block unticked | **Carried** with F-1: AC2 and the slice 3 box depend on the chosen shape | n/a |
| F-4: `features.json` states are `pending` | **Declined:** the states are harness-owned; the six commands exit 0, re-run by the reviewer | n/a |
| F-5: the `context` render names the thread file's path, so AC1's "pure function of the thread file" is weaker than the code | **Carried** with F-1: the criterion wording moves with the contract set | n/a |
| F-6: promotion candidates unanswered | **Applied** above | n/a |
