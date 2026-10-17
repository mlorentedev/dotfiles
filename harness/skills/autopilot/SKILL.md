---
generated: true
generated_from: 00_meta/skills/autopilot/SKILL.md
generated_sha: f77de9668ecb8354
id: autopilot-skill
type: skill
status: active
created: '2026-10-14'
owner: manu
name: autopilot
description: Use when the operator announces an absence (sleeping, working, eating, stepping away, workouts, AFK) and leaves sessions in pilot mode, or when explicitly switching an agent to unattended backlog execution. Covers autonomous ticket checkout, advisor escalation, deterministic worktree isolation, in-flight documentation, and pre-authorized PR landing under strict conditions. Not for interactive pairing sessions.
keywords: [autopilot, pilot-mode, pilot mode, modo piloto, unattended, afk mode, autonomous loop, unattended session, sesiones en modo piloto, ausentarme, dormir, comer, trabajar, tiempo fuera, coge otro ticket, I'll be away, keep working while I'm out, work the board]
paths: []
requires: [using-git-worktrees, handoff, pr-review-triage, test-driven-development]
---

# Autopilot — Autonomous Unattended Developer Loop

> The operator announces an absence (sleeping, working, eating, stepping away, workouts — any bounded time away). Sessions keep working the backlog autonomously: check out tickets, implement with the repo's engineering standards, document in-session, land PRs under the standing grant, and leave a clean state plus a report for the operator's return.
>
> **Core Invariant:** Autonomy is granted by scope and bounded by deterministic machinery (Bitácora issue claims, physical Git worktrees, green CI on head SHA, branch protection, the merge-grant file, and circuit breakers) — never by asking the model to "be careful". The operator's trust is spent once at activation; the protocol must not spend more than was granted.

---

## The One-Line Contract

When the operator says (any variant): *"Me voy a ausentar / a dormir / a trabajar / a comer, dejo las sesiones en modo piloto. En caso de duda consultad al advisor y si acabas la tarea coge otro ticket de la misma naturaleza o del mismo epic y ve documentando, haciendo handoff, lessons, adr y demas... Tienes permiso para mergear PRs que abras con 4 condiciones: te hayas coordinado con las otras sesiones para no entorpeceros, CI está verde, no hay conflictos, y comentarios de reviewers están atendidos."*

...this skill converts that announcement into a safe, bounded, unattended work loop. The operator's sentence IS the grant; nothing in it may be widened.

---

## 1. Activation & Cold-Start Ritual (Run Once at Session Start)

1. **Tooling & CLI Health:**
   ```bash
   dotf doctor --quick
   gh auth status
   ```
2. **Clean Base Tree & Cold-Start Test:**
   ```bash
   git status --short
   git pull --rebase
   ```
   *Cold-start verification:* Run the repo's test suite or verify script on the base tree before making changes. If tests fail before you write code, record the pre-existing defect in your session journal — never mistake an inherited failure for your own regression.
3. **Check the PR Triage Queue:**
   ```bash
   dotf pr triage-queue
   ```
   If pending review items exist on existing PRs, disposing them takes precedence over picking new work.
