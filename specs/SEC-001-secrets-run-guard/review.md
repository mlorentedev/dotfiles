---
spec: "SEC-001-secrets-run-guard"
verdict: "FAIL"
reviewed_sha: "1a03ae620783c8281a7ebce8e951bc4784986f1e"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-23"
---

## Adversarial review

**Scope**: SEC-001-secrets-run-guard, round 2 — the whole change
`git diff 0105662daf0c04c77f43adc9b02f588604b148c6...HEAD` (`1a03ae6`).
**Sources**: `specs/SEC-001-secrets-run-guard/{proposal,tasks,verification}.md` + `features.json`;
`cli/internal/cmd/secrets.go`, `secrets_test.go`, `secrets_registry_test.go`,
`secrets_select_test.go`, `secrets_child_{unix,windows}.go`, `secrets_child_unix_test.go`;
`ai/claude/settings.json`, `ai/pi/models.json`, `AGENTS.md`,
`docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`,
`docs/lessons/lesson-261-never-test-secret-guards-against-live-credentials-and-redact-at-the-stream-boundary.md`,
`setup-linux.sh` (`merge_claude_settings`), `tests/claude-settings-template.bats`,
`harness/reviewer-pool.json`.

**Base note.** The launcher's base is the parent of the commit that added the spec folder (a
retroactive spec), so the reported range carries 86 commits and ~32k lines of other specs' work
(CLI-072/075, HARNESS-*, SEC-002). This review therefore read the SEC-001 surface at HEAD *and* its
interaction points inside that range — SEC-002's pty path, CLI-078/080's secrets changes — which is
what matters for judging the late fixes against the early ones. Every claim below was produced by
running things in this session, on synthetic fixtures only (no live Bitwarden/age store was touched,
per lesson 261): `go build ./...`, `go vet ./...`, `go test ./... -count=1` (24 packages ok, 0
fail), `golangci-lint run` on the pinned 2.12.2 (0 issues), all seven `features.json` verifiers
(exit 0) plus their two negative controls (exit 1, reproduced), four one-at-a-time mutations
restored after each run, a redaction fuzz, and scratch unit probes against the real detection,
guard and resolution paths — the probe file was deleted before this review was written.

### Spec and task alignment

- **AC1/AC2 (refuse `env`, `printenv`, `export`, by base name and by path)** — implemented and
  pinned by named rows in `TestAssertSafeChildCommand` (`bare env`, `path to env`, `bare printenv`,
  `bare export`). Holds. The AC's second clause, "without decrypting or launching", does **not**
  hold: see F4 — the child is not launched, but the secrets are decrypted first.
- **AC3 (shell wrappers detected)** — round 1's two demonstrated bypasses (`sh -c '/usr/bin/env'`,
  `sh -c 'env>x'`) are closed. `snippetIntrospection` is now a whole-word split of the snippet read
  as the shell reads it (backslash-newline fold, quotes and backslashes removed); the tests pin each
  part separately and mutation M3 (path separator removed from the split class) is killed by the four
  path-form rows. Two further bypasses survive at the same layer: F5.
- **AC4 (legitimate tools unhindered)** — holds for direct binaries (named rows for `goreleaser`,
  `python3`, `dotf review`, `echo … environment`, `run-env-check`, `cat .env.example`). Round 1's
  over-block inside snippets (`bash -c 'echo env'`) is declined with a stated reason in
  `verification.md`; failing closed is the intended bias and the AC names direct tools. Accepted.
- **AC5 (table-driven unit tests)** — 31 rows, safe and unsafe, table-driven. Present, and now
  genuinely load-bearing: mutation M1 (reintroducing the round-1 write-size gate in `holdBack`)
  is killed by `TestRedactWriter_SecretInTinyChunksNeverLeaks` plus two other named tests.
- **AC6 (Claude deny list + Pi models catalog)** — both template edits are present, including the
  three entries AC6 names verbatim. `Bash(env:*)` / `Bash(printenv:*)` / `Bash(export -p:*)` are
  covered by no test and no verifier (F3), and the list never reaches an existing installation (F2).
