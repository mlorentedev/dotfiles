---
id: lesson-263
type: lesson
status: active
created: "2026-09-02"
owner: manu
tags: [lesson, github, gh-cli, measurement, guards]
---

# A population that cannot prove it is complete answers with a plausible number

## What happened

While classifying duplicated `AREA-NNN` ids for GUARD-009 (#1448), I wrote a throwaway allocator
to compute the next free number per area. It proposed:

- **`CLI-072`** — already held by #1460, filed hours earlier by a parallel session.
- **`TEST-002`**, from a computed maximum of `TEST-001`. The real maximum was `TEST-008`.

The tool written to stop duplicate ids was about to allocate a duplicate id.

## Cause

`gh api --paginate` does not return one JSON array. It **concatenates** one array per page:
`[...][...][...]`. I split them with a regex:

```python
re.findall(r'\[.*?\]\s*(?=\[|$)', raw, re.S)     # WRONG
```

Non-greedy `.*?` stops at the first `]` it finds — and issue bodies in this repo are full of them,
in markdown links and tables. So the split landed mid-page, `json.loads` raised on the fragments,
and the `except Exception: pass` beside it swallowed each failure. Some pages survived, most did
not, and the result was a well-formed dict of maxima computed over a fraction of the corpus.

Nothing errored. The number just came back low, and `CLI` max `71` is exactly as plausible as `72`.

## The rule

**A measurement that cannot demonstrate its own completeness must fail loudly, not return a
number.** State the population, and prefer a form that cannot silently lose part of it:

```bash
# One line per issue. Pagination is gh's problem, not a regex's.
gh api --paginate 'repos/<owner>/<repo>/issues?state=all&per_page=100' \
  --jq '.[] | select(.pull_request==null) | [.number,.state,.title] | @tsv'
```

`--jq` is applied per page by `gh` itself, so the output is line-oriented and concatenation is
harmless. Count the lines and say the count out loud; a population you can print is one a reviewer
can check.

Two corollaries this incident earned:

- **Never pair a parser with a bare `except: pass`.** The handler is what converted a crash into a
  wrong answer. If a chunk cannot be parsed, that is the finding.
- **Print the plan before executing it.** The allocator's output was eleven rows of
  `issue → new id`. Reading them is what caught `CLI-072`; no assertion in the script would have,
  because the script had no idea what the true maximum was. For any batch of writes, render the
  intent first and look at it.

## Why it belongs with lesson 256 and #1448

Lesson 256 says *probe the target, do not read the block's own comment*. This is the same rule
pointed at a dataset: **I read the tool's summary of the corpus instead of the corpus.** And it is
an instance of the very class #1448 exists to detect — a claim (`max = 71`) whose referent did not
support it — produced by the tooling written to serve that ticket, roughly an hour after the ticket
argued that a defect class surviving its own authors' knowledge cannot be fixed by writing it down
again.

## The class: three shapes of it were measured on one day

This is not a `gh` quirk. A parallel session hit the same defect twice the same afternoon, in
unrelated tooling, and the three together name the class better than any one of them:

| Shape | The command | What it returned | The truth |
|---|---|---|---|
| **A pipeline** | `bats tests/*.bats \| tail -15` | `exit code 0` | `tail`'s status, not bats'. The run was `BATS_EXIT=1` |
| **A filter** | `go test -run '<pattern matching nothing>'` | `ok`, exit 0 | Zero tests ran. A `features.json` command naming a deleted test passes forever |
| **A regex** | `--paginate` output split with `re.findall` | `CLI` max `= 71` | `= 72`. Pages were dropped mid-body |

**In every case the failure was success.** Not a crash, not an empty result that reads as
suspicious — a well-formed, plausible answer. Nothing downstream could tell them from a real one,
which is why all three survived until something outside the tool contradicted them.

The general rule: **when a command can return a value without doing the work, the value is not
evidence.** Ask what this invocation would print if it silently did nothing, and if the answer is
"the same thing", the check does not exist yet. Concretely — put the assertion on the producer's
exit status rather than a pipeline's (`set -o pipefail`, or capture `${PIPESTATUS[0]}`), make a
zero-match filter an error rather than a pass, and make a measurement state its population.

Instances 1 and 2 are recorded with their evidence in `specs/CLI-072-dotf-hooks-install/verification.md`.

## A second finding from the same pass

Renumbering a collision needs an **owner**, and `harness/skills/new-ticket/SKILL.md` said the
higher issue number yields. That is wrong whenever a spec folder exists, because the folder
declares its owner in frontmatter:

```
specs/CLI-065-env-persist-sweep/proposal.md    issue: mlorentedev/dotfiles#1363
```

Here #1363 is the *higher* number and the rightful owner; renumbering it by the old rule would have
orphaned an active spec — manufacturing the stale-referent defect the pass was cleaning up. The
same reversal appeared in `docs/lessons/` minutes later: two files claimed `lesson-261`, and the
later one was cited from `AGENTS.md`, `adr-028` and an open spec's acceptance criteria, while the
earlier one was cited by nothing. **The earlier one yielded**, because inbound references, not
timestamps, are what a rename actually breaks.

The rule is now: *ownership evidence first (spec frontmatter, inbound references), ordering only as
a tie-break.* Applied in `new-ticket/SKILL.md` in the same change as this lesson.