4. **Write the Arming Envelope — the loop contract (`dotfiles#239`).** Prose budgets do not survive a context reset; the envelope does. Write one file, commit it, and run against it — the objective, the in-scope paths, the allowed commands, the budget and the stop conditions, in one machine-readable place:

   ```yaml
   # .agent/unattended.yml — written at arming, committed on the run's branch
   version: 1
   run_id: ""                       # derived on first entry, written back; never regenerated
   armed_by: manu
   armed_at: 2026-10-14T22:05:00Z   # RFC3339
   expires_at: 2026-10-15T06:00:00Z # RFC3339 — required, and at most 12h after armed_at
   objective: ""                    # ONE sentence: the single outcome this run must produce
   scope:
     write_paths: ["docs/**", "specs/**"]                        # non-empty; globs the run may write
     frozen_paths: [".github/**", "harness/**", "**/AGENTS.md"] # never written
     max_files: 8
     max_loc: 300                   # EXECUTABLE lines per PR (pattern-git-workflow §10)
   allowed_commands: [dotf, gh, git, make]   # anything else needs the advisor (R2) or a park (R3)
   queue:
     source: bitacora
     filter: '<label/epic this run works>'   # the scope decision the operator made before leaving
     max_items: 5
   budget:
     max_iterations: 8
     max_minutes_per_iteration: 25
     max_consecutive_failures: 2
   authority:
     merge: standing-grant          # never | standing-grant
     push: true
     open_pr: true
     advisor_tier: top
   ```

   Three invariants, each checkable without reading prose:
   - `expires_at > armed_at`, and `expires_at − armed_at ≤ 12h` — a run that outlives a working day has no ceiling in practice.
   - `scope.write_paths` ∩ `scope.frozen_paths` = ∅.
   - **The run reads the envelope from the COMMITTED copy, never the working tree**, and refuses if the two differ:
     ```bash
     git show "$BASE:.agent/unattended.yml" > "$ENVELOPE"   # the run's ONLY authority
     git diff --exit-code "$BASE" -- .agent/unattended.yml || { echo "ENVELOPE MUTATED — halt"; exit 1; }
     ```
     *Why:* a fence that reads a file the run can rewrite is not a fence. Without this, a run widens `write_paths` locally, does the wide work, commits only the work — the widened file never enters the diff — and the change merges green. That is the single cheapest change that closes the largest hole.
   - **A PR whose diff touches `.agent/unattended.yml` is never merged** — the same rule as the grant: a PR that changes the grant is not merged under it.

   A missing or malformed envelope parks the run (`envelope-invalid`) — never a default for an authority field. Widening the envelope, the filter or the budget is the one move that voids the grant.

   **Host precondition — the failure everyone forgets.** The dominant real case for "I'm going to sleep" is a laptop whose lid closes. Arm with a wake-lock, or state in the report that the window was not covered:
   ```bash
   caffeinate -i -w $$ &                      # macOS
   systemd-inhibit --what=idle:sleep -- <cmd> # Linux
   powercfg /change standby-timeout-ac 0      # Windows
   ```

   Defaults when the operator gave no numbers: 3–5 tickets, 30 min per ticket, and `merge: standing-grant` only where `.github/merge-grant.yml` exists (otherwise `never`). **Ask degrades to deny:** in unattended mode any ambiguity, missing input or interactive prompt degrades to **Deny / Escalate / Parked**, never to silent approval ("idle is not authorization").

---

## 2. Ticket Selection & Eligibility (Decide BEFORE Claiming)

A ticket is pilot-eligible only if ALL hold:

- **The 3 a.m. Test:** *If this runs while the operator is away and does exactly what the ticket literally says, is the operator comfortable with the result unreviewed?* If no, leave an issue proposal or comment instead of executing, and move on.
- **A Plan Already Exists:** Spec, task list, or mechanical nature (bug fix <20 lines with obvious cause, tests, docs, small refactor). Never invent an architectural plan unattended.
- **Same Epic / Nature Continuity — Board Pre-Flight: two queries, always.** One search is not a detector, and a negative is established only by the scan:
  ```bash
  # 1. Text search. --limit is not optional: the CLI default is 30, so a match at 31 is invisible.
  gh issue list --state all --limit 500 --search "<the run's topic or epic>"
  # 2. The board itself, by field — this is the query the run actually picks FROM.
  gh project item-list 1 --owner mlorentedev --format json --limit 200 \
  | jq -c --arg repo "https://github.com/mlorentedev/<repo>" '
      [ .items[] | select(.status == "Backlog")
        | select(.repository == $repo)                                   # repository is a FULL URL: there is no .repo field
        | select(.type == "spec" or .type == "bug" or .type == "chore") ]
      | sort_by([(.priority // "P3"), .content.number])                  # P0<P1<P2<P3, then issue number asc
      | .[0:3]'                                                          # bounded look-ahead; never scan the whole board
  ```
  *Verified live 2026-10-14:* `gh project item-list` exposes `status`, `priority`, `type`, `repository` (full URL) and `content.{number,title,type,url}`. A query written against `.repo` selects nothing, silently, in every repository — the run then ends `queue-empty` at minute zero and reports it honestly.