- **AC7 (byte-level redactor)** — holds at HEAD, and this is the strongest part of the change. The
  hold-back is prefix-aware on every write, so chunking cannot defeat it: 200 randomized chunkings
  × 4 secrets (8, 26, 64 bytes, and one with overlapping repeats), whole-write, and the
  stream-ends-mid-secret Flush case all put no complete secret in the sink. `TestRedactWriter_SecretInTinyChunksNeverLeaks`
  (1–3 byte writes) is the round-1 regression test and M1 confirms it bites.
- **AC8 (lesson 261)** — file present and substantive; f7 checks it.
- **AC9 (`show` 1-2-3 plus agent-session refusal)** — masking, `--reveal`, `-c` and the refusal
  order (clip → agent refusal → TTY mask → print; `--reveal` never bypasses the refusal) all hold,
  pinned by `TestSecretsShow_{RejectsAgentSession,TTYMasking,RevealFlag,ClipFlag}`. The refusal's
  *coverage* does not: F1.
- **AC10 (deny-list hardening)** — entries present; coverage and deployment as AC6 (F2, F3).
- **Tasks / verification bookkeeping** — `verification.md` is now filled with per-AC evidence and
  honest dispositions, and the claims I could re-run are accurate: 24 packages ok, golangci-lint
  2.12.2 at 0 issues, both negative controls exit 1, and the mutation kills I repeated. The two
  false ticks round 1 found are corrected. Note that `[x] PR opened referencing this spec folder`
  describes the landing PR #1459; this follow-up branch carries no PR (`gh pr list --head
  fix/secrets-guard-review-findings` → empty), which is consistent with the review being requested
  through `review-request.json` instead.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Major | REAL | agent-session refusal (AC9) | `agentSessionMarkers` omits the two markers pi sets (`AI_AGENT=pi`, `PI_CODING_AGENT=true`), so in a pi session launched without a Claude Code ancestor the refusal does not fire and `dotf secrets show <id>` writes the **plaintext value** to a non-TTY stdout. `AGENTS.md` (injected into every agent) and ADR-028 promise that it refuses. The deferral in `verification.md`/`#1646` rests on "pi … export[s] no marker the list knows" and "needs deploy work in six places" — pi's entry points set both markers, `AI_AGENT` is explicitly the *generic* one, and the whole fix is one list entry plus the two doc mentions `TestAgentSessionMarkersAreDocumented` already requires. | Probe (deleted) with the real detector and a synthetic value: markers cleared, `AI_AGENT=pi`/`PI_CODING_AGENT=true` → `detectAgentSession()==false`, `err=<nil> stdout="synthetic-review-probe-value"`. pi sets them in `dist/cli/setup.js` and `dist/bundle/cli-runtime.js`; this session's `CLAUDECODE=1` is inherited (process chain `bash ← pi ← dotf ← dotf-dev ← zsh ← claude`), which is why the gap is invisible from inside this repo's own sessions. | UNTESTED — `TestDetectAgentSession_EachMarkerIsSufficient` iterates the list itself (blind to an absent member), `TestSecretsShow_RejectsAgentSession` stubs `isAgentSession`, and `TestAgentSessionMarkersAreDocumented` only fires on an *addition* (mutation M5: adding `AI_AGENT` → doc test fails; nothing fails on the omission) | code + tests + `AGENTS.md`/ADR-028 (not contract set) |
