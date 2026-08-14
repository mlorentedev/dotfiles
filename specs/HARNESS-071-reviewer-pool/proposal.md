---
id: "HARNESS-071-reviewer-pool"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-13"
issue: "mlorentedev/dotfiles#955"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-071-reviewer-pool

## Why

<!-- from issue #955 -->

Adversarial review is the red-team gate between an implemented spec and
`dotf spec archive`. Its entire value is **independence** — the skill says so and
forbids single-agent self-review outright. Today that independence rests on
convention: whoever launches the review picks the model, and nothing downstream
checks what they picked.

The convention failed twice in one session, on BUG-074:

| Round | Reviewer | Verdict | Majors |
|---|---|---|---|
| 1 | `claude-opus-5` | FAIL | 3 REAL |
| 2 | `claude-opus-5` | FAIL | 3 REAL |
| 3 | `nan/deepseek-v4-flash` | PASS | 0 (+4 Minors → #956) |

Rounds 1 and 2 were reviewed by the same model family that wrote the change.
Not by decision — by path of least resistance. Both happened to be rigorous, so
no harm resulted; that is exactly what makes it worth fixing now rather than
after a cheap PASS. The gate that stops a rubber stamp cannot itself be a habit.

`cli/internal/spec/review.go` already parses `reviewer:` and already has a
refusal point (`checkReviewGate`, CLI-034). It inspects verdict, spec-id match
and staleness. The reviewer identity is recorded and then ignored — a fact
sitting one field away from the check that would use it.

## What

Three layers. Only one of them enforces, and saying which is the point.

1. **Pool as data** — `harness/reviewer-pool.json`, an ordered array of entries.
   The first entry is the launcher's primary; the whole array is the gate's
   allow-list.

   | Field | Consumed by | Meaning |
   |---|---|---|
   | `id` | **gate** | The canonical string a reviewer records in `review.md`'s `reviewer:`, matched exactly. This is the only field that is a contract. |
   | `runner` | launcher | Which CLI invokes it (`pi`, `agy`). |
   | `provider` | launcher | Passed as `--provider`; required by `pi`, absent for `agy`, which has no such flag. |
   | `model` | launcher | Passed as `--model`. |
   | `role` | humans | `primary` / `fallback`, informational — order is what actually decides. |
   | `why` | humans | Rationale; deliberately not parsed, so editing it can never break either consumer. |

   `loadReviewerPool` derives the gate's id list from these entries, so the gate
   and the launcher can never disagree about what the pool says.
2. **Gate** — `checkReviewGate` additionally refuses a `review.md` whose
   `reviewer:` is not in the pool. This is the layer that makes the rule true
   when nobody is watching.
3. **Launcher** — `dotf spec review <spec-id>` resolves the pinned reviewer from
   the pool and runs it, instead of leaving the choice to whoever is at the
   keyboard.

The launcher does not execute a reviewer; it *selects a runner*. That separation
is the whole architecture: `pi`, `agy`, `opencode`, tmux and Orca are local,
personal and replaceable, while the pool, the gate, the entry point and the
artifacts are what survive across machines, teammates and agents.

### Why the runner cannot be the guarantee

Two findings from building this, each of which would have made a runner-based
design quietly wrong:

- **`agy` serves Anthropic models too** — `agy models` lists
  `claude-opus-4-6-thinking` and `claude-sonnet-4-6` beside the Gemini family.
  "Run the review with agy" therefore guarantees nothing about provider
  diversity.
- **BUG-074's round 3 was pinned by coincidence, not by configuration.** It was
  invoked as bare `pi -p`, which resolved to `nan/deepseek-v4-flash` only
  because `~/.pi/agent/settings.json` on this machine happens to say so. That
  file is per-machine, unversioned state; `pi`'s own default provider is
  `google`. The verdict is still valid — `review.md`'s frontmatter records what
  actually ran — but the *pinning mechanism* has never been exercised. The
  launcher must pass `--provider`/`--model` explicitly.

### Gate semantics

Decided explicitly, because these are the design rather than details:

- **Pool file absent → skip the check.** `dotf` runs in repos that have no pool,
  and requiring one everywhere would break them. Deleting the pool is a visible
  diff, which is the same auditable-escape philosophy as `review: waived`.
- **Pool present but malformed → refuse.** A gate fails toward refusing
  (`archive.go`'s own stated rule).
- **Exact string match, trimmed.** The refusal prints both the found `reviewer:`
  and the pool, so a mismatch is diagnosable in one read.
- **The launcher tells the reviewer what to write.** Reviewer ids are
  self-reported, so the prompt names the canonical id to record. A hand-run
  review that guesses wrong gets a refusal naming the pool, not a silent pass.

## Out of scope

- **Running the review in CI** (`opencode github run` exists for it). Deferred
  deliberately: encoding the policy inside a GitHub workflow re-attaches it to a
  runner, which is the coupling this spec removes. Revisit once the contract is
  stable — and note the repo is public, so a provider key in Actions secrets is
  its own decision, not a detail.
- **The full H-044 `model-map.json` / `capability-map.json` fan-out.** This is
  the load-bearing slice of it, not a replacement. A dedicated file avoids
  squatting on H-044's future schema, which maps a neutral tier
  (`top|mid|low`) to a provider *per agent* — a different shape from this
  allow-list.
- **Orca integration.** Its CLI is unusable on this machine: all four resolution
  paths from the `orca-cli` skill fail (`ORCA_CLI_COMMAND` unset,
  `ORCA_DEV_REPO_ROOT` unset, `orca-ide` and the shim both exec an AppImage that
  rejects `--no-sandbox`; scrubbing the inherited AppImage env does not help).
  Tracked separately. Observability is met with tmux, which needs nothing from
  Orca and which Orca can host anyway.
- Fixing the reviewer's own findings on BUG-074 (#956).

## Risks / open questions

- **`reviewer:` is self-reported.** The gate defends against habit and accident,
  not against an agent that lies about its identity. That is the same trust
  level as the rest of `review.md` (verdict included), so it is not a regression
  — but the spec must say it plainly rather than imply a stronger guarantee.
- **A weak reviewer is worse than no gate.** It converts the gate into a green
  checkmark. This is why the pool holds reasoning-class models and deliberately
  excludes the latency-optimised defaults (`nan/qwen3.6`, `nan/gemma4`) that the
  daily wrappers use. Evidence it matters: round 3 scored an all-A rubric where
  round 2 scored B/C, and missed a documentation inconsistency the implementer
  had deliberately left in place — a real reviewer, with less edge.
- **The fallback must be exercised, not merely configured.** A fallback never
  observed working is decoration — this repo's own #898 lesson. NaN's evidence
  is BUG-074 round 3; Gemini has none yet, so producing one is an acceptance
  criterion, not a follow-up.
- **`agy --print-timeout` defaults to 5m.** Round 3 took roughly 25 minutes of
  wall clock. The Gemini arm dies on default settings, so the launcher must
  raise it — a concrete instance of "configured is not exercised".
- **Windows.** tmux is Linux-only and `dotf` is cross-platform. The degrade must
  be explicit (no tmux → run in the foreground and say so), not implicit; the
  repo enforces Windows parity.
- **Already-archived specs are signed by models the pool forbids, and that is
  safe — verified, not assumed.** Two of the three archived `review.md` files
  are signed `claude-sonnet-5`. They do not need a grandfather clause because
  `checkReviewGate` has exactly one caller (`archive.go`, inside `Archive()`),
  and nothing scans `specs/archive/` for reviews: the gate runs at archive time
  only, so a spec that is already archived is never re-evaluated. If a future
  change adds a sweep over archived reviews — a doctor drift check, a CI audit —
  it must grandfather them or main goes red on history nobody can re-review.
- **The gate is enforced by merged code, not by the binary on PATH.** Until this
  PR merges and `dotf` redeploys, `~/.local/bin/dotf` predates the check and
  archives a Claude-signed review cleanly (confirmed live, not inferred). This is
  the same deployed-copy-lags-the-checkout class as ADR-030/#635, and it means
  "a Claude-signed review cannot be archived" is true of the repo before it is
  true of any given machine.
- **Transcript size.** `--mode json` / `--output-format stream-json` emits an
  event stream. Committing it alongside `review.md` makes *how* a review reasoned
  auditable, which is the reusable half of observability — but it needs a size
  sanity check before it becomes a habit.

## Acceptance criteria

- [ ] A `review.md` signed by a reviewer outside the pool is refused by
      `dotf spec archive`, naming the declared escapes.
- [ ] An absent pool skips the check; a malformed pool refuses.
- [ ] The pool is data readable by any harness, not a Go constant.
- [ ] `dotf spec review <spec-id>` passes the model explicitly, so the pin does
      not depend on unversioned per-machine state.
- [ ] The run is observable while it happens (named tmux session), with the
      Windows/no-tmux degrade stated and implemented.
- [ ] A machine-readable transcript lands beside `review.md`.
- [ ] **Both configured paths produce a real review**: NaN (evidence: BUG-074
      round 3) and Gemini via `agy` (evidence: one real run, to be produced).
- [ ] The gate's predicate is mutation-tested — deleting the pool check turns a
      named test red.

## References

- Bitácora board: mlorentedev/dotfiles#955
- `cli/internal/spec/review.go` — `contractFiles`, `Review.Reviewer`, `checkReviewGate`
- `harness/agent-frontmatter.schema.json` — the reserved neutral `model` key
- `scripts/compile-harness.sh` — where `model` is dropped pending H-044
- `specs/archive/HARNESS-043-curator-dogfood-slice/proposal.md` — where H-044 was deferred from
- `specs/archive/BUG-074-doctor-bw-reach/review.md` — the three rounds this spec argues from
- Related issues: #956 (the third review's own findings), #898 (a check never
  observed failing is not evidence)