- **Three vetoes, in order; take the first survivor:** (1) **already attempted** — `git log --oneline "$BASE..$BRANCH" --grep="Item: #<N>"` is non-empty; (2) **out of scope** — the item's paths are not a subset of `scope.write_paths`, or its diff exceeds `scope.max_loc` → record `parked(scope-violation)` so the operator sees what was out of bounds; (3) **WIP limit** — `dotf spec init` refuses (ADR-043); a refusal is a stop, not a retry.
- **When the eligible set is empty, the run ENDS — it does not widen.** Not another repo, not another status, not past `Priority`; and never `gh issue create` / `dotf spec init` to manufacture work *for the queue*: a run rewarded for motion produces motion. This does not forbid filing a defect or follow-up **found while working an item** (detect→ticket, `new-ticket` non-interactive mode): that is debt recorded, not work invented, and the run must file it rather than drop it. It ends with `reason: queue-empty` and says so.
- **No Path Contention:** Check open PRs and branches (`gh pr list --state open`, `git branch -r`) to ensure no peer session is editing the same files/subsystems.
- **Atomic Claim:** Self-assign the ticket to lease it:
  ```bash
  gh issue edit "$ISSUE_NUM" --add-assignee "@me"
  gh issue comment "$ISSUE_NUM" --body "Picked up in autonomous pilot mode. Branch: feat/<slug>"
  ```
  *Note:* Self-assignment updates the Bitácora status machine to `In Progress`.
- **One Ticket at a Time:** Never multitask across multiple tickets in parallel within the same session.

---

## 3. The Autonomous Execution Loop (Per Ticket)

### Step 1: Worktree Isolation & Idempotent Checkout

Each ticket MUST run in an isolated sibling worktree per `using-git-worktrees`.
Check if the worktree already exists (crash recovery / resumability):

```bash
SLUG="issue-${ISSUE_NUM}-<short-topic>"
BRANCH="feat/${SLUG}"

# NEVER derive the worktree path. It is not a convention you can compute:
#   - `git rev-parse --show-toplevel` returns the WORKTREE when you are already inside one,
#     so `../$(basename …)-wt-<slug>` resolves to nonsense on a resume;
#   - the on-disk layout is per-machine and changing (dotfiles#2527 moves it under WORKTREE_ROOT).
# Ask git, which is versioned and stable, and re-read after creating:
resolve_wt() {
  git worktree list --porcelain \
  | awk -v b="refs/heads/${BRANCH}" '/^worktree /{p=substr($0,10)} $0=="branch "b{print p}'
}
WT_DIR="$(resolve_wt)"
if [ -z "$WT_DIR" ]; then
  dotf worktree add "$SLUG" -b "$BRANCH" --issue "$ISSUE_NUM"
  WT_DIR="$(resolve_wt)"
fi
[ -n "$WT_DIR" ] || { echo "worktree resolution failed -> park(lease-lost)"; exit 1; }
cd "$WT_DIR" || exit 1

# Confirm the lease before working (not to compute the path — only to check state):
dotf worktree list --json | jq -r --arg b "$BRANCH" '.[] | select(.branch == $b) | .state'
```

### Step 2: Idempotent Implementation & Search-Before-Create

- **Search Before Create:** Before writing a helper, script, or module, search the codebase (e.g. `rg "<pattern>"`) to verify it doesn't already exist. Duplicate code is a leading failure mode of autonomous loops.
- **TDD & Small Diffs:** Write the failing test first, make it pass, adhere to the ~300 LOC cap.
- **Zero Manual Operations:** Never perform ad-hoc manual mutations on remote systems or servers. All configuration must be reproducible, idempotent IaC (`changed=0` on re-run).
- **In-Flight Documentation (Zero Debt):**
  - Record lessons in `docs/lessons/` (run `dotf lessons fmt` or update `docs/lessons/_index.md`).
  - Record ADRs in `docs/adr/` if making an architectural or non-obvious design decision.

### Step 3: Machine-Checkable Verification Gate

Run the deterministic test/lint suite. The only accepted proof of progress is command output, never self-report:

```bash
# Language/repo-specific verify command:
make verify # or npm test, go test ./..., cargo test, pytest
```

If verification passes, proceed. If verification fails, engage the Fix-Loop Breaker (§4.2).

### Step 4: PR Creation

Push the branch and open a draft PR:

```bash
git push -u origin "$BRANCH"

PR_URL=$(gh pr create --draft --title "feat(<scope>): <short-description>" \
  --body "Closes #${ISSUE_NUM}

## Summary
<description>

## Verification
Evidence:
\`\`\`
$(make verify 2>&1 | tail -15)
\`\`\`

## Knowledge
- Lesson: docs/lessons/lesson-<num>-<slug>.md
- ADR: none: <reason>
- Runbook: none: <reason>")

PR_NUM=$(echo "$PR_URL" | grep -oE '[0-9]+$')
echo "Opened PR #${PR_NUM} at ${PR_URL}"
```

*Note:* Opening as a draft prevents reviewer noise while checks run. Mark ready when checks stabilize: `gh pr ready "$PR_NUM"`.

---

## 4. Advisor Escalation & Circuit Breakers