| Major | REAL | deployment (AC6, AC10) | Both deny-list ACs are measured on the template, and the template's deny list reaches **no existing installation**: `merge_claude_settings` unions `permissions.allow` and never names `permissions.deny`, so every machine already carrying `~/.claude/settings.json` keeps whatever deny list it had. `verification.md` discloses this and measures 0 deny rules deployed on msi. The ACs read as met while the protection they were written for is inert, which is this repo's own "a guard that skips is a guard that passes" class. | `setup-linux.sh` merge jq (`… \| .permissions.allow = (((.permissions.allow // []) + $tmpl.permissions.allow) \| unique)`, no `.permissions.deny` clause); `verification.md` §"Known gap"; f6 passes against the template alone, so it cannot see the deployed file | UNTESTED — no test reads the deployed file or the merge's deny behaviour; `tests/claude-settings-template.bats` mentions `deny` zero times | spec artifacts (contract set) + code (`#1339` port) |
| Minor | REAL | verification coverage (AC6) | AC6's three named deny entries are verified by nothing: removing all three leaves f6 exit 0, `tests/claude-settings-template.bats` 33/33 ok, and the Go suite green. f6 checks AC10's four entries instead, and the "every dotfiles-owned top-level key is named in both merge policies" bats case stops at top-level keys, so `permissions` is satisfied by `.permissions.allow` and the deny half slips past — the same shape that case's own comment describes. | Mutation this session: `jq`-remove the three entries → f6 `rc=0`, bats `33 ok / 0 not ok`, `go test ./internal/cmd` ok; restored after | UNTESTED | tests (a bats case) or `features.json` (contract set) |
| Minor | REAL | guard ordering (AC1) | `dotf secrets run -- env` resolves and decrypts every mapped secret, and only then does `assertSafeChildCommand` refuse it, so AC1's "exiting non-zero **without decrypting** or launching" is false as implemented. Nothing is printed (the child never launches and the paths already refuse), so the cost is the false contract claim and a guard that runs after the work it exists to prevent. | Probe (deleted) with a synthetic age fixture and a recording decryptor: `err=refusing to run introspection command "env" … decrypted=true stdout=""`. Code path: `newSecretsRunCmd` RunE → `resolveInjectedSecrets` → `runChild` → `assertSafeChildCommand` | UNTESTED | code (guard before resolution) or spec artifacts |
| Minor | REAL | guard coverage (AC1/AC2/AC3) | Two more one-word bypasses survive at the function level: `env.exe` / a full `…\Git\usr\bin\env.exe` path (the base is lowercased but a Windows executable suffix is not stripped, so naming the file instead of the bare command passes the guard), and `busybox env` (busybox is in the shell list as a multi-call binary, but only its `-c` form is inspected). `xargs`-style wrappers are covered by F6; the redactor still scrubs injected values ≥6 chars, so what escapes is the key names of every injected secret and any shorter value, not the values themselves. | Guard-table probe (deleted): `["env.exe"] blocked=false`, `["C:\\Program Files\\Git\\usr\\bin\\env.exe"] blocked=false`, `["busybox","env"] blocked=false`, while `["env"]`, `["/usr/bin/env"]`, `["sh","-c","e\"\"nv"]` are blocked | UNTESTED — no table row for either form | code + tests (the `.exe` case is not covered by `#1650`, whose scope is shells and their flags) |
| Question | SPECULATIVE | guard scope | Wrapper/macro indirection is unimpeded and is arguably outside the proposal's declared boundary ("we block by binary and shell arguments, not inspecting arbitrary external scripts"): `nice env`, `timeout 5 env`, `xargs env`, `sh -s` with the snippet on stdin, `sh script.sh`, and run-time assembly such as `en$'v'`. Confirm the boundary is intended and give it a line under "Out of scope" in the next proposal revision; no code change is indicated. | Guard-table probe (deleted): all four wrapper forms and `en$'v'` return `blocked=false`; `bash -c 'echo env'` over-blocks (round 1 F4, declined, accepted) | UNTESTED | spec artifacts (next proposal revision) — surface only |
| Minor | REAL | board currency | The ticket cited as this residual's home, `#1646`, still says the refusal "never fires in Claude Code … and knows no other harness". The first half was fixed in this change (`60d2251`) and the second is wrong for pi (F1), so the tracker's premise no longer matches reality — Standing Order #8 territory if it is left as the pointer future sessions follow. | `gh issue view 1646` (state OPEN, title and body as quoted) against the shipped `agentSessionMarkers` and the F1 probe | n/a | forge (edit `#1646`) — or fold into F1's fix |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | The redactor, the guard's word-reading and the `show` flag/order matrix hold under fuzzing and mutation, but AC9's refusal is blind to pi (a demonstrated plaintext leak), AC6/AC10 measure a layer that cannot fail, and AC1's "without decrypting" is false. |
| Verification       | B | `verification.md` is filled, per-AC, reproducible; I re-ran its toolchain, both negative controls and three of its mutation kills, all accurate — but it contains the falsified marker rationale behind F1, and f6 leaves AC6's entries uncovered. |
| Scope              | B | The spec's own diff (~1.1k lines across the guard, redactor, markers, docs and spec) matches the proposal; the launcher's base adds 85 unrelated commits, which is the retroactive-launch artifact noted above, not creep in this change. |
| Reliability        | B | Guard fails closed on every path it covers, both child paths are guarded, hold-back is content-aware rather than a fixed window, error and flush paths are ordered; the one skipped guard (F1) is counted under Correctness rather than twice. |
| Maintainability    | B | Small functions, table-driven tests, comments that explain why; the marker list's correctness rests on a doc-coupling test rather than on the fact it is meant to encode, which is the structure F1 exploits. |
| Handoff-readiness  | B | Spec updates, dispositions, promotion checks and follow-up tickets are present; the ticket they defer to still states the pre-fix premise (F7) and no lesson was added for the pi marker beyond lesson 287's class. |

