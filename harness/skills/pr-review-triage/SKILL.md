---
id: pr-review-triage-skill
type: skill
status: active
created: "2026-08-08"
owner: manu
name: pr-review-triage
description: Triage an open pull request after its checks and reviewers have run — read the CI result, read every review comment, and give each one a disposition (apply / skip / defer) with a one-line reason. Triggers on /pr-review-triage, "triage the PR", "review the review", "what did the reviewer say", "revisa los comentarios de la PR", "check CI and the bot comments", and by default once a PR you opened has finished its checks. Never applies a change or merges without explicit human confirmation.
allowed-tools: [Bash, Read, Grep]
---

# /pr-review-triage — dispose of what the PR came back with

Opening a pull request is not the end of a change. Its checks report, its reviewers comment, and both are then routinely ignored because nothing says when to come back. This skill is that moment: read what came back, decide per item, and leave nothing floating.

It is the **Review** item of the Definition of Done (`pattern-change-lifecycle.md`) made executable, and it applies the two-exits rule from [[pattern-track-or-fix]] to a reviewer's comments rather than to your own findings.

## Preconditions

- An open PR, and `gh` authenticated. Everything here is `gh` plus shell — no agent-specific primitive, so every harness runs it identically.
- The change is yours to act on. Triaging someone else's PR is review, not this.

## Protocol

### 1. Wait for the run to finish, once

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

### 3. Read every comment, including the resolved-looking ones

```bash
gh pr view <N> --repo <owner>/<repo> --comments
gh api repos/<owner>/<repo>/pulls/<N>/comments --jq '.[] | "\(.path):\(.line)\t\(.user.login)\t\(.body)"'
```

Both calls: the first returns issue-level review bodies, the second the inline threads, and a suggestion that only exists inline is the one most often lost.

### 4. Give every comment a disposition

Three exits, and every comment takes exactly one. A comment you neither applied nor recorded is a comment you silently overruled.

| Disposition | When | What it costs you |
|---|---|---|
| **apply** | Correct, and small enough to belong in this PR's scope | A commit on the branch, in the same voice as the change |
| **defer** | Correct, but out of this PR's scope, or it needs a decision | A ticket with the root cause and a link back to the comment |
| **skip** | Wrong, or right in general but not here | One line saying why, posted as a reply — not left in your head |

Two judgements this table does not make for you:

- **Scope.** "While I'm here" is how an atomic PR stops being reviewable. If applying a comment would change the diff's subject, defer it.
- **Correctness.** A reviewer — human or model — can be confidently wrong. Check the claim against the code before applying it; a suggestion that reads well and is wrong costs more than one you skipped.

### 5. Present the table, then act

Print one row per comment with its proposed disposition and reason, and **wait for confirmation before applying anything**. What the human is confirming is your judgement, not your typing.

```
#12 src/deploy.sh:44  apply  — the guard misses the CRLF case, real defect
#13 tests/foo.bats:8  defer  — valid, but it is a second behaviour → filed as #NNN
#14 README.md:2       skip   — suggests a convention this repo decided against in ADR-012
```

After confirmation: apply the accepted ones as commits, file the deferred ones as tickets, reply to the skipped ones with the reason. Then say what you did, with the ticket numbers.

### 6. Never

- **Never merge.** Merging is a supervised human action, and auto-merge is forbidden in every repository. This skill ends at a triaged PR, never at a merged one.
- **Never apply in bulk without reading.** Accepting a whole review because it is long is the same failure as ignoring it.
- **Never close the loop silently.** If you skipped a comment, the reviewer's next reader should be able to see why.

## Output

A disposition table, the actions taken after confirmation, and the ticket numbers for everything deferred. If CI was red, the failing check and what you did about it.

## Pairs with

- [[pattern-track-or-fix]] — the two-exits rule this applies to someone else's findings
- [[pattern-change-lifecycle]] — the Definition of Done whose **Review** item this executes
- [[pattern-git-workflow]] — merge policy: human-reviewed, never automatic
- `verification-before-completion` — evidence before claiming the triage is done