### 4.1 When to Escalate to the Advisor

Escalate immediately upon:
- Unresolvable architectural forks or ambiguity.
- Uncovered security/secrets boundaries.
- Repeated failures tripping the 3x breaker.

**The advisor invocation (agent-agnostic):**

```bash
# `auto` ships the persona RECORD with the task (mandate, method, boundaries).
# `run --role` only LABELS the dispatch: the role reaches nothing but a log line.
dotf agent auto --task "Decide <the fork> for #${ISSUE_NUM}. Options: <A> / <B>. Evidence: <logs>" \
                --tier top --timeout 5m
```

- **The top tier must not degrade.** `escalated`, `chain_exhausted` and `denied` do **not** fall back to a mid model (ADR-032 §4): they park the work (§4.2). Silent degradation is the "mysterious quality drop" the tier policy exists to prevent. For the same reason a typed `task_failed` is never retried on another model — only `pool_unavailable` advances the chain, and that happens inside the dispatcher, not by your choice.
- If your harness offers its own second-opinion primitive (a native advisor or subagent tool), use it *in addition* — but the CLI above is the portable floor and the one this skill's evidence covers. Naming a vendor primitive is a coupling: say which harness it exists on rather than implying every agent has it.

### 4.2 The 3x Fix-Loop Breaker & Parking Tickets

- **Max 3 Fix Iterations:** If the same test or linter failure recurs across 3 rounds, **STOP grinding**.
- **Park the Ticket Cleanly:**
  1. One comment, in `pattern-agent-oriented-errors` form — ERROR / WHY / FIX — carrying **two** options and their consequences. A park is a *question*, not a status:
     ```bash
     gh issue comment "$ISSUE_NUM" --body "ERROR: <symptom>. WHY: <root cause>. FIX: (A) <option + consequence> / (B) <option + consequence>."
     ```
  2. Add the `blocked` label, then set Bitácora Status `Blocked` with the documented helper — resolve the field/option IDs **at runtime** and never paste a third copy of them. SSOT: `00_meta/runbooks/bitacora-project-setup.md` §5 (Status option `Blocked` = `7705a647`); the `new-ticket` skill carries the runtime-resolution snippet.
     ```bash
     gh issue edit "$ISSUE_NUM" --add-label blocked
     ```
     **There is no `dotf ticket` command.** An earlier draft of this skill called `dotf ticket update --status Blocked`, which fails with `unknown command "ticket"` — the park path silently did nothing. Do not reintroduce it.
  3. `parked` is terminal for the **item** and non-terminal for the **run** — unless the item *is* the run's objective, in which case the run ends (`reason: goal-blocked`). One hard item must not kill a night's work; an unanswerable item must never be guessed. Then record the ruling in the session journal and pick the next eligible ticket.

---

## 5. The Standing Merge Grant (4 + 2 Conditions)

An agent may merge a PR **it opened** ONLY when all six conditions hold on the **exact head commit** produced this turn:

| # | Condition | Verification Evidence Required This Turn |
|---|---|---|
| 1 | **Coordinated with Peers** | `git fetch --prune` + checking open PRs/branches for path overlap. No concurrent sessions touching identical files. |
| 2 | **CI is Green** | Every required check passes or is skipped on the **current head SHA** (`gh pr checks "$PR_NUM"`). An ancestor's green check is the *stale-green fallacy*. |
| 3 | **No Conflicts** | Merge state is `CLEAN` against current `main`/`master`. Rebase if behind. |
| 4 | **Reviewers Triaged** | All reviewer comments disposed under a `## Review triage` PR comment (applied, ticketed, or declined with documented reason). |
| 5 | **Grant in Force** | The operator authorized pilot mode AND the repository carries `.github/merge-grant.yml` on its base branch. |
| 6 | **Scope Unchanged** | Head SHA matches the reviewed commit. No unreviewed commits injected during conflict resolution. |

### Landing Execution:

```bash
# 1. Verify THIS PR's queue is clear. Scoped on purpose: the repo-wide form exits 1
#    on any other session's untriaged PR, which blocks a clean land (dotfiles#2549).
#    A PR that is not open is refused, never reported clear.
dotf pr triage-queue --pr "$PR_NUM"

# 2. Land PR via deterministic CLI
dotf pr land "$PR_NUM"
```

The repo-wide `dotf pr triage-queue` (§1 step 3) stays the session-start brief: it answers "what is waiting anywhere", not "may this PR land".

### The Always-Human Floor (Exclusions):

