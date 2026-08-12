---
generated: true
generated_from: 00_meta/skills/new-ticket/SKILL.md
generated_sha: dcc95707e91b91c7
id: new-ticket-skill
type: skill
status: active
created: "2026-06-10"
owner: manu
name: new-ticket
description: "Create a bitácora ticket interactively with suggested defaults. Triggers on /new-ticket, 'new ticket', 'create a ticket', 'file an issue on the board', 'crea un ticket', 'nuevo ticket', and the detect-then-ticket standing order. Proposes Type, Priority, Status, assignee and the AREA-NNN-slug ID, the human confirms, then it opens the issue and sets the board fields."
allowed-tools: [Bash, Read, AskUserQuestion]
---

# New Ticket Workflow

> One command turns a title (+ optional description) into a board-ready issue on the bitácora (GitHub Project #1): consistent `Status` / `Priority` / `Type` and a stable `AREA-NNN-slug` ID, with the right home repo. Implements the ID convention (CUR-008) and the status lifecycle (HARNESS-010).
> **Core principle:** the agent *proposes* sensible defaults; the human *decides*. Never create a ticket silently in interactive mode.

## When to use

- `/new-ticket "<title>"` explicitly, or "create a ticket" / "new ticket" / "file an issue on the board" / "crea un ticket" / "nuevo ticket".
- When the **detect→ticket** standing order fires: you found work outside the current change's scope → file it instead of silently dropping it or letting scope creep (see [[feedback_side_comments_are_triage]]).

## When NOT to use

- Work you are about to do *right now* in the current change — just do it.
- In-spec task tracking — that lives in the spec's `tasks.md`, not on the board.
- A trivial side-comment you can handle inline (triage it; don't manufacture a ticket).

## Environment (resolve once)

- **Board:** owner `mlorentedev`, project number `1`, Project ID `PVT_kwHOAM7xJs4BZ6GY`.
- **Canonical field/ID + convention reference:** [[bitacora-project-setup]] §2 (fields), §3 (ID convention), §1b (home repo), §5 (status lifecycle). Resolve field/option IDs **at runtime** (snippet below) — do NOT paste a third copy into a ticket; embedded copies drift (the exact failure class behind OPS-002's protected-branch/rollout lessons).
- `gh` authenticated with `repo` + `project` scope.

## The flow (interactive — suggestion-first)

### 1. Gather input

Take the title and any description/body the user gave. A title alone is enough; everything else is *proposed*.

### 2. Compute the proposal (do not ask yet)

| Field | How to propose |
|-------|----------------|
| **Home repo** | Infer from §1b: harness / methodology / skills / agents / `AGENTS.md` / SDD → `dotfiles`; vault content or structure → `knowledge`; a specific project's code/infra → that repo. Tie-breaker: where the deploy/runtime contract lives, else `dotfiles`. Fallback: the current repo (`git rev-parse --show-toplevel`). |
| **AREA prefix** | Pick the area the work belongs to (e.g. `HARNESS`, `OPS`, `CUR`, `AI`, `BUG`, `REFACTOR`, `IDEAS`). If the user typed one in the title, reuse it. |
| **Type** | Infer from the AREA prefix via the table below. |
| **Priority** | Propose from "Priority by signal" below (calibrated to the board's real distribution); `P2` when no signal fires. |
| **Status** | `Backlog`. |
| **Labels** | GitHub labels by nature, from the mapping in "Labels by nature" below (primary from Type + optional secondary from AREA). |
| **NNN** | Next free number for that AREA in the home repo — scan existing issue titles (snippet below). |
| **slug** | Short kebab-case of the title (≤5 words), AREA prefix stripped. |

→ Final ID = `AREA-NNN-slug` (e.g. `HARNESS-017-ticket-templates`). It is carried in BOTH the issue **title** (`HARNESS-017: …`) and the `ID` custom field (§3).

### 3. Confirm with the user — `AskUserQuestion`

State the proposal in one line first (repo · `AREA-NNN-slug` · title · proposed labels), then ask. Put the recommended option **first** and label it "(Recommended)". Ask the fields most likely to need a decision:

1. **Type** — `spec` · `bug` · `chore` · `ideas` (pre-selected = inferred).
2. **Priority** — the "Priority by signal" pick is pre-selected and marked `(Recommended)`; offer the other three of `P0` · `P1` · `P2` · `P3`.
3. **Assignment** — `Leave in Backlog (Recommended, unassigned)` · `Self-assign → In Progress` (self-assigning fires `bitacora-status.yml`, HARNESS-010).

If the user needs a different **repo**, **AREA**, or **slug**, they say so (or pick "Other") — re-propose and re-confirm. Do not proceed until the ID and repo are agreed.

### 4. Create the issue + set the board fields

```bash
OWNER=mlorentedev; PROJECT_NUM=1; PROJECT_ID=PVT_kwHOAM7xJs4BZ6GY
REPO="$OWNER/<home-repo>"; AREA=HARNESS; NNN=017; SLUG=ticket-templates
TITLE="<human title>"; BODY="<description, or a short stub>"
FULL_ID="$AREA-$NNN-$SLUG"

# 1. Ensure nature labels exist (idempotent; colors from the "Labels by nature" table),
#    then open the issue — AREA-NNN in the title (§3)
LABELS="feature"          # comma-separated, from the "Labels by nature" mapping
for L in $(printf '%s' "$LABELS" | tr ',' ' '); do
  gh label list --repo "$REPO" --json name -q '.[].name' | grep -qx "$L" \
    || gh label create "$L" --repo "$REPO" --color "<color-from-table>" --description "<desc-from-table>"
done
URL=$(gh issue create --repo "$REPO" --title "$AREA-$NNN: $TITLE" --body "$BODY" --label "$LABELS")

# 2. Put it on the board (idempotent — returns the existing item if already added)
ITEM=$(gh project item-add "$PROJECT_NUM" --owner "$OWNER" --url "$URL" --format json \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")

# 3. Resolve field + option IDs at runtime (no hardcoded copy).
#    NOTE: string concatenation, not f-strings — inside single-quoted `python3 -c '...'`
#    the double quotes are literal; f-strings with escaped quotes break the parser.
eval "$(gh project field-list "$PROJECT_NUM" --owner "$OWNER" --format json | python3 -c '
import json,sys
F={x["name"]: x for x in json.load(sys.stdin)["fields"] if "name" in x}
def opt(fn,on): return next(o["id"] for o in F[fn]["options"] if o["name"]==on)
print("F_STATUS=" + F["Status"]["id"]);     print("O_BACKLOG=" + opt("Status","Backlog"))
print("F_PRIORITY=" + F["Priority"]["id"]); print("O_P2=" + opt("Priority","P2"))
print("F_TYPE=" + F["Type"]["id"]);         print("O_CHORE=" + opt("Type","chore"))
print("F_ID=" + F["ID"]["id"])
')"

# 4. Set the fields (single-select by option id; ID is a text field)
gh project item-edit --id "$ITEM" --project-id "$PROJECT_ID" --field-id "$F_STATUS"   --single-select-option-id "$O_BACKLOG"
gh project item-edit --id "$ITEM" --project-id "$PROJECT_ID" --field-id "$F_PRIORITY" --single-select-option-id "$O_P2"
gh project item-edit --id "$ITEM" --project-id "$PROJECT_ID" --field-id "$F_TYPE"     --single-select-option-id "$O_CHORE"
gh project item-edit --id "$ITEM" --project-id "$PROJECT_ID" --field-id "$F_ID"       --text "$FULL_ID"

# 5. (only if the user chose Self-assign) → flips Status to In Progress via HARNESS-010
# NUM=$(basename "$URL"); gh issue edit "$NUM" --repo "$REPO" --add-assignee @me
```

Set the option variable for the *confirmed* Type/Priority (resolve `O_BUG`/`O_SPEC`/`O_IDEAS`, `O_P0`…`O_P3` the same way). Always set `Status` explicitly — do not rely on a board workflow having run yet.

### 5. Report

Print the issue URL, the `AREA-NNN-slug` ID, the home repo, and the resulting board status. If self-assigned, confirm it moved to In Progress. Confirm the fields landed (the `ID` text field surfaces under the JSON key `iD`):

```bash
gh project item-list "$PROJECT_NUM" --owner "$OWNER" --format json --limit 400 | python3 -c "
import json,sys
for i in json.load(sys.stdin)['items']:
    if i.get('content',{}).get('number')==int('$NUM'):
        print(i.get('status'), i.get('priority'), i.get('type'), i.get('iD'), '·', i['content']['title']); break"
```

## Non-interactive / autonomous mode

For agents acting under **detect→ticket** (no human in the loop), skip step 3's `AskUserQuestion`: accept the computed defaults (or explicit args), create the issue, set the fields, and report the URL. Defaults stay **Backlog / P2 / unassigned + nature labels** so a human triages priority and start later — an autonomous agent files the work, it does not self-prioritize it.

## Type inference from the AREA prefix

The `Type` field has four options; map the prefix to the closest, then let the human override:

| AREA prefix | Type |
|-------------|------|
| `BUG` | `bug` |
| `IDEAS`, `RFD` | `ideas` |
| `AI`, `HARNESS`, `SDD`, `SPEC`, `FEAT`, `MEMORY`, `SELF`, `AGENTS`, `ARCH` | `spec` |
| `OPS`, `CUR`, `REFACTOR`, `POLISH`, `DX`, `TERM`, `WIN`, `VAULT`, `SH`, `CHORE` | `chore` |
| anything else | `chore` (safe default) |

The mapping is a *suggestion*, not a rule — the human confirms it in step 3.

## Priority by signal

Propose `Priority` the same way Type and Labels are proposed — derived from the ticket's nature, not a flat default — then the human confirms in step 3. Match the **highest** signal that fires:

| Signal in the title/body | Priority |
|--------------------------|----------|
| Broken or blocking **now** — CI red, a shipped command/tool erroring, secret/data-loss risk, or it blocks other in-flight work | `P0` (rare — reserve for true "stop and fix"; ~never on this board) |
| A real **defect** in shipped code (`BUG-*`, "fails/errors/wrong"), OR work that **unblocks the active arc** / is the named next step, OR a **security/secrets** gap | `P1` |
| Normal backlog — a capability, refactor, doc, or chore with **no urgency signal** | `P2` (default) |
| Nice-to-have, speculative, **parked/undecided**, cosmetic/polish, or an `IDEAS`/`RFD` research item | `P3` |

**Calibrate, don't inflate.** Resolve the live distribution before proposing, so the proposal tracks how the board actually uses the scale (today: `P2` ~90%, `P1` selective ~5%, `P0` ~never, `P3` the long tail):

```bash
gh project item-list 1 --owner mlorentedev --format json --limit 1000 | python3 -c "
import json,sys; from collections import Counter
c=Counter(i.get('priority') for i in json.load(sys.stdin)['items'] if i.get('priority'))
print(dict(c))"
```

When two signals tie, take the higher only if there is a concrete urgency cue (a date, a blocked dependency, a red CI); absent that, prefer `P2`. The pick is a *suggestion* like Type — the human confirms it in step 3. **Autonomous mode does NOT apply this rubric's upgrade**: an unattended `detect→ticket` run still files at `P2` and leaves prioritization to a human triage (see "Non-interactive / autonomous mode").

## Labels by nature

Every ticket gets GitHub **labels** at creation (in addition to the board fields) so the bitácora can be filtered by nature cross-repo. One **primary** label inferred from Type + optional **secondary** labels from the AREA/content. Canonical set (create-if-missing with these exact colors; do NOT use `gh label create --force` — it overwrites color/description of existing labels):

| Label | Color | When |
|-------|-------|------|
| `bug` | `d73a4a` | Type=bug |
| `feature` | `a2eeef` | Type=spec (new capability) |
| `chore` | `ededed` | Type=chore (maintenance, refactor, cleanup) |
| `idea` | `fbca04` | Type=ideas |
| `docs` | `0075ca` | DOC*/README/runbook/ADR work |
| `security` | `b60205` | SEC*/auth/secrets/CVE work |
| `debt` | `5319e7` | DEBT*/known tech-debt paydown |
| `infra` | `0e8a16` | ANSIBLE/TF/K8s/VPN/CERT/HELM-class work |
| `ci` | `bfd4f2` | CI*/workflows/runners |

Idempotent ensure (per label, deterministic color):

```bash
gh label list --repo "$REPO" --json name -q '.[].name' | grep -qx "$L" \
  || gh label create "$L" --repo "$REPO" --color "<color>" --description "<short desc>"
```

The mapping is a suggestion like Type — the human can override in step 3. Labels live at repo level, so they survive board re-organizations and work for repos outside the bitácora too.

## Compute the next free NNN

```bash
REPO=mlorentedev/dotfiles; AREA=HARNESS
gh issue list --repo "$REPO" --state all --limit 800 --json title | python3 -c "
import json,sys,re
A='$AREA'
ns=[int(m.group(1)) for r in json.load(sys.stdin)
    for m in [re.match(rf'{A}-0*(\d+)', r['title'], re.I)] if m]
print('%03d' % (max(ns)+1 if ns else 1))"
```

NNN is **zero-padded to 3 digits** to match the convention (`HARNESS-016`, `OPS-002`). Scans both open and closed issues (forward-only, per §3 — never reuse a retired number). If the AREA also has `specs/<AREA>-NNN-*` directories that outrun the issues, cross-check and take the higher.

### Concurrency guard — parallel sessions race scan-then-create (2026-07-07)

`scan → create` is not atomic: on 2026-07-07 three parallel sessions filing tickets minutes apart produced three duplicate IDs (TOOL-017 ×2, DOCS-001 ×2, then TOOL-020 ×2 during remediation). Harden every create:

1. **Re-scan immediately before `gh issue create`** — at create time, not minutes earlier when the proposal was computed.
2. **Scan the board's `ID` field too, not only the home repo's issue titles** — a concurrent session may have claimed `AREA-NNN` from another repo or before its issue lands in your scan window. If GraphQL is rate-limited, fall back to REST title scans (`gh api repos/<owner>/<repo>/issues`) across the bitácora repos.
3. **Verify right after creating** — re-run the scan; if a duplicate raced in, renumber immediately. Yield rule: the ticket WITHOUT its board `ID` field set renumbers; if neither is set, the higher issue `#number` yields.

## Common mistakes

- **Hardcoding field/option IDs** into the skill or a script → silent drift when the project changes. Resolve at runtime (step 4.3); runbook §2 is the human reference, not a copy to fork.
- **Self-assigning a ticket you will not start now** — it flips Backlog → In Progress (HARNESS-010), misreporting the board. Self-assign only when you are actually starting.
- **Forgetting the `ID` text field** — the title carries `AREA-NNN` but the board's ID column stays empty, breaking the stable human ID (§3).
- **Filing in the wrong home repo** — a harness/skill ticket belongs in `dotfiles`, vault-content in `knowledge` (§1b). The `Repository` field makes the home explicit; pick it deliberately.
- **Creating silently in interactive mode** — always surface the proposal and get confirmation first.

## References

- Runbook: [[bitacora-project-setup]] — §1b (home repo), §2 (fields & IDs), §3 (ID convention, CUR-008), §5 (status lifecycle, HARNESS-010).
- `AGENTS.md` — Standing Order #8 (bitácora status reconciliation) and the detect→ticket rule.
- Related skills: [[00_meta/skills/handoff/SKILL|handoff]] (reconciles board status at session end), [[00_meta/skills/spec/SKILL|spec]] (`init` is gated on an open issue this skill can create).
- ADR: dotfiles `docs/adr/adr-018-de-vault-task-placement.md` (task state lives on the board).