### Verdict
FAIL

Two **REAL Major** findings force this on the severity axis (F1, F2) — a demonstrated plaintext
secret leak in a harness this repo runs agents on, and two security ACs whose protection never
reaches an installed machine — with Correctness at C for the same reasons. No dimension is a D, and
several things are in better shape than round 1 left them: the redactor is robust, round 1's F2
bypasses are closed for real, and the spec artifacts are now filled rather than scaffolded.

### Recommended next steps

F1 and F4 need code; F2/F3 touch the contract set, so they land in the same round and are re-reviewed
together. `verification.md` is outside the staleness set and is the right place to record each
disposition (applied / ticketed / declined with a reason).

1. **F1 (code + tests + docs) — the one that blocks.** Add pi's process markers to
   `agentSessionMarkers` (at minimum `AI_AGENT`, the documented generic marker; `PI_CODING_AGENT`
   for the specific one) and name both in `AGENTS.md` and ADR-028 — `TestAgentSessionMarkersAreDocumented`
   requires the second half, so the two edits go together. Add a test that fails on a **missing**
   harness marker, not just on an undocumented one: iterate a table of {harness → marker} and assert
   each is in the list (today `TestDetectAgentSession_EachMarkerIsSufficient` iterates the list, so it
   cannot see an absent member). Then `#1646` narrows to opencode, Codex and Copilot, and its body
   should say so.
2. **F2 + F3 (spec artifacts, contract set; tests) — make the ACs measure the layer that can fail.**
   Either give AC6/AC10 a deployment assertion (a `setup-linux.bats` case that runs the merge against
   a fixture carrying an existing `permissions` object and asserts `deny` survives/indexes correctly,
   plus f6 extended to AC6's three entries), or reword both ACs to say plainly that the template is
   hardened and deployment is `#1339`. Do not leave the current wording standing: it reports a
   protection the next `setup` run removes.
3. **F4 (code).** Call `assertSafeChildCommand` before `resolveInjectedSecrets` in
   `newSecretsRunCmd`'s RunE, so AC1's "without decrypting or launching" becomes true and a refused
   command costs nothing. If the ordering is deliberate (say, so the refusal reports a decryption
   failure first), reword AC1 instead — but then say why in `verification.md`.
4. **F5 (code + tests).** Strip a Windows executable suffix from the base name and add table rows for
   `env.exe` and `busybox env`. Worth folding the suffix into `#1650`'s scope if that lands first,
   since it is the same platform.
5. **F6 (spec artifacts).** Confirm the wrapper/indirection boundary is intended, and state it under
   "Out of scope" in the next proposal revision — including the residual disclosure (key names and
   any sub-6-character value are not scrubbed).
6. **F7 (forge).** Update `#1646`'s title and body to the post-F1 position; otherwise the ticket
   future sessions read asserts a leak that the fix removed while omitting the one that remains.
7. **Re-review.** `dotf spec review SEC-001-secrets-run-guard` after 1–3 land. `dotf spec archive`
   is **not advisable** now: the verdict is FAIL, and the archive gate would refuse this review.