**NEVER** merge or auto-land PRs that touch:
1. `.github/merge-grant.yml` (agents cannot widen their own authority).
2. `release-please` or version bump / tagging workflows.
3. CI workflows (`.github/workflows/`), security policies, or secret management configs.
4. CODEOWNERS paths requiring human approval.

*Rule:* If a PR touches any of these, leave it open for human review.

---

## 6. Exit Protocol & Pilot Report

When the operator returns, the time window expires, the backlog empties, or a stop condition triggers:

1. **Clean Worktree Hygiene:**
   ```bash
   cd "$REPO_ROOT"
   dotf worktree done "$SLUG"
   ```
   *(Never delete your own launch worktree if running inside one).*
2. **Multi-Session Memory Handoff — run the `handoff` skill, do not restate it.**
   `MEMORY.md` does **not** live in the repo root: it lives in the vault at
   `$VAULT_PATH/10_projects/<repo>/memory/MEMORY.md` (HARNESS-066). Resolve `$VAULT_PATH` in order —
   env var → `dotf env path VAULT_PATH` → `~/.config/dotfiles/machine.json` → fail closed:
   ```bash
   MEMORY_MD="$(dotf env path VAULT_PATH)/10_projects/<repo>/memory/MEMORY.md"
   dotf mem thread                        # which thread am I: wt-<slug>, or main@<host>
   printf '%s' "$BODY" | dotf mem handoff-write --memory "$MEMORY_MD" --agent <agent>
   ```
   The full checklist (journal, board reconciliation, artifact summary, one next action) is
   `00_meta/skills/handoff/SKILL.md`. `--memory "$REPO_ROOT/MEMORY.md"` — which an earlier draft used —
   would write the handoff nowhere near where the next session reads it.

3. **Write the durable disarm report** — the operator reads *this*, not the transcript:
   `.agent/runs/<run-id>/report.md` on the run branch, ordered by **what cannot wait**, not by chronology.
   1. **Verdict line** — `run <id> — ended <reason> after N iterations / Th — D done, P parked, M merged, U unmerged`,
      and the one question it answers: *what is waiting on a human, and is anything in it irreversible?*
   2. **Waiting on you — irreversible first:** anything landed or published, each PR awaiting a merge
      decision, each park with its two options. Irreversibility is the only thing that cannot wait, so a
      skimming operator still sees the part that matters.
   3. **Parked, with the question restated** — reason, the two options, the rung reached, and the cheapest
      thing that would unblock it. **If empty, say so in one line:** absence must be red, never silent.
   4. **Done but unmerged** — needs no thought from the operator.
   5. **Evidence index** — per item: the verification command, its exit code, and a pointer to the full
      output. Never a paraphrase, never a summary of a summary.
   6. **Budget spent** — iterations, wall clock, items. Say plainly that **no dollar figure exists**
      unless the provider exposes one (`nanquota` measures pool quota, not this run): an invented
      precision is a fabricated fact wearing an authoritative shape.
   7. **Host-suspend line** — whether the window was genuinely covered, or the machine slept through it.
      Without this line, "nothing happened" and "the run never started" are indistinguishable.
   8. **The one next action** (the `/catchup` Rule of One).

4. **Report in conversation** (a summary of the artifact, never a substitute for it):
   ```markdown
   ### Pilot Mode Session Report — [absence window]
   - **Done:** #123, #124 · **Landed:** #250 (`a1b2c3d`)
   - **Waiting on you:** #251 (PR open) · #125 parked — *question: A or B?*
   - **Knowledge:** lesson `docs/lessons/lesson-438-...md` · ADR none: no structural invariant
   - **Window:** covered / NOT covered (host suspended)
   - **Next action:** review #251
   ```

---

## 7. The Stop List (What Stops You — and ONLY This)

Each row names the primitive that enforces it **today**. Where none exists the row says so, because a
prohibition described as enforced while unenforced is worse than no prohibition: it occupies the slot a
real check would go in, and spends the attention that would have written one.

