---
generated: true
generated_from: 00_meta/skills/pr-review-triage/SKILL.md
generated_sha: ac110dd5b55e6992
id: pr-review-triage-skill
type: skill
status: active
created: '2026-08-08'
owner: manu
name: pr-review-triage
description: Triage an open pull request after its checks and reviewers have run —
  read the CI result, read every review comment, and give each one a disposition (apply
  / skip / defer) with a one-line reason. Triggers on /pr-review-triage, "triage the
  PR", "review the review", "what did the reviewer say", "revisa los comentarios de
  la PR", "check CI and the bot comments", and by default once a PR you opened has
  come back — from its checks and from its reviewers, whichever lands later, because
  checks finishing is not the end of the window. Never applies a change or merges
  without explicit human confirmation.
allowed-tools: [Bash, Read, Grep]
keywords: [pr review triage, triage pr, review bot comments, revisa comentarios pr,
  ci triage]
paths: [.github/workflows/**]
---
# /pr-review-triage — dispose of what the PR came back with

Opening a pull request is not the end of a change. Its checks report, its reviewers comment, and both are then routinely ignored because nothing says when to come back. This skill is that moment: read what came back, decide per item, and leave nothing floating.

It is the **Review** item of the Definition of Done (`pattern-change-lifecycle.md`) made executable, and it applies the two-exits rule from [[pattern-track-or-fix]] to a reviewer's comments rather than to your own findings.

The "by default" in the trigger list is a judgement you make, not a call something makes for you: nothing invokes this skill automatically. What binds is the `pr-stewardship` obligation to disposition what came back; this skill is how you meet it, and you are the one who decides the moment has arrived.

## Preconditions

- An open PR, and `gh` authenticated. Everything here is `gh` plus shell — no agent-specific primitive, so every harness runs it identically.
- The change is yours to act on. Triaging someone else's PR is review, not this.

## Protocol

### 1. Wait for the run to finish, once — unless the repository says not to

**Check the repository's own rule first, and let it win.** Some projects forbid
the watch loop outright and make the human the one who reports a red build (this
repo does, in `AGENTS.md`). Where such a rule exists, do not run the command
below: hand the PR over, and come back when the human or a hook tells you the
result is in. That is not a shortcut past the obligation — the `pr-stewardship`
region still requires every check and comment to be dispositioned; it only says
who watches in the meantime, and the answer there is nobody.

Absent such a rule:

```bash
gh pr checks <N> --repo <owner>/<repo> --watch --interval 30
```

Watch it once rather than polling in a loop. If a reviewer bot is still pending after the checks settle, wait for that one specific thing — a triage run against half the comments produces a disposition table you will have to redo.

### 2. Report CI honestly

```bash
gh pr view <N> --repo <owner>/<repo> --json state,mergeable,statusCheckRollup
```

Green: say so in one line and move to the comments. Red: name the failing check and read its log before theorising.

```bash
gh run view --repo <owner>/<repo> --job <job-id> --log-failed
```

A failing check is a finding about the change, not an obstacle to the triage. Two of them are worth recognising on sight:

- **A guard firing on a deliberate change.** A pinned count, an exhaustive list, a discipline gate. It is doing its job; update the guard's expectation in this PR, do not route around it.
- **A gate demanding process, not code.** Fill the artifact it asks for. Reaching for the skip label is a decision that needs a stated reason, and "the gate was inconvenient" is not one.

### 3. Give the reviewers time, then check the review actually happened

Review comments arrive **asynchronously**, from whoever is configured to produce them — a review service today, a different one tomorrow, a scanner, a human, one of your own automations. **The author does not matter and must not drive the triage**: a finding is judged by whether the claim is true, not by who signed it. Automated and human comments get the same three exits; the only thing an author's identity earns is a reply address.

**Wait a couple of minutes before reading.** Reading the instant the PR opens gets you a partial review and a disposition table you will redo. Two minutes is the working default, or let the CI watch in step 1 absorb the wait — the checks usually outlast the reviewers. Those two minutes sit *inside* the ten-minute window the `pr-stewardship` region sets, they do not compete with it: two minutes is when you first look, ten is when the obligation to keep looking expires.

**Then confirm the review ran at all.** A comment that lands seconds after the PR opens is almost never a review; it is a status notice, and it reads exactly like a pass. Measured on this repo: a reviewer posted **9 seconds** after PR creation with the body *"Review limit reached — next review available in 11 minutes"*. Quota, queue, error and skip all produce that shape.

Tell them apart by **content, not by author**: a real review names files, lines, or specific claims. A notice talks about the review itself — limits, retries, queues, waiting.

```bash
gh api repos/<owner>/<repo>/pulls/<N> --jq '.created_at'
gh api repos/<owner>/<repo>/issues/<N>/comments --jq '.[] | "\(.created_at)\t\(.user.login)\t\(.body[0:120])"'
```

Three outcomes, and only one means "no findings":

- **Reviewed, findings** → triage them (step 4).
- **Reviewed, nothing to say** → record that a review ran and found nothing.
- **Never ran** (quota, queue, error, not configured) → say so plainly, and either wait for the stated window or proceed **without** claiming review coverage you do not have.

Note the asymmetry: some checks are configured to comment, others only to set a check status. Anything that never comments is triaged in step 2, not here — an empty comment list is not evidence that nothing scanned the change.

**Skip to step 7 when there is nothing to dispose of.** Green checks and no comments — or only status notices, or only remarks with no bearing on the change — is the common case, and for the human in front of you it ends in one line: *"CI green, no review findings."* Do not manufacture findings for an empty review, do not reply to noise, and do not invent work to look thorough. Steps 4 to 6 exist for when a review actually said something.

You still record the outcome on the PR. **"Nothing to dispose of" is a disposition**, and step 7 says why it has to be written down rather than said: an unrecorded empty triage leaves the PR in the queue permanently, which is indistinguishable from nobody having looked.

### 4. Read every comment, including the resolved-looking ones

```bash
gh pr view <N> --repo <owner>/<repo> --comments
gh api repos/<owner>/<repo>/pulls/<N>/comments --jq '.[] | "\(.path):\(.line)\t\(.user.login)\t\(.body)"'
```

Both calls: the first returns issue-level review bodies, the second the inline threads, and a suggestion that only exists inline is the one most often lost.

### 5. Give every comment a disposition

Three exits, and every comment takes exactly one. A comment you neither applied nor recorded is a comment you silently overruled.

| Disposition | When | What it costs you |
|---|---|---|
| **apply** | Correct, and small enough to belong in this PR's scope | A commit on the branch, in the same voice as the change |
| **defer** | Correct, but out of this PR's scope, or it needs a decision | A ticket with the root cause and a link back to the comment |
| **skip** | Wrong, irrelevant, or right in general but not here | One line saying why. Reply on the PR only when someone is waiting on the answer — a human reviewer, or a thread the next reader will hit. Noise gets a row in your table and nothing else |

Two judgements this table does not make for you:

- **Scope.** "While I'm here" is how an atomic PR stops being reviewable. If applying a comment would change the diff's subject, defer it.
- **Correctness.** A reviewer — human or model — can be confidently wrong. Check the claim against the code before applying it; a suggestion that reads well and is wrong costs more than one you skipped.

### 6. Present the table, then act

Print one row per comment with its proposed disposition and reason, and **wait for confirmation before applying anything**. What the human is confirming is your judgement, not your typing.

```
#12 src/deploy.sh:44  apply  — the guard misses the CRLF case, real defect
#13 tests/foo.bats:8  defer  — valid, but it is a second behaviour → filed as #NNN
#14 README.md:2       skip   — suggests a convention this repo decided against in ADR-012
```

After confirmation: apply the accepted ones as commits, file the deferred ones as tickets, reply to the skipped ones with the reason. Then say what you did, with the ticket numbers.

### 7. Record the disposition on the PR — always, including the empty case

A disposition that lives only in this conversation evaporates when the session ends, and the Definition of Done's **Review** item then rests on a claim nobody can check. Post the table as a comment on the PR, under this exact heading:

```markdown
## Review triage

| Item | Disposition | Reason |
|---|---|---|
| #12 `src/deploy.sh:44` | apply | the guard misses the CRLF case, real defect |
| #13 `tests/foo.bats:8` | defer | valid, but a second behaviour → filed as #NNN |
| #14 `README.md:2` | skip | suggests a convention ADR-012 decided against |

Reviewer output dispositioned: <reviewer login>, <timestamp of their newest comment>.
```

```bash
gh pr comment <N> --repo <owner>/<repo> --body-file <file>
```

**The heading is a contract, not a formatting choice.** `dotf pr triage-queue` reads it back to decide whether a PR is still pending: a PR is pending when its newest reviewer output is newer than its newest triage record. The string is declared once, in `harness/review-attestation.json` under `triage.marker` — this skill writes it and the queue reads it. Match it exactly, at the start of a line.

**Record the empty case too.** *"CI green, no review findings"* is a disposition and it must be written down like any other — one row saying so is enough. Skipping it because there was nothing to apply leaves the PR in the queue forever, and a queue that never drains is one nobody reads. The queue re-opens by itself the moment a reviewer speaks again, so recording early costs nothing.

**Edit an existing table when re-triaging.** If a `## Review triage` comment already exists from this session, edit it (using `gh pr comment <N> --edit-last` or by comment ID) rather than posting a second identical table, avoiding duplicate comment noise on the PR.

### 8. Never

- **Never merge.** Merging is a supervised human action, and auto-merge is forbidden in every repository. This skill ends at a triaged PR, never at a merged one.
- **Never apply in bulk without reading.** Accepting a whole review because it is long is the same failure as ignoring it.
- **Never claim a review you did not get.** "No findings" and "the review never ran" are different statements, and only one of them is evidence.

## Output

One line when there was nothing to dispose of. Otherwise: the disposition table, the actions taken after confirmation, and the ticket numbers for everything deferred. If CI was red, the failing check and what you did about it.

Either way the `## Review triage` comment lands on the PR (step 7). The conversation output is for the human in front of you; the comment is what survives the session and what the queue reads.

## Pairs with

- [[pattern-track-or-fix]] — the two-exits rule this applies to someone else's findings
- [[pattern-change-lifecycle]] — the Definition of Done whose **Review** item this executes
- [[pattern-git-workflow]] — merge policy: human-reviewed, never automatic
- `verification-before-completion` — evidence before claiming the triage is done