| Forbidden | Enforced by |
|---|---|
| Committing to or pushing the default branch (`main`/`master`/`release/*`) | **deterministic**: branch protection (`dotf forge protection`); the envelope's `branch` is checked at preflight |
| `gh pr merge --auto`, or enabling the repo's auto-merge setting | **deterministic**: `allow_auto_merge=false` + the `no-auto-merge` enforced record |
| Merging a PR that touches `.agent/unattended.yml` | **deterministic** once the check lands — `dotf pr land` does not read the envelope yet, so today this is discipline plus the diff check in §1.4 |
| Printing, logging, echoing or committing a secret; `env`/`printenv` under `dotf secrets` | **deterministic**: `dotf secrets show` refuses plaintext in agent sessions; `redactWriter`; `secrets-never-in-output` |
| Deleting anything; `git reset --hard`; `git stash`; history rewrite; `git push --force` | **NOT ENFORCED — discipline only.** Archive-only is a convention, not a check |
| Reading credential stores (`~/.ssh`, `~/.aws`, any `.env`, `*credential*`) | **NOT ENFORCED — discipline only.** The operator must decide whether a real boundary is worth building |
| Publishing: `docker push`, `npm publish`, deploy CLIs, any external message/mail API | **NOT ENFORCED — discipline only**; `git revert` cannot un-send. The run simply holds no such credentials |
| Starting a new spec at the WIP limit | **deterministic**: `dotf spec init` refuses (ADR-043); `--over-wip-limit` needs a written reason |
| Editing CI workflows, `harness/**`, or the envelope's authority fields | **deterministic** on the diff via `frozen_paths`; **not** prevented at write time |
| Running a second unattended run in the same checkout or branch | **deterministic**: the declared `worktree`/`branch` must equal the actual one, else halt |
| Removing any worktree (especially the launch worktree) or `dotf worktree sweep` mid-run | **deterministic**: `dotf worktree done` refuses the current worktree; `sweep` needs a merged PR + an expired lease |
| Widening the filter, the scope or the budget to keep working | **NOT ENFORCED — discipline only**, and it voids the grant |

**Run-level stops** (nothing else is picked; the disarm report is written): `expired` · `budget-iterations` ·
`budget-items` · `stalled` (two iterations changing neither the ledger nor the envelope) · `queue-empty` ·
`goal-blocked` · `lease-lost` · `envelope-invalid` · `preflight-failed` · `advisor-unavailable`.

**A check that cannot run is `pending`, never `pass`.** If `dotf pr land` is absent, unauthenticated,
rate-limited, or returns something you cannot interpret: post the triage comment with a
`merge withheld: <check> could not run` row and leave the PR open. An unrunnable check degrades the
verdict to pending — absence must be red.

---

## 8. Rationalization Table

| Excuse | Reality |
|---|---|
| "CI was green an hour ago" | Stale-green fallacy. Re-check the current head SHA. |
| "Reviewer comments are nitpicks" | Every comment must be applied, ticketed, or declined with documented rationale. |
| "It's a tiny fix, I don't need a worktree" | Worktrees prevent uncommitted file collisions between sessions. Always use a worktree. |
| "I'll document the lesson when the operator returns" | In-flight documentation is non-negotiable. Hygiene happens in-session. |
| "The test failed 4 times, but this next tweak will work" | Tripped breaker. Past 3 rounds, failure is structural. Park it. |
| "No assignees on the issue, so I don't need to assign myself" | Assignment is your lease claim for peer sessions. Always assign before working. |
| "I can merge this quick PR modifying CI" | The always-human floor. CI workflows and merge grants are never auto-merged. |
| "The envelope is mine, so I can widen it for this one item" | A privilege the subject can widen is not a privilege. The fence reads the *committed* copy, and a mutated envelope halts the run. |
| "I'll arm without a wake-lock, it'll be fine" | The dominant absence is a closing laptop lid. No lock, no covered window — and the report must say the window was not covered. |
| "Nothing eligible in the queue, so I'll file the ticket I think we need" | The empty set ends the run. A loop rewarded for motion produces motion. |
| "It's my own PR, so I'll triage my own reviewer findings and land it" | Permitted under the grant, and it must be **disclosed**: the triage comment states it was authored by the run. Disclosure is not independence — say which one you are. |
| "I'll park it by moving the board card" | `dotf ticket` does not exist. Park with a comment carrying two options, the `blocked` label, and the documented field helper. |

---

## 9. Red Flags — STOP Immediately

- You are about to merge with any of the 6 conditions unevidenced this turn.
- You are editing `.github/merge-grant.yml`, CI workflows, or release automation.
- You are in round 4 of a fix loop or widening the diff to force a pass.
- You opened a second ticket while the first is unfinished.
- You ran a command without an isolated git worktree.
- You cannot produce the command output that proves your completion claim.
- You widened the envelope, the queue filter or the budget "just for this one item".
- You are about to report a run you cannot distinguish from one that never started (no ledger, no suspend line).
